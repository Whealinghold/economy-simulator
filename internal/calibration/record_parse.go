package calibration

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Parse reads a record. It is strict: it accepts exactly the bytes Render
// produces, so Render(Parse(b)) equals b for every b it accepts (FR-CLB-043,
// NFR-CLB-004). An unknown keyword, a wrong field count, a float, a sign on a
// non-negative integer, a leading zero, a word outside its closed set, a keyword
// out of order or repeated, a missing final line feed or a note with a line break
// is ErrRecord. Parse never panics.
func Parse(b []byte) (Record, error) {
	text, ok := strings.CutSuffix(string(b), "\n")
	if !ok {
		return Record{}, fmt.Errorf("%w: the record does not end with a line feed", ErrRecord)
	}
	var r Record
	handlers := lineHandlers()
	for n, line := range strings.Split(text, "\n") {
		kw, rest, _ := strings.Cut(line, " ")
		h, known := handlers[kw]
		if !known {
			return Record{}, fmt.Errorf("%w: line %d: unknown keyword %q", ErrRecord, n+1, kw)
		}
		if err := h(&r, rest); err != nil {
			return Record{}, fmt.Errorf("%w: line %d: %v", ErrRecord, n+1, err)
		}
	}
	out, err := Render(r)
	if err != nil {
		return Record{}, err
	}
	if !bytes.Equal(out, b) {
		return Record{}, fmt.Errorf("%w: the lines are not in canonical form or order", ErrRecord)
	}
	return r, nil
}

// cursor reads the fields of one line and keeps the first error.
type cursor struct {
	f   []string
	i   int
	err error
}

func newCursor(rest string) *cursor { return &cursor{f: strings.Fields(rest)} }

func (c *cursor) fail(format string, args ...any) {
	if c.err == nil {
		c.err = fmt.Errorf(format, args...)
	}
}

// next returns the next field.
func (c *cursor) next(what string) string {
	if c.err != nil {
		return ""
	}
	if c.i >= len(c.f) {
		c.fail("missing %s", what)
		return ""
	}
	c.i++
	return c.f[c.i-1]
}

// kw reads a keyword the grammar fixes.
func (c *cursor) kw(want string) {
	if got := c.next(want); c.err == nil && got != want {
		c.fail("got %q, want %q", got, want)
	}
}

// str reads a field; `-` is none.
func (c *cursor) str(what string) string { return c.next(what) }

// dash reads a field in which `-` means none.
func (c *cursor) dash(what string) string {
	if s := c.next(what); s != "-" {
		return s
	}
	return ""
}

// i64 reads a decimal integer.
func (c *cursor) i64(what string) int64 {
	s := c.next(what)
	if c.err != nil {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.fail("%s %q is not an integer", what, s)
	}
	return v
}

// int reads a decimal integer that fits int.
func (c *cursor) int(what string) int {
	s := c.next(what)
	if c.err != nil {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		c.fail("%s %q is not an integer", what, s)
	}
	return v
}

// u64 reads an unsigned decimal integer.
func (c *cursor) u64(what string) uint64 {
	s := c.next(what)
	if c.err != nil {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		c.fail("%s %q is not an unsigned integer", what, s)
	}
	return v
}

// opt reads an integer or `-`.
func (c *cursor) opt(what string) Opt {
	s := c.next(what)
	if s == "-" || c.err != nil {
		return Opt{}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.fail("%s %q is not an integer or -", what, s)
	}
	return Opt{Has: true, Value: v}
}

// yesno reads `yes` or `no`.
func (c *cursor) yesno(what string) bool {
	switch s := c.next(what); s {
	case "yes":
		return true
	case "no":
		return false
	default:
		c.fail("%s %q is not yes or no", what, s)
		return false
	}
}

// byte1 reads a one-character field.
func (c *cursor) byte1(what string) byte {
	s := c.next(what)
	if len(s) != 1 {
		c.fail("%s %q is not one character", what, s)
		return 0
	}
	return s[0]
}

// seeds reads `a..b`.
func (c *cursor) seeds(what string) (first, last uint64) {
	s := c.next(what)
	a, b, ok := strings.Cut(s, "..")
	if !ok {
		c.fail("%s %q is not a..b", what, s)
		return 0, 0
	}
	var err1, err2 error
	first, err1 = strconv.ParseUint(a, 10, 64)
	last, err2 = strconv.ParseUint(b, 10, 64)
	if err1 != nil || err2 != nil {
		c.fail("%s %q is not a..b", what, s)
	}
	return first, last
}

// rest returns the fields not yet read and consumes them.
func (c *cursor) rest() []string {
	if c.err != nil {
		return nil
	}
	out := c.f[c.i:]
	c.i = len(c.f)
	return out
}

// end returns the first error, or an error for fields left over.
func (c *cursor) end() error {
	if c.err == nil && c.i != len(c.f) {
		c.fail("%d extra fields", len(c.f)-c.i)
	}
	return c.err
}

// lineHandlers maps each keyword of the grammar to the reader of its line. It is
// a function, not a variable: the package holds no package-level state.
func lineHandlers() map[string]func(*Record, string) error {
	return map[string]func(*Record, string) error{
		"calibration": readVersion, "pack": readPack, "commit": readCommit, "config": readConfig,
		"document": readDocument, "experiment": readExperiment, "factor": readFactor,
		"unsearched": readUnsearched, "measure": readMeasure, "criterion": readCriterion,
		"screen": readScreen, "screened_in": readScreenedIn, "dormant": readDormant,
		"finding": readFinding, "feasible": readFeasible, "chosen": readChosen, "pareto": readPareto,
		"set": readSet, "confirm": readConfirm, "fact": readFact, "detector": readDetector,
		"stability": readStability, "claim": readClaim, "unmet": readUnmet,
		"decision": readDecision, "note": readNote,
	}
}

func readVersion(_ *Record, rest string) error {
	if rest != strconv.Itoa(recordVersion) {
		return fmt.Errorf("format version %q, want %d", rest, recordVersion)
	}
	return nil
}

func readPack(r *Record, rest string) error {
	c := newCursor(rest)
	r.Pack = c.str("pack")
	c.kw("lineage")
	r.Lineage = c.str("lineage")
	c.kw("stage")
	r.Stage = c.str("stage")
	c.kw("attempt")
	r.Attempt = c.int("attempt")
	c.kw("block")
	r.Block = c.int("block")
	return c.end()
}

func readCommit(r *Record, rest string) error {
	c := newCursor(rest)
	r.Commit = c.str("commit")
	c.kw("parent")
	r.Parent = c.dash("parent")
	return c.end()
}

func readConfig(r *Record, rest string) error {
	c := newCursor(rest)
	c.kw("base")
	r.Base = c.str("base")
	c.kw("chosen")
	r.Chosen = c.str("chosen")
	return c.end()
}

func readDocument(r *Record, rest string) error {
	c := newCursor(rest)
	d := Document{Name: c.str("name"), Digest: c.str("digest")}
	r.Documents = append(r.Documents, d)
	return c.end()
}

func readExperiment(r *Record, rest string) error {
	c := newCursor(rest)
	var e Experiment
	e.Step = c.str("step")
	c.kw("design")
	e.Design = c.str("design")
	c.kw("points")
	e.Points = c.int("points")
	c.kw("replicas")
	e.Replicas = c.int("replicas")
	c.kw("seed0")
	e.Seed0 = c.u64("seed0")
	c.kw("seeding")
	e.Seeding = c.str("seeding")
	c.kw("ticks")
	e.Ticks = c.i64("ticks")
	c.kw("transient")
	e.Transient = c.i64("transient")
	c.kw("every")
	e.Every = c.i64("every")
	c.kw("salt")
	e.Salt = c.u64("salt")
	c.kw("render")
	e.Render = c.str("render")
	r.Experiments = append(r.Experiments, e)
	return c.end()
}

func readFactor(r *Record, rest string) error {
	c := newCursor(rest)
	var f Factor
	f.Name = c.str("name")
	c.kw("key")
	f.Key = c.dash("key")
	c.kw("grid")
	f.Grid = c.str("grid")
	c.kw("base")
	f.Base = c.i64("base")
	c.kw("lower")
	f.Lower = c.opt("lower")
	c.kw("upper")
	f.Upper = c.opt("upper")
	c.kw("edge")
	f.Edge = c.str("edge")
	r.Factors = append(r.Factors, f)
	return c.end()
}

func readUnsearched(r *Record, rest string) error {
	c := newCursor(rest)
	u := Unsearched{Why: c.str("reason")}
	u.Knobs = c.rest()
	r.Unsearched = append(r.Unsearched, u)
	return c.end()
}

func readMeasure(r *Record, rest string) error {
	c := newCursor(rest)
	var m Measure
	m.Name = c.str("name")
	c.kw("fact")
	m.Fact = c.str("fact")
	c.kw("token")
	m.Token = c.dash("token")
	c.kw("class")
	m.Class = c.byte1("class")
	c.kw("kind")
	m.Kind = c.str("kind")
	r.Measures = append(r.Measures, m)
	return c.end()
}

func readCriterion(r *Record, rest string) error {
	c := newCursor(rest)
	var k Criterion
	k.Name = c.str("name")
	c.kw("measure")
	k.Measure = c.str("measure")
	c.kw("op")
	k.Op = c.str("op")
	c.kw("a")
	k.A = c.i64("a")
	c.kw("b")
	k.B = c.i64("b")
	c.kw("target")
	k.Target = c.str("target")
	r.Criteria = append(r.Criteria, k)
	return c.end()
}

func readScreen(r *Record, rest string) error {
	c := newCursor(rest)
	var s Screen
	s.Knob = c.str("knob")
	c.kw("measure")
	s.Measure = c.str("measure")
	c.kw("el_lower")
	s.ElLower = c.opt("el_lower")
	c.kw("el_upper")
	s.ElUpper = c.opt("el_upper")
	c.kw("range")
	s.Range = c.i64("range")
	c.kw("share_bp")
	s.ShareBP = c.i64("share_bp")
	r.Screens = append(r.Screens, s)
	return c.end()
}

func readScreenedIn(r *Record, rest string) error {
	c := newCursor(rest)
	r.ScreenedIn = c.rest()
	return c.end()
}

func readDormant(r *Record, rest string) error {
	c := newCursor(rest)
	r.Dormant = c.rest()
	return c.end()
}

func readFinding(r *Record, rest string) error {
	c := newCursor(rest)
	var f Finding
	f.Knob = c.str("knob")
	pair := c.str("fact/measure")
	f.Fact, f.Measure, _ = strings.Cut(pair, "/")
	c.kw("expected")
	f.Expected = c.byte1("expected")
	c.kw("measured")
	f.Measured = c.byte1("measured")
	c.kw("status")
	f.Status = c.str("status")
	r.Findings = append(r.Findings, f)
	return c.end()
}

func readFeasible(r *Record, rest string) error {
	c := newCursor(rest)
	r.Feasible.Has = true
	r.Feasible.N = c.int("n")
	c.kw("of")
	r.Feasible.Of = c.int("N")
	return c.end()
}

func readChosen(r *Record, rest string) error {
	c := newCursor(rest)
	r.Pick.Has = true
	if first := c.str("index"); first == "none" {
		r.Pick.None = true
	} else if n, err := strconv.Atoi(first); err == nil {
		r.Pick.Index = n
	} else {
		c.fail("chosen index %q is not an integer or none", first)
	}
	c.kw("distance")
	r.Pick.Distance = c.i64("distance")
	return c.end()
}

func readPareto(r *Record, rest string) error {
	c := newCursor(rest)
	for _, s := range c.rest() {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("pareto index %q is not an integer", s)
		}
		r.Pareto = append(r.Pareto, n)
	}
	return c.end()
}

func readSet(r *Record, rest string) error {
	c := newCursor(rest)
	s := Setting{Key: c.str("key")}
	s.Value = c.i64("value")
	r.Sets = append(r.Sets, s)
	return c.end()
}

func readConfirm(r *Record, rest string) error {
	c := newCursor(rest)
	r.Confirm.Has = true
	c.kw("seeds")
	r.Confirm.First, r.Confirm.Last = c.seeds("seeds")
	c.kw("control")
	r.Confirm.Control = c.yesno("control")
	return c.end()
}

func readFact(r *Record, rest string) error {
	c := newCursor(rest)
	var f Fact
	f.ID = c.str("ID")
	c.kw("class")
	f.Class = c.byte1("class")
	c.kw("key")
	f.Key = c.dash("key")
	c.kw("target")
	f.Target = c.str("target")
	c.kw("values")
	c.kw("min")
	f.Min = c.i64("min")
	c.kw("median")
	f.Median = c.i64("median")
	c.kw("max")
	f.Max = c.i64("max")
	c.kw("outside")
	f.Outside = c.seedList("outside")
	c.kw("result")
	f.Result = c.str("result")
	r.Facts = append(r.Facts, f)
	return c.end()
}

// seedList reads `seed,seed,...` or `-`.
func (c *cursor) seedList(what string) []uint64 {
	s := c.next(what)
	if s == "-" || c.err != nil {
		return nil
	}
	var out []uint64
	for _, p := range strings.Split(s, ",") {
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			c.fail("%s %q is not a list of seeds or -", what, s)
			return nil
		}
		out = append(out, v)
	}
	return out
}

func readDetector(r *Record, rest string) error {
	c := newCursor(rest)
	var d Detector
	d.ID = c.str("ID")
	d.Verdict = c.str("verdict")
	c.kw("working")
	d.Working = c.yesno("working")
	c.kw("onset")
	d.Onset = c.opt("onset")
	r.Detectors = append(r.Detectors, d)
	return c.end()
}

func readStability(r *Record, rest string) error {
	c := newCursor(rest)
	var s Stability
	s.Knob = c.str("knob")
	c.kw("pack")
	s.Pack = c.str("pack")
	c.kw("point")
	s.Point = c.str("point")
	c.kw("value")
	s.Value = c.i64("value")
	c.kw("set")
	s.Set = c.str("set")
	c.kw("seeds")
	s.First, s.Last = c.seeds("seeds")
	c.kw("in_sample")
	s.InSample = c.yesno("in_sample")
	c.kw("c_bp")
	s.CBP = c.str("c_bp")
	c.kw("c_nc")
	s.CNC = c.str("c_nc")
	c.kw("c_ws")
	s.CWS = c.str("c_ws")
	c.kw("edge")
	s.Edge = c.str("edge")
	r.Stability = append(r.Stability, s)
	return c.end()
}

func readClaim(r *Record, rest string) error {
	c := newCursor(rest)
	var k Claim
	k.Knob = c.str("knob")
	c.kw("pack")
	k.Pack = c.str("pack")
	k.Word = c.str("claim")
	r.Claims = append(r.Claims, k)
	return c.end()
}

func readUnmet(r *Record, rest string) error {
	c := newCursor(rest)
	u := Unmet{Fact: c.str("fact"), Measure: c.str("measure")}
	u.Values = strings.Join(c.rest(), " ")
	r.Unmet = append(r.Unmet, u)
	return c.end()
}

func readDecision(r *Record, rest string) error {
	r.Decision = rest
	return nil
}

func readNote(r *Record, rest string) error {
	r.Notes = append(r.Notes, rest)
	return nil
}
