package calibration

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// The rule words of a Problem (spec-calibration.md 11.2, 14). A Problem names
// one rule so a test can assert it without reading the detail.
const (
	RuleStageName      = "stage_name"
	RuleStageOrder     = "stage_order"
	RuleAttemptNumber  = "attempt_number"
	RuleAttempts       = "attempts"
	RuleOpenExact      = "open_exact_fact"
	RuleBlockRange     = "block_range"
	RuleBlockReused    = "block_reused"
	RuleSearchSeed     = "search_seed"
	RuleConfirmSeed    = "confirm_seed"
	RuleMaxFactors     = "max_factors"
	RuleStageKnob      = "stage_knob"
	RuleOwnerSplit     = "owner_split"
	RuleLineageParent  = "lineage_parent"
	RuleLineageRestart = "lineage_restart"
	RuleLineageSet     = "lineage_set"
	RuleLineageDoc     = "lineage_document"
	RuleStopExact      = "stop_exact"
	RuleStopDetector   = "stop_detector"
	RuleStopBSet       = "stop_b_set"
	RuleStopBWorking   = "stop_b_working"
	RuleStopFinding    = "stop_finding"
	RuleStopDormant    = "stop_dormant"
	RuleStopControl    = "stop_control"
)

// The constants of the process rules, each with its section of the spec.
const (
	// stages is the closed set of stage names in their order (7.3), C4 and C7 in
	// sub-stages.
	stages = "C0 C1 C2 C3 C4a C4b C4c C5 C6 C7a C7b C8 C9"
	// stageC1 is the stage every new lineage starts from (7.3.11, 9.2).
	stageC1 = "C1"
	// maxAttempts is the attempts at a stage before a review (7.1.4, FR-CLB-027).
	maxAttempts = 3
	// confirmLast is the last confirm seed; tier L uses fewer, which only narrows it.
	confirmLast = confirmSeeds
	// Decisions (8.2).
	decStop    = "stop"
	decReview  = "review"
	decReentry = "reentry "
	// Steps of an experiment that the seed rules read (7.2).
	stepScreen  = "screen"
	stepExplore = "explore"
	stepConfirm = "confirm"
	// The document names of a lineage's derived files: a calibration overlay and a
	// warm-start table are not part of the lineage (9.2) and cannot be carried to
	// another one (FR-CLB-054).
	overlayDir = "calibrated/"
	warmPrefix = "warm"
	// Results, verdicts and statuses of the lines the stop reads (8.2).
	resPass, resNA, resFail = "pass", "na", "fail"
	verdictNone             = "none"
	verdictUnknown          = "unknown"
	targetSet               = "set"
	targetWorking           = "working"
	statusOpen              = "open"
)

// ErrLineage is returned by Comparable for two records of different lineages or
// packs (FR-CLB-054).
const ErrLineage = calError("calibration: records of different lineages are not comparable")

// Problem is one failure of a rule, found in the record Record (an index into
// the sequence). Line is the 1-based line of the rendered record it points at, 0
// when it concerns the record as a whole.
type Problem struct {
	Record int
	Line   int
	Rule   string
	Detail string
}

// Comparable returns ErrLineage when a and b are of different packs or
// lineages: such records are never compared (8.5, FR-CLB-054).
func Comparable(a, b Record) error {
	if a.Pack != b.Pack || a.Lineage != b.Lineage {
		return ErrLineage
	}
	return nil
}

// stageRank is the position of a stage in the order C0 to C9, or -1.
func stageRank(name string) int {
	rank := 0
	for rest := stages; rest != ""; rank++ {
		var w string
		w, rest, _ = strings.Cut(rest, " ")
		if w == name {
			return rank
		}
	}
	return -1
}

// stageFamily is the stage without its sub-stage letter: C4a is C4.
func stageFamily(name string) string {
	return strings.TrimRight(name, "abc")
}

// packState is what CheckSequence remembers of the records of one pack.
type packState struct {
	prev      *Record
	attempts  int             // attempts in the visit since the last review
	openExact map[string]bool // X and O facts whose last result was a failure
	blocks    map[int]bool    // the search blocks already used by a choice
	owners    map[string]string
	retired   map[string]bool // digests of derived documents of earlier lineages
	current   map[string]bool // digests of derived documents of the present lineage
}

// checker holds one record under check and appends to the problems.
type checker struct {
	rs         []Record
	i          int
	st         *packState
	idx        index
	maxFactors int
	out        *[]Problem
}

// CheckSequence checks a sequence of stored records against the process rules of
// spec-calibration.md 7 and 11.2 and returns the problems sorted by record, line,
// rule and detail (FR-CLB-020, 021, 027, 035, 036, 038, 053, 054). Records of
// different packs may be interleaved; each pack is checked in its own order.
// maxFactors is the cap of EXP 13.2, a parameter so the package does not import
// experiment. The check is pure: it reads only its arguments.
func CheckSequence(rs []Record, k Known, maxFactors int) []Problem {
	var out []Problem
	states := map[string]*packState{}
	idx := newIndex(k)
	for i := range rs {
		st := states[rs[i].Pack]
		if st == nil {
			st = &packState{openExact: map[string]bool{}, blocks: map[int]bool{}, owners: map[string]string{}, retired: map[string]bool{}, current: map[string]bool{}}
			states[rs[i].Pack] = st
		}
		c := checker{rs: rs, i: i, st: st, idx: idx, maxFactors: maxFactors, out: &out}
		c.run()
	}
	slices.SortFunc(out, func(a, b Problem) int {
		return cmp.Or(cmp.Compare(a.Record, b.Record), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Detail, b.Detail))
	})
	return out
}

func (c *checker) r() *Record { return &c.rs[c.i] }

func (c *checker) add(rule, format string, args ...any) {
	*c.out = append(*c.out, Problem{Record: c.i, Rule: rule, Detail: fmt.Sprintf(format, args...)})
}

// run applies every rule to the record and then folds it into the pack's state.
func (c *checker) run() {
	r := c.r()
	rank := stageRank(r.Stage)
	if rank < 0 {
		c.add(RuleStageName, "stage %q is not one of %s", r.Stage, stages)
	}
	newLineage := c.st.prev != nil && c.st.prev.Lineage != r.Lineage
	if newLineage {
		c.st.retire()
	}
	c.lineage(newLineage)
	if !newLineage {
		c.order(rank)
	}
	c.attempts(newLineage)
	c.openExact(rank)
	c.blocks()
	c.factors()
	c.stop()
	c.fold()
}

// order enforces the stages in order with the re-entry rule (FR-CLB-035): a record
// is at its predecessor's stage, a later one after a stop, or the stage a
// `reentry` decision named.
func (c *checker) order(rank int) {
	prev := c.st.prev
	if prev == nil || rank < 0 {
		return
	}
	if msg := orderBreak(prev, c.r().Stage, rank); msg != "" {
		c.add(RuleStageOrder, "%s", msg)
	}
}

// orderBreak describes how a record at stage (of the given rank) breaks the order
// after prev, or returns "".
func orderBreak(prev *Record, stage string, rank int) string {
	if to, ok := strings.CutPrefix(prev.Decision, decReentry); ok {
		if stage != to {
			return fmt.Sprintf("stage %s follows decision %q, want %s", stage, prev.Decision, to)
		}
		return ""
	}
	switch pr := stageRank(prev.Stage); {
	case pr < 0:
	case rank < pr:
		return fmt.Sprintf("stage %s follows %s with no reentry decision", stage, prev.Stage)
	case rank == pr && prev.Decision == decStop:
		return fmt.Sprintf("stage %s was stopped and is run again with no reentry decision", stage)
	case rank > pr && prev.Decision != decStop:
		return fmt.Sprintf("stage %s follows %s, which did not stop (decision %q)", stage, prev.Stage, prev.Decision)
	}
	return ""
}

// attempts numbers the attempts of a visit to a stage and refuses a fourth one
// before a review (FR-CLB-027).
func (c *checker) attempts(newLineage bool) {
	r, prev := c.r(), c.st.prev
	continuing := prev != nil && !newLineage && prev.Stage == r.Stage &&
		prev.Decision != decStop && !strings.HasPrefix(prev.Decision, decReentry)
	want, n := 1, 1
	if continuing {
		want = prev.Attempt + 1
		n = c.st.attempts + 1
		if prev.Decision == decReview {
			n = 1
		}
	}
	c.st.attempts = n
	if r.Attempt != want {
		c.add(RuleAttemptNumber, "attempt %d, want %d", r.Attempt, want)
	}
	if n > maxAttempts {
		c.add(RuleAttempts, "attempt %d of stage %s with no review after %d attempts", r.Attempt, r.Stage, maxAttempts)
	}
}

// openExact refuses a search while an X or O fact of the pack fails (FR-CLB-036).
func (c *checker) openExact(rank int) {
	r := c.r()
	searching := len(r.Screens) > 0 || len(r.Sets) > 0 || c.searched()
	if searching && rank > 0 && len(c.st.openExact) > 0 {
		ids := make([]string, 0, len(c.st.openExact))
		for id := range c.st.openExact {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		c.add(RuleOpenExact, "stage %s searches while %s fails", r.Stage, strings.Join(ids, ", "))
	}
}

// blocks checks the seeds of the experiments: a block serves one choice, a search
// is on its block and outside 1 to 60, a confirm is on 1 to 50 and shares no seed
// with a search (FR-CLB-020, FR-CLB-021).
func (c *checker) blocks() {
	r := c.r()
	search, confirm := seedRanges(r.Experiments)
	if len(search) > 0 {
		c.searchBlock(search)
	}
	for _, s := range confirm {
		c.confirmSeeds(s, search)
	}
	if cf := r.Confirm; cf.Has && (cf.First < 1 || cf.Last > confirmLast || cf.First > cf.Last) {
		c.add(RuleConfirmSeed, "confirm line seeds %d..%d are outside 1..%d", cf.First, cf.Last, confirmLast)
	}
}

// seedRanges splits the seed ranges of the experiments into those of a search (the
// screen and the explore) and those of a confirm.
func seedRanges(es []Experiment) (search, confirm [][2]uint64) {
	for _, e := range es {
		first, last, ok := seedRange(e)
		switch {
		case !ok:
		case e.Step == stepScreen || e.Step == stepExplore:
			search = append(search, [2]uint64{first, last})
		case e.Step == stepConfirm:
			confirm = append(confirm, [2]uint64{first, last})
		}
	}
	return search, confirm
}

// confirmSeeds checks one confirm range: inside 1 to 50, and apart from every
// search range.
func (c *checker) confirmSeeds(s [2]uint64, search [][2]uint64) {
	if s[0] < 1 || s[1] > confirmLast {
		c.add(RuleConfirmSeed, "confirm seeds %d..%d are outside 1..%d", s[0], s[1], confirmLast)
	}
	for _, q := range search {
		if s[0] <= q[1] && q[0] <= s[1] {
			c.add(RuleConfirmSeed, "confirm seeds %d..%d share seeds with the search %d..%d", s[0], s[1], q[0], q[1])
		}
	}
}

// searchBlock checks one record that searches: its block is new to the pack and
// its search seeds are in the block and outside the stylised seeds.
func (c *checker) searchBlock(search [][2]uint64) {
	r := c.r()
	first, last, err := Block(r.Block)
	if err != nil {
		c.add(RuleBlockRange, "block %d is not a search block", r.Block)
		return
	}
	if c.st.blocks[r.Block] {
		c.add(RuleBlockReused, "block %d already served a choice of pack %s", r.Block, r.Pack)
	}
	for _, s := range search {
		if s[0] <= stylisedLast {
			c.add(RuleSearchSeed, "search seeds %d..%d reach the stylised seeds 1..%d", s[0], s[1], stylisedLast)
		}
		if s[0] < first || s[1] > last {
			c.add(RuleSearchSeed, "search seeds %d..%d are outside block %d (%d..%d)", s[0], s[1], r.Block, first, last)
		}
	}
}

// seedRange is the seeds an experiment draws: Seed0 for Replicas seeds. ok is
// false for no replicas or a range that leaves int64.
func seedRange(e Experiment) (first, last uint64, ok bool) {
	if e.Replicas < 1 || e.Seed0 > math.MaxInt64 {
		return 0, 0, false
	}
	end, ok := checked.Add(int64(e.Seed0), int64(e.Replicas)-1)
	if !ok {
		return 0, 0, false
	}
	return e.Seed0, uint64(end), true
}

// factors checks the number of factors against the cap, that each factor's knob
// belongs to the stage, and that an owner is not split across two sub-stages
// (FR-CLB-035).
func (c *checker) factors() {
	r := c.r()
	if len(r.Factors) > c.maxFactors {
		c.add(RuleMaxFactors, "%d factors, at most %d: split the stage by owner", len(r.Factors), c.maxFactors)
	}
	keys := make([]string, 0, len(r.Factors)+len(r.Sets))
	for _, f := range r.Factors {
		keys = append(keys, f.Key)
	}
	for _, s := range r.Sets {
		keys = append(keys, s.Key)
	}
	for _, key := range keys {
		ref, ok := c.idx.find(key)
		if !ok {
			continue
		}
		if ref.Stage != "" && ref.Stage != stageFamily(r.Stage) {
			c.add(RuleStageKnob, "%s (%s) is a knob of stage %s, not of %s", key, ref.ID, ref.Stage, r.Stage)
		}
		c.owner(ref)
	}
}

// owner records the sub-stage an owner is searched in and refuses a second one.
func (c *checker) owner(ref knobRef) {
	r := c.r()
	if r.Stage == stageFamily(r.Stage) || ref.Owner == "" {
		return
	}
	k := stageFamily(r.Stage) + "/" + ref.Owner
	if seen, ok := c.st.owners[k]; ok && seen != r.Stage {
		c.add(RuleOwnerSplit, "owner %s is searched in %s and in %s", ref.Owner, seen, r.Stage)
	}
	c.st.owners[k] = r.Stage
}

// lineage checks the first record of a new lineage: it names the old one as its
// parent, restarts at C1, carries no set, and carries no calibration overlay or
// warm-start table of an earlier lineage (FR-CLB-053, FR-CLB-054). It also
// refuses a derived document of a retired lineage in any later record.
func (c *checker) lineage(newLineage bool) {
	r, prev := c.r(), c.st.prev
	if newLineage {
		if r.Parent != prev.Lineage {
			c.add(RuleLineageParent, "lineage %s names parent %q, want %s", r.Lineage, r.Parent, prev.Lineage)
		}
		if r.Stage != stageC1 {
			c.add(RuleLineageRestart, "lineage %s starts at %s, want %s", r.Lineage, r.Stage, stageC1)
		}
		if len(r.Sets) > 0 {
			c.add(RuleLineageSet, "lineage %s carries %d set lines in its first record", r.Lineage, len(r.Sets))
		}
	}
	for _, d := range r.Documents {
		if derived(d.Name) && c.st.retired[d.Digest] {
			c.add(RuleLineageDoc, "document %s %s belongs to an earlier lineage", d.Name, d.Digest)
		}
	}
}

// derived reports whether a document is an overlay or a warm-start table.
func derived(name string) bool {
	return strings.HasPrefix(name, overlayDir) || strings.HasPrefix(name, warmPrefix)
}

// stop checks the seven conditions of 7.4.1 for a record that ends `stop`
// (FR-CLB-038).
func (c *checker) stop() {
	r := c.r()
	if r.Decision != decStop {
		return
	}
	for _, f := range r.Facts {
		if rule := stopRule(f); rule != "" {
			c.add(rule, "fact %s (class %c, target %q) is %s", f.ID, f.Class, f.Target, f.Result)
		}
	}
	for _, d := range r.Detectors {
		if d.Verdict != verdictNone && (d.Verdict != verdictUnknown || !d.Working) {
			c.add(RuleStopDetector, "detector %s is %s (working %v)", d.ID, d.Verdict, d.Working)
		}
	}
	c.stopRecord()
}

// stopRecord checks conditions 5, 6 and 7: no finding open, no moved D knob that
// stayed dormant, and a confirm with a control for a stage that searched.
func (c *checker) stopRecord() {
	r := c.r()
	for _, f := range r.Findings {
		if f.Status == statusOpen {
			c.add(RuleStopFinding, "finding %s %s/%s is open", f.Knob, f.Fact, f.Measure)
		}
	}
	for _, s := range r.Sets {
		if ref, ok := c.idx.find(s.Key); ok && ref.Class == 'D' && slices.Contains(r.Dormant, ref.ID) {
			c.add(RuleStopDormant, "dormant D knob %s is moved to %d", ref.ID, s.Value)
		}
	}
	if (len(r.Factors) > 0 || len(r.Screens) > 0 || len(r.Sets) > 0 || c.searched()) && (!r.Confirm.Has || !r.Confirm.Control) {
		c.add(RuleStopControl, "the stage searched and has no confirm with a control")
	}
}

// stopRule is the rule of conditions 1, 3 and 4 that a fact breaks, or "": an X or
// O fact must pass on every seed, and a B fact with a set key or a working target
// must pass on SR2 (a fact that does not apply, na, passes).
func stopRule(f Fact) string {
	if f.Result == resPass || f.Result == resNA {
		return ""
	}
	switch {
	case f.Class == 'X' || f.Class == 'O':
		return RuleStopExact
	case f.Class == 'B' && f.Target == targetSet:
		return RuleStopBSet
	case f.Class == 'B' && f.Target == targetWorking:
		return RuleStopBWorking
	}
	return ""
}

// retire closes the lineage the pack was in: its derived documents can no longer
// appear, and its open facts no longer bar a search of the new one.
func (st *packState) retire() {
	for d := range st.current {
		st.retired[d] = true
	}
	st.current = map[string]bool{}
	st.openExact = map[string]bool{}
	st.owners = map[string]string{} // a new lineage may split a stage by owner afresh
}

// fold folds the record into the pack's state once it is checked.
func (c *checker) fold() {
	r, st := c.r(), c.st
	for _, d := range r.Documents {
		if derived(d.Name) {
			st.current[d.Digest] = true
		}
	}
	for _, f := range r.Facts {
		if f.Class == 'X' || f.Class == 'O' {
			if f.Result == resFail {
				st.openExact[f.ID] = true
			} else if f.Result == resPass {
				delete(st.openExact, f.ID)
			}
		}
	}
	if r.Block >= 1 && c.searched() {
		st.blocks[r.Block] = true
	}
	st.prev = r
}

// searched reports whether the record has a screen or an explore experiment.
func (c *checker) searched() bool {
	for _, e := range c.r().Experiments {
		if e.Step == stepScreen || e.Step == stepExplore {
			return true
		}
	}
	return false
}
