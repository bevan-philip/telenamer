package telelib

import (
	"log"
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
	}

	for _, c := range cases {
		file := make(chan RawFileInfo)
		go parseFile(c.in, c.series, file)

		got := <-file

		if got != c.want {
			t.Errorf("parseFile(%q) == %q, want %q", c.in, got, c.want)
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
			if cmp.Equal(v, x) {
				found = true
			}
		}
		if found == false {
			t.Errorf("parseFiles(%q) == %q, could not find %q", fileList, result, expected[i])
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

	if !cmp.Equal(expected, result) {
		t.Errorf("parseFilesInOrder(%q) == %q, could not find %q", fileList, result, expected)
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
			"$s - $zx$e - $n",
			"The Good Place - 5x1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"$s - $0zx$0e - $n",
			"The Good Place - 05x01 - Backstreet's Back.mp4",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mkv", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"$zx$e - $n",
			"5x1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"$0zx$0e - $n",
			"05x01 - Backstreet's Back.mp4",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mkv", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"$s - S$zE$e - $n",
			"The Good Place - S5E1 - Backstreet's Back.mkv",
		},
		{
			ParsedFileInfo{FileName: "", Container: "mp4", Series: "The Good Place", Season: 5, Episode: 1, EpisodeName: "Backstreet's Back"},
			"$s - S$0zE$0e - $n",
			"The Good Place - S05E01 - Backstreet's Back.mp4",
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
	afero.WriteFile(fs, "test.mp4", []byte("file b"), 0644)
	afero.WriteFile(fs, "test2.mp4", []byte("file b"), 0644)
	afero.WriteFile(fs, "test3.srt", []byte("file b"), 0644)

	defer afero.NewMemMapFs().RemoveAll(".")

	result := GetFiles(".")
	expected := []string{"test.mp4", "test2.mp4", "test3.srt"}

	if !cmp.Equal(result, expected) {
		t.Errorf("GetFiles(\".\") == %q, expected %q", result, expected)
	}
}

func TestRenameFile(t *testing.T) {
	fs = afero.NewMemMapFs()
	fsutil = &afero.Afero{Fs: fs}

	rename := FileRename{OldFileName: "test.mp4", NewFileName: "new.mp4"}

	afero.WriteFile(fs, rename.OldFileName, []byte("file b"), 0644)
	defer afero.NewMemMapFs().RemoveAll(".")

	rename.RenameFile()
	result, err := afero.Exists(fs, rename.NewFileName)

	if err != nil {
		log.Fatal(err)
	}

	if !result {
		t.Errorf("%q.RenameFile() - %q was not found", rename, rename.NewFileName)
	}
}
