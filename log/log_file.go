package log

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type LogFileSt struct {
	stdLog *log.Logger
	errLog *log.Logger
	outfs  *os.File
	Prefix string `yaml:"prefix"`
	File string `yaml:"file"`
	Dir string `yaml:"dir"`
	Mask int8 `yaml:"mask"`
	symd string
}

type FileLog struct {
}

//工厂函数实例化注册
func (factory *FileLog) Open(params interface{}) IWriter {
	args   := params.(map[string]interface{})
	mask   := args["mask"].(int)
	dir    := args["dir"].(string)
	file   := args["file"].(string)
	prefix := args["prefix"].(string)
	logger := NewFileLog(int8(mask), dir, file, prefix)
	return IWriter(logger)
}

func init() {
	factory := &FileLog{}
	Register("file", IFactory(factory))
}

//生成一个文件日志实例
func NewFileLog(mask int8, dir, file, prefix string) *LogFileSt {
	logSt  := &LogFileSt{Prefix:prefix, File: file, Dir:dir, Mask:mask}
	logSt.Init() //完成结构的初始化操作处理逻辑
	return logSt
}

//完成数据的初始化处理逻辑
func (self *LogFileSt) Init() *LogFileSt {
	if self.Mask < 1 {
		self.Mask = ERROR
	}
	if self.Dir == "" {
		self.Dir = "./cachedir/log"
	}
	if self.File == "" {
		self.File = "default"
	}
	os.MkdirAll(self.Dir, 0777)
	return self.SetOutPutFile(true)
}

//设置前缀处理逻辑
func (self *LogFileSt) SetPrefix(prefix string) {
	self.Prefix = prefix
	if self.stdLog != nil {
		self.stdLog.SetPrefix(prefix)
	}
	if self.errLog != nil {
		self.errLog.SetPrefix(prefix)
	}
}

//重新创建文件处理句柄逻辑
func (self *LogFileSt) SetOutPutFile(isInit bool) *LogFileSt {
	symd := time.Now().Format("20060102")
	if !isInit && symd == self.symd {//日期不一致的话自动切换
		return self
	}
	file := fmt.Sprintf("%s/%s-%s.log", self.Dir, self.File, symd)
	fs, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		fs = os.Stdout
	}
	if self.outfs != nil {//关闭旧的句柄数据
		self.outfs.Close()
	}
	self.outfs = fs //记录句柄
	self.symd  = symd
	if self.stdLog != nil {
		self.stdLog.SetOutput(fs)
	} else {
		self.stdLog = log.New(fs, self.Prefix, log.Lshortfile|log.Ldate|log.Ltime)
	}
	return self
}

//错误输出文件句柄处理逻辑
func (self *LogFileSt) ErrorFs() *log.Logger {
	if self.errLog != nil {
		return self.errLog
	}
	file := fmt.Sprintf("%s/%s-error.log", self.Dir, self.File)
	fs, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		fs = os.Stdout
	}
	self.errLog = log.New(fs, self.Prefix, log.Lshortfile|log.Ldate|log.Ltime)
	return self.errLog
}

//记录日志
func (self *LogFileSt) Write(mask int8, v ...interface{}) {
	if mask <= ERROR {//失败的情况处理逻辑
		stackStr := []byte{}
		if mask != -1 {//非-1的情况
			stackStr = debug.Stack()
		}
		self.ErrorFs().Output(3, LevelName(mask) + fmt.Sprintln(v...)+string(stackStr))
	} else if mask <= self.Mask {
		self.SetOutPutFile(false)
		self.stdLog.Output(3, LevelName(mask) + fmt.Sprintln(v...))
	}
}

//格式化记录日志
func (self *LogFileSt) Writef(mask int8, format string, v ...interface{}) {
	if mask <= ERROR {//失败的情况处理逻辑
		stackStr := []byte{}
		if mask != -1 {//非-1的情况
			stackStr = debug.Stack()
		}
		self.ErrorFs().Output(3, LevelName(mask) + fmt.Sprintf(format, v...)+string(stackStr))
	} else if mask <= self.Mask {
		self.SetOutPutFile(false)
		self.stdLog.Output(3, LevelName(mask) + fmt.Sprintf(format, v...))
	}
}
