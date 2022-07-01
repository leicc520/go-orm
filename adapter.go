package orm

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

/******************************************************************************
	     数据库的适配器，主要调整数据库与配置类 Redis与配置类的衔接，初始化数据库缓存
 ******************************************************************************/
type XDBPoolSt struct {
	dbs map[string]*sqlx.DB
	config map[string]*DbConfig
}

type DbConfig struct {
	Driver  string 		`yaml:"driver"`
	Host string  		`yaml:"host"`
	SKey string         `yaml:"skey"`
	MaxOpenConns int	`yaml:"maxOpenConns"`
	MaxIdleConns int 	`yaml:"maxIdleConns"`
}

var (
	IsShowSql = true
	dbOnceLocker = sync.Once{}
	GdbPoolSt = XDBPoolSt{}
)

//完成数据库的初始化处理逻辑
func InitDBPoolSt() *XDBPoolSt {
	dbOnceLocker.Do(func() {
		GdbPoolSt = XDBPoolSt{
			dbs: make(map[string]*sqlx.DB),
			config: make(map[string]*DbConfig),
		}
	})
	return &GdbPoolSt
}

//获取数据库连接句柄
func (p *XDBPoolSt) Get(skey string) *sqlx.DB {
	if db, ok := p.dbs[skey]; ok && db != nil {
		if err := db.Ping(); err != nil {
			log.Write(log.ERROR, "数据库:"+skey+" Ping 失败:"+err.Error())
		}
		return db
	}
	db := p.NewEngine(skey)
	p.dbs[skey] = db
	return db
}

//获取数据库连接句柄
func (p *XDBPoolSt) Set(skey string, config *DbConfig) *sqlx.DB {
	p.config[skey] = config
	db := p.NewEngine(skey)
	p.dbs[skey] = db
	log.Write(log.DEBUG, "db load create database pool{", skey, "} ", config.Host)
	return db
}

//返回配置数据资料信息
func (p *XDBPoolSt) Config(skey string) *DbConfig {
	if config, ok := p.config[skey]; ok {
		return config
	}
	return nil
}

//释放db连接句柄信息
func (p *XDBPoolSt) Release() {
	for _, db := range p.dbs {
		db.Close()
		db = nil
	}
	if GdbCache != nil {//关闭退出时候资源释放
		GdbCache.Close()
	}
	log.Write(log.INFO, "释放数据库资源...")
}

/**
 * 数据库的配置 通过配置导入，配置必须传结构体指针 示例
 * @confPtr *Config 配置对象的指针变量
type Config struct {
	...
	Redis  cache.RedisConfig 	`yaml:"redis"`
	DbMaster  DbConfig 			`yaml:"dbmaster"`
	DbSlaver  DbConfig 			`yaml:"dbslaver"`
}
 */
func (p *XDBPoolSt)LoadDbConfig(confPtr interface{}) {
	SetCachePrefix("v01")
	confValues := reflect.ValueOf(confPtr).Elem()
	Alen := confValues.NumField()
	for i := 0; i < Alen; i++ {
		tempValues := confValues.Field(i)
		if tempValues.Type().Name() == "Config" {
			Jlen := tempValues.NumField()
			for j := 0; j < Jlen; j++ {//二级继承检索加载db
				childValues := tempValues.Field(j)
				p.loadDBCache(j, tempValues, childValues)
			}
		}
		p.loadDBCache(i, confValues, tempValues)
	}
}

//加载数据库配置完成数据库的基础初始化业务逻辑
func (p *XDBPoolSt)loadDBCache(i int, confValues reflect.Value, tempValues reflect.Value) {
	if tempValues.Type().Name() == "DbConfig" {
		key := strings.ToLower(confValues.Type().Field(i).Name)
		dbConfig := tempValues.Interface().(DbConfig)
		if len(dbConfig.Driver) > 0 && len(dbConfig.Host) > 0 {
			p.Set(key, &dbConfig)//创建数据库连接池处理逻辑
		}
	} else if tempValues.Type().Name() == "CacheConfigSt" {
		cacheConfig := tempValues.Interface().(cache.CacheConfigSt)
		vCache := cache.Factory(cacheConfig.Driver, cacheConfig.Params)
		SetDBCache(vCache) //将缓存策略注册到orm当中
		log.Write(log.DEBUG, "load dbcache "+cacheConfig.Driver+" "+cacheConfig.Params)
	}
}

//创建DB对象 提供给其他地方使用
func (p *XDBPoolSt)NewEngine(skey string) *sqlx.DB {
	dbConfig, ok := p.config[skey]
	if !ok {//未注册的数据情况
		log.Write(log.ERROR, "load dbConfig("+skey+") not Register")
		panic(errors.New("load dbConfig("+skey+") not Register"))
	}
	db, err := sqlx.Open(dbConfig.Driver, dbConfig.Host)
	if err != nil {
		log.Write(log.FATAL, "("+dbConfig.Driver+")"+dbConfig.Host + err.Error())
		panic(err)
	}
	if dbConfig.MaxOpenConns > 512 || dbConfig.MaxOpenConns < 8 {
		db.SetMaxOpenConns(8)
	} else {//防止设置错误的情况出现
		db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	}
	if dbConfig.MaxIdleConns > 256 || dbConfig.MaxIdleConns < 8 {
		db.SetMaxIdleConns(8)
	} else {//防止设置错误的情况出现
		db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	}
	//设置单个链接最多允许使用十分钟
	db.SetConnMaxLifetime(time.Second*600)
	if err = db.Ping(); err != nil {
		log.Write(log.FATAL, err.Error())
		panic(err)
	}
	return db
}