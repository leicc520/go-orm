package cache

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

type DataSt struct {
	Name string `json:"name"`
	Demo int64 `json:"demo"`
	Index int64 `json:"index"`
}

func TestJson(t *testing.T) {
	stime := time.Now()
	data := DataSt{Name: "leicc", Demo: 656}
	//g := sync.WaitGroup{}
	c := make(chan string, 1000)
	for i := 0; i < 1000; i++ {
		func(idx int, ch chan <- string) {
			lstr, err := json.Marshal(data)
			if err != nil {
				t.Log(err)
			}
			c <- string(lstr)+":"+strconv.FormatInt(int64(idx), 10)
		}(i, c)
	}
	nl := 0
	time.Sleep(time.Second)
	close(c)
	for {
		if lstr, ok := <-c; ok {
			nl++

			json.Unmarshal([]byte(lstr), &data)
			data.Index = int64(nl)
			fmt.Printf("%+v\r\n", data)
		} else {
			break
		}
	}
	//close(c)
	fmt.Println(nl, time.Since(stime))
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
