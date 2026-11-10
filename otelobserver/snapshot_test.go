package otelobserver_test

import (
	"math"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// snapOf is a hand-built snapshot over miniNames: one market, one job, two
// regions' money and two roles' rows.
func snapOf() economysim.Snapshot {
	return economysim.Snapshot{
		Names: miniNames(),
		Markets: []economysim.MarketRow{{
			State:        economysim.MarketState{Region: 1, Commodity: 2},
			HeldByAgents: 7, HeldByEstate: 3, Lots: 5, Pending: 11, EstateAtFloor: 2,
		}},
		Jobs:    []economysim.JobRow{{Region: 2, Recipe: 0, Executions: 4}},
		Regions: []economysim.RegionRow{{Region: 0, MoneyHouseholds: 100, MoneyPool: 30, MoneyContract: 9}},
		Roles: []economysim.RoleRow{
			{Region: 0, Role: 1, Agents: 6, Households: 8, HeldByFloor: 2, ClampedHouseholds: 1, MaxPrivation: 4,
				MeanRatioBP: 12_000, HasMeanRatio: true, RatioHouseholds: 8, WageObserved: 55, HasWage: true,
				ExpectedRatioBP: 9_000, Viable: true, HasExpected: true},
			{Region: 1, Role: 0, Agents: 2, Households: 0, HeldByFloor: 0},
		},
	}
}

// TestSnapshot_RowsToInstruments verifies that each snapshot-fed instrument has
// one series per row, with the attributes of the row.
//
// Requirement: FR-OBS-097, FR-OBS-098, FR-OBS-101, FR-OBS-102
func TestSnapshot_RowsToInstruments(t *testing.T) {
	h := newHarness(t)
	h.obs.RecordSnapshot(snapOf())
	got := h.collect()
	cc := []string{"commodity", "c2", "region", "r1"}
	check(t, got, "production.pending", 11, cc...)
	check(t, got, "estate.at_floor", 2, cc...)
	check(t, got, "ledger.lots", 5, cc...)
	check(t, got, "market.stock", 7, append(cc, "holder", "agents")...)
	check(t, got, "market.stock", 3, append(cc, "holder", "estate")...)
	check(t, got, "recipe.in_progress", 4, "recipe", "p0", "region", "r2")
	check(t, got, "money.supply", 100, "region", "r0", "sector", "households")
	check(t, got, "money.supply", 30, "region", "r0", "sector", "pool")
	check(t, got, "money.supply", 9, "region", "r0", "sector", "contract")
	rr := []string{"role", "k1", "region", "r0"}
	for name, want := range map[string]int64{"role.agents": 6, "welfare.clamped": 1, "household.privation.max": 4,
		"welfare.ratio": 12_000, "welfare.ratio.households": 8, "wage.observed": 55,
		"role.expected_ratio": 9_000, "role.viable": 1, "welfare.floor_held": 2_500} {
		check(t, got, name, want, rr...)
	}
	check(t, got, "role.agents", 2, "role", "k0", "region", "r1")
}

// TestSnapshot_AbsentNotZeroBeforeFirstCall verifies that no snapshot series
// exists before the first call, and that a call records none of the event-fed
// series.
//
// Requirement: FR-OBS-104
func TestSnapshot_AbsentNotZeroBeforeFirstCall(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0)
	for _, n := range []string{"role.agents", "money.supply", "market.stock", "welfare.clamped"} {
		if got := h.collect(); len(got["economysim."+n]) != 0 {
			t.Errorf("%s has %d series before the first RecordSnapshot, want 0", n, len(got["economysim."+n]))
		}
	}
	h.obs.RecordSnapshot(snapOf())
	if got := h.collect(); len(got["economysim.estate.sold"]) != 0 || len(got["economysim.role.agents"]) == 0 {
		t.Errorf("after RecordSnapshot: estate.sold series = %d (want 0), role.agents series = %d (want > 0)",
			len(got["economysim.estate.sold"]), len(got["economysim.role.agents"]))
	}
}

// TestSnapshot_HasFlagsGate verifies that a role whose Has flag is false has no
// series of the instrument it gates, and that floor_held is absent at no
// households.
//
// Requirement: FR-OBS-098, FR-OBS-101, FR-OBS-102
func TestSnapshot_HasFlagsGate(t *testing.T) {
	h := newHarness(t)
	h.obs.RecordSnapshot(snapOf())
	got := h.collect()
	k0 := []string{"role", "k0", "region", "r1"}
	for _, n := range []string{"welfare.ratio", "welfare.ratio.households", "wage.observed", "role.expected_ratio",
		"role.viable", "welfare.floor_held"} {
		absent(t, got, n, k0...)
	}
	check(t, got, "role.agents", 2, k0...)
}

// TestSnapshot_FloorHeldIsExact verifies the share is computed in 128 bits: a
// count near MaxInt64/2 does not wrap, and a negative one is absent.
//
// Requirement: FR-OBS-098
func TestSnapshot_FloorHeldIsExact(t *testing.T) {
	h := newHarness(t)
	s := economysim.Snapshot{Names: miniNames(), Roles: []economysim.RoleRow{
		{Region: 0, Role: 0, Households: math.MaxInt64, HeldByFloor: math.MaxInt64 / 2},
		{Region: 1, Role: 0, Households: 10, HeldByFloor: -1},
		{Region: 2, Role: 0, Households: 1, HeldByFloor: 1},
		{Region: 2, Role: 1, Households: 5, HeldByFloor: 0, HasExpected: true},
		{Region: 1, Role: 1, Households: -1, HeldByFloor: 3},
		{Region: 0, Role: 1, Households: 0, HeldByFloor: 0},
	}}
	h.obs.RecordSnapshot(s)
	got := h.collect()
	check(t, got, "welfare.floor_held", 4_999, "role", "k0", "region", "r0")
	absent(t, got, "welfare.floor_held", "role", "k0", "region", "r1")
	check(t, got, "welfare.floor_held", 10_000, "role", "k0", "region", "r2")
	check(t, got, "welfare.floor_held", 0, "role", "k1", "region", "r2")
	check(t, got, "role.viable", 0, "role", "k1", "region", "r2")
	absent(t, got, "welfare.floor_held", "role", "k1", "region", "r1")
	absent(t, got, "welfare.floor_held", "role", "k1", "region", "r0")
}

// TestSnapshot_HoldsBetweenCalls verifies that a value stays until the next
// call replaces it, and that a call outside a step needs no TickEnded.
func TestSnapshot_HoldsBetweenCalls(t *testing.T) {
	h := newHarness(t)
	h.obs.RecordSnapshot(snapOf())
	h.send(created())
	tick(h, 0)
	check(t, h.collect(), "role.agents", 6, "role", "k1", "region", "r0")
	s := snapOf()
	s.Roles[0].Agents = 9
	h.obs.RecordSnapshot(s)
	check(t, h.collect(), "role.agents", 9, "role", "k1", "region", "r0")
}

// TestSnapshot_OutOfRangeDropsRow verifies that an ID outside the snapshot's
// names drops its series, a nil slice is fine, and nothing panics.
//
// Requirement: FR-OBS-052
func TestSnapshot_OutOfRangeDropsRow(t *testing.T) {
	h := newHarness(t)
	h.obs.RecordSnapshot(economysim.Snapshot{})
	s := snapOf()
	s.Markets = append(s.Markets, economysim.MarketRow{State: economysim.MarketState{Region: 9, Commodity: -1}, Lots: 1})
	s.Jobs = append(s.Jobs, economysim.JobRow{Region: -1, Recipe: 5, Executions: 1})
	s.Regions = append(s.Regions, economysim.RegionRow{Region: 7, MoneyPool: 1})
	s.Roles = append(s.Roles, economysim.RoleRow{Region: 0, Role: 99, Agents: 1})
	h.obs.RecordSnapshot(s)
	got := h.collect()
	for n, want := range map[string]int{"ledger.lots": 1, "recipe.in_progress": 1, "money.supply": 3, "role.agents": 2} {
		if g := len(got["economysim."+n]); g != want {
			t.Errorf("%s series = %d, want %d", n, g, want)
		}
	}
}

// TestSnapshot_RealWorld verifies the rows of a real world's snapshot: role
// agents equal the rows, held units add up, and the household money plus the
// pool is the world's money in those regions.
//
// Requirement: FR-OBS-104
func TestSnapshot_RealWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	h := newHarness(t)
	s := runWorld(t, h, 20)
	h.obs.RecordSnapshot(s)
	got := h.collect()
	for _, r := range s.Roles {
		check(t, got, "role.agents", r.Agents, "role", s.Names.Roles[r.Role], "region", s.Names.Regions[r.Region])
	}
	held, lots, money, pool := snapSums(s)
	if got.sum("market.stock") != held || got.sum("ledger.lots") != lots {
		t.Errorf("market.stock = %d, ledger.lots = %d, want %d, %d", got.sum("market.stock"), got.sum("ledger.lots"), held, lots)
	}
	if h, p := got.sumOf("money.supply", "households"), got.sumOf("money.supply", "pool"); h != money || p != pool {
		t.Errorf("money.supply households = %d, pool = %d, want %d, %d", h, p, money, pool)
	}
}

// snapSums adds up the held units, the lots and the money of a snapshot.
func snapSums(s economysim.Snapshot) (held, lots, money, pool int64) {
	for _, m := range s.Markets {
		held += int64(m.HeldByAgents + m.HeldByEstate)
		lots += m.Lots
	}
	for _, r := range s.Regions {
		money += int64(r.MoneyHouseholds)
		pool += int64(r.MoneyPool)
	}
	return held, lots, money, pool
}

// FuzzRecordSnapshot verifies that any snapshot is recorded without a panic and
// that every series has a value of its attribute from the snapshot's names.
//
// Requirement: FR-OBS-052
func FuzzRecordSnapshot(f *testing.F) {
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(5), int64(-3), int64(math.MaxInt64))
	f.Add(int64(1), int64(2), int64(1), int64(-1), int64(0), int64(1), int64(7))
	f.Fuzz(func(t *testing.T, region, commodity, role, recipe, a, b, c int64) {
		s := snapOf()
		s.Markets = append(s.Markets, economysim.MarketRow{
			State:        economysim.MarketState{Region: economysim.RegionID(region), Commodity: economysim.CommodityID(commodity)},
			HeldByAgents: economysim.Quantity(a), Lots: b, Pending: economysim.Quantity(c)})
		s.Jobs = append(s.Jobs, economysim.JobRow{Region: economysim.RegionID(region), Recipe: economysim.RecipeID(recipe), Executions: a})
		s.Regions = append(s.Regions, economysim.RegionRow{Region: economysim.RegionID(region), MoneyPool: economysim.Money(b)})
		s.Roles = append(s.Roles, economysim.RoleRow{Region: economysim.RegionID(region), Role: economysim.RoleID(role),
			Agents: a, Households: b, HeldByFloor: c, HasMeanRatio: a > 0, HasExpected: b > 0, HasWage: c > 0})
		empty := a%3 == 0
		if empty {
			s.Names = economysim.Names{}
		}
		h := newHarness(t)
		h.obs.RecordSnapshot(s)
		got := h.collect()
		if n := len(got["economysim.role.agents"]); n > len(s.Roles) || (empty && len(got) != 0) {
			t.Errorf("role.agents series = %d for %d rows, metrics = %d with empty names = %v, want none beyond the rows",
				n, len(s.Roles), len(got), empty)
		}
	})
}

// runWorld steps a miniConfig world n ticks with the harness's adapter attached
// and returns its snapshot.
func runWorld(t *testing.T, h *harness, n int) economysim.Snapshot {
	t.Helper()
	w, err := economysim.NewWorld(miniConfig(), economysim.WithSeed(7), economysim.WithObserver(h.obs))
	if err != nil {
		t.Fatalf("NewWorld() error = %v", err)
	}
	for i := 0; i < n; i++ {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() i=%d error = %v", i, err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	return s
}
