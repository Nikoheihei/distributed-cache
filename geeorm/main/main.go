package main

import (
	"GopherStore/geeorm"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	//之前：您的 go 命令尝试直接连接国外的 proxy.golang.org 服务器去下载代码，但因为网络问题连接超时失败了
	//现在：我修改了您 Go 开发环境的配置，将模块下载代理 (GOPROXY)指向了 goproxy.cn。
)

func main() {
	engine, _ := geeorm.NewEngine("sqlite3", "gee.db")
	defer engine.Close()
	s := engine.NewSession()
	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	result, _ := s.Raw("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam").Exec()
	count, _ := result.RowsAffected()
	fmt.Printf("Exec success, %d affected\n", count)
}
