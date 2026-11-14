package economysim

import (
	"math/rand/v2"
	"reflect"
	"sync"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// treasuryEyes is a Strategy that wraps the default one and keeps the ID of every agent
// it is called for, a fake at the Strategy extension point (testing.md 6.1).
type treasuryEyes struct {
	DefaultStrategy
	mu   sync.Mutex
	seen map[AgentID]bool
}

func (s *treasuryEyes) note(a AgentID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[a] = true
}

func (s *treasuryEyes) Work(v *AgentView, r *rand.Rand) JobRequest {
	s.note(v.Agent)
	return s.DefaultStrategy.Work(v, r)
}

func (s *treasuryEyes) Trade(v *AgentView, r *rand.Rand, p *TradePlan) {
	s.note(v.Agent)
	s.DefaultStrategy.Trade(v, r, p)
}

func (s *treasuryEyes) Revise(o *OutcomeView, r *rand.Rand) Belief {
	s.note(o.Agent)
	return s.DefaultStrategy.Revise(o, r)
}

// TestPolity_StartMoneyIsExactlyTheInitialIssue verifies FR-POL-017 against the world
// without the module: the money the boundary of initial has issued is higher by exactly
// the sum of the StartMoney of the treasuries, and the construction row of initial money
// in each seat region by that treasury's, so the transfer is neither doubled nor taken
// from another reason. A seat with a StartMoney of 0 makes no transfer: the world
// issues what the world without the module does and the treasury holds 0.
//
// Arithmetic: crown (seat home) 50 000 plus inland (seat away) 5 000 is 55 000 issued
// more than regionMill; with both StartMoney 0 the difference is 0.
//
// Requirement: FR-POL-017
func TestPolity_StartMoneyIsExactlyTheInitialIssue(t *testing.T) {
	t.Parallel()
	base := newMill(t, regionMill())
	initialByRegion := func(w *World) map[RegionID]int64 {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		out := map[RegionID]int64{}
		for _, f := range s.Construction {
			if f.Reason == ReasonInitial && f.Commodity == NoCommodity {
				out[f.Region] += f.Total
			}
		}
		return out
	}
	zero := treasuryConfig()
	zero.Modules.Polity.Jurisdictions[0].StartMoney, zero.Modules.Polity.Jurisdictions[1].StartMoney = 0, 0
	for _, tt := range []struct {
		name         string
		cfg          Config
		issued       int64
		home, away   int64 // the more initial money than the base in each region
		crown, inner int64 // the treasuries' balances
	}{
		{"funded", treasuryConfig(), 55_000, 50_000, 5_000, 50_000, 5_000},
		{"seats with no money", zero, 0, 0, 0, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, tt.cfg, WithAudit())
			if got, want := w.boundaryFlowMoney(ledger.Initial), base.boundaryFlowMoney(ledger.Initial)+tt.issued; got != want {
				t.Errorf("initial money issued = %d, want %d (the base's plus %d)", got, want, tt.issued)
			}
			got, want := initialByRegion(w), initialByRegion(base)
			if got[1]-want[1] != tt.home || got[0]-want[0] != tt.away {
				t.Errorf("initial money by region differs from the base by home %d, away %d, want %d and %d", got[1]-want[1], got[0]-want[0], tt.home, tt.away)
			}
			for i, bal := range []int64{tt.crown, tt.inner} {
				if v, err := w.ledger.Account(w.polity.tr[i].account); err != nil || v.Balance != bal {
					t.Errorf("treasury %d = %+v, %v, want balance %d", i, v, err, bal)
				}
			}
		})
	}
}

// TestPolity_TreasuryMovesNothingAfterConstruction verifies FR-POL-018 and FR-POL-029
// over 30 ticks of a world with the audit on: the two treasuries hold their StartMoney
// after every step and the snapshot's money of the world is the households', the pool's
// and the treasuries', so nothing enters or leaves a treasury and the identity holds
// with the treasury term.
//
// Requirement: FR-POL-018, FR-POL-029
func TestPolity_TreasuryMovesNothingAfterConstruction(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig(), WithAudit())
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
		for i, want := range []int64{50_000, 5_000} {
			if v, _ := w.ledger.Account(w.polity.tr[i].account); v.Balance != want {
				t.Fatalf("tick %d: treasury %d holds %d, want %d", w.Tick(), i, v.Balance, want)
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		var parts Money
		for _, r := range s.Regions {
			parts += r.MoneyHouseholds + r.MoneyPool + r.MoneyTreasury
		}
		if parts != s.Money {
			t.Fatalf("tick %d: money by region sums to %d, want the world's %d", s.Tick, parts, s.Money)
		}
	}
}

// TestPolity_TreasuryNeverMeetsAStrategyOrAMarket verifies FR-POL-016 over 30 ticks: no
// strategy is called for a treasury pseudo-agent, no event names one in an agent field
// (an order, a trade, a payment, a spawn), and every household's ID is at or above
// firstHousehold, so the treasury has no order of its own.
//
// Requirement: FR-POL-016
func TestPolity_TreasuryNeverMeetsAStrategyOrAMarket(t *testing.T) {
	t.Parallel()
	c := treasuryConfig()
	for i := range c.Roles {
		c.Roles[i].Strategy = "eyes"
	}
	eyes, rec := &treasuryEyes{seen: map[AgentID]bool{}}, &evLog{}
	w := newMill(t, c, WithStrategy("eyes", eyes), WithObserver(rec), WithAudit())
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if len(eyes.seen) == 0 {
		t.Fatal("the strategy was never called, want it called for the households")
	}
	for a := range eyes.seen {
		if int(a) < w.firstHousehold {
			t.Errorf("the strategy was called for agent %d, below firstHousehold %d", a, w.firstHousehold)
		}
	}
	treasury := map[AgentID]bool{2: true, 3: true}
	for _, e := range rec.events {
		v := reflect.ValueOf(e)
		for i := range v.NumField() {
			if f := v.Field(i); f.Type() == reflect.TypeOf(AgentID(0)) && treasury[AgentID(f.Int())] {
				t.Fatalf("%T names the treasury agent %d in %s", e, f.Int(), v.Type().Field(i).Name)
			}
		}
	}
}

// TestPolity_HouseholdsKeepTheirShiftedIDs verifies FR-POL-016 and the shift of
// spec-polity.md 5.2.3 through the queries and calls that take an agent ID: Agents lists
// the households at their shifted IDs only, Agent answers them and refuses a treasury as
// an unknown agent, and a household is relocated by its shifted ID with the audit on.
//
// Requirement: FR-POL-016, FR-POL-031
func TestPolity_HouseholdsKeepTheirShiftedIDs(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig(), WithAudit())
	var all []AgentID
	for g := range RegionID(2) {
		ids, err := w.Agents(g)
		if err != nil {
			t.Fatalf("Agents(%d) error = %v, want nil", g, err)
		}
		all = append(all, ids...)
	}
	if len(all) != 4 {
		t.Fatalf("Agents lists %v, want the four households", all)
	}
	for _, id := range all {
		if int(id) < w.firstHousehold {
			t.Errorf("Agents lists %d, a pseudo-agent below firstHousehold %d", id, w.firstHousehold)
		}
		if st, err := w.Agent(id); err != nil || st.ID != id {
			t.Errorf("Agent(%d) = %+v, %v, want that household", id, st, err)
		}
	}
	for _, id := range []AgentID{0, 1, 2, 3} {
		if _, err := w.Agent(id); err == nil {
			t.Errorf("Agent(%d) error = nil, want ErrUnknownAgent for a pseudo-agent", id)
		}
	}
	st, _ := w.Agent(all[0])
	to := 1 - st.Region
	if err := w.Relocate(all[0], to); err != nil {
		t.Fatalf("Relocate(%d, %d) error = %v, want nil", all[0], to, err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() after the relocation error = %v, want nil", err)
	}
	if st, err := w.Agent(all[0]); err != nil || st.Region != to {
		t.Errorf("Agent(%d) = %+v, %v, want region %d", all[0], st, err, to)
	}
}

// TestPolity_StartMoneyAtMaxMoney verifies the upper bound of FR-POL-014 in a built
// world: a treasury created with MaxMoney builds, its snapshot row holds MaxMoney and
// the snapshot's money of the world saturates and does not wrap.
//
// Requirement: FR-POL-014, FR-POL-017
func TestPolity_StartMoneyAtMaxMoney(t *testing.T) {
	t.Parallel()
	c := treasuryConfig()
	c.Modules.Polity.Jurisdictions[0].StartMoney, c.Modules.Polity.Jurisdictions[1].StartMoney = MaxMoney, 0
	w := newMill(t, c, WithAudit())
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if s.Jurisdictions[0].Money != MaxMoney || s.Money < MaxMoney {
		t.Errorf("row money %d and world money %d, want MaxMoney %d held and a world money not below it", s.Jurisdictions[0].Money, s.Money, MaxMoney)
	}
	if err := w.Step(); err != nil {
		t.Errorf("Step() with a treasury of MaxMoney error = %v, want nil", err)
	}
}
