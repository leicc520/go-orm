package cache

import (
	"git.ziniao.com/webscraper/go-orm/log"
	"git.ziniao.com/webscraper/go-orm/sqlmap"
	jsonIter "github.com/json-iterator/go"
)

type ErrNotExists struct {
}

func (e ErrNotExists) Error() string {
	return "Data Not Exists!"
}

type Cacher interface {
	Set(key string, data interface{}, expire int64) bool
	Get(key string) interface{}
	GetStruct(key string, out interface{}) error
	Del(keys ...string) bool
	Close()
	Clear()
}

type IFactoryCache interface {
	Open(params interface{}) Cacher
}

type CacheConfigSt struct {
	Driver string `yaml:"driver"`
	Params string `yaml:"params"`
}

type CacheItemSt struct {
	Expire int64       `json:"expire" bson:"expire"`
	Data   interface{} `json:"data" bson:"data"`
}

//interface 重新解析到对象
func ToStruct(in, out interface{}) error {
	if err := sqlmap.WeakDecode(in, out); err != nil {
		log.Write(log.ERROR, "convert interface{} to struct error; "+err.Error())
		return err
	}
	return nil
}

var gDrivers = map[string]IFactoryCache{}
var json = jsonIter.ConfigCompatibleWithStandardLibrary

//注册Cache到注册器中
func Register(name string, driver IFactoryCache) {
	if driver == nil {
		panic("cache: Register driver is nil")
	}
	if _, dup := gDrivers[name]; dup {
		panic("cache: Register called twice for driver " + name)
	}
	gDrivers[name] = driver
}

//生成一个Cache执行实例
func Factory(name string, params interface{}) Cacher {
	if _, ok := gDrivers[name]; !ok {
		panic("cache: Factory not exists driver " + name)
	}
	return gDrivers[name].Open(params)
}
