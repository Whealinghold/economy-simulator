package economysim_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestSaving_DefaultsAndBounds verifies the defaults of spec-saving-and-investment.md
// 10.2 and the bounds that go with them: each default sits inside its bound, the
// products the spec needs fit in MaxMoney, and the loader and a Go-built configuration
// start from the same values.
//
// Requirement: FR-CFG-096, FR-SAV-034
//
// Guard: a default moved outside its bound would make every configuration that starts
// from it fail Validate; a Worth bound too large would let PerTick * PaybackTicks wrap.
func TestSaving_DefaultsAndBounds(t *testing.T) {
	t.Parallel()
	m, b := economysim.DefaultSaving(), economysim.DefaultSaver()
	if want := (economysim.Saving{SafetyTicks: 2, HurdleSpreadBP: 20_000, CommitBandBP: 2_000}); m != want {
		t.Errorf("DefaultSaving() = %+v, want %+v", m, want)
	}
	if want := (economysim.Saver{PrecautionCapTicks: 30, HurdleBP: 15_000, PaybackTicks: 180, RepairBelowBP: 5_000}); !reflect.DeepEqual(b, want) {
		t.Errorf("DefaultSaver() = %+v, want %+v (no buffer, no Worth rows)", b, want)
	}
	in := func(name string, v, lo, hi int64) {
		if v < lo || v > hi {
			t.Errorf("default %s = %d, want within [%d, %d]", name, v, lo, hi)
		}
	}
	in("SafetyTicks", m.SafetyTicks, 0, economysim.MaxSafetyTicks)
	in("HurdleSpreadBP", m.HurdleSpreadBP, 0, economysim.MaxHurdleSpreadBP)
	in("CommitBandBP", m.CommitBandBP, 0, economysim.MaxCommitBandBP)
	in("PrecautionShareBP", b.PrecautionShareBP, 0, economysim.MaxPrecautionShareBP)
	in("PrecautionCapTicks", b.PrecautionCapTicks, 0, economysim.MaxPrecautionCapTicks)
	in("HurdleBP", b.HurdleBP, 10_000, economysim.MaxHurdleBP)
	in("PaybackTicks", b.PaybackTicks, 1, economysim.MaxPaybackTicks)
	in("RepairBelowBP", b.RepairBelowBP, 0, 10_000)
	// MaxWorthPerTick = MaxMoney / MaxPaybackTicks, so the worth over the longest horizon
	// is at most MaxMoney: the bound the spec's 128-bit argument (11.3) rests on.
	if got := int64(economysim.MaxWorthPerTick) * economysim.MaxPaybackTicks; got > int64(economysim.MaxMoney) {
		t.Errorf("MaxWorthPerTick * MaxPaybackTicks = %d, want at most MaxMoney %d", got, int64(economysim.MaxMoney))
	}
	// The edges are valid: a bound is the largest value, not one past it.
	c := savingConfig()
	*c.Modules.Saving = economysim.Saving{SafetyTicks: economysim.MaxSafetyTicks, HurdleSpreadBP: economysim.MaxHurdleSpreadBP, CommitBandBP: economysim.MaxCommitBandBP}
	*roleOf(&c, "grower").Saver = economysim.Saver{
		PrecautionShareBP: economysim.MaxPrecautionShareBP, PrecautionCapTicks: economysim.MaxPrecautionCapTicks,
		HurdleBP: economysim.MaxHurdleBP, PaybackTicks: economysim.MaxPaybackTicks, RepairBelowBP: 10_000,
		Worth: []economysim.ProjectWorth{{Class: "mill", PerTick: economysim.MaxWorthPerTick, Count: economysim.MaxWorthCount}},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() with every field at its bound = %v, want nil", err)
	}
	lo := savingConfig()
	*lo.Modules.Saving = economysim.Saving{}
	*roleOf(&lo, "grower").Saver = economysim.Saver{HurdleBP: 10_000, PaybackTicks: 1}
	if err := lo.Validate(); err != nil {
		t.Errorf("Validate() with every field at its lower bound = %v, want nil", err)
	}
}

// TestSavingWarnings_ReferencePriceRulesNeedAWorld verifies the three warnings that read
// the reference prices (project_below_hurdle, spread_below_margin and
// saving_target_exceeds_money): each fires on the valid configuration changed to trip
// it. With no group, or no priced commodity, the project cannot be valued and all three
// are silent, with no panic. With groups of no household the two project warnings still
// fire (spec 11.2: a pure function of the configuration, the prices are the region's)
// and the target one is silent (nobody to bind: 0 is not above 0).
//
// Requirement: FR-SAV-035
//
// Guard: a project the rule cannot value must read as "no project", not as a project
// worth 0; the second would raise project_below_hurdle for every unpriced world.
func TestSavingWarnings_ReferencePriceRulesNeedAWorld(t *testing.T) {
	t.Parallel()
	trips := []struct {
		rule   string
		change func(c *economysim.Config)
	}{
		{"project_below_hurdle@roles.grower.saver", func(c *economysim.Config) { recipeOf(c, "grind").Outputs[0].Quantity = 4 }},
		{"spread_below_margin@roles.grower.saver", func(c *economysim.Config) { c.Modules.Saving.HurdleSpreadBP = 1_000 }},
		{"saving_target_exceeds_money@modules.saving", func(c *economysim.Config) {
			for i := range c.Roles {
				c.Roles[i].StartMoney = 1
			}
		}},
	}
	worlds := []struct {
		name      string
		empty     func(c *economysim.Config)
		stillFire string // the rule prefix that fires anyway
	}{
		{"no household", func(c *economysim.Config) {
			for i := range c.Groups {
				c.Groups[i].Households = 0
			}
		}, "project_below_hurdle spread_below_margin"},
		{"no group", func(c *economysim.Config) { c.Groups = nil }, ""},
		{"no priced commodity", func(c *economysim.Config) { c.Commodities = nil }, ""},
	}
	for _, tr := range trips {
		c := savingConfig()
		tr.change(&c)
		if got := savingWarningsOf(c); !slices.Contains(got, tr.rule) {
			t.Fatalf("Warnings() = %v, want %q: the control case does not fire", got, tr.rule)
		}
		for _, w := range worlds {
			t.Run(tr.rule+" with "+w.name, func(t *testing.T) {
				t.Parallel()
				c := savingConfig()
				tr.change(&c)
				w.empty(&c)
				fires := slices.Contains(savingWarningsOf(c), tr.rule)
				want := strings.Contains(w.stillFire, strings.SplitN(tr.rule, "@", 2)[0])
				if fires != want {
					t.Errorf("Warnings() with %s fires %q = %v, want %v", w.name, tr.rule, fires, want)
				}
			})
		}
	}
}

// TestSaving_OnEqualsOffOverSeeds verifies that a world built with the saving module on
// and a Saver block on a role is the world with the module off, bit for bit, over a few
// seeds: the snapshot after every tick (but its ConfigHash) and the events of the run
// are equal. This change gives the module no position, so nothing may read it.
//
// Requirement: FR-SAV-001
//
// Guard: a read of the module's data before its positions exist (a default that moves a
// spend bound, a draw from the stream) would move every golden of a consumer that
// switches the module on early.
func TestSaving_OnEqualsOffOverSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const ticks = 20
	for seed := uint64(1); seed <= 3; seed++ {
		t.Run("seed "+string(rune('0'+seed)), func(t *testing.T) {
			t.Parallel()
			on, off := savingConfig(), savingConfig()
			off.Modules.Saving = nil
			roleOf(&off, "grower").Saver = nil
			recOn, recOff := &recorder{}, &recorder{}
			a, errA := economysim.NewWorld(on, economysim.WithSeed(seed), economysim.WithObserver(recOn))
			b, errB := economysim.NewWorld(off, economysim.WithSeed(seed), economysim.WithObserver(recOff))
			if errA != nil || errB != nil {
				t.Fatalf("NewWorld(seed %d) = %v, %v", seed, errA, errB)
			}
			recOn.forget() // WorldCreated carries the configuration hash, which differs by design
			recOff.forget()
			for tick := 1; tick <= ticks; tick++ {
				if err := a.Step(); err != nil {
					t.Fatalf("seed %d tick %d: Step with the module on = %v", seed, tick, err)
				}
				if err := b.Step(); err != nil {
					t.Fatalf("seed %d tick %d: Step with the module off = %v", seed, tick, err)
				}
				sa, errA := a.Snapshot()
				sb, errB := b.Snapshot()
				if errA != nil || errB != nil {
					t.Fatalf("seed %d tick %d: Snapshot = %v, %v", seed, tick, errA, errB)
				}
				sa.ConfigHash, sb.ConfigHash = [32]byte{}, [32]byte{}
				if !reflect.DeepEqual(sa, sb) {
					t.Fatalf("seed %d tick %d: the module on changed the snapshot, want it ignored", seed, tick)
				}
			}
			if !reflect.DeepEqual(recOn.events, recOff.events) {
				t.Errorf("seed %d: events with the module on differ from off (%d against %d), want equal", seed, len(recOn.events), len(recOff.events))
			}
		})
	}
}
