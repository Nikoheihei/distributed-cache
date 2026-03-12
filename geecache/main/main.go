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
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

func connectDB(driver, dsn string) (*geeorm.Engine, error) {
	var engine *geeorm.Engine
	var err error

	// 工业级重试逻辑：尝试 5 次，每次间隔 2 秒
	for i := 0; i < 5; i++ {
		engine, err = geeorm.NewEngine(driver, dsn)
		if err == nil {
			return engine, nil
		}
		log.Printf("数据库连接失败，正在进行第 %d 次重试...", i+1)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}

func main() {

	var port int
	var api bool
	var dbName string // 新增：数据库文件名
	var metricsPort int
	flag.IntVar(&port, "port", 8001, "RPC server port")
	flag.BoolVar(&api, "api", false, "Start a web server?")
	flag.StringVar(&dbName, "db", "gopher.db", "Database file name") // 新增
	flag.IntVar(&metricsPort, "metrics-port", 9100, "metrics server port")
	flag.Parse()

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite3"
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "data/common.db"
	}

	peersEnv := os.Getenv("PEERS")
	if peersEnv == "" {
		peersEnv = "8001,8002"
	}

	engine, err := connectDB(dbType, dsn)
	if err != nil {
		return
	}

	defer engine.Close()

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
	var addrs []string
	for _, pstr := range strings.Split(peersEnv, ",") {
		pstr = strings.TrimSpace(pstr)
		if pstr == "" {
			continue
		}
		// 兼容两种格式：port 列表 or 完整地址列表
		if strings.Contains(pstr, ":") {
			addrs = append(addrs, pstr)
			continue
		}
		p, err := strconv.Atoi(pstr)
		if err != nil {
			log.Printf("Invalid peer port: %s", pstr)
			continue
		}
		addrs = append(addrs, advertiseAddr(p))
	}

	selfAddr := advertiseAddr(port)
	log.Printf("Node started | addr=%s | db=%s | driver=%s", selfAddr, dbName, dbType)

	// 初始化 RPC 池时使用 currentAddr
	pool := geecache.NewRPCPool(selfAddr)
	pool.Set(addrs...)
	group.RegisterPeers(pool)

	// 4. 如果开启 -api，则启动外网 HTTP 网关 (监听 9999)
	if api {
		go startWebServer(":9999", group)
	}
	go startMetricsServer(bindAddr(metricsPort))

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
	http.HandleFunc("/metrics", geecache.MetricsHandler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Println("Web Gateway 运行在:", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func startMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", geecache.MetricsHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Println("Metrics Server 运行在:", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
