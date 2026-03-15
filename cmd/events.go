package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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
	// events list flags
	evtListOldest   string
	evtListNewest   string
	evtListLast     string
	evtListCategory string

	// events create / update flags
	evtDate       string
	evtName       string
	evtDesc       string
	evtSport      string
	evtCategory   string
	evtWorkoutDoc string
	evtIndoor     bool
	evtLoadTarget float64
	evtDuration   string
	evtFromJSON   string
)

// ── init ─────────────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsCreateCmd)
	eventsCmd.AddCommand(eventsUpdateCmd)
	eventsCmd.AddCommand(eventsDeleteCmd)

	// list
	eventsListCmd.Flags().StringVar(&evtListOldest, "oldest", "", "start date YYYY-MM-DD")
	eventsListCmd.Flags().StringVar(&evtListNewest, "newest", "", "end date YYYY-MM-DD")
	eventsListCmd.Flags().StringVar(&evtListLast, "last", "14d", "last duration (e.g. 7d, 4w, 3m)")
	eventsListCmd.Flags().StringVar(&evtListCategory, "category", "", "filter by category: WORKOUT, NOTE, RACE, REST_DAY, etc.")

	// create
	addEventFlags(eventsCreateCmd)
	eventsCreateCmd.Flags().StringVar(&evtFromJSON, "from-json", "", `create from JSON file or stdin ("-"); supports JSONL for batch creation`)

	// update
	addEventFlags(eventsUpdateCmd)
	eventsUpdateCmd.Flags().StringVar(&evtFromJSON, "from-json", "", `update from JSON file or stdin ("-")`)
}

func addEventFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&evtDate, "date", "", "event date YYYY-MM-DD (required for create)")
	cmd.Flags().StringVar(&evtName, "name", "", "event name / title")
	cmd.Flags().StringVar(&evtDesc, "description", "", "event description (markdown supported)")
	cmd.Flags().StringVar(&evtSport, "sport", "", "sport type: Ride, Run, Swim, etc.")
	cmd.Flags().StringVar(&evtCategory, "category", "", "category: WORKOUT, NOTE, RACE, REST_DAY, etc.")
	cmd.Flags().StringVar(&evtWorkoutDoc, "workout-doc", "", "workout steps in Intervals.icu format (use \\n between steps)")
	cmd.Flags().BoolVar(&evtIndoor, "indoor", false, "mark as indoor")
	cmd.Flags().Float64Var(&evtLoadTarget, "load-target", 0, "target TSS / training load")
	cmd.Flags().StringVar(&evtDuration, "duration", "", "target duration: seconds (3600) or Go format (1h30m)")
}

// ── commands ──────────────────────────────────────────────────────────────────

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List, create, update, and delete calendar events",
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Long: `List calendar events for a date range.

In a terminal: renders a styled table grouped by date.
Piped or with --output json: outputs a JSON array of events.

Examples:
  icu events list
  icu events list --last 30d
  icu events list --category WORKOUT
  icu events list --oldest 2026-03-01 --newest 2026-03-31 --output json`,
	RunE: runEventsList,
}

var eventsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a calendar event or workout",
	Long: `Create a calendar event.

Use individual flags to build the event, or --from-json for full control.
--from-json accepts a file path or "-" for stdin. Stdin supports JSONL for
batch creation (one JSON object per line).

Workout doc format:
  "- 15m 55-75%"                    → 15 min at 55-75% FTP
  "- 2x20m 95-105% 5m 55%"         → 2×20min with 5min recovery
  "- 5x5m 105-115% 5m 50%"         → 5×5 VO2max intervals
  Use \n between steps in the flag value.

Examples:
  icu events create --date 2026-03-20 --category WORKOUT --sport Ride \
    --name "Sweet Spot Tuesday" \
    --workout-doc "- 15m 55-75%\n- 3x15m 88-93% 5m 55%\n- 10m 55%"

  icu events create --from-json workout.json --output json

  echo '{"start_date_local":"2026-03-20","category":"WORKOUT","type":"Ride","name":"AI Intervals"}' \
    | icu events create --from-json - --output json

  cat training_plan.jsonl | icu events create --from-json - --output json`,
	RunE: runEventsCreate,
}

var eventsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a calendar event",
	Long: `Update an existing calendar event by ID.

Only the flags you pass are sent — unset flags are not modified (unless --from-json is used).

Examples:
  icu events update 123456 --name "New Name" --load-target 80
  icu events update 123456 --from-json updated.json --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runEventsUpdate,
}

var eventsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a calendar event",
	Long: `Delete a calendar event by ID.

In a terminal: prompts for confirmation before deleting.
Piped or with --output json: deletes immediately, returns JSON result.

Examples:
  icu events delete 123456
  icu events delete 123456 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runEventsDelete,
}

// ── list ─────────────────────────────────────────────────────────────────────

func runEventsList(cmd *cobra.Command, args []string) error {
	oldest, newest, filters, err := resolveDateRange(evtListOldest, evtListNewest, evtListLast)
	if err != nil {
		return err
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

	if evtListCategory != "" {
		cat := strings.ToUpper(evtListCategory)
		filtered := events[:0]
		for _, e := range events {
			if strings.ToUpper(e.Category) == cat {
				filtered = append(filtered, e)
			}
		}
		events = filtered
		filters["category"] = evtListCategory
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("events list", cli.AthleteID, filters, events)
	}

	return printEventsTable(events, oldest, newest)
}

// ── create ───────────────────────────────────────────────────────────────────

func runEventsCreate(cmd *cobra.Command, args []string) error {
	if evtFromJSON != "" {
		return createFromJSON(evtFromJSON)
	}

	// Build body from flags.
	if evtDate == "" {
		return fmt.Errorf("--date is required (YYYY-MM-DD)")
	}
	body, err := buildEventBody(cmd, evtDate)
	if err != nil {
		return err
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	created, err := postEvent(jsonBody)
	if err != nil {
		return err
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("events create", cli.AthleteID, nil, created)
	}
	fmt.Println()
	fmt.Printf("  %s  Created event %d\n\n", tui.Header.Render("✓"), created.ID)
	printEventCard(created)
	return nil
}

// createFromJSON handles --from-json file/stdin, supporting both single JSON
// objects and JSONL streams (one object per line) for batch creation.
func createFromJSON(src string) error {
	var r io.Reader
	if src == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("opening %s: %w", src, err)
		}
		defer f.Close()
		r = f
	}

	// Decode potentially multiple JSON objects from the stream.
	dec := json.NewDecoder(r)
	var bodies []json.RawMessage
	for {
		var obj json.RawMessage
		if err := dec.Decode(&obj); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("decoding JSON: %w", err)
		}
		bodies = append(bodies, obj)
	}
	if len(bodies) == 0 {
		return fmt.Errorf("no JSON objects found in input")
	}

	interactive := output.IsInteractive(cfgOutput)
	var created []models.Event

	for _, b := range bodies {
		ev, err := postEvent(b)
		if err != nil {
			return err
		}
		created = append(created, ev)

		if interactive && len(bodies) > 1 {
			fmt.Printf("  Created event %d: %s\n", ev.ID, ev.Name)
		}
	}

	if len(bodies) == 1 {
		// Single event.
		if !interactive {
			return output.PrintJSON("events create", cli.AthleteID, nil, created[0])
		}
		fmt.Println()
		fmt.Printf("  %s  Created event %d\n\n", tui.Header.Render("✓"), created[0].ID)
		printEventCard(created[0])
		return nil
	}

	// Batch: agent mode → JSONL; interactive → summary already printed above.
	if !interactive {
		enc := json.NewEncoder(os.Stdout)
		for _, ev := range created {
			if err := enc.Encode(ev); err != nil {
				return err
			}
		}
		return nil
	}
	fmt.Printf("\n  %s  Created %d events.\n\n", tui.Header.Render("✓"), len(created))
	return nil
}

func postEvent(jsonBody []byte) (models.Event, error) {
	respData, err := cli.Post(cli.AthletePath("/events"), jsonBody)
	if err != nil {
		return models.Event{}, err
	}
	var ev models.Event
	if err := json.Unmarshal(respData, &ev); err != nil {
		return models.Event{}, fmt.Errorf("parsing created event: %w", err)
	}
	return ev, nil
}

// ── update ───────────────────────────────────────────────────────────────────

func runEventsUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	path := cli.AthletePath(fmt.Sprintf("/events/%s", id))

	var jsonBody []byte

	if evtFromJSON != "" {
		var r io.Reader
		if evtFromJSON == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(evtFromJSON)
			if err != nil {
				return fmt.Errorf("opening %s: %w", evtFromJSON, err)
			}
			defer f.Close()
			r = f
		}
		b, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		jsonBody = bytes.TrimSpace(b)
	} else {
		body := map[string]interface{}{}
		if cmd.Flags().Changed("date") {
			body["start_date_local"] = evtDate
		}
		if cmd.Flags().Changed("name") {
			body["name"] = evtName
		}
		if cmd.Flags().Changed("description") {
			body["description"] = evtDesc
		}
		if cmd.Flags().Changed("sport") {
			body["type"] = evtSport
		}
		if cmd.Flags().Changed("category") {
			body["category"] = evtCategory
		}
		if cmd.Flags().Changed("workout-doc") {
			body["workout_doc"] = strings.ReplaceAll(evtWorkoutDoc, `\n`, "\n")
		}
		if cmd.Flags().Changed("indoor") {
			body["indoor"] = evtIndoor
		}
		if cmd.Flags().Changed("load-target") {
			body["load_target"] = evtLoadTarget
		}
		if cmd.Flags().Changed("duration") {
			secs, err := parseDurationSecs(evtDuration)
			if err != nil {
				return err
			}
			body["duration"] = secs
		}
		if len(body) == 0 {
			return fmt.Errorf("no fields specified — use --name, --workout-doc, etc. or --from-json")
		}
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	respData, err := cli.Put(path, jsonBody)
	if err != nil {
		return err
	}

	var ev models.Event
	if err := json.Unmarshal(respData, &ev); err != nil {
		return fmt.Errorf("parsing updated event: %w", err)
	}

	if !output.IsInteractive(cfgOutput) {
		return output.PrintJSON("events update", cli.AthleteID, map[string]string{"id": id}, ev)
	}
	fmt.Println()
	fmt.Printf("  %s  Updated event %s\n\n", tui.Header.Render("✓"), id)
	printEventCard(ev)
	return nil
}

// ── delete ───────────────────────────────────────────────────────────────────

func runEventsDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	if output.IsInteractive(cfgOutput) {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fmt.Printf("\n  %s  Delete event %s? [y/N] ", dimStyle.Render("!"), id)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		if input != "y" && input != "yes" {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	_, err := cli.Delete(cli.AthletePath(fmt.Sprintf("/events/%s", id)))
	if err != nil {
		return err
	}

	if !output.IsInteractive(cfgOutput) {
		result := map[string]interface{}{"deleted": true, "id": id}
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return nil
	}

	fmt.Printf("\n  %s  Deleted event %s.\n\n", tui.Header.Render("✓"), id)
	return nil
}

// ── human renderers ───────────────────────────────────────────────────────────

func printEventsTable(events []models.Event, oldest, newest string) error {
	if len(events) == 0 {
		fmt.Printf("\n  No events for %s → %s.\n\n", oldest, newest)
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s → %s  (%d events)\n\n",
		tui.Bold.Render("EVENTS"),
		tui.Dim.Render("•"),
		oldest, newest,
		len(events),
	)

	width := tui.TerminalWidth()
	t := tui.NewTable("DATE", "CATEGORY", "SPORT", "NAME", "DURATION", "LOAD").Width(width - 2)
	for _, e := range events {
		dur := "--"
		if e.Duration != nil && *e.Duration > 0 {
			dur = format.Duration(*e.Duration)
		}
		load := "--"
		if e.LoadTarget != nil && *e.LoadTarget > 0 {
			load = fmt.Sprintf("%.0f", *e.LoadTarget)
		}
		name := e.Name
		if len(name) > 40 {
			name = name[:38] + "…"
		}
		t.Row(
			format.Date(e.StartDateLocal),
			eventCategoryStyled(e.Category),
			tui.SportStyle(e.Type).Render(e.Type),
			name,
			dur,
			load,
		)
	}
	fmt.Println(t.Render())
	fmt.Println()
	return nil
}

func printEventCard(e models.Event) {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	sportBadge := ""
	if e.Type != "" {
		sportBadge = "  " + tui.SportStyle(e.Type).Render(e.Type)
	}

	fmt.Printf("  %s%s\n", tui.Bold.Render(e.Name), sportBadge)
	fmt.Printf("  %s\n", dimStyle.Render(e.StartDateLocal+"  •  "+e.Category))

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
		fmt.Printf("  %s\n", dimStyle.Render(strings.Join(meta, "  •  ")))
	}

	if e.Description != "" {
		fmt.Println()
		fmt.Printf("  %s\n", dimStyle.Render("Description:"))
		for _, line := range strings.Split(e.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	if e.WorkoutDoc != "" {
		fmt.Println()
		fmt.Printf("  %s\n", tui.Header.Render("  WORKOUT STEPS"))
		fmt.Println()
		for _, line := range strings.Split(e.WorkoutDoc, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fmt.Printf("  %s\n", dimStyle.Render(line))
		}
	}

	fmt.Println()
}

// eventCategoryStyled applies a color to the category badge.
func eventCategoryStyled(cat string) string {
	var color lipgloss.Color
	switch strings.ToUpper(cat) {
	case "WORKOUT":
		color = lipgloss.Color("#42A5F5")
	case "RACE":
		color = lipgloss.Color("#EF5350")
	case "NOTE":
		color = lipgloss.Color("#FFA726")
	case "REST_DAY":
		color = lipgloss.Color("#66BB6A")
	default:
		color = lipgloss.Color("240")
	}
	return lipgloss.NewStyle().Foreground(color).Render(cat)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// buildEventBody constructs a JSON-serialisable map from create/update flags.
func buildEventBody(cmd *cobra.Command, date string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"start_date_local": date,
	}
	if evtName != "" {
		body["name"] = evtName
	}
	if evtDesc != "" {
		body["description"] = evtDesc
	}
	if evtSport != "" {
		body["type"] = evtSport
	}
	if evtCategory != "" {
		body["category"] = evtCategory
	}
	if evtWorkoutDoc != "" {
		// Allow literal \n in flag values to be real newlines.
		body["workout_doc"] = strings.ReplaceAll(evtWorkoutDoc, `\n`, "\n")
	}
	if cmd.Flags().Changed("indoor") {
		body["indoor"] = evtIndoor
	}
	if cmd.Flags().Changed("load-target") {
		body["load_target"] = evtLoadTarget
	}
	if evtDuration != "" {
		secs, err := parseDurationSecs(evtDuration)
		if err != nil {
			return nil, err
		}
		body["duration"] = secs
	}
	return body, nil
}

// parseDurationSecs converts a duration string (integer seconds or Go duration)
// to an integer number of seconds.
func parseDurationSecs(s string) (int, error) {
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: use seconds (3600) or Go format (1h30m)", s)
	}
	return int(d.Seconds()), nil
}

