package orm

import (
	"github.com/jmoiron/sqlx"
	"github.com/leicc520/go-orm/cache"
)

type IFModel interface {
	Format(handle FormatItemHandle) *ModelSt
	SetCache(cacheSt cache.Cacher) *ModelSt
	NoCache() *ModelSt
	GetSlot() int
	GetTable() string
	SetTable(table string) *ModelSt
	SetTx(tx *sqlx.Tx) *ModelSt
	ResetTable() *ModelSt
	SetModTable(idx int64) *ModelSt
	EqMod(idx, oidx int64) bool
	SetDevTable(idx int64) *ModelSt
	SetYmTable(format string) *ModelSt
	DBTables() []string
	CacheKey(ckey string) string
	Query() *QuerySt
	SetCacheVer()
	GetCacheVer() string
	GetHash(args ...interface{}) string
	IdClearCache(id interface{}) *ModelSt
	Conflict(fields string) *ModelSt
	NewOne(fields SqlMap, dupfields SqlMap) int64
	NewOneFromHandler(vhandler VHandler, dhandler DHandler) int64
	GetOne(id interface{}) SqlMap
	Save(id interface{}, fields SqlMap) int64
	SaveFromHandler(id interface{}, vhandler VHandler) int64
	MultiUpdate(whandler WHandler, vhandler VHandler) int64
	MultiUpdateCleanID(whandler WHandler, vhandler VHandler) int64
	Delete(id interface{}) int64
	MultiDelete(whandler WHandler) int64
	MultiDeleteCleanID(whandler WHandler) int64
	GetItem(cHandler WHandler, fields string, args ...string) SqlMap
	GetList(offset, limit int64, cHandler WHandler, fields string, args ...string) []SqlMap
	ChunkHandler(limit int64, fields string, cHandler CHandler, wHandler WHandler) (int64, error)
	GetColumn(offset, limit int64, cHandler WHandler, fields string, args ...string) []string
	GetAsMap(offset, limit int64, cHandler WHandler, fields string) SqlMap
	GetNameMap(offset, limit int64, cHandler WHandler, fields string, key string) map[string]SqlMap
	GetTotal(cHandler WHandler, fields string) SqlString
	GetValue(cHandler WHandler, fields string) SqlString
	IsExists(cHandler WHandler) SqlString
	GetAsSQL(sql string, offset, limit int64) []SqlMap
}
