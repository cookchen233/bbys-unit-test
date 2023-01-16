package main

import (
	"encoding/json"
	"fmt"
	logrusStack "github.com/Gurpartap/logrus-stack"
	"github.com/gookit/goutil"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type AdminDataError struct {
	message string
}

func (bind *AdminDataError) Error() string {
	return bind.message
}

func init() {
	log.SetLevel(log.DebugLevel)
	callerLevels := []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
	}
	stackLevels := []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel}
	log.AddHook(logrusStack.NewHook(callerLevels, stackLevels))
	log.AddHook(RotateLogHook("log", "stdout.log", 7*24*time.Hour, 24*time.Hour))
	log.SetOutput(io.Discard)
}

// AdminApi 后台接口
type AdminApi[T any] struct {
	apiClient *ApiClient
	baseUrl   string
}

func NewAdminApi() *AdminApi[any] {
	return &AdminApi[any]{
		apiClient: NewApiClient(),
		baseUrl:   os.Getenv("BASE_URL"),
	}
}

func (bind *AdminApi[T]) toRet(resp *http.Response, args ...interface{}) (*ApiRet, error) {
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	ret := ApiRet{
		Method: getCallFuncName(2),
		Resp:   resp,
		Body:   string(body),
	}
	for _, v := range args {
		ret.Args = append(ret.Args, reflect.ValueOf(v))
	}

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	code, ex := data["code"]
	_, ex2 := data["total"]
	status, ex3 := data["status"]
	var errCode int
	var errMsg string
	if err != nil {
		errCode = 1
		errMsg = "解析json数据时发生错误 " + err.Error()
	} else if !ex && !ex2 && !ex3 {
		errCode = 2
		errMsg = "无效返回数据"
	} else if ex {
		if goutil.Int(code) != 1 {
			errCode = 3
			errMsg = data["msg"].(string)
		}
	} else if ex3 {
		if goutil.Int(status) != 1 {
			errCode = 4
			errMsg = data["msg"].(string)
		}

	}
	if errCode > 0 {
		err = errors.WithStack(&AdminDataError{fmt.Sprintf("%v(%v)", errMsg, errCode)})
		var logContent = []string{
			err.Error(),
			ret.Method,
			goutil.String(ret.Args),
		}
		var logContent2 []string
		// 重定向历史请求
		for resp != nil {
			var reqBody []byte
			if resp.Request.Method == "POST" {
				var e error
				reqBody, e = func() ([]byte, error) {
					b, e2 := io.ReadAll(resp.Request.Body)
					defer resp.Request.Body.Close()
					return b, e2
				}()
				if e != nil {
					return nil, errors.WithStack(e)
				}
			}
			logContent2 = append(
				[]string{
					resp.Request.Method + " " + resp.Status + " " + resp.Request.URL.String(),
					"reqBody:" + string(reqBody),
					"respBody:" + string(body),
				}, logContent2...)
			resp = resp.Request.Response
		}
		log.Error(strings.Join(append(logContent, logContent2...), "\n"))

	}
	return &ret, err
}

/*
SignIn
登录
@param username 用户名
@param password 密码
*/
func (bind *AdminApi[T]) SignIn(username string, password string) (*ApiRet, error) {
	params := map[string]interface{}{
		"username": username,
		"password": password,
	}
	resp, err := bind.apiClient.Post(os.Getenv("BASE_URL")+"index/login.html", params)
	if err != nil {
		return nil, err
	}
	if resp.Request.Response == nil {
		return bind.toRet(resp, username, password)
	} else {
		return &ApiRet{
			Method: getCallFuncName(1),
			Resp:   resp,
		}, nil
	}
}

/*
CreateDevice
创建一台新设备
@param deviceId 设备号
@param accessCode 授权码
*/
func (bind *AdminApi[T]) CreateDevice(deviceId string, accessCode string) (*ApiRet, error) {
	params := map[string]interface{}{
		"device_code": accessCode,
		"device_id":   deviceId,
		"partner_id":  os.Getenv("PARTNER_ID"),
		"model":       "A18S",
		"id":          "8",
		"_ajax":       "1",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"terminal_device/initDevice%v", ""), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, deviceId, accessCode)
}

/*
UpdateDeviceStatus
更新设备状态
@param deviceId 设备号
*/
func (bind *AdminApi[T]) UpdateDeviceStatus(deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    deviceId,
		"params": "status=2",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"terminal_device/multi/ids/%v", deviceId), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, deviceId)
}

/*
CreateLocation
待投放点位登记
@param name 点位名称
*/
func (bind *AdminApi[T]) CreateLocation(name string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[partner_id]": os.Getenv("PARTNER_ID"),
		"row[name]":       name,
		"row[scene_picture][][file]": []string{
			"https://i.328888.xyz/2023/01/08/kKzKA.th.png",
			"https://i.328888.xyz/2023/01/08/kKFDo.th.png",
			"https://i.328888.xyz/2023/01/08/kKG3N.th.png",
			"https://i.328888.xyz/2023/01/08/kKs1z.th.jpeg",
		},
		"row[province]":             "10543",
		"row[city]":                 "10544",
		"row[district]":             "10747",
		"row[street]":               "10748",
		"row[address]":              "Shenzhen",
		"row[map_zoom]":             "18",
		"row[map_center]":           "",
		"row[coordinates]":          "26.410513364053,106.67600878863",
		"row[area_type]":            "社区",
		"row[village_total_number]": "2222",
		"row[village_name]":         "耶鲁烧烤",
		"row[occupancy_rate]":       "33.04",
		"row[property_company]":     "耶鲁物业",
		"row[property_tel]":         "13888888888",
		"row[remarks]":              "测试添加",
		"row[developer]":            "Chen",
		"row[responsible_person]":   "Chen",
		"row[rental_price]":         "21.97",
		"row[is_electricity]":       "0",
		"row[electricity]":          "800",
		"row[contract_start_time]":  time.Now().Format("2006-01-02"),
		"row[contract_end_time]":    time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		"row[expect_time]":          time.Now().Format("2006-01-02"),
		"row[contract_doc]":         "https://jwc.xaut.edu.cn/__local/4/23/46/9AC912EFF27555E7779B76CFD01_85B2E41D_57F90.pdf",
		"row[contract_no]":          "123456",
		"row[copy_add]":             "0",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_location_reg/add?dialog=1%v", ""), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, name)
}

/*
GetLocationList
获取点位列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetLocationList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"device_location_reg/index?admin_nav=10&sort=id&order=desc&offset=0&limit=10&op=%7B%22name%22%3A%22LIKE%22%7D", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
ApproveLocation
点位审批
@param id 点位id
*/
func (bind *AdminApi[T]) ApproveLocation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "check_status=1",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_location_reg/multi%v", ""), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
DeleteLocation
删除点位
@param id 点位id
*/
func (bind *AdminApi[T]) DeleteLocation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"action": "del",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_location_reg/del/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
SetInstallTime
设置预计安装时间
@param id 点位id
*/
func (bind *AdminApi[T]) SetInstallTime(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[id]":                     id,
		"row[estimated_install_time]": time.Now().Format("2006-01-02"),
		"row[install_user]":           "Chen",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_location_reg_estimated_install/updateinstall/ids/%v?dialog=1", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
CreateExwarehouse
出库申请
@param applySn 点位号
*/
func (bind *AdminApi[T]) CreateExwarehouse(applySn string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[consignee]":        "Chen",
		"row[consignee_time]":   time.Now().Format("2006-01-02"),
		"row[exwarehouse_type]": 2,
		"row[remark]":           "测试添加",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_location_reg_estimated_install/exwarehouse/apply_sn/%v?dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, applySn)
}

/*
GetExwarehouseList
查询出库列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetExwarehouseList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"device_location_reg_estimated_install/index/", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
ApproveExwarehouse
出库审批
@param id 出库id
*/
func (bind *AdminApi[T]) ApproveExwarehouse(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "audit_status=1",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"exwarehouse/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
ExwarehouseNotice
出库通知
@param id 出库id
*/
func (bind *AdminApi[T]) ExwarehouseNotice(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[message]":                  "Chen",
		"row[consignee_time]":           time.Now().Format("2006-01-02"),
		"row[exwarehouse_logistics_id]": 2,
		"row[sum]":                      1,
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"exwarehouse_notice/add?warehouse_id=%v&dialog=1", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
CreateExwarehouseDevice
设备登记
@param exwareHouseId 出库id
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateExwarehouseDevice(exwareHouseId string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[device_id]":        deviceId,
		"row[exwarehouse_time]": time.Now().Format("2006-01-02"),
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"Exwarehouse_device/add.html?warehouse_id=%v&dialog=1", exwareHouseId), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, exwareHouseId, deviceId)
}

/*
CreateExwarehouseArrive
到货登记
@param applysn 点位号
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateExwarehouseArrive(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[logistics_status]": 1,
		"row[cost]":             "1.00",
		"row[arrive_remark]":    "测试添加",
		"row[images]":           "https://i.328888.xyz/2023/01/08/kKzKA.th.png,https://i.328888.xyz/2023/01/08/kKFDo.th.png,https://i.328888.xyz/2023/01/08/kKG3N.th.png,https://i.328888.xyz/2023/01/08/kKs1z.th.jpeg",
		"row[device_id]":        deviceId,
		"row[arrive_date]":      time.Now().Format("2006-01-02"),
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"exwarehouse_arrive/edit?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, applySn, deviceId)
}

/*
GetExwarehouseArriveList
查询到货登记列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetExwarehouseArriveList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"exwarehouse_arrive/index/", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
ApproveExwarehouseArrive
到货登记审批
@param id 到货登记id
*/
func (bind *AdminApi[T]) ApproveExwarehouseArrive(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "approve_status=1",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"exwarehouse_arrive/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
CreateArrivedStatement
添加物流费用结算单
@param arriveId 到货登记id
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateArrivedStatement(arriveId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[exwarehouse_logistics_id]": 2,
		"row[arrived_earliest]":         time.Now().Add(-24 * 7 * time.Hour).Format("2006-01-02 15:04:05"),
		"row[arrived_latest]":           time.Now().Format("2006-01-02 15:04:05"),
		"row[arrived_num]":              1,
		"row[total_cost]":               1,
		"row[average_cost]":             1,
		"row[applicant]":                "Chen",
		"row[remarks]":                  "测试添加",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"logistics_arrived_statement/add/arrive_id/%v?dialog=1", arriveId), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, arriveId)
}

/*
CreateInstallation
点位安装登记
@param applySn 点位号
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateInstallation(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[apply_sn]":                   applySn,
		"row[device_id]":                  deviceId,
		"row[map_zoom]":                   18,
		"row[map_center]":                 "",
		"row[coordinates]":                "22.54845664,114.06455184",
		"row[scene_picture][0][file]":     "https://i.328888.xyz/2023/01/08/kKzKA.th.png",
		"row[scene_picture][1][file]":     "https://i.328888.xyz/2023/01/08/kKFDo.th.png",
		"row[scene_picture][2][file]":     "https://i.328888.xyz/2023/01/08/kKG3N.th.png",
		"row[scene_picture][3][file]":     "https://i.328888.xyz/2023/01/08/kKs1z.th.jpeg",
		"row[status]":                     2,
		"row[complete_time]":              time.Now().Format("2006-01-02 15:04:05"),
		"row[power_number]":               666,
		"row[power_picture]":              "https://i.328888.xyz/2023/01/08/kL4RH.th.jpeg",
		"row[installor]":                  "Chen",
		"row[install_pic_list]":           "https://i.328888.xyz/2023/01/08/kKzKA.th.png",
		"row[cost_details][0][name]":      "接电",
		"row[cost_details][0][value]":     1,
		"row[cost_details][1][name]":      "地坪",
		"row[cost_details][1][value]":     2,
		"row[cost_details][2][name]":      "安装",
		"row[cost_details][2][value]":     3,
		"row[cost_details][3][name]":      "超加",
		"row[cost_details][3][value]":     4,
		"row[cost]":                       10.00,
		"row[move_device_application_id]": 0,
		"row[remarks]":                    "测试添加",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_install_reg/add/apply_sn/%v?dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, applySn, deviceId)
}

/*
GetInstallationList
查询安装登记列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetInstallationList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"device_install_reg/index?admin_nav=11&sort=id&order=desc&offset=0&limit=10&op=%7B%22apply_name%22%3A%22EXTEND%22%7D", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
DeleteInstallation
删除安装登记
@param id 安装登记id
*/
func (bind *AdminApi[T]) DeleteInstallation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"action": "del",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"device_install_reg/del/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
CreateWeaningApplication
撤机申请
@param applySn 点位号
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateWeaningApplication(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[original_apply_sn]": applySn,
		"row[device_id]":         deviceId,
		"row[weaning_type]":      2,
		"row[apply_sn]":          "0769000066",
		"row[weaning_time]":      time.Now().Format("2006-01-02 15:04:05"),
		"row[last_time]":         time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
		"row[property_tel]":      "13888888888",
		"row[director]":          370,
		"row[remarks]":           "测试",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"terminal_weaning_device/add?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, applySn, deviceId)
}

/*
GetWeaningApplicationList
查询撤机申请列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetWeaningApplicationList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"terminal_weaning_device/index/", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
ApproveWeaningApplication
审批撤机申请
@param id 到货登记id
*/
func (bind *AdminApi[T]) ApproveWeaningApplication(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "approve_status=1",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"terminal_weaning_device/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, id)
}

/*
CreateWeaningReg
撤机登记
@param applySn 点位号
@param deviceId 设备号
*/
func (bind *AdminApi[T]) CreateWeaningReg(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[apply_sn]":     applySn,
		"row[device_id]":    deviceId,
		"row[weaning_time]": time.Now().Format("2006-01-02 15:04:05"),
		"row[pic_list]":     "https://i.328888.xyz/2023/01/08/kKzKA.th.png,https://i.328888.xyz/2023/01/08/kKFDo.th.png,https://i.328888.xyz/2023/01/08/kKG3N.th.png,https://i.328888.xyz/2023/01/08/kKs1z.th.jpeg",
		"row[operator]":     "Chen",
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"weaning_reg/add?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, applySn, deviceId)
}

/*
CreatePrintTicketTemplate
创建打印券模板
*/
func (bind *AdminApi[T]) CreatePrintTicketTemplate() (*ApiRet, error) {
	params := map[string]interface{}{
		"row[ticket_name]":       os.Getenv("TK_NAME"),
		"row[partner_id]":        os.Getenv("TK_PARTNER_ID"),
		"row[balance_type]":      1,
		"row[ticket_type]":       3,
		"row[support_devices]":   "",
		"row[unsupport_devices]": "",
		"row[type]":              2,
		"row[order_type]":        strings.Split(os.Getenv("TK_ORDER_TYPE"), ","),
		"row[max_times]":         os.Getenv("TK_MAX_TIMES"),
		"row[get_way]":           "one-off",
		"row[status]":            1,
		"row[expired_type]":      1,
		"row[expired_time]":      "",
		"row[start_time]":        os.Getenv("TK_START"),
		"row[end_time]":          os.Getenv("TK_END"),
		"row[remark]":            os.Getenv("TK_REMARK"),
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"/user_free_ticket_conf/add?dialog=1%v", ""), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp)
}

/*
GetPrintTicketTemplateList
查询打印券模板列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetPrintTicketTemplateList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"user_free_ticket_conf/index?ref=addtabs&addtabs=1&sort=id&order=desc&offset=0&limit=10&filter=%7B%7D&op=%7B%7D&_=1673672689542", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}

/*
CreatePrintTicket
创建打印券
*/
func (bind *AdminApi[T]) CreatePrintTicket(ticketTplId string, num int) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[ticket_number]": ticketTplId,
		"row[prefix]":        "CH",
		"row[count]":         num,
	}
	resp, err := bind.apiClient.Post(fmt.Sprintf(bind.baseUrl+"user_free_ticket_unique_code/add?templ_id=%v&dialog=1", ticketTplId), params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp)
}

/*
GetPrintTicketList
查询打印券列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetPrintTicketList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.apiClient.Get(bind.baseUrl+"user_free_ticket_unique_code/index?templ_id=%v&dialog=1&sort=id&order=desc", params)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp, filter)
}
