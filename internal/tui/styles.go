package tui

import "github.com/charmbracelet/lipgloss"

// Sport colors — consistent across all views.
var SportColor = map[string]lipgloss.Color{
	"Ride":           lipgloss.Color("#FF6B00"),
	"VirtualRide":    lipgloss.Color("#FF8C00"),
	"Run":            lipgloss.Color("#00B4D8"),
	"VirtualRun":     lipgloss.Color("#0096C7"),
	"Swim":           lipgloss.Color("#0077B6"),
	"WeightTraining": lipgloss.Color("#9B5DE5"),
	"Yoga":           lipgloss.Color("#F72585"),
	"Hike":           lipgloss.Color("#52B788"),
	"Walk":           lipgloss.Color("#74C69D"),
	"Alpine Ski":     lipgloss.Color("#ADE8F4"),
	"NordicSki":      lipgloss.Color("#90E0EF"),
	"Rowing":         lipgloss.Color("#48CAE4"),
	"Kayaking":       lipgloss.Color("#023E8A"),
	"Workout":        lipgloss.Color("#B5838D"),
}

// SportColorDefault is used when no specific color is defined.
const SportColorDefault = lipgloss.Color("#888888")

// SportStyle returns a lipgloss style coloured for the given sport.
func SportStyle(sport string) lipgloss.Style {
	color, ok := SportColor[sport]
	if !ok {
		color = SportColorDefault
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true)
}

// Zone colors: Z1=gray … Z7=dark red.
var ZoneColor = []lipgloss.Color{
	lipgloss.Color("#9E9E9E"), // Z1 Recovery
	lipgloss.Color("#42A5F5"), // Z2 Endurance
	lipgloss.Color("#66BB6A"), // Z3 Tempo
	lipgloss.Color("#FFA726"), // Z4 Threshold
	lipgloss.Color("#EF5350"), // Z5 VO2max
	lipgloss.Color("#B71C1C"), // Z6 Anaerobic
	lipgloss.Color("#7B1FA2"), // Z7 Neuromuscular
}

// Common styles.
var (
	Bold      = lipgloss.NewStyle().Bold(true)
	Dim       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	Header    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	Highlight = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))

	// Table border/header styles.
	TableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	TableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).
				Padding(0, 1)
	TableCellStyle = lipgloss.NewStyle().Padding(0, 1)
	TableAltStyle  = lipgloss.NewStyle().Padding(0, 1).
			Background(lipgloss.Color("235"))

	// Status bar at the bottom of list views.
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
)
