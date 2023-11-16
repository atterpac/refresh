package tui

import (
	"bufio"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
)

func Banner(text string) {
	banner := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Width(60).
		Height(1).
		Align(lipgloss.Center)

	pterm.Println(banner.Render(text))
}

func PrintSubProcess(area *pterm.AreaPrinter ,pipe io.ReadCloser, chunkSize int) {
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
			area.Update(styled.Render(lineString))
		}

	}
}
