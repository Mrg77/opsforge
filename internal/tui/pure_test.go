package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mrg77/opsforge/internal/installer"
)

func TestWindowClipsAroundCursor(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	// Fits: returned unchanged.
	if got := window(lines, 0, 10); len(got) != len(lines) {
		t.Errorf("window should not clip when it fits: got %d lines", len(got))
	}

	// Cursor at top: no leading ellipsis, trailing "↓ more".
	top := window(lines, 0, 4)
	if len(top) != 4 {
		t.Fatalf("window height not honored: %d", len(top))
	}
	if strings.Contains(top[0], "more") {
		t.Errorf("no leading ellipsis expected at top: %q", top[0])
	}
	if !strings.Contains(top[len(top)-1], "↓ more") {
		t.Errorf("trailing '↓ more' expected: %q", top[len(top)-1])
	}

	// Cursor near the bottom: leading "↑ more" appears.
	bottom := window(lines, 7, 4)
	if !strings.Contains(bottom[0], "↑ more") {
		t.Errorf("leading '↑ more' expected near bottom: %q", bottom[0])
	}
}

func TestIndentPrefixesEveryLine(t *testing.T) {
	got := indent("one\ntwo", ">>")
	if got != ">>one\n>>two" {
		t.Errorf("indent = %q", got)
	}
}

func TestSelectedToolNamesSkipsHeaders(t *testing.T) {
	m := testModel(nil)
	// Select every row, including any header rows.
	for i := range m.rows {
		m.selected[i] = true
	}
	names := m.selectedToolNames()
	// Headers must be excluded; our test catalog has 4 tools.
	if len(names) != 4 {
		t.Errorf("selectedToolNames = %v, want the 4 tools (no headers)", names)
	}
	for _, n := range names {
		if n == "" {
			t.Error("a header leaked into selectedToolNames")
		}
	}
}

func TestOrderedSelectionIsRowOrder(t *testing.T) {
	m := testModel(nil)
	// Select two tools out of order; orderedSelection must return row indexes
	// ascending regardless of insertion order.
	var idxs []int
	for i, r := range m.rows {
		if r.tool.Name == "outdated-2" || r.tool.Name == "up-to-date" {
			idxs = append(idxs, i)
		}
	}
	m.selected[idxs[1]] = true
	m.selected[idxs[0]] = true
	got := m.orderedSelection()
	if len(got) != 2 || got[0] >= got[1] {
		t.Errorf("orderedSelection not ascending: %v", got)
	}
}

func TestSummaryCountsResults(t *testing.T) {
	m := testModel(nil)
	m.results = map[int]installer.Result{
		0: {},                       // ok
		1: {Err: errString("boom")}, // failed
		2: {},                       // ok
	}
	ok, failed := m.Summary()
	if ok != 2 || failed != 1 {
		t.Errorf("Summary = (%d ok, %d failed), want (2, 1)", ok, failed)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestClampCursorAfterFilterShrinksList(t *testing.T) {
	m := testModel(nil)
	// Move the cursor down, then apply a filter that leaves a single match.
	m = press(m, "j")
	m = press(m, "j")
	m.filter.SetValue("up-to-date")
	m.clampCursor()
	if n := len(m.selectable()); m.cursor >= n {
		t.Errorf("cursor %d out of range after filter (selectable=%d)", m.cursor, n)
	}
}

func TestFilterNarrowsVisibleRows(t *testing.T) {
	m := testModel(nil)
	all := len(m.visible())
	m.filter.SetValue("outdated")
	narrowed := len(m.visible())
	if narrowed >= all {
		t.Errorf("filter did not narrow rows: %d -> %d", all, narrowed)
	}
	if narrowed == 0 {
		t.Error("filter 'outdated' should still match some tools")
	}
}

func TestWindowSizeMsgSetsDimensions(t *testing.T) {
	m := testModel(nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	if m.listHeight() <= 5 {
		t.Errorf("listHeight should reflect a 40-row terminal, got %d", m.listHeight())
	}
}
