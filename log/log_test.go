package log

import (
	"testing"
)

func Test_stdout(t *testing.T) {
	log := Factory("std", map[string]interface{}{"mask": DEBUG, "prefix":"stdout"})
	log.Write(ERROR, "aaaaaaaaa", map[string]string{"aaa": "dsafsdfsadf"})
	log.Writef(DEBUG, "%40s-%d", "leicc", 10)
}

func Test_file(t *testing.T) {
	log := Factory("file", map[string]interface{}{"mask": ERROR, "dir": "./cache", "file": "default", "prefix":"stdout"})
	log.Write(FATAL, "aaaaaaaaa", map[string]string{"aaa": "dsafsdfsadf"})
	log.Writef(ERROR, "%40s-%d", "leicc", 10)
}
