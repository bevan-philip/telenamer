package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/akamensky/argparse"
	"github.com/arrivance/telenamer/telelib"
)

func main() {
	// Create new parser object
	parser := argparse.NewParser("telenamer", "Renames episodes within the folder.")

	format := parser.String("f", "format", &argparse.Options{Required: true, Help: "Format of renamed file: {s} = series, {n} = episode name, {e}/{0e} = episode number, {z}/{0z} = series number | 0e/z 0-prefixes numbers less than 10."})
	series := parser.String("s", "series", &argparse.Options{Required: false, Help: "Name of series (if not provided, retrieved from file name.)"})
	confirm := parser.Flag("c", "confirm", &argparse.Options{Required: false, Help: "Manually confirm all name changes"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		log.Fatal(parser.Usage(err))
	}

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
	var rawFileInfo []telelib.RawFileInfo
	if *series == "" {
		rawFileInfo = telelib.ParseFiles(files)
	} else {
		rawFileInfo = telelib.ParseFilesWithSeries(files, *series)
	}

	if *confirm == false {
		var wg sync.WaitGroup
		for _, v := range rawFileInfo {
			wg.Add(1)
			go rename(v, login, &wg, *format)
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
			fmt.Println("New: " + v.NewFileName(*format))
			fmt.Print("Are you sure? y/n | ")
			fmt.Scanln(&input)

			if input == "y" {
				telelib.FileRename{OldFileName: v.FileName, NewFileName: v.NewFileName(*format)}.RenameFile()
			}

			fmt.Println("------------")
		}

	}
}

func rename(v telelib.RawFileInfo, login telelib.TVDBLogin, wg *sync.WaitGroup, format string) {
	defer wg.Done()
	epInfo := telelib.RetrieveEpisodeInfo(v, login)

	telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName(format)}.RenameFile()
}
