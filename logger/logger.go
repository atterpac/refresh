package logger

import (
	"os"
	"github.com/charmbracelet/log"
)

func init() {
	log := log.NewLogger(os.Stderr)
	return log
}
