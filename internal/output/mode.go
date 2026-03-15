package output

import (
	"os"

	"golang.org/x/term"
)

type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeTable Mode = "table"
	ModeJSON  Mode = "json"
	ModeCSV   Mode = "csv"
)

func IsInteractive(flagMode string) bool {
	switch Mode(flagMode) {
	case ModeTable:
		return true
	case ModeJSON, ModeCSV:
		return false
	}
	// auto: check TTY and env var
	if os.Getenv("ICU_AGENT_MODE") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}
