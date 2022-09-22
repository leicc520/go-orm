package sqlmap

import (
	"reflect"
	"time"
)

//格式化返回时间处理逻辑
func StringToTimeStampParse(str string) (time.Time, error) {
	if dTime, err := time.Parse("2006-01-02T15:04:05Z", str); err == nil {
		return dTime, err
	}
	if dTime, err := time.Parse(time.RFC3339, str); err == nil {
		return dTime, err
	}
	if dTime, err := time.Parse(time.RFC3339Nano, str); err == nil {
		return dTime, err
	}
	if dTime, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
		return dTime, err
	}
	//默认返回的结构体逻辑业务
	return time.Parse(time.RFC1123, str)
}

// StringToTimeHookFunc returns a DecodeHookFunc that converts
func StringToTimeHookFuncV2() DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}
		if f.Kind() != reflect.String {
			return data, nil
		}
		return StringToTimeStampParse(data.(string))
	}
}