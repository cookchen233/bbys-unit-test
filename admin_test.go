package main

import (
	"testing"
)

// 测试对象名称列表, 用于命令行参数映射
//
//	var testCaseList = map[string]func() TestCase{
//		"CreateLoc": func() TestCase { return &CreateLoc{} },
//		"DeleteLoc": func() TestCase { return &DeleteLoc{} },
//	}
//
// var testCaseName string
//
//	func init() {
//		flag.StringVar(
//			&testCaseName,
//			"t",
//			"DeleteLoc",
//			"测试对象名称,多个使用逗号分割\n"+strings.Join(maps.Keys(testCaseList), "\n")+"\n",
//		)
//	}
//
//	func Test(t *testing.T) {
//		flag.Parse()
//		testCaseNameList := strings.Split(testCaseName, ",")
//		for _, name := range testCaseNameList {
//			testCaseList[name]().Run(t)
//		}
//	}
func TestCreateLoc(t *testing.T) {
	(&CreateLoc{}).Run(t)
}
func TestDeleteLoc(t *testing.T) {
	(&DeleteLoc{}).Run(t)
}
