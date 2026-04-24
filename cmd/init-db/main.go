package main

import (
	"GopherStore/geeorm"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite3"
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "gopher.db"
	}
	engine, err := geeorm.NewEngine(dbType, dsn)
	if err != nil {
		log.Fatal(err)
	}
	s := engine.NewSession()
	if dbType == "mysql" {
		s.Raw("DROP TABLE IF EXISTS `User`;").Exec()
		s.Raw("CREATE TABLE `User`(Name VARCHAR(255) PRIMARY KEY, Score INT);").Exec()
		s.Raw("INSERT INTO `User`(Name, Score) VALUES ('Tom', 630), ('Jack', 589), ('Sam', 567), ('Alice', 412), ('Bob', 731), ('Eve', 298), ('Mike', 845), ('Lily', 512), ('Rose', 476), ('David', 663), ('Jenny', 704), ('Leo', 355) ON DUPLICATE KEY UPDATE Score=VALUES(Score);").Exec()
		return
	}
	s.Raw("DROP TABLE IF EXISTS user;").Exec()
	s.Raw("CREATE TABLE user(name TEXT PRIMARY KEY, score INTEGER);").Exec()
	s.Raw("INSERT INTO user(name, score) VALUES ('Tom', 630), ('Jack', 589), ('Sam', 567), ('Alice', 412), ('Bob', 731), ('Eve', 298), ('Mike', 845), ('Lily', 512), ('Rose', 476), ('David', 663), ('Jenny', 704), ('Leo', 355);").Exec()
}
