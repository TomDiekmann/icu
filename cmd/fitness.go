package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui"
)

var (
	cfgFitnessDate  string
	cfgFitnessRange int
)

func init() {
	rootCmd.AddCommand(fitnessCmd)
	fitnessCmd.Flags().StringVar(&cfgFitnessDate, "date", "", "end date YYYY-MM-DD (default: today)")
	fitnessCmd.Flags().IntVar(&cfgFitnessRange, "range", 42, "days of history to fetch and chart")
}

var fitnessCmd = &cobra.Command{
	Use:   "fitness",
	Short: "Show CTL/ATL/TSB fitness chart",
	Long: `Display a fitness chart showing Chronic Training Load (CTL / fitness),
Acute Training Load (ATL / fatigue), and Training Stress Balance (TSB / form).

In a terminal: renders a colored ASCII line chart with a current-values summary.
Piped or with --output json: outputs a JSON array of daily fitness values.

Examples:
  icu fitness
  icu fitness --range 90
  icu fitness --date 2026-03-01 --range 60
  icu fitness --output json`,
	RunE: runFitness,
}

func runFitness(cmd *cobra.Command, args []string) error {
	endDate := cfgFitnessDate
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid date %q: %w", endDate, err)
	}
	start := end.AddDate(0, 0, -(cfgFitnessRange - 1))
	oldest := start.Format("2006-01-02")
	newest := end.Format("2006-01-02")

	path := cli.AthletePath(fmt.Sprintf("/wellness?oldest=%s&newest=%s", oldest, newest))
	data, err := cli.Get(path)
	if err != nil {
		return err
	}

	var entries []models.WellnessEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing wellness data: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no data for range %s to %s", oldest, newest)
	}

	// Build FitnessDay slice from wellness entries.
	fitDays := make([]models.FitnessDay, len(entries))
	for i, e := range entries {
		fitDays[i] = models.FitnessDay{
			Date:     e.ID,
			CTL:      e.CTL,
			ATL:      e.ATL,
			TSB:      e.TSB(),
			RampRate: e.RampRate,
			CTLLoad:  e.CTLLoad,
			ATLLoad:  e.ATLLoad,
		}
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("fitness", cli.AthleteID, map[string]string{
			"oldest": oldest,
			"newest": newest,
		}, fitDays)
	}

	return printFitnessHuman(fitDays)
}

func printFitnessHuman(days []models.FitnessDay) error {
	ctlVals := make([]float64, len(days))
	atlVals := make([]float64, len(days))
	tsbVals := make([]float64, len(days))
	dates := make([]string, len(days))
	for i, d := range days {
		ctlVals[i] = d.CTL
		atlVals[i] = d.ATL
		tsbVals[i] = d.TSB
		dates[i] = d.Date
	}

	width := tui.TerminalWidth()
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ctlStyle := lipgloss.NewStyle().Foreground(tui.ChartCTLColor)
	atlStyle := lipgloss.NewStyle().Foreground(tui.ChartATLColor)
	tsbStyle := lipgloss.NewStyle().Foreground(tui.ChartTSBColor)

	fmt.Println()

	// Legend.
	fmt.Printf("  %s %s    %s %s    %s %s\n",
		ctlStyle.Render("───"), dimStyle.Render("CTL (Fitness)"),
		atlStyle.Render("───"), dimStyle.Render("ATL (Fatigue)"),
		tsbStyle.Render("───"), dimStyle.Render("TSB (Form)"),
	)
	fmt.Println()

	// Chart — indent by 2 spaces.
	chartWidth := width - 4
	if chartWidth < 40 {
		chartWidth = 40
	}
	chart := tui.RenderFitnessChart(ctlVals, atlVals, tsbVals, dates, chartWidth)
	for _, line := range strings.Split(strings.TrimRight(chart, "\n"), "\n") {
		fmt.Println("  " + line)
	}

	// Summary box.
	latest := days[len(days)-1]
	fmt.Println()
	fmt.Println(tui.Header.Render("  CURRENT  " + dimStyle.Render(latest.Date)))
	fmt.Println()

	ctlDelta := ""
	if len(days) > 1 {
		d := latest.CTL - days[0].CTL
		if d >= 0 {
			ctlDelta = fmt.Sprintf("  %s", dimStyle.Render(fmt.Sprintf("+%.1f over %d days", d, len(days)-1)))
		} else {
			ctlDelta = fmt.Sprintf("  %s", dimStyle.Render(fmt.Sprintf("%.1f over %d days", d, len(days)-1)))
		}
	}

	fmt.Printf("  %s  %s%s\n",
		dimStyle.Render("CTL (Fitness):"),
		tui.Highlight.Render(fmt.Sprintf("%.1f", latest.CTL)),
		ctlDelta,
	)
	fmt.Printf("  %s  %s\n",
		dimStyle.Render("ATL (Fatigue):"),
		tui.Highlight.Render(fmt.Sprintf("%.1f", latest.ATL)),
	)
	fmt.Printf("  %s  %s\n",
		dimStyle.Render("TSB (Form):   "),
		tui.Highlight.Render(fmt.Sprintf("%.1f", latest.TSB)),
	)
	if latest.RampRate != 0 {
		fmt.Printf("  %s  %s\n",
			dimStyle.Render("Ramp Rate:    "),
			tui.Highlight.Render(fmt.Sprintf("%.1f CTL/wk", latest.RampRate)),
		)
	}

	formLabel, formColor := models.FormStatus(latest.TSB)
	formStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(formColor)).Bold(true)
	fmt.Println()
	fmt.Printf("  %s  %s\n",
		dimStyle.Render("Form Status:"),
		formStyle.Render(formLabel),
	)
	fmt.Println()

	return nil
}
