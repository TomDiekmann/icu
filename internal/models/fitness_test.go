package models_test

import (
	"testing"

	"github.com/tomdiekmann/icu/internal/models"
)

func TestFormStatus(t *testing.T) {
	// Boundaries are exclusive (tsb > threshold), so exact boundary values fall
	// into the NEXT (worse) category.
	cases := []struct {
		tsb       float64
		wantLabel string
	}{
		{30, "VERY FRESH"},
		{25.1, "VERY FRESH"},
		{25, "FRESH"},    // exactly 25 is NOT > 25
		{10, "FRESH"},
		{5.1, "FRESH"},
		{5, "NEUTRAL"},   // exactly 5 is NOT > 5
		{0, "NEUTRAL"},
		{-9, "NEUTRAL"},
		{-10, "FATIGUED"}, // exactly -10 is NOT > -10
		{-20, "FATIGUED"},
		{-30, "OVERREACHING"}, // exactly -30 is NOT > -30
		{-50, "OVERREACHING"},
	}
	for _, c := range cases {
		label, color := models.FormStatus(c.tsb)
		if label != c.wantLabel {
			t.Errorf("FormStatus(%v) label = %q, want %q", c.tsb, label, c.wantLabel)
		}
		if color == "" {
			t.Errorf("FormStatus(%v) returned empty color", c.tsb)
		}
	}
}

func TestWellnessEntry_TSB(t *testing.T) {
	e := models.WellnessEntry{CTL: 60, ATL: 70}
	if got := e.TSB(); got != -10 {
		t.Errorf("TSB() = %v, want -10", got)
	}
	e2 := models.WellnessEntry{CTL: 55, ATL: 40}
	if got := e2.TSB(); got != 15 {
		t.Errorf("TSB() = %v, want 15", got)
	}
}
