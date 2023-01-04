package main

import (
	"flag"
	"golang.org/x/exp/maps"
	"strings"
	"testing"
)

// 测试对象名称列表, 用于命令行参数映射
var testCaseList = map[string]func() TestCase{
	"CreateLoc": func() TestCase { return &CreateLoc{} },
	"DelLoc":    func() TestCase { return &DelLoc{} },
}
var testCaseName string

func init() {
	flag.StringVar(&testCaseName, "t", "DelLoc", "测试对象名称\n"+strings.Join(maps.Keys(testCaseList), "\n")+"\n")
}

func Test(t *testing.T) {
	flag.Parse()
	testCaseList[testCaseName]().(TestCase).Run(t)
}
