package telelib

import (
	"io/ioutil"
	"log"
	"os"

	// While this program doesn't support piracy, torrent names are typically some of the most varied and
	// file names - ergo, a torrent name parser will be far less brittle.
	parsetorrentname "github.com/middelink/go-parse-torrent-name"
)

// RawFileInfo retrieves the raw information from the file name.
type RawFileInfo struct {
	FileName string
	Season   int
	Episode  int
	Series   string
}

// FileRename keeps both the old filename and the new filename.
type FileRename struct {
	OldFileName string
	NewFileName string
}

// ParseFiles parses a file list from GetFiles()
func ParseFiles(fileList []string) []RawFileInfo {
	var temp []RawFileInfo
	for _, fileName := range fileList {
		parsed, err := parsetorrentname.Parse(fileName)

		if err != nil {
			log.Fatal(err)
		}

		// Remove anything that isn't a video file.
		if parsed.Container != "" {
			temp = append(temp, RawFileInfo{fileName, parsed.Season, parsed.Episode, parsed.Title})
		}
	}

	return temp
}

// GetFiles retrieves a list of video files from the current directory.
func GetFiles(directory string) []string {
	files, err := ioutil.ReadDir(directory)
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

// RenameFiles renames the list of files given.
func RenameFiles(renameList []FileRename) {
	for _, file := range renameList {
		err := os.Rename(file.OldFileName, file.NewFileName)

		if err != nil {
			log.Fatal(err)
		}
	}
}
