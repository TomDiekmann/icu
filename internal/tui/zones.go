package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PowerZoneNames are the default display names for 7 power zones.
var PowerZoneNames = []string{
	"Z1 Recovery",
	"Z2 Endurance",
	"Z3 Tempo",
	"Z4 Threshold",
	"Z5 VO2max",
	"Z6 Anaerobic",
	"Z7 Neuromusc.",
}

// HRZoneNames are the default display names for 5 HR zones.
var HRZoneNames = []string{
	"Z1 Recovery",
	"Z2 Endurance",
	"Z3 Tempo",
	"Z4 Threshold",
	"Z5 VO2max",
}

// RenderZoneBars renders a coloured horizontal bar chart for zone time distribution.
// zoneTimes is a slice of seconds per zone. barWidth controls the filled-bar width in chars.
func RenderZoneBars(zoneTimes []float64, zoneNames []string, colors []lipgloss.Color, barWidth int) string {
	if len(zoneTimes) == 0 {
		return ""
	}

	var total float64
	for _, t := range zoneTimes {
		total += t
	}
	if total <= 0 {
		return ""
	}

	if barWidth < 5 {
		barWidth = 5
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	const labelWidth = 14

	var sb strings.Builder
	for i, t := range zoneTimes {
		if i >= len(zoneNames) || i >= len(colors) {
			break
		}

		pct := t / total
		filled := int(float64(barWidth) * pct)
		if filled > barWidth {
			filled = barWidth
		}
		empty := barWidth - filled

		colorStyle := lipgloss.NewStyle().Foreground(colors[i])
		label := fmt.Sprintf("%-*s", labelWidth, zoneNames[i])
		bar := colorStyle.Render(strings.Repeat("█", filled)) +
			dimStyle.Render(strings.Repeat("░", empty))
		pctStr := fmt.Sprintf("%3.0f%%", pct*100)
		timeStr := formatZoneTime(int(t))

		sb.WriteString(fmt.Sprintf("%s  %s  %s  %s\n", label, bar, pctStr, timeStr))
	}

	return strings.TrimRight(sb.String(), "\n")
}

func formatZoneTime(seconds int) string {
	if seconds <= 0 {
		return "--:--"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
