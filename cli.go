package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/arrivance/telenamer/telelib"
)

func main() {
	loginFile, err := os.Open("login.json")

	if err != nil {
		fmt.Println(err)
	}

	defer loginFile.Close()

	var login telelib.TVDBLogin
	byteValue, _ := ioutil.ReadAll(loginFile)
	json.Unmarshal(byteValue, &login)

	var fileList []telelib.FileRename
	for _, v := range telelib.ParseFiles(telelib.GetFiles(".")) {
		fmt.Print(telelib.RetrieveEpisodeInfo(v, login).NewFileName("$s - $0zx$e - $n"))
	}

	// fileList = append(fileList, telelib.FileRename{OldFileName: "test.txt", NewFileName: "test_1.txt"})
	// fileList = append(fileList, telelib.FileRename{OldFileName: "test2.txt", NewFileName: "test_2.txt"})
	telelib.RenameFiles(fileList)
}
