package telelib

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

func TestParseFile(t *testing.T) {
	cases := []struct {
		in     string
		series string
		want   RawFileInfo
	}{
		{
			"03x01 - Rainforest Shmainforest.mkv",
			"South Park",
			RawFileInfo{FileName: "03x01 - Rainforest Shmainforest.mkv", Container: "mkv", Season: 3, Episode: 1, Series: "South Park"},
		},
		{
			"The Good Place - S04E07 - Help Is Other People.mkv",
			"",
			RawFileInfo{FileName: "The Good Place - S04E07 - Help Is Other People.mkv", Container: "mkv", Season: 4, Episode: 7, Series: "The Good Place"},
		},
		{
			"the.good.place.s04e12.1080p.blu.x264.mkv",
			"",
			RawFileInfo{FileName: "the.good.place.s04e12.1080p.blu.x264.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "the good place"},
		},
		{
			"The Good Place - 04x12 - Patty.mkv",
			"",
			RawFileInfo{FileName: "The Good Place - 04x12 - Patty.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
		},
		{
			"The Walking Dead S05E03 720p HDTV x264.mp4",
			"",
			RawFileInfo{FileName: "The Walking Dead S05E03 720p HDTV x264.mp4", Container: "mp4", Season: 5, Episode: 3, Series: "The Walking Dead"},
		},
		{
			"South Park S18E05 HDTV x264.mp4",
			"",
			RawFileInfo{FileName: "South Park S18E05 HDTV x264.mp4", Container: "mp4", Season: 18, Episode: 5, Series: "South Park"},
		},
		{
			"The Simpsons S26E05 HDTV x264.mkv",
			"",
			RawFileInfo{FileName: "The Simpsons S26E05 HDTV x264.mkv", Container: "mkv", Season: 26, Episode: 5, Series: "The Simpsons"},
		},
		{
			"South Park - [01x03] - Volcano.mkv",
			"",
			RawFileInfo{FileName: "South Park - [01x03] - Volcano.mkv", Container: "mkv", Season: 1, Episode: 3, Series: "South Park"},
		},
		{
			"South Park - [01x03] - Volcano.srt",
			"",
			RawFileInfo{FileName: "South Park - [01x03] - Volcano.srt", Container: "srt", Season: 1, Episode: 3, Series: "South Park"},
		},
		{
			"Test.png",
			"",
			RawFileInfo{invalid: true},
		},
	}

	for _, c := range cases {
		file := make(chan RawFileInfo)
		go parseFile(c.in, c.series, file)

		got := <-file

		if got != c.want {
			t.Errorf("parseFile(%q) == %+v\n, want %+v\n", c.in, got, c.want)
		}
	}
}

func TestParseFiles(t *testing.T) {
	fileList := []string{
		"The Good Place - S04E07 - Help Is Other People.mkv",
		"the.good.place.s04e12.1080p.blu.x264.mkv",
		"The Good Place - 04x12 - Patty.mkv",
	}

	expected := []RawFileInfo{
		{FileName: "The Good Place - S04E07 - Help Is Other People.mkv", Container: "mkv", Season: 4, Episode: 7, Series: "The Good Place"},
		{FileName: "the.good.place.s04e12.1080p.blu.x264.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "the good place"},
		{FileName: "The Good Place - 04x12 - Patty.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
	}

	result := parseFiles(fileList, "")

	for i, v := range result {
		found := false
		for _, x := range expected {
			if cmp.Equal(v, x, cmp.AllowUnexported(RawFileInfo{})) {
				found = true
			}
		}
		if found == false {
			t.Errorf("parseFiles(%q) == %+v\n, could not find %+v\n", fileList, result, expected[i])
		}
	}
}

func TestParseFilesInOrder(t *testing.T) {
	fileList := []string{
		"The Good Place - S04E07 - Help Is Other People.mkv",
		"the.good.place.s04e12.1080p.blu.x264.mkv",
		"The Good Place - 04x12 - Patty.mkv",
	}

	expected := []RawFileInfo{
		{FileName: "The Good Place - S04E07 - Help Is Other People.mkv", Container: "mkv", Season: 4, Episode: 7, Series: "The Good Place"},
		{FileName: "the.good.place.s04e12.1080p.blu.x264.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "the good place"},
		{FileName: "The Good Place - 04x12 - Patty.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
	}

	result := parseFilesInOrder(fileList, "")

	if !cmp.Equal(expected, result, cmp.AllowUnexported(RawFileInfo{})) {
		t.Errorf("parseFilesInOrder(%q) == %+v\n, could not find %+v\n", fileList, result, expected)
	}
}

func TestNewFileName(t *testing.T) {
	cases := []struct {
		in     ParsedFileInfo
		format string
		want   string
	}{
		{
			ParsedFileInfo{FileName: "", Container: "mkv", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{s} - {z}x{e} - {n}",
			"The Good Place - 5x1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{s} - {0z}x{0e} - {n}",
			"The Good Place - 05x01 - Backstreet's Back.mp4",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mkv", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{z}x{e} - {n}",
			"5x1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{0z}x{0e} - {n}",
			"05x01 - Backstreet's Back.mp4",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mkv", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{s} - S{z}E{e} - {n}",
			"The Good Place - S5E1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"{s} - S{0z}E{0e} - {n}",
			"The Good Place - S05E01 - Backstreet's Back.mp4",
		},
		{
			ParsedFileInfo{FileName: "", Container: "srt", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back?"},
			"{s} - S{0z}E{0e} - {n}",
			"The Good Place - S05E01 - Backstreet's Back.srt",
		},
	}

	for _, v := range cases {
		result := v.in.NewFileName(v.format)

		if result != v.want {
			t.Errorf("%q.RenameFile(%q) = %q, expected %q", v.in, v.format, result, v.want)
		}
	}
}

func TestGetFiles(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = &afero.Afero{Fs: fs}
	expected := []string{"test.mp4", "test2.mp4", "test3.srt"}
	unexpectedDir := []string{"test dir", "test dir 2"}

	for _, v := range expected {
		afero.WriteFile(fs, v, []byte("random contents"), 0644)
	}

	for _, v := range unexpectedDir {
		afero.NewMemMapFs().Mkdir(v, 0644)
	}

	result := GetFiles(".")

	if !cmp.Equal(result, expected) {
		t.Errorf("GetFiles(\".\") == %q, expected %q", result, expected)
	}
}

func TestRenameFile(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = &afero.Afero{Fs: fs}

	cases := []struct {
		in   FileRename
		want string
	}{
		{
			FileRename{OldFileName: "test.mp4", NewFileName: "new.mp4"},
			"new.mp4",
		},
		{
			FileRename{OldFileName: "test2.mp4", NewFileName: "new2.mp4"},
			"new2.mp4",
		},
	}

	for _, v := range cases {
		afero.WriteFile(fs, v.in.OldFileName, []byte("random contents"), 0644)
		v.in.RenameFile()
		result, err := afero.Exists(fs, v.want)
		if err != nil {
			log.Fatal(err)
		}
		if !result {
			t.Errorf("%+v.RenameFile() - %q was not found", v.in, v.want)
		}
	}
}

func TestRetrieveEpisodeInfo(t *testing.T) {
	loginFile, err := os.Open("login.json")
	if err != nil {
		log.Fatal("Could not load login.json: ", err)
	}

	defer loginFile.Close()

	var login TVDBLogin
	byteValue, _ := ioutil.ReadAll(loginFile)
	json.Unmarshal(byteValue, &login)

	cases := []struct {
		in   RawFileInfo
		want ParsedFileInfo
	}{
		{
			RawFileInfo{Season: 4, Episode: 7, Series: "The Good Place"},
			ParsedFileInfo{Season: 4, Episode: 7, Series: "The Good Place", EpisodeName: "Help Is Other People"},
		},
		{
			RawFileInfo{Season: 4, Episode: 7, Series: "the gOOd pLAce"},
			ParsedFileInfo{Season: 4, Episode: 7, Series: "The Good Place", EpisodeName: "Help Is Other People"},
		},
	}

	for _, v := range cases {
		result := RetrieveEpisodeInfo(v.in, login)
		if !cmp.Equal(result, v.want) {
			t.Errorf("RetrieveEpisodeInfo(%v)\n == %v\n, want %v\n", v.in, result, v.want)
		}
	}
}

func TestRenameFiles(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = &afero.Afero{Fs: fs}

	cases := []struct {
		in   FileRename
		want string
	}{
		{
			FileRename{OldFileName: "test.mp4", NewFileName: "new.mp4"},
			"new.mp4",
		},
		{
			FileRename{OldFileName: "test2.mp4", NewFileName: "new?2.mp4"},
			"new2.mp4",
		},
	}

	var expected []FileRename

	for _, v := range cases {
		afero.WriteFile(fs, v.in.OldFileName, []byte("random contents"), 0644)
		expected = append(expected, v.in)
	}

	RenameFiles(expected)

	for _, v := range cases {
		result, err := afero.Exists(fs, v.want)
		if err != nil {
			log.Fatal(err)
		}
		if !result {
			t.Errorf("%+v.RenameFile() - %q was not found", v.in, v.want)
		}
	}
}

func TestParseFilesPub(t *testing.T) {
	fileList := []string{
		"The Good Place - S04E07 - Help Is Other People.mkv",
		"the.good.place.s04e12.1080p.blu.x264.mkv",
		"The Good Place - 04x12 - Patty.mkv",
	}

	expected := []RawFileInfo{
		{FileName: "The Good Place - S04E07 - Help Is Other People.mkv", Container: "mkv", Season: 4, Episode: 7, Series: "The Good Place"},
		{FileName: "the.good.place.s04e12.1080p.blu.x264.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "the good place"},
		{FileName: "The Good Place - 04x12 - Patty.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
	}

	result := ParseFiles(fileList)

	for i, v := range result {
		found := false
		for _, x := range expected {
			if cmp.Equal(v, x, cmp.AllowUnexported(RawFileInfo{})) {
				found = true
			}
		}
		if found == false {
			t.Errorf("parseFiles(%q) == %+v\n, could not find %+v\n", fileList, result, expected[i])
		}
	}
}

func TestParseFilesWithSeries(t *testing.T) {
	fileList := []string{
		"The Good Place - S04E07 - Help Is Other People.mkv",
		"the.good.place.s04e12.1080p.blu.x264.mkv",
		"The Good Place - 04x12 - Patty.mkv",
		"04x12 - Patty.srt",
	}

	expected := []RawFileInfo{
		{FileName: "The Good Place - S04E07 - Help Is Other People.mkv", Container: "mkv", Season: 4, Episode: 7, Series: "The Good Place"},
		{FileName: "the.good.place.s04e12.1080p.blu.x264.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
		{FileName: "The Good Place - 04x12 - Patty.mkv", Container: "mkv", Season: 4, Episode: 12, Series: "The Good Place"},
		{FileName: "04x12 - Patty.srt", Container: "srt", Season: 4, Episode: 12, Series: "The Good Place"},
	}

	result := ParseFilesWithSeries(fileList, "The Good Place")

	for i, v := range result {
		found := false
		for _, x := range expected {
			if cmp.Equal(v, x, cmp.AllowUnexported(RawFileInfo{})) {
				found = true
			}
		}
		if found == false {
			t.Errorf("parseFiles(%q) == %+v\n, could not find %+v\n", fileList, result, expected[i])
		}
	}
}
