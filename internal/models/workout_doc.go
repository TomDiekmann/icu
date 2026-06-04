package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WorkoutDoc is the workout_doc field of an event. The API returns it as a
// parsed object (steps, duration, zone times, ...) on reads but accepts a
// plain string in the Intervals.icu workout text format on writes. The raw
// JSON is preserved so no data is lost; Text renders a line-based view of
// the steps for human-mode display.
type WorkoutDoc []byte

func (w *WorkoutDoc) UnmarshalJSON(data []byte) error {
	*w = append((*w)[:0], data...)
	return nil
}

func (w WorkoutDoc) MarshalJSON() ([]byte, error) {
	if len(w) == 0 {
		return []byte("null"), nil
	}
	return w, nil
}

// docRange is a step target (power, hr, pace) with either a single value or
// a start-end range, plus units like "%ftp", "%lthr", "w", "bpm".
type docRange struct {
	Units string   `json:"units,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Start *float64 `json:"start,omitempty"`
	End   *float64 `json:"end,omitempty"`
}

type docStep struct {
	Text     string    `json:"text,omitempty"`
	Duration *int      `json:"duration,omitempty"`
	Reps     int       `json:"reps,omitempty"`
	Power    *docRange `json:"power,omitempty"`
	HR       *docRange `json:"hr,omitempty"`
	Pace     *docRange `json:"pace,omitempty"`
	Warmup   bool      `json:"warmup,omitempty"`
	Cooldown bool      `json:"cooldown,omitempty"`
	Steps    []docStep `json:"steps,omitempty"`
}

type docBody struct {
	Description string    `json:"description,omitempty"`
	Steps       []docStep `json:"steps,omitempty"`
}

// Text returns the workout steps as newline-separated lines in the
// Intervals.icu text format ("- 15m 55-75%"). A string workout_doc is
// returned as-is; an object is rendered from its parsed steps. Returns ""
// when there is nothing to show.
func (w WorkoutDoc) Text() string {
	trimmed := strings.TrimSpace(string(w))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(w, &s); err == nil {
			return s
		}
		return ""
	}
	var body docBody
	if err := json.Unmarshal(w, &body); err != nil {
		return ""
	}
	var lines []string
	renderDocSteps(body.Steps, "- ", &lines)
	return strings.Join(lines, "\n")
}

func renderDocSteps(steps []docStep, prefix string, lines *[]string) {
	for _, s := range steps {
		if s.Reps > 1 || len(s.Steps) > 0 {
			header := s.Text
			if header == "" && s.Reps > 1 {
				header = fmt.Sprintf("%dx", s.Reps)
			}
			if header != "" {
				*lines = append(*lines, prefix+header)
			}
			renderDocSteps(s.Steps, "  ", lines)
			continue
		}
		if line := s.line(); line != "" {
			*lines = append(*lines, prefix+line)
		}
	}
}

// line renders a single step like "15m 55-75%".
func (s docStep) line() string {
	var parts []string
	if s.Duration != nil && *s.Duration > 0 {
		parts = append(parts, formatStepDuration(*s.Duration))
	}
	for _, r := range []*docRange{s.Power, s.HR, s.Pace} {
		if t := r.target(); t != "" {
			parts = append(parts, t)
		}
	}
	if s.Text != "" && len(parts) == 0 {
		parts = append(parts, s.Text)
	}
	switch {
	case s.Warmup:
		parts = append(parts, "warmup")
	case s.Cooldown:
		parts = append(parts, "cooldown")
	}
	return strings.Join(parts, " ")
}

// target renders a range like "55-75%", "85%" or "200-250w".
func (r *docRange) target() string {
	if r == nil {
		return ""
	}
	unit := r.Units
	if strings.HasPrefix(unit, "%") { // %ftp, %lthr, %pace → just "%"
		unit = "%"
	}
	switch {
	case r.Start != nil && r.End != nil:
		return fmt.Sprintf("%s-%s%s", formatStepNum(*r.Start), formatStepNum(*r.End), unit)
	case r.Value != nil:
		return formatStepNum(*r.Value) + unit
	}
	return ""
}

// formatStepDuration renders seconds compactly: 900 → "15m", 5400 → "1h30m",
// 90 → "1m30s", 30 → "30s".
func formatStepDuration(secs int) string {
	h, m, s := secs/3600, secs%3600/60, secs%60
	var sb strings.Builder
	if h > 0 {
		fmt.Fprintf(&sb, "%dh", h)
	}
	if m > 0 {
		fmt.Fprintf(&sb, "%dm", m)
	}
	if s > 0 || sb.Len() == 0 {
		fmt.Fprintf(&sb, "%ds", s)
	}
	return sb.String()
}

func formatStepNum(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%.1f", v)
}
