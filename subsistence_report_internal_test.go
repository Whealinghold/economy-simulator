package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// twoRegions adds a second region "away" to the forage world with its own
// households of the same role and the yield of gather, so the events of two cells
// can be told apart. Regions sort to away 0, home 1.
func twoRegions(c *Config) {
	r := DefaultRegion()
	r.ID, r.ReferenceBasket = "away", 2_000
	c.Regions = append(c.Regions, r)
	c.Yields = append(c.Yields, YieldProfile{Region: "away", Recipe: "gather", HasAll: true, All: 10_000})
	c.Groups = append(c.Groups, Group{Region: "away", Role: "forager", Households: 1})
}

// TestOwnUseChanged_OnlyWhenNonZero_AscendingOrder_NoMapNoPointer verifies that the
// event is emitted once per region and role in ascending order in a tick in which
// a count is non-zero, never in a tick in which every count is zero, and that no
// field of it is a map or a pointer.
//
// Requirement: FR-SUB-040, FR-SUB-041
func TestOwnUseChanged_OnlyWhenNonZero_AscendingOrder_NoMapNoPointer(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newForage(t, 1, twoRegions, WithObserver(obs))
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	got := of[OwnUseChanged](obs)
	if len(got) != 2 || got[0].Region != 0 || got[1].Region != 1 || got[0].Tick != 0 {
		t.Fatalf("events of the first tick = %+v, want one for region 0 then one for region 1", got)
	}
	rich := &evLog{}
	r := newForage(t, 2, func(c *Config) {
		c.Roles[0].StartMoney = 1_000_000
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 40}}
	}, WithObserver(rich))
	for range 3 {
		if err := r.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(of[OwnUseChanged](rich)); n != 0 {
		t.Errorf("%d events in a world where nobody enters, want none", n)
	}
	typ := reflect.TypeOf(OwnUseChanged{})
	for i := range typ.NumField() {
		switch k := typ.Field(i).Type.Kind(); k {
		case reflect.Map, reflect.Pointer, reflect.Func, reflect.Interface:
			t.Errorf("field %s is a %v, want no map, pointer, function or interface", typ.Field(i).Name, k)
		}
	}
}

// TestOwnUseChanged_OutputInCommodityOrder verifies that Output lists the units
// created at the completion of own-use jobs by commodity in ascending order, and
// that the units are the ones created.
//
// Arithmetic: gather makes 2 grain and 3 ore an execution, one execution, so
// Output is [grain 2, ore 3].
//
// Requirement: FR-SUB-041
func TestOwnUseChanged_OutputInCommodityOrder(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newForage(t, 1, func(c *Config) {
		c.Commodities = append(c.Commodities, Commodity{ID: "ore", ReferencePrice: 1_000, Raw: true})
		c.Recipes[0].Outputs = []Output{{Commodity: "ore", Quantity: 3, CostShare: 5_000}, {Commodity: "grain", Quantity: 2, CostShare: 5_000}}
	}, WithObserver(obs))
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	got := of[OwnUseChanged](obs)
	want := []Amount{{Commodity: 0, Quantity: 2}, {Commodity: 1, Quantity: 3}}
	if len(got) != 1 || !slices.Equal(got[0].Output, want) {
		t.Errorf("Output = %+v, want %v", got, want)
	}
}

// TestOwnUseChanged_LevelFromLastEvent verifies that a change of OnOwnUse always
// carries Entered or Left, so a reader that keeps the last event knows the level
// at every tick.
//
// Requirement: FR-SUB-040
func TestOwnUseChanged_LevelFromLastEvent(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newForage(t, 3, nil, WithObserver(obs))
	level := int64(0)
	for range 12 {
		if err := w.Step(); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range of[OwnUseChanged](obs) {
		if e.OnOwnUse != level && e.Entered == 0 && e.Left == 0 {
			t.Errorf("tick %d: OnOwnUse went from %d to %d with no entered or left", e.Tick, level, e.OnOwnUse)
		}
		level = e.OnOwnUse
	}
	if level == 0 {
		t.Errorf("OnOwnUse never rose above 0, want the households on the floor")
	}
}

// warnTicks steps the world for n ticks and returns, for each tick, the Warn lines
// of that message.
func warnTicks(t *testing.T, w *World, c capture, msg string, n int) [][]logRec {
	t.Helper()
	var out [][]logRec
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
		var warns []logRec
		for _, l := range c.take(msg) {
			if l.level == slog.LevelWarn {
				warns = append(warns, l)
			}
		}
		out = append(out, warns)
	}
	return out
}

// TestWarn_RoleOnOwnUse_Ticks10And20 verifies the Warn "role on own use": more
// than half the households in the mode for ten consecutive ticks, logged at the
// tenth tick and every tenth after, once per cell, with the households and the
// count in the mode.
//
// Arithmetic: gather makes 2 grain against an upkeep of 2, so a household is fed
// but never builds the exit cover of 3 ticks and stays on the floor; the run is 1
// at tick 0, so the tenth tick of the run is tick 9 and the twentieth tick 19.
//
// Requirement: FR-SUB-042, FR-SUB-043
func TestWarn_RoleOnOwnUse_Ticks10And20(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w := newForage(t, 2, nil, WithLogger(slog.New(c)))
	lines := warnTicks(t, w, c, "role on own use", 25)
	for tick, l := range lines {
		want := 0
		if tick == 9 || tick == 19 {
			want = 1
		}
		if len(l) != want {
			t.Errorf("tick %d: %d lines of the Warn, want %d", tick, len(l), want)
		}
	}
	if l := lines[9]; len(l) == 1 && (l[0].level != slog.LevelWarn || l[0].attrs[logkeys.OwnUse] != 2 || l[0].attrs[logkeys.Households] != 2) {
		t.Errorf("line = %+v, want a Warn with households 2 and own_use 2", l[0])
	}
}

// TestWarn_OwnUseBlocked_Ticks10And20 verifies the Warn "own use blocked": a
// household in the mode blocked for ten consecutive ticks, logged at the tenth and
// every tenth after with the commonest reason.
//
// Arithmetic: the stock holds nothing, so the ration grants every request 0 and
// the household is blocked with no_endowment every tick.
//
// Requirement: FR-SUB-042
func TestWarn_OwnUseBlocked_Ticks10And20(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w := newForage(t, 1, forageField(0), WithLogger(slog.New(c)))
	lines := warnTicks(t, w, c, "own use blocked", 25)
	for tick, l := range lines {
		want := 0
		if tick == 9 || tick == 19 {
			want = 1
		}
		if len(l) != want {
			t.Errorf("tick %d: %d lines of the Warn, want %d", tick, len(l), want)
		}
	}
	if l := lines[9]; len(l) == 1 && l[0].strs[logkeys.Reason] != string(ReasonNoEndowment) {
		t.Errorf("reason = %q, want %q", l[0].strs[logkeys.Reason], ReasonNoEndowment)
	}
}

// TestNonInterference_ObserverAndLogger verifies that a run with an observer and a
// logger gives the snapshot a run without them gives.
//
// Requirement: FR-SUB-045
func TestNonInterference_ObserverAndLogger(t *testing.T) {
	t.Parallel()
	c := newCapture()
	plain := newForage(t, 3, forageField(4))
	seen := newForage(t, 3, forageField(4), WithObserver(&evLog{}), WithLogger(slog.New(c)))
	for range 15 {
		if err := plain.Step(); err != nil {
			t.Fatal(err)
		}
		if err := seen.Step(); err != nil {
			t.Fatal(err)
		}
	}
	a, err := plain.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	b, err := seen.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || !slices.Equal(plain.sub.ownUse, seen.sub.ownUse) {
		t.Errorf("a run with an observer and a logger differs from one without")
	}
}

// TestSnapshot_OnOwnUse verifies that the role row reports the households with the
// flag at the boundary, by weight, and that the value is 0 with the module off.
//
// Requirement: FR-SUB-046
func TestSnapshot_OnOwnUse(t *testing.T) {
	t.Parallel()
	w := newForage(t, 3, nil)
	if err := w.Step(); err != nil {
		t.Fatal(err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Roles[0].OnOwnUse; got != 3 {
		t.Errorf("OnOwnUse = %d, want 3: all three households entered", got)
	}
	off := newForage(t, 3, func(c *Config) { c.Modules.Subsistence = nil })
	if err := off.Step(); err != nil {
		t.Fatal(err)
	}
	so, err := off.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if so.Roles[0].OnOwnUse != 0 {
		t.Errorf("OnOwnUse with the module off = %d, want 0", so.Roles[0].OnOwnUse)
	}
}

// TestDebug_Lines verifies the per-agent Debug lines of the module: entry with its
// reason, exit with the cover reason, blocked with the production reason, and the
// retry.
//
// Requirement: FR-SUB-043
func TestDebug_Lines(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w := newForage(t, 1, forageField(0), WithLogger(slog.New(c)))
	if err := w.Step(); err != nil {
		t.Fatal(err)
	}
	if l := c.take("own use entered"); len(l) != 1 || l[0].strs[logkeys.Reason] != "short_cover" || l[0].level != slog.LevelDebug {
		t.Errorf("entered lines = %+v, want one Debug with reason short_cover", l)
	}
	if err := w.Step(); err != nil {
		t.Fatal(err)
	}
	if l := c.take("own use blocked"); len(l) == 0 || l[0].strs[logkeys.Reason] != string(ReasonNoEndowment) {
		t.Errorf("blocked lines = %+v, want a Debug with reason no_endowment", l)
	}
	a := &w.agents[1]
	holdGrain(t, w, a, 6)
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	if l := c.take("own use left"); len(l) != 1 || l[0].strs[logkeys.Reason] != "cover" {
		t.Errorf("left lines = %+v, want one with reason cover", l)
	}
	hungry := newForage(t, 1, nil, WithLogger(slog.New(c)))
	hungry.agents[1].hunger = 1
	hungry.sub.grow(len(hungry.agents))
	hungry.p24OwnUse()
	if l := c.take("own use entered"); len(l) != 1 || l[0].strs[logkeys.Reason] != "unfed" {
		t.Errorf("entered lines for a hungry household = %+v, want reason unfed", l)
	}
	v := ownView()
	v.Ratio, v.RetryJobs, v.OwnUseJobs = 1_000, 1, 1
	hungry.countRetry(t, v)
	if l := c.take("own use retry"); len(l) != 1 {
		t.Errorf("retry lines = %d, want 1", len(l))
	}
}

// countRetry feeds the report of v to the module's counts for the first
// household, as countWork does at P2.5.
func (w *World) countRetry(t *testing.T, v *AgentView) {
	t.Helper()
	r := reportOf(v)
	if !r.ownRetried {
		t.Fatalf("the fixture view does not retry")
	}
	w.sub.countWork(w, 0, &w.agents[1], &r)
}

// TestAllocs_EventNoop verifies that, with the no-op observer and logger, the
// report of P2.8 with non-zero counts allocates nothing: the event is neither
// built nor boxed.
//
// Requirement: NFR-SUB-001
func TestAllocs_EventNoop(t *testing.T) {
	w := newForage(t, 2, nil)
	if n := testing.AllocsPerRun(50, func() {
		w.sub.entered[0], w.sub.blocked[0], w.sub.output[0] = 1, 1, 1
		w.p28OwnUse()
	}); n != 0 {
		t.Errorf("allocations per P2.8 with the no-op observer = %v, want 0", n)
	}
}

// BenchmarkFlagStep reports the cost and the allocations of the flag step over 100
// households.
//
// Requirement: NFR-SUB-003
func BenchmarkFlagStep(b *testing.B) {
	c := forageConfig(100)
	w, err := NewWorld(c)
	if err != nil {
		b.Fatal(err)
	}
	w.sub.grow(len(w.agents))
	b.ReportAllocs()
	for b.Loop() {
		w.p24OwnUse()
	}
}
