package lama

import (
	"encoding/json"
	"fmt"
	"strings"
)

type sqlPart struct {
	Sql    string
	Params []any
}

type Sql struct {
	parts    []*sqlPart
	optional string
}

func NewSql(sql string, args ...any) *Sql {
	return &Sql{parts: append([]*sqlPart{}, makeSqlPart(sql, args...))}
}

func NewSqlOptional(optional string) *Sql {
	return &Sql{
		optional: optional,
	}
}

func (s *Sql) HasParts() bool {
	return len(s.parts) > 0
}

func (s *Sql) Add(sep, sql string, args ...any) *Sql {
	if len(s.parts) > 0 {
		s.parts = append(s.parts, makeSqlPart(sep+sql, args...))
	} else {
		s.parts = append(s.parts, makeSqlPart(sql, args...))
	}
	return s
}

func (s *Sql) Or(sql string, args ...any) *Sql {
	return s.Add(" OR ", sql, args...)
}

func (s *Sql) And(sql string, args ...any) *Sql {
	return s.Add(" AND ", sql, args...)
}

func (s *Sql) Space(sql string, args ...any) *Sql {
	return s.Add(" ", sql, args...)
}

func (s *Sql) Comma(sql string, args ...any) *Sql {
	return s.Add(",", sql, args...)
}

func (s *Sql) Concat(sql string, args ...any) *Sql {
	return s.Add("", sql, args...)
}

func (s *Sql) ToMysql() (sql string, params []any) {
	sql, params = s.toSql()
	sql = s.replace(MYSQL, sql, params)
	return
}

func (s *Sql) ToPgsql() (sql string, params []any) {
	sql, params = s.toSql()
	sql = s.replace(PGSQL, sql, params)
	return
}

func (s *Sql) ToRaw() string {
	sql, params := s.toSql()
	return s.replace(RAW, sql, params)
}

func (s *Sql) toSql() (string, []any) {
	var sql string
	var params []any

	if s.optional != "" && len(s.parts) > 0 {
		sql = s.optional + " "
	}

	for _, p := range s.parts {
		sql += p.Sql
		params = append(params, p.Params...)
	}

	return strings.TrimSpace(sql), params
}

func (s *Sql) Len() int {
	return len(s.parts)
}

func (s *Sql) Print() {
	sql, params := s.toSql()
	Print.Printf("SQL: %v\n", sql)
	Print.Printf("PARAMS: %v\n", params)
}

const (
	PGSQL   SqlType = "postgres"
	MYSQL   SqlType = "mysql"
	RAW     SqlType = "raw"
	paramPh         = "{{xX_PARAM_Xx}}"
)

type SqlType string
type JsonList []any
type JsonMap map[string]any

func (s *Sql) replace(sqlType SqlType, sql string, params []any) string {
	for i, param := range params {
		if sqlType == RAW {
			p := paramToRaw(param)
			sql = strings.Replace(sql, paramPh, p, 1)

		} else if sqlType == MYSQL {
			sql = strings.Replace(sql, paramPh, "?", 1)

		} else if sqlType == PGSQL {
			sql = strings.ReplaceAll(sql, "??", "?")
			sql = strings.Replace(sql, paramPh, fmt.Sprintf("$%d", i+1), 1)
		}
	}
	return sql
}

func makeSqlPart(text string, args ...any) *sqlPart {
	tempPh := "XXX___XXX"
	originalText := text
	text = strings.ReplaceAll(text, "??", tempPh)

	var newArgs []interface{}

	for _, arg := range args {
		switch v := arg.(type) {

		case []int:
			newPh := []string{}
			for _, i := range v {
				newPh = append(newPh, paramPh)
				newArgs = append(newArgs, i)
			}
			text = strings.Replace(text, "?", strings.Join(newPh, ","), 1)

		case []*int:
			newPh := []string{}
			for _, i := range v {
				newPh = append(newPh, paramPh)
				newArgs = append(newArgs, i)
			}
			if len(newPh) > 0 {
				text = strings.Replace(text, "?", strings.Join(newPh, ","), 1)
			} else {
				text = strings.Replace(text, "?", paramPh, 1)
				newArgs = append(newArgs, nil)
			}

		case []string:
			newPh := []string{}
			for _, s := range v {
				newPh = append(newPh, paramPh)
				newArgs = append(newArgs, s)
			}
			text = strings.Replace(text, "?", strings.Join(newPh, ","), 1)

		case []*string:
			newPh := []string{}
			for _, s := range v {
				newPh = append(newPh, paramPh)
				newArgs = append(newArgs, s)
			}
			if len(newPh) > 0 {
				text = strings.Replace(text, "?", strings.Join(newPh, ","), 1)
			} else {
				text = strings.Replace(text, "?", paramPh, 1)
				newArgs = append(newArgs, nil)
			}

		case []interface{}:
			newPh := []string{}
			for _, s := range v {
				newPh = append(newPh, paramPh)
				newArgs = append(newArgs, s)
			}
			text = strings.Replace(text, "?", strings.Join(newPh, ","), 1)

		case *Sql:
			if v == nil {
				text = strings.Replace(text, "?", paramPh, 1)
				newArgs = append(newArgs, nil)
				continue
			}
			sql, params := v.toSql()
			text = strings.Replace(text, "?", sql, 1)
			newArgs = append(newArgs, params...)

		case JsonMap, JsonList:
			bytes, err := json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("cann jsonify struct: %v", err))
			}
			text = strings.Replace(text, "?", paramPh, 1)
			newArgs = append(newArgs, string(bytes))

		case *JsonMap, *JsonList:
			bytes, err := json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("cann jsonify struct: %v", err))
			}
			text = strings.Replace(text, "?", paramPh, 1)
			newArgs = append(newArgs, string(bytes))

		default:
			text = strings.Replace(text, "?", paramPh, 1)
			newArgs = append(newArgs, v)
		}
	}
	extraCount := strings.Count(text, "?")
	if extraCount > 0 {
		panic(fmt.Sprintf("extra ? in text: %v (%d args)", originalText, len(newArgs)))
	}

	paramCount := strings.Count(text, paramPh)
	if paramCount < len(newArgs) {
		panic(fmt.Sprintf("missing ? in text: %v (%d args)", originalText, len(newArgs)))
	}

	text = strings.ReplaceAll(text, tempPh, "??")

	return &sqlPart{
		Sql:    text,
		Params: newArgs,
	}
}

func paramToRaw(param interface{}) string {
	switch p := param.(type) {
	case bool:
		return fmt.Sprintf("%v", p)

	case float32, float64, int, int8, int16, int32, int64,
		uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", p)

	case *int:
		if p == nil {
			return "NULL"
		}
		return fmt.Sprintf("%v", *p)

	case string:
		return fmt.Sprintf("'%v'", p)

	case *string:
		if p == nil {
			return "NULL"
		}
		return fmt.Sprintf("'%v'", *p)

	case nil:
		return "NULL"

	default:
		panic(fmt.Errorf("unsupported type for Raw query: %T", p))
	}
}
