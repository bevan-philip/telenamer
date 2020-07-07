package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
		log.Print(v)
		epInfo := telelib.RetrieveEpisodeInfo(v, login)
		fileList = append(fileList, telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName("$s - S$0zE$0e - $n")})
	}

	telelib.RenameFiles(fileList)
}
