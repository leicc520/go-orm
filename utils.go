package orm

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DATEBASICFormat  = "2006-01-02 15:04:05"
	DATEYMDFormat    = "20060102"
	DATESTRFormat    = "20060102150405"
	DATEYMDSTRFormat = "2006-01-02"
	randStr  = "aqwertyuioplkjhgfdsazxcvbnm1234567890QWERTYUIOPLKJHGFDSAZXCVBNM"
	numberStr= "1234567890"
)

//根据逗号分割的id转slice
func IdStr2Slice(idstr, seg, omit string) []string {
	idstr  = strings.TrimSpace(idstr)
	if len(idstr) < 1 {
		return []string{omit}
	}
	arstr := make([]string, 0)
	tpstr := strings.Split(idstr, seg)
	for _, idstr = range tpstr {//验证id是大于的记录
		if n, err := strconv.ParseInt(idstr, 10, 64); err == nil && n > 0 {
			arstr = append(arstr, idstr)
		}
	}
	return arstr
}

//简单的字符混淆
func SwapStringCrypt(lstr string) string {
	bstr  := []byte(lstr)
	nsize := len(bstr) / 2
	for i := 0; i < nsize; i++ {
		if i % 2 == 0 {
			continue
		}
		bstr[i], bstr[nsize+i] = bstr[nsize+i], bstr[i]
	}
	return string(bstr)
}

//生成随机字符串
func RandString(nLen int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, nLen)
	nSize := len(randStr)
	for i := 0; i < nLen; i++ {
		idx := r.Int() % nSize
		bytes[i] = randStr[idx]
	}
	return string(bytes)
}

//生成随机字符串
func NumberString(nLen int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, nLen)
	nSize := len(numberStr)
	for i := 0; i < nLen; i++ {
		idx := r.Int() % nSize
		bytes[i] = numberStr[idx]
	}
	return string(bytes)
}

//获取客户端IP
func RemoteIp(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get("XRealIP"); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get("XForwardedFor"); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}
	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}
	return remoteAddr
}

//将地址切割到path部分
func CutPath(href string) string {
	if strings.HasPrefix(href, "http") {
		npos := strings.Index(href[8:], "/")
		if npos != -1 {
			href = href[npos:]
		}
	}
	return href
}

//根据时间格式获取unix时间的记录
func DT2UnixTimeStamp(sdt, format string) int64 {
	if sdt == "" {
		return -1
	}
	if format == "" {
		format = DATEBASICFormat
	}
	st, err := time.ParseInLocation(format, sdt, time.Local)
	if err != nil {
		return -1
	}
	return st.Unix()
}

//这个时间格式填写golang诞辰 2006-01-02 15:04:05 等
func TimeStampFormat(sec int64, layout string) string {
	return time.Unix(sec, 0).Format(layout)
}

//判断文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

//下划线转成驼峰的格式
func CamelCase(str string) string {
	bstr := []byte(str)
	nlen := len(bstr)
	j := 0
	for i := 0; i < nlen; i++ {
		if i == 0 && bstr[i] >= 'a' && bstr[i] <= 'z' {
			bstr[i] -= 32
		} else if bstr[i] == '_' && bstr[i+1] >= 'a' && bstr[i+1] <= 'z' {
			bstr[i+1] -= 32
			i++
		} else {//其它位置出现大写字母统一转小写
			if bstr[i] >= 'A' && bstr[i] <= 'Z' {
				bstr[i] += 32
			}
		}
		bstr[j] = bstr[i]
		j++
	}
	if j < nlen {
		bstr = bstr[:j]
	}
	return string(bstr)
}

//将驼峰的命名格式反转过来
func SnakeCase(str string) string {
	bstr := []byte(str)
	nlen := len(bstr)
	rstr := make([]byte, 0)
	j := 0
	for i := 0; i < nlen; i++ {
		if i == 0 && bstr[i] >= 'A' && bstr[i] <= 'Z' {
			rstr = append(rstr, bstr[i]+32)
		} else if bstr[i] >= 'A' && bstr[i] <= 'Z' {
			rstr = append(rstr, '_', bstr[i]+32)
			j++
		} else {
			rstr = append(rstr, bstr[i])
		}
		j++
	}
	return string(rstr)
}

//返回本机的随机IP一个内网IP地址
func GetLocalRandomIPv4() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, add := range addrs {
		ipNet, ok := add.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String(), nil
		}
	}
	return "", errors.New("local ip not exists!")
}

//获取服务器panic 指定情况的获取写日志
func RuntimeStack(skip int) []byte {
	buf := new(bytes.Buffer)
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", func(pcPtr uintptr) []byte {
			fn := runtime.FuncForPC(pcPtr)
			if fn == nil {
				return []byte("no")
			}
			name := []byte(fn.Name())
			return name
		}(pc) , func(lines [][]byte, n int) []byte {
			n--
			if n < 0 || n >= len(lines) {
				return []byte("no")
			}
			return bytes.TrimSpace(lines[n])
		}(lines, line))
	}
	return buf.Bytes()
}
