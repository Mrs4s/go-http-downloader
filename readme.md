# go-http-downloader
simple http downloader coding by golang

# how to use
```
info, err := httpDownloader.NewDownloaderInfo([]string {"url"},"target.file",1024*1024*10,16,map[string]string {"User-Agent":"xxx"})
if err != nil{
    fmt.Println(err.Error())
    return
}
client := httpDownloader.NewClient(info)
client.OnCompleted(func() {
	fmt.Println("completed.")
})
client.OnFailed(func(err error) {
	fmt.Println(err.Error())
})
_ = client.BeginDownload()
```