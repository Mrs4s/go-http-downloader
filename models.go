package httpDownloader

import (
	"errors"
	"net/http"
)

type DownloaderInfo struct {
	Uris        []string          `json:"uris"`
	TargetFile  string            `json:"target_file"`
	Headers     map[string]string `json:"headers"`
	ContentSize int64             `json:"content_size"`
	BlockSize   int64             `json:"block_size"`
	ThreadCount int               `json:"thread_count"`
	BlockList   []DownloadBlock   `json:"block_list"`
}

type DownloadBlock struct {
	BeginOffset    int64 `json:"begin_offset"`
	EndOffset      int64 `json:"end_offset"`
	DownloadedSize int64 `json:"downloaded_size"`
	Downloading    bool  `json:"-"`
	Completed      bool  `json:"completed"`

	retryCount int `json:"-"`
}

func (info *DownloaderInfo) init() error {
	if info.Uris == nil || len(info.Uris) == 0 {
		return errors.New("uris cannot be nil")
	}
	info.BlockList = []DownloadBlock{}
	req, err := http.NewRequest("GET", info.Uris[0], nil)
	if err != nil {
		return err
	}
	if info.Headers != nil {
		for k, v := range info.Headers {
			req.Header[k] = []string{v}
		}
	}
	req.Header.Add("Range", "bytes=0-")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	info.ContentSize = resp.ContentLength
	var temp int64
	for temp+info.BlockSize < info.ContentSize {
		info.BlockList = append(info.BlockList, DownloadBlock{
			BeginOffset: temp,
			EndOffset:   temp + info.BlockSize - 1,
		})
		temp += info.BlockSize
	}
	info.BlockList = append(info.BlockList, DownloadBlock{
		BeginOffset: temp,
		EndOffset:   info.ContentSize - 1,
	})
	return nil
}

func (info *DownloaderInfo) getNextBlockN() int {
	for i, block := range info.BlockList {
		if !block.Completed && !block.Downloading {
			return i
		}
	}
	return -1
}

func (info *DownloaderInfo) allDownloaded() bool {
	for _, block := range info.BlockList {
		if !block.Completed || block.Downloading {
			return false
		}
	}
	return true
}

func NewDownloaderInfo(uris []string, targetFile string, blockSize int64, threadCount int, headers map[string]string) (*DownloaderInfo, error) {
	if blockSize <= 0 {
		blockSize = 1024 * 1024 * 100 //100MB
	}
	if threadCount <= 0 {
		threadCount = 1
	}
	info := &DownloaderInfo{
		Uris:        uris,
		TargetFile:  targetFile,
		BlockSize:   blockSize,
		ThreadCount: threadCount,
		Headers:     headers,
	}
	err := info.init()
	return info, err
}
