package economysim

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// This file is the per-position state hash of testing.md 7.1.2 (plan 1-12,
// phase 4, D174, D177): a reflection walk of the whole World and a replay of a
// step through the unexported position methods. It holds no pack, imports no
// config/ and calls no golden (A3); the external poshash_test.go does.

// skipFields are the struct fields the walk leaves out, by name (D177). Each is
// either not state or depends on whether an observer, a logger or the audit is
// attached, so a bare and an observed world hash alike.
var skipFields = []string{
	// World.obs holds the registered observers and the "supplied" flag.
	"World.obs",
	// World.log is the injected *slog.Logger.
	"World.log",
	// World.audit is the WithAudit flag.
	"World.audit",
	// World.stepping is the in-step flag: not state, not saved (spec-world.md 3.6.2).
	"World.stepping",
	// tickScratch.flows, boundary and wear are the buffers of the bucket events,
	// filled only when an observer is attached (phase_start.go).
	"tickScratch.flows", "tickScratch.boundary", "tickScratch.wear",
	// productionTick.cells and outcomeTick.short are per-tick counts kept for the
	// events, cleared at the start of the next use, and counted only when observed.
	"productionTick.cells", "outcomeTick.short",
	// productionTick.skips counts the households the halt skipped, for the events.
	"productionTick.skips",
	// Decisions.hysteresis is counted only when observed (spawner.go noteHysteresis),
	// and entryTally.rows holds the per-region counts of the entry events.
	"Decisions.hysteresis", "entryTally.rows",
	// refusalTally.counts and dirty hold the refusals counted for the P2.8a rows,
	// counted only when observed and cleared when emitted.
	"refusalTally.counts", "refusalTally.dirty",
	// exitTick.view is the spawner's view, rebuilt for each region at P7.1 and by
	// Snapshot; it copies "an observer is attached" and the tick of its last use.
	"exitTick.view",
	// Snapshot writes scratch, so a world snapshotted between steps must hash like
	// one that was not: World.lotBuf is its lot scan (world.go), estates.lots the
	// lots of the account being read, exitTick.sets, sums and unskilled the role
	// summaries it fills through fillRoles. All are rebuilt before each use.
	"World.lotBuf", "estates.lots", "exitTick.sets", "exitTick.sums", "exitTick.unskilled",
	// The region graph and the table of effective reference prices are tables derived
	// from cfg and the edge statuses (regions.go): the world's value is in cfg and in
	// regionState.status, which the walk does hash and which is nil while every status
	// is the default. A derived table that existed before the golden was recorded is in
	// the walk; one appended after it must not move the off files (testing.md 7.1.1).
	"World.refs", "regionState.g",
	// regionState.rebuilds counts the rebuilds of that table after construction, for
	// the test of the one-rebuild rule; a restored world builds it once, so the count
	// is not state and is not saved.
	"regionState.rebuilds",
	// viewCtx.refs is World.refs again, assembled with the rest of the view context
	// at the start of P2.5 and P3.2 (market.go refreshViewCtx).
	"viewCtx.refs",
}

// maxWalkDepth bounds the nesting of the walk; a deeper value is an error.
const maxWalkDepth = 64

// hasher is one walk. seen gives a pointer or map its first-visit index, so a
// cycle ends and an alias is told from a copy.
type hasher struct {
	buf  []byte
	seen map[seenKey]int
	skip map[string]bool
	mask map[reflect.Type]*fieldInfo
	err  error
}

type seenKey struct {
	t reflect.Type
	p uintptr
}

func newHasher(extraSkip ...string) *hasher {
	h := &hasher{seen: map[seenKey]int{}, skip: map[string]bool{}, mask: map[reflect.Type]*fieldInfo{}}
	for _, s := range append(slices.Clone(skipFields), extraSkip...) {
		h.skip[s] = true
	}
	return h
}

// zero reports whether v is at its zero value in the sense of D177: a nil
// pointer or interface, an empty slice, map or string, 0, false, and a struct
// or array whose every walked field is zero. It never recurses through a
// pointer, so it terminates on a cycle.
func (h *hasher) zero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		fi := h.info(v.Type())
		for i := 0; i < v.NumField(); i++ {
			if !fi.skip[i] && !h.zero(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !h.zero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	}
	return true // func, chan, unsafe.Pointer: not walked
}

// fieldInfo is what the walk needs of a struct type, read once: the fields in
// name order and which are skipped (skip list entries are "Type.field").
type fieldInfo struct {
	order []int
	names []string
	skip  []bool
}

func (h *hasher) info(t reflect.Type) *fieldInfo {
	if fi, ok := h.mask[t]; ok {
		return fi
	}
	fi := &fieldInfo{names: make([]string, t.NumField()), skip: make([]bool, t.NumField())}
	for i := range fi.names {
		fi.names[i] = t.Field(i).Name
		fi.skip[i] = h.skip[t.Name()+"."+fi.names[i]]
		fi.order = append(fi.order, i)
	}
	slices.SortFunc(fi.order, func(a, b int) int { return strings.Compare(fi.names[a], fi.names[b]) })
	h.mask[t] = fi
	return fi
}

func (h *hasher) fail(format string, args ...any) {
	if h.err == nil {
		h.err = fmt.Errorf(format, args...)
	}
}

// write appends the canonical encoding of v. It reads values only; it never
// calls Interface, which an unexported field forbids.
func (h *hasher) write(v reflect.Value, depth int) {
	if h.err != nil {
		return
	}
	if depth > maxWalkDepth {
		h.fail("walk deeper than %d at %s", maxWalkDepth, v.Type())
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		h.buf = strconv.AppendBool(h.buf, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		h.buf = strconv.AppendInt(h.buf, v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		h.buf = strconv.AppendUint(h.buf, v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		h.buf = binary.BigEndian.AppendUint64(h.buf, math.Float64bits(v.Float()))
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		h.buf = binary.BigEndian.AppendUint64(h.buf, math.Float64bits(real(c)))
		h.buf = binary.BigEndian.AppendUint64(h.buf, math.Float64bits(imag(c)))
	case reflect.String:
		h.buf = strconv.AppendQuote(h.buf, v.String())
	case reflect.Pointer:
		h.pointer(v, depth)
	case reflect.Interface:
		if v.IsNil() {
			h.buf = append(h.buf, "nil"...)
			return
		}
		h.buf = append(h.buf, v.Elem().Type().String()...)
		h.buf = append(h.buf, ':')
		h.write(v.Elem(), depth+1)
	case reflect.Struct:
		h.structure(v, depth)
	case reflect.Array, reflect.Slice:
		h.list(v, depth)
	case reflect.Map:
		h.mapping(v, depth)
	default: // func, chan, unsafe.Pointer
	}
	h.buf = append(h.buf, ';')
}

func (h *hasher) pointer(v reflect.Value, depth int) {
	if v.IsNil() {
		h.buf = append(h.buf, "nil"...)
		return
	}
	k := seenKey{v.Type(), v.Pointer()}
	if i, ok := h.seen[k]; ok {
		h.buf = append(h.buf, '&')
		h.buf = strconv.AppendInt(h.buf, int64(i), 10)
		return
	}
	h.seen[k] = len(h.seen)
	h.buf = append(h.buf, '&')
	h.buf = strconv.AppendInt(h.buf, int64(h.seen[k]), 10)
	h.write(v.Elem(), depth+1)
}

// structure writes the fields keyed by name, sorted by name, leaving out the
// skipped and the zero ones (D177 off-invariance).
func (h *hasher) structure(v reflect.Value, depth int) {
	fi := h.info(v.Type())
	h.buf = append(h.buf, '{')
	for _, i := range fi.order {
		if fi.skip[i] || h.zero(v.Field(i)) {
			continue
		}
		h.buf = append(h.buf, fi.names[i]...)
		h.buf = append(h.buf, '=')
		h.write(v.Field(i), depth+1)
	}
	h.buf = append(h.buf, '}')
}

func (h *hasher) list(v reflect.Value, depth int) {
	h.buf = append(h.buf, '[')
	h.buf = strconv.AppendInt(h.buf, int64(v.Len()), 10)
	h.buf = append(h.buf, ':')
	for i := 0; i < v.Len(); i++ {
		h.write(v.Index(i), depth+1)
	}
	h.buf = append(h.buf, ']')
}

// mapping writes the entries sorted by the encoding of the key.
func (h *hasher) mapping(v reflect.Value, depth int) {
	if v.IsNil() {
		return
	}
	k := seenKey{v.Type(), v.Pointer()}
	if i, ok := h.seen[k]; ok {
		h.buf = append(h.buf, '&')
		h.buf = strconv.AppendInt(h.buf, int64(i), 10)
		return
	}
	h.seen[k] = len(h.seen)
	type entry struct {
		key []byte
		val reflect.Value
	}
	entries := make([]entry, 0, v.Len())
	outer := h.buf
	for it := v.MapRange(); it.Next(); {
		h.buf = nil
		h.write(it.Key(), depth+1)
		entries = append(entries, entry{h.buf, it.Value()})
	}
	h.buf = outer
	slices.SortFunc(entries, func(a, b entry) int { return bytes.Compare(a.key, b.key) })
	h.buf = append(h.buf, '{')
	for _, e := range entries {
		h.buf = append(h.buf, e.key...)
		h.buf = append(h.buf, '=')
		h.write(e.val, depth+1)
	}
	h.buf = append(h.buf, '}')
}

// hashValue is the SHA-256 of the canonical encoding of v, or the error of a
// walk that went deeper than maxWalkDepth.
func hashValue(v reflect.Value, extraSkip ...string) ([32]byte, error) {
	h := newHasher(extraSkip...)
	h.write(v, 0)
	if h.err != nil {
		return [32]byte{}, h.err
	}
	return sha256.Sum256(h.buf), nil
}

// hashWorld is the state hash of D177: a walk of the whole World value graph. A
// walk that goes too deep panics, which fails the calling test.
func hashWorld(w *World) [32]byte {
	sum, err := hashValue(reflect.ValueOf(w))
	if err != nil {
		panic(fmt.Sprintf("hashWorld: %v", err))
	}
	return sum
}

// replayRow is one call of the replay: the position it records under, how to
// run it, and the positions of positionList it stands for beyond its own id
// (the pseudo-rows). on, when set, is the module guard: a row whose guard is
// not taken records nothing (D177).
type replayRow struct {
	id     string
	covers []string
	run    func(w *World) error
	on     func(w *World) bool
}

// demandOn is the guard of the rows of the demand module (D177).
func demandOn(w *World) bool { return w.demand != nil }

// moneyOn is the guard of the row of the money supply module: an off world
// records no P1.7 line, so the off position hashes do not move (plan 2-6 D331).
func moneyOn(w *World) bool { return w.money != nil }

// noErr adapts a position method with no error result.
func noErr(f func(*World)) func(*World) error {
	return func(w *World) error { f(w); return nil }
}

// replayRows is the row-to-method map: begin, the straight line of tick.go in
// its order, then end. A new position joins by one entry here (and its guard).
// p84Audit is not here: it is a row of its own that replayStep runs last and
// whose error goes to endStep, as Step does.
var replayRows = []replayRow{
	{id: "begin", covers: []string{"P0.1", "P0.2a", "P0.3"}, run: (*World).beginStep},
	{id: "P0.2", run: noErr((*World).p02)},
	{id: "P1.1", run: (*World).p11Moment},
	{id: "P1.2", run: noErr((*World).p12Anchors)},
	{id: "P1.7", run: noErr((*World).p17Flows), on: moneyOn},
	{id: "P2.1", run: noErr((*World).p21Complete)},
	{id: "P2.5", run: (*World).p25Work},
	{id: "P2.7", run: (*World).p27Start},
	{id: "P2.8a", run: noErr((*World).p28aHalted)},
	{id: "P3.2", run: (*World).p32Trade},
	{id: "P3.4", run: (*World).p34Estates},
	{id: "P3.6", run: noErr((*World).p36Aggregates), on: demandOn},
	{id: "P4.1", run: (*World).p41Clear},
	{id: "P5.1", run: noErr((*World).p51Window)},
	{id: "P5.3", run: (*World).p53Markdown},
	{id: "P5.4", run: noErr((*World).p54Outcomes)},
	{id: "P6.1", run: (*World).p61Upkeep},
	{id: "P6.2", run: (*World).p62Wants, on: demandOn},
	{id: "P6.3", run: noErr((*World).p63Welfare)},
	{id: "P6.4", run: noErr((*World).p64Shortfall)},
	{id: "P7.1", run: (*World).p71Entry},
	{id: "P8.1", run: (*World).p81Age},
	{id: "P8.2", run: noErr((*World).p82Flows)},
}

// endCovers are the positions of positionList that run inside endStep.
var endCovers = []string{"P8.5", "P8.6"}

// checkReplayRows is the guard of D174: every position of positionList is a row
// of the map or covered by a pseudo-row, and every row is in positionList. It
// returns the first mismatch, naming the position.
func checkReplayRows() error { return checkRows(replayRows) }

// checkRows is checkReplayRows over a given map, so a test can pass a broken one.
func checkRows(rows []replayRow) error {
	want := map[string]bool{}
	for _, p := range positionList {
		want[p.id] = true
	}
	have := map[string]bool{"P8.4": true}
	for _, r := range rows {
		ids := append([]string{r.id}, r.covers...)
		if r.id == "begin" {
			ids = r.covers
		}
		for _, id := range ids {
			if !want[id] {
				return fmt.Errorf("replay row %q is not a position of tick.go", id)
			}
			have[id] = true
		}
	}
	for _, id := range endCovers {
		have[id] = true
	}
	for _, p := range positionList {
		if !have[p.id] {
			return fmt.Errorf("position %s of tick.go has no method in the replay map", p.id)
		}
	}
	return nil
}

// replayStep runs one step as Step does, through the position methods, and
// calls record(position) after each position that ran, with the pseudo-rows
// "begin" and "end" for the positions inside beginStep and endStep. It stops at
// the first error of a position (the audit excepted: its error goes to endStep
// and is returned after "end"), naming tick and position. After it the stepping
// flag is clear (endStep clears it).
func replayStep(w *World, record func(position string)) error {
	if err := checkReplayRows(); err != nil {
		return err
	}
	tick := w.Tick()
	if err := runRows(w, tick, replayRows, record); err != nil {
		return err
	}
	err := w.p84Audit()
	record("P8.4")
	err = w.endStep(err)
	record("end")
	if err != nil {
		return fmt.Errorf("tick %d position P8.4: %w", tick, err)
	}
	return nil
}

// runRows runs the rows in order, recording each after it ran, and stops at the
// first error naming tick and position. It is the loop of replayStep, apart so a
// test can run it over rows of its own.
func runRows(w *World, tick int64, rows []replayRow, record func(position string)) error {
	for _, r := range rows {
		if r.on != nil && !r.on(w) {
			continue
		}
		if err := r.run(w); err != nil {
			return fmt.Errorf("tick %d position %s: %w", tick, r.id, err)
		}
		record(r.id)
	}
	return nil
}

// positionHooked runs ticks replayed steps and returns the lines
// "t=<tick> <position> <16 hex>", the tick being the tick at beginStep. hook,
// when set, is called before each line is hashed, with the tick and position.
func positionHooked(w *World, ticks int, hook func(tick int64, position string)) ([]string, error) {
	var lines []string
	for i := 0; i < ticks; i++ {
		tick := w.Tick()
		err := replayStep(w, func(position string) {
			if hook != nil {
				hook(tick, position)
			}
			sum := hashWorld(w)
			lines = append(lines, fmt.Sprintf("t=%d %s %x", tick, position, sum[:8]))
		})
		if err != nil {
			return lines, err
		}
	}
	return lines, nil
}

// positionHashes is positionHooked with no hook.
func positionHashes(w *World, ticks int) ([]string, error) { return positionHooked(w, ticks, nil) }

// divergence is what firstDivergence returns for two recorded sequences.
type divergence struct {
	Tick     int64
	Position string
	Line     int // 0-based index of the line, -1 for none
}

// firstDivergence names the tick and position of the first line at which a and
// b differ, or Line -1 when they are equal. A shorter sequence diverges at the
// first line it lacks. Lines have the form "t=<tick> <position> <hash>".
func firstDivergence(a, b []string) divergence {
	for i := 0; i < max(len(a), len(b)); i++ {
		var x, y string
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x == y {
			continue
		}
		line := x
		if line == "" {
			line = y
		}
		var d divergence
		d.Line = i
		var hash string
		if n, _ := fmt.Sscanf(line, "t=%d %s %s", &d.Tick, &d.Position, &hash); n < 2 {
			d.Position = line
		}
		return d
	}
	return divergence{Line: -1}
}

// walkProbe is a hand-built value for the unit tests of the walk.
type walkProbe struct {
	hidden int64
	Shown  string
	M      map[string]int
	P      *walkProbe
	Skip   int64
}

// walkProbeZero is walkProbe with a field added at its off value, as a module
// appends one.
type walkProbeZero struct {
	hidden int64
	Shown  string
	M      map[string]int
	P      *walkProbe
	Skip   int64
	Module *walkProbe
	Extra  []int64
	Flag   bool
}

func mustHash(t *testing.T, v any) [32]byte {
	t.Helper()
	sum, err := hashValue(reflect.ValueOf(v), "walkProbe.Skip", "walkProbeZero.Skip")
	if err != nil {
		t.Fatalf("hashValue error = %v, want nil", err)
	}
	return sum
}

// TestHashWalk_Properties verifies the walk on hand-built structs: an
// unexported field is hashed, a pointer cycle terminates, a map hashes the same
// whatever its insertion order, a skipped field changes nothing, equal values
// hash equal and a one-bit difference changes the hash, and a zero-valued field
// added to a struct leaves the hash unchanged (the off-invariance of D177).
//
// Requirement: testing.md 7.1.2, FR-WLD-044
//
// Guard: a walk that misses state, loops on a cycle or moves when a module
// appends an off field would break the positions golden.
func TestHashWalk_Properties(t *testing.T) {
	t.Parallel()
	cycle := &walkProbe{Shown: "c"}
	cycle.P = cycle
	tests := []struct {
		name     string
		a, b     any
		wantSame bool
	}{
		{"equal values", &walkProbe{hidden: 1, Shown: "x"}, &walkProbe{hidden: 1, Shown: "x"}, true},
		{"unexported field is hashed", &walkProbe{hidden: 1}, &walkProbe{hidden: 2}, false},
		{"exported field is hashed", &walkProbe{Shown: "a"}, &walkProbe{Shown: "b"}, false},
		{"one bit differs", &walkProbe{hidden: 0}, &walkProbe{hidden: 1 << 40}, false},
		{"skipped field ignored", &walkProbe{Skip: 1}, &walkProbe{Skip: 2}, true},
		{"map insertion order", &walkProbe{M: mapInOrder(false)}, &walkProbe{M: mapInOrder(true)}, true},
		{"map value differs", &walkProbe{M: map[string]int{"a": 1}}, &walkProbe{M: map[string]int{"a": 2}}, false},
		{"nil map equals empty map", &walkProbe{}, &walkProbe{M: map[string]int{}}, true},
		{"pointer cycle terminates and differs from nil", cycle, &walkProbe{Shown: "c"}, false},
		{"zero field added", &walkProbe{hidden: 3, Shown: "x"}, &walkProbeZero{hidden: 3, Shown: "x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mustHash(t, tt.a) == mustHash(t, tt.b); got != tt.wantSame {
				t.Errorf("hash(a) == hash(b) = %t, want %t", got, tt.wantSame)
			}
		})
	}
	t.Run("added non-zero field changes the hash", func(t *testing.T) {
		t.Parallel()
		a := &walkProbeZero{Shown: "x"}
		b := &walkProbeZero{Shown: "x", Extra: []int64{0}}
		if mustHash(t, a) == mustHash(t, b) {
			t.Errorf("a non-empty slice field did not change the hash")
		}
	})
}

// mapInOrder builds the same map with its keys inserted forwards or backwards.
func mapInOrder(reverse bool) map[string]int {
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	if reverse {
		slices.Reverse(keys)
	}
	m := map[string]int{}
	for _, k := range keys {
		m[k] = int(k[0])
	}
	return m
}

// TestHashWalk_DepthLimit verifies a value nested deeper than the limit is an
// error and not a crash or a silent truncation.
//
// Requirement: testing.md 7.1.2
//
// Guard: the walk must stop on a runaway structure.
func TestHashWalk_DepthLimit(t *testing.T) {
	t.Parallel()
	type node struct{ Next any }
	var deep any
	for i := 0; i < maxWalkDepth+5; i++ {
		deep = &node{Next: deep}
	}
	if _, err := hashValue(reflect.ValueOf(deep)); err == nil {
		t.Errorf("hashValue of a %d-deep value error = nil, want a depth error", maxWalkDepth+5)
	}
	var shallow any
	for i := 0; i < 5; i++ {
		shallow = &node{Next: shallow}
	}
	if _, err := hashValue(reflect.ValueOf(shallow)); err != nil {
		t.Errorf("hashValue of a 5-deep value error = %v, want nil", err)
	}
}

// TestFirstDivergence_Lines verifies firstDivergence on hand-made lines: equal
// sequences, a difference in the hash, in the position, in the tick, and a
// shorter sequence.
//
// Requirement: testing.md 7.1.2
func TestFirstDivergence_Lines(t *testing.T) {
	t.Parallel()
	base := []string{"t=0 begin aa", "t=0 P0.2 bb", "t=1 begin cc"}
	tests := []struct {
		name string
		b    []string
		want divergence
	}{
		{"equal", base, divergence{Line: -1}},
		{"hash differs", []string{"t=0 begin aa", "t=0 P0.2 XX", "t=1 begin cc"}, divergence{0, "P0.2", 1}},
		{"position differs", []string{"t=0 begin aa", "t=0 P1.1 bb", "t=1 begin cc"}, divergence{0, "P0.2", 1}},
		{"tick differs", []string{"t=0 begin aa", "t=0 P0.2 bb", "t=2 begin cc"}, divergence{1, "begin", 2}},
		{"shorter", base[:2], divergence{1, "begin", 2}},
		{"longer", append(slices.Clone(base), "t=1 P0.2 dd"), divergence{1, "P0.2", 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := firstDivergence(base, tt.b); got != tt.want {
				t.Errorf("firstDivergence = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestReplayRows_Guard verifies the row-to-method map covers positionList: the
// real table passes, and a map with a missing row or an unknown row fails
// naming the position.
//
// Requirement: FR-WLD-004, testing.md 7.1.2
//
// Guard: a position added to tick.go without a method in the map, or a method
// with no row, must fail here.
func TestReplayRows_Guard(t *testing.T) {
	if err := checkReplayRows(); err != nil {
		t.Fatalf("checkReplayRows error = %v, want nil", err)
	}
	if err := checkRows(slices.Delete(slices.Clone(replayRows), 2, 3)); err == nil || !strings.Contains(err.Error(), "P1.1") {
		t.Errorf("checkRows with P1.1 missing error = %v, want one naming P1.1", err)
	}
	if err := checkRows(append(slices.Clone(replayRows), replayRow{id: "P9.9", run: noErr((*World).p02)})); err == nil {
		t.Errorf("checkRows with an unknown row error = nil, want the unknown row")
	}
}

// TestRunRows_StopsAtTheFirstError verifies the error path of the replay loop:
// a position that returns an error stops the replay there, the error names the
// tick and the position and wraps the cause, the rows before it were recorded
// and the rows after it did not run; a row whose guard is not taken records
// nothing.
//
// Requirement: FR-WLD-004, testing.md 7.1.2
//
// Guard: a replay that swallowed a position's error would hash a half-run step
// and the golden would pin a state that Step never reaches.
func TestRunRows_StopsAtTheFirstError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	var ran []string
	ok := func(id string) replayRow {
		return replayRow{id: id, run: func(*World) error { ran = append(ran, id); return nil }}
	}
	rows := []replayRow{
		ok("A"),
		{id: "off", run: func(*World) error { ran = append(ran, "off"); return nil }, on: func(*World) bool { return false }},
		{id: "B", run: func(*World) error { ran = append(ran, "B"); return boom }},
		ok("C"),
	}
	var recorded []string
	err := runRows(nil, 7, rows, func(p string) { recorded = append(recorded, p) })
	if !errors.Is(err, boom) {
		t.Fatalf("runRows error = %v, want one wrapping %v", err, boom)
	}
	if msg := err.Error(); !strings.Contains(msg, "tick 7") || !strings.Contains(msg, "position B") {
		t.Errorf("runRows error = %q, want it to name tick 7 and position B", msg)
	}
	if !slices.Equal(recorded, []string{"A"}) {
		t.Errorf("recorded = %v, want [A]: the failed row and the later row record nothing, a guarded-off row records nothing", recorded)
	}
	if !slices.Equal(ran, []string{"A", "B"}) {
		t.Errorf("ran = %v, want [A B]: C must not run after B failed, off must not run", ran)
	}
}

// graphFields returns "TypeName.field" for every field of every named struct
// type reachable from t through pointers, slices, arrays and maps (an interface's
// dynamic type is not known statically), which is how skipFields names a field.
func graphFields(t reflect.Type, seen map[reflect.Type]bool, out map[string]bool) {
	if seen[t] {
		return
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		graphFields(t.Elem(), seen, out)
	case reflect.Map:
		graphFields(t.Key(), seen, out)
		graphFields(t.Elem(), seen, out)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			out[t.Name()+"."+t.Field(i).Name] = true
			graphFields(t.Field(i).Type, seen, out)
		}
	}
}

// TestSkipFields_NameRealFields verifies every entry of skipFields names a
// field of a struct type in the type graph of World, and that the list has no
// duplicate. The walk matches the list by "Type.field" string, so an entry for
// a field that was renamed or removed would silently stop skipping nothing and
// hide the day the field becomes state.
//
// Requirement: FR-WLD-044, testing.md 7.1.2
//
// Guard: a stale skip entry is a hash that includes (or fails to exclude) a
// field nobody can see in the list.
func TestSkipFields_NameRealFields(t *testing.T) {
	t.Parallel()
	real := map[string]bool{}
	graphFields(reflect.TypeOf(World{}), map[reflect.Type]bool{}, real)
	seen := map[string]bool{}
	for _, f := range skipFields {
		if !real[f] {
			t.Errorf("skipFields entry %q names no field of a struct in the type graph of World", f)
		}
		if seen[f] {
			t.Errorf("skipFields entry %q is listed twice", f)
		}
		seen[f] = true
	}
}
