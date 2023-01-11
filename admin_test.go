package main

import (
	"fmt"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/stdutil"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tidwall/gjson"
	"io"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	fmt.Println(getMacAddr())
	fmt.Println(getMacAddr()[12:17])
	fmt.Println(runtime.GOOS)
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

var adminApi = NewAdminApi()

var adminDataError *AdminDataError

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
func assertHasOne(ret *ApiRet, err error) {
	ret, err = RetryIfNotSignedIn(ret, err)
	if err != nil && !errors.As(err, &adminDataError) {
		log.Errorf("%+v", err)
	}
	convey.SoMsg(stdutil.GetCallerInfo(2), err, convey.ShouldBeNil)
	convey.SoMsg(stdutil.GetCallerInfo(2)+" 没有找到任何记录", gjson.Get(ret.Body, "total").Int(), convey.ShouldBeGreaterThan, 0)
}
func RetryIfNotSignedIn(ret *ApiRet, err error) (*ApiRet, error) {
	if err != nil && ret != nil && strings.Contains(ret.Resp.Request.URL.String(), "/login") {
		pp("登录")
		assertOk(adminApi.SignIn(os.Getenv("SIGN_IN_USERNAME"), os.Getenv("SIGN_IN_PASSWORD")))
		pp("再次请求")
		callResult := reflect.ValueOf(adminApi).MethodByName(ret.Method).Call(ret.Args)
		ret, _ = callResult[0].Interface().(*ApiRet)
		err, _ = callResult[1].Interface().(error)
	}
	return ret, err
}

type TestCase interface {
	Run(*testing.T)
}
type CreateDevice struct {
	device   gjson.Result
	deviceId string
}

func (bind *CreateDevice) Run(t *testing.T) {
	pp("创建设备")
	tryTimes := 0
Retry:
	tryTimes++
	file, err := os.OpenFile("./data/device_id", os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer file.Close()
	convey.So(err, convey.ShouldBeNil)
	content, err := io.ReadAll(file)
	convey.So(err, convey.ShouldBeNil)
	id := goutil.Int(string(content)) + 1
	file.Truncate(0)
	file.Seek(0, 0)
	_, err = file.WriteString(goutil.String(id))
	convey.So(err, convey.ShouldBeNil)
	deviceId := fmt.Sprintf("got-%v-%v", getMacAddr()[12:17], goutil.String(id))
	_, err = RetryIfNotSignedIn(adminApi.CreateDevice(deviceId, deviceId))
	if err != nil {
		if !strings.Contains(err.Error(), "该授权码已使用") || tryTimes > 2 {
			convey.So(err, convey.ShouldBeNil)
		}
		pp("重试获取一台新设备")
		goto Retry
	}
	pp("更新设备状态")
	assertOk(adminApi.UpdateDeviceStatus(deviceId))
	bind.device = gjson.Parse(fmt.Sprintf(`{"device_id": "%v"}`, deviceId))
	bind.deviceId = bind.device.Get("device_id").String()
	pp("创建设备成功" + bind.deviceId)

}
func TestCreateDevice(t *testing.T) {
	convey.Convey("", t, func() { (&CreateDevice{}).Run(t) })
}

type CreateLocation struct {
	location     gjson.Result
	locationId   string
	locationName string
	applySn      string
}

func (bind *CreateLocation) Run(t *testing.T) {
	pp("创建点位")
	name := "贵阳市花溪区" + time.Now().Format("2006-01-02 15:04:05")
	assertOk(adminApi.CreateLocation(name))
	pp("查询点位")
	ret, err := adminApi.GetLocList(fmt.Sprintf(`{"name": "%v"}`, name))
	assertHasOne(ret, err)
	bind.location = gjson.Get(ret.Body, "rows.0")
	bind.locationId = bind.location.Get("id").String()
	bind.locationName = bind.location.Get("name").String()
	bind.applySn = bind.location.Get("apply_sn").String()
	pp("审核点位")
	assertOk(adminApi.ApproveLocation(bind.location.Get("id").String()))
	pp("创建点位成功" + bind.locationName)

}
func TestCreateLocation(t *testing.T) {
	convey.Convey("", t, func() { (&CreateLocation{}).Run(t) })
}

type SetInstallTime struct {
	CreateLocation
}

func (bind *SetInstallTime) Run(t *testing.T) {
	bind.CreateLocation.Run(t)
	pp("设置预计安装时间")
	assertOk(adminApi.SetInstallTime(bind.location.Get("id").String()))

}
func TestSetInstallTime(t *testing.T) {
	convey.Convey("", t, func() { (&SetInstallTime{}).Run(t) })
}

type CreateExwarehouse struct {
	SetInstallTime
	CreateDevice
	exwarehouse   gjson.Result
	exwareHouseId string
}

func (bind *CreateExwarehouse) Run(t *testing.T) {
	var goFuncs = []func(wg *sync.WaitGroup){
		func(wg *sync.WaitGroup) {
			convey.Convey("", t, func() {
				bind.CreateDevice.Run(t)
			})
			wg.Done()
		},
		func(wg *sync.WaitGroup) {
			convey.Convey("", t, func() {
				bind.SetInstallTime.Run(t)
				pp("出库申请")
				assertOk(adminApi.CreateExwarehouse(bind.location.Get("apply_sn").String()))
				pp("查询出库申请")
				ret, err := adminApi.GetExwarehouseList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName))
				assertHasOne(ret, err)
				bind.exwarehouse = gjson.Get(ret.Body, "rows.0")
				bind.exwareHouseId = bind.exwarehouse.Get("warehouse_id").String()
				pp("审批出库申请")
				assertOk(adminApi.ApproveExwarehouse(bind.exwareHouseId))
				pp("出库通知")
				assertOk(adminApi.ExwarehouseNotice(bind.exwareHouseId))
				pp("获取设备号")
				for bind.deviceId == "" {
					pp("...")
					time.Sleep(time.Second)
				}
				pp("得到设备号" + bind.deviceId)
				pp("设备登记")
				assertOk(adminApi.CreateExwarehouseDevice(bind.exwareHouseId, bind.deviceId))
			})
			wg.Done()
		},
	}
	var wg sync.WaitGroup
	wg.Add(len(goFuncs))
	for _, f := range goFuncs {
		go f(&wg)
	}
	wg.Wait()
}

func TestCreateExwarehouse(t *testing.T) {
	convey.Convey("", t, func() { (&CreateExwarehouse{}).Run(t) })
}

type CreateExwarehouseArrive struct {
	CreateExwarehouse
	arrive gjson.Result
}

func (bind *CreateExwarehouseArrive) Run(t *testing.T) {
	bind.CreateExwarehouse.Run(t)
	pp("到货登记")
	assertOk(adminApi.CreateExwarehouseArrive(bind.applySn, bind.deviceId))
	pp("查询到货登记")
	ret, err := adminApi.GetExwarehouseArriveList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName))
	assertHasOne(ret, err)
	bind.arrive = gjson.Get(ret.Body, "rows.0")
	pp("审批到货登记")
	assertOk(adminApi.ApproveExwarehouseArrive(bind.arrive.Get("arrive_id").String()))

}
func TestCreateExwarehouseArrive(t *testing.T) {
	convey.Convey("", t, func() { (&CreateExwarehouseArrive{}).Run(t) })
}

type CreateInstallation struct {
	CreateExwarehouseArrive
}

func (bind *CreateInstallation) Run(t *testing.T) {
	bind.CreateExwarehouseArrive.Run(t)
	pp("安装登记")
	assertOk(adminApi.CreateInstallation(bind.applySn, bind.deviceId))

}
func TestCreateInstallation(t *testing.T) {
	convey.Convey("", t, func() { (&CreateInstallation{}).Run(t) })
}

func TestClean(t *testing.T) {
	t.Run("删除安装登记", func(t *testing.T) {
		t.Parallel()
		convey.Convey("", t, func() {
			ret, err := RetryIfNotSignedIn(adminApi.GetInstallationList(`{"apply_name": "贵阳市花溪区202"}`))
			convey.So(err, convey.ShouldBeNil)
			convey.SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), convey.ShouldBeGreaterThan, 0)
			gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
				_, err := adminApi.DeleteInstallation(row.Get("id").String())
				if err != nil {
					if !strings.Contains(err.Error(), "超过24小时不允许删除") && !strings.Contains(err.Error(), "已验收完成") {
						convey.So(err, convey.ShouldBeNil)
					}
				}
				pp(row.Get("name"))
				return true
			})
		})
	})
	t.Run("删除点位", func(t *testing.T) {
		t.Parallel()
		convey.Convey("", t, func() {
			ret, err := RetryIfNotSignedIn(adminApi.GetLocList(`{"name": "贵阳市花溪区202"}`))
			convey.So(err, convey.ShouldBeNil)
			convey.SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), convey.ShouldBeGreaterThan, 0)
			gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
				_, err := adminApi.DeleteLocation(row.Get("id").String())
				if err != nil {
					if !strings.Contains(err.Error(), "超过24小时不允许删除") {
						convey.So(err, convey.ShouldBeNil)
					}
				}
				pp(row.Get("name"))
				return true
			})
		})
	})

}

type CreateWeaningApplication struct {
	CreateInstallation
	weaningApplication gjson.Result
}

func (bind *CreateWeaningApplication) Run(t *testing.T) {
	bind.CreateInstallation.Run(t)
	pp("撤机申请")
	assertOk(adminApi.CreateWeaningApplication(bind.applySn, bind.deviceId))
	pp("查询")
	ret, err := adminApi.GetWeaningApplicationList(fmt.Sprintf(`{"device_id": "%v"}`, bind.deviceId))
	assertHasOne(ret, err)
	bind.weaningApplication = gjson.Get(ret.Body, "rows.0")
	pp("审批")
	assertOk(adminApi.ApproveWeaningApplication(bind.weaningApplication.Get("id").String()))

}
func TestCreateWeaningApplication(t *testing.T) {
	convey.Convey("", t, func() { (&CreateWeaningApplication{}).Run(t) })
}

type CreateWeaningReg struct {
	CreateWeaningApplication
}

func (bind *CreateWeaningReg) Run(t *testing.T) {
	bind.CreateWeaningApplication.Run(t)
	pp("撤机登记")
	assertOk(adminApi.CreateWeaningReg(bind.weaningApplication.Get("original_apply_sn").String(), bind.weaningApplication.Get("device_id").String()))

}
func TestCreateWeaningReg(t *testing.T) {
	convey.Convey("", t, func() { (&CreateWeaningReg{}).Run(t) })
}
