package main

import (
	"net/http"
)

type ApiRet struct {
	Body string
	Api  string
	Resp *http.Response
}
