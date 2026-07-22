package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme is a named palette. opsforge ships a few; the active one is
// chosen by OPSFORGE_THEME (or auto). All command styling derives from
// the active theme, so adding a theme recolors the whole product.
type Theme struct {
	Name   string
	Brand  string // accent (titles, prompt glyph)
	Blue   string // section headers
	Green  string // ok / success
	Orange string // warning / update
	Red    string // error / critical
	Cyan   string // selected / interactive
	Yellow string // highlights / versions
	Grey   string // secondary text
	Faint  string // very dim (bar background, hints)
}

// Themes are keyed by name. "auto" is resolved to a concrete theme at
// startup based on the terminal background.
// Themes use distinct accent hues so they're recognizable at a glance:
// forge=pink/orange, nord=blue, dracula=purple/magenta, gruvbox=orange/olive,
// light=inks for light terminals, mono=monochrome.
var Themes = map[string]Theme{
	"forge": { // default — hot pink brand, electric blue headers
		Name:  "forge",
		Brand: "205", Blue: "39", Green: "48", Orange: "214",
		Red: "196", Cyan: "51", Yellow: "220", Grey: "245", Faint: "238",
	},
	"nord": { // cool arctic blues throughout
		Name:  "nord",
		Brand: "111", Blue: "67", Green: "108", Orange: "180",
		Red: "167", Cyan: "116", Yellow: "222", Grey: "245", Faint: "239",
	},
	"dracula": { // purple brand, cyan headers — the signature look
		Name:  "dracula",
		Brand: "141", Blue: "117", Green: "84", Orange: "215",
		Red: "203", Cyan: "159", Yellow: "228", Grey: "244", Faint: "238",
	},
	"gruvbox": { // warm retro — burnt orange, olive green
		Name:  "gruvbox",
		Brand: "166", Blue: "109", Green: "142", Orange: "208",
		Red: "124", Cyan: "108", Yellow: "172", Grey: "245", Faint: "239",
	},
	"light": { // dark inks for light-background terminals
		Name:  "light",
		Brand: "162", Blue: "26", Green: "28", Orange: "130",
		Red: "124", Cyan: "31", Yellow: "136", Grey: "240", Faint: "250",
	},
	"mono": { // monochrome — for logs, CI, accessibility
		Name:  "mono",
		Brand: "255", Blue: "252", Green: "252", Orange: "250",
		Red: "255", Cyan: "252", Yellow: "250", Grey: "245", Faint: "240",
	},
}

// Active is the theme in use. Set at package init.
var Active Theme

func init() {
	forceColor()
	// Precedence: OPSFORGE_THEME env var wins (per-command override),
	// else the persisted choice (opsforge theme set), else auto.
	if env := os.Getenv("OPSFORGE_THEME"); env != "" {
		SetTheme(env)
		return
	}
	SetTheme(persistedTheme())
}

// themeFile is where `opsforge theme set` stores the chosen theme.
func themeFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "opsforge", "theme")
}

// ThemePersisted reports whether a theme has been saved via `theme set`.
func ThemePersisted() bool { return persistedTheme() != "" }

// persistedTheme reads the saved theme name, or "" if none.
func persistedTheme() string {
	f := themeFile()
	if f == "" {
		return ""
	}
	data, err := os.ReadFile(f)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveTheme persists a theme choice so every future opsforge command uses
// it — no env var, no `export`, no shell reload. Validates the name.
func SaveTheme(name string) error {
	if name != "auto" {
		if _, ok := Themes[name]; !ok {
			return errUnknownTheme(name)
		}
	}
	f := themeFile()
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(f, []byte(name+"\n"), 0o644); err != nil {
		return err
	}
	SetTheme(name)
	return nil
}

type errUnknownTheme string

func (e errUnknownTheme) Error() string {
	return "unknown theme " + string(e) + " (available: " + strings.Join(ThemeNames(), ", ") + ")"
}

// SetTheme selects a theme by name (case-insensitive), resolving "" and
// "auto" to a sensible default, then re-derives every style. Unknown
// names fall back to the default with no error, so a typo never breaks
// output.
func SetTheme(name string) {
	switch name {
	case "", "auto":
		Active = resolveAuto()
	default:
		if t, ok := Themes[name]; ok {
			Active = t
		} else {
			Active = Themes["forge"]
		}
	}
	applyTheme(Active)
}

// resolveAuto picks light on a light terminal background, else the
// default forge theme.
func resolveAuto() Theme {
	if lipgloss.HasDarkBackground() {
		return Themes["forge"]
	}
	return Themes["light"]
}

// ThemeNames returns the available theme names, sorted, for help/listing.
func ThemeNames() []string {
	names := make([]string, 0, len(Themes))
	for n := range Themes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// forceColor makes lipgloss emit 256-color codes even when stdout isn't a
// TTY (piped through a pager, captured, etc.), unless the user opts out
// with NO_COLOR. Without this, `opsforge … | less`/`| cat` and the '?'
// help panel (which pipes through less) would come out uncolored.
func forceColor() {
	if os.Getenv("NO_COLOR") != "" {
		return
	}
	lipgloss.SetColorProfile(termenv.ANSI256)
}

// applyTheme rebinds the exported color vars and styles to a theme.
func applyTheme(t Theme) {
	Brand = lipgloss.Color(t.Brand)
	Blue = lipgloss.Color(t.Blue)
	Green = lipgloss.Color(t.Green)
	Orange = lipgloss.Color(t.Orange)
	Red = lipgloss.Color(t.Red)
	Cyan = lipgloss.Color(t.Cyan)
	Yellow = lipgloss.Color(t.Yellow)
	Grey = lipgloss.Color(t.Grey)
	GreyDim = lipgloss.Color(t.Faint)

	Title = lipgloss.NewStyle().Bold(true).Foreground(Brand)
	Heading = lipgloss.NewStyle().Bold(true).Foreground(Blue)
	OK = lipgloss.NewStyle().Foreground(Green)
	OKBold = lipgloss.NewStyle().Bold(true).Foreground(Green)
	Warn = lipgloss.NewStyle().Foreground(Orange)
	Err = lipgloss.NewStyle().Foreground(Red)
	Selected = lipgloss.NewStyle().Foreground(Cyan)
	Accent = lipgloss.NewStyle().Foreground(Yellow)
	Dim = lipgloss.NewStyle().Foreground(Grey)
	Faint = lipgloss.NewStyle().Foreground(GreyDim)
}
