package orm

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	
	"github.com/jmoiron/sqlx"
	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
)

const (
	DBVERCKEY = "dbver"
)

var (
	GdbCache cache.Cacher = nil
	GmCache cache.Cacher = nil
	GCacheKeyPrefix = "go"
)

//设置数据库缓存 -默认空不做缓存
func SetDBCache(c cache.Cacher) {
	GdbCache = c
}

//获取内存存储的缓存策略
func GetMCache() cache.Cacher {
	if GmCache == nil {
		GmCache = cache.Factory("memory", map[string]interface{}{"gc":time.Minute})
	}
	return GmCache
}

//定义数据模型的基础结构
type ModelSt struct {
	table    string
	orgtable string
	prikey   string
	dbmaster string
	dbslaver string
	cachever string
	slot     int
	cache   cache.Cacher
	query   *QuerySt
	dbctx   *XDBPoolSt
	fields   map[string]reflect.Kind
}

//获取模型数据资料信息
func (self *ModelSt) Instance() *ModelSt {
	return self
}

//初始化模型 业务参数设定
func (self *ModelSt) Init (dbPool *XDBPoolSt, data map[string]interface{}, fields map[string]reflect.Kind) {
	self.dbctx    = dbPool //控制数据的连接和释放
	self.table    = data["table"].(string)
	self.orgtable = data["orgtable"].(string)
	self.prikey   = data["prikey"].(string)
	self.dbmaster = data["dbmaster"].(string)
	self.dbslaver = data["dbslaver"].(string)
	self.slot     = data["slot"].(int)
	self.cache    = GdbCache //设置默认的缓存
	self.fields   = fields
	self.query    = NewQuery(self.fields)
	if config := self.dbctx.Config(self.dbmaster); config != nil {
		self.query.SetDriver(config.Driver) //设置书库驱动做底层sql适配
	}
}

//主要用作处理数据的格式化逻辑
func (self *ModelSt) Format(handle FormatItemHandle) *ModelSt {
	self.query.SetFormat(handle)
	return self
}

func (self *ModelSt) SetCache(cacheSt cache.Cacher) *ModelSt {
	self.cache = cacheSt
	return self
}

//关闭换成策略
func (self *ModelSt) NoCache() *ModelSt {
	self.cache = nil
	return self
}

func (self *ModelSt) GetSlot() int {
	return self.slot
}

func (self *ModelSt) GetTable() string {
	return self.table
}

func (self *ModelSt) SetTable(table string) *ModelSt {
	self.table = table
	return self
}

func (self *ModelSt) SetTx(tx *sqlx.Tx) *ModelSt {
	self.query.SetTx(tx)
	return self
}

func (self *ModelSt) ResetTable() *ModelSt {
	self.table = self.orgtable
	return self
}

func (self *ModelSt) SetModTable(idx int64) *ModelSt {
	if self.slot > 1 {
		self.table = fmt.Sprintf("%s%d", self.orgtable, idx%int64(self.slot))
		self.doCreateTable(self.orgtable, self.table)
	}
	return self
}

func (self *ModelSt) EqMod(idx, oidx int64) bool {
	slot := int64(self.slot)
	if slot > 0 && idx % slot == oidx % slot {
		return true
	}
	return false
}

func (self *ModelSt) SetDevTable(idx int64) *ModelSt {
	if self.slot > 1 {
		nslot := int(idx/int64(self.slot))
		self.table = fmt.Sprintf("%s%d", self.orgtable, nslot)
		self.doCreateTable(self.orgtable, self.table)
	}
	return self
}

//这个时间格式填写golang诞辰 2006-01-02 15:04:05 等
func (self *ModelSt) SetYmTable(format string) *ModelSt {
	if len(format) < 2 || !strings.Contains(format, "06") {
		log.Write(log.ERROR, "SetYmTable format string invalid")
		return self
	}
	sStr := time.Now().Format(format)
	self.table = fmt.Sprintf("%s%s", self.orgtable, sStr)
	self.doCreateTable(self.orgtable, self.table)
	return self
}

//获取当前表的所有分表记录
func (self *ModelSt) DBTables() []string {
	sql  := "show tables like '"+self.orgtable+"%'"
	if self.query.GetDriver() == POSTGRES {
		sql = "SELECT relname FROM pg_class WHERE relkind = 'r' AND relname LIKE '"+self.orgtable+"%'"
	}
	db   := self.dbctx.Get(self.dbmaster)
	list := self.query.SetDb(db).GetColumn(sql, 0, -1)
	return list
}

//创建一个table 分表
func (self *ModelSt) doCreateTable(otable, ntable string) {
	ckey := fmt.Sprintf("table@%s_%s", self.dbmaster, ntable)
	mCache := GetMCache() //获取缓存设定
	if exists := mCache.Get(ckey); exists == 1 {
		return
	}
	vCache := cache.NewFileCache("./cachedir", 1)
	if exists := vCache.Get(ckey); exists == 1 {
		return
	}
	db := self.dbctx.Get(self.dbmaster)
	if self.query.GetDriver() == POSTGRES {
		self.postgresCreateTable(db, otable, ntable)
	} else {
		self.mysqlCreateTable(db, otable, ntable)
	}
	mCache.Set(ckey, 1, 0)
	vCache.Set(ckey, 1, 0)
}

//创建一个pgsql分表处理逻辑
func (self *ModelSt) mysqlCreateTable(db *sqlx.DB, otable, ntable string) {
	query := fmt.Sprintf("SHOW TABLES LIKE \"%s\"", ntable)
	rows, err := db.Query(query)
	if  err != nil {
		log.Write(log.FATAL, "create table query error"+query + err.Error())
		panic(errors.New("create table query error"+query))
	}
	nrow := 0
	for rows.Next() {
		nrow += 1
	}
	rows.Close()
	if nrow < 1 {//创建表数据信息
		var table, csql string = "", ""
		query := fmt.Sprintf("SHOW CREATE TABLE %s", otable)
		if err := db.QueryRow(query).Scan(&table, &csql); err != nil {
			log.Write(log.FATAL, "create table get sql template error "+query + err.Error())
			panic(errors.New("create table get sql template error"+query))
		}
		csql = strings.Replace(csql, otable, ntable, 1)
		csql = strings.Replace(csql, "CREATE TABLE", "CREATE TABLE IF NOT EXISTS ", 1)
		if _, err := db.Exec(csql); err != nil {
			log.Write(log.FATAL, "create new table error "+query + err.Error())
			panic(errors.New("create new table error"+query))
		}
	}
}

//创建一个pgsql分表处理逻辑
func (self *ModelSt) postgresCreateTable(db *sqlx.DB, otable, ntable string) {
	csql := "CREATE TABLE IF NOT EXISTS "+ntable+"() INHERITS ("+otable+")"
	if _, err := db.Exec(csql); err != nil {
		log.Write(log.FATAL, "create new table error "+csql + err.Error())
		panic(errors.New("create new table error"+csql))
	}
}

//缓存策略的key信息
func (self *ModelSt) CacheKey(ckey string) string {
	return self.table+GCacheKeyPrefix+"@"+ckey
}

//获取db查询的Query db要自行关闭哟
func (self *ModelSt) Query() *QuerySt {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	return self.query
}

//设定缓存的版本号数据信息
func (self *ModelSt) SetCacheVer() {
	version := strconv.FormatInt(time.Now().UnixNano(), 10)
	if self.cache != nil && self.cachever != version {
		self.cachever = version //版本不一致的时候更新
		self.cache.Set(self.CacheKey(DBVERCKEY), self.cachever, 0)
	}
}

//获取db存储的版本号
func (self *ModelSt) GetCacheVer() string {
	if len(self.cachever) == 0 && self.cache != nil {
		skey := self.CacheKey(DBVERCKEY)
		data := self.cache.Get(skey)
		if self.cachever, _ = data.(string); len(self.cachever) == 0 {
			self.cachever = strconv.FormatInt(time.Now().UnixNano(), 10)
			self.cache.Set(skey, self.cachever, 0)
		}
	}
	return self.cachever
}

//查询条件hash成md5字符串
func (self *ModelSt) GetHash(args ...interface{}) string {
	args = append(args, self.GetCacheVer())
	if self.query.GetFormat() != nil {//格式化
		args = append(args, "format")
	}
	bstr, err := json.Marshal(args)
	if err != nil {//业务出错的情况
		log.Write(log.ERROR, "GetHash json Marshal error "+err.Error())
		return ""
	}
	md5Str := fmt.Sprintf("%x", md5.Sum(bstr))
	log.Write(log.INFO, self.table + " " + string(bstr) + " "+md5Str)
	return self.table+"@"+md5Str
}

//根据ID删除缓存策略
func (self *ModelSt) IdClearCache(id interface{}) *ModelSt {
	if self.cache == nil {
		return self
	}
	ckey := fmt.Sprintf("%v", id)
	ckey  = self.CacheKey(ckey)
	self.cache.Del(ckey)
	self.SetCacheVer()
	return self
}

//表中新增一条记录  主键冲突且设置dup的话会执行后续配置的更新操作
func (self *ModelSt) NewOne(fields SqlMap, dupfields SqlMap) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	//设置插入主键冲突的时候使用更新字段信息
	var priKeyValue interface{} = nil
	if dupfields != nil && len(dupfields) > 0 {
		for field, value := range dupfields {
			if field == self.prikey {
				priKeyValue = value
			}
			if opSql, ok := value.([]interface{}); ok && len(opSql) == 2 {
				self.query.Duplicate(field, opSql[0], opSql[1].(string))
			} else { //非数组的情况
				self.query.Duplicate(field, value, DT_AUTO)
			}
		}
	}
	lstId := self.query.SetAutoIncr(self.prikey).Insert(fields, false)
	if lstId >= 0 && self.cache != nil {//更新缓存版本
		if priKeyValue != nil {
			self.IdClearCache(priKeyValue)
		} else {
			self.SetCacheVer()
		}
	}
	return lstId
}

//通过执行匿名函数实现数据更新关系绑定
func (self *ModelSt) NewOneFromHandler(vhandler VHandler, dhandler DHandler) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	var priKeyValue interface{} = nil
	if dhandler != nil {//设置主键冲突执行update逻辑
		priKeyValue = dhandler(self.query)
	}
	sql   := vhandler(self.query).SetAutoIncr(self.prikey).AsSql("insert")
	lstId := self.query.QueryInsert(sql) //直接sql插入
	if lstId >= 0 && self.cache != nil {//更新缓存版本
		if priKeyValue != nil {
			self.IdClearCache(priKeyValue)
		} else {
			self.SetCacheVer()
		}
	}
	return lstId
}

//获取一个对象实例
func (self *ModelSt) GetOne(id interface{}) SqlMap {
	var ckey = ""
	if self.cache != nil {//通过缓存获取
		ckey = self.CacheKey(fmt.Sprintf("%v", id))
		data:= self.cache.Get(ckey)
		if smap, ok := data.(map[string]interface{}); ok && data != nil {
			log.Write(log.INFO, "GetOne load data from cache")
			return SqlMap(smap)
		}
	}
	//数据不存在的情况
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Clear().Table(self.table).Where(self.prikey, id)
	data := self.query.GetRow("")
	if !data.IsNil() && self.cache != nil {
		self.cache.Set(ckey, data, 86400)
	}
	log.Write(log.INFO, "GetOne load data from database")
	return data
}

//更新信息记录
func (self *ModelSt) Save(id interface{}, fields SqlMap) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table).Where(self.prikey, id)
	nrow := self.query.Update(fields)
	if nrow > 0 {//更新成功的情况 更新缓存数据
		self.IdClearCache(id)
	}
	return nrow
}

//通过执行匿名函数实现数据更新关系绑定
func (self *ModelSt) SaveFromHandler(id interface{}, vhandler VHandler) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table).Where(self.prikey, id)
	sql := vhandler(self.query).AsSql("update") //执行业务values赋值
	if result := self.query.Exec(sql); result == nil {
		return -1
	} else {//更新成功的情况
		rows, _ := result.RowsAffected()
		if rows > 0 && self.cache != nil {
			self.IdClearCache(id)
		}
		return rows
	}
}

//通过执行匿名函数实现数据更新关系绑定
func (self *ModelSt) MultiUpdate(whandler WHandler, vhandler VHandler) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	whandler(self.query) //指定条件设定绑定
	sql := vhandler(self.query).AsSql("update")
	if result := self.query.Exec(sql); result == nil {
		return -1
	} else {//更新成功的情况
		rows, _ := result.RowsAffected()
		if rows > 0 && self.cache != nil {
			self.SetCacheVer()
		}
		return rows
	}
}

//通过执行匿名函数实现数据更新关系绑定 清理关联记录缓存
func (self *ModelSt) MultiUpdateCleanID(whandler WHandler, vhandler VHandler) int64 {
	ids := self.GetColumn(0, -1, whandler, self.prikey)
	db  := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	whandler(self.query) //指定条件设定绑定
	sql := vhandler(self.query).AsSql("update")
	if result := self.query.Exec(sql); result == nil {
		return -1
	} else {//更新成功的情况
		rows, _ := result.RowsAffected()
		if rows > 0 && self.cache != nil && ids != nil && len(ids) > 0 {
			for _, id := range ids {//更新id删除缓存
				self.IdClearCache(id)
			}
		}
		return rows
	}
}

//删除一条记录信息
func (self *ModelSt) Delete(id interface{}) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table).Where(self.prikey, id)
	nrow := self.query.Delete()
	if nrow > 0 {//删除成功，刷新缓存数据信息
		self.IdClearCache(id)
	}
	return nrow
}

//删除多条记录  单数缓存数据可能还是存在 通过id获取数据的情况
func (self *ModelSt) MultiDelete(whandler WHandler) int64 {
	db := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	whandler(self.query)
	//设定删除数据的条件执行
	nrow := self.query.Delete()
	if nrow > 0 {//删除成功，刷新缓存数据信息
		self.SetCacheVer()
	}
	return nrow
}

//删除多条记录  清理关联记录缓存
func (self *ModelSt) MultiDeleteCleanID(whandler WHandler) int64 {
	ids := self.GetColumn(0, -1, whandler, self.prikey)
	db  := self.dbctx.Get(self.dbmaster)
	self.query.SetDb(db).Clear().Table(self.table)
	whandler(self.query)
	//设定删除数据的条件执行
	nrow := self.query.Delete()
	if nrow > 0 && ids != nil && len(ids) > 0 {//删除成功，刷新缓存数据信息
		for _, id := range ids {//更新id删除缓存
			self.IdClearCache(id)
		}
	}
	return nrow
}

//解析字段排序信息等
func (self *ModelSt) sortDir(args []string) (sort, dir string) {
	Alen := len(args)
	sort, dir = "", DESC
	if Alen == 0 {
		return
	} else if Alen == 1 {
		sort = args[0]
		return
	} else if Alen == 2 {
		sort, dir = args[0], args[1]
		return
	}
	panic("parse fields sort dir args error!")
}

//获取一个选项记录信息
func (self *ModelSt) GetItem(cHandler WHandler, fields string, args ...string) SqlMap {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetItem", wheres, fields, args)
		data := self.cache.Get(ckey)
		if smap, ok := data.(map[string]interface{}); ok && data != nil {
			log.Write(log.INFO, "GetItem load data from cache")
			return SqlMap(smap)
		}
	}
	sort, dir := self.sortDir(args)
	if sort != "" && dir != "" {
		self.query.OrderBy(sort, dir)
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	data := self.query.GetRow("")
	if !data.IsNil() && self.cache != nil {
		self.cache.Set(ckey, data, 3600)
	}
	log.Write(log.INFO, "GetItem load data from database")
	return data
}

//只获取数据信息列表
func (self *ModelSt) GetList(offset, limit int64, cHandler WHandler, fields string, args ...string) []SqlMap {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetList", wheres, offset, limit, fields, args)
		temp := self.cache.Get(ckey)
		if tmp, ok := temp.([]interface{}); ok && tmp != nil {
			list := make([]SqlMap, len(tmp))
			for idx, smap := range tmp {
				list[idx] = smap.(map[string]interface{})
			}
			log.Write(log.INFO, "GetList load data from cache")
			return list
		}
	}
	sort, dir := self.sortDir(args)
	if sort != "" && dir != "" {
		self.query.OrderBy(sort, dir)
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	list := self.query.GetList("", offset, limit)
	if list != nil && len(list) > 0 && self.cache != nil {
		self.cache.Set(ckey, list, 3600)
	}
	log.Write(log.INFO, "GetList load data from database")
	return list
}

//执行分页回调业务处理逻辑，这里最好自行orderby一下
func (self *ModelSt) ChunkHandler(limit int64, fields string, cHandler CHandler, wHandler WHandler) (int64, error) {
	self.query.Clear().Table(self.table)
	if wHandler != nil {//预设查询条件，可能会比较复杂
		wHandler(self.query)
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	nsize, offset := int64(0), int64(0)
	if limit < 1 {//limit不允许小于1
		limit = 1
	}
	var err error = nil
	for {//遍历抓取数据
		list := self.query.GetList("", offset, limit)
		if cHandler != nil {//执行回传的业务逻辑
			err = cHandler(list)
			if err != nil {
				break
			}
		}
		offset += limit
		if list != nil && len(list) > 0 {
			nsize  += int64(len(list))
		}
		if list == nil || len(list) < int(limit) {
			break
		}
	}
	return nsize, err
}

//只获取数据信息列表
func (self *ModelSt) GetColumn(offset, limit int64, cHandler WHandler, fields string, args ...string) []string {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetColumn", wheres, offset, limit, fields, args)
		temp := self.cache.Get(ckey)
		if temp, ok := temp.([]interface{}); ok && temp != nil {
			list := make([]string, len(temp))
			for idx, lstr := range temp {
				list[idx] = lstr.(string)
			}
			log.Write(log.INFO, "GetColumn load data from cache")
			return list
		}
	}
	sort, dir := self.sortDir(args)
	if sort != "" && dir != "" {
		self.query.OrderBy(sort, dir)
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	list := self.query.GetColumn("", offset, limit)
	if list != nil && len(list) > 0 && self.cache != nil {
		self.cache.Set(ckey, list, 3600)
	}
	log.Write(log.INFO, "GetColumn load data from database")
	return list
}

//只获取数据信息列表key,val
func (self *ModelSt) GetAsMap(offset, limit int64, cHandler WHandler, fields string) SqlMap {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetAsMap", wheres, offset, limit, fields)
		data := self.cache.Get(ckey)
		if smap, ok := data.(map[string]interface{}); ok && data != nil {
			log.Write(log.INFO, "GetAsMap load data from cache")
			return SqlMap(smap)
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	data := self.query.GetMap("", offset, limit)
	if !data.IsNil() && self.cache != nil {
		self.cache.Set(ckey, data, 3600)
	}
	log.Write(log.INFO, "GetAsMap load data from database")
	return data
}

//获取命名map，key必须属于fields当中的字段
func (self *ModelSt) GetNameMap(offset, limit int64, cHandler WHandler, fields string, key string) map[string]SqlMap {
	if !strings.Contains(fields, key) {
		panic("name map key string must in fields")
	}
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil { //通过缓存获取
		ckey  = self.GetHash("GetNameMap", wheres, offset, limit, fields, key)
		temp := self.cache.Get(ckey)
		if temp, ok := temp.(map[string]interface{}); ok && temp != nil {
			data := make(map[string]SqlMap, len(temp))
			for idx, sitem := range temp {
				data[idx] = SqlMap(sitem.(map[string]interface{}))
			}
			log.Write(log.INFO, "GetNameMap load data from cache")
			return data
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	data := self.query.NameMap("", key, offset, limit)
	if data != nil && self.cache != nil {
		self.cache.Set(ckey, data, 3600)
	}
	log.Write(log.INFO, "GetNameMap load data from database")
	return data
}

//获取一个选项记录信息
func (self *ModelSt) GetTotal(cHandler WHandler, fields string) SqlString {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetTotal", wheres, fields)
		data := self.cache.Get(ckey)
		if _, ok := data.(string); ok && data != nil {
			log.Write(log.INFO, "GetTotal load data from cache")
			return SqlString(data.(string))
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	total := self.query.GetValue()
	if len(total) > 0 && self.cache != nil {
		self.cache.Set(ckey, total, 3600)
	}
	log.Write(log.INFO, "GetTotal load data from database")
	return total
}

//获取一个选项记录信息
func (self *ModelSt) GetValue(cHandler WHandler, fields string) SqlString {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("GetValue", wheres, fields)
		data := self.cache.Get(ckey)
		if _, ok := data.(string); ok && data != nil {
			log.Write(log.INFO, "GetValue load data from cache")
			return SqlString(data.(string))
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(fields)
	total := self.query.GetValue()
	if len(total) > 0 && self.cache != nil {
		self.cache.Set(ckey, total, 3600)
	}
	log.Write(log.INFO, "GetValue load data from database")
	return total
}

//获取一个选项记录信息
func (self *ModelSt) IsExists(cHandler WHandler) SqlString {
	self.query.Clear().Table(self.table)
	ckey, wheres := "", ""
	if cHandler != nil {//预设查询条件，可能会比较复杂
		wheres = cHandler(self.query)
	}
	if self.cache != nil {//通过缓存获取
		ckey  = self.GetHash("IsExists", wheres)
		data := self.cache.Get(ckey)
		if _, ok := data.(string); ok && data != nil {
			log.Write(log.INFO, "IsExists load data from cache")
			return SqlString(data.(string))
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Field(self.prikey)
	pkStr := self.query.GetValue()
	if len(pkStr) > 0 && self.cache != nil {
		self.cache.Set(ckey, pkStr, 3600)
	}
	log.Write(log.INFO, "IsExists load data from database")
	return pkStr
}

//通过SQL查询数据
func (self *ModelSt) GetAsSQL(sql string, offset, limit int64) []SqlMap {
	ckey := self.GetHash("GetAsSQL", sql, offset, limit)
	if self.cache != nil {//通过缓存获取
		temp := self.cache.Get(ckey)
		if temp, ok := temp.([]interface{}); ok && temp != nil {
			list := make([]SqlMap, len(temp))
			for idx, smap := range temp {
				list[idx] = SqlMap(smap.(map[string]interface{}))
			}
			log.Write(log.INFO, "GetAsSQL load data from cache")
			return list
		}
	}
	db := self.dbctx.Get(self.dbslaver)
	self.query.SetDb(db).Clear().Table(self.table)
	list := self.query.GetAsSql(sql, false, offset, limit)
	if list != nil && len(list) > 0 && self.cache != nil {
		self.cache.Set(ckey, list, 3600)
	}
	log.Write(log.INFO, "GetAsSQL load data from database")
	return list
}