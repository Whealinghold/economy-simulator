package economysim_test

import (
	"fmt"
	"log/slog"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the scale conformance suite on the real packs (plan 2-4 D285,
// spec-scale.md 7.1): the pack `one` with overlay/h200 (200 households in each of
// six roles, 1 200 in all) and the second setting, built at weights above 1. The
// per-module weight behaviour is tested at unit level by the Stage 1 rows.

// scalePacks are the packs of the suite: the layers, in order, with the warm
// start last (the overlay changes the households, the warm start is per
// household).
var scalePacks = []struct {
	name   string
	layers []string
}{
	{"one-h200", []string{"base.toml", "one.toml", "overlay/h200.toml", "warm-one.toml"}},
	{"second", packSecond},
}

// scaleBuilt is a world of a scaled pack with what the suite reads from it.
type scaleBuilt struct {
	cfg   economysim.Config
	w     *economysim.World
	rec   *recorder
	snap  economysim.Snapshot
	names economysim.Names
}

// buildScaled loads a pack, sets the world's scale and floor, and builds it with
// the audit on and the recording observer.
func buildScaled(t testing.TB, layers []string, scale, floor int64, edit func(*economysim.Config)) scaleBuilt {
	t.Helper()
	cfg := loadValidPack(t, layers...)
	cfg.World.Scale, cfg.World.MinAgents = scale, floor
	if edit != nil {
		edit(&cfg)
	}
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(scale %d, floor %d) error = %v, want nil", scale, floor, err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(scale %d, floor %d) error = %v, want nil", scale, floor, err)
	}
	return scaleBuilt{cfg: cfg, w: w, rec: rec, snap: snap, names: snap.Names}
}

// expectedStart restates the endowment of spec-agents.md 3.5.1 from the
// configuration alone: for each group, its households times the money and the
// goods per household of the warm start's entry, or of the role when the group
// has none.
func expectedStart(cfg economysim.Config) (money int64, goods map[string]int64) {
	goods = map[string]int64{}
	for _, g := range cfg.Groups {
		perMoney, stock := int64(0), []economysim.Item(nil)
		found := false
		if cfg.WarmStart != nil {
			for _, wg := range cfg.WarmStart.Groups {
				if wg.Region == g.Region && wg.Role == g.Role {
					perMoney, stock, found = int64(wg.Money), wg.Stock, true
				}
			}
		}
		if !found {
			for _, r := range cfg.Roles {
				if r.ID == g.Role {
					perMoney, stock = int64(r.StartMoney), r.StartStock
				}
			}
		}
		money += g.Households * perMoney
		for _, it := range stock {
			goods[it.Commodity] += g.Households * int64(it.Quantity)
		}
	}
	return money, goods
}

// spawnedByGroup sums the AgentSpawned events by (region, role) name: the agents
// and their weights, and the largest and least weight.
type spawnedGroup struct{ agents, households, lo, hi int64 }

func spawnedByGroup(b scaleBuilt) map[[2]string]spawnedGroup {
	out := map[[2]string]spawnedGroup{}
	for _, e := range b.rec.events {
		s, ok := e.(economysim.AgentSpawned)
		if !ok {
			continue
		}
		k := [2]string{b.names.Regions[s.Region], b.names.Roles[s.Role]}
		g := out[k]
		if g.agents == 0 {
			g.lo, g.hi = s.Weight, s.Weight
		}
		g.agents++
		g.households += s.Weight
		g.lo, g.hi = min(g.lo, s.Weight), max(g.hi, s.Weight)
		out[k] = g
	}
	return out
}

// checkGroupsOf asserts the exact census of every group of a built world: the
// weights add up to the households, differ by at most 1, are each at least 1,
// and there are as many agents as the formula says, whose scale is the group's
// own or the world's (FR-SCL-003, 005, 006).
func checkGroupsOf(t *testing.T, b scaleBuilt) {
	t.Helper()
	got := spawnedByGroup(b)
	for _, g := range b.cfg.Groups {
		scale := g.Scale
		if scale == 0 {
			scale = b.cfg.World.Scale
		}
		s := got[[2]string{g.Region, g.Role}]
		if want := wantAgents(g.Households, scale, b.cfg.World.MinAgents); s.agents != want || s.households != g.Households || s.hi-s.lo > 1 || s.lo < 1 {
			t.Errorf("group %s/%s (%d households, scale %d, floor %d): %d agents holding %d households, weights %d to %d; want %d agents holding %d, spread at most 1, least 1",
				g.Region, g.Role, g.Households, scale, b.cfg.World.MinAgents, s.agents, s.households, s.lo, s.hi, want, g.Households)
		}
	}
}

// checkTotalsOf asserts the money of the households and the units of each good
// equal households times the per-household amount (FR-SCL-007, 019).
func checkTotalsOf(t *testing.T, b scaleBuilt) {
	t.Helper()
	wantMoney, wantGoods := expectedStart(b.cfg)
	var money int64
	for _, r := range b.snap.Regions {
		money += int64(r.MoneyHouseholds)
	}
	if money != wantMoney {
		t.Errorf("money of the households = %d, want %d (households times money per household)", money, wantMoney)
	}
	gotGoods := map[string]int64{}
	for _, m := range b.snap.Markets {
		gotGoods[b.names.Commodities[m.State.Commodity]] += int64(m.HeldByAgents)
	}
	for c, want := range wantGoods {
		if gotGoods[c] != want {
			t.Errorf("units of %s held by the households = %d, want %d (households times units per household)", c, gotGoods[c], want)
		}
	}
}

// TestScalePack_ConstructionTotalsAreExactAtEveryScale verifies, on the packs
// `one` (with 200 households in each role) and the second setting, at scales 1, 4,
// 10 and 40 and floors 1 and 10, with the world's scale and with a group that gives
// its own, that construction is exact: the weights of each group add up to its
// households and differ by at most 1, the number of agents is ceil(H / S) raised
// to min(K, H), the money and the units of each good held by the households are
// the households times the per-household amount whatever the scale, and the audit
// of the ledger passes.
//
// Arithmetic: 136 cooks at scale 4 and floor 10 are n = ceil(136 / 4) = 34 agents
// of weight 4, and 34 * 4 = 136; 200 households at scale 40 are 5 agents of
// weight 40.
//
// Requirement: FR-SCL-003, FR-SCL-006, FR-SCL-007, FR-SCL-019
func TestScalePack_ConstructionTotalsAreExactAtEveryScale(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, p := range scalePacks {
		for _, scale := range []int64{1, 4, 10, 40} {
			for _, floor := range []int64{1, 10} {
				for _, own := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s scale %d floor %d own %t", p.name, scale, floor, own), func(t *testing.T) {
						t.Parallel()
						// A group that gives its own scale overrides the world's.
						edit := func(c *economysim.Config) {
							if own {
								c.Groups[0].Scale = 2
							}
						}
						b := buildScaled(t, p.layers, scale, floor, edit)
						checkGroupsOf(t, b)
						checkTotalsOf(t, b)
					})
				}
			}
		}
	}
}

// checkSpawnedAgainstSnapshot asserts the agents and households of the AgentSpawned
// events of b equal the snapshot's at the world, role and region level.
func checkSpawnedAgainstSnapshot(t *testing.T, b scaleBuilt, scale, wantAgents int64) {
	t.Helper()
	var agents, households int64
	byRole := map[string][2]int64{}
	for k, g := range spawnedByGroup(b) {
		agents, households = agents+g.agents, households+g.households
		r := byRole[k[1]]
		byRole[k[1]] = [2]int64{r[0] + g.agents, r[1] + g.households}
	}
	s := b.snap
	if agents != wantAgents || households != 1200 || s.Agents != wantAgents || s.Households != 1200 {
		t.Errorf("scale %d: events %d agents, %d households; snapshot %d and %d; want %d and 1200", scale, agents, households, s.Agents, s.Households, wantAgents)
	}
	checkRoleRows(t, b, scale, byRole)
	if ra, rh := regionTotals(s); ra != wantAgents || rh != 1200 {
		t.Errorf("scale %d: region rows %d agents, %d households, want %d and 1200", scale, ra, rh, wantAgents)
	}
}

// checkRoleRows asserts each role row of the snapshot against the totals of the
// AgentSpawned events of the role.
func checkRoleRows(t *testing.T, b scaleBuilt, scale int64, byRole map[string][2]int64) {
	t.Helper()
	for _, r := range b.snap.Roles {
		if want := byRole[b.names.Roles[r.Role]]; r.Agents != want[0] || r.Households != want[1] {
			t.Errorf("scale %d: role %s row %d agents, %d households, want %d and %d from AgentSpawned", scale, b.names.Roles[r.Role], r.Agents, r.Households, want[0], want[1])
		}
	}
}

// regionTotals sums the agents and the households of the region rows.
func regionTotals(s economysim.Snapshot) (agents, households int64) {
	for _, r := range s.Regions {
		agents, households = agents+r.Agents, households+r.Households
	}
	return agents, households
}

// TestScalePack_AgentSpawnedWeightsAndSnapshotCounts verifies that at each scale
// the initial AgentSpawned events carry the weights, whose sum per (region, role)
// is the group's households, and that the snapshot reports agents and households
// as separate counts, equal to the events' at the world, region and role level.
//
// Arithmetic: `one` with h200 has 6 * 200 = 1 200 households, which are 1 200
// agents at scale 1, 6 * ceil(200 / 4) = 300 at scale 4 and 6 * 20 = 120 at scale
// 10 (floor 10 is below 20).
//
// Requirement: FR-SCL-018, FR-SCL-015
func TestScalePack_AgentSpawnedWeightsAndSnapshotCounts(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tt := range []struct{ scale, agents int64 }{{1, 1200}, {4, 300}, {10, 120}} {
		t.Run(fmt.Sprintf("scale %d", tt.scale), func(t *testing.T) {
			t.Parallel()
			b := buildScaled(t, scalePacks[0].layers, tt.scale, 10, nil)
			checkSpawnedAgainstSnapshot(t, b, tt.scale, tt.agents)
		})
	}
}

// TestScalePack_HouseholdCountIsTheSumOfWeights verifies, over 200 ticks of `one`
// with 200 households per role at scale 4, a run in which the failure ladder
// reduces weights, that after every tick the households of the snapshot equal the
// sum over its region rows and over its role rows, equal the count rebuilt from
// AgentSpawned, AgentBankrupted and WeightChanged, and equal the households of the
// "tick ended" line (the live counter). The audit is on, so goods and money are
// conserved across every reduction. The run must contain a WeightChanged, or the
// guard exercises no change of weights.
//
// Requirement: FR-SCL-015, FR-SCL-014
// Guard: a reduction that changed the live counter but not the snapshot, or the
// events but not the counter, would leave the households count wrong with no
// other test noticing.
func TestScalePack_HouseholdCountIsTheSumOfWeights(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lg := newCaptureHandler()
	cfg := loadValidPack(t, scalePacks[0].layers...)
	cfg.World.Scale, cfg.World.MinAgents = 4, 10
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithObserver(rec), economysim.WithAudit(), economysim.WithLogger(slog.New(lg)))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	var tally censusTally
	seen := 0
	for tick := int64(0); tick < 200; tick++ {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", tick, err)
		}
		tally.absorb(rec.events[seen:])
		seen = len(rec.events)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot after tick %d error = %v, want nil", tick, err)
		}
		if msg := tally.disagrees(s); msg != "" {
			t.Fatalf("tick %d: %s", tick, msg)
		}
		households := tally.households
		if got := lastTickEndedHouseholds(lg); got != households {
			t.Fatalf("tick %d: the \"tick ended\" line carries %d households, want %d", tick, got, households)
		}
	}
	if tally.changes == 0 {
		t.Errorf("200 ticks at scale 4 held no WeightChanged, want at least one so that the guard exercises a change of weights (%d agents, %d households at the end)", tally.agents, tally.households)
	}
}

// censusTally rebuilds the agents and the households from the events alone
// (observability-and-logging.md 4.3): AgentSpawned and AgentBankrupted by weight,
// and WeightChanged by To less From.
type censusTally struct{ agents, households, changes int64 }

// absorb adds the events.
func (e *censusTally) absorb(events []economysim.Event) {
	for _, ev := range events {
		switch ev := ev.(type) {
		case economysim.AgentSpawned:
			e.agents, e.households = e.agents+1, e.households+ev.Weight
		case economysim.AgentBankrupted:
			e.agents, e.households = e.agents-1, e.households-ev.Weight
		case economysim.WeightChanged:
			e.households += ev.To - ev.From
			e.changes++
		}
	}
}

// disagrees returns a description of how the snapshot's counts differ from the
// tally, at the world, region and role level, or "" when all are equal.
func (e *censusTally) disagrees(s economysim.Snapshot) string {
	var ra, rh, ka, kh int64
	for _, r := range s.Regions {
		ra, rh = ra+r.Agents, rh+r.Households
	}
	for _, r := range s.Roles {
		ka, kh = ka+r.Agents, kh+r.Households
	}
	if s.Households == e.households && rh == e.households && kh == e.households && s.Agents == e.agents && ra == e.agents && ka == e.agents {
		return ""
	}
	return fmt.Sprintf("snapshot %d agents, %d households; regions %d and %d; roles %d and %d; events %d and %d; want all equal", s.Agents, s.Households, ra, rh, ka, kh, e.agents, e.households)
}

// lastTickEndedHouseholds returns the households attribute of the last "tick
// ended" line, or -1 when there is none.
func lastTickEndedHouseholds(lg *captureHandler) int64 {
	lines := lg.records()
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].msg == "tick ended" {
			if v, ok := lines[i].attr("households"); ok {
				return v.Int64()
			}
		}
	}
	return -1
}

// TestScalePack_ConstructionStateDoesNotDependOnTheWeights verifies that two
// worlds with the same number of agents and weights 1 and 7 have the same private
// state at tick 0: the same agent slots, ledger accounts, price beliefs, output
// residues and welfare ring slots, and the same snapshot rows and lots.
//
// Arithmetic: 200 households per role at scale 1 and 1 400 at scale 7 are both
// 200 agents per role (1 200 in all), of weight 1 and of weight 7.
//
// Requirement: FR-SCL-016, NFR-SCL-004
func TestScalePack_ConstructionStateDoesNotDependOnTheWeights(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	light := buildScaled(t, scalePacks[0].layers, 1, 1, nil)
	heavy := buildScaled(t, scalePacks[0].layers, 7, 1, func(c *economysim.Config) {
		for i := range c.Groups {
			c.Groups[i].Households *= 7
		}
	})
	if a, b := light.snap.Agents, heavy.snap.Agents; a != b || a != 1200 {
		t.Fatalf("agents at weight 1 and 7 = %d and %d, want 1 200 each", a, b)
	}
	if a, b := light.snap.Households, heavy.snap.Households; a != 1200 || b != 7*1200 {
		t.Errorf("households at weight 1 and 7 = %d and %d, want 1 200 and 8 400", a, b)
	}
	if a, b := economysim.ScaleStateOf(light.w), economysim.ScaleStateOf(heavy.w); a != b {
		t.Errorf("private state at weight 1 = %+v, at weight 7 = %+v, want equal", a, b)
	}
	if msg := snapshotShapeDiffers(light.snap, heavy.snap); msg != "" {
		t.Error(msg)
	}
}

// snapshotShapeDiffers returns a description of how the row counts and the lots of
// two snapshots differ, or "" when they are equal.
func snapshotShapeDiffers(a, b economysim.Snapshot) string {
	var lotsA, lotsB int64
	for i := range a.Markets {
		lotsA += a.Markets[i].Lots
	}
	for i := range b.Markets {
		lotsB += b.Markets[i].Lots
	}
	if lotsA == lotsB && len(a.Regions) == len(b.Regions) && len(a.Roles) == len(b.Roles) && len(a.Markets) == len(b.Markets) {
		return ""
	}
	return fmt.Sprintf("lots %d and %d, rows %d/%d/%d and %d/%d/%d, want equal", lotsA, lotsB,
		len(a.Regions), len(a.Roles), len(a.Markets), len(b.Regions), len(b.Roles), len(b.Markets))
}
