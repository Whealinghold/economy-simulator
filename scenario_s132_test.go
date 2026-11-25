package economysim_test

import (
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalNeutralEvent returns e as the neutrality proof compares it: the configuration hash of the
// world-created event is zeroed, and the MoneyFlowScaled events the scale-0 call emits are dropped by the
// caller.
func fiscalNeutralEvent(e economysim.Event) economysim.Event {
	if wc, ok := e.(economysim.WorldCreated); ok {
		wc.ConfigHash = [32]byte{}
		return wc
	}
	return e
}

// fiscalNeutralSnapshot returns s without the configuration hash and the module list, the two fields the
// module's presence changes (spec-fiscal.md 19, FIS-S1).
func fiscalNeutralSnapshot(s economysim.Snapshot) economysim.Snapshot {
	s.ConfigHash = [32]byte{}
	s.Modules = nil
	return s
}

// withoutScaleEvents is the events of a world whose flows were paused by SetFlowScales, less those
// MoneyFlowScaled events (the call's own trace).
func withoutScaleEvents(evs []economysim.Event) []economysim.Event {
	out := make([]economysim.Event, 0, len(evs))
	for _, e := range evs {
		if _, ok := e.(economysim.MoneyFlowScaled); !ok {
			out = append(out, e)
		}
	}
	return out
}

// pauseAll sets the scale of every flow of the module to 0.
func pauseAll(t testing.TB, w *economysim.World, ids []string) {
	t.Helper()
	changes := make([]economysim.FlowScale, len(ids))
	for i, id := range ids {
		changes[i] = economysim.FlowScale{Flow: id, BP: 0}
	}
	if err := w.SetFlowScales(changes); err != nil {
		t.Fatalf("SetFlowScales(every fiscal flow to 0) error = %v, want nil", err)
	}
}

// TestScenarioS132_OffAndZeroScaleNeutral verifies FIS-S1 of spec-fiscal.md 19 (the neutrality proof,
// FR-FIS-001) on the pack of the golden, the governed warm five with overlay/fiscal-governed.toml, seeds 1
// to 10 and 200 ticks with the audit on: the world with the module on and every flow at scale 0 against
// the same world with the module removed. Every event is identical except the configuration hash of the
// world-created event and the MoneyFlowScaled events of the call that paused the flows; the final snapshot
// is identical except the hash and the module list; no flow event and no standing order is ever emitted.
// It then checks that the proof is not vacuous: with the flows at the full scale a treasury moves.
//
// Requirement: FR-FIS-001, FR-FIS-002, FR-FIS-003, FR-FIS-062
//
// Guard: a flow that took a draw, a carry or an ID while paused, or an order that was placed at scale 0,
// would show as a difference in a later event or in the snapshot. N-124: the comparison is exact only with
// every flow at scale 0, as a standing order adds orders and so keys.
func TestScenarioS132_OffAndZeroScaleNeutral(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	on := loadValidPack(t, fiveFiscal...)
	off := on
	off.Modules.Fiscal = nil
	ids := fiscalFlowIDs(on)
	if len(ids) == 0 || on.Modules.Fiscal == nil {
		t.Fatalf("fiscal-governed.toml has %d flows and Fiscal = %v, want flows and the module on", len(ids), on.Modules.Fiscal)
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			ron, roff := &recorder{}, &recorder{}
			won := mustWorld(t, on, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(ron))
			woff := mustWorld(t, off, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(roff))
			pauseAll(t, won, ids)
			if got := len(ron.events); got < len(ids) {
				t.Fatalf("seed %d: %d events after the call, want at least the %d MoneyFlowScaled", seed, got, len(ids))
			}
			// The two worlds are compared tick by tick, so the events of 200 ticks are never all in memory.
			for tick := 0; tick < 200; tick++ {
				fiscalStepTo(t, won, int64(tick+1))
				fiscalStepTo(t, woff, int64(tick+1))
				a, b := roff.events, withoutScaleEvents(ron.events)
				if tick == 0 && len(ron.events)-len(b) != len(ids) {
					t.Errorf("seed %d: %d MoneyFlowScaled events from the call, want %d (one per flow)", seed, len(ron.events)-len(b), len(ids))
				}
				if len(a) != len(b) {
					t.Fatalf("seed %d, tick %d: %d events with the module and %d without, want equal", seed, tick, len(b), len(a))
				}
				for i := range a {
					switch b[i].(type) {
					case economysim.FiscalFlowApplied, economysim.TreasuryOrdered:
						t.Fatalf("seed %d, tick %d: event %d is %T at scale 0, want none (FR-FIS-001)", seed, tick, i, b[i])
					}
					if x, y := fiscalNeutralEvent(a[i]), fiscalNeutralEvent(b[i]); !reflect.DeepEqual(x, y) {
						t.Fatalf("seed %d, tick %d: event %d differs: %#v with the module, %#v without", seed, tick, i, y, x)
					}
				}
				ron.forget()
				roff.forget()
			}
			son, err := won.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			soff, err := woff.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			if !reflect.DeepEqual(fiscalNeutralSnapshot(son), fiscalNeutralSnapshot(soff)) {
				t.Errorf("seed %d: the final snapshots differ beyond the hash and the module list", seed)
			}
		})
	}
}

// TestScenarioS132_FullScaleMovesMoney verifies that the proof of S132 is not vacuous: the same world with
// every flow at its full scale collects a tax, pays a payout and moves a treasury within 100 ticks, the
// audit passes, and a world with the module removed does not.
//
// Requirement: FR-FIS-001, FR-FIS-002, FR-FIS-003
//
// Guard: a pack whose flows never act would make the neutrality proof pass for the wrong reason.
func TestScenarioS132_FullScaleMovesMoney(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	on := loadValidPack(t, fiveFiscal...)
	off := on
	off.Modules.Fiscal = nil
	const seed, ticks = 1, 100
	won, k := fiscalWorld(t, on, seed)
	woff, _ := fiscalWorld(t, off, seed)
	fiscalStepTo(t, won, ticks)
	fiscalStepTo(t, woff, ticks)
	if k.sumKind(economysim.KindTax) <= 0 || k.sumKind(economysim.KindPayout) <= 0 || k.ordered == 0 {
		t.Errorf("after %d ticks (seed %d): taxes %d, payouts %d, TreasuryOrdered %d, want all positive", ticks, seed, k.sumKind(economysim.KindTax), k.sumKind(economysim.KindPayout), k.ordered)
	}
	if got, base := treasurySum(t, won), treasurySum(t, woff); got == base {
		t.Errorf("treasuries after %d ticks (seed %d) = %d with the module, %d without, want different", ticks, seed, got, base)
	}
}
