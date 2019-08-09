package httpDownloader

import (
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

type DownloaderClient struct {
	Speed          int64
	DownloadedSize int64
	Downloading    bool
	Failed         bool
	FailedMessage  string
	Info           *DownloaderInfo

	onCompleted func()
	onFailed    func(err error)
}

func NewClient(info *DownloaderInfo) *DownloaderClient {
	client := &DownloaderClient{
		Info: info,
	}
	return client
}

func (client *DownloaderClient) BeginDownload() error {
	if IsDir(client.Info.TargetFile) {
		return errors.New("target file cannot be dir")
	}
	go func() {
		client.Downloading = true
		for _, block := range client.Info.BlockList {
			client.DownloadedSize += block.DownloadedSize
		}
		threadCount := int(math.Min(float64(client.Info.ThreadCount), float64(len(client.Info.BlockList))))

		ch := make(chan bool)
		for i := 0; i < threadCount; i++ {
			block := client.Info.getNextBlockN()
			if block != -1 {
				client.Info.BlockList[block].Downloading = true
				client.Info.BlockList[block].retryCount = 0
				go client.Info.BlockList[block].download(client, client.Info.Uris[0], ch) //TODO: auto switch uri
			}
		}
		go func() {
			for client.Downloading {
				stat := <-ch
				if stat == false {
					client.Downloading = false
					return
				}
				nextBlock := client.Info.getNextBlockN()
				if nextBlock == -1 {
					if client.Info.allDownloaded() {
						client.Downloading = false
						if client.onCompleted != nil {
							client.onCompleted()
						}
					}
					continue
				}
				client.Info.BlockList[nextBlock].Downloading = true
				client.Info.BlockList[nextBlock].retryCount = 0
				go client.Info.BlockList[nextBlock].download(client, client.Info.Uris[0], ch)
			}
			close(ch)
		}()
		go func() {
			for client.Downloading {
				oldSize := client.DownloadedSize
				time.Sleep(time.Duration(1) * time.Second)
				client.Speed = client.DownloadedSize - oldSize
			}
		}()
	}()
	return nil
}

func (block *DownloadBlock) download(client *DownloaderClient, uri string, ch chan bool) {
	req, err := http.NewRequest("GET", uri, nil)
	httpClient := http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     0,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 999,
		},
	}
	if err != nil {
		client.callFailed(err)
		ch <- false
		return
	}
	file, err := os.OpenFile(client.Info.TargetFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		client.callFailed(err)
		ch <- false
		return
	}
	defer file.Close()
	_, err = file.Seek(block.BeginOffset, 0)
	if err != nil {
		client.callFailed(err)
		ch <- false
		return
	}
	if client.Info.Headers != nil {
		for k, v := range client.Info.Headers {
			req.Header[k] = []string{v}
		}
	}
	req.Header.Set("range", "bytes="+strconv.FormatInt(block.BeginOffset, 10)+"-"+strconv.FormatInt(block.EndOffset, 10))
	resp, err := httpClient.Do(req)
	if err != nil {
		client.callFailed(err)
		ch <- false
		return
	}
	defer resp.Body.Close()
	var buffer = make([]byte, 1024)
	i, err := resp.Body.Read(buffer)
	for client.Downloading {
		if err != nil && err != io.EOF {
			if block.retryCount < 5 {
				block.retryCount++
				block.download(client, uri, ch)
				return
			}
			client.callFailed(err)
			block.Downloading = false
			ch <- false
			return
		}
		i64 := int64(len(buffer[:i]))
		needSize := block.EndOffset + 1 - block.BeginOffset
		if i64 > needSize {
			i64 = needSize
			err = io.EOF
		}
		_, e := file.Write(buffer[:i64])
		if e != nil {
			client.callFailed(e)
			block.Downloading = false
			ch <- false
			return
		}
		block.BeginOffset += i64
		block.DownloadedSize += i64
		client.DownloadedSize += i64
		if err == io.EOF || block.BeginOffset > block.EndOffset {
			block.Completed = true
			break
		}
		i, err = resp.Body.Read(buffer)
	}
	block.Downloading = false
	ch <- true
}

func (client *DownloaderClient) OnCompleted(fn func()) {
	client.onCompleted = fn
}

func (client *DownloaderClient) OnFailed(fn func(err error)) {
	client.onFailed = fn
}

func (client *DownloaderClient) callFailed(err error) {
	if client.onFailed != nil && !client.Failed {
		client.Failed = true
		client.FailedMessage = err.Error()
		client.onFailed(err)
	}
}
