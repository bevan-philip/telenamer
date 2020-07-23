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

// FileRenames are error prone operations, and we're performing it in a GoRoutine, so we
// ensure we have a way of reporting errors.
type fileRenameErr struct {
	FileRename telelib.FileRename
	Error      error
}

func main() {
	// Create new parser object, and arguments for CLI usage.
	parser := argparse.NewParser("telenamer", "Renames episodes within the folder.")

	format := parser.String(
		"f",
		"format",
		&argparse.Options{Required: false,
			Help: `Format of renamed file:
			{s} = series
			{n} = episode name 
			{e}/{0e} = episode number. {0e} prepends a 0 if the episode number is less than 10 
			{z}/{0z} = series number {0z} prepends a 0 if the series number is less than 10
			Default format: {s} - S{0z}E{0e} - {n}`,
			Default: "{s} - S{0z}E{0e} - {n}",
		})
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

	// If silent, we just discard all output. Might be slower than simply not outputting at all, but
	// performance difference is neglible.
	if *silent {
		log.SetOutput(ioutil.Discard)
	}

	if *undo {
		undoRenames()
		// Having multiple operations with undo just seems, excessive.
		os.Exit(1)
	}

	// Find the directory the executable is within.
	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Error finding directory of process: ", err)
	}
	exPath := filepath.Dir(ex)

	// Opens the login file.
	loginFile, err := os.Open(exPath + "\\login.json")
	if err != nil {
		log.Fatal("Could not load login.json, have you made it?: ", err)
	}

	defer loginFile.Close()

	// Converts the login file into a struct.
	var login telelib.TVDBLogin
	byteValue, _ := ioutil.ReadAll(loginFile)
	json.Unmarshal(byteValue, &login)

	// Retrieves the files from the directory. Fatal error if something goes wrong.
	files, err := telelib.GetFiles(".")
	if err != nil {
		log.Fatal("Error in retrieving files from directory | full error", err)
	}

	// Parse everything in the folder.
	var rawFileInfo []telelib.RawFileInfo
	if *series == "" {
		rawFileInfo = telelib.ParseFiles(files)
	} else {
		rawFileInfo = telelib.ParseFilesWithSeries(files, *series)
	}

	if *confirm == false {
		automatedRenames(rawFileInfo, login, *format)
	} else {
		seqeuentialRenames(rawFileInfo, login, *format)
	}
}

func writeRenames(renames []telelib.FileRename) {
	renamesJSON, err := json.Marshal(renames)
	if err != nil {
		log.Fatal(err)
	}

	// As it is a relatively small file, we'll store it within the OS' temporary store.
	// While good practice is to remove it, the purpose of this file is (temporary) persistence.
	// telenamer isn't a background task that can clean this up.
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
		// Flip it and run it through the same function again.
		telelib.FileRename{OldFileName: v.NewFileName, NewFileName: v.OldFileName}.RenameFile()
		log.Print(fmt.Sprintf("Renamed %v back to %v", v.NewFileName, v.OldFileName))
	}
}

func automatedRenames(rawFileInfo []telelib.RawFileInfo, login telelib.TVDBLogin, format string) {
	// Store file renames, so that we can offer an undo option.
	renameChan := make(chan fileRenameErr, len(rawFileInfo))

	for _, v := range rawFileInfo {
		// Create a GoRoutine that retrieves the episode for each info, and performs a rename operation.
		go func(v telelib.RawFileInfo, login telelib.TVDBLogin, format string, renameChan chan fileRenameErr) {
			epInfo, err := telelib.RetrieveEpisodeInfo(v, login)

			if err != nil {
				log.Print("error in retrieving episode info | full error: ", err)
				renameChan <- fileRenameErr{Error: err}
			} else {
				fileRename := epInfo.NewFileName(format)
				err := fileRename.RenameFile()

				if err != nil {
					log.Print("error in renaming file | full error: ", err)
					renameChan <- fileRenameErr{Error: err}
				} else {
					log.Print(fmt.Sprintf("Renamed %q to %q", fileRename.OldFileName, fileRename.NewFileName))
					renameChan <- fileRenameErr{FileRename: fileRename}
				}
			}
		}(v, login, format, renameChan)
	}

	// Ensure all the renames are performed, and add them to the renames list to write to disk.
	var renames []telelib.FileRename
	for range rawFileInfo {
		result := <-renameChan

		// We log any errors, so there is no need to actually use the info here.
		if result.Error == nil {
			renames = append(renames, result.FileRename)
		}
	}

	writeRenames(renames)
}

func seqeuentialRenames(rawFileInfo []telelib.RawFileInfo, login telelib.TVDBLogin, format string) {
	// Allowing the user to have control over the filename changes significantly slows down the operation,
	// so we'll go for a UX-best approach rather than prioritising performance.
	// The non-confirm section of the loop can deal with maximum performance.

	// Store a list of channels.
	var parsedChans []chan telelib.ParsedFileInfo
	for _, v := range rawFileInfo {
		parsedChan := make(chan telelib.ParsedFileInfo)
		// Adds each individual channel to a list, so that we can retrieve the results in-order later.
		parsedChans = append(parsedChans, parsedChan)
		go func(v telelib.RawFileInfo, login telelib.TVDBLogin, parsedChan chan telelib.ParsedFileInfo) {
			// Retireves the episode info.
			result, err := telelib.RetrieveEpisodeInfo(v, login)

			if err != nil {
				log.Print(fmt.Sprintf("Error retrieving episode info for file %v, inferred info series %v, season %v, episode %v", v.FileName, v.Series, v.Season, v.Episode))
			}

			parsedChan <- result
		}(v, login, parsedChan)
	}

	var renames []telelib.FileRename

	for _, v := range parsedChans {
		result := <-v
		// A blank struct is returned if there is an error, so we can just discard anything with a blank struct.
		if (result != telelib.ParsedFileInfo{}) {
			fileRename := result.NewFileName(format)
			var input string

			// Presents file rename for user to confirm.
			// Both isn't a log, and has to be displayed even if silent.
			fmt.Println("Old: " + fileRename.OldFileName)
			fmt.Println("New: " + fileRename.NewFileName)
			fmt.Print("Are you sure? y/n | ")
			fmt.Scanln(&input)

			// If they input a y, we'll rename the file and add it to the list of performed renames.
			if input == "y" {
				err := fileRename.RenameFile()
				if err != nil {
					log.Print("error renaming file | full error: ", err)
				} else {
					renames = append(renames, fileRename)
				}
			}
			fmt.Println("------------")
		}
	}

	writeRenames(renames)
}
