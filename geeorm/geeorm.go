package geeorm

import (
	"GopherStore/geeorm/dialect"
	"GopherStore/geeorm/log"
	"GopherStore/geeorm/session"
	"database/sql"
)

// 交互前后的准备工作交给engine来负责，engine对应的是一个数据库的操作入口。
type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func NewEngine(driver, source string) (e *Engine, err error) {
	//创建数据库句柄，初始化驱动配置
	db, err := sql.Open(driver, source)
	if err != nil {
		log.Error(err)
		return
	}
	//尝试与数据库通信
	if err = db.Ping(); err != nil {
		log.Error(err)
		return
	}
	dial, ok := dialect.GetDialect(driver)
	if !ok {
		log.Errorf("dialect %s Not Found", driver)
	}
	e = &Engine{db: db, dialect: dial}
	log.Info("Connect database successfully")
	return
}

func (e *Engine) Close() {
	if err := e.db.Close(); err != nil {
		log.Error("Failed to close database: ", err)
	}
	log.Info("Close database successfully")
}

func (e *Engine) NewSession() *session.Session {
	return session.NewSession(e.db, e.dialect)
}
