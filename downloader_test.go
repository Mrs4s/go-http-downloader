package httpDownloader

import (
	"fmt"
	"testing"
	"time"
)

func TestDownload(t *testing.T) {
	info, err := NewDownloaderInfo(
		[]string{"https://dldir1.qq.com/qqfile/qq/PCQQ9.1.6/25786/QQ9.1.6.25786.exe"},
		"I:\\temp\\test.t", 0, 10, map[string]string{})
	if err != nil {
		fmt.Println(err)
		return
	}
	cli := NewClient(info)
	err = cli.BeginDownload()
	if err != nil {
		fmt.Println(err)
		return
	}
	cli.OnCompleted(func() {
		fmt.Println("completed")
	})
	cli.OnFailed(func(err error) {
		fmt.Println("failed:", err)
	})
	time.Sleep(time.Second)
	for cli.Downloading {
		time.Sleep(time.Second)
		fmt.Println("speed:", cli.Speed)
	}
}
