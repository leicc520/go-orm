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

// @Summary 自动创建模型业务 DB隐射到Model
// @Tags 创建数据模型
// @param dbmaster string 主数据库连接名称
// @param dbslaver string 从数据库连接名称
// @param gdir string 模型的存放路径
// @param etables string 导致指定的表，默认空所有
func CreateMYSQLModels(dbmaster, dbslaver, gdir, etables string) {
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
		if len(etables) > 0 && !strings.Contains(etables, model.table) {
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

// @Summary 自动创建模型业务 DB隐射到Model
// @Tags 创建数据模型
// @param dbmaster string 主数据库连接名称
// @param dbslaver string 从数据库连接名称
// @param gdir string 模型的存放路径
// @param etables string 导致指定的表，默认空所有
func CreatePGSQLModels(dbmaster, dbslaver, gdir, etables string) {
	Gcolumnsql := `
SELECT a.column_name,a.data_type,a.column_default,b.description,c.ctype FROM information_schema.columns AS a LEFT JOIN (
	SELECT A.attname, D.description FROM pg_class as C, pg_attribute as A, pg_description as D
		WHERE A.attrelid = C.oid AND d.objoid = A.attrelid AND d.objsubid = A.attnum AND C.relname = '%s'
	) AS b ON a.column_name=b.attname
	LEFT JOIN (
	SELECT pg_attribute.attname AS attname,pg_constraint.contype AS ctype FROM pg_constraint 
	INNER JOIN pg_class ON pg_constraint.conrelid = pg_class.oid
	INNER JOIN pg_attribute ON pg_attribute.attrelid = pg_class.oid AND  pg_attribute.attnum = pg_constraint.conkey[1]
	INNER JOIN pg_type ON pg_type.oid = pg_attribute.atttypid
	WHERE pg_class.relname = '%s' AND pg_constraint.contype='p'
	) AS c ON a.column_name=c.attname
WHERE a.table_schema='public' and a.table_name='%s'
`
	Gtablesql  := "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'"
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
		if len(etables) > 0 && !strings.Contains(etables, model.table) {
			continue
		}
		model.prikey = ""
		model.field  = make([]fieldSt, 0)
		query := fmt.Sprintf(Gcolumnsql, model.table, model.table, model.table)
		columnSt, _ := db.Query(query)
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
			priStr:= ""
			for idx, value := range values {
				switch columns[idx] {
				case "column_name":
					field.name = string(value)
				case "data_type":
					field.dtype = string(value)
				case "description":
					field.comment = string(value)
				case "ctype":
					priStr = string(value)
				case "column_default":
					cstr := string(value)
					if strings.Index(cstr, "nextval") == 0{
						model.prikey  = field.name
					} else {
						field.defualt = cstr
					}
				}
			}
			if priStr == "p" {//主键字段
				model.prikey  = field.name
			}
			model.field = append(model.field, field)
		}
		//提取表的字段信息
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
	} else if strings.Contains(tystr, "timestamp") {
		lstr += "orm.DT_TIMESTAMP,"
		sstr += "time.Time"
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
		if strings.Contains(strings.Join(vstr, ";"), "\ttime.Time\t") {//开启日期的包
			lstr   = strings.Replace(lstr, "{time}", "\r\n\t\"time\"", -1)
		} else {
			lstr   = strings.Replace(lstr, "{time}", "", -1)
		}
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
	"reflect"{time}
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

