package orm

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	
	"github.com/leicc520/go-orm/log"
)

const (
	DT_SQL        = "sql"
	DT_AUTO       = "auto"
	OP_AS         = "AS"
	OP_MAX        = "MAX"
	OP_MIN        = "MIN"
	OP_SUM        = "SUM"
	OP_AVG        = "AVG"
	OP_COUNT      = "COUNT"
	OP_EQ         = "="
	OP_NE         = "<>"
	OP_GT         = ">"
	OP_LT         = "<"
	OP_GE         = ">="
	OP_LE         = "<="
	OP_FULLTEXT   = "AGAINST"
	OP_BETWEEN    = "BETWEEN"
	OP_NOTBETWEEN = "NOT BETWEEN"
	OP_LIKE       = "LIKE"
	OP_NOTLIKE    = "NOT LIKE"
	OP_REGEXP     = "REGEXP"
	OP_ISNULL     = "IS NULL"
	OP_ISNOTNULL  = "IS NOT NULL"
	OP_IN         = "IN"
	OP_NOTIN      = "NOT IN"
	OP_AND        = "AND"
	OP_OR         = "OR"
	OP_NOT        = "NOT"
	OP_SQL        = "SQL"
	ASC           = "ASC"
	DESC          = "DESC"
	POSTGRES      = "postgres"
)

type tableSt struct {
	name  string
	alias string
	on    string
}

type valueSt struct {
	name  string
	value interface{}
	ftype string
}

type whereSt struct {
	name    string
	value   interface{}
	opt     string
	ftype   string
	logical string
}

//语句的组成的结构
type mysqlSt struct {
	index int64
	driver string
	sql map[string]interface{}
	marks []interface{}
}

//初始化一个语句结构体对象
func NewMysql() *mysqlSt {
	mySql := (new(mysqlSt)).Reset()
	return mySql
}

//设置数据库的的驱动引擎
func (q *mysqlSt) SetDriver(driver string) *mysqlSt {
	q.driver = driver
	return q
}

//获取数据库设置的驱动引擎
func (q *mysqlSt) GetDriver() string {
	return q.driver
}

//结构体初始化内部结构
func (q *mysqlSt) Reset() *mysqlSt {
	q.index = 1 //重置绑定参数
	q.sql   = make(map[string]interface{})
	q.marks = make([]interface{}, 0)
	return q
}

//清除语句的部分信息
func (q *mysqlSt) Clear(parts ...string) *mysqlSt {
	if len(parts) > 0 {
		for _, part := range parts {
			if _, ok := q.sql[part]; ok {
				delete(q.sql, part)
			}
		}
	} else {
		for key, _ := range q.sql {
			delete(q.sql, key)
		}
	}
	q.index = 1
	q.marks = make([]interface{}, 0)
	return q
}

//设置查询的表信息
func (q *mysqlSt) Table(table ...string) *mysqlSt {
	var tables []tableSt
	if _, ok := q.sql["table"]; !ok {
		tables = make([]tableSt, 0)
	} else {
		tables = q.sql["table"].([]tableSt)
	}
	Alen, alias, on := len(table), "", ""
	if Alen == 3 {
		alias, on = table[1], table[2]
	} else if Alen == 2 {
		alias = table[1]
	}
	tables = append(tables, tableSt{name: table[0], alias: alias, on: on})
	q.sql["table"] = tables
	return q
}

//设置要查询的字段信息
func (q *mysqlSt) Field(fields string) *mysqlSt {
	q.sql["field"] = fields
	return q
}

//设置要冲突字段信息
func (q *mysqlSt) ConflictField(field string) *mysqlSt {
	q.sql["conflict"] = field
	return q
}

//设置GroupBy分组
func (q *mysqlSt) GroupBy(fields ...string) *mysqlSt {
	var groups []string
	if _, ok := q.sql["groupby"]; !ok {
		groups = make([]string, 0)
	} else {
		groups = q.sql["groupby"].([]string)
	}
	groups = append(groups, fields...)
	q.sql["groupby"] = groups
	return q
}

//设置GroupBy Having;一条语句只能设置一次
func (q *mysqlSt) Having(having string) *mysqlSt {
	q.sql["having"] = having
	return q
}

//设置GroupBy Having;一条语句只能设置一次
func (q *mysqlSt) ForceIndex(index string) *mysqlSt {
	q.sql["index"] = " FORCE INDEX("+index+")"
	return q
}

//设置排序信息 允许多列排序
func (q *mysqlSt) OrderBy(field, dir string) *mysqlSt {
	var orderby []string
	if _, ok := q.sql["orderby"]; !ok {
		orderby = make([]string, 0)
	} else {
		orderby = q.sql["orderby"].([]string)
	}
	orderby = append(orderby, field+" "+dir)
	q.sql["orderby"] = orderby
	return q
}

//更新字段值设置
func (q *mysqlSt) Value(field string, value interface{}, args ...string) *mysqlSt {
	var values []valueSt
	if _, ok := q.sql["value"]; !ok {
		values = make([]valueSt, 0)
	} else {
		values = q.sql["value"].([]valueSt)
	}
	ftype := DT_AUTO
	if args != nil && len(args) > 0 {
		ftype = args[0]
	}
	values = append(values, valueSt{name: field, value: value, ftype: ftype})
	q.sql["value"] = values
	return q
}

//更新字段值设置
func (q *mysqlSt) Duplicate(field string, value interface{}, args ...string) *mysqlSt {
	var values []valueSt
	if _, ok := q.sql["duplicate"]; !ok {
		values = make([]valueSt, 0)
	} else {
		values = q.sql["duplicate"].([]valueSt)
	}
	ftype := DT_AUTO
	if args != nil && len(args) > 0 {
		ftype = args[0]
	}
	values = append(values, valueSt{name: field, value: value, ftype: ftype})
	q.sql["duplicate"] = values
	return q
}

//批量请求参数的处理业务逻辑
func (self *mysqlSt) UseCond(fields []string, value interface{}, options... string) *mysqlSt {
	elemSt := reflect.ValueOf(value).Elem()
	data   := SqlMap{}
	for _, field := range fields {
		field  = CamelCase(field) //转换成驼峰模式
		valSt := elemSt.FieldByName(field)
		if !valSt.IsValid() || valSt.IsZero() {
			continue
		}
		data[field] = valSt.Interface()
	}
	if len(data) == 1 { //大于0个字段的情况
		for field, valStr := range data {
			self.Where(field, valStr, options...)
		}
	} else if len(data) > 1 {
		self.Where("(", "")
		for field, valStr := range data {
			self.Where(field, valStr, options...)
		}
		self.Where(")", "")
	}
	return self
}

//批量请求参数的处理业务逻辑 相同字段隐射到同一个东西
func (self *mysqlSt) UseBatch(fields []string, value interface{}, options... string) *mysqlSt {
	if len(fields) == 1 { //大于0个字段的情况
		for _, field := range fields {
			self.Where(field, value, options...)
		}
	} else if len(fields) > 1 {
		self.Where("(", "")
		for _, field := range fields {
			self.Where(field, value, options...)
		}
		self.Where(")", "")
	}
	return self
}

//添加条件配置
func (q *mysqlSt) Where(field string, value interface{}, args ...string) *mysqlSt {
	var wheres []whereSt
	if _, ok := q.sql["where"]; !ok {
		wheres = make([]whereSt, 0)
	} else {
		wheres = q.sql["where"].([]whereSt)
	}
	opt, ftype, logical, Alen := OP_EQ, DT_AUTO, OP_AND, len(args)
	if Alen == 1 {
		switch args[0] {
		case OP_AND, OP_OR, OP_NOT:
			logical = args[0]
		default:
			opt = args[0]
		}
	} else if Alen == 2 {
		opt = args[0]
		switch args[1] {
		case OP_AND, OP_OR, OP_NOT:
			logical = args[1]
		default:
			ftype = args[1]
		}
	} else if Alen == 3 {
		opt, ftype, logical = args[0], args[1], args[2]
	}
	if opt == OP_LIKE {//针对like的情况做特殊逻辑处理
		tmpStr := fmt.Sprintf("%v", value)
		if !strings.HasPrefix(tmpStr, "%") && !strings.HasSuffix(tmpStr, "%") {
			value = "%"+tmpStr+"%" //处理like的业务请求
		}
	}
	wheres = append(wheres, whereSt{name: field, value: value, opt: opt, ftype: ftype, logical: logical})
	q.sql["where"] = wheres
	return q
}

//获取查询的where数据信息
func (q *mysqlSt) GetWheres() string {
	var wheres []whereSt
	if _, ok := q.sql["where"]; !ok {
		wheres = make([]whereSt, 0)
	} else {
		wheres = q.sql["where"].([]whereSt)
	}
	astr := make([]string, 0)
	for _, sitem := range wheres {
		astr = append(astr, sitem.name)
		astr = append(astr, sitem.opt)
		astr = append(astr, sitem.ftype)
		astr = append(astr, sitem.logical)
		if str, ok := sitem.value.(string); ok {
			astr = append(astr, str)
		} else {
			astr = append(astr, fmt.Sprintf("%v", sitem.value))
		}
	}
	if _, ok := q.sql["orderby"]; ok {//添加排序 否则缓存不对
		items, _ := q.sql["orderby"].([]string)
		astr = append(astr, strings.Join(items, ","))
	}
	if _, ok := q.sql["having"]; ok {//添加分组条件
		items, _ := q.sql["orderby"].(string)
		astr = append(astr, items)
	}
	if _, ok := q.sql["groupby"]; ok {//添加分组处理
		items, _ := q.sql["groupby"].([]string)
		astr = append(astr, strings.Join(items, ","))
	}
	return strings.Join(astr, "")
}

//解析表结构信息
func (q *mysqlSt) sqlTable(isalias bool) string {
	sql := ""
	tables:= q.sql["table"].([]tableSt)
	for _, table := range tables {
		if table.alias != "" && isalias {
			table.name = fmt.Sprintf("%s %s %s", table.name, OP_AS, table.alias)
		}
		if table.on != "" {
			sql += fmt.Sprintf(" LEFT JOIN %s ON (%s)", table.name, table.on)
		} else if sql == "" {
			sql  = table.name
		} else {
			sql += fmt.Sprintf(" ,%s", table.name)
		}
	}
	return sql
}

//获取绑定变量的处理逻辑
func (q *mysqlSt) sqlBindSign() string {
	signer := "?"
	if q.driver == POSTGRES {
		signer = fmt.Sprintf("$%d", q.index)
		q.index++
	}
	return signer
}

//适配不同的数据库引擎适配offset/limit
func (q *mysqlSt) sqlOffsetLimit(offset, limit int64) string {
	if q.driver == POSTGRES {
		return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	} else {
		return fmt.Sprintf(" LIMIT %d, %d", offset, limit)
	}
}

//获取字段使用引号处理逻辑
func (q *mysqlSt) sqlFieldQuotes(field string) string {
	ok, err := regexp.MatchString("^[\\da-zA-Z_]+$", field)
	if !ok || err != nil {//带有特殊字符的直接跳过
		return field
	}
	if q.driver == POSTGRES {
		return fmt.Sprintf("\"%s\"", field)
	}
	return fmt.Sprintf("`%s`", field)
}

//解析Update设置值语句
func (q *mysqlSt) sqlUpdateValue(values []valueSt) string {
	sliceSql:= make([]string, len(values))
	for idx, val := range values {
		if val.ftype == DT_SQL || val.ftype == OP_SQL {
			sliceSql[idx] = fmt.Sprintf("%s = (%s)", q.sqlFieldQuotes(val.name), val.value)
		} else {
			sliceSql[idx] = fmt.Sprintf("%s = %s", q.sqlFieldQuotes(val.name), q.sqlBindSign())
			q.marks = append(q.marks, val.value)
		}
	}
	return strings.Join(sliceSql, ",")
}

//解析插入Insert语句的设置值语句
func (q *mysqlSt) sqlInsertValue() string {
	values := q.sql["value"].([]valueSt)
	valstr := make([]string, len(values))
	fields := make([]string, len(values))
	for idx, val := range values {
		fields[idx] = q.sqlFieldQuotes(val.name)
		valstr[idx] = q.sqlBindSign()
		q.marks  = append(q.marks, val.value)
	}
	return fmt.Sprintf("(%s)VALUES(%s)", strings.Join(fields, ","), strings.Join(valstr, ","))
}

//执行in查询占位符的处理逻辑
func (q *mysqlSt) sliceIn(where *whereSt, list []interface{}) string {
	arrPos := make([]string, len(list))
	for idx, val := range list {
		arrPos[idx] = q.sqlBindSign()
		q.marks = append(q.marks, val)
	}
	return fmt.Sprintf("%s %s (%s)", q.sqlFieldQuotes(where.name), where.opt, strings.Join(arrPos, ","))
}

//解析Where条件设置语句
func (q *mysqlSt) sqlWhere() string {
	whereSql := ""
	if _, ok := q.sql["where"]; !ok {
		return whereSql
	}
	//开始解析
	wheres := q.sql["where"].([]whereSt)
	conds  := make([]string, 0)
	logical := false
	for _, where := range wheres {
		upName := strings.ToUpper(where.name)
		if (logical && where.name != ")" && upName != OP_OR) || where.logical == OP_NOT {
			conds = append(conds, fmt.Sprintf("%s ", where.logical))
		}
		if where.name == "(" || where.name == ")" {
			conds = append(conds, where.name)
			logical = where.name != "("
		} else if upName == OP_OR {
			conds = append(conds, where.name)
			logical = false
		} else {
			switch where.opt {
			case OP_BETWEEN, OP_NOTBETWEEN:
				if val, ok := where.value.(string); ok {
					vals := strings.SplitN(val, ",", 2)
					if len(vals) == 2 {
						conds   = append(conds, fmt.Sprintf("(%s %s %s AND %s)",
							q.sqlFieldQuotes(where.name), where.opt, q.sqlBindSign(), q.sqlBindSign()))
						q.marks = append(q.marks, vals[0], vals[1])
					}
				}
			case OP_IN, OP_NOTIN:
				if val, ok := where.value.(string); ok {
					conds   = append(conds, fmt.Sprintf("%s %s (%s)", q.sqlFieldQuotes(where.name), where.opt, val))
				} else if val, ok := where.value.([]interface{}); ok {
					conds   = append(conds, q.sliceIn(&where, val))
				} else {
					valueOf := reflect.ValueOf(where.value)
					if valueOf.Kind() == reflect.Slice && !valueOf.IsNil() {
						valSlice := make([]interface{}, valueOf.Len())
						for idx := 0; idx < valueOf.Len(); idx++ {
							valSlice[idx] = valueOf.Index(idx).Interface()
						}
						conds   = append(conds, q.sliceIn(&where, valSlice))
					} else {
						panic("sql query in params error")
					}
				}
			case OP_REGEXP:
				if val, ok := where.value.(string); ok {
					conds   = append(conds, fmt.Sprintf("%s %s %s", q.sqlFieldQuotes(where.name), where.opt, q.sqlBindSign()))
					q.marks = append(q.marks, val)
				}
			case OP_FULLTEXT:
				if val, ok := where.value.(string); ok {
					conds   = append(conds, fmt.Sprintf("MATCH(%s) against(%s)", where.name, q.sqlBindSign()))
					q.marks = append(q.marks, val)
				}
			case OP_SQL:
				conds = append(conds, fmt.Sprintf("%s", where.name))
			case OP_ISNULL, OP_ISNOTNULL:
				conds = append(conds, fmt.Sprintf("%s %s", q.sqlFieldQuotes(where.name), where.opt))
			default:
				if where.ftype == DT_SQL {
					conds = append(conds, fmt.Sprintf("%s %s (%s)", q.sqlFieldQuotes(where.name), where.opt, where.value))
				} else {
					conds = append(conds, fmt.Sprintf("%s %s %s", q.sqlFieldQuotes(where.name), where.opt, q.sqlBindSign()))
					q.marks = append(q.marks, where.value)
				}
			}
			logical = true
		}
	}
	if len(conds) > 0 {
		whereSql = " WHERE " + strings.Join(conds, " ")
	}
	return whereSql
}

//设定插入的时候执行replace into
func (q *mysqlSt) SetIsReplace(isReplace bool) *mysqlSt {
	q.sql["is_replace"] = isReplace
	return q
}

//获取语句最终拼凑的SQL语句
func (q *mysqlSt) AsSql(mode string) string {
	q.index = 1
	q.marks = make([]interface{}, 0)
	query  := ""
	switch strings.ToLower(mode) {
	case "select":
		fields, ok := q.sql["field"].(string)
		if !ok || fields == "" {
			fields = "*"
		}
		index, ok := q.sql["index"].(string)
		if !ok && fields != "" {//
			index = ""
		}
		query = "SELECT " + fields + " FROM " + q.sqlTable(true) + index + q.sqlWhere()
		if _, ok := q.sql["groupby"]; ok {
			groupby, _ := q.sql["groupby"].([]string)
			if len(groupby) > 0 {
				query += " GROUP BY " + strings.Join(groupby, ",")
			}
			if _, ok := q.sql["having"]; ok {
				having, _ := q.sql["having"].(string)
				if having != "" {
					query += " HAVING " + having
				}
			}
		}
		//解析排序语句
		if _, ok := q.sql["orderby"]; ok {
			orderby, _ := q.sql["orderby"].([]string)
			if len(orderby) > 0 {
				query += "  ORDER BY " + strings.Join(orderby, ",")
			}
		}
	case "update":
		values  := q.sql["value"].([]valueSt)
		query = "UPDATE " + q.sqlTable(false) + " SET " +
			q.sqlUpdateValue(values) + q.sqlWhere()
	case "insert":
		query = "INSERT "
		if tmp, ok := q.sql["is_replace"]; ok && tmp.(bool) {
			query = "REPLACE "
		}
		query+= "INTO " + q.sqlTable(false) + q.sqlInsertValue()
		if _, ok := q.sql["duplicate"]; ok {
			values := q.sql["duplicate"].([]valueSt)
			query  += q.sqlDuplicateUpdate(values)
		}
	case "delete":
		query = "DELETE FROM " + q.sqlTable(false) + q.sqlWhere()
	}
	if IsShowSql {
		log.Write(log.DEBUG, q.RepSQL(query))
	}
	return query
}

//出现唯一性约束失败的情况 执行更新操作
func (q *mysqlSt) sqlDuplicateUpdate(values []valueSt) string {
	if q.driver != POSTGRES {
		return " ON DUPLICATE KEY UPDATE "+q.sqlUpdateValue(values)
	} else {
		field, _ := q.sql["conflict"].(string)
		return " ON CONFLICT ("+q.sqlFieldQuotes(field)+") DO UPDATE SET "+q.sqlUpdateValue(values)
	}
}

//获取sql的输出出来逻辑
func (q *mysqlSt) repSQLPrepare(sql string) string {
	reg, err := regexp.Compile("\\$[\\d]+")
	if err == nil && reg != nil {
		sql = reg.ReplaceAllString(sql, "?")
	}
	return sql
}

//替换绑定变量到SQL当中处理
func (q *mysqlSt) RepSQL(sql string) string {
	if q.driver == POSTGRES {
		sql = q.repSQLPrepare(sql)
	}
	for _, mark := range q.marks {
		lStr, ok := mark.(string)
		if !ok {
			lStr = fmt.Sprint(mark)
		}
		lStr = strings.ReplaceAll(lStr,"?", "¤")
		sql  = strings.Replace(sql, "?", lStr, 1)
	}
	if strings.Index(sql, "¤") != -1 {
		sql = strings.ReplaceAll(sql, "¤", "?")
	}
	return sql
}
