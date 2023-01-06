package main

import (
	"encoding/json"
	"github.com/gookit/goutil"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime"
	"strings"
)

type ApiRequest[T any] struct {
	xx *http.Request
}

func (bind *ApiRequest[T]) getHistoryRequest(response *http.Response) (history []*http.Request) {
	response2 := *response
	response3 := &response2
	for response3 != nil {
		req := response3.Request
		history = append(history, req)
		response3 = req.Response
	}
	for l, r := 0, len(history)-1; l < r; l, r = l+1, r-1 {
		history[l], history[r] = history[r], history[l]
	}
	return history
}
func (bind *ApiRequest[T]) getChromeCookieJar(urlParse *url.URL) *cookiejar.Jar {
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
func (bind *ApiRequest[T]) get(urlStr string, params map[string]string) (*http.Response, error) {
	if runtime.GOOS == "darwin" {
		urlStr = strings.Replace(urlStr, "http://loc.bbys.cn/", "https://wx-dev.bbys.cn/", 1)
	}
	req, _ := http.NewRequest("GET", urlStr, nil)
	q := req.URL.Query()
	for i, v := range params {
		q.Add(i, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	return bind.do(req)
}
func (bind *ApiRequest[T]) post(urlStr string, params map[string]interface{}) (*http.Response, error) {
	if runtime.GOOS == "darwin" {
		urlStr = strings.Replace(urlStr, "http://loc.bbys.cn/", "https://wx-dev.bbys.cn/", 1)
	}
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
	return bind.do(req)
}

func (bind *ApiRequest[T]) do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Jar: bind.getChromeCookieJar(req.URL),
	}
	if runtime.GOOS == "darwin" {
		proxyURL, _ := url.Parse("http://127.0.0.1:9090")
		client.Transport = &http.Transport{
			//DisableCompression: true,
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, err
}
