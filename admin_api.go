package main

import (
	"encoding/json"
	"fmt"
	logrusStack "github.com/Gurpartap/logrus-stack"
	"github.com/gookit/goutil/stdutil"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

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
}

type AdminApi[T any] struct {
	ApiRequest[T]
}

func NewAdminApi() *AdminApi[any] {
	return &AdminApi[any]{}
}

func (bind *AdminApi[T]) toRet(response *http.Response) (*ApiRet, error) {
	body, _ := io.ReadAll(response.Body)
	ret := ApiRet{
		Api:  stdutil.GetCallerInfo(2),
		Resp: response,
		Body: string(body),
	}
	var reqBody []byte
	if response.Request.Method == "POST" {
		reqBody, _ = io.ReadAll(response.Request.Body)
	}
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		//bodyLen, _ := mathutil.Float(len(body))
		//cutLen, _ := mathutil.Int64(math.Min(999, bodyLen))
		log.Error(strings.Join([]string{
			"响应数据解析错误",
			err.Error(),
			ret.Api,
			response.Request.Method + " " + response.Request.URL.String(),
			"reqBody:" + string(reqBody),
			"respBody:" + string(body),
		}, "\n"))
	}
	return &ret, err
}

/*
Login
登录
*/
func (bind *AdminApi[T]) Login() error {
	params := map[string]interface{}{
		"username": "chenwh",
		"password": "!chen8331",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/mobile/admin/login.html?url=/admin/qrcode_process/lists.html?admin_nav=qrcode_process_lists%v", ""), params)
	defer resp.Body.Close()
	return err
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
		//"partner_id":  "2BCFA72F-A91C-0E5C-0AFF-33BCB318CC60",
		"partner_id": "161AF9E7-F57A-9596-81AD-351677DC4203",
		"model":      "A18S",
		"id":         "8",
		"_ajax":      "1",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/terminal_device/initDevice%v", ""), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
UpDeviceStatus
更新设备状态设备
@param deviceId 设备号
*/
func (bind *AdminApi[T]) UpDeviceStatus(deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    deviceId,
		"params": "status=2",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/terminal_device/multi/ids/%v", deviceId), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateLocation
待投放点位登记
@param name 点位名称
*/
func (bind *AdminApi[T]) CreateLocation(name string) (*ApiRet, error) {
	params := map[string]interface{}{
		//"row[partner_id]": "2BCFA72F-A91C-0E5C-0AFF-33BCB318CC60",
		"row[partner_id]": "161AF9E7-F57A-9596-81AD-351677DC4203",
		"row[name]":       "贵阳市花溪区",
		"row[scene_picture][][file]": []string{
			"https://02a-certf05.bbys.cn/FS/Down/?id=56e1da4a40e3483d8c9e341a21df20fd",
			"https://02a-certf05.bbys.cn/FS/Down/?id=87b5696e5fb1452ca620b810ab109d3b",
			"https://02a-certf05.bbys.cn/FS/Down/?id=bf944062fadf49228a9c8a73b20fffa6",
			"https://02a-certf05.bbys.cn/FS/Down/?id=c82a5a7157ee4620bebd327b64fbc408",
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
		"row[property_tel]":         "13883036130",
		"row[remarks]":              "测试添加",
		"row[developer]":            "陈文豪",
		"row[responsible_person]":   "陈文豪",
		"row[rental_price]":         "21.97",
		"row[is_electricity]":       "0",
		"row[electricity]":          "800",
		"row[contract_start_time]":  "2022-12-05",
		"row[contract_end_time]":    "2023-12-28",
		"row[contract_doc]":         "https://oss-fs.bbys.cn/admin/20221111/项目计划书.pdf",
		"row[expect_time]":          "2022-12-05",
		"row[contract_no]":          "123456",
		"row[copy_add]":             "0",
	}
	params["row[name]"] = name
	params["row[contract_start_time]"] = time.Now().Format("2006-01-02")
	params["row[contract_end_time]"] = time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_location_reg/add?dialog=1%v", ""), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
GetLocList
获取点位列表
@param filter 筛选
*/
func (bind *AdminApi[T]) GetLocList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.get("http://loc.bbys.cn/admin/device_location_reg/index?admin_nav=10&sort=id&order=desc&offset=0&limit=10&op=%7B%22name%22%3A%22LIKE%22%7D", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
ApproveLocation
点位审批
@param id 点位id
*/func (bind *AdminApi[T]) ApproveLocation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "check_status=1",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_location_reg/multi%v", ""), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
DeleteLocation
删除点位
@param id 点位id
*/func (bind *AdminApi[T]) DeleteLocation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"action": "del",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_location_reg/del/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
SetInstallTime
设置预计安装时间
@param id 点位id
*/func (bind *AdminApi[T]) SetInstallTime(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[id]":                     id,
		"row[estimated_install_time]": time.Now().Format("2006-01-02"),
		"row[install_user]":           "陈文豪",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_location_reg_estimated_install/updateinstall/ids/%v?dialog=1", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateExwarehouse
出库申请
@param applySn 点位号
*/func (bind *AdminApi[T]) CreateExwarehouse(applySn string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[consignee]":        "陈文豪",
		"row[consignee_time]":   time.Now().Format("2006-01-02"),
		"row[exwarehouse_type]": 2,
		"row[remark]":           "测试添加",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_location_reg_estimated_install/exwarehouse/apply_sn/%v?dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
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
	resp, err := bind.get("http://loc.bbys.cn/admin/device_location_reg_estimated_install/index/", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
ApproveExwarehouse
出库审批
@param id 出库id
*/func (bind *AdminApi[T]) ApproveExwarehouse(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "audit_status=1",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/exwarehouse/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
ExwarehouseNotice
出库通知
@param id 出库id
*/func (bind *AdminApi[T]) ExwarehouseNotice(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[message]":                  "陈文豪",
		"row[consignee_time]":           time.Now().Format("2006-01-02"),
		"row[exwarehouse_logistics_id]": 2,
		"row[sum]":                      1,
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/exwarehouse_notice/add?warehouse_id=%v&dialog=1", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateExwarehouseDevice
设备登记
@param exwareHouseId 出库id
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateExwarehouseDevice(exwareHouseId string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[device_id]":        deviceId,
		"row[exwarehouse_time]": time.Now().Format("2006-01-02"),
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/Exwarehouse_device/add.html?warehouse_id=%v&dialog=1", exwareHouseId), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateExwarehouseArrive
到货登记
@param applysn 点位号
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateExwarehouseArrive(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[logistics_status]": 1,
		"row[cost]":             "1.00",
		"row[arrive_remark]":    "测试添加",
		"row[images]":           "https://oss-fs.bbys.cn/admin/20221208/5f35494f2e96a.jpg,https://oss-fs.bbys.cn/admin/20221208/119f7f7cf0f30f612f36e6431768e191.png,https://oss-fs.bbys.cn/admin/20221208/aa61877ea09a02eca3d26c7c1a238a76.png,https://oss-fs.bbys.cn/admin/20221208/aa61877ea09a02eca3d26c7c1a238a77.jpeg",
		"row[device_id]":        deviceId,
		"row[arrive_date]":      time.Now().Format("2006-01-02"),
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/exwarehouse_arrive/edit?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
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
	resp, err := bind.get("http://loc.bbys.cn/admin/exwarehouse_arrive/index/", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
ApproveExwarehouseArrive
到货登记审批
@param id 到货登记id
*/func (bind *AdminApi[T]) ApproveExwarehouseArrive(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "approve_status=1",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/exwarehouse_arrive/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateArrivedStatement
添加物流费用结算单
@param arriveId 到货登记id
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateArrivedStatement(arriveId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[exwarehouse_logistics_id]": 2,
		"row[arrived_earliest]":         time.Now().Add(-24 * 7 * time.Hour).Format("2006-01-02 15:04:05"),
		"row[arrived_latest]":           time.Now().Format("2006-01-02 15:04:05"),
		"row[arrived_num]":              1,
		"row[total_cost]":               1,
		"row[average_cost]":             1,
		"row[applicant]":                "陈文豪",
		"row[remarks]":                  "测试添加",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/logistics_arrived_statement/add/arrive_id/%v?dialog=1", arriveId), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateInstallation
点位安装登记
@param applySn 点位号
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateInstallation(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[apply_sn]":                   applySn,
		"row[device_id]":                  deviceId,
		"row[map_zoom]":                   18,
		"row[map_center]":                 "",
		"row[coordinates]":                "22.54845664,114.06455184",
		"row[scene_picture][0][file]":     "https://oss-fs.bbys.cn/admin/20221208/b1286f27ae7a4aa48b0e977af698eee4.jpg",
		"row[scene_picture][1][file]":     "https://oss-fs.bbys.cn/admin/20221208/f46d93f4e7fc460fa57f3b824b49b1d9.jpg",
		"row[scene_picture][2][file]":     "https://oss-fs.bbys.cn/admin/20221208/6e658e0607894966b592fa46a4cfcabe.jpg",
		"row[scene_picture][3][file]":     "https://oss-fs.bbys.cn/admin/20221208/31be3bd1f5e145398001f127c2eb20ce.jpg",
		"row[status]":                     2,
		"row[complete_time]":              time.Now().Format("2006-01-02 15:04:05"),
		"row[power_number]":               666,
		"row[power_picture]":              "https://02a-certf05.bbys.cn/FS/Down/?id=004219c2685446aa8c1f7fa9207aeb48",
		"row[installor]":                  "陈文豪",
		"row[install_pic_list]":           "https://02a-certf05.bbys.cn/FS/Down/?id=314d8d2023b94e1884a4f5c250e82018",
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
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_install_reg/add/apply_sn/%v?dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
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
	resp, err := bind.get("http://loc.bbys.cn/admin/device_install_reg/index?admin_nav=11&sort=id&order=desc&offset=0&limit=10&op=%7B%22apply_name%22%3A%22EXTEND%22%7D", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
DeleteInstallation
删除安装登记
@param id 安装登记id
*/func (bind *AdminApi[T]) DeleteInstallation(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"action": "del",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/device_install_reg/del/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateWeaningApplication
撤机申请
@param applySn 点位号
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateWeaningApplication(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[original_apply_sn]": applySn,
		"row[device_id]":         deviceId,
		"row[weaning_type]":      2,
		"row[apply_sn]":          "0769000066",
		"row[weaning_time]":      time.Now().Format("2006-01-02 15:04:05"),
		"row[last_time]":         time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
		"row[property_tel]":      13883036130,
		"row[director]":          370,
		"row[remarks]":           "测试",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/terminal_weaning_device/add?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
ApproveWeaningApplication
审批撤机申请
@param id 到货登记id
*/func (bind *AdminApi[T]) ApproveWeaningApplication(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"params": "approve_status=1",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/terminal_weaning_device/multi/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}

/*
CreateWeaningReg
撤机登记
@param applySn 点位号
@param deviceId 设备号
*/func (bind *AdminApi[T]) CreateWeaningReg(applySn string, deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[apply_sn]":     applySn,
		"row[device_id]":    deviceId,
		"row[weaning_time]": time.Now().Format("2006-01-02 15:04:05"),
		"row[pic_list]":     "https://02a-certf05.bbys.cn/FS/Down/?id=fcd1a855f5e444c0869b1a0359fb9452,https://02a-certf05.bbys.cn/FS/Down/?id=3fa0ce7515bd4d969d042c2517a392bd,https://02a-certf05.bbys.cn/FS/Down/?id=68aba7fa2bed40269ca4f15606cd03c0,https://02a-certf05.bbys.cn/FS/Down/?id=e18bb7ccd7b842b288057b286b69136a",
		"row[operator]":     "陈文豪",
	}
	resp, err := bind.post(fmt.Sprintf("http://loc.bbys.cn/admin/weaning_reg/add?apply_sn=%v&dialog=1", applySn), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.toRet(resp)
}
