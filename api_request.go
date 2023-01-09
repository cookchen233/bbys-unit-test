package main

import (
	"bytes"
	"encoding/json"
	"github.com/gookit/goutil"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

// ApiClient http请求基础对象
type ApiClient struct {
	client  *http.Client
	cookies []*http.Cookie
}

func NewApiClient() *ApiClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}
	if os.Getenv("PROXY_ENABLED") == "true" {
		proxyURL, _ := url.Parse(os.Getenv("PROXY_URL"))
		client.Transport = &http.Transport{
			//DisableCompression: true,
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	return &ApiClient{
		client:  client,
		cookies: make([]*http.Cookie, 0),
	}
}

func (bind *ApiClient) SetCookie(urlParse *url.URL, cookie *http.Cookie) error {
	file, err := os.OpenFile("./results/chrome_default_cookie.json", os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer file.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return errors.WithStack(err)
	}
	var data []map[string]interface{}
	if len(content) > 0 {
		err = json.Unmarshal(content, &data)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	var data2 []map[string]interface{}
	var cookies []*http.Cookie
	for _, v := range data {
		if v["KeyName"] != cookie.Name || v["Host"] != cookie.Domain {
			cookies = append(cookies, &http.Cookie{
				Name:   v["KeyName"].(string),
				Value:  v["Value"].(string),
				Path:   v["Path"].(string),
				Domain: v["Host"].(string),
			})
			data2 = append(data2, v)
		}
	}
	if cookie.Name != "" && cookie.Domain != "" {
		cookies = append(cookies, cookie)
		data2 = append(data2, map[string]interface{}{
			"KeyName": cookie.Name,
			"Value":   cookie.Value,
			"Path":    cookie.Path,
			"Host":    cookie.Domain,
		})
	}
	bind.client.Jar.SetCookies(urlParse, cookies)
	content, err = json.Marshal(data2)
	if err != nil {
		return errors.WithStack(err)
	}
	var jsonBuffer bytes.Buffer
	_ = json.Indent(&jsonBuffer, content, "", "    ")
	file.Truncate(0)
	file.Seek(0, 0)
	_, err = file.Write(jsonBuffer.Bytes())
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
func (bind *ApiClient) GetHistoryRequests(resp *http.Response) (history []*http.Request) {
	for resp != nil {
		req := resp.Request
		history = append(history, req)
		resp = req.Response
	}
	for l, r := 0, len(history)-1; l < r; l, r = l+1, r-1 {
		history[l], history[r] = history[r], history[l]
	}
	return history
}

func (bind *ApiClient) Get(urlStr string, params map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	q := req.URL.Query()
	for i, v := range params {
		q.Add(i, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	return bind.Do(req)
}
func (bind *ApiClient) Post(urlStr string, params map[string]interface{}) (*http.Response, error) {
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
	return bind.Do(req)
}

func (bind *ApiClient) Do(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Method == "POST" {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}
	bind.SetCookie(req.URL, &http.Cookie{})
	resp, err := bind.client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// 如果发生重定向, request的Body内容会被清空
	resp2 := resp
	for resp2 != nil {
		r := resp2.Request
		for _, cookie := range resp.Cookies() {
			err = bind.SetCookie(r.URL, &http.Cookie{
				Name:   cookie.Name,
				Value:  cookie.Value,
				Path:   cookie.Path,
				Domain: r.Host,
			})
			if err != nil {
				return resp, err
			}
		}
		if r.Response == nil {
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			break
		}
		resp2 = r.Response
	}

	return resp, nil
}
