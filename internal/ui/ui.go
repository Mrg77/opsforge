// Package ui is opsforge's shared visual identity: one palette, one set
// of status markers, and reusable header/section helpers so every command
// looks like the same product. Import this instead of hand-rolling
// lipgloss styles in each command.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Palette and styles — the single source of truth for opsforge colors.
// These are bound by the active theme (see theme.go, applyTheme); do not
// initialize them here, or the theme selected at init() would be
// overwritten by these declarations.
var (
	Brand   lipgloss.Color
	Blue    lipgloss.Color
	Green   lipgloss.Color
	Orange  lipgloss.Color
	Red     lipgloss.Color
	Cyan    lipgloss.Color
	Yellow  lipgloss.Color
	Grey    lipgloss.Color
	GreyDim lipgloss.Color

	Title    lipgloss.Style
	Heading  lipgloss.Style
	OK       lipgloss.Style
	OKBold   lipgloss.Style
	Warn     lipgloss.Style
	Err      lipgloss.Style
	Selected lipgloss.Style
	Accent   lipgloss.Style
	Dim      lipgloss.Style
	Faint    lipgloss.Style
)

// Status markers — used identically across list, profiles, doctor, audit.
const (
	MarkOK      = "✓"
	MarkUpdate  = "↑"
	MarkErr     = "✗"
	MarkWarn    = "⚠"
	MarkMissing = "·"
	MarkSel     = "▸"
	MarkArrow   = "→"
	Prompt      = "❯"
)

// OKMark / WarnMark / ErrMark / MissMark return a colored status glyph.
func OKMark() string   { return OK.Render(MarkOK) }
func WarnMark() string { return Warn.Render(MarkWarn) }
func ErrMark() string  { return Err.Render(MarkErr) }
func MissMark() string { return Dim.Render(MarkMissing) }

// Check renders a green ✓ when ok, a dim · otherwise — the doctor idiom.
func Check(ok bool) string {
	if ok {
		return OKMark()
	}
	return MissMark()
}

// Header prints a framed section header: a rule, the title (with the
// opsforge prompt glyph), an optional subtitle, and a closing rule. This
// is the signature opsforge block, reused by every command's header.
func Header(title, subtitle string) string {
	width := clampWidth(Width())
	rule := lipgloss.NewStyle().Foreground(Blue).Render(strings.Repeat("─", width))
	var b strings.Builder
	b.WriteString(rule + "\n")
	b.WriteString(Title.Render(fmt.Sprintf("  %s %s", Prompt, title)) + "\n")
	if subtitle != "" {
		b.WriteString(Dim.Render("  "+subtitle) + "\n")
	}
	b.WriteString(rule)
	return b.String()
}

// Section is a lighter, inline heading (no frame) for sub-parts.
func Section(name string) string {
	return Heading.Render(name)
}

// Bar renders a filled/empty progress bar, e.g. ███████░░░.
func Bar(done, total, width int) string {
	if total <= 0 {
		return Faint.Render(strings.Repeat("░", width))
	}
	filled := done * width / total
	if filled > width {
		filled = width
	}
	return OK.Render(strings.Repeat("█", filled)) +
		Faint.Render(strings.Repeat("░", width-filled))
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink; terminals that
// support it render a clickable link, others show the text unchanged.
func Hyperlink(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// Width returns the terminal width, defaulting to 80 when it can't be
// detected (piped output, no tty).
func Width() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func clampWidth(w int) int {
	if w > 100 {
		return 100
	}
	if w < 20 {
		return 20
	}
	return w
}
