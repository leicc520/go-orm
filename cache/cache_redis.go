package cache

import (
	"time"

	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/go-redis/redis"
)

type FactoryRedisCache struct {
}

func (factory *FactoryRedisCache) Open(params interface{}) Cacher {
	args, ok := params.(string)
	if !ok { //数据出错的情况处理逻辑
		panic("redis cache Open failed")
	}
	cache := NewRedisCache(args)
	return Cacher(cache)
}

func init() {
	factory := &FactoryRedisCache{}
	Register("redis", IFactoryCache(factory))
}

type RedisCacheSt struct {
	client *redis.Client
	config *redis.Options
}

//初始化一个Cache实例
func NewRedisCache(dsn string) *RedisCacheSt {
	if opt, err := redis.ParseURL(dsn); err != nil {
		panic("redis cache dsn parse error")
		return nil
	} else {
		client := redis.NewClient(opt)
		cache := &RedisCacheSt{client: client, config: opt}
		return cache
	}
}

//线程安全获取记录
func (self *RedisCacheSt) asyncGet(key string) []byte {
	byteStr, err := self.client.Get(key).Bytes()
	if err != nil { //数据不存在的情况
		if err != redis.Nil {
			log.Write(log.ERROR, "redis get error "+err.Error())
		}
		byteStr = nil
	}
	return byteStr
}

//获取一个缓存记录信息，过期记录不返回
func (self *RedisCacheSt) Get(key string) interface{} {
	var byteStr []byte = nil
	if byteStr = self.asyncGet(key); byteStr == nil {
		return nil //数据为空的情况
	}
	var item CacheItemSt //需要注意json解码的时候int64会变成float64
	if err := json.Unmarshal(byteStr, &item); err != nil {
		log.Write(log.ERROR, "redis get unmarshal error "+err.Error())
		return nil
	}
	log.Write(log.INFO, "redis get cache "+key+"="+string(byteStr))
	//判断是否过期
	if item.Expire > 0 && item.Expire < time.Now().Unix() {
		return nil
	}
	return item.Data
}

//直接获取数据并解析到结构当中
func (self *RedisCacheSt) GetStruct(key string, out interface{}) error {
	data := self.Get(key) //获取数据
	if data == nil {      //数据不存在的情况
		return ErrNotExists{}
	}
	err := ToStruct(data, out)
	return err
}

//线程安全获取记录
func (self *RedisCacheSt) asyncSet(key string, byteStr []byte, exp int64) bool {
	result, expire := true, time.Duration(exp)*time.Second
	if err := self.client.Set(key, byteStr, expire).Err(); err != nil {
		log.Write(log.ERROR, "redis client Set error "+err.Error())
		result = false
	}
	log.Write(log.INFO, "redis cache set "+key+"="+string(byteStr))
	return result
}

//设置一个缓存记录
func (self *RedisCacheSt) Set(key string, data interface{}, expire int64) bool {
	sTime := time.Now().Unix()
	if expire > 0 && expire < sTime {
		expire += sTime
	}
	item := CacheItemSt{Expire: expire, Data: data}
	if byteStr, err := json.Marshal(item); err != nil {
		log.Write(log.ERROR, "redis json encode error "+err.Error())
		return false
	} else {
		return self.asyncSet(key, byteStr, expire)
	}
}

//删除缓存记录信息
func (self *RedisCacheSt) Del(keys ...string) bool {
	err := self.client.Del(keys...).Err()
	if err != nil {
		log.Write(log.ERROR, "redis delete error "+err.Error())
		return false
	}
	return true
}

//清理内存缓存记录
func (self *RedisCacheSt) Clear() {
	err := self.client.FlushDB().Err()
	if err != nil {
		log.Write(log.ERROR, "redis flushdb error "+err.Error())
	}
}

//清理关闭连接处理逻辑
func (self *RedisCacheSt) Close() {
	self.client.Close()
}
