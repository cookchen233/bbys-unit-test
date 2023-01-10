package main

import (
	"fmt"
	"github.com/gookit/goutil"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tidwall/gjson"
	"io"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

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

var adminApi = NewAdminApi()

var adminDataError *AdminDataError

func assertOk(ret *ApiRet, err error) {
	ret, err = RetryIfNotSignedIn(ret, err)
	if err != nil && !errors.As(err, &adminDataError) {
		log.Errorf("%+v", err)
	}
	So(err, ShouldBeNil)
	SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "code").Int(), ShouldNotBeZeroValue)
}
func assertHasOne(ret *ApiRet, err error) {
	ret, err = RetryIfNotSignedIn(ret, err)
	if err != nil && !errors.As(err, &adminDataError) {
		log.Errorf("%+v", err)
	}
	So(err, ShouldBeNil)
	SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), ShouldBeGreaterThan, 0)
}
func RetryIfNotSignedIn(ret *ApiRet, err error) (*ApiRet, error) {
	if err != nil && ret != nil && strings.Contains(ret.Resp.Request.URL.String(), "/login") {
		pp("登录")
		ret2, _ := adminApi.SignIn(os.Getenv("SIGN_IN_USERNAME"), os.Getenv("SIGN_IN_PASSWORD"))
		if ret2.Resp.Request.Response == nil {
			SoMsg(gjson.Get(ret2.Body, "msg").String(), gjson.Get(ret2.Body, "code").Int(), ShouldNotBeZeroValue)
		}
		pp("再次请求")
		callResult := reflect.ValueOf(adminApi).MethodByName(ret.Method).Call(ret.Args)
		ret = callResult[0].Interface().(*ApiRet)
		err, _ = callResult[1].Interface().(error)
	}
	return ret, err //errors.WithStack(err)
}
func pp(items ...interface{}) (written int, err error) {
	Println()
	return Print(items...)
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

type TestCase interface {
	Run(*testing.T)
}
type CreateDevice struct {
	device   gjson.Result
	deviceId string
	loop     int
}

func (bind *CreateDevice) Run(t *testing.T) {
	Convey("获取一台新设备", t, func() {
	Retry:
		file, err := os.OpenFile("./data/device_id", os.O_RDWR|os.O_CREATE, os.ModePerm)
		defer file.Close()
		So(err, ShouldBeNil)
		content, err := io.ReadAll(file)
		So(err, ShouldBeNil)
		id := goutil.Int(string(content)) + 1
		file.Truncate(0)
		file.Seek(0, 0)
		_, err = file.WriteString(goutil.String(id))
		So(err, ShouldBeNil)
		deviceId := fmt.Sprintf("got-%v-%v", getMacAddr()[12:17], goutil.String(id))
		ret, err := adminApi.CreateDevice(deviceId, deviceId)
		So(err, ShouldBeNil)
		if gjson.Get(ret.Body, "msg").String() == "该授权码已使用" && bind.loop < 3 {
			pp("重试")
			bind.loop++
			goto Retry
		}
		SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "status").Int(), ShouldEqual, 1)
		ret, err = adminApi.UpdateDeviceStatus(deviceId)
		So(err, ShouldBeNil)
		SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "status").Int(), ShouldEqual, 1)
		bind.device = gjson.Parse(fmt.Sprintf(`{"device_id": "%v"}`, deviceId))
		bind.deviceId = bind.device.Get("device_id").String()
		pp(bind.deviceId)

	})
}
func TestCreateDevice(t *testing.T) {
	(&CreateDevice{}).Run(t)
}

type CreateLocation struct {
	location     gjson.Result
	locationId   string
	locationName string
	applySn      string
}

func (bind *CreateLocation) Run(t *testing.T) {
	Convey("添加点位", t, func() {
		name := "贵阳市花溪区" + time.Now().Format("2006-01-02 15:04:05")
		pp("添加")
		assertOk(adminApi.CreateLocation(name))
		pp("查询")
		ret, err := adminApi.GetLocList(fmt.Sprintf(`{"name": "%v"}`, name))
		assertHasOne(ret, err)
		bind.location = gjson.Get(ret.Body, "rows.0")
		bind.locationId = bind.location.Get("id").String()
		bind.locationName = bind.location.Get("name").String()
		bind.applySn = bind.location.Get("apply_sn").String()
		pp("审核")
		assertOk(adminApi.ApproveLocation(bind.location.Get("id").String()))
		pp(bind.locationName)
	})
}
func TestCreateLocation(t *testing.T) {
	(&CreateLocation{}).Run(t)
}

type SetInstallTime struct {
	CreateLocation
}

func (bind *SetInstallTime) Run(t *testing.T) {
	bind.CreateLocation.Run(t)
	Convey("设置预计安装时间", t, func() {
		assertOk(adminApi.SetInstallTime(bind.location.Get("id").String()))
	})
}
func TestSetInstallTime(t *testing.T) {
	(&SetInstallTime{}).Run(t)
}

type CreateExwarehouse struct {
	SetInstallTime
	CreateDevice
	exwarehouse   gjson.Result
	exwareHouseId string
	arrive        gjson.Result
}

func (bind *CreateExwarehouse) Run(t *testing.T) {
	bind.SetInstallTime.Run(t)
	bind.CreateDevice.Run(t)
	Convey("出库申请", t, func() {
		assertOk(adminApi.CreateExwarehouse(bind.location.Get("apply_sn").String()))
		pp("查询")
		pp(bind.locationName)
		ret, err := adminApi.GetExwarehouseList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName))
		assertHasOne(ret, err)
		bind.exwarehouse = gjson.Get(ret.Body, "rows.0")
		bind.exwareHouseId = bind.exwarehouse.Get("warehouse_id").String()
		pp("审批")
		assertOk(adminApi.ApproveExwarehouse(bind.exwareHouseId))
		pp("通知")
		assertOk(adminApi.ExwarehouseNotice(bind.exwareHouseId))
		pp("设备登记")
		assertOk(adminApi.CreateExwarehouseDevice(bind.exwareHouseId, bind.deviceId))
		pp("到货登记")
		assertOk(adminApi.CreateExwarehouseArrive(bind.applySn, bind.deviceId))
		pp("查询到货登记")
		ret, err = adminApi.GetExwarehouseArriveList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName))
		assertHasOne(ret, err)
		bind.arrive = gjson.Get(ret.Body, "rows.0")
		pp("到货登记审批")
		assertOk(adminApi.ApproveExwarehouseArrive(bind.arrive.Get("arrive_id").String()))
	})
}
func TestCreateExwarehouse(t *testing.T) {
	(&CreateExwarehouse{}).Run(t)
}

func TestClean(t *testing.T) {
	Convey("删除安装登记", t, func() {
		ret, err := adminApi.GetInstallationList(`{"apply_name": "贵阳市花溪区202"}`)
		So(err, ShouldBeNil)
		SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), ShouldBeGreaterThan, 0)
		gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
			_, err := adminApi.DeleteInstallation(row.Get("id").String())
			So(err, ShouldBeNil)
			//SoMsg(gjson.Get(ret.Body, "msg").String()+row.Get("name").String(), gjson.Get(ret.Body, "code").Int(), ShouldEqual, 1)
			return true
		})
	})
	Convey("删除点位", t, func() {
		ret, err := adminApi.GetLocList(`{"name": "贵阳市花溪区202"}`)
		So(err, ShouldBeNil)
		SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), ShouldBeGreaterThan, 0)
		gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
			_, err := adminApi.DeleteLocation(row.Get("id").String())
			So(err, ShouldBeNil)
			//SoMsg(gjson.Get(ret.Body, "msg").String()+row.Get("name").String(), gjson.Get(ret.Body, "code").Int(), ShouldEqual, 1)
			return true
		})
	})
}

func TestFoo(t *testing.T) {
	fmt.Println(getMacAddr())
	fmt.Println(getMacAddr()[12:17])
	fmt.Println(runtime.GOOS)
}

type CreateInstallation struct {
	CreateExwarehouse
}

func (bind *CreateInstallation) Run(t *testing.T) {
	bind.CreateExwarehouse.Run(t)
	Convey("安装登记", t, func() {
		assertOk(adminApi.CreateInstallation(bind.applySn, bind.deviceId))
	})
}
func TestCreateInstallation(t *testing.T) {
	(&CreateInstallation{}).Run(t)
}

type CreateWeaningApplication struct {
	CreateInstallation
	weaningApplication gjson.Result
}

func (bind *CreateWeaningApplication) Run(t *testing.T) {
	bind.CreateInstallation.Run(t)
	Convey("撤机申请", t, func() {
		assertOk(adminApi.CreateWeaningApplication(bind.applySn, bind.deviceId))
		pp("查询")
		ret, err := adminApi.GetWeaningApplicationList(fmt.Sprintf(`{"device_id": "%v"}`, bind.deviceId))
		assertHasOne(ret, err)
		bind.weaningApplication = gjson.Get(ret.Body, "rows.0")
		pp("审批")
		assertOk(adminApi.ApproveWeaningApplication(bind.weaningApplication.Get("id").String()))
	})
}
func TestCreateWeaningApplication(t *testing.T) {
	(&CreateWeaningApplication{}).Run(t)
}

type CreateWeaningReg struct {
	CreateWeaningApplication
}

func (bind *CreateWeaningReg) Run(t *testing.T) {
	bind.CreateWeaningApplication.Run(t)
	Convey("撤机登记", t, func() {
		assertOk(adminApi.CreateWeaningReg(bind.weaningApplication.Get("original_apply_sn").String(), bind.weaningApplication.Get("device_id").String()))
	})
}
func TestCreateWeaningReg(t *testing.T) {
	(&CreateWeaningReg{}).Run(t)
}
