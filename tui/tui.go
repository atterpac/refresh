package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func Banner(text string) {
	banner := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Width(60).
		Height(1).
		Align(lipgloss.Center)

	fmt.Println(banner.Render(text))
}
