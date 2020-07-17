package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/akamensky/argparse"
	"github.com/arrivance/telenamer/telelib"
)

func main() {
	// Create new parser object
	parser := argparse.NewParser("telenamer", "Renames episodes within the folder.")

	format := parser.String("f", "format", &argparse.Options{Required: false, Help: "Format of renamed file: {s} = series, {n} = episode name, {e}/{0e} = episode number, {z}/{0z} = series number | 0e/z 0-prefixes numbers less than 10. Required unless -u provided."})
	series := parser.String("s", "series", &argparse.Options{Required: false, Help: "Name of series (if not provided, retrieved from file name.)"})
	confirm := parser.Flag("c", "confirm", &argparse.Options{Required: false, Help: "Manually confirm all name changes"})
	silent := parser.Flag("z", "silent", &argparse.Options{Required: false, Help: "Silent mode (does not work with -c)"})
	undo := parser.Flag("u", "undo", &argparse.Options{Required: false, Help: "Undos previous filenames (assuming you are in the same directory), and exits."})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		log.Fatal(parser.Usage(err))
	}

	if *silent {
		log.SetOutput(ioutil.Discard)
	}

	if *undo {
		undoRenames()
		os.Exit(1)
	} else if *format == "" {
		log.Println("No format provided.")
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

	files, err := telelib.GetFiles(".")
	if err != nil {
		log.Fatal("Error in retrieving files from directory | full error", err)
	}
	var rawFileInfo []telelib.RawFileInfo
	if *series == "" {
		rawFileInfo = telelib.ParseFiles(files)
	} else {
		rawFileInfo = telelib.ParseFilesWithSeries(files, *series)
	}

	if *confirm == false {
		// Store file renames, so that we can offer an undo option.
		renameChan := make(chan telelib.FileRename, len(rawFileInfo))
		for _, v := range rawFileInfo {
			go func(v telelib.RawFileInfo, login telelib.TVDBLogin, format string, rawChan chan telelib.FileRename) {
				epInfo, err := telelib.RetrieveEpisodeInfo(v, login)

				if err != nil {
					log.Fatal("error in retrieving episode info | full error: ", err)
				}

				telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName(format)}.RenameFile()
				log.Print(fmt.Sprintf("Renamed %q to %q", epInfo.FileName, epInfo.NewFileName(format)))

				rawChan <- telelib.FileRename{OldFileName: epInfo.FileName, NewFileName: epInfo.NewFileName(format)}
			}(v, login, *format, renameChan)
		}

		var renames []telelib.FileRename
		for range rawFileInfo {
			renames = append(renames, <-renameChan)
		}

		writeRenames(renames)

	} else {
		// Allowing the user to have control over the filename changes significantly slows down the operation,
		// so we'll go for a UX-best approach rather than prioritising performance.
		// The non-confirm section of the loop can deal with maximum performance.
		var parsedChans []chan telelib.ParsedFileInfo
		for _, v := range rawFileInfo {
			parsedChan := make(chan telelib.ParsedFileInfo)
			parsedChans = append(parsedChans, parsedChan)
			go func(v telelib.RawFileInfo, login telelib.TVDBLogin, parsedChan chan telelib.ParsedFileInfo) {
				result, err := telelib.RetrieveEpisodeInfo(v, login)

				if err != nil {
					log.Print(fmt.Sprintf("Error retrieving episode info for file %v, inferred info series %v, season %v, episode %v", v.FileName, v.Series, v.Season, v.Episode))
				}
				parsedChan <- result
			}(v, login, parsedChan)
		}

		var fileList []telelib.ParsedFileInfo
		for _, v := range parsedChans {
			fileList = append(fileList, <-v)
		}

		var renames []telelib.FileRename

		for _, v := range fileList {
			var input string
			fmt.Println("Old: " + v.FileName)
			fmt.Println("New: " + v.NewFileName(*format))
			fmt.Print("Are you sure? y/n | ")
			fmt.Scanln(&input)

			if input == "y" {
				rename := telelib.FileRename{OldFileName: v.FileName, NewFileName: v.NewFileName(*format)}
				rename.RenameFile()
				renames = append(renames, rename)
			}

			fmt.Println("------------")
		}

		writeRenames(renames)
	}
}

func writeRenames(renames []telelib.FileRename) {
	renamesJSON, err := json.Marshal(renames)
	if err != nil {
		log.Fatal(err)
	}

	ioutil.WriteFile(os.TempDir()+"\\telenamer_renames.json", renamesJSON, 0644)
}

func undoRenames() {
	renamesFile, err := os.Open(os.TempDir() + "\\telenamer_renames.json")
	if err != nil {
		log.Fatal("Could not load telenamer_renames.json: ", err)
	}

	defer renamesFile.Close()

	var renames []telelib.FileRename
	byteValue, _ := ioutil.ReadAll(renamesFile)
	json.Unmarshal(byteValue, &renames)

	for _, v := range renames {
		telelib.FileRename{OldFileName: v.NewFileName, NewFileName: v.OldFileName}.RenameFile()
	}

}
