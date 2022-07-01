package log

import (
	"fmt"
	"log"
	"os"
)

type LogStdout struct {
	*log.Logger
	mask int8
}

type StdoutLog struct {
}

//工厂函数实例化注册
func (factory *StdoutLog) Open(params interface{}) IWriter {
	args := params.(map[string]interface{})
	mask := args["mask"].(int)
	prefix := args["prefix"].(string)
	logger := NewStdout(int8(mask), prefix)
	return IWriter(logger)
}

func init() {
	factory := &StdoutLog{}
	Register("stdout", IFactory(factory))
}

//生成一个文件日志实例
func NewStdout(mask int8, prefix string) *LogStdout {
	if mask < 1 {
		mask = ERROR
	}
	log := &LogStdout{log.New(os.Stdout, prefix, log.Lshortfile|log.Ldate|log.Ltime), mask}
	return log
}

func (self *LogStdout) Prefix(prefix string) {
	self.SetPrefix(prefix)
}

//记录日志
func (self *LogStdout) Write(mask int8, v ...interface{}) {
	if mask <= self.mask {
		self.Output(3, LevelName(mask) + fmt.Sprintln(v...))
	}
}

//格式化记录日志
func (self *LogStdout) Writef(mask int8, format string, v ...interface{}) {
	if mask <= self.mask {
		self.Output(3, LevelName(mask) + fmt.Sprintf(format, v...))
	}
}
