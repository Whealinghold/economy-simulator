package economysim

import (
	"cmp"
	"slices"
	"testing"
)

// The world-level tests of the ration and the start of the tenure module that the first
// tests of the phase left: the order of the requests, the rent in force, the atomic refusal,
// a payee at the balance bound and the own-use households the tenure refuses. The worlds
// and the replay of a tick are in tenure_ration_internal_test.go.

// grantsOf replays the tick of w to P2.5 and rations the endowment 0 itself, with the
// requests listed in ascending agent order or in the reverse of it, and returns the grant
// of every request by agent with the counts of the endowment.
func grantsOf(t *testing.T, w *World, reversed bool) (map[AgentID]int64, tenureCount) {
	t.Helper()
	toSnapshot(t, w)
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work() error = %v, want nil", err)
	}
	r := w.res
	for len(r.grants) < len(w.agents) {
		r.grants = append(r.grants, 0)
	}
	order := make([]int, len(r.reqs))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(r.reqs[a].agent, r.reqs[b].agent) })
	if reversed {
		slices.Reverse(order)
	}
	if !w.tenure.ration(order) {
		t.Fatalf("ration() = false, want the tiers of a titled endowment")
	}
	got := make(map[AgentID]int64, len(order))
	for _, i := range order {
		got[r.reqs[i].agent] = r.reqs[i].grant
	}
	return got, w.tenure.rt.counts[0]
}

// TestRation_OrderIndependentGrants verifies FR-LND-011 at the world level: the grants and
// the counts of a titled endowment are the same whatever order its requests are listed in,
// over several seeds and three shapes of tenure (a pool title, a closed pool title and a
// household title), because every tier ranks the requests by the key of the tick and the
// agent and not by their place in the list.
//
// Guard: a tier that walked the list in the order it was given would pass every test
// that lists the requests in agent order, and change the grants with the order of the run.
//
// Requirement: FR-LND-011
func TestRation_OrderIndependentGrants(t *testing.T) {
	t.Parallel()
	for _, sh := range []struct {
		name   string
		pool   int64
		closed bool
		own    int64
	}{{"pool", 3_000, false, 0}, {"closed pool", 3_000, true, 0}, {"households", 0, false, 400}} {
		for seed := uint64(1); seed <= 4; seed++ {
			t.Run(sh.name, func(t *testing.T) {
				c := rationConfig(40, sh.pool, sh.closed)
				if sh.own > 0 {
					c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: sh.own}}
				}
				asc, ascCounts := grantsOf(t, newMill(t, c, WithSeed(seed)), false)
				desc, descCounts := grantsOf(t, newMill(t, c, WithSeed(seed)), true)
				if !slices.Equal(mapValues(asc), mapValues(desc)) {
					t.Errorf("seed %d: grants listed in reverse = %v, want %v (the order of ascending agent)", seed, desc, asc)
				}
				if ascCounts != descCounts {
					t.Errorf("seed %d: counts listed in reverse = %+v, want %+v", seed, descCounts, ascCounts)
				}
			})
		}
	}
}

// mapValues is the values of grants in ascending agent order.
func mapValues(m map[AgentID]int64) []int64 {
	ids := make([]AgentID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = m[id]
	}
	return out
}

// TestRentInForce_ScaleSetsTheRentCharged verifies FR-LND-015 through the world: a posted
// rent of 10 and a scale set by SetRentScales give the rent in force that the one tenant
// of the let tier pays, 3 units at that rent, and a scale of 0 makes the let free.
//
// Arithmetic: rent in force = posted * scale / 10 000 truncated, at least 1 when both are
// positive. Posted 10 at scale 0 is 0; at 1 it is 0 truncated, so 1; at 5 000 it is 5; at
// 10 000 it is 10. The tenant draws 3 units, so it pays 0, 3, 15 and 30.
//
// Requirement: FR-LND-015
func TestRentInForce_ScaleSetsTheRentCharged(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		scale, rent, paid int64
	}{{0, 0, 0}, {1, 1, 3}, {5_000, 5, 15}, {10_000, 10, 30}} {
		t.Run("", func(t *testing.T) {
			c := rationConfig(10, 5_000, false)
			c.Modules.Tenure.Terms[0].Rent = 10
			w := newMill(t, c, WithSeed(1), WithAudit())
			if err := w.SetRentScales([]EndowmentScale{{Region: 0, Resource: 0, BP: tt.scale}}); err != nil {
				t.Fatalf("SetRentScales(%d) error = %v, want nil", tt.scale, err)
			}
			toStart(t, w)
			got := w.tenure.rt.counts[0]
			if got.rentInForce != tt.rent || got.paid != tt.paid || got.letUnits != 3 {
				t.Errorf("scale %d: rent in force %d, paid %d, let %d, want %d, %d and 3", tt.scale, got.rentInForce, got.paid, got.letUnits, tt.rent, tt.paid)
			}
		})
	}
}

// TestStart_AtomicOnRefusal verifies FR-LND-016 on the two refusals of the tenure: a start
// refused no_title or rent_funds takes nothing from the stock, reserves no rent against
// its principal, moves no money and leaves no job, so nothing is taken and nothing is
// carried to the next start or the next tick.
//
// Requirement: FR-LND-014, FR-LND-016
func TestStart_AtomicOnRefusal(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		cfg   func() Config
		stock int64 // the stock after the starts
		jobs  int64
	}{
		{"no title", func() Config { return rationConfig(100, 10_000, true) }, 100, 0},
		{"rent funds", func() Config {
			c := rationConfig(100, 10_000, false)
			c.Roles[0].StartMoney = 0
			return c
		}, 100, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newMill(t, tt.cfg(), WithSeed(1), WithAudit())
			money := tnTotalMoney(t, w)
			toStartForced(t, w)
			if got := w.res.stock[0]; got != tt.stock {
				t.Errorf("stock after the refused starts = %d, want %d", got, tt.stock)
			}
			if got := startedBy(w); got != tt.jobs {
				t.Errorf("households with a job = %d, want %d", got, tt.jobs)
			}
			for id := AgentID(1); id <= 10; id++ {
				if got := w.tenure.reservedOf(id); got != 0 {
					t.Errorf("rent reserved against household %d = %d, want 0", id, got)
				}
			}
			if got := tnTotalMoney(t, w); got != money {
				t.Errorf("money after the refused starts = %d, want %d", got, money)
			}
			if c := w.tenure.rt.counts[0]; !drewNothing(c) || len(w.tenure.rt.starts) != 0 {
				t.Errorf("counts = %+v, starts = %d, want nothing drawn, paid or carried", c, len(w.tenure.rt.starts))
			}
		})
	}
}

// drewNothing reports whether the counts of an endowment show no unit drawn and no rent paid.
func drewNothing(c tenureCount) bool {
	return c.paid == 0 && c.letUnits == 0 && c.ownUnits == 0 && c.commonsUnits == 0
}

// TestStart_PayeeAtMaxMoneyForfeitsNoRefusal verifies FR-LND-016 end to end for the
// payee: the lessor of the let tier holds all but 1 of MaxMoney, so it receives 1 of the
// rent, the rest is forfeited and counted, the tenants are charged only the 1 and no start
// is refused for it.
//
// Arithmetic: the stock is 20 and the lessor, a household added to the world with the
// directed strategy that asks nothing, holds 5 000 basis points: entitled 10, all unused,
// and the commons are 10. The ten households ask 30 units of the 10 commons units: 3
// executions (9 units). The seven requests left ask 21 units of the 10 let: 3 executions,
// 9 units, at the rent of 2 a unit 18. The lessor receives 1 and 17 are forfeited. Six
// households start.
//
// Requirement: FR-LND-016
func TestStart_PayeeAtMaxMoneyForfeitsNoRefusal(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(20, 5_000, false), WithSeed(1), WithAudit())
	lessor, err := w.AddAgent(NewAgent{Region: 0, Money: Money(MaxMoney - 1)})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, 0, lessor, 5_000, 0)}); err != nil {
		t.Fatalf("the gift of the pool's title error = %v, want nil", err)
	}
	money := tnTotalMoney(t, w)
	toStart(t, w)
	c := w.tenure.rt.counts[0]
	if c.letUnits != 9 || c.paid != 1 || c.forfeited != 17 {
		t.Errorf("counts = %+v, want 9 let units, 1 paid and 17 forfeited", c)
	}
	if got := tnMoney(t, w, lessor); got != int64(MaxMoney) {
		t.Errorf("lessor holds %d, want MaxMoney", got)
	}
	if got := tnTotalMoney(t, w); got != money {
		t.Errorf("money after the rent = %d, want %d (rent moves money and creates none)", got, money)
	}
	if c.rentFunds != 0 {
		t.Errorf("requests refused rent_funds = %d, want 0 (a payee at the bound forfeits and refuses nobody)", c.rentFunds)
	}
	if got := startedBy(w); got != 6 {
		t.Errorf("households with a job = %d, want 6 (commons 3, let 3)", got)
	}
}

// forageTenure is the forage world with the field endowment titled: the pool holds
// poolBP basis points of it, the rent is 5, and the endowment is closed when closed is
// set and its own-use requests pay no rent when exempt is set.
func forageTenure(poolBP int64, closed, exempt bool) func(*Config) {
	return func(c *Config) {
		forageField(100)(c)
		c.Modules.Estates = &Estates{}
		c.Modules.Tenure = &TenureConfig{
			OwnUseExempt: exempt,
			Terms:        []TenureTerm{{Region: "home", Resource: "field", HasRent: true, Rent: 5, HasClosed: true, Closed: closed}},
			Holdings:     []Holding{{Region: "home", Resource: "field", Holder: HolderRef{Kind: HolderKindPool}, ShareBP: poolBP}},
		}
	}
}

// TestSubsistence_BlockedCountsNoTitleRentFunds verifies FR-LND-046: four own-use
// households whose draws on the titled field the tenure refuses, no_title when the pool
// holds a closed endowment and rent_funds when the pool lets it all at a rent the
// households (who hold no money) cannot pay, are counted as Blocked by the subsistence
// module in the tick they enter, and with OwnUseExempt they pay no rent, start, and are
// not blocked.
//
// Requirement: FR-LND-046, FR-LND-018
func TestSubsistence_BlockedCountsNoTitleRentFunds(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                    string
		poolBP                  int64
		closed, exempt          bool
		wantBlocked, wantStarts int64
	}{
		{"closed", 10_000, true, false, 4, 0},
		{"priced out", 10_000, false, false, 4, 0},
		{"exempt from rent", 10_000, false, true, 0, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			obs := &evLog{}
			w := newForage(t, 4, forageTenure(tt.poolBP, tt.closed, tt.exempt), WithObserver(obs), WithAudit())
			step(t, w, 1)
			got := of[OwnUseChanged](obs)
			if len(got) != 1 || got[0].Entered != 4 || got[0].Blocked != tt.wantBlocked || got[0].Started != tt.wantStarts {
				t.Errorf("OwnUseChanged = %+v, want one event with entered 4, blocked %d and started %d", got, tt.wantBlocked, tt.wantStarts)
			}
		})
	}
}
