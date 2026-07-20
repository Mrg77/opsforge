// Package tui implements the interactive tool picker: a categorized
// checkbox list with fuzzy-ish filtering, followed by a sequential
// installation screen with per-tool status.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
)

type phase int

const (
	phaseSelect phase = iota
	phaseInstall
	phaseDone
	phaseRescan
)

// row is one display line: either a category header or a tool entry.
type row struct {
	header string
	tool   catalog.Tool
	status detect.Status
}

func (r row) isHeader() bool { return r.header != "" }

type installDoneMsg struct {
	rowIndex int
	result   installer.Result
}

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleCategory = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleCursor   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	styleOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green: installed
	styleUpdate   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange: update available
	styleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))  // cyan: queued for install
	styleErr      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// Rescanner re-detects installed/outdated status for all tools. The TUI
// calls it after an install/upgrade batch so the list reflects the new
// reality (freshly installed tools turn green) before returning to the
// picker.
type Rescanner func() map[string]detect.Status

// Model is the Bubble Tea model for the picker.
type Model struct {
	rows     []row
	cursor   int // position within selectable()
	selected map[int]bool
	results  map[int]installer.Result

	filter    textinput.Model
	filtering bool

	spin    spinner.Model
	phase   phase
	queue   []int
	qpos    int
	rescan  Rescanner
	didWork bool // at least one install/upgrade succeeded this session

	// height is the terminal height from the last WindowSizeMsg, used
	// to window the list — the full catalog no longer fits on screen.
	height int
}

// New builds the picker from the catalog and current detection state.
// rescan may be nil, in which case returning to the menu after an
// install keeps the pre-install status (still correct, just not
// refreshed).
func New(categories []catalog.Category, statuses map[string]detect.Status, rescan Rescanner) Model {
	var rows []row
	for _, c := range categories {
		rows = append(rows, row{header: c.Name})
		for _, t := range c.Tools {
			rows = append(rows, row{tool: t, status: statuses[t.Name]})
		}
	}
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.Prompt = "/ "
	ti.CharLimit = 40
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Model{
		rows:     rows,
		selected: map[int]bool{},
		results:  map[int]installer.Result{},
		filter:   ti,
		spin:     sp,
		rescan:   rescan,
	}
}

// applyStatuses updates each tool row from a fresh detection map.
func (m *Model) applyStatuses(statuses map[string]detect.Status) {
	for i := range m.rows {
		if !m.rows[i].isHeader() {
			m.rows[i].status = statuses[m.rows[i].tool.Name]
		}
	}
}

// DidWork reports whether any install/upgrade succeeded, so the caller
// can decide to refresh shell completions on exit.
func (m Model) DidWork() bool { return m.didWork }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// visible returns row indexes to display given the current filter;
// headers appear only when at least one of their tools matches.
func (m Model) visible() []int {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	var out []int
	pendingHeader := -1
	for i, r := range m.rows {
		if r.isHeader() {
			pendingHeader = i
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(r.tool.Name), query) &&
			!strings.Contains(strings.ToLower(r.tool.Description), query) {
			continue
		}
		if pendingHeader != -1 {
			out = append(out, pendingHeader)
			pendingHeader = -1
		}
		out = append(out, i)
	}
	return out
}

// selectable returns the subset of visible rows the cursor can land on.
func (m Model) selectable() []int {
	var out []int
	for _, i := range m.visible() {
		if !m.rows[i].isHeader() {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) clampCursor() {
	if n := len(m.selectable()); m.cursor >= n {
		m.cursor = max(0, n-1)
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case installDoneMsg:
		m.results[msg.rowIndex] = msg.result
		if msg.result.Err == nil {
			m.didWork = true
		}
		m.qpos++
		if m.qpos < len(m.queue) {
			return m, m.installNext()
		}
		m.phase = phaseDone
		return m, nil
	case rescanDoneMsg:
		m.applyStatuses(msg.statuses)
		m.selected = map[int]bool{}
		m.results = map[int]installer.Result{}
		m.queue = nil
		m.phase = phaseSelect
		return m, nil
	case tea.KeyMsg:
		return m.updateKeys(msg)
	}
	return m, nil
}

// rescanDoneMsg carries a fresh detection map back to the model.
type rescanDoneMsg struct{ statuses map[string]detect.Status }

// backToMenu re-detects tool status (if a rescanner is set) and returns
// to the picker. Detection runs off the UI goroutine.
func (m Model) backToMenu() (tea.Model, tea.Cmd) {
	if m.rescan == nil {
		m.selected = map[int]bool{}
		m.results = map[int]installer.Result{}
		m.queue = nil
		m.phase = phaseSelect
		return m, nil
	}
	m.phase = phaseRescan
	rescan := m.rescan
	return m, tea.Batch(m.spin.Tick, func() tea.Msg {
		return rescanDoneMsg{statuses: rescan()}
	})
}

func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase == phaseInstall {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.phase == phaseRescan {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.phase == phaseDone {
		switch msg.String() {
		case "enter", "m":
			return m.backToMenu()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filter.SetValue("")
			m.filter.Blur()
		case "enter":
			m.filtering = false
			m.filter.Blur()
		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.clampCursor()
			return m, cmd
		}
		m.clampCursor()
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.selectable())-1 {
			m.cursor++
		}
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "u":
		// Select every outdated tool at once — the "update all" shortcut.
		for i, r := range m.rows {
			if !r.isHeader() && r.status.Outdated {
				m.selected[i] = true
			}
		}
	case "a":
		// Toggle-select every not-yet-installed tool currently visible
		// (respects the active filter), for bulk installs.
		for _, i := range m.visible() {
			if r := m.rows[i]; !r.isHeader() && !r.status.Installed {
				m.selected[i] = true
			}
		}
	case " ":
		sel := m.selectable()
		if len(sel) == 0 {
			break
		}
		i := sel[m.cursor]
		// Toggle when the tool is not installed, or installed but
		// outdated (selecting it queues an upgrade). Up-to-date tools
		// are locked — nothing to do.
		st := m.rows[i].status
		if !st.Installed || st.Outdated {
			m.selected[i] = !m.selected[i]
		}
	case "i", "enter":
		if len(m.selected) == 0 {
			break
		}
		m.queue = nil
		for _, i := range m.orderedSelection() {
			m.queue = append(m.queue, i)
		}
		m.qpos = 0
		m.phase = phaseInstall
		return m, tea.Batch(m.spin.Tick, m.installNext())
	}
	return m, nil
}

// orderedSelection returns selected row indexes in catalog order.
func (m Model) orderedSelection() []int {
	var out []int
	for i := range m.rows {
		if m.selected[i] {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) installNext() tea.Cmd {
	i := m.queue[m.qpos]
	t := m.rows[i].tool
	return func() tea.Msg {
		return installDoneMsg{rowIndex: i, result: installer.Install(t)}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.phase {
	case phaseInstall, phaseDone:
		return m.viewInstall()
	case phaseRescan:
		return styleTitle.Render("opsforge") + "\n\n" +
			m.spin.View() + " re-scanning your tools…\n"
	default:
		return m.viewSelect()
	}
}

func (m Model) viewSelect() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("opsforge — pick your tools") + "\n")
	if m.filtering || m.filter.Value() != "" {
		b.WriteString(m.filter.View() + "\n")
	}
	b.WriteString("\n")

	sel := m.selectable()
	cursorRow := -1
	if len(sel) > 0 {
		cursorRow = sel[m.cursor]
	}
	var lines []string
	cursorLine := 0
	for _, i := range m.visible() {
		r := m.rows[i]
		if r.isHeader() {
			lines = append(lines, styleCategory.Render("▸ "+r.header))
			continue
		}
		cursor := "  "
		if i == cursorRow {
			cursor = styleCursor.Render("❯ ")
			cursorLine = len(lines)
		}
		// Four visually distinct states, same glyph width so the columns
		// never shift:
		//   [✓] green   already installed, up to date
		//   [✓] orange  installed, newer version available
		//   [▸] cyan    selected for install this run
		//   [ ] grey    not installed
		var box string
		note := styleDim.Render(r.tool.Description)
		switch {
		case r.status.Outdated:
			box = styleUpdate.Render("[✓]")
			label := "update available"
			if r.status.Version != "" {
				label = r.status.Version + "  · update available"
			}
			note = styleUpdate.Render(label)
		case r.status.Installed:
			box = styleOK.Render("[✓]")
			if r.status.Version != "" {
				note = styleDim.Render(r.status.Version)
			}
		case m.selected[i]:
			box = styleSelected.Render("[▸]")
		default:
			box = "[ ]"
		}
		line := fmt.Sprintf("%s%s %-16s", cursor, box, r.tool.Name)
		lines = append(lines, line+note)
	}

	for _, l := range window(lines, cursorLine, m.listHeight()) {
		b.WriteString(l + "\n")
	}

	legend := styleOK.Render("[✓] installed") + styleHelp.Render(" · ") +
		styleUpdate.Render("[✓] update available") + styleHelp.Render(" · ") +
		styleSelected.Render("[▸] selected")
	b.WriteString("\n" + legend + "\n")

	status := fmt.Sprintf("%d selected", len(m.selected))
	if n := m.outdatedCount(); n > 0 {
		status += styleUpdate.Render(fmt.Sprintf(" · %d update(s) available — press u to select all", n))
	}
	b.WriteString(styleHelp.Render(status + "\n" +
		"space toggle · u update-all · a select-all · / filter · i install · q quit"))
	return b.String()
}

// outdatedCount counts installed tools with an available update.
func (m Model) outdatedCount() int {
	n := 0
	for _, r := range m.rows {
		if !r.isHeader() && r.status.Outdated {
			n++
		}
	}
	return n
}

// listHeight is the number of list lines that fit between the header
// and the footer; a sane default applies before the first resize event.
func (m Model) listHeight() int {
	const chrome = 5 // title + filter + blank + blank + help
	if m.height == 0 {
		return 30
	}
	return max(5, m.height-chrome)
}

// window slices lines so the cursor stays on screen, marking clipped
// content with ellipsis rows.
func window(lines []string, cursorLine, height int) []string {
	if len(lines) <= height {
		return lines
	}
	start := 0
	if cursorLine >= height-1 {
		start = min(cursorLine-height+2, len(lines)-height)
	}
	out := append([]string{}, lines[start:start+height]...)
	if start > 0 {
		out[0] = styleDim.Render("  ↑ more")
	}
	if start+height < len(lines) {
		out[len(out)-1] = styleDim.Render("  ↓ more")
	}
	return out
}

func (m Model) viewInstall() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("opsforge — installing") + "\n\n")
	var lines []string
	activeLine := 0
	for pos, i := range m.queue {
		name := m.rows[i].tool.Name
		switch {
		case m.phase == phaseInstall && pos == m.qpos:
			activeLine = len(lines)
			lines = append(lines, fmt.Sprintf("%s installing %s", m.spin.View(), name))
		case pos >= m.qpos && m.phase == phaseInstall:
			lines = append(lines, styleDim.Render(fmt.Sprintf("  queued     %s", name)))
		default:
			res := m.results[i]
			if res.Err != nil {
				lines = append(lines, styleErr.Render(fmt.Sprintf("✗ failed     %s", name)))
				lines = append(lines, styleDim.Render(indent(res.OutputTail, "    ")))
			} else {
				lines = append(lines, styleOK.Render(fmt.Sprintf("✓ installed  %s", name)))
			}
		}
	}
	if m.phase == phaseDone {
		activeLine = len(lines) - 1
	}
	for _, l := range window(lines, activeLine, m.listHeight()) {
		b.WriteString(l + "\n")
	}
	if m.phase == phaseDone {
		ok, failed := m.Summary()
		b.WriteString(fmt.Sprintf("\n%d installed, %d failed\n", ok, failed))
		b.WriteString(styleHelp.Render("enter/m back to menu · q quit"))
	}
	return b.String()
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// Summary counts successful and failed installations.
func (m Model) Summary() (ok, failed int) {
	for _, res := range m.results {
		if res.Err != nil {
			failed++
		} else {
			ok++
		}
	}
	return ok, failed
}
