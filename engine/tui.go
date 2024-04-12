package engine

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
var Tui *tea.Program

type TuiMsg struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Execute     string
}

var MsgChan = make(chan TuiMsg)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle      = helpStyle.Copy().UnsetMargins()
	durationStyle = dotStyle.Copy()
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
)


func (r TuiMsg) String() string {
	if r.Duration == 0 {
		return dotStyle.Render(strings.Repeat(".", 30))
	}
	return fmt.Sprintf("%s %s", r.Execute,
		durationStyle.Render(r.Duration.String()))
}

type model struct {
	spinner  spinner.Model
	results  []TuiMsg
	quitting bool
}

func newModel() model {
	const numLastResults = 5
	s := spinner.New()
	s.Style = spinnerStyle
	return model{
		spinner: s,
		results: make([]TuiMsg, numLastResults),
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case TuiMsg:
		m.results = append(m.results[1:], msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m model) View() string {
	var s string

	if m.quitting {
		s += "That’s all for today!"
	} else {
		s += m.spinner.View() + " Building Application..."
	}

	s += "\n\n"

	for _, res := range m.results {
		s += res.String() + "\n"
	}

	if !m.quitting {
		s += helpStyle.Render("Press any key to exit")
	}

	if m.quitting {
		s += "\n"
	}

	return appStyle.Render(s)
}

func StartTea() {
	Tui = tea.NewProgram(newModel())

	_ , err:= NewEngineFromTOML("./example/example.toml")
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	// go engine.Start()
	go func() {
		for {
			msg := <-MsgChan
			if msg.Execute == "" {
				continue
			}
			Tui.Send(msg)
		}
	}()


	if _, err := Tui.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func randomExecute() string {
	exe := []string{
		"go mod tidy", "go build", "go run .", "/bin/myApp", "go test ./...",
	}
	return exe[rand.Intn(len(exe))] // nolint:gosec
}

