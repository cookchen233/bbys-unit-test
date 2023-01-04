package main

import (
	"encoding/json"
	"fmt"
	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/stdutil"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
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
	log.AddHook(logrus_stack.NewHook(callerLevels, stackLevels))
	log.AddHook(RotateLogHook("log", "stdout.log", 7*24*time.Hour, 24*time.Hour))
}

type ApiRet struct {
	Body string
	Api  string
	Resp *http.Response
}
type Api[T any] struct {
	client *http.Client
}

func NewApi() *Api[any] {
	return (&Api[any]{}).Init()
}
func (bind *Api[T]) Init() *Api[T] {
	bind.client = &http.Client{}
	//proxyURL, _ := url.Parse("http://127.0.0.1:9090")
	transport := http.Transport{
		//DisableCompression: true,
		//Proxy:              http.ProxyURL(proxyURL),
	}
	bind.client.Transport = &transport
	return bind
}

func (bind *Api[T]) getChromeCookieJar(urlParse *url.URL) *cookiejar.Jar {
	file, err := os.Open("./results/chrome_default_cookie.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var data []map[string]interface{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}

	jar, _ := cookiejar.New(nil)
	var cookies []*http.Cookie
	for _, v := range data {
		if strings.Contains(urlParse.String(), v["Host"].(string)) {
			cookies = append(cookies, &http.Cookie{
				Name:   v["KeyName"].(string),
				Value:  v["Value"].(string),
				Path:   v["Path"].(string),
				Domain: v["Host"].(string),
			})
		}
	}
	jar.SetCookies(urlParse, cookies)
	return jar
}
func (bind *Api[T]) get(urlStr string, params map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	q := req.URL.Query()
	for i, v := range params {
		q.Add(i, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	bind.client.Jar = bind.getChromeCookieJar(req.URL)
	return bind.client.Do(req)
}
func (bind *Api[T]) post(urlStr string, params map[string]interface{}) (*http.Response, error) {
	formData := url.Values{}
	for i, v := range params {
		switch v := v.(type) {
		case []string:
			for _, s := range v {
				formData.Add(i, s)
			}
		default:
			formData.Add(i, goutil.String(v))
		}
	}
	req, _ := http.NewRequest("POST", urlStr, strings.NewReader(formData.Encode()))
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bind.client.Jar = bind.getChromeCookieJar(req.URL)
	return bind.client.Do(req)
}
func (bind *Api[T]) ToRet(response *http.Response) (*ApiRet, error) {
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
	log.Trace(ret.Api + "\n" + strings.Join([]string{
		"\n" + response.Request.Method + " " + response.Request.URL.String(),
		"reqBody:" + string(reqBody),
		"respBody:" + string(body),
	}, "\n"))
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		//bodyLen, _ := mathutil.Float(len(body))
		//cutLen, _ := mathutil.Int64(math.Min(999, bodyLen))
		log.Error(strings.Join([]string{"响应数据解析错误", err.Error(), ret.Api, "\n" + string(body)}, " "))
	}
	return &ret, err
}

/*
CreateDevice
创建一台新设备
@param deviceId 设备号
@param accessCode 授权码
*/
func (bind *Api[T]) CreateDevice(deviceId string, accessCode string) (*ApiRet, error) {
	params := map[string]interface{}{
		"device_code": accessCode,
		"device_id":   deviceId,
		"partner_id":  "2BCFA72F-A91C-0E5C-0AFF-33BCB318CC60",
		"model":       "A18S",
		"id":          "8",
		"_ajax":       "1",
	}
	resp, err := bind.post(fmt.Sprintf("https://wx-dev.bbys.cn/admin/terminal_device/initDevice%v", ""), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.ToRet(resp)
}

/*
UpDeviceStatus
更新设备状态设备
@param deviceId 设备号
*/
func (bind *Api[T]) UpDeviceStatus(deviceId string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    deviceId,
		"params": "status=2",
	}
	resp, err := bind.post(fmt.Sprintf("https://wx-dev.bbys.cn/admin/terminal_device/multi/ids/%v", deviceId), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.ToRet(resp)
}

/*
CreateLoc
待投放点位登记
@param name 点位名称
*/
func (bind *Api[T]) CreateLoc(name string) (*ApiRet, error) {
	params := map[string]interface{}{
		"row[partner_id]": "2BCFA72F-A91C-0E5C-0AFF-33BCB318CC60",
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
	resp, err := bind.post(fmt.Sprintf("https://wx-dev.bbys.cn/admin/device_location_reg/add?dialog=1%v", ""), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.ToRet(resp)
}

func (bind *Api[T]) GetLocList(filter string) (*ApiRet, error) {
	params := map[string]string{
		"filter": filter,
	}
	resp, err := bind.get("https://wx-dev.bbys.cn/admin/device_location_reg/index?admin_nav=10&sort=id&order=desc&offset=0&limit=10&op=%7B%22name%22%3A%22LIKE%22%7D", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.ToRet(resp)
}

func (bind *Api[T]) DelLoc(id string) (*ApiRet, error) {
	params := map[string]interface{}{
		"ids":    id,
		"action": "del",
	}
	resp, err := bind.post(fmt.Sprintf("https://wx-dev.bbys.cn/admin/device_location_reg/del/ids/%v", id), params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bind.ToRet(resp)
}
