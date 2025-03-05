package engine

import (
	"bufio"
	_ "embed"
	"strings"
	"testing"
)

//go:embed testdata/ignore.txt
var testIgnoreData string

func Test_patternCompare(t *testing.T) {
	// Process the ignore data by reading each line, trimming whitespace, ignore line if first char is #, split into fields
	// and test the values
	scanner := bufio.NewScanner(strings.NewReader(testIgnoreData))
	for scanner.Scan() {
		// trim whitespace
		line := strings.TrimSpace(scanner.Text())
		// ignore empty lines or lines that start with #
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// split into fields
		fields := strings.Fields(line)

		// Call patternCompare with the fields
		result := patternCompare(fields[0], fields[1])
		want := fields[2] == "true"
		if result != want {
			t.Errorf("patternCompare(%s, %s) = %t; want %s", fields[0], fields[1], result, fields[2])
		}
	}
}

func Test_isWatchedExtension(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		watchedExten  []string
		wantIsWatched bool
	}{
		{
			name:          "go file with full path should match *.go",
			path:          "/Users/atterpac/projects/atterpac/refresh/example/test/monitored/ignore.go",
			watchedExten:  []string{"*.go"},
			wantIsWatched: true,
		},
		{
			name:          "go file with just extension match",
			path:          "/some/path/file.go",
			watchedExten:  []string{".go"},
			wantIsWatched: true,
		},
		{
			name:          "go file with *extension match",
			path:          "/some/path/file.go",
			watchedExten:  []string{"*.go"},
			wantIsWatched: true,
		},
		{
			name:          "txt file should not match go patterns",
			path:          "/some/path/file.txt",
			watchedExten:  []string{"*.go", ".go"},
			wantIsWatched: false,
		},
		{
			name:          "multiple extensions",
			path:          "/some/path/file.js",
			watchedExten:  []string{"*.go", "*.js", "*.html"},
			wantIsWatched: true,
		},
		{
			name:          "no extension in path",
			path:          "/some/path/noextension",
			watchedExten:  []string{"*.go", "*.js"},
			wantIsWatched: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Ignore{
				WatchedExten: tt.watchedExten,
			}
			got := i.isWatchedExtension(tt.path)
			if got != tt.wantIsWatched {
				t.Errorf("isWatchedExtension() = %v, want %v", got, tt.wantIsWatched)
			}
		})
	}
}
