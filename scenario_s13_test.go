package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	// s13Ticks is the golden horizon. The scenario of the spec runs four years
	// for the price drift, which is reported only; on pack one the grower class
	// is extinct long before (about tick 150), the exact accounting is the same at
	// any horizon, and a longer run only costs gate F time (plan 2-6 Phase 5).
	s13Ticks int64 = 400
	// s13Seeds caps the seeds of S13: the checks are exact identities of the
	// ledger, so a few seeds prove as much as ten.
	s13Seeds = 3
	// s13Stipend is the amount per household-tick of overlay/faucet.toml.
	s13Stipend = 20
)

// TestScenarioS13_FaucetAlone verifies scenario S13 of spec-reference-chain.md 7.2
// on the two packs that define it, one + overlay/faucet.toml (a grant and no sink)
// and one + overlay/faucet-sunk.toml (the same with a rate levy), up to seeds 1 to
// 3 and 400 ticks, with the audit on. Exact: the flow's total equals the stipend
// times the grower households read from the snapshot before each step; the
// boundary balance of its reason is minus the total and the money supply is its
// initial value plus the grants less the levies, at the end and at every tick; the
// audit holds at every tick. Reported only (no band exists until STY): the staple
// price at the end against the start.
//
// Requirement: FR-MON-002, FR-MON-004, FR-MON-026, FR-MON-029, FR-MON-043, FR-REF-020
//
// Guard: money enters only by a named flow, and by exactly what the flow says.
func TestScenarioS13_FaucetAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario = "S13"
	packs := []struct {
		name   string
		layers []string
	}{
		{"one+faucet", append(slices.Clone(packOne), "overlay/faucet.toml")},
		{"one+faucet-sunk", packOneFaucetSunk},
	}
	for _, pk := range packs {
		cfg := loadValidPack(t, pk.layers...)
		rep := newPackScenReport("one", fmt.Sprintf("S13 %s seeds 1-%d, %d ticks", pk.name, min(s13Seeds, scenarioSeedCount()), s13Ticks))
		t.Run(pk.name, func(t *testing.T) {
			for _, seed := range seedList(min(s13Seeds, scenarioSeedCount())) {
				t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
					t.Parallel()
					s13Run(t, rep, scenario, pk.name, cfg, seed)
				})
			}
		})
		t.Cleanup(func() { rep.flush(t) })
	}
}

// s13Run runs one seed of S13 and records its checks.
func s13Run(t *testing.T, rep *scenReport, scenario, pack string, cfg economysim.Config, seed uint64) {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	snap := func() economysim.Snapshot {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot error = %v, want nil", seed, err)
		}
		return s
	}
	start := snap()
	grower, north := slices.Index(start.Names.Roles, "grower"), slices.Index(start.Names.Regions, "north")
	staple := slices.Index(start.Names.Commodities, "staple")
	var grants int64
	identity := ""
	for tick := range s13Ticks {
		pre := snap()
		for _, r := range pre.Roles {
			if int(r.Role) == grower && int(r.Region) == north {
				grants += s13Stipend * r.Households
			}
		}
		if err := w.Step(); err != nil { // the audit runs in every step
			t.Fatalf("seed %d: Step at tick %d error = %v, want nil", seed, tick, err)
		}
		if identity == "" {
			identity = s13Identity(snap(), start.Money)
		}
	}
	end := snap()
	byID := map[string]economysim.FlowRow{}
	for _, f := range end.Flows {
		byID[f.Flow] = f
	}
	checkExact(t, rep, scenario, pack, seed, "stipend total equals 20 x grower households per tick", int64(byID["stipend"].Total) == grants,
		"total %d, want %d", byID["stipend"].Total, grants)
	var boundary int64
	for _, b := range end.Boundary {
		if b.Reason == "stipend" {
			boundary = b.Balance
		}
	}
	checkExact(t, rep, scenario, pack, seed, "boundary balance of the stipend is minus its total", boundary == -grants, "balance %d, want %d", boundary, -grants)
	checkExact(t, rep, scenario, pack, seed, "money supply moves by grants less levies at every tick", identity == "", "%s", identity)
	price := func(s economysim.Snapshot) int64 {
		for _, m := range s.Markets {
			if int(m.State.Commodity) == staple && int(m.State.Region) == north {
				return int64(m.State.Price)
			}
		}
		return -1
	}
	rep.num(seed, "staple price at the start", price(start))
	rep.num(seed, "staple price at the end", price(end))
	rep.num(seed, "money supply at the end", int64(end.Money))
}

// s13Identity returns a message when the money of the snapshot is not the initial
// money plus the grants less the levies in the flow totals of the same snapshot,
// and "" when it is.
func s13Identity(s economysim.Snapshot, initial economysim.Money) string {
	var granted, levied economysim.Money
	for _, f := range s.Flows {
		switch f.Flow {
		case "stipend":
			granted = f.Total
		case "charge":
			levied = f.Total
		}
	}
	if want := initial + granted - levied; s.Money != want {
		return fmt.Sprintf("tick %d: money %d, want initial %d + grants %d - levies %d = %d", s.Tick, s.Money, initial, granted, levied, want)
	}
	return ""
}
