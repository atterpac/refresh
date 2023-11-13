package tui
import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
)

const timeout = time.Second * 5

type Model struct {
	Timer    timer.Model
	Keymap   Keymap
	Help     help.Model
	Quitting bool
}

type Keymap struct {
	Start key.Binding
	Stop  key.Binding
	Reset key.Binding
	Quit  key.Binding
}

func (m Model) Init() tea.Cmd {
	return m.Timer.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timer.TickMsg:
		var cmd tea.Cmd
		m.Timer, cmd = m.Timer.Update(msg)
		return m, cmd

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.Timer, cmd = m.Timer.Update(msg)
		m.Keymap.Stop.SetEnabled(m.Timer.Running())
		m.Keymap.Start.SetEnabled(!m.Timer.Running())
		return m, cmd

	case timer.TimeoutMsg:
		m.Quitting = true
		return m, tea.Quit

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keymap.Quit):
			m.Quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.Keymap.Reset):
			m.Timer.Timeout = timeout
		case key.Matches(msg, m.Keymap.Start, m.Keymap.Stop):
			return m, m.Timer.Toggle()
		}
	}

	return m, nil
}

func (m Model) helpView() string {
	return "\n" + m.Help.ShortHelpView([]key.Binding{
		m.Keymap.Start,
		m.Keymap.Stop,
		m.Keymap.Reset,
		m.Keymap.Quit,
	})
}

func (m Model) View() string {
	// For a more detailed timer view you could read m.timer.Timeout to get
	// the remaining time as a time.Duration and skip calling m.timer.View()
	// entirely.
	s := m.Timer.View()

	if m.Timer.Timedout() {
		s = "All done!"
	}
	s += "\n"
	if !m.Quitting {
		s = "Exiting in " + s
		s += m.helpView()
	}
	return s
}

