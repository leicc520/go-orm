package log

import (
	"fmt"
	"os"
	"strconv"
)

const (
	_ = iota
	FATAL
	ERROR
	DEBUG
	INFO
)

type IWriter interface {
	SetPrefix(prefix string)
	Write(mask int8, v ...interface{})
	Writef(mask int8, format string, v ...interface{})
}

type IFactory interface {
	Open(params interface{}) IWriter
}

var (
	gMask			 = int8(INFO)
	gDrivers         = map[string]IFactory{}
	Logger   IWriter = nil
)

//初始化环境遍历
func init() {
	mask := os.Getenv("_LOGMASK_")
	if mask != "" && len(mask) > 1 {
		if s, err := strconv.ParseInt(mask, 10, 64); err == nil {
			if s > INFO {
				s = INFO
			} else if s < ERROR {
				s = ERROR
			}
			gMask = int8(s)
		}
	}
}

//注册Log到注册器中
func Register(name string, driver IFactory) {
	if driver == nil {
		panic("log: Register driver is nil")
	}
	if _, dup := gDrivers[name]; dup {
		panic("log: Register called twice for driver " + name)
	}
	gDrivers[name] = driver
}

//生成一个Cache执行实例
func Factory(name string, params interface{}) IWriter {
	if _, ok := gDrivers[name]; !ok {
		panic("log: Factory not exists driver " + name)
	}
	return gDrivers[name].Open(params)
}

//记录日志
func Write(mask int8, v ...interface{}) {
	if Logger == nil {
		Logger = NewStdout(gMask, "[web]")
	}
	Logger.Write(mask, v...)
}

//格式化记录日志
func Writef(mask int8, format string, v ...interface{}) {
	if Logger == nil {
		Logger = NewStdout(gMask, "[web]")
	}
	Logger.Writef(mask, format, v...)
}

//设置前缀数据资料信息
func SetPrefix(prefix string) {
	if Logger != nil {
		Logger.SetPrefix(prefix)
	}
}

//设置全局的日志
func SetLogger(logger IWriter) {
	Logger = logger
}

//获取文件的错误信息
func ErrorLog() string {
	logSt, ok := Logger.(*LogFileSt)
	if !ok {
		return "OK"
	}
	file := fmt.Sprintf("%s/%s-error.log", logSt.Dir, logSt.File)
	fp, err := os.Open(file)
	if err != nil {
		return err.Error()
	}
	defer fp.Close()
	nSize   := int64(1024*1024*2)
	stat, _ := os.Stat(file)
	startAt := stat.Size() - nSize
	if startAt < 0 {
		startAt = 0
	}
	buf := make([]byte, nSize)
	_, err = fp.ReadAt(buf, startAt)
	if err != nil {
		return err.Error()
	}
	return string(buf)
}

//获取日志等级名称
func LevelName(mask int8) string {
	switch mask {
	case FATAL:
		return "FATAL:"
	case ERROR:
		return "ERROR:"
	case DEBUG:
		return "DEBUG:"
	case INFO:
		return "INFO:"
	}
	return "WARNING:"
}
