package lama

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cast"
	"reflect"
)

type SqlxDB = *sqlx.DB
type DelFn func(where *Sql)
type SaveFn func(where *Sql)
type SelectFn func(sel *Sql, where *Sql)

var DB *Database
var DefaultDB SqlType

type Database struct {
	DB SqlxDB
}

func (s *Database) Init(db SqlxDB) error {
	s.DB = db
	DB = s
	return nil
}

func (s *Database) Add(table string, row any) *SqlResult {
	if table == "" {
		panic("table is empty")
	}

	v, ok := row.(map[string]any)
	if ok {
		return s.AddMap(table, v)
	}

	vf := reflect.ValueOf(row)
	if vf.Kind() != reflect.Ptr {
		panic("row is not ptr")
	}

	if vf.Elem().NumField() == 0 {
		panic("row is zero")
	}

	return s.AddStruct(table, row)
}

func (s *Database) AddMap(table string, row map[string]any) *SqlResult {
	var fields string
	var values []any

	for field, value := range row {
		if fields == "" {
			fields = field
		} else {
			fields += "," + field
		}
		values = append(values, value)
	}

	return s.AddRow(table, fields, values)
}

func (s *Database) AddStruct(table string, row any) *SqlResult {
	vf := reflect.ValueOf(row)
	var fields string
	var values []any

	elem := vf.Elem()
	for i := 0; i < elem.NumField(); i++ {
		v := elem.Field(i)
		t := elem.Type().Field(i)
		field := t.Tag.Get("db")
		if t.IsExported() && !t.Anonymous && field != "" && !v.IsZero() {
			if fields == "" {
				fields = field
			} else {
				fields += "," + field
			}
			value := v.Interface()
			values = append(values, value)
		}
	}

	return s.AddRow(table, fields, values)
}

func (s *Database) AddRow(table string, fields string, values []any) *SqlResult {
	if fields == "" {
		panic("fields is empty")
	}
	if len(values) == 0 {
		panic("values is empty")
	}

	insert := NewSql(fmt.Sprintf("INSERT INTO %s(%s) VALUES(?)", table, fields), values)
	query, args := s.toSql(insert)

	res, err := s.DB.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return &SqlResult{res}
}

func (s *Database) Save(table string, row any, fn SaveFn) *SqlResult {
	if table == "" {
		panic("table is empty")
	}

	v, ok := row.(map[string]any)
	if ok {
		return s.SaveMap(table, v, fn)
	}

	vf := reflect.ValueOf(row)
	if vf.Kind() != reflect.Ptr {
		panic("row is not ptr")
	}

	if vf.Elem().NumField() == 0 {
		panic("row is zero")
	}

	return s.SaveStruct(table, row, fn)
}

func (s *Database) SaveMap(table string, row map[string]any, fn SaveFn) *SqlResult {
	var fields []string
	var values []any

	for field, value := range row {
		fields = append(fields, field)
		values = append(values, value)
	}

	where := NewSqlOptional("WHERE")
	if fn != nil {
		fn(where)
	}

	return s.SaveRow(table, fields, values, where)
}

func (s *Database) SaveStruct(table string, row any, fn SaveFn) *SqlResult {
	vf := reflect.ValueOf(row)
	var fields []string
	var values []any

	elem := vf.Elem()
	for i := 0; i < elem.NumField(); i++ {
		v := elem.Field(i)
		t := elem.Type().Field(i)
		field := t.Tag.Get("db")
		if t.IsExported() && !t.Anonymous && field != "" && !v.IsZero() {
			fields = append(fields, field)
			values = append(values, v.Interface())
		}
	}

	where := NewSqlOptional("WHERE")
	if fn != nil {
		fn(where)
	}

	return s.SaveRow(table, fields, values, where)
}

func (s *Database) SaveRow(table string, fields []string, values []any, where *Sql) *SqlResult {
	if len(fields) == 0 {
		panic("fields is empty")
	}
	if len(values) == 0 {
		panic("values is empty")
	}

	update := NewSql(fmt.Sprintf("UPDATE %s SET", table))

	for idx, field := range fields {
		if idx == 0 {
			update.Space(fmt.Sprintf("%s=?", field), values[idx])
		} else {
			update.Comma(fmt.Sprintf("%s=?", field), values[idx])
		}
	}

	query, args := s.toSql(NewSql("? ?", update, where))
	res, err := s.DB.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return &SqlResult{res}
}

func (s *Database) toSql(sql *Sql) (query string, args []any) {
	if DefaultDB == PGSQL {
		query, args = sql.ToPgsql()
	} else if DefaultDB == MYSQL {
		query, args = sql.ToMysql()
	} else {
		panic("error DefaultDB")
	}
	Print.Info(query)
	return
}

func (s *Database) Del(table string, fn DelFn) *SqlResult {
	if table == "" {
		panic("table is empty")
	}
	if fn == nil {
		panic("del fn is empty")
	}
	from := NewSql(fmt.Sprintf("DELETE FROM %s", table))
	where := NewSqlOptional("WHERE")
	fn(where)

	del := NewSql("? ?", from, where)
	query, args := s.toSql(del)
	res, err := s.DB.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return &SqlResult{res}
}

func (s *Database) querySql(table string, fn SelectFn) (string, []any) {
	sel := NewSqlOptional("SELECT")
	from := NewSql(fmt.Sprintf("FROM %s", table))
	where := NewSqlOptional("WHERE")

	if fn != nil {
		fn(sel, where)
	}

	if !sel.HasParts() {
		sel.Space("*")
	}

	return s.toSql(NewSql("? ? ?", sel, from, where))
}

func (s *Database) Select(table string, fn SelectFn) (rows Rows) {
	query, args := s.querySql(table, fn)
	res, err := s.DB.Queryx(query, args...)
	defer res.Close()
	if err != nil {
		panic(err)
	}

	cols, _ := res.Columns()
	colTypes, _ := res.ColumnTypes()

	for res.Next() {
		values, err := res.SliceScan()
		if err != nil {
			panic(err)
		}

		row := NewRow()
		for idx, col := range cols {
			row.put(col, &Field{values[idx], colTypes[idx].DatabaseTypeName()})
		}

		rows = append(rows, row)
	}

	return
}

func (s *Database) Get(table string, fn SelectFn) (row *Row) {
	query, args := s.querySql(table, fn)
	res := s.DB.QueryRowx(query, args...)

	cols, err := res.Columns()
	colTypes, err := res.ColumnTypes()
	values, err := res.SliceScan()

	if err == sql.ErrNoRows {
		return
	}

	if err != nil {
		panic(err)
	}

	if len(values) > 0 {
		row = NewRow()
		for idx, col := range cols {
			row.put(col, &Field{values[idx], colTypes[idx].DatabaseTypeName()})
		}
	}

	return
}

type Rows []*Row

var ErrRowEmpty = errors.New("db: row is empty")
var ErrFieldNotExist = errors.New("db: field not exist")

func NewRow() *Row {
	return &Row{linkedhashmap.New()}
}

type Row struct {
	m *linkedhashmap.Map
}

func (s *Row) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteRune('{')

	it := s.m.Iterator()
	lastIndex := s.m.Size() - 1
	index := 0

	for it.Next() {
		km, err := json.Marshal(it.Key())
		if err != nil {
			return nil, err
		}
		buf.Write(km)
		buf.WriteRune(':')

		vm, err := it.Value().(*Field).MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(vm)

		if index != lastIndex {
			buf.WriteRune(',')
		}

		index++
	}

	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (s *Row) Each(f func(key string, value *Field)) {
	if s == nil {
		panic(ErrRowEmpty)
	}

	iterator := s.m.Iterator()
	for iterator.Next() {
		f(iterator.Key().(string), iterator.Value().(*Field))
	}
}

func (s *Row) Map(f func(key1 string, value1 *Field) (string, *Field)) *Row {
	if s == nil {
		panic(ErrRowEmpty)
	}

	newRow := NewRow()
	iterator := s.m.Iterator()
	for iterator.Next() {
		key2, value2 := f(iterator.Key().(string), iterator.Value().(*Field))
		newRow.put(key2, value2)
	}
	return newRow
}

func (s *Row) put(key string, value *Field) {
	s.m.Put(key, value)
}

func (s *Row) Get(key string) *Field {
	if s == nil {
		panic(ErrRowEmpty)
	}

	val, found := s.m.Get(key)
	if !found {
		panic(ErrFieldNotExist)
	}
	return val.(*Field)
}

type Field struct {
	Val  any
	Type string
}

func (s *Field) String() string {
	return cast.ToString(s.Val)
}

func (s *Field) Int() int64 {
	return cast.ToInt64(s.String())
}

func (s *Field) Float() float64 {
	return cast.ToFloat64(s.String())
}

func (s *Field) Bool() bool {
	return cast.ToBool(s.String())
}

func (s *Field) IsNull() bool {
	return s.Val == nil
}

func (s *Field) NotNull() bool {
	return !s.IsNull()
}

func (s *Field) MarshalJSON() ([]byte, error) {
	var val interface{}

	if s.NotNull() {
		switch s.Type {
		case "DECIMAL", "FLOAT", "DOUBLE", "NUMERIC":
			val = s.Float()
		case "INTEGER", "INT", "SMALLINT", "TINYINT", "MEDIUMINT", "BIGINT",
			"INT2", "INT4", "INT8":
			val = s.Int()
		case "CHAR", "VARCHAR", "BINARY", "VARBINARY", "BLOB", "TEXT", "ENUM", "SET", "JSON":
			val = s.String()
		case "DATE", "TIME", "DATETIME", "TIMESTAMP", "YEAR",
			"TIMETZ", "TIMESTAMPTZ":
			val = s.String()
		}
	}

	return json.Marshal(val)
}

type SqlResult struct {
	res sql.Result
}

func (s *SqlResult) Ok() bool {
	if s == nil {
		return false
	}
	affNum, err := s.res.RowsAffected()
	if err != nil {
		panic(err)
	}
	return affNum > 0
}

func (s *SqlResult) Fail() bool {
	if s == nil {
		return true
	}
	affNum, err := s.res.RowsAffected()
	if err != nil {
		panic(err)
	}
	return affNum <= 0
}
