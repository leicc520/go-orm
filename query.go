package orm

import (
	"database/sql"
	"reflect"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/leicc520/go-orm/log"
)


type FormatItemHandle func(sm SqlMap)
type TransactionHandle func(st *QuerySt) bool

type QuerySt struct {
	*mysqlSt
	format   FormatItemHandle
	autoIncr string
	fields   map[string]reflect.Kind
	sqlDb    *sqlx.DB
	sqlTx    *sqlx.Tx
}

//定义注册到条件的数据资料信息
type WHandler func(st *QuerySt) string
type VHandler func(st *QuerySt) *QuerySt
type DHandler func(st *QuerySt) interface{}
type CHandler func([]SqlMap) error //执行业务分页的执行业务逻辑

//生成一个查询Session
func NewQuery(fields map[string]reflect.Kind) *QuerySt {
	query := &QuerySt{&mysqlSt{}, nil, "", fields, nil, nil}
	query.Reset()
	return query
}

//主要用作处理数据的格式化逻辑
func (q *QuerySt) SetFormat(handle FormatItemHandle) {
	q.format = handle
}

//主要用作处理数据的格式化逻辑
func (q *QuerySt) GetFormat() FormatItemHandle {
	return q.format
}

//主要用作自增字段，插入返回自增ID
func (q *QuerySt) SetAutoIncr(field string) *QuerySt {
	q.autoIncr = field
	return q
}

//获取当前执行的查询DB信息 需要及时释放，否则有问题
func (q *QuerySt) GetDb() *sqlx.DB {
	return q.sqlDb
}

//获取当前执行的查询DB信息 需要及时释放，否则有问题
func (q *QuerySt) SetDb(db *sqlx.DB) *QuerySt {
	q.sqlDb = db
	return q
}

//获取当前执行的查询DB信息 需要及时释放，否则有问题
func (q *QuerySt) GetTx() *sqlx.Tx {
	return q.sqlTx
}

//获取当前执行的查询DB信息 需要及时释放，否则有问题
func (q *QuerySt) SetTx(tx *sqlx.Tx) *QuerySt {
	q.sqlTx = tx
	return q
}

//执行事务处理的业务逻辑封装
func (q *QuerySt) Transaction(handle TransactionHandle) bool {
	var err error = nil
	q.sqlTx, err = q.GetDb().Beginx()
	if err != nil {
		log.Write(log.ERROR, "db 事务获取出错 ", err)
		return false
	}
	defer func() {
		q.sqlTx = nil
	}()
	if !handle(q) {
		if err = q.sqlTx.Rollback(); err != nil {
			log.Write(log.ERROR, "db 事务回滚出错", err)
		}
		return false
	}
	if err = q.sqlTx.Commit(); err != nil {
		log.Write(log.ERROR, "db 事务提交出错", err)
		return false
	}
	return true
}

//获取当前执行的查询DB信息
func (q *QuerySt) CloseDB() {
	if q.sqlDb != nil {
		q.sqlDb.Close()
		q.sqlDb = nil
	}
}

//执行一条SQL语句
func (q *QuerySt) Exec(query string) sql.Result {
	var stmt *sql.Stmt = nil
	var err error = nil
	if q.sqlTx != nil {//如果开启了事务的情况
		stmt, err = q.sqlTx.Prepare(query)
	} else {
		stmt, err = q.sqlDb.Prepare(query)
	}
	if err != nil {
		log.Write(log.ERROR, "exec sql prepare:"+query+" failed "+err.Error())
		return nil
	}
	defer stmt.Close()
	result, err := stmt.Exec(q.marks...)
	if err != nil {
		log.Write(log.ERROR, "exec sql:"+query+" error "+err.Error())
		return nil
	}
	return result
}

//执行一条SQL语句
func (q *QuerySt) queryRow(query string) *sql.Row {
	var stmt *sql.Stmt = nil
	var err error = nil
	if q.sqlTx != nil {//如果开启了事务的情况
		stmt, err = q.sqlTx.Prepare(query)
	} else {
		stmt, err = q.sqlDb.Prepare(query)
	}
	if err != nil {
		log.Write(log.ERROR, "exec sql prepare:"+query+" failed "+err.Error())
		return nil
	}
	defer stmt.Close()
	result := stmt.QueryRow(q.marks...)
	return result
}

//执行数据插入
func (q *QuerySt) Insert(fields SqlMap, isReplace bool) int64 {
	q.SetIsReplace(isReplace) //替换插入数据
	if fields != nil && len(fields) > 0 {
		for field, value := range fields {
			q.Value(field, value, DT_AUTO)
		}
	}
	query := q.AsSql("insert")
	return q.QueryInsert(query) //查询插入数据库
}

//查询插入数据库-直接执行sql插入处理逻辑
func (q *QuerySt) QueryInsert(query string) int64 {
	var lastId int64 = -1
	if q.GetDriver() == POSTGRES {//如果是pg数据库的情况
		if len(q.autoIncr) > 0 {//设置返回自增ID
			query += " RETURNING "+q.autoIncr
			if result := q.queryRow(query); result != nil {
				if err := result.Scan(&lastId); err != nil {
					log.Write(log.ERROR, err, query)
					lastId = -1
				}
			}
		} else {//普通的插入 不用返回自增ID
			if result := q.Exec(query); result != nil {
				lastId, _ = result.RowsAffected()
			}
		}
	} else {//mysql的执行逻辑
		if result := q.Exec(query); result != nil {
			lastId, _ = result.LastInsertId()
		}
	}
	return lastId
}

//执行数据更新操作
func (q *QuerySt) Update(fields SqlMap) int64 {
	if fields != nil && len(fields) > 0 {
		for field, value := range fields {
			if opSql, ok := value.([]interface{}); ok && len(opSql) == 2 {
				q.Value(field, opSql[0], opSql[1].(string))
			} else { //非数组的情况
				q.Value(field, value, DT_AUTO)
			}
		}
	}
	query := q.AsSql("update")
	if result := q.Exec(query); result == nil {
		return -1
	} else {
		rows, _ := result.RowsAffected()
		return rows
	}
}

//执行数据删除操作
func (q *QuerySt) Delete() int64 {
	query := q.AsSql("delete")
	if result := q.Exec(query); result == nil {
		return -1
	} else {
		rows, _ := result.RowsAffected()
		return rows
	}
}

//执行一次SQL查询
func (q *QuerySt) query(query string) *sql.Rows {
	var rows *sql.Rows = nil
	var err error = nil
	if q.sqlTx != nil {//如果开启了事务的情况
		rows, err = q.sqlTx.Query(query, q.marks...)
	} else {
		rows, err = q.sqlDb.Query(query, q.marks...)
	}
	if err != nil {
		log.Write(log.ERROR, "query sql:"+query+" error "+err.Error())
		return nil
	}
	return rows
}

//获取数据信息到数组中
func (q *QuerySt) fetch(rows *sql.Rows, isFirst bool) []SqlMap {
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	columns, _ := rows.Columns()
	list := make([]SqlMap, 0)
	values := make([]sql.RawBytes, len(columns))
	rowSlice := make([]interface{}, len(columns))
	for i := range values {
		rowSlice[i] = &values[i]
	}
	//遍历每个表的字段信息
	for rows.Next() {
		rows.Scan(rowSlice...)
		item := make(SqlMap)
		for idx, col := range values {
			if col == nil {
				col = sql.RawBytes{}
			}
			item[columns[idx]] = q.convertItem(columns[idx], col)
		}
		//完成额外的格式化操作 例如数据拼凑等
		if q.format != nil {
			q.format(item)
		}
		list = append(list, item)
		if isFirst {
			break
		}
	}
	return list
}

//对查询的字段进行字符转义
func (q *QuerySt) convertItem(fieldName string, value sql.RawBytes) interface{} {
	str := string(value)
	if q.fields == nil {//默认直接转字符串
		return str
	}
	dtType, ok := q.fields[fieldName]
	if !ok { //不存在的情况直接转码
		return str
	}
	var err error = nil
	switch dtType {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		data := uint64(0)
		if len(str) > 0 {
			data, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				log.Write(log.ERROR, "orm convertItem strconv.ParseUint error "+err.Error())
				return str
			}
		}
		if dtType == reflect.Uint {
			return uint(data)
		} else if dtType == reflect.Uint8 {
			return uint8(data)
		} else if dtType == reflect.Uint16 {
			return uint16(data)
		} else if dtType == reflect.Uint32 {
			return uint32(data)
		} else {//默认长整数 直接返回字符串类别
			return strconv.FormatUint(data, 10)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		data := int64(0)
		if len(str) > 0 {
			data, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				log.Write(log.ERROR, "orm convertItem strconv.ParseInt error "+err.Error())
				return str
			}
		}
		if dtType == reflect.Int {
			return int(data)
		} else if dtType == reflect.Int8 {
			return int8(data)
		} else if dtType == reflect.Int16 {
			return int16(data)
		} else if dtType == reflect.Int32 {
			return int32(data)
		} else {//直接返回字符串类别
			return strconv.FormatInt(data, 10)
		}
	case reflect.Float64, reflect.Float32:
		data := float64(0)
		if len(str) > 0 {
			data, err = strconv.ParseFloat(str, 64)
			if err != nil {
				log.Write(log.ERROR, "orm convertItem strconv.ParseFloat error "+err.Error())
				return str
			}
		}
		if dtType == reflect.Float32 {
			return float32(data)
		}
		return data
	}
	return str
}

//如果查不到记录返回nil
func (q *QuerySt) GetRow(query string) SqlMap {
	if query == "" {//为空的情况
		query = q.AsSql("select")
	}
	query   += " LIMIT 1"
	if rows := q.query(query); rows != nil {
		list := q.fetch(rows, true)
		if len(list) >= 1 {
			return list[0]
		}
	}
	return nil
}

//获取数据信息列表
func (q *QuerySt) GetList(query string, offset, limit int64) []SqlMap {
	if query == "" {//为空的情况
		query = q.AsSql("select")
	}
	if limit != -1 {
		query += q.sqlOffsetLimit(offset, limit)
	}
	if rows := q.query(query); rows != nil {
		list := q.fetch(rows, false)
		return list
	}
	return nil
}

//通过sql语句查询
func (q *QuerySt) GetAsSql(query string, isFirst bool, offset, limit int64) []SqlMap {
	if limit != -1 {
		query += q.sqlOffsetLimit(offset, limit)
	}
	if rows := q.query(query); rows != nil {
		list := q.fetch(rows, isFirst)
		return list
	}
	return nil
}

//获取单列信息
func (q *QuerySt) GetColumn(query string, offset, limit int64) []string {
	if query == "" {//记录为空的情况
		query = q.AsSql("select")
	}
	if limit != -1 {
		query += q.sqlOffsetLimit(offset, limit)
	}
	column := make([]string, 0)
	if rows := q.query(query); rows != nil {
		defer rows.Close()
		for rows.Next() {
			sliceValue := make(sql.RawBytes, 0)
			rows.Scan(&sliceValue)
			column = append(column, string(sliceValue))
		}
	}
	return column
}

//获取单列信息 请求设置field 必须 `key`,val结构
func (q *QuerySt) GetMap(query string, offset, limit int64) SqlMap {
	if query == "" {//记录为空的情况
		query = q.AsSql("select")
	}
	if limit != -1 {
		query += q.sqlOffsetLimit(offset, limit)
	}
	data := make(SqlMap, 0)
	if rows := q.query(query); rows != nil {
		defer rows.Close()
		for rows.Next() {
			sliceKey := make(sql.RawBytes, 0)
			sliceValue := make(sql.RawBytes, 0)
			rows.Scan(&sliceKey, &sliceValue)
			data[string(sliceKey)] = string(sliceValue)
		}
	}
	return data
}

//获取数据信息到数组中
func (q *QuerySt) NameMap(query, key string, offset, limit int64) map[string]SqlMap {
	if query == "" {//记录为空的情况
		query = q.AsSql("select")
	}
	if limit != -1 {
		query += q.sqlOffsetLimit(offset, limit)
	}
	nMap := make(map[string]SqlMap)
	if rows := q.query(query); rows != nil {
		defer rows.Close()
		columns, _ := rows.Columns()
		values   := make([]sql.RawBytes, len(columns))
		rowSlice := make([]interface{}, len(columns))
		for i := range values {
			rowSlice[i] = &values[i]
		}
		keyVal := ""
		for rows.Next() {
			rows.Scan(rowSlice...)
			item := make(SqlMap)
			for idx, col := range values {
				if col == nil {
					col = sql.RawBytes{}
				}
				if columns[idx] == key {
					keyVal = string(col)
				}
				item[columns[idx]] = q.convertItem(columns[idx], col)
			}
			//完成额外的格式化操作 例如数据拼凑等
			if q.format != nil {
				q.format(item)
			}
			nMap[keyVal] = item
		}
	}
	return nMap
}

//获取某个值信息
func (q *QuerySt) GetValue() SqlString {
	strVal := SqlString("")
	query  := q.AsSql("select") + " LIMIT 1"
	if rows := q.query(query); rows != nil {
		defer rows.Close()
		for rows.Next() {
			sliceValue := make(sql.RawBytes, 0)
			rows.Scan(&sliceValue)
			strVal = SqlString(sliceValue)
			break
		}
	}
	return strVal
}
