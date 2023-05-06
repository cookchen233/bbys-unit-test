package main

import (
	"bbys-unit-test/routine"
	"encoding/json"
	"fmt"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/stdutil"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tidwall/gjson"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var adminApi *AdminApi[any]

var adminDataError *AdminDataError

func init() {
	godotenv.Load("pro.env")
	adminApi = NewAdminApi()
}
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
