package log

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
)

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

type LogStyles struct {
	Debug lipgloss.Style
	Info  lipgloss.Style
	Warn  lipgloss.Style
	Error lipgloss.Style
	Fatal lipgloss.Style
}

type ColorScheme struct {
	Debug string `toml:"debug"`
	Info  string `toml:"info"`
	Warn  string `toml:"warn"`
	Error string `toml:"error"`
	Fatal string `toml:"fatal"`
}

type StyledLogger struct {
	area 	     *pterm.AreaPrinter
	styles       LogStyles
	loggingLevel int
}

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func GetLogger() *StyledLogger {
	return &StyledLogger{}
}

func NewStyledLogger(area *pterm.AreaPrinter, scheme ColorScheme, level int) *StyledLogger {
	styles := setColorScheme(scheme)
	return &StyledLogger{area, styles, level}
}

func (l *StyledLogger) Debug(format string, args ...interface{}) {
	if l.loggingLevel <= DebugLevel {
		l.log(l.styles.Debug, format,args...)
	}
}

func (l *StyledLogger) DebugString(format string, args ...interface{}) string {
	if l.loggingLevel <= DebugLevel {
		message := fmt.Sprintf(format, args...)
		styledMessage := applyStyle(message, l.styles.Debug)
		return styledMessage
	}
	return ""
}

func (l *StyledLogger) Info(format string, args ...interface{}) {
	if l.loggingLevel <= InfoLevel {
		l.log(l.styles.Info, format ,args...)
	}
}

func (l *StyledLogger) InfoString(format string, args ...interface{}) string {
	if l.loggingLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		styledMessage := applyStyle(message, l.styles.Info)
		return styledMessage
	}
	return ""
}

func (l *StyledLogger) Warn(format string, args ...interface{}) {
	if l.loggingLevel <= WarnLevel {
		l.log(l.styles.Warn, format ,args...)
	}
}

func (l *StyledLogger) Error(format string, args ...interface{}) {
	if l.loggingLevel <= ErrorLevel {
		l.log(l.styles.Error, format ,args...)
	}
}

func (l *StyledLogger) Fatal(format string, args ...interface{}) {
	if l.loggingLevel <= FatalLevel {
		l.log(l.styles.Fatal, format,args...)
	}
	os.Exit(1)
}

func (l *StyledLogger) log(style lipgloss.Style, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	styledMessage := applyStyle(message, style)
	l.area.Update(styledMessage)
}

// Set Logging Level
// 0 - Debug
// 1 - Info
// 2 - Warn
// 3 - Error
// 4 - Fatal
func (l *StyledLogger) SetLoggingLevel(level int) {
	if level >= DebugLevel && level <= FatalLevel {
		l.loggingLevel = level
	} else {
		l.loggingLevel = InfoLevel
	}
}

func (l *StyledLogger) SetColorScheme(scheme LogStyles) {
	l.styles = scheme
}

func applyStyle(message string, style lipgloss.Style) string {
	// Should look like fmt.Sprintf("style.Rener(%s))
	styledMessage := style.Render(fmt.Sprint("", message))
	return styledMessage
}

func setColorScheme(scheme ColorScheme) LogStyles {
	styles := LogStyles{}
	styles.Debug = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Debug))
	styles.Info = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Info))
	styles.Warn = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Warn))
	styles.Error = lipgloss.NewStyle().Foreground(lipgloss.Color(scheme.Error))
	styles.Fatal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(scheme.Error)).
		Border(lipgloss.RoundedBorder()).
		Align(lipgloss.Center).
		Bold(true).
		Width(60)
	return styles
}

