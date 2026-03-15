package format

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Duration formats seconds into HH:MM:SS (or H:MM:SS for long activities).
func Duration(seconds int) string {
	if seconds <= 0 {
		return "--"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// DistanceKm formats meters into kilometres with one decimal place.
func DistanceKm(meters float64) string {
	if meters <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.1f km", meters/1000)
}

// DistanceMiles formats meters into miles with one decimal place.
func DistanceMiles(meters float64) string {
	if meters <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.1f mi", meters/1609.344)
}

// Watts formats a power value; returns "--" for zero.
func Watts(w float64) string {
	if w <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.0fw", w)
}

// Heartrate formats a HR value; returns "--" for zero.
func Heartrate(hr float64) string {
	if hr <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.0f", hr)
}

// ElevationM formats meters of elevation gain; returns "--" for zero.
func ElevationM(m float64) string {
	if m <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.0f m", m)
}

// Calories formats a calorie value; returns "--" for zero.
func Calories(cal float64) string {
	if cal <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.0f kcal", cal)
}

// TSS formats a training load value.
func TSS(tss float64) string {
	if tss <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.0f", tss)
}

// IF formats an intensity factor value.
func IF(intensity float64) string {
	if intensity <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.2f", intensity)
}

// Date formats an ISO-8601 date-time string as "Mon 02 Jan 2006".
func Date(s string) string {
	for _, layout := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t.Format("Mon 02 Jan")
		}
	}
	// fallback: return first 10 chars
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// ParseLast converts a --last flag value (e.g. "7d", "4w", "3m", "1y") into
// oldest and newest date strings (YYYY-MM-DD) relative to today.
func ParseLast(last string, today time.Time) (oldest, newest string, err error) {
	last = strings.ToLower(strings.TrimSpace(last))
	if last == "" {
		return "", "", fmt.Errorf("empty duration")
	}

	unit := last[len(last)-1]
	numStr := last[:len(last)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return "", "", fmt.Errorf("invalid duration %q: number must be a positive integer", last)
	}

	var days int
	switch unit {
	case 'd':
		days = n
	case 'w':
		days = n * 7
	case 'm':
		days = n * 30
	case 'y':
		days = n * 365
	default:
		return "", "", fmt.Errorf("invalid duration %q: unit must be d, w, m, or y", last)
	}

	newest = today.Format("2006-01-02")
	oldest = today.AddDate(0, 0, -days).Format("2006-01-02")
	return oldest, newest, nil
}
