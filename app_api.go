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

type AppDataError struct {
	message string
}

func (bind *AppDataError) Error() string {
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

// AppApi 后台接口
type AppApi[T any] struct {
	apiClient *ApiClient
	baseUrl   string
}

func NewAppApi() *AppApi[any] {
	return &AppApi[any]{
		apiClient: NewApiClient(),
		baseUrl:   os.Getenv("BASE_URL") + "mobile/",
	}
}

func (bind *AppApi[T]) toRet(resp *http.Response, args ...interface{}) (*ApiRet, error) {
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
		err = errors.WithStack(&AppDataError{fmt.Sprintf("%v(%v)", errMsg, errCode)})
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
CreatePrintOrder
下单
*/
func (bind *AppApi[T]) CreatePrintOrder() (*ApiRet, error) {
	params, _ := json.Marshal(map[string]interface{}{
		"token":           "xx",
		"order_type":      0,
		"printer_id":      "302048",
		"source":          "redPacket",
		"redPacketAmount": 1,
		"frame_id":        1,
		"p_id":            1,
		"is_water_mask":   1,
	})
	req, _ := http.NewRequest("POST", fmt.Sprintf(bind.baseUrl+"app_add_order/add?templ_id=%v&dialog=1", ""), strings.NewReader(string(params)))
	resp, err := bind.apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	return bind.toRet(resp)
}
