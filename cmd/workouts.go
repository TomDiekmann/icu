package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/format"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui"
)

// ── flag storage ──────────────────────────────────────────────────────────────

var (
	wktOldest string
	wktNewest string
	wktLast   string
)

// ── init ─────────────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(workoutsCmd)
	workoutsCmd.AddCommand(workoutsListCmd)
	workoutsCmd.AddCommand(workoutsShowCmd)

	workoutsListCmd.Flags().StringVar(&wktOldest, "oldest", "", "start date YYYY-MM-DD (default: today)")
	workoutsListCmd.Flags().StringVar(&wktNewest, "newest", "", "end date YYYY-MM-DD (default: today+14d)")
	workoutsListCmd.Flags().StringVar(&wktLast, "last", "", "last duration (e.g. 7d, 4w); overrides --oldest/--newest")
}

// ── commands ──────────────────────────────────────────────────────────────────

var workoutsCmd = &cobra.Command{
	Use:   "workouts",
	Short: "View planned workouts",
}

var workoutsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List upcoming (or recent) planned workouts",
	Long: `List planned workouts from the calendar.

By default shows the next 14 days. Use --last or --oldest/--newest for other ranges.

In a terminal: renders a styled table with workout names and steps preview.
Piped or with --output json: outputs a JSON array of workout events.

Examples:
  icu workouts list
  icu workouts list --last 30d
  icu workouts list --oldest 2026-03-01 --newest 2026-03-31 --output json`,
	RunE: runWorkoutsList,
}

var workoutsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a workout in detail",
	Long: `Show full details for a single workout event by ID.

In a terminal: renders a styled card with description and workout steps.
Piped or with --output json: outputs the full event JSON.

Examples:
  icu workouts show 123456
  icu workouts show 123456 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkoutsShow,
}

// ── list ─────────────────────────────────────────────────────────────────────

func runWorkoutsList(cmd *cobra.Command, args []string) error {
	var oldest, newest string
	var filters map[string]string

	if wktLast != "" {
		// --last overrides date flags
		var err error
		oldest, newest, filters, err = resolveDateRange("", "", wktLast)
		if err != nil {
			return err
		}
	} else if wktOldest != "" || wktNewest != "" {
		var err error
		oldest, newest, filters, err = resolveDateRange(wktOldest, wktNewest, "")
		if err != nil {
			return err
		}
	} else {
		// Default: today → +14 days
		today := time.Now()
		oldest = today.Format("2006-01-02")
		newest = today.AddDate(0, 0, 14).Format("2006-01-02")
		filters = map[string]string{"oldest": oldest, "newest": newest}
	}

	path := cli.AthletePath(fmt.Sprintf("/events?oldest=%s&newest=%s", oldest, newest))
	data, err := cli.Get(path)
	if err != nil {
		return err
	}

	var events []models.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return fmt.Errorf("parsing events: %w", err)
	}

	// Filter to WORKOUT category only.
	var workouts []models.Event
	for _, e := range events {
		if strings.ToUpper(e.Category) == "WORKOUT" {
			workouts = append(workouts, e)
		}
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("workouts list", cli.AthleteID, filters, workouts)
	}

	return printWorkoutsTable(workouts, oldest, newest)
}

// ── show ─────────────────────────────────────────────────────────────────────

func runWorkoutsShow(cmd *cobra.Command, args []string) error {
	id := args[0]
	data, err := cli.Get(cli.AthletePath(fmt.Sprintf("/events/%s", id)))
	if err != nil {
		return err
	}

	var ev models.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("workouts show", cli.AthleteID, map[string]string{"id": id}, ev)
	}

	return printWorkoutCard(ev)
}

// ── human renderers ───────────────────────────────────────────────────────────

func printWorkoutsTable(workouts []models.Event, oldest, newest string) error {
	if len(workouts) == 0 {
		fmt.Printf("\n  No workouts scheduled for %s → %s.\n\n", oldest, newest)
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s → %s  (%d workouts)\n\n",
		tui.Bold.Render("WORKOUTS"),
		tui.Dim.Render("•"),
		oldest, newest,
		len(workouts),
	)

	width := tui.TerminalWidth()
	t := tui.NewTable("DATE", "SPORT", "NAME", "DURATION", "LOAD", "STEPS PREVIEW").Width(width - 2)

	for _, w := range workouts {
		dur := "--"
		if w.Duration != nil && *w.Duration > 0 {
			dur = format.Duration(*w.Duration)
		}
		load := "--"
		if w.LoadTarget != nil && *w.LoadTarget > 0 {
			load = fmt.Sprintf("%.0f", *w.LoadTarget)
		}
		name := w.Name
		if len(name) > 30 {
			name = name[:28] + "…"
		}
		preview := workoutDocPreview(w.WorkoutDoc, 35)
		t.Row(
			format.Date(w.StartDateLocal),
			tui.SportStyle(w.Type).Render(w.Type),
			name,
			dur,
			load,
			preview,
		)
	}

	fmt.Println(t.Render())
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fmt.Printf("\n  %s\n\n", dimStyle.Render("Run `icu workouts show <id>` to see full workout steps."))
	return nil
}

func printWorkoutCard(e models.Event) error {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boldStyle := lipgloss.NewStyle().Bold(true)

	fmt.Println()

	// Header line.
	sportBadge := ""
	if e.Type != "" {
		sportBadge = "  " + tui.SportStyle(e.Type).Render(e.Type)
	}
	fmt.Printf("  %s%s\n", boldStyle.Render(e.Name), sportBadge)
	fmt.Printf("  %s\n\n", dimStyle.Render(e.StartDateLocal))

	// Meta row.
	var meta []string
	if e.Duration != nil && *e.Duration > 0 {
		meta = append(meta, format.Duration(*e.Duration))
	}
	if e.LoadTarget != nil && *e.LoadTarget > 0 {
		meta = append(meta, fmt.Sprintf("%.0f TSS target", *e.LoadTarget))
	}
	if e.Indoor != nil && *e.Indoor {
		meta = append(meta, "Indoor")
	}
	if len(meta) > 0 {
		fmt.Printf("  %s\n\n", dimStyle.Render(strings.Join(meta, "  •  ")))
	}

	// Description.
	if e.Description != "" {
		fmt.Println(tui.Header.Render("  DESCRIPTION"))
		fmt.Println()
		for _, line := range strings.Split(e.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}

	// Workout steps.
	if e.WorkoutDoc != "" {
		fmt.Println(tui.Header.Render("  WORKOUT STEPS"))
		fmt.Println()
		renderWorkoutSteps(e.WorkoutDoc)
		fmt.Println()
	}

	// Footer hint.
	fmt.Printf("  %s\n\n", dimStyle.Render(
		fmt.Sprintf("ID: %d  •  use `icu events update %d` to modify", e.ID, e.ID),
	))
	return nil
}

// renderWorkoutSteps prints workout_doc lines with indentation and step styling.
func renderWorkoutSteps(doc string) {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	stepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA726"))

	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Lines starting with "-" are top-level steps; others are sub-steps (recovery, etc.)
		if strings.HasPrefix(trimmed, "-") {
			// Highlight the effort/intensity part (numbers with %).
			rendered := highlightIntensity(trimmed, stepStyle, accentStyle)
			fmt.Printf("  %s\n", rendered)
		} else {
			// Indented sub-step (e.g. recovery interval in a repeat).
			fmt.Printf("      %s\n", dimStyle.Render(trimmed))
		}
	}
}

// highlightIntensity renders a workout step with intensity values (e.g. 88-93%)
// in accent color and the rest in normal style.
func highlightIntensity(line string, base, accent lipgloss.Style) string {
	// Split on spaces and re-join, colorizing tokens that look like intensities.
	parts := strings.Fields(line)
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(" ")
		}
		if looksLikeIntensity(p) {
			sb.WriteString(accent.Render(p))
		} else {
			sb.WriteString(base.Render(p))
		}
	}
	return sb.String()
}

// looksLikeIntensity returns true for tokens like "88-93%", "105%", "150%".
func looksLikeIntensity(s string) bool {
	if !strings.Contains(s, "%") {
		return false
	}
	// Must contain at least one digit.
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// workoutDocPreview returns the first N runes of the first non-empty workout step.
func workoutDocPreview(doc string, maxLen int) string {
	if doc == "" {
		return "--"
	}
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		runes := []rune(trimmed)
		if len(runes) > maxLen {
			trimmed = string(runes[:maxLen-1]) + "…"
		}
		return dimStyle.Render(trimmed)
	}
	return "--"
}
