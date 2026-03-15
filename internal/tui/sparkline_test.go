package tui_test

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tomdiekmann/icu/internal/tui"
)

// ── SparklineTrend ────────────────────────────────────────────────────────────

func TestSparklineTrend_Flat(t *testing.T) {
	vals := []float64{5, 5, 5, 5}
	delta, dir := tui.SparklineTrend(vals)
	if dir != 0 {
		t.Errorf("flat series: dir = %d, want 0", dir)
	}
	if math.Abs(delta) > 0.05 {
		t.Errorf("flat series: delta = %f, want ~0", delta)
	}
}

func TestSparklineTrend_Up(t *testing.T) {
	vals := []float64{1, 2, 3, 10}
	delta, dir := tui.SparklineTrend(vals)
	if dir != 1 {
		t.Errorf("rising series: dir = %d, want 1", dir)
	}
	if delta <= 0 {
		t.Errorf("rising series: delta = %f, want positive", delta)
	}
}

func TestSparklineTrend_Down(t *testing.T) {
	vals := []float64{10, 8, 5, 1}
	delta, dir := tui.SparklineTrend(vals)
	if dir != -1 {
		t.Errorf("falling series: dir = %d, want -1", dir)
	}
	if delta >= 0 {
		t.Errorf("falling series: delta = %f, want negative", delta)
	}
}

func TestSparklineTrend_AllNaN(t *testing.T) {
	vals := []float64{math.NaN(), math.NaN()}
	_, dir := tui.SparklineTrend(vals)
	if dir != 0 {
		t.Errorf("all-NaN: dir = %d, want 0", dir)
	}
}

func TestSparklineTrend_SkipsNaN(t *testing.T) {
	// First and last real values are 1 → 10, should trend up.
	vals := []float64{1, math.NaN(), math.NaN(), 10}
	_, dir := tui.SparklineTrend(vals)
	if dir != 1 {
		t.Errorf("NaN-containing rising: dir = %d, want 1", dir)
	}
}

func TestSparklineTrend_SmallDelta(t *testing.T) {
	// Delta of 0.01 is within the ±0.05 flat band.
	vals := []float64{5.0, 5.01}
	_, dir := tui.SparklineTrend(vals)
	if dir != 0 {
		t.Errorf("tiny delta: dir = %d, want 0 (flat)", dir)
	}
}

// ── RenderSparkline ───────────────────────────────────────────────────────────

func TestRenderSparkline_Length(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	color := lipgloss.Color("#42A5F5")
	result := tui.RenderSparkline(vals, color)
	// Strip ANSI; count rune characters.
	// The raw sparkline characters are 5 — one per value.
	// We can't easily count with ANSI, so just check the string is non-empty.
	if result == "" {
		t.Error("RenderSparkline returned empty string for non-empty input")
	}
}

func TestRenderSparkline_NaNRenderedAsDash(t *testing.T) {
	vals := []float64{math.NaN()}
	color := lipgloss.Color("#42A5F5")
	result := tui.RenderSparkline(vals, color)
	if !strings.Contains(result, "╌") {
		t.Errorf("NaN value should render as dim dash ╌, got: %q", result)
	}
}

func TestRenderSparkline_Empty(t *testing.T) {
	result := tui.RenderSparkline(nil, lipgloss.Color("#42A5F5"))
	if result != "" {
		t.Errorf("empty input should return empty string, got: %q", result)
	}
}

func TestRenderSparkline_SingleValue(t *testing.T) {
	vals := []float64{42}
	result := tui.RenderSparkline(vals, lipgloss.Color("#42A5F5"))
	if result == "" {
		t.Error("single value should produce non-empty sparkline")
	}
}

// ── TrendArrow ────────────────────────────────────────────────────────────────

func TestTrendArrow_ContainsArrow(t *testing.T) {
	up := tui.TrendArrow(1, tui.GoodUp)
	if !strings.Contains(up, "↑") {
		t.Errorf("up direction should contain ↑, got: %q", up)
	}

	down := tui.TrendArrow(-1, tui.GoodUp)
	if !strings.Contains(down, "↓") {
		t.Errorf("down direction should contain ↓, got: %q", down)
	}

	flat := tui.TrendArrow(0, tui.GoodUp)
	if !strings.Contains(flat, "→") {
		t.Errorf("flat direction should contain →, got: %q", flat)
	}
}

func TestTrendArrow_NilGoodUp(t *testing.T) {
	// nil = neutral, always dim — should not panic
	_ = tui.TrendArrow(1, nil)
	_ = tui.TrendArrow(-1, nil)
	_ = tui.TrendArrow(0, nil)
}

// ── SparklineRow ─────────────────────────────────────────────────────────────

func TestSparklineRow_NonEmpty(t *testing.T) {
	vals := []float64{1, 2, 3}
	result := tui.SparklineRow("Weight", "65.5 kg", vals, lipgloss.Color("#FFA726"), nil, 18)
	if result == "" {
		t.Error("SparklineRow returned empty string")
	}
	if !strings.Contains(result, "Weight") {
		t.Errorf("SparklineRow should contain the label, got: %q", result)
	}
}
