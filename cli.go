package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arrivance/telenamer/telelib"
)

func main() {
	confirm := true
	format := "$s - S$0zE$0e - $n"

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

	files := telelib.GetFiles(".")
	rawFileInfo := telelib.ParseFilesWithSeries(files, "South Park")

	if confirm == false {
		var wg sync.WaitGroup
		for _, v := range rawFileInfo {
			wg.Add(1)
			go rename(v, login, &wg, format)
		}

		wg.Wait()
	} else {
		// Allowing the user to have control over the filename changes significantly slows down the operation,
		// so we'll go for a UX-best approach rather than prioritising performance.
		// The non-confirm section of the loop can deal with maximum performance.
		var parsedChans []chan telelib.ParsedFileInfo
		for _, v := range rawFileInfo {
			parsedChan := make(chan telelib.ParsedFileInfo)
			parsedChans = append(parsedChans, parsedChan)
			go func(v telelib.RawFileInfo, login telelib.TVDBLogin, parsedChan chan telelib.ParsedFileInfo) {
				parsedChan <- telelib.RetrieveEpisodeInfo(v, login)
			}(v, login, parsedChan)
		}

		var fileList []telelib.ParsedFileInfo
		for _, v := range parsedChans {
			fileList = append(fileList, <-v)
		}

		for _, v := range fileList {
			var input string
			fmt.Println("Old: " + v.FileName)
			fmt.Println("New: " + v.NewFileName(format))
			fmt.Print("Are you sure? y/n | ")
			fmt.Scanln(&input)

			if input == "y" {
				telelib.FileRename{OldFileName: v.FileName, NewFileName: v.NewFileName(format)}.RenameFile()
			}

			fmt.Println("------------")
		}

	}

	elapsed := time.Since(start)
	log.Printf("Program took took %s", elapsed)
}

func rename(v telelib.RawFileInfo, login telelib.TVDBLogin, wg *sync.WaitGroup, format string) {
	defer wg.Done()
	epInfo := telelib.RetrieveEpisodeInfo(v, login)

	telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName(format)}.RenameFile()
}
