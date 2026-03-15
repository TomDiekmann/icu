package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Chart line colors.
var (
	ChartCTLColor = lipgloss.Color("#42A5F5") // blue  — CTL / fitness
	ChartATLColor = lipgloss.Color("#EF5350") // red   — ATL / fatigue
	ChartTSBColor = lipgloss.Color("#66BB6A") // green — TSB / form
)

// RenderFitnessChart draws an ASCII line chart of CTL, ATL, and TSB over time.
// All three series share the same Y axis so they can be compared directly.
// dates is a []string of "YYYY-MM-DD" values aligned with the value slices.
// width is the total available terminal width (including Y-axis column).
func RenderFitnessChart(ctl, atl, tsb []float64, dates []string, width int) string {
	const chartH = 16
	const yAxisW = 9 // e.g. "  123.4 ┤"

	chartW := width - yAxisW
	if chartW < 20 {
		chartW = 20
	}

	// Scale all three series to chartW columns via linear interpolation.
	ctlS := scaleToWidth(ctl, chartW)
	atlS := scaleToWidth(atl, chartW)
	tsbS := scaleToWidth(tsb, chartW)

	// Compute global Y bounds across all series.
	vMin, vMax := math.Inf(1), math.Inf(-1)
	for _, s := range [][]float64{ctlS, atlS, tsbS} {
		for _, v := range s {
			if v < vMin {
				vMin = v
			}
			if v > vMax {
				vMax = v
			}
		}
	}
	// Add a small margin so lines don't hug the edges.
	margin := (vMax - vMin) * 0.06
	if margin < 1 {
		margin = 1
	}
	vMin -= margin
	vMax += margin

	// Grid cell.
	type cell struct {
		ch    rune
		color lipgloss.Color
		prio  int
	}
	grid := make([][]cell, chartH)
	for i := range grid {
		grid[i] = make([]cell, chartW)
	}

	// set writes ch at (row, col) only when prio >= existing priority.
	set := func(row, col int, ch rune, color lipgloss.Color, prio int) {
		if row < 0 || row >= chartH || col < 0 || col >= chartW {
			return
		}
		if prio >= grid[row][col].prio {
			grid[row][col] = cell{ch, color, prio}
		}
	}

	// toRow maps a data value to a grid row (row 0 = top = highest value).
	toRow := func(v float64) int {
		if vMax == vMin {
			return chartH / 2
		}
		norm := (v - vMin) / (vMax - vMin)
		r := int(math.Round(float64(chartH-1) * (1 - norm)))
		if r < 0 {
			r = 0
		}
		if r >= chartH {
			r = chartH - 1
		}
		return r
	}

	// drawSeries plots one data series onto the grid.
	// Vertical transitions are drawn entirely in column x (the "from" column),
	// which means:
	//   • row r0 gets a corner character pointing toward r1
	//   • intermediate rows get │
	//   • row r1 in column x gets the landing corner (╰ or ╭)
	//   • column x+1's own marker (─) is placed in the next iteration
	// This produces clean ╰─ and ╮ connections with no overwrite conflicts.
	drawSeries := func(values []float64, color lipgloss.Color, prio int) {
		rows := make([]int, len(values))
		for x, v := range values {
			rows[x] = toRow(v)
		}
		// Step 1: mark every value position with a horizontal dash.
		for x, r := range rows {
			set(r, x, '─', color, prio)
		}
		// Step 2: draw verticals and corners for each consecutive pair.
		for x := 0; x < len(rows)-1; x++ {
			r0, r1 := rows[x], rows[x+1]
			if r0 == r1 {
				continue
			}
			if r1 > r0 { // next value lower (row index increases = going down visually)
				set(r0, x, '╮', color, prio)
				for r := r0 + 1; r < r1; r++ {
					set(r, x, '│', color, prio)
				}
				set(r1, x, '╰', color, prio)
			} else { // next value higher (row index decreases = going up visually)
				set(r0, x, '╯', color, prio)
				for r := r1 + 1; r < r0; r++ {
					set(r, x, '│', color, prio)
				}
				set(r1, x, '╭', color, prio)
			}
		}
	}

	// Draw in ascending priority: TSB < ATL < CTL (CTL wins on conflict).
	drawSeries(tsbS, ChartTSBColor, 1)
	drawSeries(atlS, ChartATLColor, 2)
	drawSeries(ctlS, ChartCTLColor, 3)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var sb strings.Builder

	// Render chart rows with Y-axis labels on the left.
	for row := 0; row < chartH; row++ {
		rowValue := vMax - float64(row)/float64(chartH-1)*(vMax-vMin)

		// Show a label at top, middle, and bottom rows; blanks elsewhere.
		if row == 0 || row == chartH/2 || row == chartH-1 {
			sb.WriteString(dimStyle.Render(fmt.Sprintf(" %6.1f ┤", rowValue)))
		} else {
			sb.WriteString(dimStyle.Render("         │"))
		}

		// Grid cells.
		for col := 0; col < chartW; col++ {
			c := grid[row][col]
			if c.ch == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(c.color).Render(string(c.ch)))
			}
		}
		sb.WriteRune('\n')
	}

	// X-axis baseline.
	sb.WriteString(dimStyle.Render(strings.Repeat(" ", yAxisW) + "└" + strings.Repeat("─", chartW)))
	sb.WriteRune('\n')

	// X-axis date labels.
	n := len(dates)
	if n > 0 {
		labelLine := []byte(strings.Repeat(" ", chartW))

		tickEvery := 7
		if n > 90 {
			tickEvery = 30
		} else if n > 45 {
			tickEvery = 14
		}

		lastEnd := -1
		for i := 0; i < n; i += tickEvery {
			col := 0
			if n > 1 {
				col = int(math.Round(float64(i) * float64(chartW-1) / float64(n-1)))
			}
			label := fitnessDateLabel(dates[i])
			if col+len(label) > chartW {
				col = chartW - len(label)
			}
			if col <= lastEnd {
				continue
			}
			copy(labelLine[col:], []byte(label))
			lastEnd = col + len(label)
		}
		sb.WriteString(strings.Repeat(" ", yAxisW))
		sb.WriteString(dimStyle.Render(string(labelLine)))
		sb.WriteRune('\n')
	}

	return sb.String()
}

// scaleToWidth maps len(values) data points to targetLen via linear interpolation.
func scaleToWidth(values []float64, targetLen int) []float64 {
	n := len(values)
	if n == 0 || targetLen == 0 {
		return nil
	}
	if n == 1 {
		out := make([]float64, targetLen)
		for i := range out {
			out[i] = values[0]
		}
		return out
	}
	out := make([]float64, targetLen)
	for i := range out {
		src := float64(i) * float64(n-1) / float64(targetLen-1)
		lo := int(src)
		if lo >= n-1 {
			out[i] = values[n-1]
			continue
		}
		frac := src - float64(lo)
		out[i] = values[lo]*(1-frac) + values[lo+1]*frac
	}
	return out
}

// fitnessDateLabel formats "YYYY-MM-DD" as "Jan 02".
func fitnessDateLabel(date string) string {
	if len(date) < 10 {
		return date
	}
	months := [...]string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var month int
	fmt.Sscanf(date[5:7], "%d", &month)
	if month < 1 || month > 12 {
		return date[:7]
	}
	return months[month] + " " + date[8:10]
}
