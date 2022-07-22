package sqlmap

import (
	"fmt"
	"testing"
	"time"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	IsCheck bool `json:"is_check"`
	Val float64  `json:"val"`
}

func TestDemo(t *testing.T) {
	data := map[string]interface{}{"name":"1111", "age":"22", "is_check":1, "val":0}
	user := User{}
	stime := time.Now()
	for i := 0; i < 1000; i++ {
		err := WeakDecode(data, &user)
		fmt.Printf("%d %+v %v \r\n", i, user, err)
	}
	//34.5389ms
	fmt.Println(time.Since(stime))
}
