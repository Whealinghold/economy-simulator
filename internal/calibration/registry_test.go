package calibration

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The closed sets of 4.2 and 4.4 are rule constants of the check, not read from
// the spec (plan 2-21 D623). The flag `d` is not in the set: it was removed
// with ADR 0008, and a fixture row that carries it must fail (FR-CLB-104).
var (
	reFlag    = regexp.MustCompile(`^(in|chk|out|tbl|z|v1x|off=(min|max|flat|-|\d+))$`)
	reMove    = regexp.MustCompile(`^([A-Z]\d\d)/([A-Za-z0-9_.]+):([+\-0~][SMHE]|\?)$`)
	reBandKey = regexp.MustCompile(`^[a-z]\d\d?\.[a-z_]+$`)
	reLin     = regexp.MustCompile(`^lin\d+$`)
	reStage   = regexp.MustCompile(`^(C[1-9]|-)$`)
	reLoopTok = regexp.MustCompile(`^L\d+$`)
	reDampTok = regexp.MustCompile(`^D\d{2,3}$`)
	reSearchE = regexp.MustCompile(`^E\d$`)
	reSearchI = regexp.MustCompile(`^I\d\d$`)
	reLinkI   = regexp.MustCompile(`^I\d\d$`)
	reLinkA   = regexp.MustCompile(`^A\d\d$`)
)

const units = " bp ticks units minor count years share name flag list "

// measureFields is the closed list of 4.4: the fields of each IND preset that a
// Moves token may name, in STY's spelling ("" is the preset alone). Each is
// cross-checked against the Result fields column of IND 9.1, so a rename in IND
// is found (D623).
func measureFields() map[string][]string {
	return map[string][]string{
		"Pinned": {"Share"}, "Turnover": {"Volume", "Velocity"}, "PriceRange": {""},
		"PriceIndex": {"Fisher"}, "Offset": {"", "abs"}, "Dispersion": {""},
		"RoleShares": {"Swing"}, "Households": {""}, "Unfed": {""},
		"WelfareDistribution": {"Mean"}, "PriceGap": {"Spread", "GapAfterFreight"},
		"StockCover": {""}, "Sectors": {"M"}, "HouseholdIncome": {""},
	}
}

// unmutated returns the names of the mutations after which check finds
// nothing: a check that cannot fail is not a check. It takes no *testing.T
// (testing.md 3.6).
func unmutated(r *registry, ms []mutation, check func(*registry) []string) []string {
	var missed []string
	for _, m := range ms {
		c := r.clone()
		m.apply(c)
		if len(check(c)) == 0 {
			missed = append(missed, m.name)
		}
	}
	return missed
}

type mutation struct {
	name  string
	apply func(*registry)
}

// setKnob returns a mutation that writes a cell of the first knob row whose
// class is cls ("" for any), and setGroup one of the first group row.
func setKnob(name string, cell int, v string) mutation {
	return mutation{name, func(r *registry) { r.knobs[0][cell] = v }}
}

func setGroup(name string, cell int, v string) mutation {
	return mutation{name, func(r *registry) { r.groups[0][cell] = v }}
}

func setClassKnob(name, cls string, cell int, v string) mutation {
	return mutation{name, func(r *registry) {
		for _, k := range r.knobs {
			if k[cClass] == cls {
				k[cell] = v
				return
			}
		}
	}}
}

// checkRows is FR-CLB-001 and FR-CLB-101: 12 and 7 cells, and a unique ID.
func checkRows(r *registry) []string {
	var out []string
	seen := map[string]bool{}
	for _, k := range r.knobs {
		if len(k) != 12 {
			out = append(out, "knob row "+k[0]+" has not 12 cells")
		}
	}
	for _, g := range r.groups {
		if len(g) != 7 {
			out = append(out, "group row "+g[0]+" has not 7 cells")
		}
	}
	for _, row := range append(slices.Clone(r.knobs), r.groups...) {
		if seen[row[0]] {
			out = append(out, "repeated ID "+row[0])
		}
		seen[row[0]] = true
	}
	return out
}

// TestRegistry_RowsAndIDs verifies that every knob row has the 12 cells and
// every structure group the 7 cells of 4.2, that no registry ID repeats, and
// that an ID retired in 6.6 (A11) is not used by a live row.
//
// Requirement: FR-CLB-001, FR-CLB-101
func TestRegistry_RowsAndIDs(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	if len(r.knobs) < 100 || len(r.groups) < 100 {
		t.Fatalf("parsed %d knob rows and %d groups, want at least 100 each", len(r.knobs), len(r.groups))
	}
	if got := checkRows(r); len(got) > 0 {
		t.Errorf("checkRows(spec) = %v, want none", got)
	}
	ms := []mutation{
		{"knob row with 11 cells", func(r *registry) { r.knobs[0] = r.knobs[0][:11] }},
		{"group row with 6 cells", func(r *registry) { r.groups[0] = r.groups[0][:6] }},
		{"repeated ID", func(r *registry) { r.knobs[1][0] = r.knobs[0][0] }},
	}
	if got := unmutated(r, ms, checkRows); len(got) > 0 {
		t.Errorf("checkRows did not fail on %v", got)
	}
	retired := between(s.clb, "### 6.6 Retired rows", "## 7. The procedure")
	for _, m := range regexp.MustCompile(`\*\*([A-Z]\d\d)\*\*`).FindAllStringSubmatch(retired, -1) {
		if strings.Contains(s.clb, "\n| "+m[1]+" |") {
			t.Errorf("retired ID %s is used by a row", m[1])
		}
	}
}

// checkClassOffFlags is FR-CLB-003, 103 and 104 with the unit, stage and
// search-kind sets: one class of K D S P, `off=` on D rows only, a flag of the
// closed set.
func checkClassOffFlags(r *registry) []string {
	var out []string
	for _, k := range r.knobs {
		out = append(out, knobProblems(k)...)
	}
	for _, g := range r.groups {
		if !strings.Contains(" K D S P ", " "+g[gClass]+" ") {
			out = append(out, g[gID]+": class "+g[gClass])
		}
		if !reStage.MatchString(g[gStage]) {
			out = append(out, g[gID]+": stage "+g[gStage])
		}
		for _, f := range words(g[gFlags]) {
			if !reFlag.MatchString(f) {
				out = append(out, g[gID]+": flag "+f)
			}
		}
	}
	return out
}

func knobProblems(k []string) []string {
	var out []string
	id, cls := k[cID], k[cClass]
	if !strings.Contains(" K D S P ", " "+cls+" ") {
		out = append(out, id+": class "+cls)
	}
	if !strings.Contains(units, " "+k[cUnit]+" ") {
		out = append(out, id+": unit "+k[cUnit])
	}
	if !reStage.MatchString(k[cStage]) {
		out = append(out, id+": stage "+k[cStage])
	}
	return append(out, offProblems(id, cls, k[cFlags])...)
}

// offProblems checks the flags of a row against the closed set and the rule
// that `off=` is on D rows only.
func offProblems(id, cls, flags string) []string {
	var out []string
	off := false
	for _, f := range words(flags) {
		if !reFlag.MatchString(f) {
			out = append(out, id+": flag "+f)
		}
		off = off || strings.HasPrefix(f, "off=")
	}
	if cls == "D" && !off {
		out = append(out, id+": a D row without off=")
	}
	if cls != "D" && off {
		out = append(out, id+": off= on a non-D row")
	}
	return out
}

// TestRegistry_ClassOffAndFlags verifies that every row has one class of K D S
// P, that `off=` is on D rows only, and that a flag is in the closed set of
// 4.2.1, which no longer holds `d`: a row that carries it fails.
//
// Requirement: FR-CLB-003, FR-CLB-103, FR-CLB-104
func TestRegistry_ClassOffAndFlags(t *testing.T) {
	t.Parallel()
	r := parseRegistry(loadSpecs(t).clb)
	if got := checkClassOffFlags(r); len(got) > 0 {
		t.Errorf("checkClassOffFlags(spec) = %v, want none", got)
	}
	ms := []mutation{
		setKnob("class X", cClass, "X"),
		setKnob("unit dollars", cUnit, "dollars"),
		setKnob("stage C10", cStage, "C10"),
		setKnob("flag d (removed by N-214)", cFlags, "d"),
		setKnob("flag q", cFlags, "in q"),
		setGroup("group flag d", gFlags, "d"),
		setGroup("group class Z", gClass, "Z"),
		setClassKnob("D row without off=", "D", cFlags, "z"),
		setClassKnob("off= on a K row", "K", cFlags, "off=0"),
	}
	if got := unmutated(r, ms, checkClassOffFlags); len(got) > 0 {
		t.Errorf("checkClassOffFlags did not fail on %v", got)
	}
}

// checkSearchGrid is FR-CLB-006 and 107: a Search cell of U, E<n> (an anchor),
// I<nn> (an interaction) or `-`, a K or D row not `-`, a grid of the closed
// forms, and a searched K or D row with no grid carries an A<nn> link.
func checkSearchGrid(r *registry, f *refs) []string {
	var out []string
	for _, k := range r.knobs {
		out = append(out, searchProblems(k, f)...)
		out = append(out, gridProblems(k)...)
	}
	return out
}

func searchProblems(k []string, f *refs) []string {
	id, cls, search := k[cID], k[cClass], k[cSearch]
	out := searchKindProblems(id, search, f)
	if (cls == "K" || cls == "D") && search == "-" {
		out = append(out, id+": a K or D row with Search -")
	}
	if (cls == "S" || cls == "P") && search != "-" && !reSearchE.MatchString(search) {
		out = append(out, id+": an S or P row with a search kind")
	}
	return out
}

func searchKindProblems(id, search string, f *refs) []string {
	switch {
	case search == "U" || search == "-":
	case reSearchE.MatchString(search):
		if !f.anchors[search] {
			return []string{id + ": " + search + " is not an anchor"}
		}
	case reSearchI.MatchString(search):
		if !f.inters[search] {
			return []string{id + ": " + search + " is not an interaction"}
		}
	default:
		return []string{id + ": search " + search}
	}
	return nil
}

func gridProblems(k []string) []string {
	id, cls, grid := k[cID], k[cClass], k[cGrid]
	var out []string
	if !(grid == "-" || grid == "geo" || reLin.MatchString(grid) || strings.HasPrefix(grid, "set:")) {
		out = append(out, id+": grid "+grid)
	}
	if (cls == "K" || cls == "D") && grid == "-" && k[cStage] != "-" && !hasA(k[cLinks]) {
		out = append(out, id+": a searched row without a grid or an A link")
	}
	return out
}

func hasA(links string) bool {
	return slices.ContainsFunc(words(links), func(w string) bool { return strings.HasPrefix(w, "A") })
}

// TestRegistry_SearchAndGrid verifies the Search and Grid cells of 4.2 and 6.1.
//
// Requirement: FR-CLB-006, FR-CLB-107
func TestRegistry_SearchAndGrid(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r, f := parseRegistry(s.clb), loadRefs(s)
	if len(f.anchors) == 0 || len(f.inters) == 0 {
		t.Fatalf("read %d anchors and %d interactions, want some", len(f.anchors), len(f.inters))
	}
	check := func(r *registry) []string { return checkSearchGrid(r, f) }
	if got := check(r); len(got) > 0 {
		t.Errorf("checkSearchGrid(spec) = %v, want none", got)
	}
	ms := []mutation{
		setKnob("search Z", cSearch, "Z"),
		setKnob("search E9 is no anchor", cSearch, "E9"),
		setKnob("search I99 is no interaction", cSearch, "I99"),
		setClassKnob("K row with Search -", "K", cSearch, "-"),
		setClassKnob("S row with Search U", "S", cSearch, "U"),
		setKnob("grid lin", cGrid, "lin"),
		{"searched row with no grid and no A link", func(r *registry) {
			k := r.knobs[0]
			k[cClass], k[cGrid], k[cStage], k[cLinks] = "K", "-", "C4", "-"
		}},
	}
	if got := unmutated(r, ms, check); len(got) > 0 {
		t.Errorf("checkSearchGrid did not fail on %v", got)
	}
}

// checkMoves is FR-CLB-007 and 105: a token of the form of 4.4, a fact of STY or
// an S<nn> row with its band key, a measure that is an IND preset field of the
// closed list (also in the Result fields column of IND 9.1), a band key of STY
// or an OBS event, and at most four tokens.
func checkMoves(r *registry, f *refs, styText string) []string {
	var out []string
	for _, k := range r.knobs {
		if k[cMoves] == "-" {
			continue
		}
		toks := strings.Split(k[cMoves], ";")
		if len(toks) > 4 {
			out = append(out, k[cID]+": more than four move tokens")
		}
		for _, tok := range toks {
			out = append(out, moveProblems(k[cID], strings.TrimSpace(tok), f, styText)...)
		}
	}
	return out
}

func moveProblems(id, tok string, f *refs, styText string) []string {
	m := reMove.FindStringSubmatch(tok)
	if m == nil {
		return []string{id + ": move " + tok}
	}
	fact, meas := m[1], m[2]
	var out []string
	switch {
	case f.facts[fact]:
	case f.scenarios[fact]:
		if !strings.Contains(f.scenarioText[fact], meas) {
			out = append(out, id+": "+fact+" does not write the band key "+meas)
		}
	default:
		out = append(out, id+": unknown fact "+fact)
	}
	preset, field, _ := strings.Cut(meas, ".")
	if allowed, ok := measureFields()[preset]; ok {
		out = append(out, presetProblems(id, meas, preset, field, allowed, f)...)
	} else if reBandKey.MatchString(meas) {
		if !strings.Contains(styText, meas) {
			out = append(out, id+": band key "+meas+" is not in STY")
		}
	} else if !f.events[meas] {
		out = append(out, id+": unknown measure "+meas)
	}
	return out
}

func presetProblems(id, meas, preset, field string, allowed []string, f *refs) []string {
	var out []string
	if !f.presets[preset] {
		out = append(out, id+": preset "+preset+" is not in IND 9.1")
	}
	if !slices.Contains(allowed, field) {
		out = append(out, id+": field of "+meas)
	}
	if field != "" && field != "abs" && !f.presetFields[preset][field] {
		out = append(out, id+": "+meas+" is not in the Result fields of IND 9.1")
	}
	return out
}

// TestRegistry_MovesTokens verifies the expected-effect tokens of 4.4.
//
// Requirement: FR-CLB-007, FR-CLB-105
func TestRegistry_MovesTokens(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r, f := parseRegistry(s.clb), loadRefs(s)
	check := func(r *registry) []string { return checkMoves(r, f, s.sty) }
	if got := check(r); len(got) > 0 {
		t.Errorf("checkMoves(spec) = %v, want none", got)
	}
	ms := []mutation{
		setKnob("no sign", cMoves, "P03/PriceRange:"),
		setKnob("unknown fact", cMoves, "P99/PriceRange:+S"),
		setKnob("unknown preset field", cMoves, "P03/Pinned.Nope:+S"),
		setKnob("unknown measure", cMoves, "P03/Nothing:+S"),
		setKnob("a field that IND does not list", cMoves, "P03/Pinned.Outcomes:+S"),
		setKnob("five tokens", cMoves, "P03/PriceRange:?; P03/PriceRange:?; P03/PriceRange:?; P03/PriceRange:?; P03/PriceRange:?"),
		setKnob("a band key not in STY", cMoves, "P03/z99.nothing_here:+S"),
	}
	if got := unmutated(r, ms, check); len(got) > 0 {
		t.Errorf("checkMoves did not fail on %v", got)
	}
	// Guard: the closed list of 4.4 is cross-checked with IND, so deleting a
	// field from the Result fields column of IND 9.1 fails the check.
	f2 := *f
	f2.presetFields = map[string]map[string]bool{"Pinned": {}}
	if len(checkMoves(r, &f2, s.sty)) == 0 {
		t.Errorf("checkMoves did not fail with the field list of IND emptied")
	}
}

// checkReferences is FR-CLB-008: a loop L<nn> and a damper D<nn> are in
// spec-stability.md, an I<nn> or A<nn> link is in 6.4 or 6.5, and the counter
// of D<nn>=<Counter> is an event of the observability rules.
func checkReferences(r *registry, f *refs) []string {
	var out []string
	for _, k := range r.knobs {
		for _, w := range words(k[cLoops]) {
			out = append(out, loopProblems(k[cID], w, f)...)
		}
		for _, w := range words(k[cLinks]) {
			if !(reLinkI.MatchString(w) && f.inters[w]) && !(reLinkA.MatchString(w) && f.aliases[w]) {
				out = append(out, k[cID]+": link "+w)
			}
		}
	}
	return out
}

func loopProblems(id, w string, f *refs) []string {
	base, counter, hasCounter := strings.Cut(w, "=")
	switch {
	case reLoopTok.MatchString(base):
		if !f.loops[base] {
			return []string{id + ": unknown loop " + base}
		}
	case reDampTok.MatchString(base):
		var out []string
		if !f.dampers[base] {
			out = append(out, id+": unknown damper "+base)
		}
		if hasCounter && !f.counter(counter) {
			out = append(out, id+": "+counter+" is no event or counter of the observability specs")
		}
		return out
	default:
		return []string{id + ": loop token " + w}
	}
	return nil
}

// TestRegistry_References verifies the Loops and Links cells of 4.2.
//
// Requirement: FR-CLB-008
func TestRegistry_References(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r, f := parseRegistry(s.clb), loadRefs(s)
	check := func(r *registry) []string { return checkReferences(r, f) }
	if got := check(r); len(got) > 0 {
		t.Errorf("checkReferences(spec) = %v, want none", got)
	}
	ms := []mutation{
		setKnob("unknown loop", cLoops, "L9999"),
		setKnob("unknown damper", cLoops, "D999"),
		setKnob("counter is no event", cLoops, "D14=NoSuchEvent"),
		setKnob("loop token", cLoops, "X1"),
		setKnob("unknown interaction", cLinks, "I99"),
		setKnob("unknown alias", cLinks, "A99"),
	}
	if got := unmutated(r, ms, check); len(got) > 0 {
		t.Errorf("checkReferences did not fail on %v", got)
	}
}

// checkNoRestatedNumber is FR-CLB-005: no Default or Bound column, and no bare
// number in a cell; a number is only in a `lin` step, a `set:` level or `off=`.
func checkNoRestatedNumber(r *registry, clb string) []string {
	var out []string
	for _, ln := range strings.Split(clb, "\n") {
		if (strings.HasPrefix(ln, "| ID | Key |") || strings.HasPrefix(ln, "| ID | Scope |")) &&
			(strings.Contains(ln, "Default") || strings.Contains(ln, "Bound")) {
			out = append(out, "a column named Default or Bound: "+ln)
		}
	}
	for _, row := range append(slices.Clone(r.knobs), r.groups...) {
		for _, c := range row {
			if reNum.MatchString(c) {
				out = append(out, row[0]+": a number "+c+" in a cell")
			}
		}
	}
	return out
}

// TestRegistry_NoRestatedNumber verifies that the registry holds no default,
// bound or calibrated value (4.5).
//
// Requirement: FR-CLB-005
func TestRegistry_NoRestatedNumber(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	if got := checkNoRestatedNumber(r, s.clb); len(got) > 0 {
		t.Errorf("checkNoRestatedNumber(spec) = %v, want none", got)
	}
	if got := checkNoRestatedNumber(r, s.clb+"\n| ID | Key | Default |\n"); len(got) == 0 {
		t.Errorf("a Default column did not fail the check")
	}
	ms := []mutation{setKnob("a bare number in the Unit cell", cUnit, "7000")}
	if got := unmutated(r, ms, func(r *registry) []string { return checkNoRestatedNumber(r, "") }); len(got) > 0 {
		t.Errorf("checkNoRestatedNumber did not fail on %v", got)
	}
}
