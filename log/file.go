package log

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ReadLinesFromFile(path string, start,end int) (string, error) {
	var lines []string
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read lines within the specified range
	currentLine := 1
	for scanner.Scan() {
		if currentLine >= start && currentLine <= end {
			lines = append(lines, scanner.Text()+"\n")
		}
		currentLine++
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return joinLines(lines), nil
}

func joinLines(lines []string) string {
	return fmt.Sprintf("%s", lines)
}

func DeleteTempFile(filePath string) bool {
	log := GetLogger()
	log.Debug(fmt.Sprintf("Deleting file %s", filePath))
	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Debug(fmt.Sprintf("File %s does not exist.\n", filePath))
		return true
	} else if err != nil {
		return false
	}
	// Delete the file
	err = os.Remove(filePath)
	if err != nil {
		log.Debug(fmt.Sprintf("Error deleting file %s", filePath))
		return false
	}
	return true
}


func CreateTmpFile(label string) *os.File {
	log := GetLogger()
	// Trim any spaces from label
	tmpDir, err := os.MkdirTemp("", "gotato")
	if err != nil {
		log.Error(fmt.Sprintf("Error creating tmp dir: %s", err.Error()))
	}
	label = strings.ReplaceAll(label, " ", "")
	logFile, err := os.CreateTemp(tmpDir, fmt.Sprintf("gotato-%s", label))
	if err != nil {
		log.Error(fmt.Sprintln("Error creating log file", err.Error()))
	}
	return logFile
}

func ClearTmpFolders() error {
	matchingDirs, err := filepath.Glob(filepath.Join(os.TempDir(), "gotato*"))
	if err != nil {
		return err
	}
	for _, dir := range matchingDirs {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetFileSize(file *os.File) int64 { 
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file size", err.Error())
	}
	return fileInfo.Size()
}

