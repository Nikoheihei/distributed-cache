// mock_data.go (随便建个临时文件跑一下就行)
package main

import (
	"GopherStore/geeorm" // 替换为你的真实包路径
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql" // 如果换成了 MySQL
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 连接 docker-compose 里的 MySQL（默认暴露到宿主机 127.0.0.1:3307）
	dbType := getenv("DB_TYPE", "mysql")
	dsn := getenv("DB_DSN", "root:root@tcp(127.0.0.1:3307)/gopherstore?charset=utf8mb4&parseTime=True")

	engine, err := geeorm.NewEngine(dbType, dsn)
	if err != nil {
		log.Fatal(err)
	}
	s := engine.NewSession()

	log.Println("开始灌入 10000 条测试数据...")
	for i := 1; i <= 10000; i++ {
		name := fmt.Sprintf("User%d", i)
		score := rand.Intn(100)
		// 批量插入或者单条插入
		_, _ = s.Raw("INSERT IGNORE INTO `User`(Name, Score) VALUES (?, ?);", name, score).Exec()
	}
	log.Println("数据准备完毕！")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
