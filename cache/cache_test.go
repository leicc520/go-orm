package cache

import (
	"fmt"
	"testing"
	"time"
)

type DataSt struct {
	Name string `json:"name"`
	Demo int64 `json:"demo"`
}

func Test_mcache(t *testing.T) {
	cache := Factory("memory", map[string]interface{}{"gc": time.Second*10})
	cache.Set("leicc", "simlife", 3)
	fmt.Println(cache.Get("leicc"))
	time.Sleep(time.Second * 1)
	fmt.Println(cache.Get("leicc"))

	data := DataSt{Demo: 111, Name: "leicc"}
	cache.Set("data", data, 30)

	data2 := DataSt{}
	err := cache.GetStruct("data", &data2)
	fmt.Println(data2, err)
	fmt.Println(cache)
}

func Test_fcache(t *testing.T) {
	cache := Factory("file", map[string]interface{}{"dir": "./cache", "dept": 2})
	cache.Set("leicc", map[string]interface{}{"dir": "./cache", "dept": 2}, 60)
	data := cache.Get("leicc")
	fmt.Println(data)
	//cache.Del("leicc")
	data = cache.Get("leicc")
	fmt.Println(data)

	cache.Clear()
}
