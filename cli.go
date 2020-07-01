package main

import (
	"fmt"

	"github.com/arrivance/telenamer/telelib"
)

func main() {
	for _, v := range telelib.ParseFiles(telelib.GetFiles(".")) {
		fmt.Println(v)
	}

	var fileList []telelib.FileRename
	fileList = append(fileList, telelib.FileRename{OldFileName: "test.txt", NewFileName: "test_1.txt"})
	fileList = append(fileList, telelib.FileRename{OldFileName: "test2.txt", NewFileName: "test_2.txt"})
	telelib.RenameFiles(fileList)
}
