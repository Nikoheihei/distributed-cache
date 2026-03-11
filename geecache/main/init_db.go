package main

import (
	"GopherStore/geeorm"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	engine, _ := geeorm.NewEngine("sqlite3", "gopher.db")
	s := engine.NewSession()
	s.Raw("DROP TABLE IF EXISTS user;").Exec()
	s.Raw("CREATE TABLE user(name TEXT PRIMARY KEY, score INTEGER);").Exec()
	s.Raw("INSERT INTO user(name, score) VALUES ('Tom', 630), ('Jack', 589), ('Sam', 999);").Exec()
}
