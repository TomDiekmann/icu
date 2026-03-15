package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
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
	wellOldest string
	wellNewest string
	wellLast   string
)

var (
	wellUpdWeight    float64
	wellUpdRHR       float64
	wellUpdHRV       float64
	wellUpdSleepSecs int
	wellUpdSteps     int
	wellUpdMood      int
	wellUpdReadiness int
	wellUpdLocked    bool
)

// ── init ─────────────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(wellnessCmd)
	wellnessCmd.AddCommand(wellnessShowCmd)
	wellnessCmd.AddCommand(wellnessListCmd)
	wellnessCmd.AddCommand(wellnessUpdateCmd)

	wellnessListCmd.Flags().StringVar(&wellOldest, "oldest", "", "start date YYYY-MM-DD")
	wellnessListCmd.Flags().StringVar(&wellNewest, "newest", "", "end date YYYY-MM-DD")
	wellnessListCmd.Flags().StringVar(&wellLast, "last", "14d", "fetch wellness for the last duration (e.g. 7d, 4w, 3m)")

	wellnessUpdateCmd.Flags().Float64Var(&wellUpdWeight, "weight", 0, "body weight in kg")
	wellnessUpdateCmd.Flags().Float64Var(&wellUpdRHR, "resting-hr", 0, "resting heart rate (bpm)")
	wellnessUpdateCmd.Flags().Float64Var(&wellUpdHRV, "hrv", 0, "HRV (RMSSD, ms)")
	wellnessUpdateCmd.Flags().IntVar(&wellUpdSleepSecs, "sleep-secs", 0, "sleep duration in seconds")
	wellnessUpdateCmd.Flags().IntVar(&wellUpdSteps, "steps", 0, "step count")
	wellnessUpdateCmd.Flags().IntVar(&wellUpdMood, "mood", 0, "mood score 1–10")
	wellnessUpdateCmd.Flags().IntVar(&wellUpdReadiness, "readiness", 0, "readiness score 1–10")
	wellnessUpdateCmd.Flags().BoolVar(&wellUpdLocked, "locked", false, "lock this wellness entry")
}

// ── commands ──────────────────────────────────────────────────────────────────

var wellnessCmd = &cobra.Command{
	Use:   "wellness",
	Short: "View and update wellness data",
}

var wellnessShowCmd = &cobra.Command{
	Use:   "show [date]",
	Short: "Show wellness for a date (default: today)",
	Long: `Show wellness metrics for a specific date.

In a terminal: renders a styled card with all available fields.
Piped or with --output json: outputs the full wellness JSON.

Examples:
  icu wellness show
  icu wellness show 2026-03-10
  icu wellness show --output json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWellnessShow,
}

var wellnessListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show wellness trends over a date range",
	Long: `Show wellness dashboard with sparkline trends.

In a terminal: renders sparklines for CTL, ATL, weight, RHR, HRV, sleep.
Piped or with --output json: outputs a JSON array of wellness entries.

Examples:
  icu wellness list
  icu wellness list --last 30d
  icu wellness list --oldest 2026-01-01 --newest 2026-03-15 --output json`,
	RunE: runWellnessList,
}

var wellnessUpdateCmd = &cobra.Command{
	Use:   "update [date]",
	Short: "Update wellness metrics for a date (default: today)",
	Long: `Update wellness metrics for a specific date.

Only the flags you pass are sent to the API — unset flags are not modified.

Examples:
  icu wellness update --weight 65.5 --mood 8
  icu wellness update 2026-03-10 --resting-hr 48 --hrv 82
  icu wellness update --sleep-secs 27000 --steps 9500 --output json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWellnessUpdate,
}

// ── show ──────────────────────────────────────────────────────────────────────

func runWellnessShow(cmd *cobra.Command, args []string) error {
	date := time.Now().Format("2006-01-02")
	if len(args) == 1 {
		date = args[0]
	}

	data, err := cli.Get(cli.AthletePath(fmt.Sprintf("/wellness/%s", date)))
	if err != nil {
		return err
	}

	var entry models.WellnessEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("parsing wellness: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("wellness show", cli.AthleteID,
			map[string]string{"date": date}, entry)
	}
	return printWellnessCard(entry)
}

// ── list ──────────────────────────────────────────────────────────────────────

func runWellnessList(cmd *cobra.Command, args []string) error {
	oldest, newest, filters, err := resolveDateRange(wellOldest, wellNewest, wellLast)
	if err != nil {
		return err
	}

	path := cli.AthletePath(fmt.Sprintf("/wellness?oldest=%s&newest=%s", oldest, newest))
	data, err := cli.Get(path)
	if err != nil {
		return err
	}

	var entries []models.WellnessEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing wellness: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("wellness list", cli.AthleteID, filters, entries)
	}
	return printWellnessDashboard(entries, oldest, newest)
}

// ── update ────────────────────────────────────────────────────────────────────

func runWellnessUpdate(cmd *cobra.Command, args []string) error {
	date := time.Now().Format("2006-01-02")
	if len(args) == 1 {
		date = args[0]
	}

	body := map[string]interface{}{}
	if cmd.Flags().Changed("weight") {
		body["weight"] = wellUpdWeight
	}
	if cmd.Flags().Changed("resting-hr") {
		body["restingHR"] = wellUpdRHR
	}
	if cmd.Flags().Changed("hrv") {
		body["hrv"] = wellUpdHRV
	}
	if cmd.Flags().Changed("sleep-secs") {
		body["sleepSecs"] = wellUpdSleepSecs
	}
	if cmd.Flags().Changed("steps") {
		body["steps"] = wellUpdSteps
	}
	if cmd.Flags().Changed("mood") {
		body["mood"] = wellUpdMood
	}
	if cmd.Flags().Changed("readiness") {
		body["readiness"] = wellUpdReadiness
	}
	if cmd.Flags().Changed("locked") {
		body["locked"] = wellUpdLocked
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields specified — use --weight, --resting-hr, --hrv, etc.")
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	respData, err := cli.Put(cli.AthletePath(fmt.Sprintf("/wellness/%s", date)), jsonBody)
	if err != nil {
		return err
	}

	var entry models.WellnessEntry
	if err := json.Unmarshal(respData, &entry); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("wellness update", cli.AthleteID,
			map[string]string{"date": date}, entry)
	}

	fmt.Printf("Updated wellness for %s\n", date)
	return printWellnessCard(entry)
}

// ── human renderers ───────────────────────────────────────────────────────────

func printWellnessCard(w models.WellnessEntry) error {
	today := time.Now().Format("2006-01-02")
	dateLabel := w.ID
	if w.ID == today {
		dateLabel += "  (today)"
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n\n", tui.Bold.Render("WELLNESS"), tui.Dim.Render(dateLabel))

	// ── Fitness ───────────────────────────────────────────────────────────────
	fmt.Println(tui.Header.Render("  FITNESS"))
	fmt.Println()
	printStatRow([]statCell{
		{"CTL (Fitness)", fmt.Sprintf("%.1f", w.CTL)},
		{"ATL (Fatigue)", fmt.Sprintf("%.1f", w.ATL)},
		{"TSB (Form)", fmt.Sprintf("%+.1f", w.TSB())},
		{"Ramp Rate", fmt.Sprintf("%+.1f/wk", w.RampRate)},
	})
	fmt.Println()

	// ── Body ─────────────────────────────────────────────────────────────────
	fmt.Println(tui.Header.Render("  BODY"))
	fmt.Println()
	printStatRow([]statCell{
		{"Weight", formatOptFloat(w.Weight, "%.1f kg")},
		{"Resting HR", formatOptFloat(w.RestingHR, "%.0f bpm")},
		{"HRV", formatOptFloat(w.HRV, "%.1f ms")},
		{"SpO₂", formatOptFloat(w.SpO2, "%.1f%%")},
	})
	printStatRow([]statCell{
		{"Sleep", formatOptSleep(w.SleepSecs)},
		{"Sleep Score", formatOptFloat(w.SleepScore, "%.0f")},
		{"Steps", formatOptInt(w.Steps, "%d")},
		{"Body Fat", formatOptFloat(w.BodyFat, "%.1f%%")},
	})
	fmt.Println()

	// ── Subjective ───────────────────────────────────────────────────────────
	fmt.Println(tui.Header.Render("  SUBJECTIVE"))
	fmt.Println()
	printStatRow([]statCell{
		{"Mood", formatOptInt(w.Mood, "%d/10")},
		{"Readiness", formatOptInt(w.Readiness, "%d/10")},
		{"Fatigue", formatOptInt(w.Fatigue, "%d/10")},
		{"Soreness", formatOptInt(w.Soreness, "%d/10")},
	})
	printStatRow([]statCell{
		{"Stress", formatOptInt(w.Stress, "%d/10")},
		{"Motivation", formatOptInt(w.Motivation, "%d/10")},
		{"Injury", formatOptInt(w.Injury, "%d/10")},
		{},
	})
	fmt.Println()

	if w.Comments != nil && *w.Comments != "" {
		fmt.Printf("  %s  %s\n\n", tui.Dim.Render("Comments:"), *w.Comments)
	}

	return nil
}

type statCell struct{ label, value string }

// printStatRow renders up to 4 stats in a fixed-width grid.
func printStatRow(cells []statCell) {
	const colW = 22
	for i, c := range cells {
		if c.label == "" {
			fmt.Printf("%-*s", colW+2, "")
		} else {
			cell := tui.Dim.Render(c.label+": ") + tui.Highlight.Render(c.value)
			fmt.Printf("  %-*s", colW, cell)
		}
		if (i+1)%4 == 0 || i == len(cells)-1 {
			fmt.Println()
		}
	}
}

// printWellnessDashboard renders the sparkline dashboard for a date range.
func printWellnessDashboard(entries []models.WellnessEntry, oldest, newest string) error {
	if len(entries) == 0 {
		fmt.Println("\n  No wellness data for this period.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s → %s  (%d days)\n\n",
		tui.Bold.Render("WELLNESS DASHBOARD"),
		tui.Dim.Render("•"),
		oldest, newest,
		len(entries),
	)

	const labelW = 18

	// ── Fitness sparklines (always present) ──────────────────────────────────
	fmt.Println(tui.Header.Render("  FITNESS TRENDS"))
	fmt.Println()

	ctlVals := extractFloat64(entries, func(e models.WellnessEntry) float64 { return e.CTL })
	atlVals := extractFloat64(entries, func(e models.WellnessEntry) float64 { return e.ATL })
	tsbVals := extractFloat64(entries, func(e models.WellnessEntry) float64 { return e.TSB() })

	last := entries[len(entries)-1]
	blue := lipgloss.Color("#42A5F5")
	red := lipgloss.Color("#EF5350")
	green := lipgloss.Color("#66BB6A")

	fmt.Println(tui.SparklineRow("CTL (Fitness)", fmt.Sprintf("%.1f", last.CTL),
		ctlVals, blue, tui.GoodUp, labelW))
	fmt.Println(tui.SparklineRow("ATL (Fatigue)", fmt.Sprintf("%.1f", last.ATL),
		atlVals, red, tui.GoodDown, labelW))
	fmt.Println(tui.SparklineRow("Form (TSB)", fmt.Sprintf("%+.1f", last.TSB()),
		tsbVals, green, tui.GoodUp, labelW))
	fmt.Println()

	// ── Personal wellness sparklines (show only if any data exists) ───────────
	type pMetric struct {
		label   string
		vals    []float64
		current string
		color   lipgloss.Color
		goodUp  *bool
	}

	pMetrics := []pMetric{
		{
			label:   "Weight",
			vals:    extractOptFloat64(entries, func(e models.WellnessEntry) *float64 { return e.Weight }),
			current: formatOptFloat(last.Weight, "%.1f kg"),
			color:   lipgloss.Color("#FFA726"),
			goodUp:  nil, // neutral
		},
		{
			label:   "Resting HR",
			vals:    extractOptFloat64(entries, func(e models.WellnessEntry) *float64 { return e.RestingHR }),
			current: formatOptFloat(last.RestingHR, "%.0f bpm"),
			color:   red,
			goodUp:  tui.GoodDown,
		},
		{
			label:   "HRV",
			vals:    extractOptFloat64(entries, func(e models.WellnessEntry) *float64 { return e.HRV }),
			current: formatOptFloat(last.HRV, "%.1f ms"),
			color:   green,
			goodUp:  tui.GoodUp,
		},
		{
			label:   "Sleep",
			vals:    extractSleepFloat(entries),
			current: formatOptSleep(last.SleepSecs),
			color:   blue,
			goodUp:  tui.GoodUp,
		},
		{
			label:   "Steps",
			vals:    extractOptIntAsFloat(entries, func(e models.WellnessEntry) *int { return e.Steps }),
			current: formatOptInt(last.Steps, "%d"),
			color:   lipgloss.Color("#AB47BC"),
			goodUp:  tui.GoodUp,
		},
		{
			label:   "Mood",
			vals:    extractOptIntAsFloat(entries, func(e models.WellnessEntry) *int { return e.Mood }),
			current: formatOptInt(last.Mood, "%d/10"),
			color:   green,
			goodUp:  tui.GoodUp,
		},
		{
			label:   "Readiness",
			vals:    extractOptIntAsFloat(entries, func(e models.WellnessEntry) *int { return e.Readiness }),
			current: formatOptInt(last.Readiness, "%d/10"),
			color:   green,
			goodUp:  tui.GoodUp,
		},
	}

	// Filter to only metrics that have at least one non-NaN value.
	var active []pMetric
	for _, m := range pMetrics {
		if hasAnyData(m.vals) {
			active = append(active, m)
		}
	}

	if len(active) > 0 {
		fmt.Println(tui.Header.Render("  PERSONAL WELLNESS"))
		fmt.Println()
		for _, m := range active {
			fmt.Println(tui.SparklineRow(m.label, m.current, m.vals, m.color, m.goodUp, labelW))
		}
		fmt.Println()
	} else {
		fmt.Println(tui.Dim.Render("  No personal wellness data logged in this period."))
		fmt.Println()
	}

	// Status bar
	status := fmt.Sprintf(" %d entries  •  %s → %s  •  use `icu wellness update` to log data",
		len(entries), oldest, newest)
	fmt.Fprintln(os.Stdout, tui.StatusBarStyle.Render(status))

	return nil
}

// ── formatting helpers ────────────────────────────────────────────────────────

func formatOptFloat(v *float64, fmtStr string) string {
	if v == nil {
		return "--"
	}
	return fmt.Sprintf(fmtStr, *v)
}

func formatOptInt(v *int, fmtStr string) string {
	if v == nil {
		return "--"
	}
	return fmt.Sprintf(fmtStr, *v)
}

func formatOptSleep(secs *int) string {
	if secs == nil || *secs == 0 {
		return "--"
	}
	return format.Duration(*secs)
}

// ── data extraction helpers ───────────────────────────────────────────────────

// extractFloat64 extracts a required float64 field from each entry.
func extractFloat64(entries []models.WellnessEntry, fn func(models.WellnessEntry) float64) []float64 {
	out := make([]float64, len(entries))
	for i, e := range entries {
		out[i] = fn(e)
	}
	return out
}

// extractOptFloat64 extracts an optional *float64 field; missing → NaN.
func extractOptFloat64(entries []models.WellnessEntry, fn func(models.WellnessEntry) *float64) []float64 {
	out := make([]float64, len(entries))
	for i, e := range entries {
		v := fn(e)
		if v == nil {
			out[i] = math.NaN()
		} else {
			out[i] = *v
		}
	}
	return out
}

// extractOptIntAsFloat extracts an optional *int field as float64; missing → NaN.
func extractOptIntAsFloat(entries []models.WellnessEntry, fn func(models.WellnessEntry) *int) []float64 {
	out := make([]float64, len(entries))
	for i, e := range entries {
		v := fn(e)
		if v == nil {
			out[i] = math.NaN()
		} else {
			out[i] = float64(*v)
		}
	}
	return out
}

// extractSleepFloat extracts sleepSecs as float64 hours; missing → NaN.
func extractSleepFloat(entries []models.WellnessEntry) []float64 {
	out := make([]float64, len(entries))
	for i, e := range entries {
		if e.SleepSecs == nil || *e.SleepSecs == 0 {
			out[i] = math.NaN()
		} else {
			out[i] = float64(*e.SleepSecs) / 3600.0
		}
	}
	return out
}

// hasAnyData returns true if at least one value in the slice is not NaN.
func hasAnyData(vals []float64) bool {
	for _, v := range vals {
		if !math.IsNaN(v) {
			return true
		}
	}
	return false
}

