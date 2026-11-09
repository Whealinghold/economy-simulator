package indicator_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// peopleWindow is the window of 18.7: alpha grower 40 and baker 10 at tick 10;
// WeightChanged 4 to 7 (three births done) and one agent of weight 1 bankrupted
// at the end of tick 10; a weight-1 role change to baker and a weight-2
// relocation to beta at the end of tick 11. UpkeepShortfall at alpha for grower
// is 4 unfed at tick 10 and 6 at tick 11. The close is grower 39 at alpha and 2 at
// beta, baker 11, with 15 agents standing for the 39 households.
func peopleWindow(t *testing.T) (indicator.Window, economysim.Snapshot, economysim.Snapshot) {
	t.Helper()
	open, closing := valleySnap(10, [4]int64{40, 10, 0, 0}), valleySnap(12, [4]int64{39, 11, 2, 0})
	closing.Roles[0].Agents, closing.Roles[0].Unfed = 15, 2
	tl := indicator.NewTally()
	if err := tl.Open(open); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	step(tl, 10,
		economysim.UpkeepShortfall{Tick: 10, Region: alpha, Role: grower, Households: 40, Unfed: 4, Agents: 2, MaxHunger: 2},
		economysim.WeightChanged{Tick: 10, Region: alpha, Role: grower, From: 4, To: 7},
		economysim.PopulationChanged{Tick: 10, Region: alpha, Role: grower, BirthsDone: 3},
		economysim.AgentBankrupted{Tick: 10, Region: alpha, Role: grower, Weight: 1},
	)
	step(tl, 11,
		economysim.UpkeepShortfall{Tick: 11, Region: alpha, Role: grower, Households: 42, Unfed: 6, Agents: 1, MaxHunger: 3},
		economysim.RoleChanged{Tick: 11, Region: alpha, From: grower, To: baker, Weight: 1},
		economysim.AgentRelocated{Tick: 11, Role: grower, From: alpha, To: beta, Weight: 2},
	)
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	return w, open, closing
}

// TestHouseholds_AccountOfChangeCloses verifies 18.7: 39 + 11 + 2 - (40 + 10) is
// +2, which is 3 births less 1 bankruptcy; the role change and the relocation net
// to zero in all; the residual is 0 in every row; agents and households are
// separate fields; and a closing snapshot that disagrees with the events shows in
// Residual.
//
// Requirement: FR-IND-038
func TestHouseholds_AccountOfChangeCloses(t *testing.T) {
	w, open, closing := peopleWindow(t)
	got, err := indicator.Households(w, open, closing, 365)
	if err != nil {
		t.Fatalf("Households() error = %v, want nil", err)
	}
	tot := got.Total
	if tot.Open != 50 || tot.Close != 52 || tot.Bankrupted != 1 || tot.WeightChanges != 3 || tot.RoleChanges != 0 || tot.Relocated != 0 || tot.Births != 3 || tot.Residual != 0 {
		t.Errorf("Households().Total = %+v, want open 50, close 52, +3 weight, -1 bankrupted, role and relocation net 0, 3 births, residual 0", tot)
	}
	g := got.Rows[0]
	if g.Open != 40 || g.Close != 39 || g.RoleChanges != -1 || g.Relocated != -2 || g.Residual != 0 {
		t.Errorf("Households() alpha grower = %+v, want 40 -> 39 with role -1, relocated -2, residual 0", g)
	}
	if g.CloseAgents != 15 || g.Close == g.CloseAgents {
		t.Errorf("Households() alpha grower agents = %d households = %d, want 15 agents standing for 39 households", g.CloseAgents, g.Close)
	}
	if b := got.Rows[2]; b.Region != beta || b.Role != grower || b.Relocated != 2 || b.Close != 2 || b.Residual != 0 {
		t.Errorf("Households() beta grower = %+v, want the migrant (+2) counted at its destination", b)
	}
	if !got.Rate.OK || got.Rate.Value != 107_353 {
		t.Errorf("Households().Rate = %+v, want rdiv(10 000 * 3 * 365, 102) = 107 353 (3 net births over 102 household-ticks, per year)", got.Rate)
	}

	w.CloseHouseholds[0]++
	got, _ = indicator.Households(w, open, closing, 365)
	if got.Rows[0].Residual != 1 || got.Total.Residual != 1 {
		t.Errorf("Households(counts off by one) residual = %d (total %d), want 1: the account must show it", got.Rows[0].Residual, got.Total.Residual)
	}
}

// TestHouseholds_Misuse verifies that a snapshot of another window or world, a
// calendar of no ticks and a role row outside the names table are errors.
//
// Requirement: FR-IND-038
func TestHouseholds_Misuse(t *testing.T) {
	w, open, closing := peopleWindow(t)
	if _, err := indicator.Households(w, open, closing, 0); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Households(0 ticks per year) error = %v, want ErrRange", err)
	}
	if _, err := indicator.Households(w, closing, closing, 365); !errors.Is(err, indicator.ErrWindowOrder) {
		t.Errorf("Households(open at the close tick) error = %v, want ErrWindowOrder", err)
	}
	other := closing
	other.ConfigHash = [32]byte{9}
	if _, err := indicator.Households(w, open, other, 365); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Households(other world) error = %v, want ErrConfigMismatch", err)
	}
	stray := valleySnap(12, [4]int64{39, 11, 2, 0})
	stray.Roles = append(stray.Roles, economysim.RoleRow{Region: 5, Role: 0})
	if _, err := indicator.Households(w, open, stray, 365); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Households(role row of region 5) error = %v, want ErrRange", err)
	}
}

// TestUnfed_DenominatorIsTheRunningCount verifies 18.7: 4 + 6 unfed over the
// running counts 40 and 42 at the start of the ticks is rdiv(100 000, 82) = 1 220
// and not 1 235; the largest hunger is reported; the share at the boundary is the
// closing snapshot's; a window in which no UpkeepShortfall was emitted has share
// 0 and not undefined; and an agent of weight 6 unfed counts 6.
//
// Requirement: FR-IND-041
func TestUnfed_DenominatorIsTheRunningCount(t *testing.T) {
	w, _, closing := peopleWindow(t)
	got, err := indicator.Unfed(w, closing)
	if err != nil {
		t.Fatalf("Unfed() error = %v, want nil", err)
	}
	g := got.Rows[0]
	if g.Unfed != 10 || g.HouseholdTicks != 82 || g.Share.Value != 1220 || !g.Share.OK || g.MaxHunger != 3 {
		t.Errorf("Unfed() alpha grower = %+v, want 10 unfed over 82 household-ticks = 1 220 (not 1 235), max hunger 3", g)
	}
	if !g.AtClose.OK || g.AtClose.Value != 513 {
		t.Errorf("Unfed() alpha grower AtClose = %+v, want rdiv(10 000 * 2, 39) = 513", g.AtClose)
	}
	// Alpha baker was fed throughout: 10 + 10 household-ticks, no event, share 0.
	if b := got.Rows[1]; b.Unfed != 0 || b.HouseholdTicks != 20 || !b.Share.OK || b.Share.Value != 0 {
		t.Errorf("Unfed() alpha baker = %+v, want share 0 defined: no UpkeepShortfall is emitted when all are fed", b)
	}
	tot := got.Total
	if tot.Unfed != 10 || tot.HouseholdTicks != 102 || tot.Share.Value != 980 || tot.MaxHunger != 3 || tot.AtClose.Value != 385 {
		t.Errorf("Unfed().Total = %+v, want 10 over 102 = 980, max hunger 3, at close 2 of 52 = 385", tot)
	}
	// A role with no households at all has no denominator.
	if r := got.Rows[3]; r.Share.OK {
		t.Errorf("Unfed() beta baker = %+v, want undefined: no household-ticks", r)
	}
	other := closing
	other.Tick++
	if _, err := indicator.Unfed(w, other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Unfed(snapshot of another tick) error = %v, want ErrConfigMismatch", err)
	}
	other = valleySnap(12, [4]int64{})
	other.Roles = append(other.Roles, economysim.RoleRow{Region: 0, Role: 9})
	if _, err := indicator.Unfed(w, other); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Unfed(role row outside the table) error = %v, want ErrRange", err)
	}
}

// TestRoleShares_ARoleAtZeroIsReported verifies that the shares of the window's
// two boundaries are exact, that a role with no households at the close has share
// 0 and is reported, that a region with none has its share undefined, and that
// the swing of a series is its largest minus its smallest value.
//
// Requirement: FR-IND-045
func TestRoleShares_ARoleAtZeroIsReported(t *testing.T) {
	w, _, _ := peopleWindow(t)
	got, err := indicator.RoleShares(w)
	if err != nil {
		t.Fatalf("RoleShares() error = %v, want nil", err)
	}
	if len(got.Rows) != 4 {
		t.Fatalf("RoleShares() has %d rows, want 4: no role is dropped", len(got.Rows))
	}
	g := got.Rows[0]
	if g.OpenShare.Value != 8000 || g.CloseShare.Value != 7800 || g.OpenRealm.Value != 8000 || g.CloseRealm.Value != 7500 {
		t.Errorf("RoleShares() alpha grower = %+v, want 8 000 -> 7 800 of alpha and 8 000 -> 7 500 of the realm", g)
	}
	if b := got.Rows[3]; !b.CloseShare.OK || b.CloseShare.Value != 0 || b.OpenShare.OK {
		t.Errorf("RoleShares() beta baker = %+v, want close share 0 (reported) and the open share undefined (beta had no households)", b)
	}
	if b := got.Rows[2]; !b.CloseShare.OK || b.CloseShare.Value != 10_000 || b.CloseRealm.Value != 385 {
		t.Errorf("RoleShares() beta grower = %+v, want all of beta (10 000) and 385 of the realm", b)
	}
	series := []indicator.Point{{Tick: 1, Value: 8000}, {Tick: 2, Value: 0}, {Tick: 3, Value: 3000}}
	if s, ok := indicator.Swing(series); !ok || s != 8000 {
		t.Errorf("Swing() = %d, %v, want 8 000: a role that vanished is a share of 0", s, ok)
	}
	if _, ok := indicator.Swing(nil); ok {
		t.Error("Swing(nil) ok = true, want false")
	}
}

// realWageConfig is the 18.8 world: alpha prices grain 10 and loaf 25, an upkeep
// of 2 grain and 1 loaf.
func realWageConfig() economysim.Config {
	cfg := basketConfig()
	cfg.Roles = []economysim.Role{{ID: "grower"}, {ID: "baker"}}
	return cfg
}

// TestRealWage_AbsentWithoutAWage verifies 18.8: a wage of 24 against an upkeep
// of 2 * 10 + 1 * 25 = 45 is rdiv(240 000, 45) = 5 333; a role with no observed
// wage has no ratio and is not 0; a role's own upkeep list replaces the world's
// and a region's extra is added.
//
// Requirement: FR-IND-043
func TestRealWage_AbsentWithoutAWage(t *testing.T) {
	cfg := realWageConfig()
	s := priced(cfg, 10, 25, 13, 28, 0)
	s.Roles[0].WageObserved, s.Roles[0].HasWage = 24, true
	p, err := indicator.PricesOfSnapshot(s)
	if err != nil {
		t.Fatalf("PricesOfSnapshot() error = %v, want nil", err)
	}
	got, err := indicator.RealWage(cfg, s, p)
	if err != nil {
		t.Fatalf("RealWage() error = %v, want nil", err)
	}
	if r := got.Rows[0]; !r.HasWage || r.Upkeep != 45 || !r.Ratio.OK || r.Ratio.Value != 5333 {
		t.Errorf("RealWage() alpha grower = %+v, want upkeep 45 and ratio 5 333", r)
	}
	if r := got.Rows[1]; r.HasWage || r.Ratio.OK || r.Ratio.Value != 0 {
		t.Errorf("RealWage() alpha baker = %+v, want absent (HasWage false), not a ratio of 0", r)
	}

	cfg.Roles[0].Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	cfg.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "loaf", Quantity: 1}}
	s.ConfigHash = economysim.HashConfig(cfg)
	p, _ = indicator.PricesOfSnapshot(s)
	got, err = indicator.RealWage(cfg, s, p)
	if err != nil || got.Rows[0].Upkeep != 35 || got.Rows[0].Ratio.Value != 6857 {
		t.Errorf("RealWage(role upkeep + region extra) = %+v, %v, want upkeep 10 + 25 = 35 and rdiv(240 000, 35) = 6 857", got.Rows[0], err)
	}
}

// TestRealWage_Misuse verifies the sentinels of RealWage.
//
// Requirement: FR-IND-043
func TestRealWage_Misuse(t *testing.T) {
	cfg := realWageConfig()
	s := priced(cfg, 10, 25, 13, 28, 0)
	s.Roles[0].WageObserved, s.Roles[0].HasWage = 24, true
	p, _ := indicator.PricesOfSnapshot(s)

	other := s
	other.ConfigHash = [32]byte{3}
	if _, err := indicator.RealWage(cfg, other, p); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("RealWage(snapshot of another world) error = %v, want ErrConfigMismatch", err)
	}
	other = s
	p.ConfigHash = [32]byte{3}
	if _, err := indicator.RealWage(cfg, other, p); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("RealWage(table of another world) error = %v, want ErrConfigMismatch", err)
	}
	p, _ = indicator.PricesOfSnapshot(s)
	bad := s
	bad.Roles = append([]economysim.RoleRow(nil), s.Roles...)
	bad.Roles[0].Role = 7
	if _, err := indicator.RealWage(cfg, bad, p); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("RealWage(role 7) error = %v, want ErrRange", err)
	}
	cfg.World.Upkeep = append(cfg.World.Upkeep, economysim.Item{Commodity: "gold", Quantity: 1})
	s.ConfigHash = economysim.HashConfig(cfg)
	p, _ = indicator.PricesOfSnapshot(s)
	if _, err := indicator.RealWage(cfg, s, p); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("RealWage(upkeep of an unknown commodity) error = %v, want ErrRange", err)
	}
}
