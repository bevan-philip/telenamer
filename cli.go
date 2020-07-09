package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/arrivance/telenamer/telelib"
)

func main() {
	start := time.Now()
	// Find the directory the executable is within.
	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Error finding directory of process: ", err)
	}
	exPath := filepath.Dir(ex)

	loginFile, err := os.Open(exPath + "\\login.json")
	if err != nil {
		log.Fatal("Could not load login.json: ", err)
	}

	defer loginFile.Close()

	var login telelib.TVDBLogin
	byteValue, _ := ioutil.ReadAll(loginFile)
	json.Unmarshal(byteValue, &login)

	for _, v := range telelib.ParseFilesWithSeries(telelib.GetFiles("."), "South Park") {
		epInfo := telelib.RetrieveEpisodeInfo(v, login)
		go telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName("$s - S$0zE$0e - $n")}.RenameFile()
	}

	elapsed := time.Since(start)
	log.Printf("Program took took %s", elapsed)
}
