package main

import (
	"github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

func ExecTempCrontab(t *testing.T) {
	args := os.Args
	url := args[2]
	execTime := time.Now()
	if execTime.Second() > 50 {
		ss, _ := time.ParseDuration("10s")
		execTime = execTime.Add(ss)
	}
	ss, _ := time.ParseDuration("1m")
	execTime = execTime.Add(ss)
	convey.Convey("执行临时任务", t, func() {
		pp(url)
		pp(execTime.Format("2006-01-02 15:04"))
		ret, err := adminApi.ExecTempCrontab(url, execTime, execTime.Add(ss))
		assertOk(ret, err)
	})
}
