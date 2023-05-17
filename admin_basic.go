package main

import (
	"fmt"
	"github.com/gohouse/converter"
	"github.com/gookit/goutil/stdutil"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tidwall/gjson"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var adminApi *AdminApi[any]

var adminDataError *AdminDataError

func init() {
	godotenv.Load("pro.env")
	adminApi = NewAdminApi()
}

func TestFoo(t *testing.T) {
	fmt.Println(getMacAddr())
	fmt.Println(getMacAddr()[12:17])
	fmt.Println(runtime.GOOS)
	// 初始化
	t2t := converter.NewTable2Struct()
	// 个性化配置
	t2t.Config(&converter.T2tConfig{
		// 如果字段首字母本来就是大写, 就不添加tag, 默认false添加, true不添加
		RmTagIfUcFirsted: false,
		// tag的字段名字是否转换为小写, 如果本身有大写字母的话, 默认false不转
		TagToLower: false,
		// 字段首字母大写的同时, 是否要把其他字母转换为小写,默认false不转换
		UcFirstOnly: false,
		//// 每个struct放入单独的文件,默认false,放入同一个文件(暂未提供)
		//SeperatFile: false,
	})
	// 开始迁移转换
	err := t2t.
		// 指定某个表,如果不指定,则默认全部表都迁移
		Table("user").
		// 表前缀
		Prefix("tp_").
		// 是否添加json tag
		EnableJsonTag(true).
		// 生成struct的包名(默认为空的话, 则取名为: package model)
		PackageName("model").
		// tag字段的key值,默认是orm
		TagKey("orm").
		TagKey("form").
		// 是否添加结构体方法获取表名
		RealNameMethod("TableName").
		// 生成的结构体保存路径
		SavePath("./model/user.go").
		// 数据库dsn,这里可以使用 t2t.DB() 代替,参数为 *sql.DB 对象
		Dsn("root:@tcp(localhost:3306)/fast-admin?charset=utf8").
		// 执行
		Run()

	fmt.Println(err)
}
func getMacAddr() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err.Error()
	}
	maxIndexInterface := interfaces[0]
	for _, inter := range interfaces {
		if inter.HardwareAddr == nil {
			continue
		}
		if inter.Flags&net.FlagUp == 1 {
			maxIndexInterface = inter
		}
	}
	return maxIndexInterface.HardwareAddr.String()
}

// 测试对象名称列表, 用于命令行参数映射
//
//	var testCaseList = map[string]func() TestCase{
//		"CreateLocation": func() TestCase { return &CreateLocation{} },
//		"DeleteLocation": func() TestCase { return &DeleteLocation{} },
//	}
//
// var testCaseName string
//
//	func init() {
//		flag.StringVar(
//			&testCaseName,
//			"t",
//			"DeleteLocation",
//			"测试对象名称,多个使用逗号分割\n"+strings.Join(maps.Keys(testCaseList), "\n")+"\n",
//		)
//	}
//
//	func Test(t *testing.T) {
//		flag.Parse()
//		testCaseNameList := strings.Split(testCaseName, ",")
//		for _, name := range testCaseNameList {
//			testCaseList[name]().Run(t)
//		}
//	}

func pp(items ...interface{}) (written int, err error) {
	convey.Println()
	return convey.Print(items...)
}

func assertOk(ret *ApiRet, err error) {
	ret, err = RetryIfNotSignedIn(ret, err)
	if err != nil && !errors.As(err, &adminDataError) {
		log.Errorf("%+v", err)
	}
	convey.SoMsg(stdutil.GetCallerInfo(2), err, convey.ShouldBeNil)
}
func assertHasOne(ret *ApiRet, err error) *ApiRet {
	ret, err = RetryIfNotSignedIn(ret, err)
	if err != nil && !errors.As(err, &adminDataError) {
		log.Errorf("%+v", err)
	}
	convey.SoMsg(stdutil.GetCallerInfo(2), err, convey.ShouldBeNil)
	convey.SoMsg(stdutil.GetCallerInfo(2)+" 没有找到任何记录", gjson.Get(ret.Body, "total").Int(), convey.ShouldBeGreaterThan, 0)
	return ret
}
func RetryIfNotSignedIn(ret *ApiRet, err error) (*ApiRet, error) {
	if err != nil && ret != nil && (strings.Contains(ret.Resp.Request.URL.String(), "/login") || gjson.Get(ret.Body, "msg").String() == "请登录后操作") {
		pp("登录")
		_, err2 := adminApi.SignIn(os.Getenv("SIGN_IN_USERNAME"), os.Getenv("SIGN_IN_PASSWORD"))
		convey.So(err2, convey.ShouldBeNil)
		pp("再次请求")
		callResult := reflect.ValueOf(adminApi).MethodByName(ret.Method).Call(ret.Args)
		ret, _ = callResult[0].Interface().(*ApiRet)
		err, _ = callResult[1].Interface().(error)
	}
	return ret, err
}
