package cache

import (
	"sync"
	"time"

	"github.com/leicc520/go-orm/log"
)

type FactoryMemoryCache struct {
}

func (factory *FactoryMemoryCache) Open(params interface{}) Cacher {
	args := params.(map[string]interface{})
	gc := args["gc"].(time.Duration)
	cache := NewMemoryCache(gc)
	return Cacher(cache)
}

func init() {
	factory := &FactoryMemoryCache{}
	Register("memory", IFactoryCache(factory))
}

type MemoryCacheSt struct {
	store  sync.Map
	gc     time.Duration
}

//初始化一个Cache实例
func NewMemoryCache(gc time.Duration) *MemoryCacheSt {
	cache := &MemoryCacheSt{gc: gc}
	time.AfterFunc(cache.gc, cache.GC)
	return cache
}

//获取一个缓存记录信息，过期记录不返回
func (self *MemoryCacheSt) Get(key string) interface{} {
	data, exists := self.store.Load(key)
	if !exists {
		return nil
	}
	if item, ok := data.(CacheItemSt); !ok {
		return nil
	} else {
		if item.Expire > 0 && item.Expire < time.Now().Unix() {
			return nil
		}
		return item.Data
	}
}

//直接获取数据并解析到结构当中
func (self *MemoryCacheSt) GetStruct(key string, out interface{}) error {
	data := self.Get(key) //获取数据
	if data == nil {//数据不存在的情况
		return ErrNotExists{}
	}
	err := ToStruct(data, out)
	return err
}

//设置一个缓存记录
func (self *MemoryCacheSt) Set(key string, data interface{}, expire int64) bool {
	ntime := time.Now().Unix()
	if expire > 0 && expire < ntime {
		expire += ntime
	}
	item := CacheItemSt{Expire: expire, Data: data}
	self.store.Store(key, item)
	return true
}

//删除缓存记录信息
func (self *MemoryCacheSt) Del(keys ...string) bool {
	for i := 0; i < len(keys); i++ {
		self.store.Delete(keys[i])
	}
	return true
}

//清理内存缓存记录
func (self *MemoryCacheSt) Clear() {
	self.store.Range(func(key, _ interface{}) bool {
		self.store.Delete(key)
		return true
	})
}

//清理内存缓存记录
func (self *MemoryCacheSt) Close() {
	self.gc = -1
	self.Clear()
}

//GC过期资源回收
func (self *MemoryCacheSt) GC() {
	ntime := time.Now().Unix()
	self.store.Range(func(key, value interface{}) bool {
		if item, ok := value.(CacheItemSt); ok {
			if item.Expire > 0 && item.Expire < ntime {
				self.store.Delete(key)
			}
		} else {
			self.store.Delete(key)
		}
		return true
	})
	log.Write(log.INFO,"memory cache GC ending")
	if self.gc > 0 {//配置gc循环执行时间
		time.AfterFunc(self.gc, self.GC)
	}
}
