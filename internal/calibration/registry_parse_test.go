package calibration

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The registry test is the Go port of doc/specs/models/clb/registry.py, check
// by check (plan 2-21 D622): it reads the specs as plain text, builds no World
// and runs under -short. The files below are read by relative path
// (testing.md 10.4).

// readDoc returns a file of doc/ ("specs/README.md", "rules/testing.md").
func readDoc(t testing.TB, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../doc", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// cells splits a table row on pipes that are not escaped, trims each cell and
// strips backticks, as registry.py does. The cells outside the first and the
// last pipe are returned.
func cells(line string) []string {
	line = strings.TrimSpace(line)
	var parts []string
	start := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '|' && (i == 0 || line[i-1] != '\\') {
			parts = append(parts, line[start:i])
			start = i + 1
		}
	}
	parts = append(parts, line[start:])
	if len(parts) < 2 {
		return nil
	}
	out := make([]string, 0, len(parts)-2)
	for _, p := range parts[1 : len(parts)-1] {
		out = append(out, strings.Trim(strings.TrimSpace(p), "`"))
	}
	return out
}

// registry is the parsed registry of spec-calibration.md 6: knob rows (IDs
// K-..., classes K, D, S, P, 12 cells) and structure groups (IDs S-..., 7 cells).
type registry struct {
	knobs, groups [][]string
}

// Knob row cells (4.2).
const (
	cID, cKey, cClass, cOwner, cUnit, cSearch, cGrid, cStage, cFlags, cMoves, cLoops, cLinks = 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11
)

// Group row cells (4.2).
const (
	gID, gScope, gFields, gClass, gOwner, gStage, gFlags = 0, 1, 2, 3, 4, 5, 6
)

func parseRegistry(clb string) *registry {
	r := &registry{}
	for _, line := range strings.Split(clb, "\n") {
		switch {
		case strings.HasPrefix(line, "| K-"):
			r.knobs = append(r.knobs, cells(line))
		case strings.HasPrefix(line, "| S-"):
			r.groups = append(r.groups, cells(line))
		}
	}
	return r
}

// clone returns a deep copy, so a failing fixture edits the parsed text in
// memory without touching the real registry.
func (r *registry) clone() *registry {
	c := &registry{}
	for _, k := range r.knobs {
		c.knobs = append(c.knobs, append([]string(nil), k...))
	}
	for _, g := range r.groups {
		c.groups = append(c.groups, append([]string(nil), g...))
	}
	return c
}

// words returns the space-separated tokens of a cell, none for "-".
func words(cell string) []string {
	if cell == "-" {
		return nil
	}
	return strings.Fields(cell)
}

// keyParts splits a knob key "Scope.Field" at the last dot.
func keyParts(key string) (scope, field string) {
	i := strings.LastIndexByte(key, '.')
	return key[:i], key[i+1:]
}

// specs holds the texts the registry test reads.
type specs struct {
	clb, readme, sty, stb, ind, obs, obsSpec, cfg, via string
}

func loadSpecs(t testing.TB) *specs {
	t.Helper()
	return &specs{
		clb:     readDoc(t, "specs/spec-calibration.md"),
		readme:  readDoc(t, "specs/README.md"),
		sty:     readDoc(t, "specs/spec-stylised-facts.md"),
		stb:     readDoc(t, "specs/spec-stability.md"),
		ind:     readDoc(t, "specs/spec-indicators.md"),
		obs:     readDoc(t, "rules/observability-and-logging.md"),
		obsSpec: readDoc(t, "specs/spec-observability.md"),
		cfg:     readDoc(t, "specs/spec-config.md"),
		via:     readDoc(t, "specs/spec-viability.md"),
	}
}

// between returns the text after the first from and before the first to
// following it, or "" when from is missing.
func between(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	s = s[i+len(from):]
	if j := strings.Index(s, to); j >= 0 {
		s = s[:j]
	}
	return s
}

func matchSet(re *regexp.Regexp, s string, group int) map[string]bool {
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		out[m[group]] = true
	}
	return out
}

var (
	reFact     = regexp.MustCompile(`\*\*STY-([A-Z]\d\d)\b`)
	reScenario = regexp.MustCompile(`(?m)^\| (S\d\d) \|(.*)$`)
	reLoop     = regexp.MustCompile(`\*\*STB-(L\d+)\b`)
	reDamper   = regexp.MustCompile(`(?m)^\| (D\d{2,3}) \|`)
	reInter    = regexp.MustCompile(`(?m)^\| (I\d\d) \|`)
	reAlias    = regexp.MustCompile(`(?m)^\| (A\d\d) \|`)
	reAnchor   = regexp.MustCompile(`(?m)^\| (E\d) \| anchor \|`)
	rePreset   = regexp.MustCompile("(?m)^\\| \\d+ \\| `([A-Za-z]+)` \\|")
	reEventTok = regexp.MustCompile("`([A-Z][A-Za-z]+)`")
	reNum      = regexp.MustCompile(`^-?[0-9]+$`)
)

// refs are the identifiers the registry's cells refer to.
type refs struct {
	facts, scenarios, loops, dampers, inters, aliases, anchors map[string]bool
	scenarioText                                               map[string]string
	presets                                                    map[string]bool
	presetFields                                               map[string]map[string]bool
	events                                                     map[string]bool
	obsText                                                    string
}

func loadRefs(s *specs) *refs {
	r := &refs{
		facts:        matchSet(reFact, s.sty, 1),
		loops:        matchSet(reLoop, s.stb, 1),
		dampers:      matchSet(reDamper, s.stb, 1),
		inters:       matchSet(reInter, s.clb, 1),
		aliases:      matchSet(reAlias, s.clb, 1),
		anchors:      matchSet(reAnchor, s.clb, 1),
		presets:      matchSet(rePreset, s.ind, 1),
		scenarios:    map[string]bool{},
		scenarioText: map[string]string{},
		events:       map[string]bool{},
		obsText:      s.obs + s.obsSpec,
	}
	for _, m := range reScenario.FindAllStringSubmatch(s.sty, -1) {
		r.scenarios[m[1]] = true
		r.scenarioText[m[1]] = m[2]
	}
	r.presetFields = indFields(s.ind)
	sec := between(s.obs, "## 4. Events", "## 5. Metrics")
	for _, ln := range strings.Split(sec, "\n") {
		if !strings.HasPrefix(ln, "| `") {
			continue
		}
		first := strings.Split(ln, "|")[1]
		for _, m := range reEventTok.FindAllStringSubmatch(first, -1) {
			r.events[m[1]] = true
		}
	}
	return r
}

// indFields reads the Result fields column of the table of IND 9.1: preset
// name to the set of backticked names in the fourth cell.
func indFields(ind string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	tab := between(ind, "| # | Preset | Definition | Result fields |", "\n\n")
	for _, ln := range strings.Split(tab, "\n") {
		if !strings.HasPrefix(ln, "| ") {
			continue
		}
		c := strings.Split(strings.Trim(strings.TrimSpace(ln), "|"), "|")
		if len(c) < 4 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(c[1]), "`")
		set := map[string]bool{}
		for _, m := range reEventTok.FindAllStringSubmatch(c[3], -1) {
			set[m[1]] = true
		}
		for _, m := range regexp.MustCompile("`([A-Za-z0-9]+)`").FindAllStringSubmatch(c[3], -1) {
			set[m[1]] = true
		}
		out[name] = set
	}
	return out
}

func atoi(s string) (int, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if !reNum.MatchString(s) {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	return v, err == nil
}

// counter reports whether name is an event of the observability rules or a
// word of those rules or of spec-observability.md: a counter of an event (a
// field such as Unfilled) or of the snapshot (HeldByFloor).
func (r *refs) counter(name string) bool {
	if r.events[name] {
		return true
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(r.obsText)
}
