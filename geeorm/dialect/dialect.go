package dialect

import "reflect"

//抽象出各个数据库差异的部分

var dialectsMap = map[string]Dialect{}

type Dialect interface {
	//将go语言的类型转换为该数据库的数据类型
	DataTypeOf(typ reflect.Value) string
	//返回某个表是否存在的sql语句和参数
	TableExistSQL(tableName string) (string, []interface{})
}

// 注册dialect实例
func RegisterDialect(name string, d Dialect) {
	dialectsMap[name] = d
}

// 获取dialect实例
func GetDialect(name string) (d Dialect, ok bool) {
	d, ok = dialectsMap[name]
	return
}
