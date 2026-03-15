package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tomdiekmann/icu/internal/format"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/output"
	"github.com/tomdiekmann/icu/internal/tui"
	"github.com/tomdiekmann/icu/internal/tui/activities"
)

// flag storage for activities list
var (
	actOldest string
	actNewest string
	actLast   string
	actType   string
)

// flag storage for activities download
var (
	actDownloadOutputDir string
	actDownloadICUFit    bool
)

// flag storage for activities upload
var (
	actUploadName        string
	actUploadDescription string
	actUploadExternalID  string
)

func init() {
	rootCmd.AddCommand(activitiesCmd)
	activitiesCmd.AddCommand(activitiesListCmd)
	activitiesCmd.AddCommand(activitiesShowCmd)
	activitiesCmd.AddCommand(activitiesDownloadCmd)
	activitiesCmd.AddCommand(activitiesUploadCmd)

	activitiesListCmd.Flags().StringVar(&actOldest, "oldest", "", "start date YYYY-MM-DD")
	activitiesListCmd.Flags().StringVar(&actNewest, "newest", "", "end date YYYY-MM-DD")
	activitiesListCmd.Flags().StringVar(&actLast, "last", "7d", "fetch activities for the last duration (e.g. 7d, 4w, 3m, 1y)")
	activitiesListCmd.Flags().StringVar(&actType, "type", "", "filter by sport type (e.g. Ride, Run, Swim)")

	activitiesDownloadCmd.Flags().StringVar(&actDownloadOutputDir, "output-dir", ".", "directory to save the downloaded file")
	activitiesDownloadCmd.Flags().BoolVar(&actDownloadICUFit, "icu-fit", false, "download Intervals.icu FIT file instead of original")

	activitiesUploadCmd.Flags().StringVar(&actUploadName, "name", "", "override activity name")
	activitiesUploadCmd.Flags().StringVar(&actUploadDescription, "description", "", "activity description")
	activitiesUploadCmd.Flags().StringVar(&actUploadExternalID, "external-id", "", "external ID for deduplication")
}

var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Manage and view activities",
}

var activitiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent activities",
	Long: `List activities from Intervals.icu.

In a terminal: renders a colour-coded lipgloss table.
Piped or with --output json: outputs structured JSON with a meta wrapper.

Examples:
  icu activities list
  icu activities list --last 30d
  icu activities list --oldest 2026-01-01 --newest 2026-03-15
  icu activities list --type Ride --output json`,
	RunE: runActivitiesList,
}

var activitiesShowCmd = &cobra.Command{
	Use:   "show <activity-id>",
	Short: "Show activity detail",
	Long: `Show full detail for a single activity.

In a terminal: renders a summary card, zone distribution bars, and intervals table.
Piped or with --output json: outputs the complete activity JSON.

Examples:
  icu activities show i132173665
  icu activities show i132173665 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runActivitiesShow,
}

var activitiesDownloadCmd = &cobra.Command{
	Use:   "download <activity-id>",
	Short: "Download activity file",
	Long: `Download the original activity file (FIT/GPX/TCX) to disk.

Use --icu-fit to download the Intervals.icu processed FIT file instead.

Examples:
  icu activities download i132173665
  icu activities download i132173665 --output-dir ~/Downloads
  icu activities download i132173665 --icu-fit`,
	Args: cobra.ExactArgs(1),
	RunE: runActivitiesDownload,
}

var activitiesUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload an activity file",
	Long: `Upload a FIT/GPX/TCX activity file to Intervals.icu.

Examples:
  icu activities upload morning_ride.fit
  icu activities upload activity.fit --name "Epic Saturday Ride" --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runActivitiesUpload,
}

// ── list ─────────────────────────────────────────────────────────────────────

func runActivitiesList(cmd *cobra.Command, args []string) error {
	// Resolve date range: explicit flags take priority over --last.
	oldest, newest, filters, err := resolveDateRange(actOldest, actNewest, actLast)
	if err != nil {
		return err
	}

	// Build query path.
	path := cli.AthletePath(fmt.Sprintf(
		"/activities?oldest=%s&newest=%s",
		oldest, newest,
	))

	data, err := cli.Get(path)
	if err != nil {
		return err
	}

	var activities_slice []models.Activity
	if err := json.Unmarshal(data, &activities_slice); err != nil {
		return fmt.Errorf("parsing activities: %w", err)
	}

	// Optional sport type filter (the API doesn't support it natively).
	if actType != "" {
		filtered := activities_slice[:0]
		for _, a := range activities_slice {
			if a.Type == actType {
				filtered = append(filtered, a)
			}
		}
		activities_slice = filtered
		filters["type"] = actType
	}

	interactive := output.IsInteractive(cfgOutput)

	if interactive {
		fetchIntervals := func(id string) ([]models.Interval, error) {
			iData, err := cli.Get(fmt.Sprintf("/api/v1/activity/%s/intervals", id))
			if err != nil {
				return nil, err
			}
			var ivResp models.IntervalsResponse
			if err := json.Unmarshal(iData, &ivResp); err != nil {
				return nil, err
			}
			return ivResp.IcuIntervals, nil
		}

		download := func(id string) (string, error) {
			path := fmt.Sprintf("/api/v1/activity/%s/file", id)
			resp, err := cli.DownloadWithResponse(path)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			filename := extractFilename(resp.Header.Get("Content-Disposition"))
			if filename == "" {
				filename = fmt.Sprintf("activity-%s.fit", id)
			}

			outPath := filepath.Join(".", filename)
			f, err := os.Create(outPath)
			if err != nil {
				return "", fmt.Errorf("creating file: %w", err)
			}
			defer f.Close()

			if _, err := io.Copy(f, resp.Body); err != nil {
				return "", fmt.Errorf("writing file: %w", err)
			}
			return outPath, nil
		}

		m := activities.New(activities_slice, oldest, newest, fetchIntervals, download)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}
	return printActivitiesJSON(activities_slice, filters)
}

// ── show ──────────────────────────────────────────────────────────────────────

func runActivitiesShow(cmd *cobra.Command, args []string) error {
	id := args[0]
	path := fmt.Sprintf("/api/v1/activity/%s", id)

	data, err := cli.Get(path)
	if err != nil {
		return err
	}

	var activity models.Activity
	if err := json.Unmarshal(data, &activity); err != nil {
		return fmt.Errorf("parsing activity: %w", err)
	}

	// Fetch intervals (best-effort; skip on error).
	var intervals []models.Interval
	iData, iErr := cli.Get(fmt.Sprintf("/api/v1/activity/%s/intervals", id))
	if iErr == nil {
		var ivResp models.IntervalsResponse
		if json.Unmarshal(iData, &ivResp) == nil {
			intervals = ivResp.IcuIntervals
		}
	}

	interactive := output.IsInteractive(cfgOutput)
	if interactive {
		width := tui.TerminalWidth()
		fmt.Print(activities.RenderDetail(activity, intervals, width))
		status := fmt.Sprintf(" %s  •  %s  •  esc/q to quit", activity.ID, format.Date(activity.StartDateLocal))
		fmt.Fprintln(os.Stdout, tui.StatusBarStyle.Render(status))
		return nil
	}
	return printActivityDetailJSON(activity)
}

// ── download ──────────────────────────────────────────────────────────────────

func runActivitiesDownload(cmd *cobra.Command, args []string) error {
	id := args[0]

	var path string
	if actDownloadICUFit {
		path = fmt.Sprintf("/api/v1/activity/%s/fit-file", id)
	} else {
		path = fmt.Sprintf("/api/v1/activity/%s/file", id)
	}

	resp, err := cli.DownloadWithResponse(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Determine file name from Content-Disposition or fall back to a default.
	filename := extractFilename(resp.Header.Get("Content-Disposition"))
	if filename == "" {
		ext := ".fit"
		if !actDownloadICUFit {
			// Guess extension from Content-Type.
			ct := resp.Header.Get("Content-Type")
			switch {
			case strings.Contains(ct, "gpx"):
				ext = ".gpx"
			case strings.Contains(ct, "tcx"):
				ext = ".tcx"
			}
		}
		filename = fmt.Sprintf("activity-%s%s", id, ext)
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(actDownloadOutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outPath := filepath.Join(actDownloadOutputDir, filename)
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if output.IsInteractive(cfgOutput) {
		fmt.Printf("Downloaded: %s\n", outPath)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]string{"file": outPath, "activity_id": id})
	}
	return nil
}

// extractFilename parses the filename from a Content-Disposition header.
func extractFilename(header string) string {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			name := strings.TrimPrefix(part, "filename=")
			name = strings.Trim(name, `"`)
			return filepath.Base(name)
		}
	}
	return ""
}

// ── upload ────────────────────────────────────────────────────────────────────

func runActivitiesUpload(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}

	extraFields := map[string]string{}
	if actUploadName != "" {
		extraFields["name"] = actUploadName
	}
	if actUploadDescription != "" {
		extraFields["description"] = actUploadDescription
	}
	if actUploadExternalID != "" {
		extraFields["external_id"] = actUploadExternalID
	}

	path := cli.AthletePath("/activities")
	data, err := cli.Upload(path, filePath, extraFields)
	if err != nil {
		return err
	}

	if output.IsInteractive(cfgOutput) {
		// Try to extract name from response for a friendly message.
		var result map[string]interface{}
		if json.Unmarshal(data, &result) == nil {
			if name, ok := result["name"].(string); ok && name != "" {
				fmt.Printf("Uploaded: %s\n", name)
				return nil
			}
		}
		fmt.Println("Activity uploaded successfully.")
	} else {
		fmt.Println(string(data))
	}
	return nil
}

// ── human renderers ───────────────────────────────────────────────────────────

// resolveDateRange returns oldest/newest date strings and a filter map for the
// meta wrapper.  Explicit --oldest/--newest flags override --last.
func resolveDateRange(oldest, newest, last string) (string, string, map[string]string, error) {
	filters := map[string]string{}

	if oldest != "" || newest != "" {
		// explicit range
		today := time.Now()
		if oldest == "" {
			oldest = today.AddDate(0, 0, -7).Format("2006-01-02")
		}
		if newest == "" {
			newest = today.Format("2006-01-02")
		}
		filters["oldest"] = oldest
		filters["newest"] = newest
		return oldest, newest, filters, nil
	}

	// --last duration
	o, n, err := format.ParseLast(last, time.Now())
	if err != nil {
		return "", "", nil, fmt.Errorf("--last: %w", err)
	}
	filters["oldest"] = o
	filters["newest"] = n
	filters["last"] = last
	return o, n, filters, nil
}

// printActivitiesJSON outputs structured JSON with a meta wrapper.
func printActivitiesJSON(activities []models.Activity, filters map[string]string) error {
	return output.PrintJSON(
		"activities list",
		cli.AthleteID,
		filters,
		activities,
	)
}

// printActivityDetailJSON outputs the full activity as structured JSON.
func printActivityDetailJSON(a models.Activity) error {
	return output.PrintJSON(
		"activities show",
		cli.AthleteID,
		map[string]string{"id": a.ID},
		a,
	)
}

