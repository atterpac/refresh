package log

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
	Debug string
	Info  string
	Warn  string
	Error string
	Fatal string
}

type StyledLogger struct {
	styles      LogStyles
	loggingLevel int
}

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func NewStyledLogger(styles LogStyles, level int) *StyledLogger {
	return &StyledLogger{styles, level}
}


func (l *StyledLogger) Debug(format string, args ...interface{}) {
	if l.loggingLevel <= DebugLevel {
		l.log(l.styles.Debug, format, args...)
	}
}

func (l *StyledLogger) Info(format string, args ...interface{}) {
	if l.loggingLevel <= InfoLevel {
		l.log(l.styles.Info, format, args...)
	}
}

func (l *StyledLogger) Warn(format string, args ...interface{}) {
	if l.loggingLevel <= WarnLevel {
		l.log(l.styles.Warn, format, args...)
	}
}

func (l *StyledLogger) Error(format string, args ...interface{}) {
	if l.loggingLevel <= ErrorLevel {
		l.log(l.styles.Error, format, args...)
	}
}

func (l *StyledLogger) Fatal(format string, args ...interface{}) {
	if l.loggingLevel <= FatalLevel {
		l.log(l.styles.Fatal, format, args...)
	}
}

func (l *StyledLogger) log(style lipgloss.Style, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	styledMessage := applyStyle(message, style)
	fmt.Println(styledMessage)
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


func CreateStyles(debug string, info string, warn string, err string, fatal string) LogStyles {
	return LogStyles{
		Debug: lipgloss.NewStyle().Foreground(lipgloss.Color(debug)),
		Info:  lipgloss.NewStyle().Foreground(lipgloss.Color(info)),
		Warn:  lipgloss.NewStyle().Foreground(lipgloss.Color(warn)),
		Error: lipgloss.NewStyle().Foreground(lipgloss.Color(err)),
		Fatal: lipgloss.NewStyle().Foreground(lipgloss.Color(fatal)),
	}
}
