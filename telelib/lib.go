package telelib

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"

	"github.com/pioz/tvdb"

	// While this program doesn't support piracy, torrent names are typically some of the most varied and
	// file names - ergo, a torrent name parser will be far less brittle.
	parsetorrentname "github.com/middelink/go-parse-torrent-name"
)

// RawFileInfo retrieves the raw information from the file name.
type RawFileInfo struct {
	FileName  string
	Container string
	Season    int
	Episode   int
	Series    string
	invalid   bool
}

// ParsedFileInfo is the info about the file retrieved from an API provider.
type ParsedFileInfo struct {
	FileName    string
	Container   string
	Season      int
	Episode     int
	EpisodeName string
	Series      string
}

// FileRename keeps both the old filename and the new filename.
type FileRename struct {
	OldFileName string `json:"oldfilename"`
	NewFileName string `json:"newfilename"`
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

var (
	fs     afero.Fs
	fsutil *afero.Afero
)

func init() {
	// Utilise afero to handle file operations for easier testing.
	fs = afero.NewOsFs()
	fsutil = &afero.Afero{Fs: fs}
}

func parseFile(fileName string, series string, files chan RawFileInfo) {
	// ptn sometimes fails with well defined file names, that have seperators - this aims to find such
	// seperators and strip them from the filename and title..
	// e.g. "Test - EG", the Title would be "Test - ", instead of "Test".
	dividerRe, err := regexp.Compile(` ?(-|\||:|\[|\]) ?`)
	if err != nil {
		log.Fatal(err)
	}

	cleanFileName := dividerRe.ReplaceAllString(fileName, " ")

	parsed, err := parsetorrentname.Parse(cleanFileName)
	if err != nil {
		log.Fatal(err)
	}

	// Checks if file is a subtitle. Not included in base parser.
	subtitleRe, err := regexp.Compile(`(\.srt|\.txt|\.vtt\.scc\.stl)`)
	if err != nil {
		log.Fatal(err)
	}

	subtitle := subtitleRe.FindString(fileName)

	if series == "" {
		series = dividerRe.ReplaceAllString(parsed.Title, " ")
	}

	// Remove anything that isn't a video file.
	if parsed.Container != "" {
		files <- RawFileInfo{fileName, parsed.Container, parsed.Season, parsed.Episode, series, false}
	} else if subtitle != "" {
		// Note: while Golang does interpret strings as UTF8, and thus, if we were dealing with unknown strings, subtitle[1:]
		// would be error prone, we both know the string exists, and starts with ".", therefore, there is no risk.
		files <- RawFileInfo{fileName, subtitle[1:], parsed.Season, parsed.Episode, series, false}
	} else {
		// Can't just silently discard due to the new concurrency model.
		files <- RawFileInfo{invalid: true}
	}
}

// parseFiles parses a file list from GetFiles() and a series parameter.
// If series is "", it will attempt to retrieve this from the file name.
// Public functions are ParseFiles() and ParseFilesWithSeries()
func parseFiles(fileList []string, series string) []RawFileInfo {
	var temp []RawFileInfo
	files := make(chan RawFileInfo, len(fileList))
	for _, fileName := range fileList {
		go parseFile(fileName, series, files)
	}

	for range fileList {
		result := <-files
		if !result.invalid {
			temp = append(temp, result)
		}
	}

	return temp
}

// parseFilesInOrder
// parseFiles, but slightly worse, for UX/backwards compatability.
// The difference in execution is neglible.
func parseFilesInOrder(fileList []string, series string) []RawFileInfo {
	var temp []RawFileInfo
	var fileChans []chan RawFileInfo

	for _, fileName := range fileList {
		files := make(chan RawFileInfo)
		fileChans = append(fileChans, files)
		go parseFile(fileName, series, files)
	}

	for _, v := range fileChans {
		result := <-v
		if !result.invalid {
			temp = append(temp, result)
		}

	}

	return temp
}

// ParseFilesWithSeries parses a file list from GetFiles() where the title is not included within the file name.
// This assumes all the files in a folder are of the same series (reasonable when considering it would be impossible to sort otherwise)
func ParseFilesWithSeries(fileList []string, series string) []RawFileInfo {
	return parseFilesInOrder(fileList, series)
}

// ParseFiles parses a file list from GetFiles()
func ParseFiles(fileList []string) []RawFileInfo {
	return parseFilesInOrder(fileList, "")
}

// GetFiles retrieves a list of files from the current directory.
func GetFiles(directory string) []string {
	files, err := afero.ReadDir(fs, ".")
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
	var wg sync.WaitGroup
	for _, file := range renameList {
		wg.Add(1)
		go func(wg *sync.WaitGroup, file FileRename) {
			defer wg.Done()
			file.RenameFile()
		}(&wg, file)
	}
	wg.Wait()
}

// RetrieveEpisodeInfo retrieves the information for a episode.
func RetrieveEpisodeInfo(fileInfo RawFileInfo, login TVDBLogin) ParsedFileInfo {
	c := tvdb.Client{Apikey: login.Apikey, Userkey: login.Userkey, Username: login.Username}
	newFileInfo := ParsedFileInfo{FileName: fileInfo.FileName, Season: fileInfo.Season, Container: fileInfo.Container}

	err := c.Login()
	if err != nil {
		log.Fatal("Error in logging in.")
	}

	series, err := c.BestSearch(fileInfo.Series)
	if err != nil {
		log.Fatal("Error searching for series.")
	}
	// Retrieving this info from the API ensures capitalisation is correct.
	newFileInfo.Series = series.SeriesName

	err = c.GetSeriesEpisodes(&series, nil)
	if err != nil {
		log.Fatal("Error in searching for episode.")
	}
	episode := series.GetEpisode(fileInfo.Season, fileInfo.Episode)

	if episode == nil {
		log.Fatal("Unable to find episode.")
	}

	newFileInfo.EpisodeName = episode.EpisodeName
	newFileInfo.Episode = episode.AiredEpisodeNumber

	return newFileInfo
}

// NewFileName returns a file name.
func (p ParsedFileInfo) NewFileName(customFormat string) string {
	// Due to optional format strings {0e} and {0z}, I'm going to keep this simple text replacement vs a smarter templating
	// system for now...
	customFormat = strings.ReplaceAll(customFormat, "{s}", p.Series)
	customFormat = strings.ReplaceAll(customFormat, "{n}", p.EpisodeName)
	customFormat = strings.ReplaceAll(customFormat, "{e}", strconv.Itoa(p.Episode))
	customFormat = strings.ReplaceAll(customFormat, "{0e}", fmt.Sprintf("%02d", p.Episode))
	customFormat = strings.ReplaceAll(customFormat, "{z}", strconv.Itoa(p.Season))
	customFormat = strings.ReplaceAll(customFormat, "{0z}", fmt.Sprintf("%02d", p.Season))

	// Removes characters that aren't accepted in Windows file names.
	winInvalidName, err := regexp.Compile(`(\?|\\|\/|\*|\:|"|<|>|\|)`)
	if err != nil {
		log.Fatal(err)
	}
	customFormat = winInvalidName.ReplaceAllString(customFormat, "")

	return fmt.Sprintf("%s.%s", customFormat, p.Container)
}

// RenameFile renames the file based on the contents of the struct.
func (file FileRename) RenameFile() {
	err := fs.Rename(file.OldFileName, file.NewFileName)

	if err != nil {
		log.Fatal("Error in renaming files. ", err)
	}
}
