package format_test

import (
	"testing"
	"time"

	"github.com/tomdiekmann/icu/internal/format"
)

// ── Duration ──────────────────────────────────────────────────────────────────

func TestDuration(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{0, "--"},
		{-1, "--"},
		{60, "0:01:00"},
		{90, "0:01:30"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{7384, "2:03:04"},
		{36000, "10:00:00"},
	}
	for _, c := range cases {
		got := format.Duration(c.secs)
		if got != c.want {
			t.Errorf("Duration(%d) = %q, want %q", c.secs, got, c.want)
		}
	}
}

// ── DistanceKm ────────────────────────────────────────────────────────────────

func TestDistanceKm(t *testing.T) {
	cases := []struct {
		meters float64
		want   string
	}{
		{0, "--"},
		{-100, "--"},
		{1000, "1.0 km"},
		{1500, "1.5 km"},
		{100000, "100.0 km"},
		{42195, "42.2 km"},
	}
	for _, c := range cases {
		got := format.DistanceKm(c.meters)
		if got != c.want {
			t.Errorf("DistanceKm(%v) = %q, want %q", c.meters, got, c.want)
		}
	}
}

// ── Watts ─────────────────────────────────────────────────────────────────────

func TestWatts(t *testing.T) {
	cases := []struct {
		w    float64
		want string
	}{
		{0, "--"},
		{-1, "--"},
		{200, "200w"},
		{250.6, "251w"},
	}
	for _, c := range cases {
		got := format.Watts(c.w)
		if got != c.want {
			t.Errorf("Watts(%v) = %q, want %q", c.w, got, c.want)
		}
	}
}

// ── TSS ───────────────────────────────────────────────────────────────────────

func TestTSS(t *testing.T) {
	cases := []struct {
		tss  float64
		want string
	}{
		{0, "--"},
		{-5, "--"},
		{100, "100"},
		{85.7, "86"},
	}
	for _, c := range cases {
		got := format.TSS(c.tss)
		if got != c.want {
			t.Errorf("TSS(%v) = %q, want %q", c.tss, got, c.want)
		}
	}
}

// ── IF ────────────────────────────────────────────────────────────────────────

func TestIF(t *testing.T) {
	cases := []struct {
		intensity float64
		want      string
	}{
		{0, "--"},
		{0.85, "0.85"},
		{1.0, "1.00"},
		{0.834, "0.83"},
	}
	for _, c := range cases {
		got := format.IF(c.intensity)
		if got != c.want {
			t.Errorf("IF(%v) = %q, want %q", c.intensity, got, c.want)
		}
	}
}

// ── ElevationM ────────────────────────────────────────────────────────────────

func TestElevationM(t *testing.T) {
	cases := []struct {
		m    float64
		want string
	}{
		{0, "--"},
		{100, "100 m"},
		{1234.6, "1235 m"},
		{1234.4, "1234 m"},
	}
	for _, c := range cases {
		got := format.ElevationM(c.m)
		if got != c.want {
			t.Errorf("ElevationM(%v) = %q, want %q", c.m, got, c.want)
		}
	}
}

// ── Calories ──────────────────────────────────────────────────────────────────

func TestCalories(t *testing.T) {
	cases := []struct {
		cal  float64
		want string
	}{
		{0, "--"},
		{500, "500 kcal"},
		{1234.7, "1235 kcal"},
	}
	for _, c := range cases {
		got := format.Calories(c.cal)
		if got != c.want {
			t.Errorf("Calories(%v) = %q, want %q", c.cal, got, c.want)
		}
	}
}

// ── Date ──────────────────────────────────────────────────────────────────────

func TestDate(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"2026-03-15", "Sun 15 Mar"},
		{"2026-03-15T08:30:00", "Sun 15 Mar"},
		{"2026-01-01T00:00:00Z", "Thu 01 Jan"},
		{"bad", "bad"},
		{"2026-03", "2026-03"}, // shorter than 10 chars → returned as-is
	}
	for _, c := range cases {
		got := format.Date(c.input)
		if got != c.want {
			t.Errorf("Date(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── ParseLast ─────────────────────────────────────────────────────────────────

func TestParseLast(t *testing.T) {
	// Fixed reference date: 2026-03-15
	today := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		last        string
		wantOldest  string
		wantNewest  string
		wantErr     bool
	}{
		{"7d", "2026-03-08", "2026-03-15", false},
		{"14d", "2026-03-01", "2026-03-15", false},
		{"4w", "2026-02-15", "2026-03-15", false}, // 4×7=28 days back
		{"1m", "2026-02-13", "2026-03-15", false}, // 1×30=30 days back
		{"1y", "2025-03-15", "2026-03-15", false},
		{"", "", "", true},
		{"7x", "", "", true},
		{"0d", "", "", true},
		{"-3d", "", "", true},
		{"abc", "", "", true},
	}

	for _, c := range cases {
		oldest, newest, err := format.ParseLast(c.last, today)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseLast(%q) expected error, got nil", c.last)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLast(%q) unexpected error: %v", c.last, err)
			continue
		}
		if oldest != c.wantOldest {
			t.Errorf("ParseLast(%q) oldest = %q, want %q", c.last, oldest, c.wantOldest)
		}
		if newest != c.wantNewest {
			t.Errorf("ParseLast(%q) newest = %q, want %q", c.last, newest, c.wantNewest)
		}
	}
}
