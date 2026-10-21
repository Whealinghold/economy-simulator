package economysim

import (
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// commodityIndex returns the canonical index of the commodity named id.
func commodityIndex(t *testing.T, w *World, id string) int {
	t.Helper()
	for i := range w.cfg.Commodities {
		if w.cfg.Commodities[i].ID == id {
			return i
		}
	}
	t.Fatalf("commodity %q is not in the configuration", id)
	return -1
}

// TestPhase8_AuditPassesEveryTickAndTotalsClearOnlyAtP86 verifies that Audit
// passes after every tick of a seed range under WithAudit, that the flow totals
// are cleared only at P8.6 (an Audit or a read between ticks changes none), and
// that a faucet move made between ticks falls in the next tick's BoundaryFlow.
//
// Arithmetic: the move is 50 money from the Initial boundary to agent 1 between
// tick 0 and tick 1, so the totals read 50 for (initial, money) before the step
// of tick 1, and the one BoundaryFlow row of that reason in tick 1 is a faucet
// of 50; after the step the totals are empty again.
//
// Requirement: FR-WLD-074, FR-LED-031, FR-LED-035, FR-LED-036
func TestPhase8_AuditPassesEveryTickAndTotalsClearOnlyAtP86(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 4; seed++ {
		rec := &evLog{}
		w := newMill(t, panConfig(), WithAudit(), WithSeed(seed), WithObserver(rec))
		for range 5 {
			step(t, w, 1)
			if err := w.Audit(); err != nil {
				t.Fatalf("seed %d tick %d: Audit() error = %v, want nil", seed, w.Tick(), err)
			}
			if got := w.ledger.AppendFlows(nil, false); len(got) != 0 {
				t.Fatalf("seed %d tick %d: totals after the step = %v, want cleared at P8.6", seed, w.Tick(), got)
			}
		}
		src, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
		if !ok {
			t.Fatalf("seed %d: no Initial money boundary", seed)
		}
		if err := w.ledger.Transfer(src, w.agents[1].account, 50, ledger.Initial); err != nil {
			t.Fatalf("seed %d: Transfer() error = %v, want nil", seed, err)
		}
		before := w.ledger.AppendFlows(nil, false)
		if err := w.Audit(); err != nil {
			t.Fatalf("seed %d: Audit() between ticks error = %v, want nil", seed, err)
		}
		if after := w.ledger.AppendFlows(nil, false); !slices.Equal(before, after) || len(before) != 1 || before[0].Total != 50 {
			t.Fatalf("seed %d: totals before %v and after an Audit %v, want the same single total of 50", seed, before, after)
		}
		rec.events = nil
		step(t, w, 1)
		name, _ := reasonName(ledger.Initial)
		want := BoundaryFlow{Tick: 5, Region: 0, Reason: name, Kind: KindFaucet, Commodity: NoCommodity, Total: 50}
		if got := of[BoundaryFlow](rec); !slices.Contains(got, want) {
			t.Errorf("seed %d: BoundaryFlow rows of tick 5 = %v, want to contain %+v", seed, got, want)
		}
	}
}

// TestAuditFailure_MapsAndBracketsTheFailedStep verifies the mapping of an audit
// failure and the bracketing of the step that fails (D190). The real
// Ledger.Audit cannot be made to fail from the root, because ClearTotals
// re-anchors the bases and every operation conserves: this test covers the pure
// mapping and the order in endStep, not that p84Audit calls Ledger.Audit, which
// is one line behind WithAudit; the failing branch of Ledger.Audit is covered by
// internal/ledger/audit_internal_test.go.
//
// Requirement: FR-WLD-005, FR-WLD-012, FR-WLD-052
func TestAuditFailure_MapsAndBracketsTheFailedStep(t *testing.T) {
	t.Parallel()
	syn := &ledger.AuditError{Identity: ledger.IdentityConservation, Account: ledger.None, Commodity: ledger.None, Claim: -1, Detail: "synthetic"}
	if got := auditFailure(3, nil); got != nil {
		t.Errorf("auditFailure(3, nil) = %v, want nil", got)
	}
	err := auditFailure(3, syn)
	var ae *ledger.AuditError
	if !errors.Is(err, ErrAuditFailed) || !errors.As(err, &ae) || ae != syn {
		t.Fatalf("auditFailure(3, syn) = %v, want to wrap ErrAuditFailed and the *ledger.AuditError", err)
	}
	if want := "audit at tick 3: "; err.Error()[:len(want)] != want {
		t.Errorf("auditFailure(3, syn) = %q, want it to name tick 3", err)
	}

	var w *World
	var failedAtDelivery error
	var ended []TickEnded
	obs := ObserverFunc(func(e Event) {
		if te, ok := e.(TickEnded); ok {
			ended = append(ended, te)
			failedAtDelivery = w.failed
		}
	})
	w = newMill(t, millConfig(1, 2), WithObserver(obs))
	step(t, w, 2)
	flowsBefore := w.ledger.AppendFlows(nil, false)
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	src, _ := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if err := w.ledger.Transfer(src, w.agents[1].account, 1, ledger.Initial); err != nil {
		t.Fatalf("Transfer() error = %v, want nil", err)
	}
	flowsMid := w.ledger.AppendFlows(nil, false)
	ended = nil
	got := w.endStep(auditFailure(w.Tick(), syn))
	if !errors.Is(got, ErrAuditFailed) {
		t.Fatalf("endStep() = %v, want ErrAuditFailed", got)
	}
	if len(ended) != 1 || !ended[0].Failed || ended[0].Tick != 2 {
		t.Errorf("TickEnded events = %+v, want one with Failed true at tick 2", ended)
	}
	if !errors.Is(failedAtDelivery, ErrAuditFailed) {
		t.Errorf("world failed field at delivery = %v, want the audit failure already set", failedAtDelivery)
	}
	if w.Tick() != 2 {
		t.Errorf("Tick() = %d, want 2: a failed step does not advance", w.Tick())
	}
	if after := w.ledger.AppendFlows(nil, false); !slices.Equal(after, flowsMid) || slices.Equal(flowsBefore, flowsMid) {
		t.Errorf("totals after the failed step = %v, want unchanged from %v (not cleared)", after, flowsMid)
	}
	if w.stepping {
		t.Errorf("stepping = true after endStep, want cleared")
	}
	next := w.Step()
	if !errors.Is(next, ErrWorldFailed) || !errors.Is(next, ErrAuditFailed) {
		t.Errorf("Step() after the failure = %v, want ErrWorldFailed wrapping the cause", next)
	}
}

// TestP82Flows_WearRowsFollowTheBoundaryRowsInCommodityOrder verifies the events
// of P8.2: the BoundaryFlow rows, then the WearApplied rows in ascending
// commodity order, then TickEnded; no WearApplied row for a tick with no wear;
// and the construction bucket's rows carry Tick -1.
//
// Arithmetic: between tick 0 and tick 1 the axe is worn by 2 units and the pan
// by 3; the mill, if it runs in tick 1, wears each by 1 more, so tick 1 reports
// at least axe 2 and pan 3 (the pan's index is above the axe's).
//
// Requirement: FR-OBS-092, FR-OBS-014
func TestP82Flows_WearRowsFollowTheBoundaryRowsInCommodityOrder(t *testing.T) {
	t.Parallel()
	c := panConfig()
	c.Commodities = append(c.Commodities, Commodity{ID: "axe", ReferencePrice: 5_000, Durability: 10, Raw: true})
	c.Recipes[0].Requires = append(c.Recipes[0].Requires, Tool{Commodity: "axe", WearPerUse: 1})
	c.Roles[0].StartStock = append(c.Roles[0].StartStock, Item{Commodity: "axe", Quantity: 1})
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	axe, pn := commodityIndex(t, w, "axe"), commodityIndex(t, w, "pan")
	if axe >= pn {
		t.Fatalf("axe index %d, pan index %d, want axe first", axe, pn)
	}
	step(t, w, 1)
	rec.events = nil
	if err := w.ledger.WearItems(w.agents[1].account, pn, 1, 3); err != nil {
		t.Fatalf("WearItems(pan) error = %v, want nil", err)
	}
	if err := w.ledger.WearItems(w.agents[1].account, axe, 1, 2); err != nil {
		t.Fatalf("WearItems(axe) error = %v, want nil", err)
	}
	step(t, w, 1)
	wear := of[WearApplied](rec)
	if len(wear) != 2 || wear[0].Commodity != CommodityID(axe) || wear[0].Units < 2 || wear[1].Commodity != CommodityID(pn) || wear[1].Units < 3 {
		t.Fatalf("WearApplied rows = %+v, want axe at least 2 then pan at least 3, in ascending commodity", wear)
	}
	lastFlow, firstWear, ended := -1, -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case BoundaryFlow:
			lastFlow = i
		case WearApplied:
			if firstWear < 0 {
				firstWear = i
			}
		case TickEnded:
			ended = i
		}
	}
	if lastFlow > firstWear || firstWear < 0 || ended != len(rec.events)-1 || ended < firstWear {
		t.Errorf("event order: last BoundaryFlow %d, first WearApplied %d, TickEnded %d of %d, want flows, wear, TickEnded last", lastFlow, firstWear, ended, len(rec.events))
	}
	// A tick with no wear has no WearApplied row (the mill has no tool).
	rec2 := &evLog{}
	w2 := newMill(t, millConfig(1, 2), WithObserver(rec2))
	step(t, w2, 3)
	for _, e := range of[WearApplied](rec2) {
		t.Errorf("WearApplied %+v in a world with no wear, want none", e)
	}
}

// TestTickEnded_DebugLineCarriesTheLiveCounts verifies the Debug line "tick
// ended" (D163): once per step, with the tick, the agents and the households
// read from the world's counters.
//
// Requirement: FR-WLD-004
func TestTickEnded_DebugLineCarriesTheLiveCounts(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := newMill(t, millConfig(1, 2), WithLogger(slog.New(lg)))
	lg.take("tick ended")
	step(t, w, 2)
	lines := lg.take("tick ended")
	if len(lines) != 2 {
		t.Fatalf("\"tick ended\" lines = %d, want 2 (one per step)", len(lines))
	}
	for i, l := range lines {
		if l.level != slog.LevelDebug || l.attrs[logkeys.Tick] != int64(i) || l.attrs[logkeys.Agents] != w.live.agents || l.attrs[logkeys.Households] != w.live.households {
			t.Errorf("line %d = level %v attrs %v, want Debug, tick %d, agents %d, households %d",
				i, l.level, l.attrs, i, w.live.agents, w.live.households)
		}
	}
}

// TestP81Age_SpoilageIsASinkInTheTickItHappens verifies P8.1 and P8.2 together:
// a lot that spoils moves to the spoilage boundary at the end of its last tick
// and is reported once as a sink BoundaryFlow of that tick, with the audit
// passing.
//
// Arithmetic: grain has a shelf life of 2 and the start stock is born at tick
// 0, so Born + ShelfLife - 1 = 1: no spoilage is reported in tick 0, and the
// lots still held spoil at the end of tick 1. The agents hold 10 grain each at
// the start and use at most 2 per tick for 2 ticks, so some grain is left to
// spoil.
//
// Requirement: FR-WLD-004, FR-LED-031
func TestP81Age_SpoilageIsASinkInTheTickItHappens(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 2)
	for i := range c.Commodities {
		if c.Commodities[i].ID == "grain" {
			c.Commodities[i].ShelfLife = 2
		}
	}
	rec := &evLog{}
	w := newMill(t, c, WithAudit(), WithObserver(rec))
	step(t, w, 2)
	name, _ := reasonName(ledger.Spoilage)
	grain := CommodityID(commodityIndex(t, w, "grain"))
	var got []BoundaryFlow
	for _, f := range of[BoundaryFlow](rec) {
		if f.Reason == name {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Tick != 1 || got[0].Kind != KindSink || got[0].Commodity != grain || got[0].Total <= 0 {
		t.Errorf("spoilage rows = %+v, want one sink row for grain at tick 1 with a positive total", got)
	}
}
