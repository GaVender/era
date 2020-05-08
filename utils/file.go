package utils

import (
	"io/ioutil"
	"os"
	"path"
)

func MustReadFile(fileName string) []byte {
	file, err := os.Open(path.Clean(fileName))
	if err != nil {
		panic("open file: " + err.Error())
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic("read file: " + err.Error())
	}

	return data
}
