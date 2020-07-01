package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	// While this program doesn't support piracy, torrent names are typically some of the most varied and
	// file names - ergo, a torrent name parser will be far less brittle.
	parsetorrentname "github.com/middelink/go-parse-torrent-name"
)

type rawFileInfo struct {
	FileName string
	Season   int
	Episode  int
	Series   string
}

type fileRename struct {
	OldFileName string
	NewFileName string
}

func main() {
	for _, v := range parseFiles(getFiles()) {
		fmt.Println(v)
	}

	var fileList []fileRename
	fileList = append(fileList, fileRename{"test.txt", "test_1.txt"})
	fileList = append(fileList, fileRename{"test2.txt", "test_2.txt"})
	renameFiles(fileList)
}

func parseFiles(fileList []string) []rawFileInfo {
	var temp []rawFileInfo
	for _, fileName := range fileList {
		parsed, err := parsetorrentname.Parse(fileName)

		if err != nil {
			log.Fatal(err)
		}

		// Remove anything that isn't a video file.
		if parsed.Container != "" {
			temp = append(temp, rawFileInfo{fileName, parsed.Season, parsed.Episode, parsed.Title})
		}
	}

	return temp
}

func getFiles() []string {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}
	var fileList []string

	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, file.Name())
		}
	}

	return fileList
}

func renameFiles(renameList []fileRename) {
	for _, file := range renameList {
		err := os.Rename(file.OldFileName, file.NewFileName)

		if err != nil {
			log.Fatal(err)
		}
	}
}
