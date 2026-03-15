package activities

import (
	"fmt"
	"strings"

	"github.com/tomdiekmann/icu/internal/format"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/tui"
)

// RenderDetail returns a fully-rendered string for an activity detail view.
// Used by both the bubbletea viewport and the static `icu activities show` output.
func RenderDetail(a models.Activity, intervals []models.Interval, width int) string {
	var sb strings.Builder

	// Header
	sportBadge := tui.SportStyle(a.Type).Render(a.Type)
	sb.WriteString(fmt.Sprintf("\n  %s  %s\n", sportBadge, tui.Bold.Render(a.Name)))
	sb.WriteString(fmt.Sprintf("  %s\n\n", tui.Dim.Render(format.Date(a.StartDateLocal))))

	// Summary grid
	sb.WriteString(tui.Header.Render("  SUMMARY") + "\n\n")
	type stat struct{ label, value string }
	stats := []stat{
		{"Duration", format.Duration(a.MovingTime)},
		{"Distance", format.DistanceKm(a.Distance)},
		{"Elevation", format.ElevationM(a.TotalElevationGain)},
		{"Calories", format.Calories(a.Calories)},
		{"TSS", format.TSS(a.IcuTrainingLoad)},
		{"IF", format.IF(a.IntensityFactor())},
		{"Avg Power", format.Watts(a.IcuAverageWatts)},
		{"Weighted Power", format.Watts(a.IcuWeightedWatts)},
		{"Max Power", format.Watts(a.MaxWatts)},
		{"Avg HR", format.Heartrate(a.AverageHeartrate)},
		{"Max HR", format.Heartrate(a.MaxHeartrate)},
	}
	colWidth := 22
	cols := (width - 4) / colWidth
	if cols < 2 {
		cols = 2
	}
	for i, s := range stats {
		cell := tui.Dim.Render(s.label+":") + " " + tui.Highlight.Render(s.value)
		sb.WriteString(fmt.Sprintf("  %-*s", colWidth, cell))
		if (i+1)%cols == 0 || i == len(stats)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	// Power zone bars
	if pz := zoneTimesToSecs(a.IcuZoneTimes); len(pz) > 0 {
		sb.WriteString(tui.Header.Render("  POWER ZONES") + "\n\n")
		barW := clampBarWidth(width - 36)
		for _, line := range strings.Split(tui.RenderZoneBars(pz, tui.PowerZoneNames, tui.ZoneColor, barW), "\n") {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// HR zone bars
	if hz := zoneTimesToSecs(a.IcuHRZoneTimes); len(hz) > 0 {
		sb.WriteString(tui.Header.Render("  HR ZONES") + "\n\n")
		barW := clampBarWidth(width - 36)
		for _, line := range strings.Split(tui.RenderZoneBars(hz, tui.HRZoneNames, tui.ZoneColor, barW), "\n") {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// Intervals table — WORK intervals only, max 20
	var work []models.Interval
	for _, iv := range intervals {
		if iv.Type == "WORK" {
			work = append(work, iv)
		}
	}
	if len(work) > 20 {
		work = work[:20]
	}
	if len(work) > 0 {
		sb.WriteString(tui.Header.Render("  INTERVALS") + "\n\n")
		t := tui.NewTable("№", "LABEL", "DURATION", "AVG W", "NP", "AVG HR", "IF").Width(width - 4)
		for i, iv := range work {
			label := iv.Label
			if label == "" {
				label = fmt.Sprintf("Interval %d", i+1)
			}
			t.Row(
				fmt.Sprintf("%d", i+1),
				label,
				format.Duration(iv.MovingTime),
				format.Watts(iv.AverageWatts),
				format.Watts(iv.WeightedAverageWatts),
				format.Heartrate(iv.AverageHR),
				format.IF(iv.IntensityFactor()),
			)
		}
		sb.WriteString(t.Render() + "\n")
	}

	return sb.String()
}

// zoneTimesToSecs converts []ZoneTime to []float64, dropping the "SS" meta-zone.
func zoneTimesToSecs(zones []models.ZoneTime) []float64 {
	var out []float64
	for _, z := range zones {
		if z.ID != "SS" {
			out = append(out, z.Secs)
		}
	}
	return out
}

func clampBarWidth(w int) int {
	if w > 20 {
		return 20
	}
	if w < 5 {
		return 5
	}
	return w
}
