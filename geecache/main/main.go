package main

import (
	"GopherStore/geecache" // 替换为你的实际包路径
	"GopherStore/geeorm"   // 替换为你的实际包路径
	"GopherStore/geerpc"   // 替换为你的实际包路径
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// User 对应数据库中的表结构
type User struct {
	Name  string `geeorm:"PRIMARY KEY"`
	Score int
}

// 绑定地址（监听用），容器和本机都监听 0.0.0.0
func bindAddr(port int) string {
	return fmt.Sprintf("0.0.0.0:%d", port)
}

// 对外宣告地址（peer 互联用）
func advertiseAddr(port int) string {
	host := "127.0.0.1"
	// Docker Compose 默认会将服务名作为 DNS 主机名
	if os.Getenv("IN_DOCKER") == "true" {
		host = fmt.Sprintf("node%d", port%8000) // 8001 -> node1, 8002 -> node2
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func main() {
	var port int
	var api bool
	var dbName string // 新增：数据库文件名
	flag.IntVar(&port, "port", 8001, "RPC server port")
	flag.BoolVar(&api, "api", false, "Start a web server?")
	flag.StringVar(&dbName, "db", "gopher.db", "Database file name") // 新增
	flag.Parse()

	// 1. 初始化 GeeORM 引擎
	engine, _ := geeorm.NewEngine("sqlite3", dbName)
	defer engine.Close()

	// 不再区分 port，大家都确保数据库里有全量数据
	s := engine.NewSession()
	_, _ = s.Raw("CREATE TABLE IF NOT EXISTS User(Name TEXT PRIMARY KEY, Score INTEGER);").Exec()
	_, _ = s.Raw("INSERT OR REPLACE INTO User(Name, Score) VALUES ('Tom', 630);").Exec()
	_, _ = s.Raw("INSERT OR REPLACE INTO User(Name, Score) VALUES ('Jack', 589);").Exec()

	// 2. 初始化 GeeCache 组，并注入 GeeORM 查询逻辑
	group := geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Printf("[GeeCache] 缓存未命中，GeeORM 正在从数据库捞取数据: %s\n", key)
			s := engine.NewSession()
			user := &User{}
			err := s.Where("name = ?", key).First(user)
			if err != nil {
				return nil, fmt.Errorf("数据库查无此人: %s", key)
			}
			return []byte(fmt.Sprint(user.Score)), nil
		}))

	// 3. 配置分布式节点列表
	allPorts := []int{8001, 8002}
	var addrs []string
	for _, p := range allPorts {
		addrs = append(addrs, advertiseAddr(p))
	}

	selfAddr := advertiseAddr(port)
	log.Printf("Node started | addr=%s | db=%s", selfAddr, dbName)

	// 初始化 RPC 池时使用 currentAddr
	pool := geecache.NewRPCPool(selfAddr)
	pool.Set(addrs...)
	group.RegisterPeers(pool)

	// 4. 如果开启 -api，则启动外网 HTTP 网关 (监听 9999)
	if api {
		go startWebServer(":9999", group)
	}

	// 5. 启动内网 RPC 服务
	bind := bindAddr(port)
	log.Printf("RPC Server 正在监听: %s\n", bind)
	l, _ := net.Listen("tcp", bind)
	server := geerpc.NewServer()
	// 注册缓存服务，让别的节点能通过 RPC 调我
	_ = server.Register(&geecache.CacheService{})
	server.Accept(l)
}

// startWebServer 极简版 Web 网关逻辑
func startWebServer(addr string, g *geecache.Group) {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := g.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	})
	log.Println("Web Gateway 运行在:", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
