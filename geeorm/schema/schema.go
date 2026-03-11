package schema

import (
	"GopherStore/geeorm/dialect"
	"go/ast"
	"reflect"
)

type Field struct {
	Name string //字段名
	Type string //类型
	Tag  string //约束条件
}

type Schema struct {
	Model      interface{}       //被映射的对象
	Name       string            //表名
	Fields     []*Field          //字段
	FieldNames []string          //所有字段名
	fieldMap   map[string]*Field //字段名和Field的映射关系
}

// 获取一个表的字段
func (s *Schema) GetField(name string) *Field {
	return s.fieldMap[name]
}

// 将任意的对象解析为Schema实例,dest是被映射的对象，d是使用的方言
func Parse(dest interface{}, d dialect.Dialect) *Schema {
	//因为设计的入参是一个对象的指针
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:    dest,
		Name:     modelType.Name(),
		fieldMap: make(map[string]*Field),
	}
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		//不是匿名字段，也不是非公开字段
		if !p.Anonymous && ast.IsExported(p.Name) {
			field := &Field{
				Name: p.Name,
				Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}
			if v, ok := p.Tag.Lookup("geeorm"); ok {
				field.Tag = v
			}
			schema.Fields = append(schema.Fields, field)
			schema.FieldNames = append(schema.FieldNames, p.Name)
			schema.fieldMap[p.Name] = field
		}
	}
	return schema
}

// 将对象中对应找到的值按顺序平铺成数据库一行的格式
func (schema *Schema) RecordValues(dest interface{}) []interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldValues []interface{}
	for _, field := range schema.Fields {
		fieldValues = append(fieldValues, destValue.FieldByName(field.Name).Interface())
	}
	return fieldValues
}
