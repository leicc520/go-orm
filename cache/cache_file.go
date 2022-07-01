package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leicc520/go-orm/log"
)

var GFileCacheDir = ""
type FactoryFileCache struct {
}

//获取缓存日志配置信息
func (factory *FactoryFileCache) Open(params interface{}) Cacher {
	dirPath, dirDept := "./cache", 1
	if dirConfig, ok := params.(map[string]interface{}); ok {
		dirPath = dirConfig["dir"].(string)
		dirDept = dirConfig["dept"].(int)
	} else if dirArgs, ok := params.(string); ok {
		args := strings.Split(dirArgs, "=")
		if len(args) > 1 {
			dirPath = args[0]
			if len(args) == 2 {
				if nDept, err := strconv.ParseInt(args[1], 10, 64); err != nil {
					dirDept = int(nDept)
				}
			}
		}
	}
	cache := NewFileCache(dirPath, dirDept)
	return Cacher(cache)
}

func init() {
	factory := &FactoryFileCache{}
	Register("file", IFactoryCache(factory))
}

type FileCacheSt struct {
	dir     string
	dirDept int
}

//初始化一个Cache实例
func NewFileCache(dir string, dirDept int) *FileCacheSt {
	if dirDept < 1 || dirDept > 5 {
		dirDept = 1
	}
	if len(GFileCacheDir) > 0 {//有配置缓存地址的情况
		dir = GFileCacheDir
	}
	if _, err := os.Stat(dir); err != nil {
		dir = "/tmp"
	}
	cache := &FileCacheSt{dir: dir, dirDept: dirDept}
	return cache
}

//定位Key映射到的目录文件
func (self *FileCacheSt) matchFile(key string) string {
	npos := strings.IndexByte(key, '@')
	dir  := self.dir
	if npos > 3 {//切一级目录
		dir = filepath.Join(dir, key[0:npos])
	}
	md5Str := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	for i := 0; i < self.dirDept; i++ {
		seg := i * 3
		dir = filepath.Join(dir, md5Str[seg:seg+3])
	}
	os.MkdirAll(dir,0777)
	file := filepath.Join(dir, key)
	return file
}

//获取一个缓存记录信息，过期记录不返回
func (self *FileCacheSt) Get(key string) interface{} {
	var item CacheItemSt
	file := self.matchFile(key)
	vstr, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(vstr, &item); err != nil {
		os.Remove(file)
		return nil
	}
	//判断是否过期
	if item.Expire > 0 && item.Expire < time.Now().Unix() {
		os.Remove(file)
		return nil
	}
	return item.Data
}

//直接获取数据并解析到结构当中
func (self *FileCacheSt) GetStruct(key string, out interface{}) error {
	data := self.Get(key) //获取数据
	if data == nil {//数据不存在的情况
		return ErrNotExists{}
	}
	err := ToStruct(data, out)
	return err
}

//设置一个缓存记录
func (self *FileCacheSt) Set(key string, data interface{}, expire int64) bool {
	file  := self.matchFile(key)
	ntime := time.Now().Unix()
	if expire > 0 && expire < ntime {
		expire += ntime
	}
	item := &CacheItemSt{Expire: expire, Data: data}
	vstr, err := json.Marshal(item)
	if err != nil {
		log.Write(log.ERROR, "golang json encode error "+err.Error())
		return false
	}
	err = ioutil.WriteFile(file, vstr, 0777)
	if err != nil {
		log.Write(log.ERROR, file, "cache writer file error "+err.Error())
		return false
	}
	return true
}

//删除缓存记录信息
func (self *FileCacheSt) Del(keys ...string) bool {
	for i := 0; i < len(keys); i++ {
		file := self.matchFile(keys[i])
		if err := os.Remove(file); err != nil {
			return false
		}
	}
	return true
}

//清理内存缓存记录
func (self *FileCacheSt) Clear() {
	dir, _ := filepath.Abs(self.dir)
	os.RemoveAll(dir)
}

//清理内存缓存记录
func (self *FileCacheSt) Close() {
	//todo 释放资源
}