package tui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// sparkChars are the 8-level block characters used to build sparklines.
var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline renders values as a single-line sparkline string.
// Use math.NaN() for missing/null values — those render as a dim dash.
// The sparkline is coloured with the given lipgloss color.
func RenderSparkline(values []float64, color lipgloss.Color) string {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		if !math.IsNaN(v) {
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
	}

	barStyle := lipgloss.NewStyle().Foreground(color)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	n := len(sparkChars)

	var sb strings.Builder
	for _, v := range values {
		if math.IsNaN(v) {
			sb.WriteString(dimStyle.Render("╌"))
			continue
		}
		idx := 0
		if hi > lo {
			idx = int(math.Round((v - lo) / (hi - lo) * float64(n-1)))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		sb.WriteString(barStyle.Render(string(sparkChars[idx])))
	}
	return sb.String()
}

// SparklineTrend returns (delta, direction) where direction is 1 (up), -1 (down), 0 (flat).
// It compares the last non-NaN value to the first non-NaN value.
func SparklineTrend(values []float64) (delta float64, dir int) {
	var first, last float64
	hasFirst := false
	for _, v := range values {
		if math.IsNaN(v) {
			continue
		}
		if !hasFirst {
			first = v
			hasFirst = true
		}
		last = v
	}
	if !hasFirst {
		return 0, 0
	}
	delta = last - first
	switch {
	case delta > 0.05:
		return delta, 1
	case delta < -0.05:
		return delta, -1
	default:
		return delta, 0
	}
}

// TrendArrow returns a styled arrow string for a trend direction.
// goodUp: true  → up is green, down is red
// goodUp: false → down is green, up is red (e.g. fatigue, resting HR)
// goodUp: nil   → always dim (neutral metric)
func TrendArrow(dir int, goodUp *bool) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#66BB6A"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	switch dir {
	case 1:
		if goodUp == nil {
			return dim.Render("↑")
		}
		if *goodUp {
			return green.Render("↑")
		}
		return red.Render("↑")
	case -1:
		if goodUp == nil {
			return dim.Render("↓")
		}
		if *goodUp {
			return red.Render("↓")
		}
		return green.Render("↓")
	default:
		return dim.Render("→")
	}
}

// boolPtr is a helper for creating *bool literals inline.
func boolPtr(b bool) *bool { return &b }

// SparklineRow renders one dashboard row:
//
//	  Label           current   ▁▂▃▄▅▆▇█  ↑
//
// labelW controls the fixed label column width.
func SparklineRow(label, currentStr string, values []float64, color lipgloss.Color, goodUp *bool, labelW int) string {
	spark := RenderSparkline(values, color)
	_, dir := SparklineTrend(values)
	arrow := TrendArrow(dir, goodUp)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	labelPad := label
	if len(labelPad) < labelW {
		labelPad = labelPad + strings.Repeat(" ", labelW-len(labelPad))
	}

	return "  " + labelStyle.Render(labelPad) + "  " +
		dimStyle.Render(currentStr) + "  " +
		spark + "  " + arrow
}

// GoodUp and GoodDown are convenience *bool values for TrendArrow.
var (
	GoodUp   = boolPtr(true)
	GoodDown = boolPtr(false)
)
