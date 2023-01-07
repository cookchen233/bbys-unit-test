package main

import (
	"net/http"
	"reflect"
)

type ApiRet struct {
	Body   string
	Method string
	Args   []reflect.Value
	Resp   *http.Response
}
