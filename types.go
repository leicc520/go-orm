package orm

import (
	"errors"
	"strconv"

	"github.com/leicc520/go-orm/log"
	"github.com/leicc520/go-orm/sqlmap"
)

type SqlString string
type SqlMap map[string]interface{}

//map 直接转成结构体返回
func (s SqlMap) ToStruct(stPtr interface{}) error {
	if s == nil || len(s) < 1 {
		return errors.New("sqlmap to struct data is nil")
	}
	if err := sqlmap.WeakDecode(s, stPtr); err != nil {
		log.Write(log.ERROR, "convert sqlmap to struct error; "+err.Error())
		return err
	}
	return nil
}

//格式化处理逻辑
func (s SqlMap) Merge(m SqlMap) SqlMap {
	for key, val := range m {
		s[key] = val
	}
	return s
}

//删除执行的key信息
func (s SqlMap) Delete(keys... string) {
	for _, key := range keys {
		delete(s, key)
	}
}

//清空sqlmap
func (s SqlMap) Clear() {
	for key, _ := range s {
		delete(s, key)
	}
}

//判断是否为空对象
func (s SqlMap) IsNil() bool {
	if s != nil && len(s) > 0 {
		return false
	}
	return true
}

//强行转成整数
func (s SqlString) ToInt64() int64 {
	if s == "" {
		return 0
	}
	tmpStr := string(s)
	if n, err := strconv.ParseInt(tmpStr, 10, 64); err != nil {
		return -1
	} else {
		return n
	}
}

//强行转成整数
func (s SqlString) ToFloat64() float64 {
	if s == "" {
		return 0.0
	}
	tmpStr := string(s)
	if n, err := strconv.ParseFloat(tmpStr, 64); err != nil {
		return -1
	} else {
		return n
	}
}

//强行转成整数
func (s SqlString) ToString() string {
	return string(s)
}
