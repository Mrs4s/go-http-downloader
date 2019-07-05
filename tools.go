package httpDownloader

import "os"

func PathExists(path string) bool{
	_,err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func IsDir(path string)bool{
	if !PathExists(path){
		return false
	}
	info, _ := os.Stat(path)
	return info.IsDir()
}