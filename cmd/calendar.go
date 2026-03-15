package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui/calendar"
)

var calMonth string

func init() {
	rootCmd.AddCommand(calendarCmd)
	calendarCmd.Flags().StringVar(&calMonth, "month", "", "month to display: YYYY-MM (default: current month)")
}

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Month view combining activities and planned events",
	Long: `Show a monthly calendar with completed activities and planned events.

In a terminal: interactive month grid — navigate days, weeks, and months.
Press Enter on any day to see its activities and events in detail.
Piped or with --output json: outputs a JSON array of day entries for the month.

Key bindings (grid):
  ←/→  h/l     navigate days (crosses month boundary)
  ↑/↓  k/j     navigate weeks (clamped to month)
  [/]           previous / next month
  Enter         open day detail
  q             quit

Key bindings (day detail):
  ↑/↓  ←/→     scroll
  Esc / q       back to grid

Examples:
  icu calendar
  icu calendar --month 2026-02
  icu calendar --output json
  icu calendar --month 2026-01 --output json`,
	RunE: runCalendar,
}

func runCalendar(cmd *cobra.Command, args []string) error {
	// Resolve the display month.
	var year int
	var month time.Month
	if calMonth != "" {
		t, err := time.Parse("2006-01", calMonth)
		if err != nil {
			return fmt.Errorf("invalid month %q: use YYYY-MM", calMonth)
		}
		year = t.Year()
		month = t.Month()
	} else {
		now := time.Now()
		year = now.Year()
		month = now.Month()
	}

	// Fetch a 3-month window (prev, current, next) for smooth navigation.
	// time.Date handles month overflow/underflow correctly.
	oldest := time.Date(year, month-1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	newest := time.Date(year, month+2, 0, 0, 0, 0, 0, time.Local).Format("2006-01-02")

	// Fetch events.
	eventsRaw, err := cli.Get(cli.AthletePath(
		fmt.Sprintf("/events?oldest=%s&newest=%s", oldest, newest)))
	if err != nil {
		return err
	}
	var events []models.Event
	if err := json.Unmarshal(eventsRaw, &events); err != nil {
		return fmt.Errorf("parsing events: %w", err)
	}

	// Fetch activities.
	activitiesRaw, err := cli.Get(cli.AthletePath(
		fmt.Sprintf("/activities?oldest=%s&newest=%s", oldest, newest)))
	if err != nil {
		return err
	}
	var activities []models.Activity
	if err := json.Unmarshal(activitiesRaw, &activities); err != nil {
		return fmt.Errorf("parsing activities: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return calendarAgentOutput(year, month, activities, events)
	}

	// Build the CalEntry map for the bubbletea model.
	entries := make(map[string]calendar.CalEntry)

	for _, a := range activities {
		date := dateOnly(a.StartDateLocal)
		e := entries[date]
		e.Activities = append(e.Activities, a)
		entries[date] = e
	}
	for _, ev := range events {
		date := dateOnly(ev.StartDateLocal)
		e := entries[date]
		e.Events = append(e.Events, ev)
		entries[date] = e
	}

	m := calendar.New(entries, year, month, oldest, newest)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// calendarAgentOutput emits a JSON array of day entries for the requested month.
func calendarAgentOutput(year int, month time.Month, activities []models.Activity, events []models.Event) error {
	monthOldest := time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	monthNewest := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Format("2006-01-02")

	type CalDay struct {
		Date       string             `json:"date"`
		Activities []models.Activity  `json:"activities"`
		Events     []models.Event     `json:"events"`
	}

	dayMap := make(map[string]*CalDay)

	for _, a := range activities {
		date := dateOnly(a.StartDateLocal)
		if date < monthOldest || date > monthNewest {
			continue
		}
		if dayMap[date] == nil {
			dayMap[date] = &CalDay{Date: date}
		}
		dayMap[date].Activities = append(dayMap[date].Activities, a)
	}

	for _, e := range events {
		date := dateOnly(e.StartDateLocal)
		if date < monthOldest || date > monthNewest {
			continue
		}
		if dayMap[date] == nil {
			dayMap[date] = &CalDay{Date: date}
		}
		dayMap[date].Events = append(dayMap[date].Events, e)
	}

	// Build a chronologically ordered slice covering all days in the month.
	totalDays := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
	var calDays []CalDay
	for d := 1; d <= totalDays; d++ {
		date := fmt.Sprintf("%d-%02d-%02d", year, int(month), d)
		if cd := dayMap[date]; cd != nil {
			calDays = append(calDays, *cd)
		}
	}

	return output.PrintJSON("calendar", cli.AthleteID, map[string]string{
		"oldest": monthOldest,
		"newest": monthNewest,
	}, calDays)
}

// dateOnly returns the YYYY-MM-DD portion of a date(-time) string.
func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
