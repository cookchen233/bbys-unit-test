package main

import (
	"github.com/gookit/goutil"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tidwall/gjson"
	"io"
	"os"
	"testing"
	"time"
)

var adminApi = NewAdminApi()

type TestCase interface {
	Run(*testing.T)
}
type CreateDevice struct {
	Device map[string]string
	loop   int
}

func (bind *CreateDevice) Run(t *testing.T) {
	Convey("获取一台新设备", t, func() {
		file, err := os.OpenFile("./data/device_id", os.O_RDWR, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		id := goutil.Int(string(content)) + 1
		deviceId := "got-" + goutil.String(id)
		ret, err := adminApi.CreateDevice(deviceId, deviceId)
		So(err, ShouldBeNil)
		if gjson.Get(ret.Body, "msg").String() == "该授权码已使用" && bind.loop < 3 {
			bind.loop++
			bind.Run(t)
		}
		SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "status").Int(), ShouldEqual, 1)
		ret, err = adminApi.UpDeviceStatus(deviceId)
		So(err, ShouldBeNil)
		SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "status").Int(), ShouldEqual, 1)
		file.Truncate(0)
		file.Seek(0, 0)
		if _, err = file.WriteString(goutil.String(id)); err != nil {
			panic(err)
		}
		bind.Device = map[string]string{"device_id": deviceId}

	})
}

type CreateLoc struct {
	Loc string
}

func (bind *CreateLoc) Run(t *testing.T) {
	Convey("添加点位", t, func() {
		locName := "贵阳市花溪区" + time.Now().Format("2006-01-02 15:04:05")
		Println(locName)
		ret, err := adminApi.CreateLoc(locName)
		So(err, ShouldBeNil)
		SoMsg(gjson.Get(ret.Body, "msg").String(), gjson.Get(ret.Body, "code").Int(), ShouldNotBeZeroValue)
	})
}

type SetInstallTime struct {
	CreateLoc
}

func (bind *SetInstallTime) Run(t *testing.T) {
	bind.CreateLoc.Run(t)
	Convey("设置预计安装时间", t, func() {
		So(1 == 1, ShouldBeTrue)
		Println(bind.Loc)
	})
}

type AddExwarehouse struct {
	SetInstallTime
}

func (bind *AddExwarehouse) Run(t *testing.T) {
}

type DeleteLoc struct{}

func (bind *DeleteLoc) Run(t *testing.T) {
	ret, err := adminApi.GetLocList(`{"name": "贵阳市花溪区202"}`)
	Convey("删除点位", t, func() {
		So(err, ShouldBeNil)
		SoMsg("没有找到任何记录", gjson.Get(ret.Body, "total").Int(), ShouldBeGreaterThan, 0)
		gjson.Get(ret.Body, "rows").ForEach(func(_, row gjson.Result) bool {
			_, err := adminApi.DelLoc(row.Get("id").String())
			So(err, ShouldBeNil)
			//SoMsg(gjson.Get(ret.Body, "msg").String()+row.Get("name").String(), gjson.Get(ret.Body, "code").Int(), ShouldEqual, 1)
			return true
		})
	})
}
