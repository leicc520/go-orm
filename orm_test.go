package orm

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"reflect"
	"testing"

	_ "github.com/lib/pq"
)

func TestMysql(t *testing.T) {
	sqlSt := NewMysql().SetDriver(POSTGRES)

	args := struct {
		Name string
		Tag string
		Ename string
	}{"leicc", "A", "xxx"}

	tsql := sqlSt.Clear().Table("test").UseCond([]string{"Name", "Tag", "Ename"}, &args, OP_NE, OP_OR).AsSql("SELECT")
	t.Log(tsql)
	sql := sqlSt.Clear().Table("test").Where("id", 1).AsSql("select")
	t.Log(sql)
	return
	sql = sqlSt.Clear().Table("test").Value("id", 1).Value("name", "leicc").AsSql("insert")
	t.Log(sql)

	sql = sqlSt.Clear().Table("test").Value("name", "demo").Value("desc", "demoe222").Where("id", 1, OP_EQ).AsSql("update")
	t.Log(sql)

	sqlSt.Clear().Table("test", "a").Field("a.id,b.refid").Table("user", "b", "a.id=b.refid")
	sql = sqlSt.Where("a.name", "demo").GroupBy("a.refid").OrderBy("a.id", DESC).AsSql("select")
	t.Log(sql)
}

func TestQuery(t *testing.T) {
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=demo sslmode=disable")
	if err != nil {
		t.Error(err)
		return
	}
	fields := map[string]reflect.Kind{
		"id":		reflect.Int64,	//账号id
		"desc":		reflect.String,	//第三方Openid
		"idx":		reflect.Int8,	//性别 1-男 2-女
	}
	query := NewQuery(fields).SetDb(db)
	query.SetDriver(POSTGRES).Reset().Table("demo").Where("id", 1, OP_GT).Where("idx", "2,5", OP_BETWEEN).Where("desc", "%demo%", OP_LIKE).GroupBy("id").Field("id,\"desc\",idx")
	data := query.GetRow("")
	t.Log(data)

	query.Reset().Table("demo").Duplicate("desc", "demov99999").ConflictField("idx")
	lastid := query.SetAutoIncr("id").Insert(SqlMap{"desc":"demov3333", "idx":11}, false)
	fmt.Println(lastid)

	query.Reset().Table("demo").Where("id", 12)
	nrow := query.Update(SqlMap{"desc":"demo888", "idx":99})
    fmt.Println(nrow)

	query.Reset().Table("demo").Where("idx", "", OP_ISNULL)
	nrow = query.Delete()
	fmt.Println(nrow)
	return

	user := struct {
		Id 	int64 `json:"id"`
		Desc string `json:"desc"`
		Idx int64 `json:"idx"`
	}{}
	err = query.GetRow("").ToStruct(&user)
	t.Log(err, user)

	query.Clear().Table("demo").Field("id,\"desc\",idx")
	list := query.GetList("", 0 , 2)
	t.Log(list)

	query.Clear().Table("demo").Field("id").Where("id", []int64{1,2,3,4}, OP_IN)
	column := query.GetColumn("", 0, -1)
	t.Log(column)

	query.Clear().Table("demo").Field("id as \"key\", \"desc\" as \"val\"").Where("id", []int64{1,2,3,4}, OP_IN)
	asmap := query.GetMap("", 0, -1)
	t.Log(asmap)

	query.Clear().Table("demo").Field("id,\"desc\",idx").Where("id", []int64{1,2,3,4}, OP_IN)
	nsmap := query.NameMap("", "id", 0, -1)
	t.Log(nsmap)

}

/****************************************************************************************
	在这个类是动态生成，
 */
type demoTest struct {
	*ModelSt
}

//这里的dbPool
func newDemoUser() *demoTest {
	fields := map[string]reflect.Kind{
		"id":		reflect.Int64,	//账号id
		"desc":		reflect.String,	//第三方Openid
		"idx":		reflect.Int,	//最后操作时间
	}
	args  := map[string]interface{}{
		"table":		"demo",
		"orgtable":		"demo",
		"prikey":		"id",
		"dbmaster":		"dbmaster",
		"dbslaver":		"dbslaver",
		"slot":			10,
	}
	data := &demoTest{&ModelSt{}}
	data.Init(&GdbPoolSt, args, fields)
	return data
}

func TestModel(t *testing.T) {
	master := DbConfig{"postgres", "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=demo sslmode=disable", "dbmaster", 128, 64}
	slaver := DbConfig{"postgres", "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=demo sslmode=disable", "dbslaver", 128, 64}

	InitDBPoolSt().Set(master.SKey, &master)
	InitDBPoolSt().Set(slaver.SKey, &slaver)

	CreatePSQLModels(master.SKey, slaver.SKey, "../models/sys")
	return
	sorm := newDemoUser()

	table := sorm.DBTables()
	fmt.Println(table)
	//sorm.SetModTable(105).NewOne(SqlMap{"desc":"aaaa", "idx":999}, nil)
	sorm.SetYmTable(DATEYMDFormat).NewOne(SqlMap{"desc":"aaaa", "idx":999}, nil)
	return

	user := struct {
		Id 	int64 `json:"id"`
		Desc string `json:"desc"`
		Idx int64 `json:"idx"`
	}{}
	errv3 := sorm.GetItem(func(st *QuerySt) string {
		st.Where("id", 111)
		return st.GetWheres()
	}, "*").ToStruct(&user)
	fmt.Println(errv3, user)
	return

	nsizev2, errv2 := sorm.ChunkHandler(10, "id,idx", func(maps []SqlMap) error {
		fmt.Println(maps)
		return nil
	}, nil)
	fmt.Println(nsizev2, errv2)
	return
	query := "SELECT * FROM demo"
	listv2 := sorm.GetAsSQL(query, 0, -1)
	fmt.Println(listv2)
	return
	datav2 := sorm.GetValue(func(st *QuerySt) string {
		st.Where("id", 99)
		return st.GetWheres()
	}, "idx")
	fmt.Println(datav2)

	return
	for idx := 0; idx < 100; idx++ {
		nrow:= sorm.NewOne(SqlMap{
			"desc":"1111111", "idx":idx+99,
		}, nil)
		fmt.Println(nrow)
	}

	result := sorm.Query().Transaction(func(st *QuerySt) bool {
		arow := sorm.Delete(24)
		fmt.Println(arow)
		arows := sorm.MultiDelete(func(st *QuerySt) string {
			st.Where("id", 20, OP_LT)
			return st.GetWheres()
		})
		fmt.Println(arow, arows)
		return true
	})
	fmt.Println(result)

	return

	ids := sorm.GetColumn(1, 5, nil, "DISTINCT id")
	fmt.Println(ids)


	return
	data := sorm.GetOne(24)
	fmt.Println(data)
	sorm.Query().ConflictField("idx")
	id := sorm.NewOne(SqlMap{"desc":"xxxxx", "idx":88}, SqlMap{"desc": "vvvvvvvvv"})
	fmt.Println(id)
	id2 := sorm.NewOneFromHandler(func(st *QuerySt) *QuerySt {
		st.Value("desc", "xxxxxxxxx").Value("idx", 88)
		return st
	}, func(st *QuerySt) interface{} {
		st.ConflictField("idx")
		st.Duplicate("desc", "vvvvvvvvvvvvvvvvv")
		return nil
	})
	fmt.Println(id2)

	nsize := sorm.GetTotal(nil, "COUNT(1)")
	fmt.Println(nsize)
	
	list := sorm.GetList(1, 10, func(st *QuerySt) string {
		st.Where("idx", "", OP_ISNOTNULL)
		st.Where("id", 10, OP_GT)
		return st.GetWheres()
	}, "*")
	fmt.Println(list)

}

