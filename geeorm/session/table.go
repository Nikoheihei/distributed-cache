package session

import (
	"GopherStore/geeorm/log"
	"GopherStore/geeorm/schema"
	"fmt"
	"reflect"
	"strings"
)

// 用于给refTable赋值。如果空或者映射对象不匹配，更新session。不会改变数据库连接和dialect。只会改变映射对象
func (s *Session) Model(value interface{}) *Session {

	if s.refTable == nil || reflect.TypeOf(value) != reflect.TypeOf(s.refTable.Model) {
		s.refTable = schema.Parse(value, s.dialect)
	}
	return s
}

// 返回refTable的值，如果没有赋值，则打印错误日志。
func (s *Session) RefTable() *schema.Schema {
	if s.refTable == nil {
		log.Error("Model is not set")
	}
	return s.refTable
}

// 创建表格
func (s *Session) CreateTable() error {
	table := s.RefTable()
	var columns []string
	for _, field := range table.Fields {
		columns = append(columns, fmt.Sprintf("%s %s %s", field.Name, field.Type, field.Tag))
	}
	desc := strings.Join(columns, ",")
	_, err := s.Raw(fmt.Sprintf("CREATE TABLE %s(%s)", table.Name, desc)).Exec() //这里没有修改sqlVars
	return err
}

// 删除表格
func (s *Session) DropTable() error {
	_, err := s.Raw(fmt.Sprintf("DROP TABLE IF EXISTS %s", s.RefTable().Name)).Exec()
	return err
}

// 是否存在表格
func (s *Session) HasTable() bool {
	sql, values := s.dialect.TableExistSQL(s.RefTable().Name)
	row := s.Raw(sql, values).QueryRow()
	var tmp string
	_ = row.Scan(&tmp) //scan函数复制
	return tmp == s.RefTable().Name
}
