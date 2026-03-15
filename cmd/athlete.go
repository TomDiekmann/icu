package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui"
)

func init() {
	rootCmd.AddCommand(athleteCmd)
}

var athleteCmd = &cobra.Command{
	Use:   "athlete",
	Short: "Show athlete profile",
	Long: `Show the athlete profile.

In a terminal: renders a styled card with profile info and sport settings summary.
Piped or with --output json: outputs the full athlete JSON.

Examples:
  icu athlete
  icu athlete --output json`,
	RunE: runAthlete,
}

func runAthlete(cmd *cobra.Command, args []string) error {
	data, err := cli.Get(cli.AthletePath(""))
	if err != nil {
		return err
	}
	var athlete models.Athlete
	if err := json.Unmarshal(data, &athlete); err != nil {
		return fmt.Errorf("parsing athlete: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("athlete", cli.AthleteID, nil, athlete)
	}

	// Fetch sport settings for the FTP/LTHR summary (best-effort).
	var settings []models.SportSettings
	if sd, serr := cli.Get(cli.AthletePath("/sport-settings")); serr == nil {
		_ = json.Unmarshal(sd, &settings)
	}

	return printAthleteCard(athlete, settings)
}

func printAthleteCard(a models.Athlete, settings []models.SportSettings) error {
	fmt.Println()

	// ── Identity ──────────────────────────────────────────────────────────────
	displayName := strings.TrimSpace(a.Firstname + " " + a.Lastname)
	if displayName == "" {
		displayName = a.Name
	}
	fmt.Printf("  %s  %s\n", tui.Bold.Render(displayName), tui.Dim.Render(a.ID))

	if a.Email != "" {
		fmt.Printf("  %s\n", tui.Dim.Render(a.Email))
	}

	var meta []string
	if parts := locationStr(a); parts != "" {
		meta = append(meta, parts)
	}
	if a.Timezone != "" {
		meta = append(meta, a.Timezone)
	}
	if a.Sex == "M" {
		meta = append(meta, "♂")
	} else if a.Sex == "F" {
		meta = append(meta, "♀")
	}
	if len(meta) > 0 {
		fmt.Printf("  %s\n", tui.Dim.Render(strings.Join(meta, "  •  ")))
	}

	// ── Body metrics ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(tui.Header.Render("  BODY METRICS"))
	fmt.Println()

	weightStr := "--"
	if a.IcuWeight > 0 {
		weightStr = fmt.Sprintf("%.1f kg", a.IcuWeight)
	}
	rhrStr := "--"
	if a.IcuRestingHR != nil {
		rhrStr = fmt.Sprintf("%.0f bpm", *a.IcuRestingHR)
	}
	fmt.Printf("  %-28s  %s\n",
		tui.Dim.Render("Weight: ")+tui.Highlight.Render(weightStr),
		tui.Dim.Render("Resting HR: ")+tui.Highlight.Render(rhrStr),
	)

	// ── Sport settings summary ────────────────────────────────────────────────
	shown := 0
	for _, s := range settings {
		if s.FTP == nil && s.LTHR == nil {
			continue
		}
		if shown == 0 {
			fmt.Println()
			fmt.Println(tui.Header.Render("  SPORT SETTINGS"))
		}
		shown++
		fmt.Println()

		typeLabel := strings.Join(s.Types, ", ")
		if len(s.Types) > 3 {
			typeLabel = strings.Join(s.Types[:3], ", ") + fmt.Sprintf(" +%d", len(s.Types)-3)
		}
		fmt.Printf("  %s\n", tui.Bold.Render(typeLabel))

		var parts []string
		if s.FTP != nil {
			parts = append(parts, fmt.Sprintf("FTP: %.0fw", *s.FTP))
		}
		if s.LTHR != nil {
			parts = append(parts, fmt.Sprintf("LTHR: %.0f bpm", *s.LTHR))
		}
		if s.MaxHR != nil {
			parts = append(parts, fmt.Sprintf("Max HR: %.0f bpm", *s.MaxHR))
		}
		fmt.Printf("  %s\n", tui.Dim.Render(strings.Join(parts, "   •   ")))
	}

	fmt.Println()
	fmt.Printf("  %s\n\n", tui.Dim.Render("Run `icu zones` to see full power and HR zone tables."))
	return nil
}

func locationStr(a models.Athlete) string {
	var parts []string
	if a.City != "" {
		parts = append(parts, a.City)
	}
	if a.State != "" {
		parts = append(parts, a.State)
	}
	if a.Country != "" {
		parts = append(parts, a.Country)
	}
	return strings.Join(parts, ", ")
}
