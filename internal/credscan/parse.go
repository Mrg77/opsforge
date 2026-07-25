package credscan

import (
	"bufio"
	"bytes"
	"strings"
	"time"
)

// iniFile is a minimal INI representation: section name -> key -> value.
// Good enough for the credential files we read (~/.aws/*, ~/.oci/config,
// ovh.conf); we deliberately don't pull an INI dependency for a format this
// small, and we never write these files.
type iniFile map[string]map[string]string

// parseINI parses INI content. Section headers are "[name]"; keys are
// "key = value" (spaces around = trimmed, inline "#"/";" comments stripped
// only when the line starts with them). Keys before any section land in the
// "" (default) section.
func parseINI(b []byte) iniFile {
	out := iniFile{"": {}}
	section := ""
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := out[section]; !ok {
				out[section] = map[string]string{}
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[section][strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// parseTime tries the date formats the various credential stores use, in
// order: RFC3339 (AWS/GCP JSON), the Python-datetime shape gcloud/Azure-legacy
// emit ("2006-01-02 15:04:05.999999", UTC assumed), and a couple of close
// variants. ok is false when nothing parses.
func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			// The Python-datetime shapes carry no zone; .UTC() normalizes both
			// them and the RFC3339 forms to a single comparable clock.
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
