目录结构的说明
--cache   系统缓存类
--log     基础的日志库
--sqlmap  基础的map转struct 与 struct 转map的第三方依赖，直接整合进来
--根目录   三个类完成基础orm的封装，内置缓存的管理，以及数据库连接的基础配置

git tag 查看tag
git tag -a vx.x.x -m "test" 新建一个tag
git push origin vx.x.x  标签推送到服务器
go github私有仓库的配置


#这个步骤的操作是讲git https请求转换成协议转成git请求  需要在githunb上配置服务器的公钥
cat ~/.ssh/id_rsa.pub  #配置添加到githun sshkey当中
如果没有公钥文件生成一下
ssh-keygen -t rsa -C “your@email.com”

git config --global user.name "xxx"
git config --global user.email "your@email.com"
git config --global url."git@github.com:".insteadof https://github.com/


1、封装数据库的常规操作，有orm的影子，但是又不是强orm
insert/replace into [table] [value] [duplicate]
update [table] [value] [where]
delete [table] [where]
select [fields] from [table] [where] [group by] [having] [order by]

mysql.go 实现上诉语句的拆解封装，通过调用函数生成代码片段，最后执行的时候拼接成完整的sql语句执行,同时将要绑定的变量放到marks当中
允许通过Reset/Clear等方法清理重置sql片段   
具体参考测试代码片段
func TestMysql(t *testing.T) {
	sqlSt := NewMysql()
	//sqlSt := NewMysql().SetDriver(POSTGRES) 设置PG SLQ的特殊语法 默认Mysql语法

	args := struct {
		Name string
		Tag string
		Ename string
	}{"leicc", "A", "xxx"}

	//批量字段查询隐射到结构体
	tsql := sqlSt.Clear().Table("test").UseCond([]string{"Name", "Tag", "Ename"}, &args, OP_NE, OP_OR).AsSql("SELECT")
	t.Log(tsql)

	//批量相同字段映射到统一的查询条件
	tsql = sqlSt.Clear().Table("test").UseBatch([]string{"Name", "Tag", "Ename"}, "%leicc%", OP_LIKE, OP_OR).AsSql("SELECT")
	t.Log(tsql)

	sql := sqlSt.Clear().Table("test").Where("id", 1).AsSql("select")
	t.Log(sql)

	sql := sqlSt.Clear().Table("test").Where("id", 1).AsSql("delete")
	t.Log(sql)

	sql = sqlSt.Clear().Table("test").Value("id", 1).Value("name", "leicc").AsSql("insert")
	t.Log(sql)

	sql = sqlSt.Clear().Table("test").Value("name", "demo").Where("id", 1, OP_EQ).AsSql("update")
	t.Log(sql)

	sqlSt.Clear().Table("test", "a").Field("a.id, b.refid").Table("user", "b", "a.id=b.refid")
	sql = sqlSt.Where("a.name", "demo").GroupBy("a.refid").OrderBy("a.id", DESC).AsSql("select")
	t.Log(sql)
}

query.go 主要是集成mysqlST，增加数据据句柄达成真正的查询目的
Insert/Update/Delete/GetRow/GetList/GetItem/GetMap/NameMap/GetColumn/GetAsSql/GetValue
返回的数据类型依赖注入的field字段做映射，完成获取数据之后自动转换
func TestQuery(t *testing.T) {
	db, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1:3306)/admin?charset=utf8mb4")
	if err != nil {
		t.Error(err)
		return
	}

	fields := map[string]reflect.Kind{
	"id":			reflect.Int64,	//账号id
	"openid":		reflect.String,	//第三方Openid
	"account":		reflect.String,	//登录账号
	"avatar":		reflect.String,	//用户的头像信息
	"loginpw":		reflect.String,	//登录密码 要求客户端md5之后传到服务端做二次校验
	"sex":			reflect.Int8,	//性别 1-男 2-女
	"nickname":		reflect.String,	//昵称
	}
	query := NewQuery(fields).SetDb(db)
	query.Table("sys_user").Where("id", 1, OP_EQ).Field("id,account,nickname")
	data := query.GetRow()
	t.Log(data)

	user := struct {
	Id 	int64 `json:"id"`
	Account string `json:"account"`
	NickName string `json:"nickname"`
	}{}
	err = query.GetRow().ToStruct(&user)
	t.Log(err, user)

	query.Clear().Table("sys_user").Field("id,account,nickname")
	list := query.GetList("", 0 , 2)
	t.Log(list)

	query.Clear().Table("sys_user").Field("id").Where("id", []int64{1,2,3,4}, OP_IN)
	column := query.GetColumn("", 0, -1)
	t.Log(column)

	query.Clear().Table("sys_user").Field("id as `key`, account as `val`").Where("id", []int64{1,2,3,4}, OP_IN)
	asmap := query.GetMap("", 0, -1)
	t.Log(asmap)

	query.Clear().Table("sys_user").Field("id,account,nickname").Where("id", []int64{1,2,3,4}, OP_IN)
	nsmap := query.NameMap("", "id", 0, -1)
	t.Log(nsmap)
}

model.go主要集成query+数据库连接池+缓存策略+数据表模型 
实现表记录的增加、删除、修改、查询（取列表、取指定的列、取map结构等等）
缓存策略使用表级别的缓存版本号，一个表维护一个版本号，只要这个表有update/inster/delete 操作就更新版本号，这样这个表的所有数据都会自动失效。
例如GetList 操作缓存key user@hash(查询条件+版本号) 只要版本号变动，下一次取数据的时候缓存就无法命中，就可以取db，然后重新建立缓存，达到数据对数据的保护效果
model配置主/从数据库的获取配置的key信息，update/inster/delete 主库操作，select默认都是从库操作的。

通过自动化脚本生成orm
示例代码
package main

import (
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/cache"
)

func main() {
	cacheSt := cache.CacheConfigSt{"redis", "redis://:@127.0.0.1:6379/1"}
	dbmaster:= orm.DbConfig{"mysql", "root:@tcp(127.0.0.1:3306)/admin?charset=utf8mb4", 32, 32}
	config  := struct {
		Redis  string
		Cache  cache.CacheConfigSt
		DbMaster orm.DbConfig
		DbSlaver orm.DbConfig
	}{"redis://:@127.0.0.1:6379/1", cacheSt, dbmaster, dbmaster}
	orm.LoadDbConfig(config)//配置数据库结构注册到数据库调用配置当中
	orm.CreateOrmModels("dbmaster", "dbslaver", "./models")
}

数据库映射到代码的工具，将每个表生成model，放到指定的目录，提供给项目使用，配置Redis的话将会使用Redis作为缓存策略

2、每个表继承orm.modelSt,也就集成了这个结构体的方法，包含了插入、删除、修改、取列表等等操作，查询的话自动会缓存数据，更新的时候清理缓存数据，通过每个表维护一个版本号更新.

/****************************************************************************************
在这个类是动态生成，
*/
type demoUser struct {
	*ModelSt
}

//这里的dbPool
func newDemoUser() *demoUser {
	fields := map[string]reflect.Kind{
		"id":		reflect.Int64,	//账号id
		"openid":	reflect.String,	//第三方Openid
		"account":	reflect.String,	//登录账号
		...
		"stime":	reflect.Int,	//最后操作时间
	}

	args  := map[string]interface{}{
		"table":		"sys_user",
		"orgtable":		"sys_user",
		"prikey":		"id",
		"dbmaster":		"dbmaster",
		"dbslaver":		"dbslaver",
		"slot":			0,
	}

	data := &demoUser{&orm.ModelSt{}}
	data.Init(&orm.GdbPoolSt, args, fields)
	return data
}

4、根据id主键获取一条记录，默认返回SqlMap结构，可以加ToStruct转为结构体
	sorm := models.NewSysSyslog(dbCtx).SetYmTable("200601")
	data := struct {
		Id int `json:"id"`
		Ip string `json:"ip"`
		Msg string `json:"msg"`
		Stime int64 `json:"stime"`
	}{}
	err1 := sorm.GetOne(3).ToStruct(&data)
	fmt.Println(data, err1)

5、根据id主键更新记录	两种方式，直接使用SqlMap更新或者使用匿名函数设置要更新的字段
	sorm.Save(3, orm.SqlMap{"msg":"leicc"})
	sorm.SaveFromHandler(3, func(st *orm.QuerySt) *orm.QuerySt {
		st.Value("ip", "129.65.23.123")
		return st
	})
	err2 := sorm.GetOne(3).ToStruct(&data)
	fmt.Println(data, err2)

6、根据条件获取满足条件的某一条记录，同样使用匿名函数设置查询条件
	err3 := sorm.GetItem(func(st *orm.QuerySt) string {
		st.Where("id", 3)
		return st.GetWheres()
	}, "id,msg,ip,stime").ToStruct(&data)
	fmt.Println(data, "=========", err3)

7、根据获取获取列表，返回SqlMap切片列表
	list := sorm.GetList(0, 5, func(st *orm.QuerySt) string {
		st.Where("id", 3, orm.OP_GE)
		return st.GetWheres()
	}, "id,msg,ip,stime")
	fmt.Println(list)

8、根据条件返回表中指定的一列数据
	ids := sorm.GetColumn(0, 3, func(st *orm.QuerySt) string {
		return st.GetWheres()
	}, "id")
	fmt.Println(ids)

9、根据条件返回指定表的key=>val结构的map，例如返回字典id映射=>名称的map结构
	smap := sorm.GetAsMap(0, -1, func(st *orm.QuerySt) string {
		return st.GetWheres()
	}, "id as `key`, msg as `val`")
	fmt.Println(smap)

10、根据条件返回指定表的map结构，但是这里是key=>item(记录)SqlMap结构,指定一个key隐射到一条记录
	nmap := sorm.GetNameMap(0, 2, func(st *orm.QuerySt) string {
		return st.GetWheres()
	}, "id,ip,msg,stime", "id")
	fmt.Println(nmap)

11、获取查询聚合，例如count(1) 返回查询条件的记录数,SUM(xx)统计累计和的数据等
	total := sorm.GetTotal(func(st *orm.QuerySt) string {
		return st.GetWheres()
	}, "COUNT(1)").ToInt64()
	fmt.Println(total, "===========")

12、查询指定条件的记录是否存在，返回记录ID，例如判定名称是否被使用等等情况
	oldid := sorm.IsExists(func(st *orm.QuerySt) string {
		st.Where("id", 3333)
		return st.GetWheres()
	}).ToInt64()
	fmt.Println(oldid, "====")

13、根据SQL查询表记录，返回SqlMap切片结构
	sql := "SELECT * FROM "+sorm.GetTable()
	slist := sorm.GetAsSQL(sql, 0, 3)
	fmt.Println(slist, "=========end", sql)

14、根据条件设置执行update语句
	sorm.MultiUpdate(func(st *orm.QuerySt) string {
		st.Where("id", 3)
		return st.GetWheres()
	}, func(st *orm.QuerySt) *orm.QuerySt {
	st.Value("stime", time.Now().Unix())
	return st
	})

15、根据条件设置执行delete语句
	sorm.MultiDelete(func(st *orm.QuerySt) string {
		st.Where("id", 111, orm.OP_GE)
		return st.GetWheres()
	})
	return

16、分表策略的管理，这里支持三种分表策略取模/整除/日期归档(适合日志类)
	sorm ：= models.NewOsUser()
	sorm->SetModTable(id) //根据id取模做分表,slot=16代表总共16张分表0-15  id%16=?代表在第几张分表
	//这里会自动检查表是否存在，不存在的话创建表，然后做两层缓存结构，内存缓存、文件缓存，如果内存中记录表已经存在则跳过建表的操作
	sorm->SetDevTable(id) //根据id除法分表，例如slot=100w 代表1-100在分表 0 101-200分表1 以此类推
	sorm->SetYmTable(id) //根据年或者月份做分表归档处理逻辑

17、sqlmap分表主要利用一个开源的包做结构体到map 或者 map到结构体的逆向反转

18、开始go plugins的支持
	使用go plugins的话需要编译开启cgo的支持
	CGO_ENABLED=1 go build -buildmode=plugin -o greeter.so main.go
	CGO_ENABLED=1 go build -o main demo.go


19、git免密push
生成公钥-然后配置到githun->setting->ssh & GPG kyes->添加配置上去
ssh-keygen -t ed25519 -C "xxx@xxx.com"
或者
ssh-keygen -t rsa -C "xxxxxx@yy.com"

验证配置是否正确
ssh -T git@github.com

找到.gitconfig新增如下配置
[url "https://github.com"]   
    insteadOf = git://github.com