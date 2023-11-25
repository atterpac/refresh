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
