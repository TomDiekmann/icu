package activities

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tomdiekmann/icu/internal/format"
	"github.com/tomdiekmann/icu/internal/models"
	"github.com/tomdiekmann/icu/internal/tui"
)

// Column widths (visual characters).
const (
	colIDW   = 11 // "i132173665"
	colDateW = 10 // "Sun 15 Mar"
	colSport = 13 // "VirtualRide "
	colDurW  = 8  // "2:42:09"
	colDistW = 9  // "103.3 km"
	colTSSW  = 5  // "188"
	colIFW   = 5  // "0.83"
	colWattW = 6  // "207w"
	colHRW   = 6  // "142"
	colSep   = 2  // "  "
)

type viewState int

const (
	stateList viewState = iota
	stateDetail
)

// ── Message types ─────────────────────────────────────────────────────────────

type intervalsLoadedMsg struct {
	intervals []models.Interval
	err       error
}

type downloadedMsg struct {
	path string
	err  error
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the bubbletea model for the activities list and drill-down detail.
type Model struct {
	// data
	activities []models.Activity
	filtered   []models.Activity
	oldest     string
	newest     string

	// list state
	cursor int
	offset int

	// search
	searching bool
	search    textinput.Model

	// view
	state viewState
	vp    viewport.Model

	// detail
	detailAct     models.Activity
	detailIntvls  []models.Interval
	detailLoading bool

	// callbacks (set by New)
	fetchIntervalsFn func(id string) ([]models.Interval, error)
	downloadFn       func(id string) (string, error)

	// terminal dimensions
	width  int
	height int

	// transient status message (download result, errors)
	statusMsg string
}

// New creates the list model. fetchIntervals and download are called
// asynchronously via tea.Cmd when the user presses Enter or d.
func New(
	activities []models.Activity,
	oldest, newest string,
	fetchIntervals func(id string) ([]models.Interval, error),
	download func(id string) (string, error),
) Model {
	si := textinput.New()
	si.Placeholder = "filter by name, sport or id…"
	si.CharLimit = 60

	return Model{
		activities:       activities,
		filtered:         activities,
		oldest:           oldest,
		newest:           newest,
		fetchIntervalsFn: fetchIntervals,
		downloadFn:       download,
		search:           si,
		vp:               viewport.New(0, 0),
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
		m.vp.Height = m.height - 1 // 1 line for status bar
		if m.state == stateDetail {
			m.vp.SetContent(m.detailContent())
		}

	case intervalsLoadedMsg:
		m.detailLoading = false
		if msg.err == nil {
			m.detailIntvls = msg.intervals
		}
		m.vp.SetContent(m.detailContent())

	case downloadedMsg:
		if msg.err != nil {
			m.statusMsg = "error: " + msg.err.Error()
		} else {
			m.statusMsg = "downloaded → " + msg.path
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case stateDetail:
			return m.handleDetailKey(msg)
		case stateList:
			if m.searching {
				return m.handleSearchKey(msg)
			}
			return m.handleListKey(msg)
		}
	}

	// Pass non-key messages to the viewport when in detail view.
	if m.state == stateDetail {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleRows()

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.offset = adjustOffset(m.offset, m.cursor, visible)
			m.statusMsg = ""
		}

	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.offset = adjustOffset(m.offset, m.cursor, visible)
			m.statusMsg = ""
		}

	case "pgup":
		m.cursor -= visible
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset = adjustOffset(m.offset, m.cursor, visible)

	case "pgdown":
		m.cursor += visible
		if m.cursor >= len(m.filtered) {
			m.cursor = maxInt(0, len(m.filtered)-1)
		}
		m.offset = adjustOffset(m.offset, m.cursor, visible)

	case "home", "g":
		m.cursor = 0
		m.offset = 0

	case "end", "G":
		m.cursor = maxInt(0, len(m.filtered)-1)
		m.offset = adjustOffset(m.offset, m.cursor, visible)

	case "/":
		m.searching = true
		m.search.Focus()
		return m, textinput.Blink

	case "enter":
		if len(m.filtered) == 0 {
			break
		}
		a := m.filtered[m.cursor]
		m.detailAct = a
		m.detailIntvls = nil
		m.detailLoading = true
		m.state = stateDetail
		m.vp.Width = m.width
		m.vp.Height = m.height - 1
		m.vp.GotoTop()
		m.vp.SetContent(m.detailContent())
		return m, m.cmdFetchIntervals(a.ID)

	case "d":
		if len(m.filtered) == 0 {
			break
		}
		a := m.filtered[m.cursor]
		m.statusMsg = fmt.Sprintf("downloading %s…", a.ID)
		return m, m.cmdDownload(a.ID)
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.search.Blur()
		// Clear filter on Esc.
		m.search.SetValue("")
		m.filtered = m.activities
		m.cursor = 0
		m.offset = 0
		return m, nil
	case "enter":
		m.searching = false
		m.search.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.filtered = filterActivities(m.activities, m.search.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = maxInt(0, len(m.filtered)-1)
	}
	m.offset = adjustOffset(m.offset, m.cursor, m.visibleRows())
	return m, cmd
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = stateList
		m.statusMsg = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m Model) cmdFetchIntervals(id string) tea.Cmd {
	fn := m.fetchIntervalsFn
	return func() tea.Msg {
		intervals, err := fn(id)
		return intervalsLoadedMsg{intervals: intervals, err: err}
	}
}

func (m Model) cmdDownload(id string) tea.Cmd {
	fn := m.downloadFn
	return func() tea.Msg {
		path, err := fn(id)
		return downloadedMsg{path: path, err: err}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.state == stateDetail {
		statusText := "  esc/q: back  •  ↑↓: scroll  •  pgup/pgdn: page"
		return m.vp.View() + "\n" + tui.StatusBarStyle.Width(m.width).Render(statusText)
	}
	return m.renderList()
}

func (m Model) renderList() string {
	var sb strings.Builder

	// Header + separator
	sb.WriteString(m.renderHeader() + "\n")
	sb.WriteString(tui.TableBorderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Data rows
	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := m.offset; i < end; i++ {
		sb.WriteString(m.renderRow(m.filtered[i], i == m.cursor) + "\n")
	}
	// Blank filler so the status bar stays at the bottom.
	for i := end - m.offset; i < visible; i++ {
		sb.WriteString(strings.Repeat(" ", m.width) + "\n")
	}

	sb.WriteString(m.renderStatus())
	return sb.String()
}

func (m Model) renderHeader() string {
	cc := m.colConfig()
	row := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		colIDW, "ID",
		colDateW, "DATE",
		colSport, "SPORT",
		cc.nameW, "NAME",
		colDurW, "DURATION",
		colDistW, "DISTANCE",
	)
	if cc.medium {
		row += fmt.Sprintf("  %-*s", colTSSW, "TSS")
	}
	if cc.wide {
		row += fmt.Sprintf("  %-*s  %-*s  %-*s", colIFW, "IF", colWattW, "AVG W", colHRW, "AVG HR")
	}
	return tui.TableHeaderStyle.Render(row)
}

func (m Model) renderRow(a models.Activity, selected bool) string {
	cc := m.colConfig()

	cursor := "  "
	if selected {
		cursor = "▶ "
	}

	// ID: dim when not selected, plain when selected (highlight bg makes it visible)
	idStr := fmt.Sprintf("%-*s", colIDW, a.ID)
	if !selected {
		idStr = tui.Dim.Render(idStr)
	}

	date := fmt.Sprintf("%-*s", colDateW, format.Date(a.StartDateLocal))

	// Sport: colour-coded, padded before styling so width is correct
	sportStr := tui.SportStyle(a.Type).Render(fmt.Sprintf("%-*s", colSport, a.Type))

	name := padRight(truncate(a.Name, cc.nameW), cc.nameW)
	dur := fmt.Sprintf("%-*s", colDurW, format.Duration(a.MovingTime))
	dist := fmt.Sprintf("%-*s", colDistW, format.DistanceKm(a.Distance))

	row := cursor + idStr + "  " + date + "  " + sportStr + "  " + name + "  " + dur + "  " + dist
	if cc.medium {
		row += "  " + fmt.Sprintf("%-*s", colTSSW, format.TSS(a.IcuTrainingLoad))
	}
	if cc.wide {
		row += "  " + fmt.Sprintf("%-*s", colIFW, format.IF(a.IntensityFactor()))
		row += "  " + fmt.Sprintf("%-*s", colWattW, format.Watts(a.IcuAverageWatts))
		row += "  " + fmt.Sprintf("%-*s", colHRW, format.Heartrate(a.AverageHeartrate))
	}

	if selected {
		return lipgloss.NewStyle().Background(lipgloss.Color("236")).Width(m.width).Render(row)
	}
	return row
}

func (m Model) renderStatus() string {
	if m.searching {
		prompt := tui.Highlight.Render("/") + " " + m.search.View()
		hints := tui.Dim.Render("enter: confirm  esc: clear")
		gap := m.width - lipgloss.Width(prompt) - lipgloss.Width(hints) - 2
		if gap < 1 {
			gap = 1
		}
		line := " " + prompt + strings.Repeat(" ", gap) + hints + " "
		return tui.StatusBarStyle.Width(m.width).Render(line)
	}

	var left string
	if m.statusMsg != "" {
		left = m.statusMsg
	} else {
		cnt, total := len(m.filtered), len(m.activities)
		if cnt == total {
			left = fmt.Sprintf("%d activities  •  %s → %s", total, m.oldest, m.newest)
		} else {
			left = fmt.Sprintf("%d/%d  •  %s → %s", cnt, total, m.oldest, m.newest)
		}
	}
	right := "↑↓: nav  enter: detail  d: download  /: search  q: quit"

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := " " + left + strings.Repeat(" ", gap) + right + " "
	return tui.StatusBarStyle.Width(m.width).Render(line)
}

// ── Detail helpers ────────────────────────────────────────────────────────────

func (m Model) detailContent() string {
	content := RenderDetail(m.detailAct, m.detailIntvls, m.width)
	if m.detailLoading {
		content += "\n  " + tui.Dim.Render("Loading intervals…") + "\n"
	}
	return content
}

// ── Layout helpers ────────────────────────────────────────────────────────────

type colCfg struct {
	nameW  int
	medium bool
	wide   bool
}

func (m Model) colConfig() colCfg {
	wide := m.width >= 120
	medium := m.width >= 90

	// cursor(2) + ID + sep + DATE + sep + SPORT + sep + [name] + sep + DUR + sep + DIST
	fixed := 2 + colIDW + colSep + colDateW + colSep + colSport + colSep + colSep + colDurW + colSep + colDistW
	if medium {
		fixed += colSep + colTSSW
	}
	if wide {
		fixed += colSep + colIFW + colSep + colWattW + colSep + colHRW
	}

	nameW := m.width - fixed
	if nameW < 8 {
		nameW = 8
	}

	return colCfg{nameW: nameW, medium: medium, wide: wide}
}

func (m Model) visibleRows() int {
	// header(1) + separator(1) + status(1) = 3 overhead lines
	v := m.height - 3
	if v < 1 {
		v = 1
	}
	return v
}

// ── Pure helpers ──────────────────────────────────────────────────────────────

func adjustOffset(offset, cursor, visible int) int {
	if cursor < offset {
		return cursor
	}
	if cursor >= offset+visible {
		return cursor - visible + 1
	}
	return offset
}

func filterActivities(activities []models.Activity, query string) []models.Activity {
	if query == "" {
		return activities
	}
	q := strings.ToLower(query)
	var out []models.Activity
	for _, a := range activities {
		if strings.Contains(strings.ToLower(a.Name), q) ||
			strings.Contains(strings.ToLower(a.Type), q) ||
			strings.Contains(strings.ToLower(a.ID), q) {
			out = append(out, a)
		}
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

// padRight pads s to width visual characters using spaces.
// Unlike fmt.Sprintf, this works correctly with plain ASCII strings.
func padRight(s string, width int) string {
	w := len(s) // safe for ASCII names
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
