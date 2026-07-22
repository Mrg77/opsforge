package ui

import (
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
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
var Themes = map[string]Theme{
	"forge": { // default — pink accent on the classic 256-color palette
		Name:  "forge",
		Brand: "212", Blue: "39", Green: "42", Orange: "214",
		Red: "196", Cyan: "51", Yellow: "220", Grey: "241", Faint: "238",
	},
	"nord": { // cool blues, easy on the eyes
		Name:  "nord",
		Brand: "110", Blue: "111", Green: "108", Orange: "180",
		Red: "167", Cyan: "116", Yellow: "222", Grey: "245", Faint: "239",
	},
	"dracula": { // vivid, high-contrast
		Name:  "dracula",
		Brand: "212", Blue: "117", Green: "84", Orange: "215",
		Red: "203", Cyan: "159", Yellow: "228", Grey: "244", Faint: "238",
	},
	"gruvbox": { // warm, retro
		Name:  "gruvbox",
		Brand: "208", Blue: "109", Green: "142", Orange: "214",
		Red: "167", Cyan: "108", Yellow: "214", Grey: "245", Faint: "239",
	},
	"light": { // for light-background terminals (darker inks)
		Name:  "light",
		Brand: "162", Blue: "26", Green: "28", Orange: "130",
		Red: "124", Cyan: "31", Yellow: "136", Grey: "240", Faint: "250",
	},
	"mono": { // no color, just weight — for logs/CI/accessibility
		Name:  "mono",
		Brand: "255", Blue: "255", Green: "255", Orange: "255",
		Red: "255", Cyan: "255", Yellow: "255", Grey: "245", Faint: "240",
	},
}

// Active is the theme in use. Set at package init from OPSFORGE_THEME.
var Active Theme

func init() {
	SetTheme(os.Getenv("OPSFORGE_THEME"))
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
