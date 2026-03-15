package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tomdiekmann/icu/internal/format"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/tui"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type viewState int

const (
	stateGrid viewState = iota
	stateDay
)

// CalEntry holds all items for a single calendar day.
type CalEntry struct {
	Activities []models.Activity
	Events     []models.Event
}

// Model is the bubbletea model for the month calendar.
type Model struct {
	entries map[string]CalEntry // YYYY-MM-DD → items

	// Boundaries of the loaded data range.
	oldest string // YYYY-MM-DD
	newest string // YYYY-MM-DD

	// Currently displayed month.
	year  int
	month time.Month

	// Selected day within the displayed month (1-based).
	cursorDay int

	state viewState
	vp    viewport.Model

	width  int
	height int
	today  string // YYYY-MM-DD for highlighting
}

// New creates a calendar model centred on the given year/month.
// oldest/newest bound how far the user can navigate.
func New(entries map[string]CalEntry, year int, month time.Month, oldest, newest string) Model {
	today := time.Now().Format("2006-01-02")

	cursor := 1
	t := time.Now()
	if t.Year() == year && t.Month() == month {
		cursor = t.Day()
	}

	return Model{
		entries:   entries,
		oldest:    oldest,
		newest:    newest,
		year:      year,
		month:     month,
		cursorDay: cursor,
		today:     today,
		vp:        viewport.New(0, 0),
	}
}

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.Width = msg.Width
		m.vp.Height = msg.Height - 1
		if m.state == stateDay {
			m.vp.SetContent(m.dayContent())
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case stateGrid:
			return m.handleGridKey(msg)
		case stateDay:
			return m.handleDayKey(msg)
		}
	}

	if m.state == stateDay {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleGridKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	curDays := daysInMonth(m.year, m.month)

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "left", "h":
		if m.cursorDay > 1 {
			m.cursorDay--
		} else {
			next := m.navigatePrevMonth()
			if next.month != m.month || next.year != m.year {
				m = next
				m.cursorDay = daysInMonth(m.year, m.month)
			}
		}

	case "right", "l":
		if m.cursorDay < curDays {
			m.cursorDay++
		} else {
			next := m.navigateNextMonth()
			if next.month != m.month || next.year != m.year {
				m = next
				m.cursorDay = 1
			}
		}

	case "up", "k":
		m.cursorDay -= 7
		if m.cursorDay < 1 {
			m.cursorDay = 1
		}

	case "down", "j":
		m.cursorDay += 7
		if m.cursorDay > curDays {
			m.cursorDay = curDays
		}

	case "[", "pgup":
		next := m.navigatePrevMonth()
		m = next
		d := daysInMonth(m.year, m.month)
		if m.cursorDay > d {
			m.cursorDay = d
		}

	case "]", "pgdn":
		next := m.navigateNextMonth()
		m = next
		d := daysInMonth(m.year, m.month)
		if m.cursorDay > d {
			m.cursorDay = d
		}

	case "enter", " ":
		m.state = stateDay
		m.vp.Width = m.width
		m.vp.Height = m.height - 1
		m.vp.GotoTop()
		m.vp.SetContent(m.dayContent())
	}

	return m, nil
}

func (m Model) handleDayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = stateGrid
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) navigatePrevMonth() Model {
	t := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
	oldest, err := time.Parse("2006-01-02", m.oldest)
	if err == nil {
		firstOfNew := t
		if firstOfNew.Before(oldest.AddDate(0, 0, -oldest.Day()+1)) {
			return m // at boundary
		}
	}
	m.year = t.Year()
	m.month = t.Month()
	return m
}

func (m Model) navigateNextMonth() Model {
	t := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	newest, err := time.Parse("2006-01-02", m.newest)
	if err == nil {
		// Block if the new month starts after newest.
		if t.After(newest) {
			return m // at boundary
		}
	}
	m.year = t.Year()
	m.month = t.Month()
	return m
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.state == stateDay {
		hint := "  esc/q: back  •  ↑↓: scroll"
		return m.vp.View() + "\n" + tui.StatusBarStyle.Width(m.width).Render(hint)
	}
	return m.renderGrid()
}

func (m Model) renderGrid() string {
	cellW := m.width / 7
	if cellW < 10 {
		cellW = 10
	}

	// Available height: total - 1 status - 1 month header - 1 blank - 1 weekday - 1 separator.
	const overhead = 5
	available := m.height - overhead
	if available < 6 {
		available = 6
	}
	cellH := available / 6
	if cellH < 2 {
		cellH = 2
	}
	if cellH > 4 {
		cellH = 4
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var lines []string

	// ── Month header ──────────────────────────────────────────────────────────
	monthName := fmt.Sprintf("%s %d", m.month.String(), m.year)
	lines = append(lines, fmt.Sprintf("  ◀  %s  ▶", tui.Bold.Render(monthName)))
	lines = append(lines, "")

	// ── Weekday header ────────────────────────────────────────────────────────
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	wdRow := ""
	for _, d := range weekdays {
		wdRow += lipgloss.NewStyle().Width(cellW).Render(dim.Render(d))
	}
	lines = append(lines, wdRow)
	lines = append(lines, dim.Render(strings.Repeat("─", m.width)))

	// ── Build 6×7 day grid ───────────────────────────────────────────────────
	firstDay := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	startWd := int(firstDay.Weekday()) // 0=Sun
	totalDays := daysInMonth(m.year, m.month)

	var grid [6][7]int
	day := 1
	for row := 0; row < 6 && day <= totalDays; row++ {
		for col := 0; col < 7 && day <= totalDays; col++ {
			pos := row*7 + col
			if pos >= startWd {
				grid[row][col] = day
				day++
			}
		}
	}

	// ── Render grid rows ─────────────────────────────────────────────────────
	for row := 0; row < 6; row++ {
		// Skip entirely empty rows.
		hasDay := false
		for col := 0; col < 7; col++ {
			if grid[row][col] > 0 {
				hasDay = true
				break
			}
		}
		if !hasDay {
			break
		}
		for line := 0; line < cellH; line++ {
			rowStr := ""
			for col := 0; col < 7; col++ {
				rowStr += m.renderCellLine(grid[row][col], line, cellW, cellH)
			}
			lines = append(lines, rowStr)
		}
	}

	// ── Pad to fill the screen (keeps status bar pinned to the bottom) ────────
	for len(lines) < m.height-1 {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	// ── Status bar ────────────────────────────────────────────────────────────
	curDate := fmt.Sprintf("%d-%02d-%02d", m.year, int(m.month), m.cursorDay)
	entry := m.entries[curDate]
	entrySummary := ""
	if n := len(entry.Activities) + len(entry.Events); n > 0 {
		entrySummary = fmt.Sprintf("  (%d items)", n)
	}
	left := " " + curDate + entrySummary
	right := "←→/hl: day  ↑↓/jk: week  []: month  enter: details  q: quit "
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	statusText := left + strings.Repeat(" ", gap) + right

	return strings.Join(lines[:m.height-1], "\n") + "\n" +
		tui.StatusBarStyle.Width(m.width).Render(statusText)
}

// renderCellLine renders one horizontal line of one day cell.
func (m Model) renderCellLine(day, line, cellW, cellH int) string {
	selected := day == m.cursorDay
	dateKey := fmt.Sprintf("%d-%02d-%02d", m.year, int(m.month), day)
	isToday := day > 0 && dateKey == m.today
	entry := m.entries[dateKey]

	var content string

	switch {
	case day == 0:
		content = ""
	case line == 0:
		// Day number line.
		dayStr := fmt.Sprintf("%2d", day)
		var dayStyle lipgloss.Style
		if isToday {
			dayStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA726"))
		} else {
			dayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		}
		// Count items so we can show "+N" indicator on the last item line.
		itemCount := len(entry.Activities) + len(entry.Events)
		indicator := ""
		if itemCount > 0 && cellH > 1 {
			maxItems := cellH - 1
			if itemCount > maxItems {
				indicator = dim.Render(fmt.Sprintf(" +%d", itemCount))
			}
		}
		content = " " + dayStyle.Render(dayStr) + indicator
	default:
		// Item line (line 1 = first item, line 2 = second, etc.).
		itemIdx := line - 1
		content = m.cellItemStr(entry, itemIdx, cellW)
	}

	cellStyle := lipgloss.NewStyle().Width(cellW)
	if selected {
		cellStyle = cellStyle.Background(lipgloss.Color("236"))
	}
	return cellStyle.Render(content)
}

// dim is a package-level style for grey text (avoids repeated allocations).
var dim = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// cellItemStr returns a styled string for the n-th item in a day cell.
func (m Model) cellItemStr(entry CalEntry, idx, cellW int) string {
	type item struct {
		text  string
		color lipgloss.Color
	}

	var items []item

	for _, a := range entry.Activities {
		h := a.MovingTime / 3600
		min := (a.MovingTime % 3600) / 60
		dur := ""
		if h > 0 {
			dur = fmt.Sprintf("%d:%02d", h, min)
		} else if min > 0 {
			dur = fmt.Sprintf("%dm", min)
		}
		text := "●" + abbrevSport(a.Type)
		if dur != "" {
			text += " " + dur
		}
		if a.IcuTrainingLoad > 0 {
			text += fmt.Sprintf(" %.0fT", a.IcuTrainingLoad)
		}
		c, ok := tui.SportColor[a.Type]
		if !ok {
			c = tui.SportColorDefault
		}
		items = append(items, item{text, c})
	}

	for _, e := range entry.Events {
		name := e.Name
		if name == "" {
			name = e.Category
		}
		text := "○" + name
		var c lipgloss.Color
		switch strings.ToUpper(e.Category) {
		case "WORKOUT":
			c = lipgloss.Color("#42A5F5")
		case "RACE":
			c = lipgloss.Color("#EF5350")
		case "REST_DAY":
			c = lipgloss.Color("#66BB6A")
		default:
			c = lipgloss.Color("240")
		}
		items = append(items, item{text, c})
	}

	if idx >= len(items) {
		return ""
	}

	it := items[idx]
	maxLen := cellW - 2
	if maxLen < 3 {
		maxLen = 3
	}
	runes := []rune(it.text)
	if len(runes) > maxLen {
		it.text = string(runes[:maxLen-1]) + "…"
	}
	return " " + lipgloss.NewStyle().Foreground(it.color).Render(it.text)
}

// abbrevSport shortens long sport names to fit narrow calendar cells.
func abbrevSport(s string) string {
	switch s {
	case "VirtualRide":
		return "VRide"
	case "VirtualRun":
		return "VRun"
	case "WeightTraining":
		return "Wt"
	case "NordicSki":
		return "NSki"
	default:
		if len(s) > 7 {
			return s[:7]
		}
		return s
	}
}

// ── Day detail ────────────────────────────────────────────────────────────────

func (m Model) dayContent() string {
	dateKey := fmt.Sprintf("%d-%02d-%02d", m.year, int(m.month), m.cursorDay)
	entry := m.entries[dateKey]

	var sb strings.Builder

	t, _ := time.Parse("2006-01-02", dateKey)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", tui.Bold.Render(t.Format("Monday, January 2, 2006"))))

	if len(entry.Activities) == 0 && len(entry.Events) == 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n\n", dim.Render("Nothing logged or planned on this day.")))
		return sb.String()
	}

	// ── Activities ────────────────────────────────────────────────────────────
	if len(entry.Activities) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n",
			tui.Header.Render(fmt.Sprintf("  COMPLETED  (%d)", len(entry.Activities)))))
		sb.WriteString(fmt.Sprintf("  %s\n", dim.Render(strings.Repeat("─", max(m.width-4, 10)))))
		for _, a := range entry.Activities {
			sport := tui.SportStyle(a.Type).Render(fmt.Sprintf("%-13s", a.Type))
			name := a.Name
			if len([]rune(name)) > 32 {
				name = string([]rune(name)[:31]) + "…"
			}
			meta := format.Duration(a.MovingTime)
			if d := format.DistanceKm(a.Distance); d != "--" {
				meta += "  " + d
			}
			if a.IcuTrainingLoad > 0 {
				meta += fmt.Sprintf("  TSS:%.0f", a.IcuTrainingLoad)
			}
			sb.WriteString(fmt.Sprintf("  ● %s  %-34s  %s\n",
				sport, name, dim.Render(meta)))
		}
	}

	// ── Events ────────────────────────────────────────────────────────────────
	if len(entry.Events) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n",
			tui.Header.Render(fmt.Sprintf("  PLANNED  (%d)", len(entry.Events)))))
		sb.WriteString(fmt.Sprintf("  %s\n", dim.Render(strings.Repeat("─", max(m.width-4, 10)))))

		for _, e := range entry.Events {
			var catStyle lipgloss.Style
			switch strings.ToUpper(e.Category) {
			case "WORKOUT":
				catStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#42A5F5"))
			case "RACE":
				catStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350"))
			default:
				catStyle = dim
			}

			name := e.Name
			sport := ""
			if e.Type != "" {
				sport = "  " + tui.SportStyle(e.Type).Render(e.Type)
			}
			meta := ""
			if e.Duration != nil && *e.Duration > 0 {
				meta += "  " + format.Duration(*e.Duration)
			}
			if e.LoadTarget != nil && *e.LoadTarget > 0 {
				meta += fmt.Sprintf("  TSS:%.0f", *e.LoadTarget)
			}

			sb.WriteString(fmt.Sprintf("  ○ %s  %s%s%s\n",
				catStyle.Render(fmt.Sprintf("%-10s", e.Category)),
				name, sport, dim.Render(meta)))

			if e.Description != "" {
				desc := e.Description
				if len([]rune(desc)) > 80 {
					desc = string([]rune(desc)[:79]) + "…"
				}
				sb.WriteString(fmt.Sprintf("    %s\n", dim.Render(desc)))
			}
			if e.WorkoutDoc != "" {
				for _, step := range strings.Split(e.WorkoutDoc, "\n") {
					step = strings.TrimSpace(step)
					if step == "" {
						continue
					}
					sb.WriteString(fmt.Sprintf("    %s\n", dim.Render(step)))
				}
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func daysInMonth(year int, month time.Month) int {
	// time.Date with day=0 gives the last day of the previous month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
