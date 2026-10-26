package economysim

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file tests the write path of the checkpoint from inside the package: the
// section table against the spec, the options, and the layout of each section
// against the state it was written from, read with the test's own codec.Reader
// calls (plan 2-2 Phase 4). The synthetic values are set on the world's own
// fields, because only a few ticks of a small mill give a state with a job, an
// estate listing or a run above 0.

// saveTag writes the section with the tag of w and returns its bytes.
func saveTag(t *testing.T, w *World, tag uint64) []byte {
	t.Helper()
	for i := range checkpointSections {
		if checkpointSections[i].tag == tag {
			cw := codec.NewWriter(nil)
			if err := checkpointSections[i].save(w, cw); err != nil {
				t.Fatalf("save of tag %d error = %v, want nil", tag, err)
			}
			return cw.Out()
		}
	}
	t.Fatalf("no section with tag %d", tag)
	return nil
}

// ckptMill builds the world of millConfig(3, 4): one estate (agent 0) and four
// agents (1 to 4) of weight 1, with a recipe of 3 ticks, stepped n times.
func ckptMill(t *testing.T, n int, opts ...Option) *World {
	t.Helper()
	w := newMill(t, millConfig(3, 4), append([]Option{WithSeed(5)}, opts...)...)
	step(t, w, n)
	return w
}

// TestSections_AgreeWithSpec verifies that the section table and the list of
// sections not built are the table of spec-checkpoint.md 4.2, read by a plain
// text parse: every tag of the spec is built or listed, once, in ascending order,
// with the name of the spec and, for a section of a module, the module that gates
// it; a section with no gate is present with no condition.
//
// Requirement: FR-CHK-008, FR-CHK-040
//
// Guard: a section dropped from the table, or one built and still listed, would
// change the shape of every checkpoint without a test noticing.
func TestSections_AgreeWithSpec(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("doc/specs/spec-checkpoint.md")
	if err != nil {
		t.Fatalf("read the spec: %v", err)
	}
	type row struct{ name, gate string }
	spec := map[uint64]row{}
	rowRe := regexp.MustCompile(`^\| (\d+) \| ([A-Za-z]+) \|`)
	gateRe := regexp.MustCompile("`Modules\\.([A-Za-z]+)`")
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "### 4.2 The sections"):
			in = true
		case strings.HasPrefix(line, "4.2.1."):
			in = false
		case in && rowRe.MatchString(line):
			m := rowRe.FindStringSubmatch(line)
			tag, _ := strconv.ParseUint(m[1], 10, 64)
			cells := strings.Split(line, "|")
			last := strings.TrimSpace(cells[len(cells)-2])
			r := row{name: m[2]}
			if g := gateRe.FindStringSubmatch(last); g != nil {
				r.gate = g[1]
			} else if !strings.HasPrefix(last, "always") {
				t.Fatalf("tag %d: Present when = %q, want \"always\" or a module", tag, last)
			}
			spec[tag] = r
		}
	}
	if len(spec) != 34 {
		t.Fatalf("parsed %d rows of 4.2, want 34: the parse is broken", len(spec))
	}
	seen := map[uint64]bool{}
	var prev uint64
	for _, s := range checkpointSections {
		want, ok := spec[s.tag]
		switch {
		case !ok:
			t.Errorf("section %d (%s) is not a row of 4.2", s.tag, s.name)
		case s.name != want.name:
			t.Errorf("section %d is named %q, want %q", s.tag, s.name, want.name)
		case (want.gate != "") != (s.present != nil):
			t.Errorf("section %d is built with present set = %v, the spec gates it by %q: present is set exactly for a module's section", s.tag, s.present != nil, want.gate)
		case s.tag <= prev:
			t.Errorf("section %d follows %d, want ascending tags", s.tag, prev)
		}
		prev, seen[s.tag] = s.tag, true
	}
	prev = 0
	for _, n := range notBuiltSections {
		want, ok := spec[n.tag]
		switch {
		case !ok:
			t.Errorf("not built section %d (%s) is not a row of 4.2", n.tag, n.name)
		case seen[n.tag]:
			t.Errorf("section %d is built and still listed in notBuiltSections: delete the stale entry", n.tag)
		case n.name != want.name || n.gate != want.gate:
			t.Errorf("not built section %d = %q gated by %q, want %q gated by %q", n.tag, n.name, n.gate, want.name, want.gate)
		case n.tag <= prev:
			t.Errorf("not built section %d follows %d, want ascending tags", n.tag, prev)
		}
		prev, seen[n.tag] = n.tag, true
	}
	for tag := range spec {
		if !seen[tag] {
			t.Errorf("tag %d of 4.2 is neither built nor listed in notBuiltSections", tag)
		}
	}
}

// seedOptions is the part of options the seed options set, which is comparable.
type seedOptions struct {
	seed, reseed           uint64
	seedGiven, reseedGiven bool
}

// TestOptions_ReseedFields verifies what the options record for the restore: a
// seed given is recorded as given even at 0, a salt is recorded as given even at
// 0, the last of each wins, and none given leaves both unset.
//
// Requirement: FR-CHK-020
func TestOptions_ReseedFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []Option
		want seedOptions
	}{
		{"none", nil, seedOptions{}},
		{"WithSeed(0) is given", []Option{WithSeed(0)}, seedOptions{seedGiven: true}},
		{"WithReseed(0) is given", []Option{WithReseed(0)}, seedOptions{reseedGiven: true}},
		{"last of each wins", []Option{WithSeed(1), WithReseed(2), WithSeed(3), WithReseed(4)},
			seedOptions{seed: 3, seedGiven: true, reseed: 4, reseedGiven: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := applyOptions(tt.opts)
			if err != nil {
				t.Fatalf("applyOptions() error = %v, want nil", err)
			}
			got := seedOptions{seed: o.seed, seedGiven: o.seedGiven, reseed: o.reseed, reseedGiven: o.reseedGiven}
			if got != tt.want {
				t.Errorf("applyOptions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestSave_Sections verifies the layout of tags 1, 9, 10, 11 and 34 against bytes
// computed by hand, for the world of millConfig(3, 4): one region, one role, an
// estate and four agents.
//
// Arithmetic. Tag 1: the next agent ID is 5 (the estate is 0, the agents 1 to 4)
// and the world has not stepped, so U(5) B(true) = 05 01; after one step the flag
// is 00. Tag 9 with the synthetic values renewal carry 7, unfunded run 2, empty
// run 3 and no run below the floor: the renewal rows are count 1 then (0, 0, 7) =
// 01 00 00 07; the bit "had an agent" is set for (0, 0): 01 00 00; the unfunded run
// by region is count 1 then (0, 2) = 01 00 02; the empty run is 01 00 00 03; the
// run below the floor is 00. Tag 10 is four counts of 0 and tag 11 one count of 0
// (the skeletons of spec-checkpoint.md 4.2.8). Tag 34: one entry, kind 00, the
// name "directed" as its length 08 and the eight bytes, version 00 and no state
// (length 00), after the count 01.
//
// Requirement: FR-CHK-040, FR-CHK-043, FR-CHK-044
func TestSave_Sections(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 0)
	w.state.renewalCarry[0], w.state.unfundedRun[0], w.state.extinctRun[0] = 7, 2, 3
	// The step comes last: it runs the spawner, which resets the synthetic runs.
	tests := []struct {
		name string
		tag  uint64
		prep func()
		want []byte
	}{
		{"world before the first step", 1, nil, []byte{5, 1}},
		{"spawner", 9, nil, []byte{1, 0, 0, 7, 1, 0, 0, 1, 0, 2, 1, 0, 0, 3, 0}},
		{"skills skeleton", 10, nil, []byte{0, 0, 0, 0}},
		{"regions skeleton", 11, nil, []byte{0}},
		{"extensions", 34, nil, append([]byte{1, 0, 8}, append([]byte("directed"), 0, 0)...)},
		{"world after a step", 1, func() { step(t, w, 1) }, []byte{5, 0}},
	}
	for _, tt := range tests {
		if tt.prep != nil {
			tt.prep()
		}
		if got := saveTag(t, w, tt.tag); !bytes.Equal(got, tt.want) {
			t.Errorf("%s: tag %d = %x, want %x", tt.name, tt.tag, got, tt.want)
		}
	}
}

// TestSave_SparseRowsSkipDefaults verifies that a run at 0 is not a row: an all
// zero list is one count of 0, and the rows that differ are written in ascending
// key order as (region, role, value), or (region, value) for a run kept by
// region.
//
// Arithmetic: with 3 roles the index 1 is (region 0, role 1) and the index 5 is
// (region 1, role 2). The rows are count 2, then 00 01 05 and 01 02 09.
//
// Requirement: FR-CHK-014
func TestSave_SparseRowsSkipDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		f    func(*codec.Writer)
		want []byte
	}{
		{"all zero", func(cw *codec.Writer) { writeSparse(cw, make([]int64, 6), 3) }, []byte{0}},
		{"two rows", func(cw *codec.Writer) { writeSparse(cw, []int64{0, 5, 0, 0, 0, 9}, 3) }, []byte{2, 0, 1, 5, 1, 2, 9}},
		{"by region, all zero", func(cw *codec.Writer) { writeSparseByRegion(cw, make([]int64, 3)) }, []byte{0}},
		{"by region", func(cw *codec.Writer) { writeSparseByRegion(cw, []int64{0, 4, 0, 130}) }, []byte{2, 1, 4, 3, 0x82, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cw := codec.NewWriter(nil)
			tt.f(cw)
			if got := cw.Out(); !bytes.Equal(got, tt.want) {
				t.Errorf("rows = %x, want %x", got, tt.want)
			}
		})
	}
}

// ckptAgent is one decoded agent record of tag 3.
type ckptAgent struct {
	id, account, role, weight, flags, strat, unfed, hunger, privation uint64
	roleSince, deathCarry                                             int64
	ownUse, ownUseJobs                                                uint64
	journey, directive                                                bool
}

// readAgent reads the fixed part of an agent record.
func readAgent(r *codec.Reader) ckptAgent {
	var a ckptAgent
	a.id, a.account, a.role, a.weight, a.flags, a.strat = r.U(), r.U(), r.U(), r.U(), r.U(), r.U()
	a.unfed, a.hunger, a.privation = r.U(), r.U(), r.U()
	a.roleSince, a.deathCarry = r.I(), r.I()
	a.ownUse, a.ownUseJobs, a.journey, a.directive = r.U(), r.U(), r.B(), r.B()
	return a
}

// TestSave_AgentsAndHostedFields verifies tag 3: the table of strategy names, the
// count, one record per live agent in ascending ID with the fields of
// spec-checkpoint.md 4.2 in order, a removed agent leaving a gap and no row, and
// the hosted fields written always: DeathCarry as offset(ID), OwnUse and
// OwnUseJobs as 0, the journey and the directive as absent.
//
// Arithmetic: offset(17) = ((17 * 2 654 435 769) mod 2^32) * D / 2^32 = 1 849 008
// for D = 3 650 000 (spec-population.md 3.8.3 and its worked example), which is
// 10 000 times the 365 ticks of the default year.
//
// Requirement: FR-CHK-041, FR-CHK-046; N-217
func TestSave_AgentsAndHostedFields(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 4)
	if got := w.deathCarry(17); got != 1_849_008 {
		t.Errorf("deathCarry(17) = %d, want 1849008 (spec-population.md 3.8.3)", got)
	}
	// Agent 2 leaves: its slot is the zero agent and the next ID stays 5.
	if err := w.removeAgent(&w.agents[2], CauseStarved); err != nil {
		t.Fatalf("removeAgent() error = %v, want nil", err)
	}
	w.state.unfedRun[0] = 3
	r := codec.NewReader(saveTag(t, w, 3))
	if n := r.Count(1); n != 1 || string(r.Bytes()) != "default" {
		t.Fatalf("strategy name table has %d entries, want the one name \"default\"", n)
	}
	if n := r.Count(1); n != 3 {
		t.Fatalf("agent count = %d, want 3 (agents 1, 3 and 4)", n)
	}
	for _, id := range []AgentID{1, 3, 4} {
		a := w.agents[id]
		want := ckptAgent{
			id: uint64(id), account: uint64(a.account), role: 0, weight: 1, strat: 0,
			unfed: uint64(a.unfed), hunger: uint64(a.hunger), privation: uint64(a.privation),
			roleSince: a.roleSince, deathCarry: w.deathCarry(id),
		}
		if got := readAgent(r); got != want {
			t.Errorf("record of agent %d = %+v, want %+v", id, got, want)
		}
	}
	// The unfed run: one row, (region 0, role 0, 3).
	if rows := r.Count(3); rows != 1 || r.U() != 0 || r.U() != 0 || r.U() != 3 {
		t.Errorf("unfed run rows = %d, want the one row (0, 0, 3)", rows)
	}
	if err := r.Done(); err != nil {
		t.Errorf("tag 3 has trailing bytes or an error: %v", err)
	}
}

// TestSave_Directive verifies the bytes of a directive in the record of an agent:
// the flag, the job request when there is one, the intents, the discards, the
// holds and Sticky.
//
// Arithmetic: the directive holds the job (recipe 1, 2 executions, NoJob false),
// one intent (commodity 1, Ask, 3 units, limit 4, target 5, reserve 6, unit cost
// 7, purpose input) and one discard (commodity 1, 9 units), the hold list {0} and
// Sticky. The bytes are present 01, job 01, NoJob 00, 01 02, one intent 01, then
// 01 side 03 04 05 06 07 purpose, one discard 01 then 01 09, one hold 01 then 00
// and Sticky 01. Side is the int value of Ask and purpose that of PurposeInput.
//
// Requirement: FR-CHK-041
func TestSave_Directive(t *testing.T) {
	t.Parallel()
	d := &Directive{
		Job:      &JobRequest{Recipe: 1, Executions: 2},
		Intents:  []Intent{{Commodity: 1, Side: Ask, Quantity: 3, Limit: 4, Target: 5, Reserve: 6, UnitCost: 7, Purpose: PurposeInput}},
		Discards: []Discard{{Commodity: 1, Quantity: 9}},
		Hold:     []CommodityID{0},
		Sticky:   true,
	}
	cw := codec.NewWriter(nil)
	saveDirective(cw, d)
	want := []byte{1, 1, 0, 1, 2, 1, 1, byte(Ask), 3, 4, 5, 6, 7, byte(PurposeInput), 1, 1, 9, 1, 0, 1}
	if got := cw.Out(); !bytes.Equal(got, want) {
		t.Errorf("directive = %x, want %x", got, want)
	}
	cw = codec.NewWriter(nil)
	saveDirective(cw, &Directive{})
	if got, want := cw.Out(), []byte{1, 0, 0, 0, 0, 0}; !bytes.Equal(got, want) {
		t.Errorf("empty directive = %x, want %x (present, no job, three counts of 0, not sticky)", got, want)
	}
	cw = codec.NewWriter(nil)
	saveDirective(cw, nil)
	if got := cw.Out(); !bytes.Equal(got, []byte{0}) {
		t.Errorf("no directive = %x, want 00", got)
	}
	// The agent record carries the same bytes at its end, with the flags.
	w := ckptMill(t, 0)
	w.agents[1].external, w.agents[1].mortal, w.agents[1].directive = true, true, d
	body := saveTag(t, w, 3)
	if !bytes.Contains(body, want) {
		t.Errorf("tag 3 = %x, want it to contain the directive %x", body, want)
	}
	r := codec.NewReader(body[1+len("default")+1+1:]) // table: count, length, name; then the agent count
	if a := readAgent(r); a.flags != 3 || !a.directive {
		t.Errorf("first record = %+v, want flags 3 (External and Mortal) and a directive", a)
	}
}

// TestSave_DirectiveNoJobIgnoresTheOtherJobFields verifies that a job request with
// NoJob writes 0 for its recipe and executions, whatever they hold.
//
// Fix: review of stage-2-2b. The fields are documented as ignored with NoJob, but
// were written as they were, and a recipe out of range made a checkpoint that
// RestoreWorld refused.
func TestSave_DirectiveNoJobIgnoresTheOtherJobFields(t *testing.T) {
	t.Parallel()
	cw := codec.NewWriter(nil)
	saveDirective(cw, &Directive{Job: &JobRequest{NoJob: true, Recipe: 9999, Executions: 7}})
	if got, want := cw.Out(), []byte{1, 1, 1, 0, 0, 0, 0, 0, 0}; !bytes.Equal(got, want) {
		t.Errorf("saveDirective(NoJob with recipe 9999) = %x, want %x (present, job, NoJob, recipe 0, executions 0, three counts of 0, not sticky)", got, want)
	}
}

// TestSave_MarketsBeliefsJobsWelfare verifies tags 4 to 7 against the state of a
// mill that has run 4 ticks, with the synthetic values the run leaves at 0 set on
// it: a best bid and ask in a report, a bound run, a yield shock and the runs of
// the floor and the clamp.
//
// Requirement: FR-CHK-042
func TestSave_MarketsBeliefsJobsWelfare(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 4)
	w.markets.reports[0].HasBestBid, w.markets.reports[0].BestBid = true, 1_234
	w.markets.reports[1].HasBestAsk, w.markets.reports[1].BestAsk = true, 4_321
	w.markets.boundRun[1] = 3
	if err := w.shocks.set(0, 0, 5_000); err != nil {
		t.Fatalf("shock set error = %v, want nil", err)
	}
	w.state.floorRun[0], w.state.clampRun[0] = 2, 0
	t.Run("markets", func(t *testing.T) {
		r := codec.NewReader(saveTag(t, w, 4))
		for i := range w.markets.reports {
			rep := w.markets.reports[i]
			got := marketReport{Price: Price(r.U()), Age: int64(r.U()), Volume: Quantity(r.U()), UnmatchedBid: Quantity(r.U()), UnmatchedAsk: Quantity(r.U())}
			if got.HasBestBid = r.B(); got.HasBestBid {
				got.BestBid = Price(r.U())
			}
			if got.HasBestAsk = r.B(); got.HasBestAsk {
				got.BestAsk = Price(r.U())
			}
			if got != rep {
				t.Errorf("market %d report = %+v, want %+v", i, got, rep)
			}
			var prices []int64
			for range r.Count(1) {
				prices = append(prices, int64(r.U()))
			}
			if want := w.markets.windows[i].Values(nil); !slices.Equal(prices, want) {
				t.Errorf("market %d window = %v, want %v, oldest first", i, prices, want)
			}
			if run := int64(r.U()); run != w.markets.boundRun[i] {
				t.Errorf("market %d bound run = %d, want %d", i, run, w.markets.boundRun[i])
			}
		}
		if err := r.Done(); err != nil {
			t.Errorf("tag 4: %v", err)
		}
	})
	t.Run("beliefs", func(t *testing.T) {
		r := codec.NewReader(saveTag(t, w, 5))
		for id := 1; id <= 4; id++ {
			for i, b := range w.agents[id].beliefs {
				if c, wd := r.U(), r.U(); int64(c) != b.Centre || int64(wd) != b.Width {
					t.Errorf("agent %d belief %d = (%d, %d), want (%d, %d)", id, i, c, wd, b.Centre, b.Width)
				}
			}
		}
		if err := r.Done(); err != nil {
			t.Errorf("tag 5: %v", err)
		}
	})
	t.Run("jobs", func(t *testing.T) {
		r := codec.NewReader(saveTag(t, w, 6))
		jobs := 0
		for id := 1; id <= 4; id++ {
			a := &w.agents[id]
			if has := r.B(); has != a.hasJob() {
				t.Fatalf("agent %d has a job = %v, want %v", id, has, a.hasJob())
			} else if has {
				jobs++
				j := a.job
				if got := [5]int64{int64(r.U()), int64(r.U()), int64(r.U()), int64(r.U()), int64(r.U())}; got != [5]int64{int64(j.recipe), j.remaining, j.executions, int64(j.principal), j.yield} {
					t.Errorf("agent %d job = %v, want %+v", id, got, j)
				}
				for i, p := range j.pending {
					if got := int64(r.U()); got != p {
						t.Errorf("agent %d pending %d = %d, want %d", id, i, got, p)
					}
				}
			}
			for i, res := range a.residues {
				if got := int64(r.U()); got != res {
					t.Errorf("agent %d residue %d = %d, want %d", id, i, got, res)
				}
			}
		}
		if jobs == 0 {
			t.Errorf("no agent has a job after 4 ticks of a 3-tick recipe, want at least one to read")
		}
		if n := r.Count(3); n != 1 || r.U() != 0 || r.U() != 0 || r.U() != 5_000 {
			t.Errorf("yield shock rows = %d, want the one row (0, 0, 5000)", n)
		}
		if err := r.Done(); err != nil {
			t.Errorf("tag 6: %v", err)
		}
	})
	t.Run("welfare", func(t *testing.T) {
		r := codec.NewReader(saveTag(t, w, 7))
		for id := 1; id <= 4; id++ {
			a := &w.agents[id]
			recs := a.history.AppendRecords(nil)
			if n := r.Count(1); n != len(recs) || n == 0 {
				t.Fatalf("agent %d holds %d records, want %d and at least one", id, n, len(recs))
			}
			for i, rec := range recs {
				got := welfare.Record{Income: r.I(), Cost: r.I(), Relief: r.I(), Unearned: r.I(), OwnReceipt: r.I(), OwnHH: int64(r.U())}
				got.Working, got.WorkHH = r.B(), int64(r.U())
				if got != rec {
					t.Errorf("agent %d record %d = %+v, want %+v", id, i, got, rec)
				}
			}
			if target := int64(r.U()); target != a.target {
				t.Errorf("agent %d target = %d, want %d", id, target, a.target)
			}
		}
		if n := r.Count(3); n != 1 || r.U() != 0 || r.U() != 0 || r.U() != 2 {
			t.Errorf("floor run rows = %d, want the one row (0, 0, 2)", n)
		}
		if n := r.Count(3); n != 0 {
			t.Errorf("clamp run rows = %d, want 0", n)
		}
		if err := r.Done(); err != nil {
			t.Errorf("tag 7: %v", err)
		}
	})
}

// TestSave_EstateListings verifies tag 8: after an agent is removed its estate
// holds its goods, a listing for one of the lots is written as (commodity, born,
// durability, price), a held lot with no listing writes no row, and a region with
// none writes a count of 0.
//
// Arithmetic: the agent holds 10 grain at the start and eats 1 per tick of the 4
// ticks it lived, so the estate gets the rest of its stock; the one lot of grain
// (born at the creation tick, durability 0) is listed at 777.
//
// Requirement: FR-CHK-042, FR-EST-024; N-256
func TestSave_EstateListings(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 0)
	if got := saveTag(t, w, 8); !bytes.Equal(got, []byte{0}) {
		t.Errorf("tag 8 before any removal = %x, want 00 (one region, no listing)", got)
	}
	if err := w.removeAgent(&w.agents[1], CauseStarved); err != nil {
		t.Fatalf("removeAgent() error = %v, want nil", err)
	}
	lots := w.estates.heldLots(w.ledger, 0)
	if len(lots) == 0 {
		t.Fatalf("the estate holds no lot after a removal, want the agent's goods")
	}
	if got := saveTag(t, w, 8); !bytes.Equal(got, []byte{0}) {
		t.Errorf("tag 8 with held lots and no listing = %x, want 00: a lot with no row is not written", got)
	}
	w.estates.listed[0] = []listing{{key: keyOf(lots[0]), price: 777}}
	r := codec.NewReader(saveTag(t, w, 8))
	if n := r.Count(4); n != 1 {
		t.Fatalf("listing rows = %d, want 1", n)
	}
	if c, born, dur, price := int(r.U()), r.I(), int64(r.U()), r.U(); c != lots[0].Commodity || born != lots[0].Born || dur != lots[0].Durability || price != 777 {
		t.Errorf("listing = (%d, %d, %d, %d), want (%d, %d, %d, 777)", c, born, dur, price, lots[0].Commodity, lots[0].Born, lots[0].Durability)
	}
	if err := r.Done(); err != nil {
		t.Errorf("tag 8: %v", err)
	}
}

// A negative value in a field the checkpoint writes unsigned is a violated
// invariant at write time, not a number the restore refuses.
//
// Requirement: FR-CHK-040
func TestUnsignedRefusesNegative(t *testing.T) {
	if got := unsigned(int64(7)); got != 7 {
		t.Errorf("unsigned(7) = %d, want 7", got)
	}
	if got := unsigned(0); got != 0 {
		t.Errorf("unsigned(0) = %d, want 0", got)
	}
	if msg := invariantMessage(func() { unsigned(int64(-1)) }); !strings.Contains(msg, "negative value") {
		t.Errorf("unsigned(-1) panic = %q, want the negative-value invariant", msg)
	}
}

// TestSave_EstatesHaveNoAgentRowAndRestoreRebuildsThem verifies that tag 3 holds
// a record for each of the four agents and none for the estate (ID 0, past the
// regions), and that the restored world binds its estate to the same ledger
// account the original held.
//
// Arithmetic: ckptMill has one region, so the estate is agent 0 and the agents are
// 1 to 4; the records carry the IDs 1, 2, 3 and 4.
//
// Requirement: FR-CHK-048
func TestSave_EstatesHaveNoAgentRowAndRestoreRebuildsThem(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 3)
	r := codec.NewReader(saveTag(t, w, 3)[1+len("default")+1:]) // past the strategy table
	var ids []uint64
	for range r.Count(1) {
		ids = append(ids, readAgent(r).id)
	}
	if want := []uint64{1, 2, 3, 4}; !slices.Equal(ids, want) {
		t.Errorf("tag 3 record IDs = %v, want %v (no row for the estate)", ids, want)
	}
	got, err := restoreOf(t, w)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if !slices.Equal(got.estates.accounts, w.estates.accounts) {
		t.Errorf("restored estate accounts = %v, want %v", got.estates.accounts, w.estates.accounts)
	}
}

// TestEachLive_SkipsEstatesAndRemovedSlots verifies that eachLive, the one liveness
// rule of tags 3, 5, 6 and 7, visits exactly the slots past the estates that are
// not the zero agent of a removed agent, in ascending ID.
//
// Arithmetic: ckptMill has one estate (slot 0) and agents 1 to 4; zeroing slot 3
// leaves 1, 2 and 4.
//
// Requirement: FR-CHK-014
//
// Guard: a liveness test that drifts from the removed-agent slot would add or drop
// a record in every per-agent section with no error.
func TestEachLive_SkipsEstatesAndRemovedSlots(t *testing.T) {
	t.Parallel()
	w := ckptMill(t, 0)
	w.agents[3] = agent{}
	var want, got []AgentID
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if !reflect.DeepEqual(w.agents[i], agent{}) {
			want = append(want, w.agents[i].id)
		}
	}
	w.eachLive(func(a *agent) { got = append(got, a.id) })
	if !slices.Equal(got, want) || len(got) != 3 {
		t.Errorf("eachLive visited %v, want %v (three agents)", got, want)
	}
}

// TestCheckpointConstructors_GoThroughParseFrame verifies that the bytes of a
// Checkpoint are set in two places only, Checkpoint (which makes the frame) and
// UnmarshalBinary (which parses it first), so Info may skip the digest.
//
// Requirement: FR-CHK-001
//
// Guard: a third place that sets the bytes would let Info report the header of
// bytes nobody checked.
func TestCheckpointConstructors_GoThroughParseFrame(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("checkpoint.go")
	if err != nil {
		t.Fatalf("read checkpoint.go: %v", err)
	}
	set := regexp.MustCompile(`Checkpoint\{b:|\bc\.b = `)
	if n := len(set.FindAll(src, -1)); n != 2 {
		t.Errorf("checkpoint.go sets the bytes of a Checkpoint in %d places, want 2", n)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the directory: %v", err)
	}
	for _, f := range files {
		name := f.Name()
		if name == "checkpoint.go" || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if set.Match(b) {
			t.Errorf("%s sets the bytes of a Checkpoint outside checkpoint.go", name)
		}
	}
}

// savedOrDerived says, for every field of the state structs, where its value lives in
// a checkpoint ("saved N": the section with tag N) or why it needs none ("derived":
// CHK 4.3's rows; "scratch": rewritten before it is read; "not state": an option, the
// error of a failed step). A field of these structs that is in no row fails
// TestFields_EveryLiveFieldSavedOrDerived: a new field must be placed here, in the
// change that adds it, so a field that holds state cannot be forgotten by a section.
var savedOrDerived = map[string]string{
	"World.cfg": "derived: the configuration is the caller's (CHK 4.3, 6.2 step 3)", "World.hash": "derived: of cfg",
	"World.obs": "not state: an option", "World.log": "not state: an option", "World.audit": "not state: an option",
	"World.ext": "not state: the registered extensions, matched at restore (CHK 7)", "World.reg": "not state: the registrations, walked for tag 34",
	"World.ledger": "saved 2", "World.recipes": "derived: of cfg", "World.roles": "derived: of cfg", "World.days": "derived: CAL row",
	"World.yields": "derived: PRO row", "World.geo": "saved 11: the edge statuses; the tables are derived of cfg and the statuses",
	"World.refs": "derived: of cfg (REG 5.3.5)", "World.bounds": "derived: built at construction by the modules that are on, empty with none (REG 6.5.2)", "World.shocks": "saved 6", "World.rnd": "derived: the seed is in the header (ADR 0008)",
	"World.demand":  "derived: the ladders of cfg; the carries and the choke runs are saved 12 when the module's hooks are built (DEM 12)",
	"World.estates": "saved 8", "World.agents": "saved 3", "World.live": "derived: counters rebuilt from the agents",
	"World.markets": "saved 4", "World.bp": "derived: of cfg", "World.clr": "scratch", "World.receipts": "scratch",
	"World.prod": "scratch: the cost anchors, derived at P1.2 (PRO row)", "World.refused": "scratch", "World.welfare": "scratch",
	"World.reads": "scratch", "World.views": "scratch", "World.tk": "scratch", "World.pr": "scratch", "World.oc": "scratch",
	"World.ex": "scratch", "World.tally": "scratch", "World.runnable": "scratch: rebuilt at P1.2", "World.seedWage": "derived: of cfg",
	"World.state": "saved 1, 3, 7, 9", "World.lotBuf": "scratch", "World.failed": "not state: a failed world is not checkpointed",
	"World.stepping": "not state: the in-step flag",

	"worldState.seed": "saved: the header", "worldState.nextAgent": "saved 1", "worldState.unfedRun": "saved 3",
	"worldState.floorRun": "saved 7", "worldState.extinctRun": "saved 9", "worldState.belowFloorRun": "saved 9",
	"worldState.clampRun": "saved 7", "worldState.unfundedRun": "saved 9", "worldState.renewalCarry": "saved 9",
	"worldState.hadAgent": "saved 9",

	"agent.id": "saved 3", "agent.account": "saved 3", "agent.role": "saved 3", "agent.weight": "saved 3",
	"agent.external": "saved 3", "agent.mortal": "saved 3", "agent.strategy": "saved 3", "agent.beliefs": "saved 5",
	"agent.job": "saved 6", "agent.residues": "saved 6", "agent.history": "saved 7", "agent.target": "saved 7",
	"agent.unfed": "saved 3", "agent.hunger": "saved 3", "agent.privation": "saved 3", "agent.roleSince": "saved 3",
	"agent.directive": "saved 3",

	"job.recipe": "saved 6", "job.remaining": "saved 6", "job.executions": "saved 6", "job.principal": "saved 6",
	"job.yield": "saved 6", "job.pending": "saved 6",

	"marketTable.commodities": "derived: of cfg", "marketTable.reports": "saved 4", "marketTable.windows": "saved 4",
	"marketTable.boundRun": "saved 4", "marketTable.books": "scratch: the book of the tick in progress",
	"marketTable.intents": "scratch", "marketTable.slotOf": "scratch", "marketTable.asks": "scratch", "marketTable.trades": "scratch",

	"estates.accounts": "derived: the first non-boundary accounts of the ledger (attachEstates)", "estates.listed": "saved 8",
	"estates.lots": "scratch", "estates.scratch": "scratch",
}

// TestFields_EveryLiveFieldSavedOrDerived verifies, by reading the source, that every
// field of World, worldState, agent, job, marketTable and estates is named in
// savedOrDerived, and that the table names no field that is gone. A field added to
// one of these structs fails the test until the author says whether a section
// saves it or why none need.
//
// Requirement: FR-CHK-045
//
// Guard: the restore tests compare a restored world with the original over a few
// ticks, and a new field holding state that no section saves gives the same
// bytes until the day it matters; this fails the day the field is added.
func TestFields_EveryLiveFieldSavedOrDerived(t *testing.T) {
	t.Parallel()
	structs := map[string]bool{"World": true, "worldState": true, "agent": true, "job": true, "marketTable": true, "estates": true}
	files := []string{"world.go", "agent.go", "market.go", "production.go", "welfare.go", "spawner.go", "estates.go"}
	fset := token.NewFileSet()
	found := map[string]bool{}
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v, want nil", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || !structs[ts.Name.Name] {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, fl := range st.Fields.List {
				if len(fl.Names) == 0 {
					t.Errorf("%s.%s: an embedded field has no name to list; name it", ts.Name.Name, name)
				}
				for _, id := range fl.Names {
					key := ts.Name.Name + "." + id.Name
					found[key] = true
					if _, ok := savedOrDerived[key]; !ok {
						t.Errorf("%s (%s) is in no row of savedOrDerived: say which section saves it or why none need", key, name)
					}
				}
			}
			return true
		})
	}
	for key := range savedOrDerived {
		if !found[key] {
			t.Errorf("savedOrDerived names %s, which is not a field of the structs in %v", key, files)
		}
	}
}

// TestCheckpoint_NoFloatsNoMapsNoTime verifies by syntax that the code which writes
// and reads a checkpoint (checkpoint*.go and internal/codec, tests excluded) has no
// floating-point type, no map type, no reference to the time or rand packages and no
// goroutine: the bytes are the same on every platform and every run.
//
// Requirement: FR-CHK-015, NFR-CHK-005
//
// Guard: a map range in a writer gives a different byte order on each run, and a
// float gives a different one on arm64 (architecture.md 3.5, 7.5); neither shows in
// a single-platform test.
func TestCheckpoint_NoFloatsNoMapsNoTime(t *testing.T) {
	t.Parallel()
	var files []string
	for _, pattern := range []string{"checkpoint*.go", "internal/codec/*.go"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("Glob(%q) error = %v, want nil", pattern, err)
		}
		files = append(files, m...)
	}
	files = slices.DeleteFunc(files, func(f string) bool { return strings.HasSuffix(f, "_test.go") })
	if len(files) < 5 {
		t.Fatalf("found %v, want the checkpoint files and the codec", files)
	}
	for _, f := range files {
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.MapType:
				t.Errorf("%s: map type", f)
			case *ast.GoStmt:
				t.Errorf("%s: go statement", f)
			case *ast.Ident:
				if n.Name == "float32" || n.Name == "float64" {
					t.Errorf("%s: %s", f, n.Name)
				}
			case *ast.SelectorExpr:
				if x, ok := n.X.(*ast.Ident); ok && (x.Name == "time" || x.Name == "rand") {
					t.Errorf("%s: %s.%s", f, x.Name, n.Sel.Name)
				}
			}
			return true
		})
	}
}
