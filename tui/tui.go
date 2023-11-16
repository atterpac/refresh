package tui

import (
	"bufio"
	"gotato/log"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
)

func NewTui(color log.ColorScheme, logLevel int) (log.Logger, log.Logger) {
	processArea, _ := pterm.DefaultArea.Start()
	defer pterm.DefaultArea.Stop()
	logArea, _ := pterm.DefaultArea.Start()
	defer pterm.DefaultArea.Stop()
	processColors := log.ColorScheme{
		Info:  "",
		Debug: "",
		Error: "",
		Warn:  "",
		Fatal: "",
	}
	process := log.NewStyledLogger(processArea ,processColors, logLevel)
	logger := log.NewStyledLogger(logArea, color, logLevel)
	Banner("Gotato v0.0.1")
	return process, logger
}


func Banner(text string) {
	banner := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Width(60).
		Height(1).
		Align(lipgloss.Center)

	pterm.Println(banner.Render(text))
}

func PrintSubProcess(logger log.Logger, pipe io.ReadCloser, chunkSize int) {
	scanner := bufio.NewScanner(pipe)
	var lines []string

	styled := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(60).
		Height(chunkSize).
		Padding(0, 1)

	for {
		for scanner.Scan() {
			lines = append(lines, scanner.Text()+"\n")
			var lineString string
			if len(lines) > chunkSize {
				lines = lines[len(lines)-(chunkSize):]
				lineString = strings.Join(lines, "")
			} else {
				lineString = strings.Join(lines, "")
			}
			logger.Info(styled.Render(lineString))
		}

	}
}
