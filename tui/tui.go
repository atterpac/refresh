package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func Banner() {
	style := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Width(60).
		Height(1).
		Align(lipgloss.Center)

	fmt.Println(style.Render("GOTATO v0.0.1"))
}
