package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// NewTable returns a pre-styled lipgloss table with the given headers.
// Alternate rows get a subtle background so wide tables stay readable.
func NewTable(headers ...string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(TableBorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return TableHeaderStyle
			}
			if row%2 == 0 {
				return TableAltStyle
			}
			return TableCellStyle
		}).
		Headers(headers...)
}

// TerminalWidth returns the current terminal width, falling back to 80.
func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
