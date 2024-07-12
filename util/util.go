package util

import (
	"os"
	"strings"
)

func IsCSV(fileName string) bool {
	splittedDir := strings.Split(fileName, "/")
	splittedFileName := strings.Split(splittedDir[len(splittedDir)-1], ".")
	return len(splittedFileName) > 1 && splittedFileName[1] == "csv"
}

func IsFileExist(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	return true
}
