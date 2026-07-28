// Package i18n is opsforge's lightweight command-line localization. It picks a
// language once (from $OPSFORGE_LANG, else $LANG/$LC_ALL) and resolves message
// keys against per-language dictionaries, falling back to English so a missing
// translation shows readable text, never a raw key.
//
// It is deliberately tiny: a flat key→string map per language and a T() lookup
// with {placeholder} substitution. English is the source of truth; French is the
// first (and, for now, only) translation. Adding a language is adding a map.
package i18n

import (
	"os"
	"strings"
)

// Lang is a supported UI language.
type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
)

// current is resolved once, at first use, from the environment.
var current = detect()

// detect resolves the language: $OPSFORGE_LANG wins (explicit opsforge choice),
// then the standard locale envs. Anything starting "fr" → French, else English.
func detect() Lang {
	for _, env := range []string{"OPSFORGE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv(env))); v != "" {
			if strings.HasPrefix(v, "fr") {
				return FR
			}
			if strings.HasPrefix(v, "en") {
				return EN
			}
		}
	}
	return EN
}

// Current returns the active language.
func Current() Lang { return current }

// SetLang overrides the detected language (used by tests and a future --lang flag).
func SetLang(l Lang) { current = l }

// dicts holds the per-language message tables. Keys live in the *.go message
// files in this package (messages_en.go / messages_fr.go).
var dicts = map[Lang]map[string]string{
	EN: en,
	FR: fr,
}

// T looks up a message by key in the current language, falling back to English,
// then to the key itself. Optional vars replace {name} placeholders.
func T(key string, vars ...map[string]string) string {
	s, ok := dicts[current][key]
	if !ok || s == "" {
		s, ok = dicts[EN][key]
		if !ok {
			return key
		}
	}
	if len(vars) > 0 {
		for k, v := range vars[0] {
			s = strings.ReplaceAll(s, "{"+k+"}", v)
		}
	}
	return s
}

// V is a tiny convenience to build the vars map inline: i18n.T("k", i18n.V("n", "3")).
func V(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}
