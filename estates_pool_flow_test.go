package economysim_test

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// poolFlow is a flow of the kind aimed at the pool of the region, or of every
// region when region is empty, with an amount of 100 a tick.
func poolFlow(id, kind, region string) economysim.MoneyFlow {
	return economysim.MoneyFlow{ID: id, Kind: kind, Amount: 100, Target: economysim.FlowTarget{Kind: economysim.TargetPool, Region: region}}
}

// poolConfig is the world of the step tests with two regions, "home" with 20
// households and "away" with 5, the estates module on and relief off
// (PoolRelease 0), so that only the flows move a pool.
func poolConfig(flows ...economysim.MoneyFlow) economysim.Config {
	c := moneyWorldConfig(10, 10, flows...)
	c.Modules.Estates = &economysim.Estates{}
	away := economysim.DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	c.Groups = append(c.Groups, economysim.Group{Region: "away", Role: "farmer", Households: 5})
	for i := range c.Regions {
		c.Regions[i].PoolRelease = 0
	}
	return c
}

// pools returns the pool of "away" and of "home", in that order (the regions are in
// ID order), from the snapshot.
func pools(t *testing.T, w *economysim.World) []economysim.Money {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	var out []economysim.Money
	for _, r := range s.Regions {
		out = append(out, r.MoneyPool)
	}
	if !slices.Equal(s.Names.Regions, []string{"away", "home"}) {
		t.Fatalf("regions = %v, want [away home]", s.Names.Regions)
	}
	return out
}

// TestValidate_PoolTargetNeedsEstates verifies the rule of spec-money-supply.md
// 4.3.2 and 12.1: a grant or a levy with a pool target is ErrModuleDependency
// while the estates module is off and valid with it on, a pool target with a role
// is ErrFlowShape and one that names an undefined region is ErrUnknownRegion.
//
// Requirement: FR-MON-038
func TestValidate_PoolTargetNeedsEstates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		flow   economysim.MoneyFlow
		on     bool
		want   error
		suffix string
	}{
		{"grant, module off", poolFlow("aid", economysim.FlowGrant, ""), false, economysim.ErrModuleDependency, "target.kind"},
		{"levy, module off", poolFlow("tax", economysim.FlowLevy, ""), false, economysim.ErrModuleDependency, "target.kind"},
		{"grant, module on", poolFlow("aid", economysim.FlowGrant, ""), true, nil, ""},
		{"levy, module on", poolFlow("tax", economysim.FlowLevy, "home"), true, nil, ""},
		{"a role on a pool", func() economysim.MoneyFlow {
			f := poolFlow("aid", economysim.FlowGrant, "")
			f.Target.Role = "farmer"
			return f
		}(), true, economysim.ErrFlowShape, "target.role"},
		{"an undefined region", poolFlow("aid", economysim.FlowGrant, "nowhere"), true, economysim.ErrUnknownRegion, "target.region"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := poolConfig(tt.flow)
			if !tt.on {
				c.Modules.Estates = nil
			}
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate = %v, want nil", err)
				}
				return
			}
			var found bool
			for _, ve := range validationErrors(err) {
				found = found || (errors.Is(ve, tt.want) && strings.HasSuffix(strings.Join(ve.Path, "."), tt.suffix))
			}
			if !found {
				t.Errorf("Validate = %v, want %v at a path ending %s", err, tt.want, tt.suffix)
			}
		})
	}
}

// TestValidate_GrantToPoolNoOutlet verifies the warning of spec-money-supply.md
// 12.2: a grant to a pool in a region whose PoolRelease is 0 warns at the path of
// the flow, a grant to a region that releases does not, a levy on a pool does not,
// and none of them fails Validate.
//
// Requirement: FR-MON-039
func TestValidate_GrantToPoolNoOutlet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		flow    economysim.MoneyFlow
		release map[string]int64
		warns   bool
	}{
		{"every region, one closed", poolFlow("aid", economysim.FlowGrant, ""), map[string]int64{"home": 100, "away": 0}, true},
		{"a closed region named", poolFlow("aid", economysim.FlowGrant, "away"), map[string]int64{"home": 100, "away": 0}, true},
		{"the closed region is not the one named", poolFlow("aid", economysim.FlowGrant, "home"), map[string]int64{"home": 100, "away": 0}, false},
		{"every region releases", poolFlow("aid", economysim.FlowGrant, ""), map[string]int64{"home": 100, "away": 100}, false},
		{"a levy", poolFlow("tax", economysim.FlowLevy, ""), map[string]int64{"home": 0, "away": 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := poolConfig(tt.flow)
			for i, r := range c.Regions {
				c.Regions[i].PoolRelease = tt.release[r.ID]
			}
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate = %v, want nil: a warning never fails", err)
			}
			var got bool
			for _, w := range c.Warnings() {
				got = got || (w.Rule == "grant_to_pool_no_outlet" && slices.Equal(w.Path, []string{"modules", "money_supply", "flows", tt.flow.ID}))
			}
			if got != tt.warns {
				t.Errorf("warning grant_to_pool_no_outlet = %v, want %v (warnings %v)", got, tt.warns, c.Warnings())
			}
		})
	}
}

// TestFlow_PoolClassMembers verifies the class of a pool target
// (spec-money-supply.md 4.3.2, 4.3.3): the pool of the named region alone, or the
// pool of every region, each as one recipient of weight 1 whatever its households.
//
// Arithmetic: the grant is 100 a tick and the world has no relief, no removal and
// no trade, so after one tick a pool the flow names holds 100 and the others 0.
//
// Requirement: FR-MON-008
func TestFlow_PoolClassMembers(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		region string
		want   []economysim.Money
	}{{"", []economysim.Money{100, 100}}, {"home", []economysim.Money{0, 100}}, {"away", []economysim.Money{100, 0}}} {
		t.Run("region "+tt.region, func(t *testing.T) {
			t.Parallel()
			w := moneyWorld(t, poolConfig(poolFlow("aid", economysim.FlowGrant, tt.region)))
			step(t, w, 1)
			if got := pools(t, w); !slices.Equal(got, tt.want) {
				t.Errorf("pools [away home] after one tick = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFlow_PoolGrantFeedsRelief verifies that a grant to a pool is a faucet into
// the pool at P1.7 and reaches the households only by relief: none is paid in the
// tick of the first grant (relief at P1.6 runs before the flow), the next tick
// pays relief out of the pool the grant filled, and the audit holds throughout.
//
// Arithmetic: the grant is 100 a tick to the pool of "home"; PoolRelease is 10 000
// (all of it) so the release at tick 1 is the pool as tick 0 left it, 100.
//
// Requirement: FR-MON-019, FR-MON-046
func TestFlow_PoolGrantFeedsRelief(t *testing.T) {
	t.Parallel()
	c := poolConfig(poolFlow("aid", economysim.FlowGrant, "home"))
	c.Regions[0].PoolRelease, c.Regions[0].ReliefBelow = 10_000, economysim.MaxBasisPoints
	var paid []economysim.ReliefPaid
	w := moneyWorld(t, c, economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
		if r, ok := e.(economysim.ReliefPaid); ok {
			paid = append(paid, r)
		}
	})))
	step(t, w, 1)
	if len(paid) != 0 {
		t.Errorf("ReliefPaid in tick 0 = %+v, want none: the grant of tick 0 is not in the pool at P1.6", paid)
	}
	if got := pools(t, w)[1]; got != 100 {
		t.Fatalf("pool after tick 0 = %d, want the grant 100", got)
	}
	step(t, w, 1)
	if len(paid) != 1 || paid[0].Tick != 1 || paid[0].Paid != 100 {
		t.Errorf("ReliefPaid after tick 1 = %+v, want one at tick 1 paying 100, the pool the grant filled", paid)
	}
	if got := flowRow(t, w, "aid").Total; got != 200 {
		t.Errorf("flow total after two ticks = %d, want 200", got)
	}
}

// TestFlow_PoolLevyAfterRelief verifies that a levy on a pool collects after relief
// was paid in the same tick (P1.6 precedes P1.7) and never takes money relief
// committed: with PoolRelease at its bound the whole pool goes out as relief, the
// levy finds 0 and is reported short.
//
// Arithmetic: the pool starts at 1 000 and releases all of it; the 20 households
// of "home" have a need of at least 1 000 together (upkeep 1 000 a tick at a
// relief line of 10 000 and a ratio of 0), so 1 000 is paid and the pool is 0 when
// the levy of 100 runs.
//
// Requirement: FR-MON-019, FR-EST-017
func TestFlow_PoolLevyAfterRelief(t *testing.T) {
	t.Parallel()
	c := poolConfig(poolFlow("tax", economysim.FlowLevy, "home"))
	c.Regions[0].PoolRelease, c.Regions[0].ReliefBelow = 10_000, economysim.MaxBasisPoints
	c.WarmStart = &economysim.WarmStart{Pools: []economysim.WarmPool{{Region: "home", Money: 1_000}}}
	var events []economysim.Event
	w := moneyWorld(t, c, economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
		switch e.(type) {
		case economysim.ReliefPaid, economysim.MoneyFlowLimited:
			events = append(events, e)
		}
	})))
	step(t, w, 1)
	if len(events) != 2 {
		t.Fatalf("events of tick 0 = %+v, want ReliefPaid then MoneyFlowLimited", events)
	}
	if r, ok := events[0].(economysim.ReliefPaid); !ok || r.Paid != 1_000 {
		t.Errorf("first event = %+v, want ReliefPaid of 1 000: relief runs before the levy", events[0])
	}
	if l, ok := events[1].(economysim.MoneyFlowLimited); !ok || l.Flow != "tax" || l.Due != 100 || l.Applied != 0 || l.Short != 1 {
		t.Errorf("second event = %+v, want the levy short: due 100, applied 0, one household", events[1])
	}
	if got := pools(t, w)[1]; got != 0 {
		t.Errorf("pool after the tick = %d, want 0", got)
	}
}

// poolCarryConfig is poolConfig with one rate grant of 7 a year to every pool.
func poolCarryConfig() economysim.Config {
	f := poolFlow("aid", economysim.FlowGrant, "")
	f.Amount, f.Per = 7, economysim.PerYear
	return poolConfig(f)
}

// poolCarryRaw steps the world of poolCarryConfig n ticks and returns its
// checkpoint bytes.
func poolCarryRaw(t *testing.T, n int) []byte {
	t.Helper()
	w := moneyWorld(t, poolCarryConfig())
	step(t, w, n)
	return mustMarshal(t, mustCheckpoint(t, w))
}

// TestPoolCarry_StartByRegionIndex verifies that the carry of a pool starts at the
// region's index modulo the period times 10^4 and is saved as (region, carry)
// pairs (spec-money-supply.md 6.2, 14): none before the first visit, and after
// tick 0 the pools of "away" (index 0) and "home" (index 1) hold 0 + 7 * 10^4 and
// 10^4 + 7 * 10^4, because a year is 365 ticks and the first due is below 1.
//
// Requirement: FR-MON-013
func TestPoolCarry_StartByRegionIndex(t *testing.T) {
	t.Parallel()
	if body := bodyValues(t, tagBody(t, poolCarryRaw(t, 0), 13)); !slices.Equal(body, []uint64{0, 10_000, 0, 0}) {
		t.Errorf("tag 13 before the first step = %v, want one flow at total 0, scale 10 000, not ended, and no carry", body)
	}
	if body := bodyValues(t, tagBody(t, poolCarryRaw(t, 1), 13)); !slices.Equal(body, []uint64{0, 10_000, 0, 2, 0, 70_000, 1, 80_000}) {
		t.Errorf("tag 13 after tick 0 = %v, want two (region, carry) pairs: (0, 70 000) and (1, 80 000)", body)
	}
}

// TestPoolCarry_AdvancesByDue verifies that a pool rate grant over a year pays
// exactly its amount to each pool: 7 a year over 365 ticks is 7 to each of the two
// pools, 14 in the flow's total, however the carry's start spreads the payments.
//
// Arithmetic: over n ticks a recipient is paid floor((n * 7 * 10^4 + c0) / (365 *
// 10^4)) and c0 is below 365 * 10^4, so n = 365 gives 7.
//
// Requirement: FR-MON-012, FR-MON-013
func TestPoolCarry_AdvancesByDue(t *testing.T) {
	t.Parallel()
	w := moneyWorld(t, poolCarryConfig())
	step(t, w, 365)
	if got := pools(t, w); !slices.Equal(got, []economysim.Money{7, 7}) {
		t.Errorf("pools after a year = %v, want [7 7]", got)
	}
	if got := flowRow(t, w, "aid").Total; got != 14 {
		t.Errorf("flow total after a year = %d, want 14", got)
	}
}

// TestCheckpoint_PoolCarryRoundTrip verifies that tag 13 of a world with a pool
// flow holds the region pairs, restores to a world whose checkpoint has the same
// bytes, and that the restored world pays what the original pays.
//
// Requirement: FR-MON-046, FR-CHK-023
func TestCheckpoint_PoolCarryRoundTrip(t *testing.T) {
	t.Parallel()
	raw := poolCarryRaw(t, 40)
	w, err := economysim.RestoreWorld(poolCarryConfig(), reframe(t, raw, func(s []section) []section { return s }))
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if again := mustMarshal(t, mustCheckpoint(t, w)); !bytes.Equal(again, raw) {
		t.Error("the restored world's checkpoint differs from the one it was restored from")
	}
	orig := moneyWorld(t, poolCarryConfig())
	step(t, orig, 40+400)
	step(t, w, 400)
	if a, b := flowRow(t, orig, "aid").Total, flowRow(t, w, "aid").Total; a != b {
		t.Errorf("flow total of the original %d and of the restored world %d after 400 more ticks, want them equal", a, b)
	}
}

// TestCheckpoint_PoolCarryCorrupt verifies that tag 13 is refused when a pool pair
// names a region outside the regions, when the regions are repeated or descending,
// when a carry is at its denominator or a count is above the bytes.
//
// Requirement: FR-MON-046, FR-CHK-023
func TestCheckpoint_PoolCarryCorrupt(t *testing.T) {
	t.Parallel()
	raw := poolCarryRaw(t, 5)
	base := bodyValues(t, tagBody(t, raw, 13))
	if len(base) != 8 {
		t.Fatalf("tag 13 after 5 ticks = %v, want a header, a count and two pairs", base)
	}
	const den = 365 * 10_000
	tests := []struct {
		name string
		edit func(v []uint64)
	}{
		{"region beyond the regions", func(v []uint64) { v[6] = 2 }},
		{"regions repeated", func(v []uint64) { v[6] = 0 }},
		{"regions descending", func(v []uint64) { v[4], v[6] = 1, 0 }},
		{"carry at its denominator", func(v []uint64) { v[5] = den }},
		{"count above the bytes", func(v []uint64) { v[3] = 1 << 40 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ck := reframe(t, raw, func(s []section) []section {
				return editSection(s, 13, func([]byte) []byte {
					v := slices.Clone(base)
					tt.edit(v)
					return bodyBytes(v)
				})
			})
			if _, err := economysim.RestoreWorld(poolCarryConfig(), ck); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestFork_Estates verifies fork-at-N with the estates module on, on pack famine
// with a seeded pool and a rate grant to the pool of the north (so a pool carry
// is mid-year), at tick 0, inside a job (37), inside a season (150), and after a
// call between ticks (a yield shock at 90). Relief has paid and the estate holds
// listed lots at the later points. The original, the restore and the restore of
// the restore must agree on snapshots, events and the state hash.
//
// Requirement: FR-EST-024, FR-CHK-023, FR-CHK-050, FR-MON-046
func TestFork_Estates(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packLayers["famine"]...)
	cfg.WarmStart = &economysim.WarmStart{Pools: []economysim.WarmPool{{Region: "north", Money: 400_000}}}
	cfg.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{ID: "pool_grant", Kind: economysim.FlowGrant,
		Target: economysim.FlowTarget{Kind: economysim.TargetPool, Region: "north"}, Amount: 700, Per: economysim.PerYear}}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate = %v, want nil", err)
	}
	for _, n := range []int{0, 37, 150} {
		forkAtN(t, forkCase{Name: "famine+estates", Cfg: cfg, Seed: 1, N: n, M: forkM})
	}
	shock := func(w *economysim.World) {
		if err := economysim.SetYieldShockForTest(w, 0, 0, 3_000); err != nil {
			t.Fatalf("SetYieldShockForTest error = %v, want nil", err)
		}
	}
	forkAtN(t, forkCase{Name: "famine+estates after a shock", Cfg: cfg, Seed: 1, N: 90, M: forkM, Before: shock})
}
