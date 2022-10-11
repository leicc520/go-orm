package orm

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"time"

	"github.com/leicc520/go-orm/log"
	"github.com/leicc520/go-orm/sqlmap"
)

type SqlString string
type SqlMap map[string]interface{}
type SqlMapSliceSt []SqlMap
type SqlTime time.Time

//格式化输出日期 --实现数据库的Value接口
func (t SqlTime) Value() (driver.Value, error) {
	return time.Time(t).Format(DATEZONEFormat), nil
}

//格式化输出日期
func (t SqlTime) String() string {
	return time.Time(t).Format(DATEZONEFormat)
}

//清理列表缓存的策略
func (s SqlMapSliceSt) Clear()  {
	for _, item := range s {
		item.Clear()
	}
}

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
		//相同key的map直接覆盖合并
		if vData, ok := val.(SqlMap); ok {
			if eVal, ok := s[key]; ok {
				if eData, ok := eVal.(SqlMap); ok {
					s[key] = eData.Merge(vData)
					continue
				}
			}
		}
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
	for key, val := range s {
		if tmp, ok := val.(SqlMap); ok {
			tmp.Clear()
		}
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
