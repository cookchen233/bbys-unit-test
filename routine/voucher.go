package routine

import (
	"fmt"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font/gofont/goregular"
	"image"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Voucher 二维码优惠券
type Voucher struct {
	Name         string
	TplFilename  string
	QrcodeW      int
	QrcodeX      int
	QrcodeY      int
	HasText      bool
	TextX        float64
	TextY        float64
	TextSize     float64
	SaveDir      string
	FontFilename string
	font         *truetype.Font
	bg           image.Image
	fm           string
}

type VoucherData struct {
	Text string `json:"ticket_number"`
	Url  string `json:"qrcode"`
}

var dirSep = string(os.PathSeparator)

func (bind *Voucher) setFont(filename string) {
	//fontBytes, _ := os.ReadFile(filename)
	//bind.font, _ = truetype.Parse(fontBytes)
	bind.font, _ = truetype.Parse(goregular.TTF)
}
func (bind *Voucher) setBg(filename string) {
	ff, err := os.Open(bind.TplFilename)
	defer ff.Close()
	if err != nil {
		panic(err)
	}
	bind.bg, bind.fm, err = image.Decode(ff)
	if err != nil {
		panic(err)
	}
}

func (bind *Voucher) MakeBatch(data []VoucherData) string {
	saveDir := bind.SaveDir + dirSep + time.Now().Format("20060102"+dirSep+"")
	dir := saveDir + strings.Replace(bind.Name, "/", "", -1)
	os.RemoveAll(dir)
	if _, err := os.Stat(dir); err != nil {
		os.MkdirAll(dir, 0755)
	}
	bind.setFont(bind.FontFilename)
	bind.setBg(bind.TplFilename)
	var wg sync.WaitGroup
	ch := make(chan string, len(data))
	go func() {
		for i, _ := range data {
			filename := <-ch
			fmt.Printf("\n%v complted %v/%v", filename, i+1, len(data))
		}
	}()
	ths := runtime.NumCPU() * 2
	step := int(math.Ceil(float64(len(data)) / float64(ths)))
	for i := 0; i < len(data); i += step {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < step; j++ {
				n := i + j
				if n >= len(data) {
					break
				}
				filename := fmt.Sprintf("%v%v%v.%v", dir, dirSep, n+1, bind.fm)
				bind.drawImage(data[n], filename)
				ch <- filename
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	return dir
}

func (bind *Voucher) drawImage(data VoucherData, filename string) {
	qr, err := qrcode.New(data.Url, qrcode.Medium)
	if err != nil {
		panic(err)
	}
	qr.DisableBorder = true
	dc := gg.NewContextForImage(bind.bg)
	dc.DrawImage(qr.Image(bind.QrcodeW), bind.QrcodeX, bind.QrcodeY)
	if bind.HasText {
		dc.SetFontFace(truetype.NewFace(bind.font, &truetype.Options{
			Size: bind.TextSize,
			DPI:  180,
		}))
		dc.SetRGB(0, 0, 0)
		dc.DrawString(data.Text, bind.TextX, bind.TextY)
	}
	if err := dc.SavePNG(filename); err != nil {
		panic(err)
	}
}
