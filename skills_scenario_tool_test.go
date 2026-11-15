package economysim_test

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestConfigOnly_SkillNeedsOnlyConfiguration verifies architecture.md 9.4 for the skill
// barriers: pack second, a setting that shares nothing with the reference chain, is
// given a role that trains and a recipe a trainee may run by a layer of data alone
// (overlay/second-skill.toml), and runs 400 ticks for seeds 1 to 3 with the audit; a
// course starts and completes in each. The test names no commodity, recipe or role.
//
// Requirement: FR-SKL-010, FR-SKL-028, FR-REF-031
//
// Guard: a branch on a commodity, recipe or role identifier in the traineeship would fail
// on a setting it never heard of.
func TestConfigOnly_SkillNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, append(slices.Clone(packSecond), "overlay/second-skill.toml")...)
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var started, completed int
			obs := economysim.ObserverFunc(func(e economysim.Event) {
				switch e.(type) {
				case economysim.TraineeshipStarted:
					started++
				case economysim.TraineeshipCompleted:
					completed++
				}
			})
			w := newPackWorld(t, cfg, seed, economysim.WithAudit(), economysim.WithObserver(obs))
			for i := range 400 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
				}
			}
			if started == 0 || completed == 0 {
				t.Errorf("seed %d: %d traineeships started and %d completed in 400 ticks, want both above 0", seed, started, completed)
			}
		})
	}
}

// TestConfig_NoToolPriceField verifies FR-SKL-003: no configuration struct has a
// field for the price of a tool, so a tool's price is whatever the market makes it. It
// reads every field name of the configuration types reachable from Config by reflection.
//
// Requirement: FR-SKL-003
//
// Guard: a "ToolPrice" configuration field would put a number that the market owns into
// the data and make the entry rule read it instead of the price the world publishes.
func TestConfig_NoToolPriceField(t *testing.T) {
	t.Parallel()
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		for i := range typ.NumField() {
			f := typ.Field(i)
			if n := strings.ToLower(f.Name); strings.Contains(n, "toolprice") || (strings.Contains(n, "tool") && strings.Contains(n, "price")) {
				t.Errorf("configuration field %s.%s names a tool price, want none (FR-SKL-003)", path, f.Name)
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeFor[economysim.Config](), "Config")
	if len(seen) < 10 {
		t.Fatalf("walked %d configuration types, want the whole tree", len(seen))
	}
}

// toolsOf returns, per role name, the commodities its recipes require and the commodities
// its recipes make, from the configuration alone.
func toolsOf(cfg economysim.Config) (requires, makes map[string]map[string]bool) {
	rec := map[string]economysim.Recipe{}
	for _, r := range cfg.Recipes {
		rec[r.ID] = r
	}
	requires, makes = map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, role := range cfg.Roles {
		requires[role.ID], makes[role.ID] = map[string]bool{}, map[string]bool{}
		for _, name := range role.Recipes {
			for _, tool := range rec[name].Requires {
				requires[role.ID][tool.Commodity] = true
			}
			for _, o := range rec[name].Outputs {
				makes[role.ID][o.Commodity] = true
			}
		}
	}
	return requires, makes
}

// TestStrategy_ToolHolderDoesNotResell verifies FR-SKL-007: over 200 ticks of pack one,
// warm, with the traineeship, no agent places an ask for a commodity that a recipe of its
// role requires unless its role makes it, so a tool is bound to its holder; the estate,
// which is no agent of a role, sells the tools of the dead.
//
// Requirement: FR-SKL-007
//
// Guard: the binding is the default strategy's, there is no other mechanism (spec-skill-
// barriers.md 11); a strategy that sold its tool for cash would be seen here.
func TestStrategy_ToolHolderDoesNotResell(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	requires, makes := toolsOf(cfg)
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 200, Drive: driveStep, Observe: true})
	names := res.Final.Names
	role := map[economysim.AgentID]string{}
	asks := 0
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.AgentSpawned:
			role[x.Agent] = names.Roles[x.Role]
		case economysim.RoleChanged:
			role[x.Agent] = names.Roles[x.To]
		case economysim.OrderPlaced:
			if x.Side != economysim.Ask {
				continue
			}
			asks++
			r, c := role[x.Agent], names.Commodities[x.Commodity]
			if r == "" {
				continue // an estate: no role
			}
			if requires[r][c] && !makes[r][c] {
				t.Errorf("tick %d: agent %d of role %s asked for %s, a tool its role requires and does not make (FR-SKL-007)", x.Tick, x.Agent, r, c)
			}
		}
	}
	if asks == 0 {
		t.Fatalf("no ask in 200 ticks, want the strategy to have traded")
	}
}

// TestScenario_ToolHolderDiesEstateSells verifies FR-SKL-008 and FR-SKL-009 end to end on
// pack one, warm, with the audit: over 400 ticks agents holding a tool die and the
// estate lists it (EstateOpened with lots), and the estate sells a tool afterwards
// (a TradeExecuted of a commodity that some recipe requires, with an estate as the
// seller) while the audit holds the books of goods and money exact every tick.
//
// Requirement: FR-SKL-008, FR-SKL-009
//
// Guard: the estate's sale only moves the tool between accounts, so conservation holds
// (FR-SKL-009); a sale that created or lost a unit would fail the audit of this run.
func TestScenario_ToolHolderDiesEstateSells(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	tools := map[string]bool{}
	for _, r := range cfg.Recipes {
		for _, tool := range r.Requires {
			tools[tool.Commodity] = true
		}
	}
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 400, Drive: driveStep, Observe: true, Audit: true})
	names := res.Final.Names
	listed, sold := 0, 0
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.EstateOpened:
			if x.GoodsListed > 0 {
				listed++
			}
		case economysim.TradeExecuted:
			if tools[names.Commodities[x.Commodity]] && int(x.Seller) < len(names.Regions) {
				sold++
			}
		}
	}
	if listed == 0 || sold == 0 {
		t.Errorf("estates opened with lots %d times and sold a tool %d times in 400 ticks, want both above 0", listed, sold)
	}
}

// TestDeterminism_Skills verifies the cases of TestDeterminism_SameBytes that detCases(true)
// keeps on pack one, warm, with the traineeship overlay, seeds 1 and 2, 400 ticks: a run
// split into Run calls, with a logger, an observer or the audit, ends in the snapshot,
// the state hash and the config hash of the bare Step run, and its events and rendering
// equal the reference's. The 400 ticks hold placements, completions and the removal of
// trainers, so the queue, the accumulator and the courses are under it.
//
// Requirement: FR-CFG-026, FR-OBS-023, FR-WLD-063, FR-SKL-002, FR-SKL-027, FR-SKL-028
//
// Guard: the traineeships draw no random number and read no wall clock; a draw or a
// dependence on the observer would fail here.
func TestDeterminism_Skills(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	report := solveCold(t)
	render := func(seed uint64, res result) []byte {
		return renderGolden("one", seed, detTicks, samplesOnly(res, detTicks), report, cfg.Warnings())
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			refGolden := render(seed, ref)
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
			}
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
				if spec.Observe && len(spec.Drive.chunks) == 0 {
					if d := firstLineDiff(refGolden, render(seed, got)); d != "" {
						t.Errorf("seed %d, case %q: rendering differs from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
}

// TestSkillPremium_CheckD_NoOverflow verifies that check D and the cook share of S06 read
// the largest figures without wrapping: a predicted premium at the top of int64 accepts
// itself and rejects the bottom, a household count whose sum does not fit has no share, and
// the cook share of a region of two roles is exact at the largest count that fits.
//
// Requirement: FR-SKL-024
//
// Guard: with the plain sums a predicted premium of MaxInt64 wrapped its upper edge to a
// negative number and rejected the premium it equals.
func TestSkillPremium_CheckD_NoOverflow(t *testing.T) {
	t.Parallel()
	if ok, _ := premiumCheckD(math.MaxInt64, math.MaxInt64, 0); !ok {
		t.Errorf("premiumCheckD(MaxInt64, MaxInt64) rejected, want accepted")
	}
	if ok, _ := premiumCheckD(0, math.MaxInt64, 0); ok {
		t.Errorf("premiumCheckD(0, MaxInt64) accepted, want rejected")
	}
	if ok, _ := premiumCheckD(math.MinInt64, math.MinInt64, 0); !ok {
		t.Errorf("premiumCheckD(MinInt64, MinInt64) rejected, want accepted")
	}
	rows := func(a, b int64) economysim.Snapshot {
		return economysim.Snapshot{Names: economysim.Names{Roles: []string{"cook", "grower"}},
			Roles: []economysim.RoleRow{{Role: 0, Households: a}, {Role: 1, Households: b}}}
	}
	if _, ok := cookShare(rows(math.MaxInt64, 1)); ok {
		t.Errorf("cookShare(MaxInt64 + 1 households) = ok, want no share for a count that does not fit")
	}
	if got, ok := cookShare(rows(math.MaxInt64/2, math.MaxInt64/2)); !ok || got != 5_000 {
		t.Errorf("cookShare(half of %d) = %d, %v, want 5000, true", int64(math.MaxInt64), got, ok)
	}
}

// wearOf returns the WearApplied events of a run of pack one with the traineeship over
// ticks ticks, in the order the world emitted them.
func wearOf(t testing.TB, cfg economysim.Config, ticks int64) (rows []economysim.WearApplied, names economysim.Names) {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: ticks, Drive: driveStep, Observe: true})
	for _, e := range res.Events {
		if x, ok := e.(economysim.WearApplied); ok {
			rows = append(rows, x)
		}
	}
	return rows, res.Final.Names
}

// TestScenario_ToolWearIsDeterministicAndOnlyOfTools verifies FR-SKL-001 and FR-SKL-002
// through the public events: over 150 ticks of pack one, warm, with the traineeship, the
// commodities that wear are tools some recipe requires and never an input or an output
// that no recipe requires, at least one tool wears by a positive amount, and a second
// run of the same seed reports the same wear rows in the same order, so the counter is
// a durability that nothing random moves.
//
// Requirement: FR-SKL-001, FR-SKL-002
//
// Guard: wear on a non-tool would mean an input was consumed as a tool, and a different
// row order or amount between two runs would mean a draw or a map-order dependence.
func TestScenario_ToolWearIsDeterministicAndOnlyOfTools(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	tools := map[string]bool{}
	for _, r := range cfg.Recipes {
		for _, tool := range r.Requires {
			tools[tool.Commodity] = true
		}
	}
	first, names := wearOf(t, cfg, 150)
	second, _ := wearOf(t, cfg, 150)
	if len(first) == 0 {
		t.Fatalf("no WearApplied in 150 ticks, want the tool recipes to wear their tools")
	}
	for _, row := range first {
		if c := names.Commodities[row.Commodity]; !tools[c] {
			t.Errorf("tick %d: %s wore by %d, want only commodities that a recipe requires as a tool", row.Tick, c, row.Units)
		}
		if row.Units < 1 {
			t.Errorf("tick %d: WearApplied of %d units, want at least 1", row.Tick, row.Units)
		}
	}
	if !slices.Equal(first, second) {
		t.Errorf("two runs of seed 1 reported different wear rows (%d against %d)", len(first), len(second))
	}
}
