package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

func testModel(rescan Rescanner) Model {
	cats := []catalog.Category{{
		Name: "Test",
		Tools: []catalog.Tool{
			{Name: "up-to-date", Bin: "a", Brew: "a"},
			{Name: "outdated-1", Bin: "b", Brew: "b"},
			{Name: "outdated-2", Bin: "c", Brew: "c"},
			{Name: "not-installed", Bin: "d", Brew: "d"},
		},
	}}
	statuses := map[string]detect.Status{
		"up-to-date":    {Installed: true, Version: "1.0"},
		"outdated-1":    {Installed: true, Version: "1.0", Outdated: true},
		"outdated-2":    {Installed: true, Version: "2.0", Outdated: true},
		"not-installed": {},
	}
	return New(cats, statuses, rescan)
}

func press(m Model, key string) Model {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return next.(Model)
}

func TestUpdateAllSelectsOnlyOutdated(t *testing.T) {
	m := press(testModel(nil), "u")
	if got := len(m.selected); got != 2 {
		t.Fatalf("u selected %d rows, want 2 (only outdated tools)", got)
	}
	for i, r := range m.rows {
		if r.isHeader() {
			continue
		}
		wantSel := r.status.Outdated
		if m.selected[i] != wantSel {
			t.Errorf("tool %q selected=%v, want %v", r.tool.Name, m.selected[i], wantSel)
		}
	}
}

func TestSelectedOutdatedShowsSelectionMarker(t *testing.T) {
	m := testModel(nil)
	// Select an outdated tool, then render; the cyan [▸] marker must
	// appear so the selection is visible despite the orange update state.
	for i, r := range m.rows {
		if r.tool.Name == "outdated-1" {
			m.selected[i] = true
		}
	}
	view := m.viewSelect()
	if !strings.Contains(view, "[▸]") {
		t.Error("selected outdated tool does not render the [▸] selection marker")
	}
	// The update-available note should still be present.
	if !strings.Contains(view, "update available") {
		t.Error("outdated note disappeared when the tool was selected")
	}
}

func TestSpaceLocksUpToDateTools(t *testing.T) {
	m := testModel(nil)
	// Cursor starts on the first selectable row: "up-to-date".
	m = press(m, " ")
	if len(m.selected) != 0 {
		t.Error("space selected an up-to-date tool; it should be locked")
	}
}

func TestBackToMenuRescansAndClears(t *testing.T) {
	rescanned := false
	rescan := func() map[string]detect.Status {
		rescanned = true
		// Simulate outdated-1 now up to date after an upgrade.
		return map[string]detect.Status{
			"up-to-date":    {Installed: true, Version: "1.0"},
			"outdated-1":    {Installed: true, Version: "1.1"},
			"outdated-2":    {Installed: true, Version: "2.0", Outdated: true},
			"not-installed": {},
		}
	}
	m := testModel(rescan)
	m.phase = phaseDone
	m.selected[1] = true

	// "m" from the done screen triggers a rescan.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m = next.(Model)
	if m.phase != phaseRescan {
		t.Fatalf("phase after 'm' = %v, want phaseRescan", m.phase)
	}
	if cmd == nil {
		t.Fatal("expected a rescan command")
	}
	// Execute the rescan command and feed its message back in.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if ok {
		// spinner tick + rescan are batched; run each until we get the rescan msg.
		for _, c := range batch {
			if rm, isRescan := c().(rescanDoneMsg); isRescan {
				msg = rm
				break
			}
		}
	}
	next, _ = m.Update(msg)
	m = next.(Model)

	if !rescanned {
		t.Error("rescanner was not invoked")
	}
	if m.phase != phaseSelect {
		t.Errorf("phase after rescan = %v, want phaseSelect", m.phase)
	}
	if len(m.selected) != 0 {
		t.Error("selection should be cleared after returning to menu")
	}
	if m.outdatedCount() != 1 {
		t.Errorf("outdatedCount after rescan = %d, want 1", m.outdatedCount())
	}
}
