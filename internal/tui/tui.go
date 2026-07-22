// Package tui implements the interactive tool picker: a categorized
// checkbox list with fuzzy-ish filtering, followed by a sequential
// installation screen with per-tool status.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/ui"
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

// All picker styling comes from internal/ui, so the active theme
// (OPSFORGE_THEME or `opsforge theme set`) colors the picker exactly like
// every other command. The mapping is:
//   ui.Title    bold brand — screen title & tab cursor
//   ui.Heading  bold blue  — category headers
//   ui.OK       green      — installed / up to date
//   ui.Warn     orange     — update available
//   ui.Selected cyan       — queued for install
//   ui.Err      red        — failed
//   ui.Dim      grey       — help, notes, secondary text

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

	// Save-as-profile mode: prompts for a name, then calls saveProfile.
	saving      bool
	nameInput   textinput.Model
	saveProfile ProfileSaver
	notice      string // transient status line (e.g. "saved profile 'x'")

	// Tabs: 0=Tools (full picker), 1=Updates (outdated only),
	// 2=Security (async CVE scan of installed tools).
	tab         int
	secTargets  []audit.ToolTarget
	secFindings []audit.Finding
	secLoading  bool
	secLoaded   bool

	// height is the terminal height from the last WindowSizeMsg, used
	// to window the list — the full catalog no longer fits on screen.
	height int
}

// ProfileSaver persists a named profile from the given tool names. The
// TUI calls it when the user saves a selection; nil disables saving.
type ProfileSaver func(name string, tools []string) error

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
	ni := textinput.New()
	ni.Placeholder = "profile name"
	ni.Prompt = "save as: "
	ni.CharLimit = 40
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Model{
		rows:      rows,
		selected:  map[int]bool{},
		results:   map[int]installer.Result{},
		filter:    ti,
		nameInput: ni,
		spin:      sp,
		rescan:    rescan,
	}
}

// WithProfileSaver enables the "save selection as profile" action (key s).
func (m Model) WithProfileSaver(f ProfileSaver) Model {
	m.saveProfile = f
	return m
}

// WithSecurityTargets enables the Security tab (key 3), scanning the
// given targets against OSV when the tab is first opened.
func (m Model) WithSecurityTargets(targets []audit.ToolTarget) Model {
	m.secTargets = targets
	return m
}

// selectedToolNames returns the catalog names of currently selected tools.
func (m Model) selectedToolNames() []string {
	var names []string
	for i := range m.rows {
		if m.selected[i] && !m.rows[i].isHeader() {
			names = append(names, m.rows[i].tool.Name)
		}
	}
	return names
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

// visible returns row indexes to display given the current filter and
// tab; headers appear only when at least one of their tools matches.
func (m Model) visible() []int {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	var out []int
	pendingHeader := -1
	for i, r := range m.rows {
		if r.isHeader() {
			pendingHeader = i
			continue
		}
		if m.tab == 1 && !r.status.Outdated {
			continue // Updates tab shows only outdated tools
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
	case secScanDoneMsg:
		m.secFindings = msg.findings
		m.secLoading = false
		m.secLoaded = true
		return m, nil
	case tea.KeyMsg:
		return m.updateKeys(msg)
	}
	return m, nil
}

// secScanDoneMsg carries the security scan results back to the model.
type secScanDoneMsg struct{ findings []audit.Finding }

// sevStyle maps a CVE severity to a theme-bound ui style so the security
// tab colors findings like `opsforge audit`.
func sevStyle(s audit.Severity) lipgloss.Style {
	switch s {
	case audit.SevCritical:
		return ui.SevCritical
	case audit.SevHigh:
		return ui.SevHigh
	case audit.SevMedium:
		return ui.SevMedium
	default:
		return ui.SevLow
	}
}

// startSecurityScan fires the OSV scan for the security tab.
func (m Model) startSecurityScan() tea.Cmd {
	targets := m.secTargets
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		findings := audit.ScanTools(ctx, targets)
		sort.Slice(findings, func(a, b int) bool {
			if findings[a].TopSeverity() != findings[b].TopSeverity() {
				return findings[a].TopSeverity() > findings[b].TopSeverity()
			}
			return findings[a].Tool < findings[b].Tool
		})
		return secScanDoneMsg{findings: findings}
	}
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

	if m.saving {
		switch msg.String() {
		case "esc":
			m.saving = false
			m.nameInput.SetValue("")
			m.nameInput.Blur()
		case "enter":
			name := strings.TrimSpace(m.nameInput.Value())
			m.saving = false
			m.nameInput.Blur()
			if name != "" && m.saveProfile != nil {
				if err := m.saveProfile(name, m.selectedToolNames()); err != nil {
					m.notice = ui.Err.Render("save failed: " + err.Error())
				} else {
					m.notice = ui.OK.Render(fmt.Sprintf("saved profile '%s' (%d tools)",
						name, len(m.selectedToolNames())))
				}
			}
			m.nameInput.SetValue("")
		default:
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1":
		m.tab = 0
		m.clampCursor()
	case "2":
		m.tab = 1
		m.clampCursor()
	case "3":
		if len(m.secTargets) > 0 {
			m.tab = 2
			if !m.secLoaded && !m.secLoading {
				m.secLoading = true
				return m, tea.Batch(m.spin.Tick, m.startSecurityScan())
			}
		}
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
	case "s":
		// Save current selection as a named user profile. When there is
		// nothing selected (or saving is disabled), the key is a no-op —
		// tell the user why instead of appearing broken.
		switch {
		case m.saveProfile == nil:
			m.notice = ui.Dim.Render("saving profiles isn't available here")
		case len(m.selected) == 0:
			m.notice = ui.Warn.Render("nothing to save — select some tools first (space)")
		default:
			m.saving = true
			m.notice = ""
			return m, m.nameInput.Focus()
		}
	case "u":
		// Select every outdated tool at once — the "update all" shortcut.
		before := len(m.selected)
		for i, r := range m.rows {
			if !r.isHeader() && r.status.Outdated {
				m.selected[i] = true
			}
		}
		if len(m.selected) == before {
			m.notice = ui.OK.Render("everything is already up to date — nothing to update")
		} else {
			m.notice = ""
		}
	case "a":
		// Toggle-select every not-yet-installed tool currently visible
		// (respects the active filter), for bulk installs.
		for _, i := range m.visible() {
			if r := m.rows[i]; !r.isHeader() && !r.status.Installed {
				m.selected[i] = true
			}
		}
		m.notice = ""
	case " ":
		sel := m.selectable()
		if len(sel) == 0 {
			break
		}
		i := sel[m.cursor]
		// Toggle when the tool is not installed, or installed but
		// outdated (selecting it queues an upgrade). Up-to-date tools
		// are locked — say so rather than silently ignoring the key.
		st := m.rows[i].status
		if !st.Installed || st.Outdated {
			m.selected[i] = !m.selected[i]
			m.notice = ""
		} else {
			m.notice = ui.Dim.Render(m.rows[i].tool.Name + " is already up to date — nothing to do")
		}
	case "i", "enter":
		if len(m.selected) == 0 {
			m.notice = ui.Warn.Render("nothing selected — press space to pick tools, then i to install")
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
		return ui.Title.Render("opsforge") + "\n\n" +
			m.spin.View() + " re-scanning your tools…\n"
	default:
		if m.tab == 2 {
			return m.viewSecurity()
		}
		return m.viewSelect()
	}
}

// tabBar renders the k9s-style view switcher.
func (m Model) tabBar() string {
	names := []string{"1:Tools", "2:Updates", "3:Security"}
	if len(m.secTargets) == 0 {
		names = names[:2]
	}
	var parts []string
	for i, n := range names {
		if i == m.tab {
			parts = append(parts, ui.Title.Render("["+n+"]"))
		} else {
			parts = append(parts, ui.Dim.Render(" "+n+" "))
		}
	}
	return strings.Join(parts, ui.Dim.Render("·"))
}

// viewSecurity renders the CVE findings for installed tools.
func (m Model) viewSecurity() string {
	var b strings.Builder
	b.WriteString(ui.Title.Render("opsforge — security") + "  " + m.tabBar() + "\n\n")

	if m.secLoading {
		b.WriteString(m.spin.View() + " scanning " +
			fmt.Sprintf("%d installed tool(s) against OSV.dev…\n\n", len(m.secTargets)))
		b.WriteString(ui.Dim.Render("1/2/3 tabs · q quit"))
		return b.String()
	}

	var lines []string
	vulnerable := 0
	for _, f := range m.secFindings {
		if len(f.Vulns) == 0 {
			lines = append(lines, ui.OK.Render("✓ ")+fmt.Sprintf("%-14s", f.Tool)+
				ui.Dim.Render(f.Version+" — no known vulnerabilities"))
			continue
		}
		vulnerable++
		lines = append(lines, ui.Err.Render("⚠ ")+fmt.Sprintf("%-14s", f.Tool)+
			ui.Dim.Render(f.Version))
		for _, v := range f.Vulns {
			sev := sevStyle(v.Severity)
			fix := ""
			if v.FixedIn != "" {
				fix = ui.Dim.Render("  → fixed in " + v.FixedIn)
			}
			text := v.ID + " " + v.Summary
			if len(text) > 80 {
				text = text[:79] + "…"
			}
			lines = append(lines, "    "+sev.Render(fmt.Sprintf("[%s]", v.Severity))+" "+text+fix)
		}
	}
	for _, l := range window(lines, 0, m.listHeight()) {
		b.WriteString(l + "\n")
	}

	b.WriteString("\n")
	if vulnerable == 0 {
		b.WriteString(ui.OK.Render("All audited tools are free of known vulnerabilities.") + "\n")
	} else {
		b.WriteString(ui.Warn.Render(fmt.Sprintf("%d vulnerable tool(s) — upgrade them from the Updates tab.", vulnerable)) + "\n")
	}
	b.WriteString(ui.Dim.Render("1/2/3 tabs · q quit"))
	return b.String()
}

func (m Model) viewSelect() string {
	var b strings.Builder
	title := "opsforge — pick your tools"
	if m.tab == 1 {
		title = "opsforge — updates"
	}
	b.WriteString(ui.Title.Render(title) + "  " + m.tabBar() + "\n")
	if m.saving {
		b.WriteString(m.nameInput.View() + ui.Dim.Render("  (enter to save · esc to cancel)") + "\n")
	} else if m.filtering || m.filter.Value() != "" {
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
			lines = append(lines, ui.Heading.Render("▸ "+r.header))
			continue
		}
		cursor := "  "
		if i == cursorRow {
			cursor = ui.Title.Render("❯ ")
			cursorLine = len(lines)
		}
		// The box always reflects selection first, so toggling any row —
		// including an outdated one — gives immediate feedback. The note
		// still carries the underlying state (version, update available).
		// Marker widths match so columns never shift:
		//   [▸] cyan    checked this run (install or upgrade)
		//   [✓] green   installed, up to date
		//   [✓] orange  installed, newer version available
		//   [ ] grey    not installed
		var box string
		var note string
		switch {
		case r.status.Outdated:
			note = ui.Warn.Render("update available")
			if r.status.Version != "" {
				note = ui.Warn.Render(r.status.Version + "  · update available")
			}
		case r.status.Installed:
			note = ui.Dim.Render(r.tool.Description)
			if r.status.Version != "" {
				note = ui.Dim.Render(r.status.Version)
			}
		default:
			note = ui.Dim.Render(r.tool.Description)
		}
		switch {
		case m.selected[i]:
			box = ui.Selected.Render("[▸]")
		case r.status.Outdated:
			box = ui.Warn.Render("[✓]")
		case r.status.Installed:
			box = ui.OK.Render("[✓]")
		default:
			box = "[ ]"
		}
		line := fmt.Sprintf("%s%s %-16s", cursor, box, r.tool.Name)
		lines = append(lines, line+note)
	}

	for _, l := range window(lines, cursorLine, m.listHeight()) {
		b.WriteString(l + "\n")
	}

	legend := ui.OK.Render("[✓] installed") + ui.Dim.Render(" · ") +
		ui.Warn.Render("[✓] update available") + ui.Dim.Render(" · ") +
		ui.Selected.Render("[▸] selected")
	b.WriteString("\n" + legend + "\n")

	if m.notice != "" {
		b.WriteString(m.notice + "\n")
	}

	status := fmt.Sprintf("%d selected", len(m.selected))
	if n := m.outdatedCount(); n > 0 {
		status += ui.Warn.Render(fmt.Sprintf(" · %d update(s) available — press u to select all", n))
	}
	help := "space toggle · u update-all · a select-all · / filter · i install · 1/2/3 tabs · q quit"
	if m.saveProfile != nil {
		help = "space toggle · u update-all · a select-all · s save-profile · / filter · i install · 1/2/3 tabs · q quit"
	}
	b.WriteString(ui.Dim.Render(status + "\n" + help + "\n"))
	b.WriteString(ui.Faint.Render("Tip: " + m.tip()))
	return b.String()
}

// tips are discreet, non-obvious pointers surfaced at the bottom of the
// picker so the deeper features (security audit, secret scanning, shell
// layer) get discovered without a wall of help text.
var tips = []string{
	"press 3 to scan your installed tools for known CVEs (OSV.dev)",
	"`opsforge audit --secrets` sweeps shell history & .env files for leaked credentials",
	"`opsforge shell install` wires completions, aliases and a kube-aware prompt",
	"in the opsforge shell, `??` explains your last failed command with AI",
}

// tip rotates through tips so a different one shows each render tick; it
// keys off the selection count and outdated count to feel non-static
// without needing extra state.
func (m Model) tip() string {
	return tips[(len(m.selected)+m.outdatedCount())%len(tips)]
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
		out[0] = ui.Dim.Render("  ↑ more")
	}
	if start+height < len(lines) {
		out[len(out)-1] = ui.Dim.Render("  ↓ more")
	}
	return out
}

func (m Model) viewInstall() string {
	var b strings.Builder
	b.WriteString(ui.Title.Render("opsforge — installing") + "\n\n")
	var lines []string
	activeLine := 0
	for pos, i := range m.queue {
		name := m.rows[i].tool.Name
		switch {
		case m.phase == phaseInstall && pos == m.qpos:
			activeLine = len(lines)
			lines = append(lines, fmt.Sprintf("%s installing %s", m.spin.View(), name))
		case pos >= m.qpos && m.phase == phaseInstall:
			lines = append(lines, ui.Dim.Render(fmt.Sprintf("  queued     %s", name)))
		default:
			res := m.results[i]
			if res.Err != nil {
				lines = append(lines, ui.Err.Render(fmt.Sprintf("✗ failed     %s", name)))
				lines = append(lines, ui.Dim.Render(indent(res.OutputTail, "    ")))
			} else {
				lines = append(lines, ui.OK.Render(fmt.Sprintf("✓ installed  %s", name)))
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
		b.WriteString(ui.Dim.Render("enter/m back to menu · q quit"))
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
