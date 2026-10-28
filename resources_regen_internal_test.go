package economysim

import (
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// This file tests the module at P8.3 and between ticks through the unexported seams
// (the helpers the integration calls will wrap, plan 2-9 D387): regeneration, the
// event, the Warn, the shocks and the audit. The public-API tests are in
// resources_world_test.go.

// regenMill is oreMill with a constant source of two units a tick: 730 units a year
// over the 365 ticks of the default calendar year.
func regenMill(duration, households, stock int64) Config {
	c := oreMill(duration, households, stock)
	c.Modules.Resources.Resources[0].RatePerYear = 2 * 365
	return c
}

// TestRegen_RunsAfterDrawsInOrder verifies that regeneration is the last step of a
// tick, after the draws, and uses the season of the tick: the event of the tick has
// the stock the draws left plus the units regenerated, it comes after the tick's
// BoundaryFlow rows and before TickEnded, and a profile of one season regenerates in
// that season only.
//
// Arithmetic: stock 20, ten agents ask 3 each, six are served: 18 drawn, 2 left,
// and the constant source adds 730 * 10 000 * 10 000 * 10 000 / (10^12 * 365) = 2,
// so the stock is 4 after the tick. With a profile of spring at 20 000 the source
// gives 4 a tick in spring (days 59 to 150) and 0 in the other seasons: 0 after
// tick 58, 4 * 92 = 368 after tick 150.
//
// Requirement: FR-RES-007, FR-RES-012
func TestRegen_RunsAfterDrawsInOrder(t *testing.T) {
	t.Parallel()
	t.Run("after the draws", func(t *testing.T) {
		t.Parallel()
		obs := &evLog{}
		w := newMill(t, regenMill(5, 10, 20), WithObserver(obs))
		step(t, w, 1)
		got := of[EndowmentChanged](obs)
		want := []EndowmentChanged{{Tick: 0, Region: 0, Resource: 0, Stock: 4, Capacity: 1_000, AbundanceBP: 10_000,
			Requested: 30, Granted: 18, Drawn: 18, Short: 12, Regenerated: 2, AgentsDenied: 4}}
		if !slices.Equal(got, want) {
			t.Fatalf("EndowmentChanged = %+v, want %+v", got, want)
		}
		var at, flow, ended int
		for i, e := range obs.events {
			switch e.(type) {
			case EndowmentChanged:
				at = i
			case BoundaryFlow:
				flow = i
			case TickEnded:
				ended = i
			}
		}
		if !(flow < at && at < ended) {
			t.Errorf("positions of BoundaryFlow, EndowmentChanged, TickEnded = %d, %d, %d, want ascending", flow, at, ended)
		}
		if got := w.Endowment(0, 0); got != 4 {
			t.Errorf("Endowment(home, ore) = %d, want 4 (20 - 18 + 2)", got)
		}
	})
	t.Run("seasonal profile", func(t *testing.T) {
		t.Parallel()
		c := regenMill(5, 10, 0)
		c.Modules.Resources.Resources[0].Seasons = []SeasonRegen{{Season: "spring", BP: 20_000}}
		c.Recipes[0].Draws = nil
		w := newMill(t, c)
		step(t, w, 59)
		if got := w.Endowment(0, 0); got != 0 {
			t.Errorf("Endowment after tick 58 = %d, want 0: the profile is complete, so a season it does not list regenerates nothing", got)
		}
		step(t, w, 92)
		if got := w.Endowment(0, 0); got != 368 {
			t.Errorf("Endowment after tick 150 = %d, want 368 (4 a tick over the 92 days of spring)", got)
		}
	})
}

// TestEvent_EndowmentChangedFields verifies the counts of the event and that it is
// emitted only when a count is non-zero: the units asked, granted, drawn and short
// (Requested - Granted == Short, Drawn <= Granted), the requests granted 0 and the
// requests granted less than asked, and no event for an endowment nothing moved.
//
// Arithmetic: ten agents of weight 1 ask 3 each against a stock of 20: six are
// served, four denied. Twenty cohorts of weight 50 ask 150 each (3000 in all)
// against 1 000: each is granted 1000 * 50 / 3000 = 16 executions whole, and the
// 1 000 - 20 * 16 * 3 = 40 units left go 3 at a time to 13 of them, so all twenty
// are granted less than the 50 they asked and more than 0.
//
// Requirement: FR-RES-026, FR-RES-027
func TestEvent_EndowmentChangedFields(t *testing.T) {
	t.Parallel()
	cohorts := oreMill(1, 1_000, 1_000)
	cohorts.World.Scale, cohorts.World.MinAgents = 50, 1
	quiet := oreMill(5, 10, 20)
	quiet.Recipes[0].Draws = nil
	for _, tt := range []struct {
		name   string
		cfg    Config
		events int
		denied int64
		reduce int64
		short  int64
	}{
		{"scarce stock denies some", oreMill(5, 10, 20), 1, 4, 0, 12},
		{"cohorts are reduced", cohorts, 1, 0, 20, 1_000 - 0},
		{"covered stock is only drawn", oreMill(5, 10, 1_000), 1, 0, 0, 0},
		{"nothing moves, nothing is emitted", quiet, 0, 0, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obs := &evLog{}
			step(t, newMill(t, tt.cfg, WithObserver(obs)), 1)
			got := of[EndowmentChanged](obs)
			if len(got) != tt.events {
				t.Fatalf("EndowmentChanged events = %d, want %d: %+v", len(got), tt.events, got)
			}
			if tt.events == 0 {
				return
			}
			e := got[0]
			if e.Requested-e.Granted != e.Short || e.Drawn > e.Granted {
				t.Errorf("event %+v: want Requested - Granted == Short and Drawn <= Granted", e)
			}
			if e.AgentsDenied != tt.denied || e.AgentsReduced != tt.reduce {
				t.Errorf("AgentsDenied, AgentsReduced = %d, %d, want %d, %d", e.AgentsDenied, e.AgentsReduced, tt.denied, tt.reduce)
			}
			if tt.name == "scarce stock denies some" && e.Short != tt.short {
				t.Errorf("Short = %d, want %d", e.Short, tt.short)
			}
		})
	}
}

// TestEvent_NoMapNoPointer verifies that the two events of the module hold only plain
// data, no map, pointer, slice, function or interface (observability-and-logging.md
// 4.1), so the golden renderer and a marshalling consumer get a fixed form.
//
// Requirement: FR-RES-050
func TestEvent_NoMapNoPointer(t *testing.T) {
	t.Parallel()
	for _, e := range []Event{EndowmentChanged{}, EndowmentAdjusted{}} {
		typ := reflect.TypeOf(e)
		for i := range typ.NumField() {
			switch k := typ.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
				t.Errorf("%s.%s has kind %s, want plain data", typ.Name(), typ.Field(i).Name, k)
			}
		}
	}
}

// TestEvent_AdjustedKinds verifies the helpers that shocks and adjustments go
// through: a capacity scale clamps a stock above the new capacity and counts the
// excess as lost, a regeneration scale leaves the stock, an adjustment is clamped
// to [0, capacity] and reported as the change that took effect, a call that changes
// nothing emits nothing, and the event is stamped with the tick the next step runs.
//
// Arithmetic: capacity 1 000, stock 800. A capacity scale of 5 000 makes the
// capacity 500 and loses 300. A delta of -600 then clamps at 0 (effective -500). A
// delta of +2 000 clamps at the capacity 500 (effective +500).
//
// Requirement: FR-RES-029, FR-RES-030
func TestEvent_AdjustedKinds(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, oreMill(5, 10, 800), WithObserver(obs))
	step(t, w, 2)
	tick, before := w.Tick(), w.Endowment(0, 0)
	obs.events = nil
	w.setCapacityScales(0, 0, 5_000)
	w.setCapacityScales(0, 0, 5_000)
	w.setRegenScales(0, 0, 3_000)
	w.adjustEndowments(0, 0, -600)
	w.adjustEndowments(0, 0, 0)
	w.adjustEndowments(0, 0, 2_000)
	w.adjustEndowments(0, 0, 10)
	stock := w.Endowment(0, 0)
	got := of[EndowmentAdjusted](obs)
	want := []EndowmentAdjusted{
		{Tick: tick, Kind: AdjustKindShock, CapacityBP: 5_000, RegenBP: 10_000, StockBefore: before, StockAfter: 500, Lost: before - 500},
		{Tick: tick, Kind: AdjustKindShock, CapacityBP: 5_000, RegenBP: 3_000, StockBefore: 500, StockAfter: 500},
		{Tick: tick, Kind: AdjustKindAdjust, Delta: -500, CapacityBP: 5_000, RegenBP: 3_000, StockBefore: 500, StockAfter: 0},
		{Tick: tick, Kind: AdjustKindAdjust, Delta: 500, CapacityBP: 5_000, RegenBP: 3_000, StockBefore: 0, StockAfter: 500},
	}
	if !slices.Equal(got, want) {
		t.Errorf("EndowmentAdjusted = %+v, want %+v", got, want)
	}
	if stock != 500 {
		t.Errorf("Endowment = %d, want 500", stock)
	}
	r := w.res
	if r.lost[0] != want[0].Lost || r.adjusted[0] != -500+500 {
		t.Errorf("lost, adjusted = %d, %d, want %d, 0", r.lost[0], r.adjusted[0], want[0].Lost)
	}
}

// TestWarn_EndowmentExhausted verifies the Warn: at the first tick with units short
// and every tenth consecutive one after, once per endowment per tick, with the tick,
// region, resource, stock and short; that ExhaustedRun counts consecutive ticks and
// resets when nothing is short; and that the Debug line "draw rationed" is written
// each tick the ration cut.
//
// Arithmetic: one-tick jobs, a source of 8 units a tick, stock 20, ten agents ask 3
// each every tick (a view asks only while the stock covers one draw). Tick 0 serves
// six (18 drawn), 2 are left and 8 arrive: stock 10, short 30 - 18 = 12. The stock
// then stays between 8 and 10, always above a draw and below the 30 asked, so units
// are short in every one of the 25 ticks, and the Warn is written at runs 1, 10 and
// 20, that is at ticks 0, 9, 19.
//
// Requirement: FR-RES-028
func TestWarn_EndowmentExhausted(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	c := oreMill(1, 10, 20)
	c.Modules.Resources.Resources[0].RatePerYear = 8 * 365
	// Grain for the inputs and the upkeep of all 25 ticks: 2 + 1 a tick.
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 100}}
	w := newMill(t, c, WithLogger(slog.New(lg)))
	for range 25 {
		step(t, w, 1)
	}
	warns := slices.DeleteFunc(append([]logRec(nil), *lg.lines...), func(l logRec) bool { return l.msg != "endowment exhausted" })
	var ticks []int64
	for _, l := range warns {
		ticks = append(ticks, l.attrs["tick"])
		if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["resource"] != 0 {
			t.Errorf("line %+v, want Warn for region 0, resource 0", l)
		}
	}
	if want := []int64{0, 9, 19}; !slices.Equal(ticks, want) {
		t.Errorf("Warn ticks = %v, want %v", ticks, want)
	}
	if warns[0].attrs["stock"] != 10 || warns[0].attrs["short"] != 12 {
		t.Errorf("first line carries stock %d, short %d, want 10 and 12", warns[0].attrs["stock"], warns[0].attrs["short"])
	}
	if n := len(slices.DeleteFunc(append([]logRec(nil), *lg.lines...), func(l logRec) bool { return l.msg != "draw rationed" })); n != 25 {
		t.Errorf("draw rationed lines = %d, want 25, one for each tick the ration cut", n)
	}
	if run := w.res.exhaustedRun[0]; run != 25 {
		t.Errorf("ExhaustedRun = %d, want 25", run)
	}
	w.adjustEndowments(0, 0, 1_000)
	step(t, w, 1)
	if run := w.res.exhaustedRun[0]; run != 0 {
		t.Errorf("ExhaustedRun after a tick with nothing short = %d, want 0", run)
	}
}

// TestAudit_ReportsIdentityBreak verifies that P8.4 under WithAudit checks the
// identity of an endowment on the counts P8.3 left, and names the endowment, the
// tick and the difference: a stock corrupted after P8.3 fails the step with
// ErrAuditFailed, and a stock above the effective capacity does too.
//
// Requirement: FR-RES-025
func TestAudit_ReportsIdentityBreak(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		break_ func(*resourcesState)
		want   string
	}{
		{"a unit missing", func(r *resourcesState) { r.stock[0]-- }, "difference -1"},
		{"above the capacity", func(r *resourcesState) { r.stock[0], r.base[0] = 1_001, 1_001 }, "capacity 1000"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, regenMill(5, 10, 20), WithAudit())
			if err := w.beginStep(); err != nil {
				t.Fatal(err)
			}
			err := replayTo41(w)
			if err == nil {
				err = replayFrom51Until83(w)
			}
			tt.break_(w.res)
			if err == nil {
				err = w.p84Audit()
			}
			if !errors.Is(err, ErrAuditFailed) || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), `"ore" in "home"`) {
				t.Errorf("audit error = %v, want ErrAuditFailed naming the endowment and %q", err, tt.want)
			}
		})
	}
}

// replayFrom51Until83 calls the positions after P4.1 up to and including P8.3,
// leaving the audit to the caller.
func replayFrom51Until83(w *World) error {
	w.p51Window()
	if err := w.p53Markdown(); err != nil {
		return err
	}
	w.p54Outcomes()
	if err := w.p61Upkeep(); err != nil {
		return err
	}
	w.p63Welfare()
	w.p64Shortfall()
	if err := w.p81Age(); err != nil {
		return err
	}
	w.p82Flows()
	w.p83Regen()
	return nil
}

// TestScratch_ClearedAtP86 verifies that the counts of the tick, the units
// regenerated, lost and adjusted included, are cleared at P8.6 and that a call
// between ticks counts in the next tick: the loss of a clamp made after a tick is
// in the event of the following tick and in no later one.
//
// Requirement: FR-RES-025
func TestScratch_ClearedAtP86(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	c := regenMill(5, 10, 800)
	c.Recipes[0].Draws = nil
	w := newMill(t, c, WithObserver(obs), WithAudit())
	step(t, w, 1)
	r := w.res
	if r.regenerated[0] != 0 || r.lost[0] != 0 || r.drawn[0] != 0 || r.base[0] != r.stock[0] {
		t.Errorf("after the tick regenerated, lost, drawn, base = %d, %d, %d, %d, want 0, 0, 0 and base == stock %d", r.regenerated[0], r.lost[0], r.drawn[0], r.base[0], r.stock[0])
	}
	w.setCapacityScales(0, 0, 5_000)
	if r.lost[0] != 302 {
		t.Fatalf("lost after the clamp = %d, want 302 (802 - 500)", r.lost[0])
	}
	obs.events = nil
	step(t, w, 2)
	got := of[EndowmentChanged](obs)
	// The second tick moves nothing: the stock is full, so the source adds nothing.
	if len(got) != 1 || got[0].Lost != 302 || got[0].Tick != 1 {
		t.Errorf("EndowmentChanged = %+v, want one event, at tick 1 with Lost 302, and none for the tick that moved nothing", got)
	}
}

// TestNoopObserverAllocates0 verifies that with the default logger and observer the
// snapshot, the ration and the regeneration of a step allocate nothing beyond the
// scratch the world holds (observability-and-logging.md 4.5, 8.2).
//
// Requirement: NFR-RES-001
func TestNoopObserverAllocates0(t *testing.T) {
	// Not parallel: testing.AllocsPerRun panics in a parallel test.
	w := newMill(t, regenMill(5, 10, 20))
	tickInspecting(t, w, func() {
		for name, f := range map[string]func(){"p23Snapshot": w.p23Snapshot, "p83Regen": w.p83Regen} {
			if got := testing.AllocsPerRun(20, f); got != 0 {
				t.Errorf("%s allocated %v times, want 0", name, got)
			}
		}
		// The requests of the tick are still listed, and the ration reuses its scratch.
		if got := testing.AllocsPerRun(20, w.p26Ration); got != 0 {
			t.Errorf("p26Ration allocated %v times, want 0", got)
		}
	})
}

// BenchmarkP83 reports the cost of the regeneration of the endowments of a world
// (NFR-RES-004).
func BenchmarkP83(b *testing.B) {
	w, err := NewWorld(regenMill(5, 10, 20))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		w.p83Regen()
	}
}
