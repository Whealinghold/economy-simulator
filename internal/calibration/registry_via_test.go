package calibration

import (
	"regexp"
	"strings"
	"testing"
)

var reBacktick = regexp.MustCompile("`([^`]+)`")

// viaPath is one configuration path of the first table of VIA 11 and whether it
// moves the table ("in") or is read by a check or a report only ("chk").
type viaPath struct{ path, kind string }

// readVIA reads the first table of VIA 11, "the configuration paths the
// programme reads", as text: a path is a backticked token with a dot; a token
// that starts with a dot continues the stem of the one before.
func readVIA(via string) (paths []viaPath, unmapped []string) {
	tab := between(via, "**The configuration paths the programme reads.**", "Not read:")
	known := viaPaths()
	for _, ln := range strings.Split(tab, "\n") {
		if !strings.HasPrefix(ln, "| `") {
			continue
		}
		cs := strings.Split(strings.Trim(strings.TrimSpace(ln), "|"), "|")
		if len(cs) < 3 {
			continue
		}
		kind := viaKind(strings.TrimSpace(cs[2]))
		for _, tok := range pathTokens(cs[0]) {
			if _, ok := known[tok]; !ok || kind == "" {
				unmapped = append(unmapped, tok)
				continue
			}
			paths = append(paths, viaPath{tok, kind})
		}
	}
	return paths, unmapped
}

// viaKind reads the "Moves the table" cell: "in" for yes, "chk" for no.
func viaKind(moves string) string {
	switch {
	case strings.HasPrefix(moves, "yes") || strings.HasPrefix(moves, "the jobs"):
		return "in"
	case strings.HasPrefix(moves, "no"):
		return "chk"
	}
	return ""
}

// pathTokens returns the paths of a first cell: a token with a dot, where a token
// that starts with a dot continues the stem of the one before; a token with no dot
// (cost_share, all) is an annotation.
func pathTokens(cell string) []string {
	var out []string
	stem := ""
	for _, m := range reBacktick.FindAllStringSubmatch(cell, -1) {
		tok := m[1]
		if strings.HasPrefix(tok, ".") {
			tok = stem + tok
		} else if !strings.Contains(tok, ".") {
			continue
		}
		stem = tok[:strings.LastIndexByte(tok, '.')]
		out = append(out, tok)
	}
	return out
}

// solverKeys are the fields that name a row, and solverContainers the fields
// that only hold rows: neither is a data field the programme reads (4.2.1).
func solverKeys() map[string]bool {
	out := map[string]bool{}
	for _, k := range strings.Fields("ID Region Role Recipe Resource Season Want From To Commodity") {
		out[k] = true
	}
	return out
}

func solverContainers() map[string]bool {
	out := map[string]bool{}
	for _, k := range strings.Fields("Wants Flows Modes Nodes Lines Gates Buy Sell Resources Endowments Edges RegionPrices Items Roles Regions WarmStart Yields Tastes RegionTastes") {
		out[k] = true
	}
	return out
}

// checkSolverFlags is FR-CLB-009: `in` and `chk` are on exactly the data fields
// that a path of the first table of VIA 11 marks as moving the table or as read
// by a check or a report; a key and a container are not checked.
func checkSolverFlags(r *registry, paths []viaPath) []string {
	flags := flagMap(r)
	wantIn, wantChk := map[pair]bool{}, map[pair]bool{}
	for _, vp := range paths {
		want := wantChk
		if vp.kind == "in" {
			want = wantIn
		}
		for _, p := range viaPaths()[vp.path] {
			for _, q := range expand(p, flags) {
				want[q] = true
			}
		}
	}
	return append(flagDiffs(flags, wantIn, wantChk), unreadFields(flags, union(wantIn, wantChk))...)
}

// flagDiffs lists the data fields whose `in` or `chk` flag differs from what
// VIA 11 gives.
func flagDiffs(flags map[pair]map[string]bool, wantIn, wantChk map[pair]bool) []string {
	var out []string
	keys, conts := solverKeys(), solverContainers()
	for p, fl := range flags {
		if keys[p.field] || conts[p.field] || p.scope == "Solve option" {
			continue
		}
		if fl["in"] != wantIn[p] {
			out = append(out, "`in` on "+p.scope+"."+p.field+" disagrees with VIA 11")
		}
		if fl["chk"] != wantChk[p] {
			out = append(out, "`chk` on "+p.scope+"."+p.field+" disagrees with VIA 11")
		}
	}
	return out
}

// unreadFields lists the fields VIA 11 reads that the registry does not have.
func unreadFields(flags map[pair]map[string]bool, read map[pair]bool) []string {
	var out []string
	keys, conts := solverKeys(), solverContainers()
	for p := range read {
		if _, ok := flags[p]; !ok && !keys[p.field] && !conts[p.field] {
			out = append(out, "VIA 11 reads "+p.scope+"."+p.field+" and the registry has no such field")
		}
	}
	return out
}

// flagMap is the set of flags of each (scope, field) of the registry.
func flagMap(r *registry) map[pair]map[string]bool {
	flags := map[pair]map[string]bool{}
	set := func(p pair, fl string) {
		m := map[string]bool{}
		for _, f := range words(fl) {
			m[f] = true
		}
		flags[p] = m
	}
	for _, k := range r.knobs {
		s, f := keyParts(k[cKey])
		set(pair{s, f}, k[cFlags])
	}
	for _, g := range r.groups {
		for _, f := range strings.Split(g[gFields], ",") {
			set(pair{g[gScope], strings.TrimSpace(f)}, g[gFlags])
		}
	}
	return flags
}

// expand turns a star into every registry field of its scope.
func expand(p pair, flags map[pair]map[string]bool) []pair {
	if p.field != star {
		return []pair{p}
	}
	var out []pair
	for q := range flags {
		if q.scope == p.scope {
			out = append(out, q)
		}
	}
	return out
}

func union(a, b map[pair]bool) map[pair]bool {
	out := map[pair]bool{}
	for p := range a {
		out[p] = true
	}
	for p := range b {
		out[p] = true
	}
	return out
}

// TestRegistry_SolverFlags verifies that `in` and `chk` equal the first table of
// VIA 11: every path of the table is mapped to a registry scope and field, a
// data field it names carries the flag and no other data field does.
//
// Requirement: FR-CLB-009
func TestRegistry_SolverFlags(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	paths, unmapped := readVIA(s.via)
	if len(paths) < 40 {
		t.Fatalf("read %d paths of VIA 11, want at least 40", len(paths))
	}
	if len(unmapped) > 0 {
		t.Errorf("paths of VIA 11 with no registry mapping: %v", unmapped)
	}
	if got := checkSolverFlags(r, paths); len(got) > 0 {
		t.Errorf("checkSolverFlags(spec) = %v, want none", got)
	}
	// A data field of the table that loses its flag: CostFloor is read as a check.
	c := r.clone()
	for _, k := range c.knobs {
		if k[cKey] == "World.CostFloor" {
			k[cFlags] = "off=min"
		}
	}
	if len(checkSolverFlags(c, paths)) == 0 {
		t.Errorf("a data field without its `chk` flag did not fail")
	}
	// A data field that carries `in` and is not in the table.
	c = r.clone()
	c.knobs[len(c.knobs)-1][cFlags] += " in"
	if len(checkSolverFlags(c, paths)) == 0 {
		t.Errorf("an `in` flag that VIA 11 does not give did not fail")
	}
	// A path the map lacks is reported, not skipped.
	if _, un := readVIA(strings.Replace(s.via, "`world.cost_floor`", "`world.new_field`", 1)); len(un) == 0 {
		t.Errorf("a new path of VIA 11 was not reported as unmapped")
	}
}

// TestScopePaths_Pascal verifies the snake rule of spec-config.md 6.2 that
// scope_paths_test.go uses to respell a key of a configuration file: the
// acronyms BP and ID keep their capitals and in_progress is InProgressBP.
//
// Requirement: FR-CLB-009, FR-CLB-050
func TestScopePaths_Pascal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ in, want string }{
		{"cost_floor", "CostFloor"},
		{"belief_step", "BeliefStep"},
		{"rate_bp", "RateBP"},
		{"role_id", "RoleID"},
		{"in_progress", "InProgressBP"},
		{"expected_work_ticks", "ExpectedWorkTicks"},
	} {
		if got := pascal(tc.in); got != tc.want {
			t.Errorf("pascal(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
