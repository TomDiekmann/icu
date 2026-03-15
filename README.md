# icu — A fast, beautiful CLI for Intervals.icu

**For humans and AI agents.**

`icu` wraps the [Intervals.icu](https://intervals.icu) REST API as a dual-mode
command-line tool:

- **Interactive / human mode** (default when stdout is a terminal): rich TUI
  with coloured tables, sparkline trend charts, ASCII fitness charts, an
  interactive calendar, and drill-down navigation powered by
  [Charmbracelet](https://charm.sh).
- **Agent / machine mode** (default when piped, or `--output json`): clean
  structured JSON with consistent schemas and meaningful exit codes — designed
  so AI agents like Claude Code can call `icu`, parse the output, reason about
  training data, and call `icu` again to create or modify events.

---

## Installation

### go install (recommended)

```bash
go install github.com/tomdiekmann/icu@latest
```

Requires Go 1.22+. The binary is installed as `icu`.

### Build from source

```bash
git clone https://github.com/tomdiekmann/icu
cd icu
make install
```

### Shell completions

After installing, add completions for your shell:

```bash
# Bash (~/.bashrc)
source <(icu completion bash)

# Zsh (~/.zshrc)
source <(icu completion zsh)

# Fish
icu completion fish | source

# PowerShell
icu completion powershell | Out-String | Invoke-Expression
```

Or generate completion files with `make completions`.

---

## Quick start

```bash
# 1. Configure your API key (from Settings → Developer Settings in Intervals.icu)
icu configure

# 2. Browse your recent activities
icu activities list

# 3. See your fitness chart
icu fitness

# 4. Open the calendar
icu calendar
```

---

## Configuration

Config file location: `~/.config/icu/config.yaml`

```yaml
api_key: "your_api_key_here"
athlete_id: "0"          # "0" = API key owner
default_output: "auto"   # auto | table | json | csv
units: "metric"          # metric | imperial
```

**Environment variables** (override the config file):

| Variable         | Description                       |
|------------------|-----------------------------------|
| `ICU_API_KEY`    | API key                           |
| `ICU_ATHLETE_ID` | Athlete ID (default `0`)          |
| `ICU_OUTPUT`     | Output format (`auto/json/table`) |
| `ICU_AGENT_MODE` | Set to `1` to force agent mode    |

**Flag precedence:** flags > env vars > config file > defaults.

---

## Commands

### `icu configure`

Interactive prompt to enter and validate your API key. Saves to
`~/.config/icu/config.yaml`.

### `icu athlete`

Show your profile card with body metrics, FTP, LTHR, and a sport settings
summary.

```bash
icu athlete
icu athlete --output json
```

### `icu activities list`

```bash
icu activities list                        # last 7 days
icu activities list --last 30d
icu activities list --type Ride --last 4w
icu activities list --oldest 2026-01-01 --newest 2026-03-31
icu activities list --output json
```

Interactive TUI: `↑↓/jk` navigate • `Enter` drill into detail • `/` search •
`d` download FIT file • `q` quit.

### `icu activities show <id>`

```bash
icu activities show i132173665
icu activities show i132173665 --output json
```

Shows a detail card: summary grid, power/HR zone bars, intervals table.

### `icu activities download <id>`

```bash
icu activities download i132173665
icu activities download i132173665 --output-dir ~/Downloads
icu activities download i132173665 --icu-fit   # Intervals.icu processed FIT
```

### `icu activities upload <file>`

```bash
icu activities upload morning_ride.fit
icu activities upload activity.fit --name "Epic Saturday Ride" --output json
```

### `icu wellness show [date]`

```bash
icu wellness show              # today
icu wellness show 2026-03-10
```

### `icu wellness list`

```bash
icu wellness list              # last 14 days, sparkline dashboard
icu wellness list --last 30d
icu wellness list --output json
```

### `icu wellness update [date]`

```bash
icu wellness update --weight 65.5 --mood 8
icu wellness update 2026-03-10 --resting-hr 48 --hrv 82
icu wellness update --sleep-secs 27000 --steps 9500 --output json
```

### `icu fitness`

ASCII line chart of CTL (fitness), ATL (fatigue), and TSB (form) over time.

```bash
icu fitness                    # 42-day chart ending today
icu fitness --range 90
icu fitness --date 2026-03-01 --range 60
icu fitness --output json
```

### `icu zones`

```bash
icu zones                      # coloured power & HR zone tables
icu zones --output json
```

### `icu events list`

```bash
icu events list
icu events list --last 30d
icu events list --category WORKOUT
icu events list --oldest 2026-03-01 --newest 2026-03-31 --output json
```

### `icu events create`

```bash
# Create from flags
icu events create \
  --date 2026-03-20 \
  --category WORKOUT \
  --sport Ride \
  --name "Sweet Spot Tuesday" \
  --workout-doc "- 15m 55-75%\n- 3x15m 88-93% 5m 55%\n- 10m 55%"

# Create from a JSON file
icu events create --from-json workout.json --output json

# Create from stdin (pipe)
echo '{"start_date_local":"2026-03-20","category":"WORKOUT","type":"Ride","name":"AI Intervals"}' \
  | icu events create --from-json - --output json

# Batch JSONL: create multiple events from a stream
cat training_plan.jsonl | icu events create --from-json - --output json
```

All create flags:

| Flag             | Description                                          |
|------------------|------------------------------------------------------|
| `--date`         | Date YYYY-MM-DD (required without --from-json)       |
| `--category`     | WORKOUT, NOTE, RACE, REST_DAY, etc.                  |
| `--sport`        | Sport type: Ride, Run, Swim, etc.                    |
| `--name`         | Event title                                          |
| `--description`  | Description (markdown supported)                     |
| `--workout-doc`  | Workout steps (see Workout Doc Format below)         |
| `--indoor`       | Mark as indoor                                       |
| `--load-target`  | Target TSS / training load                           |
| `--duration`     | Target duration: `3600` (seconds) or `1h30m`        |
| `--from-json`    | Full JSON payload from file or stdin (`-`)           |

### `icu events update <id>`

Same flags as create (only changed flags are sent). Or `--from-json` for full replacement.

```bash
icu events update 123456 --name "New Name" --load-target 80
icu events update 123456 --from-json updated.json
```

### `icu events delete <id>`

```bash
icu events delete 123456          # prompts for confirmation in terminal
icu events delete 123456 --output json   # deletes immediately, returns JSON
```

### `icu workouts list`

```bash
icu workouts list                 # next 14 days
icu workouts list --last 30d
icu workouts list --output json
```

### `icu workouts show <id>`

```bash
icu workouts show 123456
icu workouts show 123456 --output json
```

### `icu calendar`

Interactive month grid combining completed activities and planned events.

```bash
icu calendar                      # current month
icu calendar --month 2026-02
icu calendar --output json        # JSON array of day entries for the month
```

Navigation: `←→/hl` days • `↑↓/jk` weeks • `[/]` months • `Enter` day detail
• `q` quit.

---

## Agent mode

When stdout is not a terminal, or when `--output json` / `ICU_AGENT_MODE=1` is
set, every command outputs structured JSON:

```json
{
  "meta": {
    "command": "activities list",
    "athlete_id": "i12345",
    "timestamp": "2026-03-15T10:30:00Z",
    "count": 5,
    "filters": {"oldest": "2026-03-08", "newest": "2026-03-15"}
  },
  "data": [ ... ]
}
```

**Exit codes:**

| Code | Meaning                        |
|------|--------------------------------|
| `0`  | Success                        |
| `1`  | General / unknown error        |
| `2`  | Authentication error (401/403) |
| `3`  | Not found (404)                |
| `4`  | Validation error (422)         |
| `5`  | Rate limited after retries     |
| `6`  | Network error                  |

### Example agent workflows

```bash
# Analyse training load with Claude Code
icu fitness --output json | claude "Am I ready for a race this weekend?"

# Get last 30 days of activities and ask for analysis
icu activities list --last 30d --output json \
  | claude "Analyse my training load trend and suggest next week's focus"

# Let an AI agent create a structured training week
claude "Generate a 5-day training plan in JSONL format for an FTP of 280w" \
  | icu events create --from-json - --output json

# Check a specific wellness day
icu wellness show 2026-03-10 --output json | jq '.data.ctl'

# Find all completed rides over 100 TSS in the last month
icu activities list --last 30d --type Ride --output json \
  | jq '[.data[] | select(.icu_training_load > 100)]'
```

---

## Workout Doc Format

Intervals.icu uses a text format to describe workout steps. Pass it via
`--workout-doc` (use `\n` between steps) or embed it in JSON as `workout_doc`.

```
Format                            Meaning
──────────────────────────────────────────────────────
- 15m 55-75%                      15 min at 55–75% FTP
- 2x20m 95-105% 5m 55%           2 × 20 min threshold, 5 min recovery
- 5x5m 105-115% 5m 50%           5 × 5 min VO2max, 5 min recovery
- 30s 150% 30s 50%               30/30 sprints
- 1h 65-75%                       1 hour endurance
- ramp 50-100% 20m               20 min ramp
```

Steps are separated by newlines. In a flag value use `\n`:

```bash
--workout-doc "- 15m 55-75%\n- 3x15m 88-93% 5m 55%\n- 10m 55%"
```

In a JSON payload use real newlines:

```json
{
  "start_date_local": "2026-03-20",
  "category": "WORKOUT",
  "type": "Ride",
  "name": "Sweet Spot Intervals",
  "workout_doc": "- 15m 55-75%\n- 3x15m 88-93% 5m 55%\n- 10m 55%"
}
```

---

## Development

```bash
make build          # build ./icu binary
make install        # install to $GOPATH/bin
make test           # run all tests
make lint           # go vet
make completions    # generate shell completions to ./completions/
make release-dry-run  # goreleaser snapshot build (requires goreleaser)
```

---

## License

MIT
