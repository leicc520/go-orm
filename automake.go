package orm

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leicc520/go-orm/log"
)

type fieldSt struct {
	name    string
	defualt interface{}
	comment string
	dtype   string
}

type modelSt struct {
	table   string
	prikey  string
	field   []fieldSt
}

//自动创建模型业务 DB隐射到Model
func CreateOrmModels(dbmaster, dbslaver, gdir string) {
	Gcolumnsql := "show full columns from %s"
	Gtablesql  := "show tables"
	if len(gdir) == 0 {
		gdir = "./models"
	}
	gdir, _ = filepath.Abs(gdir)
	if ss, err := os.Stat(gdir); err != nil || !ss.IsDir() {
		os.MkdirAll(gdir, 0777)
	}
	packStr := filepath.Base(gdir)
	db := InitDBPoolSt().NewEngine(dbmaster)
	tableSt, err := db.Query(Gtablesql)
	if err != nil {
		log.Write(log.ERROR, "sql query error "+Gtablesql + " "+err.Error())
		os.Exit(0)
	}
	defer func() {
		tableSt.Close()
		db.Close()
	}()
	//开始获取表结构信息
	model := modelSt{table: "", prikey: ""}
	for tableSt.Next() {
		tableSt.Scan(&model.table)
		ok, err := regexp.MatchString(`[\d]$`, model.table)
		if ok || err != nil {//直接跳过 不做处理
			continue
		}
		model.prikey = ""
		model.field  = make([]fieldSt, 0)
		//提取表的字段信息
		columnSt, _ := db.Query(fmt.Sprintf(Gcolumnsql, model.table))
		columns, _ := columnSt.Columns()
		values   := make([]sql.RawBytes, len(columns))
		rowSlice := make([]interface{}, len(columns))
		for idx := range values {
			rowSlice[idx] = &values[idx]
		}
		//遍历每个表的字段信息
		for columnSt.Next() {
			columnSt.Scan(rowSlice...)
			field := fieldSt{}
			for idx, value := range values {
				switch columns[idx] {
				case "Field":
					field.name = string(value)
				case "Type":
					field.dtype = string(value)
				case "Comment":
					field.comment = strings.ReplaceAll(string(value), "\n", ";")
					field.comment = strings.ReplaceAll(field.comment, "\r", "")
				case "Default":
					field.defualt = string(value)
				case "Key":
					if string(value) == "PRI" {
						model.prikey = field.name
					}
				}
			}
			model.field = append(model.field, field)
		}
		buildModels(packStr, gdir, dbmaster, dbslaver, &model)
	}
}

//获取结构体 数据资料信息
func getFieldsString(field fieldSt) (string, string) {
	lstr := "\t\t\""+field.name+"\":\t\t"
	sstr := "\t"+CamelCase(strings.ReplaceAll(field.name, "-", "_"))+"\t\t"
	tystr := strings.ToLower(field.dtype)
	if strings.Contains(tystr, "int") {
		if strings.Contains(tystr, "unsigned") {
			if strings.Contains(tystr, "bigint") || strings.Contains(tystr, "int8") {
				lstr += "reflect.Uint64,"
				sstr += "uint64"
			} else if strings.Contains(tystr, "tinyint") {
				lstr += "reflect.Uint8,"
				sstr += "uint8"
			} else {
				lstr += "reflect.Uint,"
				sstr += "uint"
			}
		} else {
			if strings.Contains(tystr, "bigint") || strings.Contains(tystr, "int8") {
				lstr += "reflect.Int64,"
				sstr += "int64"
			} else if strings.Contains(tystr, "tinyint") {
				lstr += "reflect.Int8,"
				sstr += "int8"
			} else {
				lstr += "reflect.Int,"
				sstr += "int"
			}
		}
	} else if strings.Contains(tystr, "double") || strings.Contains(tystr, "float") || strings.Contains(tystr, "decimal") {
		lstr += "reflect.Float64,"
		sstr += "float64"
	} else {
		lstr += "reflect.String,"
		sstr += "string"
	}
	sstr += "\t\t`json:\""+field.name+"\"`\t\t"
	lstr += "\t\t//"+strings.TrimSpace(field.comment)
	return lstr, sstr
}

func buildModels(packStr, dir, dbmaster, dbslaver string, model *modelSt) {
	class := CamelCase(model.table)
	astr  := make([]string, 0, len(model.field))
	vstr  := make([]string, 0, len(model.field))
	for idx, _ := range model.field {
		tstr, sstr := getFieldsString(model.field[idx])
		astr  = append(astr, tstr)
		vstr  = append(vstr, sstr)
	}
	lstr, rstr := "", ""
	gofile  := filepath.Join(dir, model.table+".go")
	bstr, err := ioutil.ReadFile(gofile)
	if err == nil && len(bstr) > 0 {
		lstr = string(bstr)
		lstr = regexp.MustCompile(",[\\s]*//[^\n]+").ReplaceAllString(lstr, rstr)
		rstr = "reflect.Kind{\n"+strings.Join(astr, "\n")+ "\n\t}"
		lstr = regexp.MustCompile("reflect\\.Kind\\{[^\\}]+\\}").ReplaceAllString(lstr, rstr)
		rstr = "St struct {\n"+strings.Join(vstr, "\n")+"\n}"
		lstr = regexp.MustCompile("St[\\s]+struct[\\s]+\\{[^\\}]+\\}").ReplaceAllString(lstr, rstr)
		rstr = "\"table\":\t\t\""+model.table+"\""
		lstr = regexp.MustCompile("\"table\":\t\t\"[^\"]+\"").ReplaceAllString(lstr, rstr)
		rstr = "\"orgtable\":\t\t\""+model.table+"\""
		lstr = regexp.MustCompile("\"orgtable\":\t\t\"[^\"]+\"").ReplaceAllString(lstr, rstr)
		rstr = "\"prikey\":\t\t\""+model.prikey+"\""
		lstr = regexp.MustCompile("\"prikey\":\t\t\"[^\"]+\"").ReplaceAllString(lstr, rstr)
		rstr = "\"dbmaster\":\t\t\""+dbmaster+"\""
		lstr = regexp.MustCompile("\"dbmaster\":\t\t\"[^\"]+\"").ReplaceAllString(lstr, rstr)
		rstr = "\"dbslaver\":\t\t\""+dbslaver+"\""
		lstr = regexp.MustCompile("\"dbslaver\":\t\t\"[^\"]+\"").ReplaceAllString(lstr, rstr)
	} else {//新生成数据的情况
		lstr   = strings.Replace(Gmodelstpl, "{package}", packStr, -1)
		lstr   = strings.Replace(lstr, "{gfields}", strings.Join(astr, "\n"), -1)
		lstr   = strings.Replace(lstr, "{xfields}", strings.Join(vstr, "\n"), -1)
		lstr   = strings.Replace(lstr, "{struct}", class, -1)
		lstr   = strings.Replace(lstr, "{table}", model.table, -1)
		lstr   = strings.Replace(lstr, "{prikey}", model.prikey, -1)
		lstr   = strings.Replace(lstr, "{dbmaster}", dbmaster, -1)
		lstr   = strings.Replace(lstr, "{dbslaver}", dbslaver, -1)
		lstr   = strings.Replace(lstr, "{dbslot}", "0", -1)
	}
	os.Remove(gofile)
	ioutil.WriteFile(gofile, []byte(lstr), 0777)
	log.Write(log.DEBUG, "db "+dbmaster+" create models "+model.table+" build success!")
}

var Gmodelstpl = `package {package}

import (
	"reflect"
	"github.com/leicc520/go-orm"
)

type {struct} struct {
	*orm.ModelSt
}

//结构体实例的结构说明
type {struct}St struct {
{xfields}
}

//这里默认引用全局的连接池句柄
func New{struct}() *{struct} {
	fields := map[string]reflect.Kind{
{gfields}
	}
	
	args  := map[string]interface{}{
		"table":		"{table}",
		"orgtable":		"{table}",
		"prikey":		"{prikey}",
		"dbmaster":		"{dbmaster}",
		"dbslaver":		"{dbslaver}",
		"slot":			{dbslot},
	}

	data := &{struct}{&orm.ModelSt{}}
	data.Init(&orm.GdbPoolSt, args, fields)
	return data
}
`

