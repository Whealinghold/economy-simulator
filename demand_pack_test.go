package economysim_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// demandWarningRules are the seven warnings of spec-demand.md 10.2.
var demandWarningRules = []string{"want_unused", "want_unsourced", "want_inelastic", "wants_exceed_spend_cap",
	"target_above_demand", "class_without_income", "region_taste_without_taste"}

// TestPacks_OneDemand verifies that the pack one-demand (the layers base, demand,
// one-demand and warm-one-demand) validates, builds a world with the module on
// that audits clean over a few ticks, names its wants in the snapshot, and raises
// no warning of the module but the expected target_above_demand (spec-demand.md
// 7.1: the ceiling of the closed economy there is about 1.3 against the target
// 1.5). The warnings of the whole pack are logged, not pinned: they are the
// census of plan 8 (O7 of plan 2-5).
//
// Requirement: FR-DEM-043, FR-DEM-033, FR-REF-005
func TestPacks_OneDemand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	for _, w := range cfg.Warnings() {
		t.Logf("warning %s at %q", w.Rule, w.Path)
		if slices.Contains(demandWarningRules, w.Rule) && w.Rule != "target_above_demand" {
			t.Errorf("Warnings(one-demand) raised %s at %q, want no demand warning but target_above_demand", w.Rule, w.Path)
		}
	}
	w, err := economysim.NewWorld(cfg, economysim.WithAudit(), economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(one-demand) error = %v, want nil", err)
	}
	for i := range 10 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	if !slices.Equal(s.Modules, []string{"demand"}) || !slices.Equal(s.Names.Wants, []string{"comfort", "treat"}) {
		t.Errorf("Snapshot modules = %q, wants = %q, want [demand] and [comfort treat]", s.Modules, s.Names.Wants)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit after 10 ticks = %v, want nil", err)
	}
}

// TestNewWorld_DemandAnnouncement verifies what the demand module adds to the
// announcement of a world (spec-world.md 5.8, spec-observability.md 3.2.4): each
// warning of the configuration, the module's included, is one Warn line
// "configuration warning" with its rule and dotted path, before the Info line;
// the Info line "world created" lists "demand" among the modules; and WorldCreated
// carries the string ID of each want in the names table, in WantID order.
//
// Requirement: FR-DEM-033, FR-WLD-037, FR-OBS-010
//
// Guard: the construction summary and the names table are what an operator and an
// adapter read to tell that the module is on and which wants it holds; a line or a
// table that dropped the module would leave both blind to it.
func TestNewWorld_DemandAnnouncement(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	rec := &recorder{}
	h := newCaptureHandler()
	if _, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithObserver(rec), economysim.WithLogger(slog.New(h))); err != nil {
		t.Fatalf("NewWorld(one-demand) error = %v, want nil", err)
	}
	wc, ok := rec.events[0].(economysim.WorldCreated)
	if !ok || !slices.Equal(wc.Names.Wants, []string{"comfort", "treat"}) {
		t.Errorf("first event = %+v, want WorldCreated with wants [comfort treat]", rec.events[0])
	}
	lines, warnings := h.records(), cfg.Warnings()
	// The pack has a warm start, so one Info line "warm start applied" sits between.
	if len(lines) != len(warnings)+2 {
		t.Fatalf("construction wrote %d lines, want %d warnings, \"warm start applied\" and the Info line: %v", len(lines), len(warnings), lines)
	}
	var sawDemand bool
	for i, wn := range warnings {
		rule, _ := lines[i].attr("rule")
		path, _ := lines[i].attr("path")
		if lines[i].level != slog.LevelWarn || lines[i].msg != "configuration warning" || rule.String() != wn.Rule || path.String() != strings.Join(wn.Path, ".") {
			t.Errorf("line %d = %v, want a Warn \"configuration warning\" with rule %q and path %q", i, lines[i], wn.Rule, strings.Join(wn.Path, "."))
		}
		sawDemand = sawDemand || slices.Contains(demandWarningRules, wn.Rule)
	}
	if !sawDemand {
		t.Errorf("Warnings(one-demand) = %v, want one rule of the module (target_above_demand) to be logged", warnings)
	}
	info := lines[len(lines)-1]
	mods, _ := info.attr("modules")
	if m, ok := mods.Any().([]string); info.level != slog.LevelInfo || info.msg != "world created" || !ok || !slices.Equal(m, []string{"demand"}) {
		t.Errorf("last line = %v, want an Info \"world created\" with modules [demand]", info)
	}
}

// secondDemand is a demand module written by hand for the second setting of
// testdata/chains/second: a crater colony, with wants and tastes named by the
// colony's own commodities, roles and region. No Go in the library knows any of
// these names.
const secondDemand = `schema = 1
[modules.demand.want.spare_parts]
items = [ { commodity = "wafer" }, { commodity = "solder" } ]
quantity = 1
period = 20
rank = 1
elasticity = 20000
[modules.demand.want.rock_garden]
items = [ { commodity = "regolith" } ]
quantity = 1
period = 4
rank = 2
[modules.demand.taste.algae_tender.spare_parts]
[modules.demand.taste.driller.spare_parts]
scale = 5000
[modules.demand.taste.driller.rock_garden]
[modules.demand.region_taste.crater_rim.rock_garden]
scale = 12000
`

// TestConfigOnly_SecondSettingWithWants verifies that a demand module written for
// the second setting, with other names and no Go change, loads, validates, puts
// its items in the tradable sets of the roles that like them, orders each ladder
// by rank, builds a world and runs it with the audit on. No code branches on a
// commodity, role, region or want (architecture.md 9.1).
//
// Requirement: FR-DEM-002, FR-DEM-004, FR-DEM-006
//
// Guard: the first setting's pack names its wants "comfort" and "treat"; a rule
// that depended on those names, or on the roles of that pack, would fail here on
// a setting that has neither.
func TestConfigOnly_SecondSettingWithWants(t *testing.T) {
	t.Parallel()
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(chainsDir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return b
	}
	cfg, err := config.Parse(read("second/second.toml"), read("second/warm-second.toml"), []byte(secondDemand))
	if err != nil {
		t.Fatalf("Parse(second + wants) error = %v, want nil", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(second + wants) = %v, want nil", err)
	}
	plain := cfg
	plain.Modules.Demand = nil
	if economysim.HashConfig(cfg) == economysim.HashConfig(plain) {
		t.Errorf("HashConfig(second with wants) = HashConfig(second), want different hashes")
	}
	c := economysim.Canonical(cfg)
	ladders, on := economysim.DemandLadders(&c)
	if !on {
		t.Fatal("DemandLadders on = false, want true")
	}
	tests := []struct {
		role     string
		ladder   []economysim.WantID // rock_garden is 0 and spare_parts 1, by ID
		wafer    bool
		regolith bool
		comment  string
	}{
		{"algae_tender", []economysim.WantID{1}, true, false, "likes spare_parts only"},
		{"driller", []economysim.WantID{1, 0}, true, true, "spare_parts has rank 1, rock_garden rank 2"},
		{"fabber", nil, true, false, "no taste row; wafer is a recipe input of its own"},
		{"regolith_miner", nil, false, true, "no taste row; regolith is its output"},
	}
	table := economysim.NewRoleTable(&c)
	sets := economysim.TradableSets(&c)
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()
			id := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == tt.role })
			if id < 0 {
				t.Fatalf("no role %q in the second setting", tt.role)
			}
			if got := ladders[id]; !slices.Equal(got, tt.ladder) {
				t.Errorf("ladder of %s = %v, want %v (%s)", tt.role, got, tt.ladder, tt.comment)
			}
			if got := sets[id]["wafer"]; got != tt.wafer {
				t.Errorf("tradable set of %s has wafer = %v, want %v (%s)", tt.role, got, tt.wafer, tt.comment)
			}
			if got := sets[id]["regolith"]; got != tt.regolith {
				t.Errorf("tradable set of %s has regolith = %v, want %v (%s)", tt.role, got, tt.regolith, tt.comment)
			}
			row := economysim.RoleTableRow(&table, economysim.RoleID(id))
			if len(row.Tradable) != countTrue(sets[id]) {
				t.Errorf("role table of %s has %d tradable commodities, the validator's set has %d", tt.role, len(row.Tradable), countTrue(sets[id]))
			}
		})
	}
	w, err := economysim.NewWorld(cfg, economysim.WithAudit(), economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(second + wants) error = %v, want nil", err)
	}
	for i := range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit after 30 ticks = %v, want nil", err)
	}
}

// countTrue returns the number of true values of a set.
func countTrue(set map[string]bool) int {
	n := 0
	for _, in := range set {
		if in {
			n++
		}
	}
	return n
}
