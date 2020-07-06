package telelib

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pioz/tvdb"

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

// ParsedFileInfo is the info about the file retrieved from an API provider.
type ParsedFileInfo struct {
	FileName    string
	Season      int
	Episode     int
	EpisodeName string
	Series      string
}

// FileRename keeps both the old filename and the new filename.
type FileRename struct {
	OldFileName string
	NewFileName string
}

// TVDBLogin replicates https://github.com/pioz/tvdb/blob/master/client.go's Client struct, with some additional JSON support.
type TVDBLogin struct {
	// The TVDB API key, User key, User name. You can get them here http://thetvdb.com/?tab=apiregister
	Apikey   string `json:"apikey"`
	Userkey  string `json:"userkey"`
	Username string `json:"username"`
	// The language with which you want to obtain the data (if not set english is
	// used)
	Language string
	token    string
	client   http.Client
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

// RetrieveEpisodeInfo retrieves the information for a episode.
func RetrieveEpisodeInfo(fileInfo RawFileInfo, login TVDBLogin) ParsedFileInfo {
	c := tvdb.Client{Apikey: login.Apikey, Userkey: login.Userkey, Username: login.Username}
	newFileInfo := ParsedFileInfo{FileName: fileInfo.FileName, Season: fileInfo.Season}

	err := c.Login()
	if err != nil {
		panic(err)
	}

	series, err := c.BestSearch(fileInfo.Series)
	if err != nil {
		panic(err)
	}
	// Retrieving this info from the API ensures capitalisation is correct.
	newFileInfo.Series = series.SeriesName

	err = c.GetSeriesEpisodes(&series, nil)
	if err != nil {
		panic(err)
	}
	episode := series.GetEpisode(fileInfo.Season, fileInfo.Episode)

	newFileInfo.EpisodeName = episode.EpisodeName
	newFileInfo.Episode = episode.AiredEpisodeNumber

	return newFileInfo
}

// NewFileName returns a file name.
func (p ParsedFileInfo) NewFileName(customFormat string) string {
	// Due to optional format strings $0e and $0z, I'm going to keep this simple text replacement vs a smarter templating
	// system for now...
	customFormat = strings.ReplaceAll(customFormat, "$s", p.Series)
	customFormat = strings.ReplaceAll(customFormat, "$n", p.EpisodeName)
	customFormat = strings.ReplaceAll(customFormat, "$e", strconv.Itoa(p.Episode))
	customFormat = strings.ReplaceAll(customFormat, "$0e", fmt.Sprintf("%02d", p.Episode))
	customFormat = strings.ReplaceAll(customFormat, "$z", strconv.Itoa(p.Season))
	customFormat = strings.ReplaceAll(customFormat, "$0z", fmt.Sprintf("%02d", p.Season))

	return customFormat
}
