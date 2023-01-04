package main

import (
	"flag"
	"strings"
	"testing"
)

var testCaseName string

// 测试对象名称列表, 用于命令行参数映射
var testCaseList = map[string]func() TestCase{
	"CreateLoc": func() TestCase { return &CreateLoc{} },
	"DelLoc":    func() TestCase { return &DelLoc{} },
}

func init() {
	var testCaseNameList []string
	for k, _ := range testCaseList {
		testCaseNameList = append(testCaseNameList, k)
	}
	flag.StringVar(&testCaseName, "t", "", "测试对象名称\n"+strings.Join(testCaseNameList, "\n")+"\n")
}

func Test(t *testing.T) {
	flag.Parse()
	testCaseList[testCaseName]().Run(t)
}
