package tui

import (
	"fmt"
	"os"

	"github.com/atterpac/refresh/engine"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func StartTui() {
	engine, err := engine.NewEngineFromTOML("./example/example.toml")
	if err != nil {
		panic(err)
	}
	executes := engine.ProcessManager.GetExecutes()
	println("count", len(executes))
	items := make([]list.Item, len(executes))
	for _, execute := range executes {
		items = append(items, item{title: execute, desc: "Description"})
	}
	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 5)}
	m.list.Title = "Dev Mode Active"

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}
