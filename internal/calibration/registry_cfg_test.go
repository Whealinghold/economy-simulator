package calibration

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	reCatNested = regexp.MustCompile("`([A-Z][A-Za-z]+)(?:\\[\\])?` \\(((?:[^()]|\\([^()]*\\))*)\\)")
	reCatField  = regexp.MustCompile("`([A-Z][A-Za-z]+)(?:\\[\\])?`")
	reNotP      = regexp.MustCompile(`^\s*\(p\)`)
	// reRemark is a remark inside a nested table that names a type, not a field: the
	// catalogue writes `Feed[]` ... (the ids of `Item`) for the type of its fields.
	reRemark = regexp.MustCompile(`\(the ids of [^()]*\)`)
)

// catalogue is what the catalogue of spec-config.md 3.7 names: the field names,
// and the (nested table, field) pairs.
type catalogue struct {
	names  map[string]bool
	nested []pair
}

// fieldTokens returns the backticked field names of text that are not marked
// (p), presentation-only, with their offsets.
func fieldTokens(text string) (names []string, spans [][2]int) {
	for _, loc := range reCatField.FindAllStringSubmatchIndex(text, -1) {
		if reNotP.MatchString(text[loc[1]:]) {
			continue
		}
		names = append(names, text[loc[2]:loc[3]])
		spans = append(spans, [2]int{loc[0], loc[1]})
	}
	return names, spans
}

// readCatalogue parses 3.7 of spec-config.md: a nested table is `Name` or
// `Name[]` followed by a parenthesis, and its fields are the names inside it.
func readCatalogue(cfg string) catalogue {
	c := catalogue{names: map[string]bool{}}
	cat := between(cfg, "### 3.7 The catalogue of fields", "## 4. Constants and bounds")
	for _, line := range strings.Split(cat, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cs := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cs) < 2 {
			continue
		}
		text := strings.TrimSpace(cs[1])
		var nested [][2]int
		for _, loc := range reCatNested.FindAllStringSubmatchIndex(text, -1) {
			nested = append(nested, [2]int{loc[0], loc[1]})
			inner := reRemark.ReplaceAllString(text[loc[4]:loc[5]], "")
			if strings.TrimSpace(inner) == "p" {
				continue
			}
			names, _ := fieldTokens(inner)
			for _, n := range names {
				c.nested = append(c.nested, pair{text[loc[2]:loc[3]], n})
			}
		}
		names, spans := fieldTokens(text)
		for i, n := range names {
			if !inSpan(spans[i][0], nested) {
				c.names[n] = true
			}
		}
	}
	return c
}

func inSpan(at int, spans [][2]int) bool {
	for _, s := range spans {
		if s[0] <= at && at < s[1] {
			return true
		}
	}
	return false
}

func singular(n string) string {
	switch {
	case strings.HasSuffix(n, "ies"):
		return n[:len(n)-3] + "y"
	case strings.HasSuffix(n, "s"):
		return n[:len(n)-1]
	}
	return n
}

// nestedScopes names, for a table of the catalogue whose registry scopes are not
// named for it (the catalogue flattens deeper tables), the scope fragments to
// look for.
func nestedScopes() map[string][]string {
	return map[string][]string{
		"DeathSchedule": {"knot"}, "BirthSchedule": {"knot"}, "Flows": {"moneyflow", "flowtarget"},
		"Nodes": {"outside", "seasonfactor"}, "Modes": {"mode"},
		"Resources": {"resource", "abundance", "seasonregen"}, "Requires": {"tool"},
		"Wants": {"want"}, "Clearing": {"world"}, "Scale": {"group"},
	}
}

// flattened are the fields of a sub-structure that the catalogue flattens into its
// parent's parentheses: they are checked by name only.
func flattened() map[string]bool {
	out := map[string]bool{}
	for _, n := range strings.Fields("Region Role Kind Season BP Share Factor Commodity Quantity Buy Sell ID Window") {
		out[n] = true
	}
	return out
}

// checkCatalogue is FR-CLB-002, the third direction: a field name of the
// catalogue that is not presentation-only is a registry key or a group field,
// and a field of a nested table is under a scope named for that table.
func checkCatalogue(r *registry, cat catalogue) []string {
	fields := map[string][]string{}
	for p := range r.covered() {
		fields[p.field] = append(fields[p.field], p.scope)
	}
	presentation := map[string]bool{"Label": true, "UnitLabel": true, "DistanceLabel": true}
	var out []string
	for n := range cat.names {
		if len(fields[n]) == 0 && !presentation[n] {
			out = append(out, "catalogue field "+n+" is a registry key or group field of no scope")
		}
	}
	for _, p := range cat.nested {
		if !presentation[p.field] && p.field != "Items" && p.scope != "Clearing" && !nestedOK(p, fields[p.field]) {
			out = append(out, "nested catalogue field "+p.scope+"."+p.field+" is under no scope named for its table")
		}
	}
	sort.Strings(out)
	return out
}

// nestedOK reports whether the nested field p is under a registry scope named
// for its table, or, for a table the catalogue flattens, is a flattened field of
// some scope.
func nestedOK(p pair, scopes []string) bool {
	subs, listed := nestedScopes()[p.scope]
	if !listed {
		subs = []string{strings.ToLower(singular(p.scope))}
	}
	if hasScopeLike(scopes, subs) {
		return true
	}
	return listed && flattened()[p.field] && len(scopes) > 0
}

func hasScopeLike(scopes, subs []string) bool {
	for _, s := range scopes {
		for _, x := range subs {
			if strings.Contains(strings.ToLower(s), x) {
				return true
			}
		}
	}
	return false
}

// TestRegistry_CoverageCatalogue verifies that every field of the catalogue of
// spec-config.md 3.7 (a field not marked (p)) is a registry key or a group
// field, and that a field of a nested table is under a scope of that name.
//
// Requirement: FR-CLB-002
func TestRegistry_CoverageCatalogue(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	cat := readCatalogue(s.cfg)
	if len(cat.names) < 200 || len(cat.nested) < 50 {
		t.Fatalf("read %d catalogue names and %d nested fields, want at least 200 and 50", len(cat.names), len(cat.nested))
	}
	if got := checkCatalogue(r, cat); len(got) > 0 {
		t.Errorf("checkCatalogue(spec) = %v, want none", got)
	}
	// The two names N-220 found uncovered, absent from every group.
	c := r.clone()
	for _, g := range c.groups {
		g[gFields] = strings.ReplaceAll(g[gFields], "InProgressBP", "Other")
		g[gFields] = strings.ReplaceAll(g[gFields], "Days", "Other")
	}
	if got := checkCatalogue(c, cat); len(got) < 2 {
		t.Errorf("checkCatalogue without Days and InProgressBP = %v, want both reported", got)
	}
	// A nested field under a wrongly named scope.
	c = r.clone()
	for _, g := range c.groups {
		if g[gScope] == "Tool" {
			g[gScope] = "Elsewhere"
		}
	}
	if len(checkCatalogue(c, cat)) == 0 {
		t.Errorf("a nested field under a scope not named for its table did not fail")
	}
}

// TestRegistry_AgreesWithModel verifies that the parse of this test and the
// parse of doc/specs/models/clb/registry.py, whose output is registry.out, read
// the same spec: the same rows, groups, references, owner pairs and VIA 11
// paths. The script is the executable specification of the parse (plan 2-21
// D622); a count that differs means one of the two parses drifted.
//
// Guard: the Go test is a port, and a port that reads fewer rows than the model
// would pass while checking less.
func TestRegistry_AgreesWithModel(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	out := readDoc(t, "specs/models/clb/registry.out")
	r, f := parseRegistry(s.clb), loadRefs(s)
	owners, _ := ownerTable(s.readme)
	paths, _ := readVIA(s.via)
	extra := 0
	for p := range r.covered() {
		if _, ok := owners[p]; !ok {
			extra++
		}
	}
	want := map[string]int{}
	for k, re := range map[string]string{
		"knob rows": `(\d+) knob rows`, "groups": `(\d+) groups`, "presets": `IND presets (\d+)`,
		"facts": `facts (\d+)`, "scenarios": `scenarios (\d+)`, "loops": `loops (\d+)`,
		"dampers": `dampers (\d+)`, "interactions": `, I (\d+)`, "aliases": `, A (\d+)`,
		"owner pairs": `(\d+) owner pairs`, "registry-only": `(\d+) registry-only`, "via paths": `VIA 11: (\d+) paths`,
	} {
		m := regexp.MustCompile(re).FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("registry.out has no %q", k)
		}
		want[k], _ = atoi(m[1])
	}
	got := map[string]int{
		"knob rows": len(r.knobs), "groups": len(r.groups), "presets": len(f.presets),
		"facts": len(f.facts), "scenarios": len(f.scenarios), "loops": len(f.loops),
		"dampers": len(f.dampers), "interactions": len(f.inters), "aliases": len(f.aliases),
		"owner pairs": len(owners), "registry-only": extra, "via paths": len(paths),
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: the Go test reads %d, registry.out has %d", k, got[k], w)
		}
	}
}
