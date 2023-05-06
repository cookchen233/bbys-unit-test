package main

import (
	"bbys-unit-test/routine"
	"encoding/json"
	"fmt"
	"github.com/gohouse/converter"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/stdutil"
	"github.com/joho/godotenv"
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
	if err != nil && ret != nil && strings.Contains(ret.Resp.Request.URL.String(), "/login") {
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
	if bind.deviceId == "" {
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
		bind.deviceId = fmt.Sprintf("got-%v-%v", getMacAddr()[12:17], goutil.String(id))
	}
	_, err := RetryIfNotSignedIn(adminApi.CreateDevice(bind.deviceId, bind.deviceId))
	if err != nil {
		if !strings.Contains(err.Error(), "该授权码已使用") || tryTimes > 2 {
			convey.So(err, convey.ShouldBeNil)
		}
		pp("重试获取一台新设备")
		goto Retry
	}
	pp("更新设备状态")
	assertOk(adminApi.UpdateDeviceStatus(bind.deviceId))
	bind.device = gjson.Parse(fmt.Sprintf(`{"device_id": "%v"}`, bind.deviceId))
	pp("创建设备成功 " + bind.device.Get("device_id").String())

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
	if bind.locationName == "" {
		bind.locationName = "贵阳市花溪区" + time.Now().Format("2006-01-02 15:04:05")
	}
	assertOk(adminApi.CreateLocation(bind.locationName))
	pp("查询点位")
	ret := assertHasOne(adminApi.GetLocationList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName)))
	bind.location = gjson.Get(ret.Body, "rows.0")
	bind.locationId = bind.location.Get("id").String()
	bind.applySn = bind.location.Get("apply_sn").String()
	pp("审核点位")
	assertOk(adminApi.ApproveLocation(bind.location.Get("id").String()))
	pp("创建点位成功 " + bind.locationName)

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
				ret := assertHasOne(adminApi.GetExwarehouseList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName)))
				bind.exwarehouse = gjson.Get(ret.Body, "rows.0")
				bind.exwareHouseId = bind.exwarehouse.Get("warehouse_id").String()
				pp("审批出库申请")
				assertOk(adminApi.ApproveExwarehouse(bind.exwareHouseId))
				pp("出库通知")
				assertOk(adminApi.ExwarehouseNotice(bind.exwareHouseId))
				pp("获取设备号")
				//bind.deviceId = "600100"
				for bind.deviceId == "" {
					pp("...")
					time.Sleep(time.Second)
				}
				pp("得到设备号 " + bind.deviceId)
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
	ret := assertHasOne(adminApi.GetExwarehouseArriveList(fmt.Sprintf(`{"name": "%v"}`, bind.locationName)))
	bind.arrive = gjson.Get(ret.Body, "rows.0")
	pp("审批到货登记")
	assertOk(adminApi.ApproveExwarehouseArrive(bind.arrive.Get("arrive_id").String()))

}
func TestCreateExwarehouseArrive(t *testing.T) {
	convey.Convey("", t, func() { (&CreateExwarehouseArrive{}).Run(t) })
}

type CreateInstallation struct {
	CreateExwarehouseArrive
	completeTime string
}

func (bind *CreateInstallation) Run(t *testing.T) {
	bind.CreateExwarehouseArrive.Run(t)
	pp("安装登记")
	if bind.completeTime == "" {
		bind.completeTime = time.Now().Format("2006-01-02 15:04:05")
	}
	assertOk(adminApi.CreateInstallation(bind.applySn, bind.deviceId, bind.completeTime))

}
func TestCreateInstallation(t *testing.T) {
	convey.Convey("", t, func() { (&CreateInstallation{}).Run(t) })
}
func TestCreateMultiInstallation(t *testing.T) {
	ins := CreateInstallation{}
	tests := [][]string{
		{"H4", "H4", "2023-01-31"},
		{"H5", "H5", "2023-01-15"},
		{"H6", "H6", "2023-01-20"},
		{"H7", "H7", "2023-01-08"},
		{"H8", "H8", "2023-01-08"},
		{"H9", "H9", "2023-01-16"},
		{"H10", "H10", "2023-03-02"},
		{"H11", "H11", "2023-03-06"},
		{"H12", "H12", "2023-03-06"},
		{"H13", "H13", "2023-03-06"},
		{"H14", "H14", "2023-03-06"},
	}
	for _, test := range tests {
		ins.deviceId = test[0]
		ins.locationName = test[1]
		ins.completeTime = test[2]
		convey.Convey("", t, func() { (&ins).Run(t) })
	}
}

func TestClean(t *testing.T) {
	t.Run("删除安装登记", func(t *testing.T) {
		t.Parallel()
		convey.Convey("", t, func() {
			ret := assertHasOne(adminApi.GetInstallationList(`{"apply_name": "贵阳市花溪区202"}`))
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
			ret := assertHasOne(adminApi.GetLocationList(`{"name": "贵阳市花溪区202"}`))
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
	ret := assertHasOne(adminApi.GetWeaningApplicationList(fmt.Sprintf(`{"device_id": "%v"}`, bind.deviceId)))
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

func makeBatchTicket(name string, data []routine.VoucherData) {
	isTest, _ := goutil.ToBool(os.Getenv("VCH_IS_TEST"))
	qw, _ := mathutil.Int(os.Getenv("VCH_QRCODE_W"))
	qx, _ := mathutil.Int(os.Getenv("VCH_QRCODE_X"))
	qy, _ := mathutil.Int(os.Getenv("VCH_QRCODE_Y"))
	hasTx, _ := goutil.ToBool(os.Getenv("VCH_HAS_TEXT"))
	tx, _ := mathutil.ToFloat(os.Getenv("VCH_TEXT_X"))
	ty, _ := mathutil.ToFloat(os.Getenv("VCH_TEXT_Y"))
	ts, _ := mathutil.ToFloat(os.Getenv("VCH_TEXT_SIZE"))
	vc := routine.Voucher{
		Name:        name,
		TplFilename: os.Getenv("VCH_TPL_FILE"),
		QrcodeW:     qw,
		QrcodeX:     qx,
		QrcodeY:     qy,
		HasText:     hasTx,
		TextX:       tx,
		TextY:       ty,
		TextSize:    ts,
		SaveDir:     os.Getenv("VCH_SAVE_DIR"),
		//FontFilename: "./data/font/inter/inter-VariableFont_slnt,wght.ttf",
	}
	if isTest {
		pp("测试制作3张图片")
		data = data[0:3]
	}
	dir := vc.MakeBatch(data)

	if os.Getenv("VCH_EMP_ASSIGN") != "" {
		//分配给每个运维人员各自的打印券数量
		empStrs := strings.Split(os.Getenv("VCH_EMP_ASSIGN"), ",")
		var emps [][]string
		dir2 := dir + "/../" + name + "-运维分配"
		os.RemoveAll(dir2)
		for _, empStr := range empStrs {
			emp := strings.Split(empStr, ":")
			if len(emp) < 2 || emp[1] == "" || emp[1] == "0" {
				continue
			}
			emp = append(emp, dir2+"/"+name+"-"+emp[0])
			if _, err := os.Stat(emp[2]); err != nil {
				os.MkdirAll(emp[2], 0755)
			}
			emps = append(emps, emp)

		}
		files, _ := os.ReadDir(dir)
		empI := 0
		empAssigns := 0
		for fileI, file := range files {
			if empAssigns < goutil.Int(emps[empI][1]) && fileI < len(files)-1 {
				_, err := Copy(dir+"/"+file.Name(), emps[empI][2]+"/"+file.Name())
				if err != nil {
					panic(err)
				}
				os.RemoveAll(dir + "/" + file.Name())
				empAssigns++
			} else {
				if err := routine.Archive(emps[empI][2], emps[empI][2]+".zip"); err != nil {
					panic(err)
				}
				empFiles, _ := os.ReadDir(emps[empI][2])
				for _, empFile := range empFiles {
					os.RemoveAll(emps[empI][2] + "/" + empFile.Name())
				}
				os.RemoveAll(emps[empI][2])
				empAssigns = 0
				empI++
				if empI >= len(emps) {
					break
				}
			}
		}
		pp("\n\n创建压缩文件...")
		files, _ = os.ReadDir(dir)
		if len(files) == 0 {
			os.RemoveAll(dir)
		} else {
			if err := routine.Archive(dir, dir+".zip"); err != nil {
				panic(err)
			}
		}
		if err := routine.Archive(dir2, dir2+".zip"); err != nil {
			panic(err)
		}
		pp("创建完成,位置:" + dir)
	} else {
		pp("\n\n创建压缩文件...")
		if err := routine.Archive(dir, dir+".zip"); err != nil {
			panic(err)
		}
		pp("创建完成,位置:" + dir)
	}

}

func TestCreatePrintTicketTemplate(t *testing.T) {
	convey.Convey("创建打印券并测试制作一张打印券图片", t, func() {
		pp("创建打印券模板")
		assertOk(adminApi.CreatePrintTicketTemplate())
		pp("查询打印券模板")
		ret := assertHasOne(adminApi.GetPrintTicketTemplateList(fmt.Sprintf(`{%v}`, "")))
		ticketTemplate := gjson.Get(ret.Body, "rows.0")
		pp(ticketTemplate.Get("ticket_name"), ticketTemplate.Get("remark"))
		pp("测试制作打印券图片")
		pp("创建打印券数据(1条)")
		assertOk(adminApi.CreatePrintTicket(ticketTemplate.Get("ticket_number").String(), 1))
		pp("查询打印券")
		ret = assertHasOne(adminApi.GetPrintTicketList(fmt.Sprintf(`{"templ_id":"%v"}`, ticketTemplate.Get("ticket_number").String())))
		pp("制作打印券图片")
		var data []routine.VoucherData
		if err := json.Unmarshal([]byte(gjson.Get(ret.Body, "rows").String()), &data); err != nil {
			panic(err)
		}
		makeBatchTicket(ticketTemplate.Get("ticket_name").String(), data)
	})
}
func TestCreatePrintTicket(t *testing.T) {
	convey.Convey("确定没问题后, 创建打印券数据", t, func() {
		pp("查询打印券模板")
		ret := assertHasOne(adminApi.GetPrintTicketTemplateList(fmt.Sprintf(`{%v}`, "")))
		ticketTemplate := gjson.Get(ret.Body, "rows.0")
		pp("创建打印券数据")
		assertOk(adminApi.CreatePrintTicket(ticketTemplate.Get("ticket_number").String(), goutil.Int(os.Getenv("VCH_TOTAL"))-1))
	})
}
func TestMakeBatchTicket(t *testing.T) {
	convey.Convey("然后制作打印券图片", t, func() {
		pp("查询打印券模板")
		ret := assertHasOne(adminApi.GetPrintTicketTemplateList(fmt.Sprintf(`{%v}`, "")))
		ticketTemplate := gjson.Get(ret.Body, "rows.1")
		pp("查询打印券")
		ret = assertHasOne(adminApi.GetPrintTicketList(fmt.Sprintf(`{"templ_id":"%v"}`, ticketTemplate.Get("ticket_number").String())))
		pp("制作打印券图片")
		var data []routine.VoucherData
		if err := json.Unmarshal([]byte(gjson.Get(ret.Body, "rows").String()), &data); err != nil {
			panic(err)
		}
		t := time.Now()
		makeBatchTicket(ticketTemplate.Get("ticket_name").String(), data[:1])
		fmt.Println(time.Now().Sub(t).Seconds())
	})
}

func TestMakeBatchTicketByTplId(t *testing.T) {
	convey.Convey("直接根据打印券模板ID制作打印券", t, func() {
		pp("查询打印券模板")
		ret := assertHasOne(adminApi.GetPrintTicketTemplateList(fmt.Sprintf(`{%v}`, "")))
		var ticketTemplate gjson.Result
		tplId := os.Getenv("VCH_TPL_ID")
		gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
			if row.Get("id").String() == tplId {
				pp(row.Get("ticket_name").String())
				ticketTemplate = row
				return false
			}
			return true
		})
		if ticketTemplate.Get("id").String() == "" {
			panic("没有找到该打印券模板")
		}
		pp("查询打印券")
		ret = assertHasOne(adminApi.GetPrintTicketList(fmt.Sprintf(`{"templ_id":"%v"}`, ticketTemplate.Get("ticket_number").String())))
		pp("制作打印券图片")
		var data []routine.VoucherData
		if err := json.Unmarshal([]byte(gjson.Get(ret.Body, "rows").String()), &data); err != nil {
			panic(err)
		}
		t := time.Now()
		makeBatchTicket(ticketTemplate.Get("ticket_name").String(), data)
		pp(fmt.Sprintf("耗时: %.2fs", time.Now().Sub(t).Seconds()))
	})
}

func TestMakeBatchCashTicketByTplId(t *testing.T) {
	convey.Convey("直接根据代金券模板ID制作代金券", t, func() {
		pp("查询代金券模板")
		ret := assertHasOne(adminApi.GetCashPrintTicketTemplateList(fmt.Sprintf(`{%v}`, "")))
		var ticketTemplate gjson.Result
		tplId := os.Getenv("VCH_TPL_ID")
		gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
			if row.Get("id").String() == tplId {
				pp(row.Get("coupon_name").String())
				ticketTemplate = row
				return false
			}
			return true
		})
		if ticketTemplate.Get("id").String() == "" {
			panic("没有找到该代金券模板")
		}
		pp("查询代金券")
		ret = assertHasOne(adminApi.GetCashPrintTicketList(fmt.Sprintf(`{"templ_id":"%v"}`, ticketTemplate.Get("coupon_number").String())))
		pp("制作代金券图片")
		var jsonData []map[string]interface{}
		if err := json.Unmarshal([]byte(gjson.Get(ret.Body, "rows").String()), &jsonData); err != nil {
			panic(err)
		}
		var data []routine.VoucherData
		for _, v := range jsonData {
			data = append(data, routine.VoucherData{
				Text: v["coupon_number"].(string),
				Url:  v["qrcode"].(string),
			})
		}
		t := time.Now()
		makeBatchTicket(ticketTemplate.Get("coupon_name").String(), data)
		pp(fmt.Sprintf("耗时: %.2fs", time.Now().Sub(t).Seconds()))
	})
}

func TestExecTempCrontab(t *testing.T) {
	convey.Convey("执行临时任务", t, func() {
		url := os.Getenv("CRON_URL")
		status := os.Getenv("CRON_STATUS")
		pp(fmt.Sprintf("%v %v", status, url))
		assertOk(adminApi.ExecTempCrontab(url, status))
	})
}
