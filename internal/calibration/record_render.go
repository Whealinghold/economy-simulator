package calibration

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
)

// ErrRecord is returned for a record that Render cannot write or Parse cannot
// read: an unknown keyword, a wrong field count, a word outside its closed set, a
// float, a field that is not a token, or a line out of order. Match it with
// errors.Is; the message names the field or the line.
const ErrRecord = calError("calibration: invalid record")

// recordVersion is the number on the first line of a record (8.2).
const recordVersion = 1

// The closed sets of the grammar of 8.2, each space-padded so a word is found by
// " word " (rule constants of the check of 11.2).
const (
	wDesign   = " oat spread grid none "
	wSeeding  = " same per_point "
	wEdge     = " none below:bound below:rule below:off above:bound above:rule above:off "
	wWhy      = " at_off invisible v1x "
	wKind     = " fact detector counter "
	wOp       = " AtMost AtLeast Between "
	wTarget   = " set working report "
	wFactTarg = " set working report none "
	wStatus   = " open explained row_corrected "
	wResult   = " pass fail report na "
	wVerdict  = " none detected unknown "
	wPoint    = " default lower upper "
	wSet      = " gate holdout "
	wStab     = " STABLE NOT_STABLE UNDETERMINED "
	wClaim    = " stable not_stable one_neighbour undetermined "
	wDecision = " " + decStop + " " + decEscalate + " " + decReview + " " + decRevert + " "
	classSet  = "XOBRP"
	signSet   = "+-0~?"
	hexDigits = "0123456789abcdef"
	hexWidth  = 16
	sameWord  = "same"
)

// writer builds the canonical text of a record and keeps the first error.
type writer struct {
	b   strings.Builder
	err error
}

func (w *writer) fail(format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf("%w: %s", ErrRecord, fmt.Sprintf(format, args...))
	}
}

// line writes a keyword and its fields separated by one space.
func (w *writer) line(kw string, fields ...string) {
	w.b.WriteString(kw)
	for _, f := range fields {
		w.b.WriteByte(' ')
		w.b.WriteString(f)
	}
	w.b.WriteByte('\n')
}

// line2 writes a keyword and groups of fields separated by two spaces, as the
// pack, commit and config lines are.
func (w *writer) line2(kw string, groups ...string) {
	w.b.WriteString(kw + " " + strings.Join(groups, "  ") + "\n")
}

// tok checks that s is a token: printable ASCII with no space.
func (w *writer) tok(what, s string) string {
	if s == "" {
		w.fail("%s is empty", what)
	}
	for i := range len(s) {
		if s[i] <= ' ' || s[i] > '~' {
			w.fail("%s %q is not a token", what, s)
			break
		}
	}
	return s
}

// toks checks and joins a list of tokens.
func (w *writer) toks(what string, ss []string) string {
	for _, s := range ss {
		w.tok(what, s)
	}
	return strings.Join(ss, " ")
}

// one checks that s is a word of a closed set.
func (w *writer) one(what, s, set string) string {
	if s == "" || strings.ContainsAny(s, " ") || !strings.Contains(set, " "+s+" ") {
		w.fail("%s %q is not one of%s", what, s, strings.TrimRight(set, " "))
	}
	return s
}

// hex checks 16 lower-case hex digits.
func (w *writer) hex(what, s string) string {
	if len(s) != hexWidth || strings.Trim(s, hexDigits) != "" {
		w.fail("%s %q is not 16 lower-case hex digits", what, s)
	}
	return s
}

// nat checks a count that must not be negative and writes it.
func (w *writer) nat(what string, v int64) string {
	if v < 0 {
		w.fail("%s %d is negative", what, v)
	}
	return strconv.FormatInt(v, 10)
}

// opt writes an integer or `-`, and checks that an absent one holds 0.
func (w *writer) opt(what string, o Opt) string {
	if !o.Has {
		if o.Value != 0 {
			w.fail("%s holds %d with no value", what, o.Value)
		}
		return "-"
	}
	return strconv.FormatInt(o.Value, 10)
}

// dash writes s, or `-` for none.
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func itoa(v int) string { return strconv.Itoa(v) }

func i64(v int64) string { return strconv.FormatInt(v, 10) }

func u64(v uint64) string { return strconv.FormatUint(v, 10) }

// Render returns the canonical text of a record: LF line ends, decimal integers
// with no sign for a non-negative one, 16 lower-case hex digits for a digest, `-`
// for none, two spaces between the groups of the pack, commit and config lines
// and one space elsewhere, and the lines in the order of 8.2 (FR-CLB-041,
// NFR-CLB-002). It returns ErrRecord for a field the grammar cannot write: a
// word outside its closed set, an empty or spaced token, a negative count.
func Render(r Record) ([]byte, error) {
	var w writer
	w.header(r)
	for _, d := range r.Documents {
		w.line("document", w.tok("document name", d.Name), w.hex("document digest", d.Digest))
	}
	w.plan(r)
	w.screen(r)
	w.result(r)
	w.confirmed(r)
	w.stability(r)
	w.end(r)
	if w.err != nil {
		return nil, w.err
	}
	return []byte(w.b.String()), nil
}

// header writes the calibration, pack, commit and config lines.
func (w *writer) header(r Record) {
	w.line("calibration", strconv.Itoa(recordVersion))
	if r.Attempt < 1 {
		w.fail("attempt %d is below 1", r.Attempt)
	}
	w.line2("pack", w.tok("pack", r.Pack), "lineage "+w.hex("lineage", r.Lineage), "stage "+w.stage(r.Stage),
		"attempt "+itoa(r.Attempt), "block "+w.nat("block", int64(r.Block)))
	parent := "-"
	if r.Parent != "" {
		parent = w.hex("parent", r.Parent)
	}
	w.line2("commit", w.tok("commit", r.Commit), "parent "+parent)
	chosen := r.Chosen
	if chosen != sameWord {
		w.hex("config chosen", chosen)
	}
	w.line2("config", "base "+w.hex("config base", r.Base), "chosen "+chosen)
}

// stage checks a stage name.
func (w *writer) stage(s string) string {
	if stageRank(s) < 0 {
		w.fail("stage %q is not one of %s", s, stages)
	}
	return s
}

// plan writes the experiment, factor and unsearched lines.
func (w *writer) plan(r Record) {
	for _, e := range r.Experiments {
		w.line("experiment", w.tok("step", e.Step), "design", w.one("design", e.Design, wDesign),
			"points", w.nat("points", int64(e.Points)), "replicas", w.nat("replicas", int64(e.Replicas)),
			"seed0", u64(e.Seed0), "seeding", w.one("seeding", e.Seeding, wSeeding),
			"ticks", w.nat("ticks", e.Ticks), "transient", w.nat("transient", e.Transient), "every", w.nat("every", e.Every),
			"salt", u64(e.Salt), "render", w.hex("render", e.Render))
	}
	for _, f := range r.Factors {
		w.line("factor", w.tok("factor name", f.Name), "key", dash(w.keyOrNone("factor key", f.Key)), "grid", w.grid(f.Grid),
			"base", i64(f.Base), "lower", w.opt("lower", f.Lower), "upper", w.opt("upper", f.Upper),
			"edge", w.one("edge", f.Edge, wEdge))
	}
	for _, u := range r.Unsearched {
		w.line("unsearched", w.one("unsearched reason", u.Why, wWhy), w.toksNonEmpty("unsearched knob", u.Knobs))
	}
}

// keyOrNone checks a key that may be empty (written `-`).
func (w *writer) keyOrNone(what, s string) string {
	if s == "-" { // the written form of empty: it would read back as no key
		w.fail("%s %q is the word for none", what, s)
	}
	if s != "" {
		w.tok(what, s)
	}
	return s
}

// phrase checks one or more tokens separated by single spaces and writes them.
func (w *writer) phrase(what, s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 || strings.Join(fields, " ") != s {
		w.fail("%s %q is not tokens separated by one space", what, s)
	}
	return w.toks(what, fields)
}

// toksNonEmpty checks a list that must have at least one token and joins it.
func (w *writer) toksNonEmpty(what string, ss []string) string {
	if len(ss) == 0 {
		w.fail("%s list is empty", what)
	}
	return w.toks(what, ss)
}

// grid checks `geo`, `set` or `lin` followed by a step.
func (w *writer) grid(s string) string {
	step, ok := strings.CutPrefix(s, "lin")
	if s != "geo" && s != "set" && (!ok || step == "" || strings.Trim(step, "0123456789") != "") {
		w.fail("grid %q is not geo, set or lin<step>", s)
	}
	return s
}

// screen writes the measure, criterion and screen lines and the knob sets.
func (w *writer) screen(r Record) {
	for _, m := range r.Measures {
		w.line("measure", w.tok("measure name", m.Name), "fact", w.tok("measure fact", m.Fact),
			"token", dash(w.keyOrNone("measure token", m.Token)), "class", w.class(m.Class), "kind", w.one("kind", m.Kind, wKind))
	}
	for _, c := range r.Criteria {
		w.line("criterion", w.tok("criterion name", c.Name), "measure", w.tok("criterion measure", c.Measure),
			"op", w.one("op", c.Op, wOp), "a", i64(c.A), "b", i64(c.B), "target", w.one("target", c.Target, wTarget))
	}
	for _, s := range r.Screens {
		w.line("screen", w.tok("screen knob", s.Knob), "measure", w.tok("screen measure", s.Measure),
			"el_lower", w.opt("el_lower", s.ElLower), "el_upper", w.opt("el_upper", s.ElUpper),
			"range", i64(s.Range), "share_bp", i64(s.ShareBP))
	}
	if len(r.ScreenedIn) > 0 {
		w.line("screened_in", w.toks("screened_in knob", r.ScreenedIn))
	}
	if len(r.Dormant) > 0 {
		w.line("dormant", w.toks("dormant knob", r.Dormant))
	}
}

// class checks a fact class and writes it.
func (w *writer) class(c byte) string {
	if !strings.ContainsRune(classSet, rune(c)) {
		w.fail("class %q is not one of %s", string(rune(c)), classSet)
	}
	return string(rune(c))
}

// sign checks a sign and writes it.
func (w *writer) sign(c byte) string {
	if !strings.ContainsRune(signSet, rune(c)) {
		w.fail("sign %q is not one of %s", string(rune(c)), signSet)
	}
	return string(rune(c))
}

// result writes the finding, feasible, chosen, pareto and set lines.
func (w *writer) result(r Record) {
	for _, f := range r.Findings {
		w.line("finding", w.tok("finding knob", f.Knob), w.tok("finding fact", f.Fact)+"/"+w.tok("finding measure", f.Measure),
			"expected", w.sign(f.Expected), "measured", w.sign(f.Measured), "status", w.one("status", f.Status, wStatus))
	}
	if f := r.Feasible; f.Has {
		if f.N < 0 || f.Of < 0 {
			w.fail("feasible %d of %d is negative", f.N, f.Of)
		}
		w.line("feasible", itoa(f.N), "of", itoa(f.Of))
	} else if f != (Feasible{}) {
		w.fail("feasible holds values with Has false")
	}
	w.pick(r.Pick)
	if len(r.Pareto) > 0 {
		idx := make([]string, len(r.Pareto))
		for i, p := range r.Pareto {
			idx[i] = w.nat("pareto index", int64(p))
		}
		w.line("pareto", idx...)
	}
	for _, s := range r.Sets {
		w.line("set", w.tok("set key", s.Key), i64(s.Value))
	}
}

// pick writes the chosen line.
func (w *writer) pick(p Pick) {
	switch {
	case !p.Has:
		if p != (Pick{}) {
			w.fail("chosen holds values with Has false")
		}
	case p.None:
		if p.Index != 0 {
			w.fail("chosen none holds index %d", p.Index)
		}
		w.line("chosen", "none", "distance", w.nat("distance", p.Distance))
	default:
		w.line("chosen", w.nat("chosen index", int64(p.Index)), "distance", w.nat("distance", p.Distance))
	}
}

// confirmed writes the confirm, fact and detector lines.
func (w *writer) confirmed(r Record) {
	if c := r.Confirm; c.Has {
		w.line("confirm", "seeds", u64(c.First)+".."+u64(c.Last), "control", yesno(c.Control))
	} else if c != (ConfirmLine{}) {
		w.fail("confirm holds values with Has false")
	}
	for _, f := range r.Facts {
		outside := "-"
		if len(f.Outside) > 0 {
			ss := make([]string, len(f.Outside))
			for i, s := range f.Outside {
				ss[i] = u64(s)
			}
			outside = strings.Join(ss, ",")
		}
		w.line("fact", w.tok("fact ID", f.ID), "class", w.class(f.Class), "key", dash(w.keyOrNone("fact key", f.Key)),
			"target", w.one("fact target", f.Target, wFactTarg), "values", "min", i64(f.Min), "median", i64(f.Median), "max", i64(f.Max),
			"outside", outside, "result", w.one("result", f.Result, wResult))
	}
	for _, d := range r.Detectors {
		w.line("detector", w.tok("detector ID", d.ID), w.one("verdict", d.Verdict, wVerdict),
			"working", yesno(d.Working), "onset", w.opt("onset", d.Onset))
	}
}

// stability writes the stability and claim lines.
func (w *writer) stability(r Record) {
	for _, s := range r.Stability {
		w.line("stability", w.tok("stability knob", s.Knob), "pack", w.tok("stability pack", s.Pack),
			"point", w.one("point", s.Point, wPoint), "value", i64(s.Value), "set", w.one("set", s.Set, wSet),
			"seeds", u64(s.First)+".."+u64(s.Last), "in_sample", yesno(s.InSample),
			"c_bp", w.one("c_bp", s.CBP, wStab), "c_nc", w.one("c_nc", s.CNC, wStab), "c_ws", w.one("c_ws", s.CWS, wStab),
			"edge", w.one("edge", s.Edge, wEdge))
	}
	for _, c := range r.Claims {
		w.line("claim", w.tok("claim knob", c.Knob), "pack", w.tok("claim pack", c.Pack), w.one("claim", c.Word, wClaim))
	}
}

// end writes the unmet, decision and note lines.
func (w *writer) end(r Record) {
	for _, u := range r.Unmet {
		w.line("unmet", w.tok("unmet fact", u.Fact), w.tok("unmet measure", u.Measure), w.phrase("unmet values", u.Values))
	}
	w.line("decision", w.decision(r.Decision))
	for _, n := range r.Notes {
		if strings.ContainsAny(n, "\r\n") {
			w.fail("note contains a line break")
		}
		w.b.WriteString("note " + n + "\n")
	}
}

// decision checks a decision: a closed word, or `reentry` and a stage.
func (w *writer) decision(d string) string {
	if to, ok := strings.CutPrefix(d, decReentry); ok {
		w.stage(to)
		return d
	}
	w.one("decision", d, wDecision)
	return d
}

// Digest returns the first 8 bytes of the SHA-256 of b as 16 lower-case hex
// digits: the digest of a document and of a rendered text (FR-CLB-040, EXP
// 8.2.2).
func Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:8])
}
