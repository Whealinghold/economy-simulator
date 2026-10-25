package economysim_test

import (
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the log tests that need a world (plan-phase-1-11.md Phase 10,
// spec-observability.md 9.2): every Warn of the slice at the tick its pattern
// says, the bound on its volume, its counterpart, no Info in the tick, no Error,
// the keys, the messages and the "world created" line. The rows, keys and
// messages are read from the spec and the rules by a plain text parse, as
// testing.md 6.7 does for the parameter table.

// warnRunTicks is warnRun (observer.go): the run length of the bounded Warns.
// TestWarnRun_IsTen asserts that the constant is ten.
const warnRunTicks = 10

// warnRow is one row of the table of spec-observability.md 7.3.
type warnRow struct {
	n           int
	msg         string
	keys        []string
	counterpart string
}

var (
	rowStartRe = regexp.MustCompile(`^\| (\d+) \|`)
	quotedRe   = regexp.MustCompile(`"([^"]+)"`)
)

// specSection returns the lines of the spec from the line that starts with from
// to the line that starts with to.
func specSection(t *testing.T, path, from, to string) []string {
	t.Helper()
	lines := strings.Split(readFile(t, path), "\n")
	start, end := -1, len(lines)
	for i, l := range lines {
		switch {
		case start < 0 && strings.HasPrefix(l, from):
			start = i
		case start >= 0 && strings.HasPrefix(l, to):
			end = i
			return lines[start:end]
		}
	}
	if start < 0 {
		t.Fatalf("%s has no line starting %q", path, from)
	}
	return lines[start:end]
}

// ticked returns the backticked tokens of a table cell, in order.
func ticked(cell string) []string {
	var out []string
	for i, part := range strings.Split(cell, "`") {
		if i%2 == 1 {
			out = append(out, part)
		}
	}
	return out
}

// warnRows parses the table of 7.3 into its rows by number.
func warnRows(t *testing.T) map[int]warnRow {
	t.Helper()
	rows := map[int]warnRow{}
	for _, l := range specSection(t, "doc/specs/spec-observability.md", "### 7.3", "7.3.1.") {
		m := rowStartRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		cells := strings.Split(l, "|")
		if len(cells) != 9 {
			t.Fatalf("row %s of 7.3 has %d cells, want 9: %q", m[1], len(cells), l)
		}
		var n int
		fmt.Sscan(m[1], &n)
		msg := quotedRe.FindStringSubmatch(cells[2])
		if msg == nil {
			t.Fatalf("row %d of 7.3 names no message: %q", n, cells[2])
		}
		rows[n] = warnRow{n: n, msg: msg[1], keys: ticked(cells[4]), counterpart: stripParens(cells[6])}
	}
	if len(rows) != 26 {
		t.Fatalf("parsed %d rows of 7.3, want 26", len(rows))
	}
	return rows
}

// builtWarnRows are the rows of 7.3 that the slice emits: configuration warning,
// households unfed, price beliefs at bound, work floor binding, role extinct,
// entry unfunded, group below floor, recipe output disposed, welfare record
// clamped. The test fails when another row's message is written, since the
// message set below is built from them.
var builtWarnRows = []int{1, 2, 3, 4, 6, 7, 8, 19, 26}

// knownMessages is every message the library may write: the quoted names of 7.2,
// 7.3 and 7.5 of the spec.
func knownMessages(t *testing.T) map[string]bool {
	t.Helper()
	known := map[string]bool{}
	for _, sec := range [][]string{
		specSection(t, "doc/specs/spec-observability.md", "### 7.2", "### 7.3"),
		specSection(t, "doc/specs/spec-observability.md", "### 7.3", "7.3.1."),
		specSection(t, "doc/specs/spec-observability.md", "### 7.5", "### 7.6"),
	} {
		// A message may wrap across two lines of the paragraph of 7.5.2.
		for _, m := range quotedRe.FindAllStringSubmatch(strings.Join(sec, " "), -1) {
			known[strings.Join(strings.Fields(m[1]), " ")] = true
		}
	}
	return known
}

// ruleKeys is the set of keys of the table of observability-and-logging.md 3.3.2.
func ruleKeys(t *testing.T) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	for _, l := range specSection(t, "doc/rules/observability-and-logging.md", "3.3.2.", "3.3.3.") {
		if !strings.HasPrefix(l, "| `") {
			continue
		}
		cells := strings.Split(l, "|")
		for _, k := range ticked(cells[1]) {
			keys[k] = true
		}
	}
	if len(keys) < 40 {
		t.Fatalf("parsed %d keys of the table 3.3.2, want the whole table", len(keys))
	}
	return keys
}

// cellOf names the cell of a line by the values of its identifying attributes.
func cellOf(l logLine, attrs []string) string {
	parts := make([]string, len(attrs))
	for i, a := range attrs {
		parts[i] = fmt.Sprintf("%s=%d", a, l.intAttr(a))
	}
	return strings.Join(parts, " ")
}

// warnTicksByCell groups the Warn lines of a message by cell and returns, for
// each cell, the ticks at which it was written, in order.
func warnTicksByCell(r observedRun, msg string, cell []string) map[string][]int64 {
	out := map[string][]int64{}
	for _, l := range r.warnLines(msg) {
		k := cellOf(l.logLine, cell)
		out[k] = append(out[k], l.tick)
	}
	return out
}

// startOf is the tick from which the condition of a cell holds. Every cell of the
// cases starts at start: the run of a market advances only in ticks with an
// outcome, and every market of the cases has one from the first tick (before
// N-313 the second commodity of the bound case had its first outcome at tick 2).
func (c warnCase) startOf(cell string) int64 {
	return c.start
}

// rowOf returns the row of 7.3 whose message is msg, and fails the test when it
// is not a built row.
func rowOf(t *testing.T, rows map[int]warnRow, msg string) warnRow {
	t.Helper()
	for _, r := range rows {
		if r.msg == msg && slices.Contains(builtWarnRows, r.n) {
			return r
		}
	}
	t.Fatalf("message %q is not a built row of 7.3", msg)
	return warnRow{}
}

// sameLines reports whether two runs wrote the same lines: tick, message and
// attributes.
func sameLines(a, b []stampedLine) bool {
	return slices.EqualFunc(a, b, func(x, y stampedLine) bool {
		return x.tick == y.tick && x.msg == y.msg && reflect.DeepEqual(x.attrs, y.attrs)
	})
}

// checkWarnKeys checks that every line has the keys of its row, in order.
func checkWarnKeys(t *testing.T, name string, lines []stampedLine, row warnRow) {
	t.Helper()
	for _, l := range lines {
		var keys []string
		for _, a := range l.attrs {
			keys = append(keys, a.Key)
		}
		if !slices.Equal(keys, row.keys) {
			t.Errorf("%s (seed %d): line keys %v, want the keys %v of row %d", name, obsSeed, keys, row.keys, row.n)
			return
		}
	}
}

// checkWarnTicks checks the ticks at which each cell of a case wrote its line.
func checkWarnTicks(t *testing.T, c warnCase, run observedRun) {
	t.Helper()
	if c.pattern == 0 {
		if got, want := ticksOf(run.warnLines(c.msg)), []int64{0, 1, 2}; !slices.Equal(got, want) {
			t.Errorf("%s (seed %d): %q at ticks %v, want %v (one per recipe per tick)", c.name, obsSeed, c.msg, got, want)
		}
		return
	}
	for cell, ticks := range warnTicksByCell(run, c.msg, c.cell) {
		start := c.startOf(cell)
		if want := warnTicks(c.pattern, start, c.ticks-start); !slices.Equal(ticks, want) {
			t.Errorf("%s (seed %d): cell {%s} at ticks %v, want %v (pattern %c, run from %d)", c.name, obsSeed, cell, ticks, want, c.pattern, start)
		}
	}
}

// TestWarn_EachRowAtItsPatternAndAgainAtTheNextMultipleOfTen verifies that each
// Warn of the slice that a run reaches is written at the tick its pattern says,
// with the level, the constant message and the keys of its row of 7.3, and again
// at the next multiple of 10, in a scenario in which the condition holds from a
// known tick; and that a logger with no observer writes the same Warn lines
// (D159). The bounded Warns of rows 6 to 8 are the cover of FR-SPN-036 and 047.
//
// Arithmetic: a run that begins at tick s has the run length t - s + 1 at tick t.
// Pattern B writes at run 10 and 20 (ticks s + 9 and s + 19: 9 and 19 for s = 0);
// pattern A also at run 1 (ticks 0, 9 and 19). Row 19 has no run: one line per
// recipe in each tick in which a completion overflowed (ticks 0, 1 and 2).
//
// Requirement: FR-OBS-071, FR-SPN-036, FR-SPN-047
func TestWarn_EachRowAtItsPatternAndAgainAtTheNextMultipleOfTen(t *testing.T) {
	t.Parallel()
	rows := warnRows(t)
	for _, c := range warnCases() {
		row := rowOf(t, rows, c.msg)
		both := play(t, c.scenario, who{observer: true, logger: true})
		logOnly := play(t, c.scenario, who{logger: true})
		if len(both.warnLines(c.msg)) == 0 {
			t.Fatalf("%s (seed %d): no %q line in %d ticks, want the Warn of row %d", c.name, obsSeed, c.msg, c.ticks, row.n)
		}
		if !sameLines(both.warnLines(c.msg), logOnly.warnLines(c.msg)) {
			t.Errorf("%s (seed %d): the Warn lines differ without an observer, want the same lines", c.name, obsSeed)
		}
		checkWarnKeys(t, c.name, both.warnLines(c.msg), row)
		checkWarnTicks(t, c, both)
	}
}

// ticksOf returns the ticks of the lines.
func ticksOf(lines []stampedLine) []int64 {
	out := make([]int64, len(lines))
	for i, l := range lines {
		out[i] = l.tick
	}
	return out
}

// TestWarn_NoCellWritesMoreThanOnePerRunPlusOne verifies the bound of 7.3.1: in a
// run of N ticks no cell writes more than ceil(N / warnRun) + 1 lines of one
// message (row 19 is exempt: it is bounded by the recipes), and the
// "configuration warning" lines are written once per warning, at construction.
//
// Arithmetic: 25 ticks give ceil(25 / 10) + 1 = 4 lines at most per cell; pattern
// A writes 3 (runs 1, 10 and 20), pattern B 2.
//
// Requirement: FR-OBS-072
func TestWarn_NoCellWritesMoreThanOnePerRunPlusOne(t *testing.T) {
	t.Parallel()
	for _, c := range warnCases() {
		if c.pattern == 0 {
			continue
		}
		run := play(t, c.scenario, who{logger: true})
		limit := (int(c.ticks)+warnRunTicks-1)/warnRunTicks + 1
		for cell, ticks := range warnTicksByCell(run, c.msg, c.cell) {
			if len(ticks) > limit {
				t.Errorf("%s (seed %d): cell {%s} wrote %d lines in %d ticks, want at most %d", c.name, obsSeed, cell, len(ticks), c.ticks, limit)
			}
		}
	}
	one := play(t, scenarioNamed(t, scenarios(), "one"), who{logger: true})
	if got, want := len(one.warnLines("configuration warning")), len(loadPack(t, "base.toml", "one.toml").Warnings()); got != want {
		t.Errorf("one (seed %d): %d configuration warnings written, want %d, once each", obsSeed, got, want)
	}
	for _, l := range one.warnLines("configuration warning") {
		if l.tick != -1 {
			t.Errorf("one (seed %d): a configuration warning at tick %d, want construction only", obsSeed, l.tick)
		}
	}
}

// declaredTypes is the event types and the snapshot row types by name: what the
// counterpart column of 7.3 may name.
func declaredTypes() map[string]reflect.Type {
	types := map[string]reflect.Type{}
	for _, e := range declaredEvents {
		types[typeName(e)] = reflect.TypeOf(e)
	}
	for _, v := range []any{economysim.RoleRow{}, economysim.MarketRow{}, economysim.RegionRow{}, economysim.MarketState{}} {
		types[reflect.TypeOf(v).Name()] = reflect.TypeOf(v)
	}
	return types
}

// resolves reports whether a token of a counterpart cell names something in the
// root package: the method Warnings of Config, a type, Type.Field, or a field of
// the type named last.
func resolves(tok string, types map[string]reflect.Type, cur *reflect.Type) bool {
	if tok == "Warnings" {
		_, ok := reflect.TypeOf(economysim.Config{}).MethodByName("Warnings")
		return ok
	}
	name, field, dotted := strings.Cut(tok, ".")
	ty, isType := types[name]
	if !isType {
		if *cur == nil {
			return false
		}
		_, has := (*cur).FieldByName(tok)
		return has
	}
	*cur = ty
	_, has := ty.FieldByName(field)
	return !dotted || has
}

// unresolved returns the tokens of the counterpart column of a row that do not
// resolve in the root package.
func unresolved(row warnRow) []string {
	types := declaredTypes()
	var bad []string
	var cur reflect.Type
	for _, tok := range ticked(row.counterpart) {
		if !resolves(tok, types, &cur) {
			bad = append(bad, tok)
		}
	}
	return bad
}

// counterpartCheck compares the lines of one Warn with the event or the snapshot
// that is its counterpart.
type counterpartCheck func(t *testing.T, r observedRun, lines []stampedLine, snap economysim.Snapshot)

// counterparts are the checks of the rows that a run reaches, in the order of the
// cases.
var counterparts = []struct {
	msg   string
	check counterpartCheck
}{
	{"households unfed", unfedCounterpart},
	{"price beliefs at bound", boundCounterpart},
	{"role extinct", func(t *testing.T, _ observedRun, _ []stampedLine, snap economysim.Snapshot) {
		t.Helper()
		if row := snap.Roles[0]; row.Agents != 0 {
			t.Errorf("role extinct: RoleRow.Agents = %d, want 0 (seed %d)", row.Agents, obsSeed)
		}
	}},
	{"entry unfunded", unfundedCounterpart},
	{"group below floor", func(t *testing.T, _ observedRun, lines []stampedLine, snap economysim.Snapshot) {
		t.Helper()
		if row := snap.Roles[0]; row.Agents != lines[0].intAttr("agents") {
			t.Errorf("group below floor: RoleRow.Agents = %d, line agents %d (seed %d)", row.Agents, lines[0].intAttr("agents"), obsSeed)
		}
	}},
	{"recipe output disposed", disposedCounterpart},
}

// unfedCounterpart: UpkeepShortfall at the tick and RoleRow carry the households
// and the unfed that the line reports.
func unfedCounterpart(t *testing.T, r observedRun, lines []stampedLine, snap economysim.Snapshot) {
	t.Helper()
	for _, l := range lines {
		for _, e := range eventsOf[economysim.UpkeepShortfall](r) {
			if e.Tick == l.tick && (e.Households != l.intAttr("households") || e.Unfed != l.intAttr("unfed")) {
				t.Errorf("households unfed at tick %d: UpkeepShortfall %+v, line %v (seed %d)", l.tick, e, l.attrs, obsSeed)
			}
		}
		if row := snap.Roles[0]; row.Unfed != l.intAttr("unfed") || row.Households != l.intAttr("households") {
			t.Errorf("households unfed: RoleRow %+v, line %v (seed %d), want the same Unfed and Households", row, l.attrs, obsSeed)
		}
	}
}

// boundCounterpart: BeliefsUpdated of the market and the tick has the outcomes of
// the line and, on the bound the line names, the agents of the line.
func boundCounterpart(t *testing.T, r observedRun, lines []stampedLine, _ economysim.Snapshot) {
	t.Helper()
	for _, l := range lines {
		matched := false
		for _, e := range eventsOf[economysim.BeliefsUpdated](r) {
			if e.Tick != l.tick || int64(e.Region) != l.intAttr("region") || int64(e.Commodity) != l.intAttr("commodity") {
				continue
			}
			matched = true
			onBound := e.AtFloor
			if bound, _ := l.attr("bound"); bound.String() == "ceiling" {
				onBound = e.AtCeiling
			}
			if e.Outcomes != l.intAttr("outcomes") || onBound != l.intAttr("agents") {
				t.Errorf("price beliefs at bound at tick %d: BeliefsUpdated %+v, line %v (seed %d), want the same outcomes and the count on the named bound equal to agents", l.tick, e, l.attrs, obsSeed)
			}
		}
		if !matched {
			t.Errorf("price beliefs at bound at tick %d (seed %d): no BeliefsUpdated of the market", l.tick, obsSeed)
		}
	}
}

// unfundedCounterpart: EntryLimited.Unfunded of the tick is the households of the
// line.
func unfundedCounterpart(t *testing.T, r observedRun, lines []stampedLine, _ economysim.Snapshot) {
	t.Helper()
	for _, l := range lines {
		for _, e := range eventsOf[economysim.EntryLimited](r) {
			if e.Tick == l.tick && e.Unfunded != l.intAttr("households") {
				t.Errorf("entry unfunded at tick %d: EntryLimited.Unfunded = %d, line households %d (seed %d)", l.tick, e.Unfunded, l.intAttr("households"), obsSeed)
			}
		}
	}
}

// disposedCounterpart: a RecipeExecuted of the tick has, in Overflow, the
// commodity and the quantity of the line.
func disposedCounterpart(t *testing.T, r observedRun, lines []stampedLine, _ economysim.Snapshot) {
	t.Helper()
	for _, l := range lines {
		found := false
		for _, e := range eventsOf[economysim.RecipeExecuted](r) {
			for _, o := range e.Overflow {
				found = found || (e.Tick == l.tick && int64(e.Recipe) == l.intAttr("recipe") &&
					int64(o.Commodity) == l.intAttr("commodity") && int64(o.Quantity) == l.intAttr("quantity"))
			}
		}
		if !found {
			t.Errorf("recipe output disposed at tick %d: no RecipeExecuted.Overflow with line %v (seed %d)", l.tick, l.attrs, obsSeed)
		}
	}
}

// checkConfigWarnings compares the lines "configuration warning" of the pack
// `one` with Config.Warnings, in order.
func checkConfigWarnings(t *testing.T) {
	t.Helper()
	one := play(t, scenarioNamed(t, scenarios(), "one"), who{logger: true})
	warnings := loadPack(t, "base.toml", "one.toml").Warnings()
	got := one.warnLines("configuration warning")
	if len(got) != len(warnings) {
		t.Fatalf("configuration warning: %d lines, %d Warnings() (seed %d)", len(got), len(warnings), obsSeed)
	}
	for i, w := range warnings {
		rule, _ := got[i].attr("rule")
		path, _ := got[i].attr("path")
		if rule.String() != w.Rule || path.String() != strings.Join(w.Path, ".") {
			t.Errorf("configuration warning %d = %v, want rule %q and path %q", i, got[i].attrs, w.Rule, strings.Join(w.Path, "."))
		}
	}
}

// TestWarn_EveryRowHasItsCounterpartCarryingTheNumberTheLineReports verifies the
// counterpart of each Warn the slice writes (spec-observability.md 7.4): the event
// or snapshot field of the column "Counterpart" of 7.3, read by a plain text
// parse, exists in the root package, and in the scenario it carries the number
// the line reports. The counterparts of rows 4 and 26 (RoleRow.HeldByFloor and
// ClampedHouseholds) are checked in the internal test of their scenarios.
//
// Requirement: FR-OBS-051
func TestWarn_EveryRowHasItsCounterpartCarryingTheNumberTheLineReports(t *testing.T) {
	t.Parallel()
	rows := warnRows(t)
	for _, n := range builtWarnRows {
		if bad := unresolved(rows[n]); len(bad) != 0 {
			t.Errorf("row %d of 7.3: the counterpart tokens %v do not resolve in the root package (cell %q)", n, bad, rows[n].counterpart)
		}
	}
	byMsg := map[string]warnCase{}
	for _, c := range warnCases() {
		byMsg[c.msg] = c
	}
	for _, cp := range counterparts {
		r := play(t, byMsg[cp.msg].scenario, who{observer: true, logger: true})
		lines := r.warnLines(cp.msg)
		if len(lines) == 0 {
			t.Fatalf("%s (seed %d): no line to compare", cp.msg, obsSeed)
		}
		cp.check(t, r, lines, mustSnapshot(t, r.w))
	}
	checkConfigWarnings(t)
}

// logScenarios is every scenario whose lines the log tests read: the event
// scenarios, the Warn cases and the pack `one` with its warm start.
func logScenarios() []scenario {
	list := scenarios()
	for _, c := range warnCases() {
		list = append(list, c.scenario)
	}
	return append(list, scenario{name: "warm one", ticks: 3, cfg: func(t *testing.T) economysim.Config {
		return loadPack(t, "base.toml", "one.toml", "warm-one.toml")
	}})
}

// builtWarnMessages is the messages of the built rows of 7.3 and of the Run: what
// may be written at Info or above after construction.
func builtWarnMessages(t *testing.T) map[string]bool {
	t.Helper()
	rows := warnRows(t)
	warns := map[string]bool{"run started": true, "run ended": true}
	for _, n := range builtWarnRows {
		warns[rows[n].msg] = true
	}
	return warns
}

// TestLog_NoInfoInTheTickAndNoError verifies that no record at Info or above is
// written between TickStarted and TickEnded other than a row of 7.3, that every
// such record is at Warn, and that no record at Error is written in any scenario.
// The lines of a Run ("run started", "run ended") are outside any step.
//
// Requirement: FR-OBS-070, FR-OBS-074
func TestLog_NoInfoInTheTickAndNoError(t *testing.T) {
	t.Parallel()
	warns := builtWarnMessages(t)
	for _, sc := range logScenarios() {
		for _, l := range play(t, sc, who{observer: true, logger: true}).lines {
			if why := misplaced(l, warns); why != "" {
				t.Errorf("%s (seed %d) tick %d: a record at %v %q, %s", sc.name, obsSeed, l.tick, l.level, l.msg, why)
			}
		}
	}
}

// misplaced says why a line may not be written where it was: any Error, and in a
// tick, any record at Info or above that is not a Warn row of 7.3. It returns ""
// for a line that may be.
func misplaced(l stampedLine, warns map[string]bool) string {
	switch {
	case l.level >= slog.LevelError:
		return "want no Error"
	case l.tick < 0 || l.level < slog.LevelInfo || l.msg == "run started" || l.msg == "run ended":
		return ""
	case l.level != slog.LevelWarn || !warns[l.msg]:
		return "want only the Warns of 7.3 at Info or above in the tick"
	}
	return ""
}

// TestLog_KeysAreTheKeysOfTheRulesTable verifies that every attribute key of
// every record in every scenario is a key of the table of
// observability-and-logging.md 3.3.2, read by a plain text parse, and that no
// attribute is a floating-point number (money, prices and quantities are int64).
//
// Requirement: FR-OBS-073
func TestLog_KeysAreTheKeysOfTheRulesTable(t *testing.T) {
	t.Parallel()
	keys := ruleKeys(t)
	for _, sc := range logScenarios() {
		for _, l := range play(t, sc, who{observer: true, logger: true}).lines {
			for _, a := range l.attrs {
				if !keys[a.Key] || a.Value.Kind() == slog.KindFloat64 {
					t.Errorf("%s (seed %d): line %q has the attribute %q of kind %v, want a key of the table 3.3.2 and an integer, string or bool",
						sc.name, obsSeed, l.msg, a.Key, a.Value.Kind())
				}
			}
		}
	}
}

// messageRe is a constant message: lower-case words, no digit and no value.
var messageRe = regexp.MustCompile(`^[a-z]+( [a-z]+)*$`)

// keysOfLine is the keys of a line, in order.
func keysOfLine(l logLine) []string {
	var keys []string
	for _, a := range l.attrs {
		keys = append(keys, a.Key)
	}
	return keys
}

// siteOf names the call site of a line: its message, and for "world created" the
// warm start, which adds four counts.
func siteOf(l logLine) string {
	if warm, ok := l.attr("warm_start"); ok && warm.Bool() {
		return l.msg + " with a warm start"
	}
	return l.msg
}

// TestLog_MessagesAreConstantAndInTheCatalogue verifies that every message the
// world writes is a lower-case phrase without a value, is one of the messages
// that the spec names (7.2, 7.3 and 7.5), and is written with the same keys in the
// same order at every record, so one call site writes one message.
//
// Requirement: FR-OBS-075
func TestLog_MessagesAreConstantAndInTheCatalogue(t *testing.T) {
	t.Parallel()
	known, keysOf := knownMessages(t), map[string][]string{}
	for _, sc := range logScenarios() {
		for _, l := range play(t, sc, who{observer: true, logger: true}).lines {
			if !messageRe.MatchString(l.msg) || !known[l.msg] {
				t.Errorf("%s (seed %d): message %q is not a constant message of the catalogue", sc.name, obsSeed, l.msg)
			}
			site, keys := siteOf(l.logLine), keysOfLine(l.logLine)
			if prev, ok := keysOf[site]; ok && !slices.Equal(prev, keys) {
				t.Errorf("%s (seed %d): message %q written with keys %v and %v, want one set", sc.name, obsSeed, site, prev, keys)
			}
			keysOf[site] = keys
		}
	}
	for _, msg := range []string{"world created", "households unfed", "run started", "belief updated", "market cleared", "tick ended"} {
		if _, ok := keysOf[msg]; !ok {
			t.Errorf("no scenario wrote %q (seed %d), want every message reached", msg, obsSeed)
		}
	}
}

// worldCreatedKeys is the keys of the row "world created" of 7.2, in order.
func worldCreatedKeys(t *testing.T) []string {
	t.Helper()
	for _, l := range specSection(t, "doc/specs/spec-observability.md", "### 7.2", "### 7.3") {
		if strings.HasPrefix(l, `| "world created"`) {
			return ticked(strings.Split(l, "|")[2])
		}
	}
	t.Fatalf("no row \"world created\" in 7.2")
	return nil
}

// checkWorldCreated checks the line of one scenario: the first ten keys of the row
// in order, then edges and components of the region graph (spec-regions.md 10.2,
// which the row lists after the warm start counts), and the values of the
// configuration and of the world at tick 0.
func checkWorldCreated(t *testing.T, sc scenario, specKeys []string) {
	t.Helper()
	w, _, h := build(t, sc, who{logger: true})
	var line logLine
	for _, l := range h.records() {
		if l.msg == "world created" {
			line = l
		}
	}
	want10 := append(slices.Clone(specKeys[:10]), "edges", "components")
	if i, j := slices.Index(specKeys, "edges"), slices.Index(specKeys, "components"); i < 10 || j != i+1 {
		t.Fatalf("the row \"world created\" has edges at %d and components at %d, want them in order after the first ten", i, j)
	}
	if keys := keysOfLine(line); line.level != slog.LevelInfo || !slices.Equal(keys, want10) {
		t.Fatalf("%s (seed %d): world created has the keys %v at %v, want an Info line with %v", sc.name, obsSeed, keys, line.level, want10)
	}
	cfg, snap := w.Config(), mustSnapshot(t, w)
	want := map[string]int64{
		"commodities": int64(len(cfg.Commodities)), "recipes": int64(len(cfg.Recipes)), "roles": int64(len(cfg.Roles)),
		"regions": int64(len(cfg.Regions)), "agents": snap.Agents, "households": snap.Households,
	}
	for _, k := range specKeys[2:8] {
		if got := line.intAttr(k); got != want[k] {
			t.Errorf("%s (seed %d): world created %s = %d, want %d", sc.name, obsSeed, k, got, want[k])
		}
	}
	if seed, _ := line.attr("seed"); seed.Uint64() != obsSeed {
		t.Errorf("%s: world created seed = %v, want %d", sc.name, seed, obsSeed)
	}
}

// TestLog_WorldCreatedHasTheKeysOfTheSpecWithTheScenariosValues verifies the
// Info line "world created": its keys are the first ten of the row of 7.2 (the
// keys of the modules that are on come after, and none is on), in that order,
// then edges and components, and its values are those of the configuration and of
// the world at tick 0.
//
// Requirement: FR-OBS-076
func TestLog_WorldCreatedHasTheKeysOfTheSpecWithTheScenariosValues(t *testing.T) {
	t.Parallel()
	specKeys := worldCreatedKeys(t)
	if len(specKeys) < 10 {
		t.Fatalf("parsed %d keys of the row \"world created\", want at least 10", len(specKeys))
	}
	for _, sc := range scenarios() {
		checkWorldCreated(t, sc, specKeys)
	}
}

// mustSnapshot returns the snapshot of w or fails the test.
func mustSnapshot(t *testing.T, w *economysim.World) economysim.Snapshot {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s
}

// TestLog_TheBeliefUpdateWritesNoInfoAndDebugOnly verifies that the belief
// update writes its line "belief updated" at Debug and nothing at Info or above
// (spec-beliefs.md 12.2), with a logger that reports every level enabled, and
// that a logger that reports Debug disabled receives no Debug record at all
// (the guards: the call-site allocation test of the internal tests asserts the
// cost). The line is written once per outcome, so its count equals the orders
// placed.
//
// Requirement: FR-BEL-032
func TestLog_TheBeliefUpdateWritesNoInfoAndDebugOnly(t *testing.T) {
	t.Parallel()
	sc := scenarioNamed(t, scenarios(), "bad order")
	run := play(t, sc, who{observer: true, logger: true})
	beliefs := 0
	for _, l := range run.lines {
		if strings.Contains(l.msg, "belief") {
			beliefs++
			if l.level != slog.LevelDebug {
				t.Errorf("%s (seed %d): %q at %v, want Debug", sc.name, obsSeed, l.msg, l.level)
			}
		}
	}
	if want := len(eventsOf[economysim.OrderPlaced](run)); beliefs != want || want == 0 {
		t.Errorf("%s (seed %d): %d belief lines, want one per admitted order (%d)", sc.name, obsSeed, beliefs, want)
	}
	quiet := &levelHandler{min: slog.LevelInfo}
	w, _, _ := build(t, sc, who{}, economysim.WithLogger(slog.New(quiet)))
	if _, err := w.Run(t.Context(), sc.ticks); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if want := len(sc.cfg(t).Warnings()) + 3; quiet.handled != want {
		t.Errorf("a logger at Info received %d records (seed %d), want %d: the configuration's warnings, \"world created\" and the two lines of the Run, nothing at Debug", quiet.handled, obsSeed, want)
	}
}
