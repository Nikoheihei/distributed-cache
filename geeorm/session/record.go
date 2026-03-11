package session

import (
	"GopherStore/geeorm/clause"
	"errors"
	"reflect"
)

// 实现记录增删查改相关的代码

// 希望的调用方式是s := geeorm.NewEngine("sqlite3", "gee.db").NewSession()
// u1 := &User{Name: "Tom", Age: 18}
// u2 := &User{Name: "Sam", Age: 25}
// s.Insert(u1, u2, ...)
func (s *Session) Insert(values ...interface{}) (int64, error) {
	recordValues := make([]interface{}, 0)
	for _, value := range values {
		table := s.Model(value).RefTable()
		//这里INSERT写入clause会被反复覆盖，但是由于是values...说明插入的数据来自同一个表
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames)
		//只有recordValues才会不断地累积
		recordValues = append(recordValues, table.RecordValues(value))
	}
	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

//希望的调用方式是s := geeorm.NewEngine("sqlite3", "gee.db").NewSession()
//var users []User
//s.Find(&users);

func (s *Session) Find(values interface{}) error {
	destSlice := reflect.Indirect(reflect.ValueOf(values))
	destType := destSlice.Type().Elem() //返回切片的单个元素实例
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable()
	//reflect.New(destType) 返回的是“指向该类型的指针值”，不是这个类型本身的值。
	//所以需要调用Elem函数

	s.clause.Set(clause.SELECT, table.Name, table.FieldNames)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	rows, err := s.Raw(sql, vars...).QueryRows()

	if err != nil {
		return err
	}
	//把数据库查询得到的列的数据读入到dest中，每次循环创建一个新的dest，将字段名平铺开后，利用scan函数读取数据
	for rows.Next() {
		dest := reflect.New(destType)
		destValue := dest.Elem()
		var values []interface{}
		//把dest的所有字段平铺开来
		for _, name := range table.FieldNames {
			//获取字段对应值的地址
			values = append(values, destValue.FieldByName(name).Addr().Interface())
		}
		//调用scan将该行记录每一列的值依次赋值给values中的每一个字段
		if err := rows.Scan(values...); err != nil {
			return err
		}
		destSlice.Set(reflect.Append(destSlice, destValue))
	}
	return rows.Close()
}

// Update接收2种入参，平铺开来的键值对和map类型的键值对
func (s *Session) Update(kv ...interface{}) (int64, error) {
	m, ok := kv[0].(map[string]interface{})
	if !ok {
		//如果不是map类型则转化
		m = make(map[string]interface{})
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
	}
	s.clause.Set(clause.UPDATE, s.RefTable().Name, m)
	sql, vars := s.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Session) Delete() (int64, error) {
	s.clause.Set(clause.DELETE, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.DELETE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// 以下是链式调用，因为我们想向下面一样调用函数
// s := geeorm.NewEngine("sqlite3", "gee.db").NewSession()
// var users []User
// s.Where("Age > 18").Limit(3).Find(&users)
func (s *Session) Count() (int64, error) {
	s.clause.Set(clause.COUNT, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.COUNT, clause.WHERE)
	row := s.Raw(sql, vars...).QueryRow()
	//查询结果只有一行一列
	var tmp int64
	//将数据填入tmp中
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}
	return tmp, nil
}
func (s *Session) Limit(num int) *Session {
	s.clause.Set(clause.LIMIT, num)
	return s
}
func (s *Session) Where(desc string, args ...interface{}) *Session {
	var vars []interface{}
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}
func (s *Session) OrderBy(desc string) *Session {
	s.clause.Set(clause.ORDERBY, desc)
	return s
}

// first方法可以这么调用
// u := &User{}
// _ = s.OrderBy("Age DESC").First(u)
func (s *Session) First(value interface{}) error {
	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()
	if err := s.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return errors.New("not found")
	}
	dest.Set(destSlice.Index(0))
	return nil
}
