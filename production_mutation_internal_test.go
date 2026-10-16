package economysim

import (
	"slices"
	"strings"
	"testing"
)

// rebuild refreshes the tables of a fixture after its configuration changed.
func (f *stepFixture) rebuild() {
	f.rt = newRecipeTable(f.cfg)
	f.s = newProductionScratch(f.cfg, &f.rt)
	f.resid = make([]int64, residueLen(f.all, &f.rt))
}

// panicMessage runs fn and returns the message it panics with, or "".
func panicMessage(fn func()) (msg string) {
	defer func() { msg, _ = recover().(string) }()
	fn()
	return ""
}

// TestYieldShocks_SetAboveBasePoints verifies that a factor above 10 000 is
// stored, inserted for an absent key and updated for a present one, and that
// only exactly 10 000 removes the key.
//
// Requirement: FR-PRO-023
func TestYieldShocks_SetAboveBasePoints(t *testing.T) {
	t.Parallel()
	var s yieldShocks
	steps := []struct {
		bp      int64
		want    int64
		entries int
	}{
		{15_000, 15_000, 1}, // absent key, above 10 000: inserted
		{20_000, 20_000, 1}, // present key, above 10 000: updated, not removed
		{10_000, 10_000, 0}, // present key at 10 000: removed
		{10_000, 10_000, 0}, // absent key at 10 000: nothing
	}
	for i, st := range steps {
		if err := s.set(1, 2, st.bp); err != nil {
			t.Fatalf("step %d: set(1, 2, %d) = %v, want nil", i, st.bp, err)
		}
		if got := s.factor(1, 2); got != st.want || len(s.entries) != st.entries {
			t.Errorf("step %d: factor after set(1, 2, %d) = %d with %d entries, want %d with %d", i, st.bp, got, len(s.entries), st.want, st.entries)
		}
	}
}

// TestLimitList_ToolInputPairing verifies that the defensive cap of 4.2.3
// applies only to the commodity that is both a tool and an input, and counts
// reserved units out of the stock.
//
// Requirement: FR-PRO-009
func TestLimitList_ToolInputPairing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		tool     string
		input    string
		stock    map[string]int64
		reserved map[string]int64
		want     int64
	}{
		// Tool peel (5) and input slag (6) differ: the tool cap is 4 / 1 = 4.
		{"input id above tool id", "peel", "slag", map[string]int64{"peel": 4, "slag": 10}, nil, 4},
		// Same commodity, q = 1: available 6 - 3 = 3, tool cap 3 / (1 + 1) = 1
		// (input cap 3); with 6 + 3 = 9 it would be 4, and the least would be 3.
		{"same commodity with reserved", "oven", "oven", map[string]int64{"oven": 6}, map[string]int64{"oven": 3}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			r := &f.cfg.Recipes[f.recipe("polish")]
			r.Requires = []Tool{{Commodity: tc.tool, WearPerUse: 1}}
			r.Inputs = []Item{{Commodity: tc.input, Quantity: 1}}
			f.rebuild()
			for c, n := range tc.stock {
				f.seed(c, n)
			}
			for c, n := range tc.reserved {
				f.reserve(c, n)
			}
			k, _, err := f.s.maxExecutions(f.l, f.req("polish", "north", spring, 100, 0))
			if err != nil || k != tc.want {
				t.Errorf("maxExecutions(polish) = (%d, %v), want (%d, nil)", k, err, tc.want)
			}
		})
	}
}

// TestStart_LongResiduesPanic verifies that a residue slice longer than the
// outputs of the recipe is refused, as a shorter one is.
//
// Guard: a residue slice of the wrong length would shift every later recipe's
// residues of the agent.
func TestStart_LongResiduesPanic(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	q := f.req("farm", "north", spring, 1, 0)
	mustPanicInvariant(t, "residues longer than the outputs", func() { f.s.start(f.l, &f.job, make([]int64, 3), q) })
}

// TestComplete_RefusesBadJobs verifies that a job with a negative execution
// count or a negative remaining is not a finished job.
//
// Guard: completing such a job would create output from a corrupt record.
func TestComplete_RefusesBadJobs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		executions int64
		remaining  int64
	}{{"negative executions", -1, 0}, {"negative remaining", 1, -1}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			if _, err := f.start(f.req("farm", "north", spring, 1, 0)); err != nil {
				t.Fatal(err)
			}
			f.job.executions, f.job.remaining = tc.executions, tc.remaining
			mustPanicInvariant(t, tc.name, func() { f.s.complete(f.l, &f.job, f.acct, MaxQuantity) })
		})
	}
}

// TestComplete_RoomPerCommodity verifies that the room of an output counts
// only the lines already cut for the same commodity, whatever the order of the
// commodity ids of the outputs.
//
// Requirement: FR-PRO-012, FR-PRO-029
func TestComplete_RoomPerCommodity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		recipe  string
		outputs []Output
		held    [2]string // commodities of the two outputs, in order
	}{
		{"lower id first", "fract", []Output{{Commodity: "ash", Quantity: 1, CostShare: 5_000}, {Commodity: "bread", Quantity: 1, CostShare: 5_000}}, [2]string{"ash", "bread"}},
		{"higher id first", "trace", []Output{{Commodity: "bread", Quantity: 1, CostShare: 5_000}, {Commodity: "ash", Quantity: 1, CostShare: 5_000}}, [2]string{"bread", "ash"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			f.cfg.Recipes[f.recipe(tc.recipe)].Outputs = tc.outputs
			f.rebuild()
			if res, err := f.start(f.req(tc.recipe, "north", spring, 1, 0)); err != nil || res.k != 1 {
				t.Fatalf("start = (%+v, %v)", res, err)
			}
			f.job.remaining = 0
			// Room 1 for each commodity: the second line is not charged for the first.
			c := f.s.complete(f.l, &f.job, f.acct, 1)
			if want := []int64{0, 0}; !slices.Equal(c.disposed, want) {
				t.Errorf("complete(%s).disposed = %v, want %v", tc.recipe, c.disposed, want)
			}
			for _, name := range tc.held {
				if h, _ := f.held(name); h != 1 {
					t.Errorf("%s held after complete(%s) = %d, want 1", name, tc.recipe, h)
				}
			}
		})
	}
}

// TestComplete_UnknownAccountMessages verifies which step refuses the
// completion into an account the ledger does not know: the read of the holding
// for a held output, the disposal for a discard-only output.
//
// Guard: the panic message names the step, so a swallowed read error cannot
// pass for the later refusal.
func TestComplete_UnknownAccountMessages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ recipe, want string }{
		{"farm", "holding of the principal unreadable"},
		{"scrap", "disposal of the surplus refused"},
	} {
		t.Run(tc.recipe, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			if _, err := f.start(f.req(tc.recipe, "north", spring, 1, 0)); err != nil {
				t.Fatal(err)
			}
			f.job.remaining = 0
			got := panicMessage(func() { f.s.complete(f.l, &f.job, 1<<20, MaxQuantity) })
			if !strings.Contains(got, tc.want) {
				t.Errorf("complete(%s) into an unknown account panicked with %q, want it to contain %q", tc.recipe, got, tc.want)
			}
		})
	}
}
