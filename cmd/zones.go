package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui"
)

func init() {
	rootCmd.AddCommand(zonesCmd)
}

var zonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "Show power and HR zone definitions",
	Long: `Show power and HR zones for all configured sport types.

In a terminal: renders coloured zone tables with watt and BPM ranges.
Piped or with --output json: outputs the full sport settings as JSON.

Examples:
  icu zones
  icu zones --output json`,
	RunE: runZones,
}

func runZones(cmd *cobra.Command, args []string) error {
	data, err := cli.Get(cli.AthletePath("/sport-settings"))
	if err != nil {
		return err
	}
	var settings []models.SportSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing sport settings: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("zones", cli.AthleteID, nil, settings)
	}

	width := tui.TerminalWidth()
	for _, s := range settings {
		if len(s.PowerZones) == 0 && len(s.HRZones) == 0 {
			continue
		}

		typeLabel := strings.Join(s.Types, ", ")
		if len(s.Types) > 3 {
			typeLabel = strings.Join(s.Types[:3], ", ") + fmt.Sprintf(" +%d", len(s.Types)-3)
		}
		fmt.Println()
		fmt.Println(tui.Header.Render("  " + typeLabel))

		if len(s.PowerZones) > 0 && s.FTP != nil {
			fmt.Println()
			fmt.Printf("  %s\n\n", tui.Bold.Render(fmt.Sprintf("Power Zones  (FTP: %.0fw)", *s.FTP)))
			fmt.Println(renderPowerZoneTable(s, width))
		}

		if len(s.HRZones) > 0 {
			var headerParts []string
			if s.LTHR != nil {
				headerParts = append(headerParts, fmt.Sprintf("LTHR: %.0f bpm", *s.LTHR))
			}
			if s.MaxHR != nil {
				headerParts = append(headerParts, fmt.Sprintf("Max HR: %.0f bpm", *s.MaxHR))
			}
			hrHeader := "HR Zones"
			if len(headerParts) > 0 {
				hrHeader += "  (" + strings.Join(headerParts, "  •  ") + ")"
			}
			fmt.Println()
			fmt.Printf("  %s\n\n", tui.Bold.Render(hrHeader))
			fmt.Println(renderHRZoneTable(s, width))
		}
	}

	fmt.Println()
	return nil
}

// renderPowerZoneTable builds a lipgloss table of power zones with coloured zone names.
// PowerZones entries are upper boundaries as % of FTP; the last entry (999) means "no upper limit".
func renderPowerZoneTable(s models.SportSettings, width int) string {
	if s.FTP == nil || len(s.PowerZones) == 0 {
		return ""
	}
	ftp := *s.FTP
	t := tui.NewTable("ZONE", "% FTP", "WATTS").Width(width - 2)

	for i, upperPct := range s.PowerZones {
		var lowerPct float64
		if i > 0 {
			lowerPct = s.PowerZones[i-1]
		}
		lowerW := int(lowerPct * ftp / 100)

		zoneName := zoneLabel(i, s.PowerZoneNames)

		var pctRange, wattRange string
		if upperPct >= 999 {
			pctRange = fmt.Sprintf("%3.0f%%+", lowerPct)
			wattRange = fmt.Sprintf("> %d w", lowerW)
		} else {
			upperW := int(upperPct * ftp / 100)
			pctRange = fmt.Sprintf("%3.0f – %3.0f%%", lowerPct, upperPct)
			wattRange = fmt.Sprintf("%d – %d w", lowerW, upperW)
		}

		t.Row(zoneStyled(i, zoneName), pctRange, wattRange)
	}
	return t.Render()
}

// renderHRZoneTable builds a lipgloss table of HR zones with coloured zone names.
// HRZones entries are upper boundaries in BPM.
func renderHRZoneTable(s models.SportSettings, width int) string {
	if len(s.HRZones) == 0 {
		return ""
	}
	t := tui.NewTable("ZONE", "BPM").Width(width - 2)

	for i, upperBPM := range s.HRZones {
		var lowerBPM float64
		if i > 0 {
			lowerBPM = s.HRZones[i-1]
		}
		zoneName := zoneLabel(i, s.HRZoneNames)
		t.Row(zoneStyled(i, zoneName), fmt.Sprintf("%.0f – %.0f bpm", lowerBPM, upperBPM))
	}
	return t.Render()
}

// zoneLabel returns "Z{n+1}  {name}" or just "Z{n+1}" when no name list is provided.
func zoneLabel(i int, names []string) string {
	if i < len(names) {
		return fmt.Sprintf("Z%d  %s", i+1, names[i])
	}
	return fmt.Sprintf("Z%d", i+1)
}

// zoneStyled applies the zone colour (from tui.ZoneColor) and bold to a zone name string.
func zoneStyled(i int, name string) string {
	idx := i
	if idx >= len(tui.ZoneColor) {
		idx = len(tui.ZoneColor) - 1
	}
	return lipgloss.NewStyle().Foreground(tui.ZoneColor[idx]).Bold(true).Render(name)
}
