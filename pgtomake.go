package orm

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	
	"github.com/leicc520/go-orm/log"
)

//自动创建模型业务 DB隐射到Model
func CreatePSQLModels(dbmaster, dbslaver, gdir string) {
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
			//continue
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