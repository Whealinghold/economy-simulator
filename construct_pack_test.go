package economysim_test

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// constructionPacks are the packs without a warm start that the slice builds
// (plan-phase-1-11.md Phase 3: `one`, `second`, and `two`, which is built and
// audited here). The warm layers are the tests of the application of the warm
// start.
var constructionPacks = []struct {
	name   string
	layers []string
}{
	{"one", []string{"base.toml", "one.toml"}},
	{"second", []string{"second/second.toml"}},
	{"two", []string{"base.toml", "two.toml"}},
}

// wantAgents is the number of agents of a group: ceil(h / s), raised to
// min(k, h) (spec-scale.md 3.2), written out here without the function under
// test.
func wantAgents(h, s, k int64) int64 {
	n := (h + s - 1) / s
	return max(n, min(k, h))
}

// TestNewWorld_IDsAccountsAndEndowment verifies, over the packs one, second and
// two and with no code specific to any, the layout of spec-world.md 5.2 and 5.3
// and the endowment of 5.4.2: the boundary accounts come first (1 + 7 * C, so 64
// for the nine commodities of `one`), then the estate account of each region,
// then one account per household agent in ascending ID; the agent IDs are the
// estates 0 to R-1 and then the households in (region, role, index) order; each
// group has ceil(H / S) agents whose weights sum to H and differ by at most 1;
// each agent holds weight times StartMoney and weight times each StartStock
// item and nothing else; its beliefs start at the width BeliefWidthInit and are
// the same for every agent of the role (no randomness); its target ratio lies in
// the drawn range and its residues are one value in [0, 10 000); every market
// starts at the reference price with age 0, no volume and an empty window; the
// audit passes at tick 0, also when WithAudit runs it at construction.
//
// Requirement: FR-WLD-030, FR-WLD-031, FR-WLD-032, FR-WLD-033, FR-WLD-034, FR-WLD-035, FR-AGT-001, FR-LED-052, FR-SCL-002, FR-SCL-003, FR-CLR-020, FR-BEL-022
func TestNewWorld_IDsAccountsAndEndowment(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range constructionPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			raw := loadPack(t, pk.layers...)
			cfg := economysim.Canonical(raw)
			w, err := economysim.NewWorld(raw, economysim.WithAudit())
			if err != nil {
				t.Fatalf("NewWorld(%s, WithAudit) error = %v, want nil", pk.name, err)
			}
			checkAccounts(t, w, &cfg)
			rows := economysim.ConstructionRows(w)
			checkGroups(t, rows, &cfg)
			checkEndowments(t, rows, &cfg)
			checkMarkets(t, w, &cfg)
			if err := w.Audit(); err != nil {
				t.Errorf("Audit() at tick 0 = %v, want nil", err)
			}
			if w.Tick() != 0 {
				t.Errorf("Tick() = %d after construction, want 0", w.Tick())
			}
			var households int64
			for _, r := range rows {
				households += r.Weight
			}
			if a, h := economysim.WorldCounts(w); a != int64(len(rows)) || h != households {
				t.Errorf("counters = (%d agents, %d households), want (%d, %d)", a, h, len(rows), households)
			}
		})
	}
}

// checkAccounts checks the account layout of spec-world.md 5.2: 1 + 7 * C
// boundary accounts, the estate of region r at the account after them plus r,
// the agents after the estates, one account each, in ascending ID.
func checkAccounts(t *testing.T, w *economysim.World, cfg *economysim.Config) {
	t.Helper()
	p := economysim.ConstructionAccounts(w)
	boundary := 1 + 7*len(cfg.Commodities) // the initial money account, and seven goods reasons per commodity
	if p.Boundary != boundary {
		t.Errorf("boundary accounts = %d, want 1 + 7 * %d = %d", p.Boundary, len(cfg.Commodities), boundary)
	}
	for r, acct := range p.Estates {
		if acct != boundary+r {
			t.Errorf("estate account of region %d = %d, want %d", r, acct, boundary+r)
		}
	}
	rows := economysim.ConstructionRows(w)
	first := boundary + len(cfg.Regions)
	for i, r := range rows {
		if r.Account != first+i {
			t.Errorf("agent %d account = %d, want %d: the accounts follow the estates in ascending agent ID", r.ID, r.Account, first+i)
			break
		}
	}
	if p.Total != first+len(rows) {
		t.Errorf("accounts = %d, want %d boundary + %d estates + %d agents", p.Total, boundary, len(cfg.Regions), len(rows))
	}
}

// checkGroups checks the agent ID space and the weights of each group, in the
// canonical order of the groups.
func checkGroups(t *testing.T, rows []economysim.AgentRow, cfg *economysim.Config) {
	t.Helper()
	next := 0
	for _, g := range cfg.Groups {
		scale := g.Scale
		if scale == 0 {
			scale = cfg.World.Scale
		}
		n := int(wantAgents(g.Households, scale, cfg.World.MinAgents))
		if next+n > len(rows) {
			t.Fatalf("group %s/%s: %d agents expected from row %d, but there are %d rows", g.Region, g.Role, n, next, len(rows))
		}
		var sum int64
		lo, hi := int64(1<<62), int64(0)
		for i := range n {
			r := rows[next+i]
			if wantID := economysim.AgentID(len(cfg.Regions) + next + i); r.ID != wantID {
				t.Errorf("group %s/%s agent %d has ID %d, want %d", g.Region, g.Role, i, r.ID, wantID)
			}
			if cfg.Regions[r.Region].ID != g.Region || cfg.Roles[r.Role].ID != g.Role {
				t.Errorf("agent %d is (%s, %s), want the group (%s, %s): groups come in (region, role) order",
					r.ID, cfg.Regions[r.Region].ID, cfg.Roles[r.Role].ID, g.Region, g.Role)
			}
			sum += r.Weight
			lo, hi = min(lo, r.Weight), max(hi, r.Weight)
		}
		// Weights sum to H and differ by at most 1: H = 136 over 136 agents of 1,
		// or H = 253 at scale 1 over 253 agents.
		if sum != g.Households || hi-lo > 1 || lo < 1 {
			t.Errorf("group %s/%s weights sum %d (min %d, max %d), want %d, a spread of at most 1 and a minimum of 1",
				g.Region, g.Role, sum, lo, hi, g.Households)
		}
		next += n
	}
	if next != len(rows) {
		t.Errorf("groups account for %d agents, the world has %d", next, len(rows))
	}
}

// checkEndowments checks the money, goods, strategy, beliefs, target and
// residues of each agent against its role.
func checkEndowments(t *testing.T, rows []economysim.AgentRow, cfg *economysim.Config) {
	t.Helper()
	byRole := make([][]economysim.Belief, len(cfg.Roles)) // the beliefs of the first agent of each role
	for _, r := range rows {
		role := &cfg.Roles[r.Role]
		wantStock := make([]int64, len(cfg.Commodities))
		for _, it := range role.StartStock {
			for c := range cfg.Commodities {
				if cfg.Commodities[c].ID == it.Commodity {
					wantStock[c] = r.Weight * int64(it.Quantity)
				}
			}
		}
		if want := r.Weight * int64(role.StartMoney); r.Money != want {
			t.Errorf("agent %d money = %d, want weight %d * StartMoney %d = %d", r.ID, r.Money, r.Weight, role.StartMoney, want)
		}
		if !slices.Equal(r.Stock, wantStock) {
			t.Errorf("agent %d stock = %v, want %v: weight times each StartStock item", r.ID, r.Stock, wantStock)
		}
		if r.Strategy != int(r.Role) {
			t.Errorf("agent %d strategy index = %d, want its role index %d", r.ID, r.Strategy, r.Role)
		}
		for _, b := range r.Beliefs {
			if b.Width != cfg.World.BeliefWidthInit || b.Centre < 1 {
				t.Errorf("agent %d belief = %+v, want width %d and a centre of at least 1", r.ID, b, cfg.World.BeliefWidthInit)
			}
		}
		if first := byRole[r.Role]; first != nil && !slices.Equal(first, r.Beliefs) {
			t.Errorf("agent %d beliefs = %v, want those of the first agent of its role %v: no randomness", r.ID, r.Beliefs, first)
		}
		byRole[r.Role] = r.Beliefs
		// The target ratio is drawn in [lo, hi] with lo = ratio * (10 000 - spread) / 10 000
		// and hi = ratio * (10 000 + spread) / 10 000 (spec-world.md 5.7.1).
		lo := role.TargetRatio * (10_000 - role.TargetSpread) / 10_000
		hi := role.TargetRatio * (10_000 + role.TargetSpread) / 10_000
		if r.Target < lo || r.Target > hi {
			t.Errorf("agent %d target ratio = %d, want in [%d, %d]", r.ID, r.Target, lo, hi)
		}
		for _, v := range r.Residues {
			if v != r.Residues[0] || v < 0 || v >= 10_000 {
				t.Errorf("agent %d residues = %v, want one value in [0, 10000) copied to every residue", r.ID, r.Residues)
				break
			}
		}
	}
}

// checkMarkets checks the market table: region-major, every market at the
// reference price of its commodity with age 0, no volume, an empty window and a
// run of 0 (FR-CLR-020, FR-BEL-039).
func checkMarkets(t *testing.T, w *economysim.World, cfg *economysim.Config) {
	t.Helper()
	ms := economysim.MarketRecords(w)
	if len(ms) != len(cfg.Regions)*len(cfg.Commodities) {
		t.Fatalf("markets = %d, want %d regions x %d commodities", len(ms), len(cfg.Regions), len(cfg.Commodities))
	}
	for i, m := range ms {
		want := economysim.MarketRecord{
			Region: economysim.RegionID(i / len(cfg.Commodities)), Commodity: economysim.CommodityID(i % len(cfg.Commodities)),
			Price: cfg.Commodities[i%len(cfg.Commodities)].ReferencePrice,
		}
		if m != want {
			t.Errorf("market %d = %+v, want %+v: table order, seed price, age 0, empty window", i, m, want)
		}
	}
}

// TestNewWorld_EventsAndLogsAtConstruction verifies the events and the logs of
// spec-world.md 5.8 over the pack `one`: with an observer the first event is
// WorldCreated (Tick 0, the seed, the whole hash, no warm start, the names table
// of the configuration), then one AgentSpawned for each household agent in
// ascending ID with Tick 0, its weight and its money, and nothing else (no
// BoundaryFlow: the construction bucket comes at the first step); the Info line
// "world created" has the keys of spec-observability.md 7.2 with the counts of
// the configuration; the pack has no warm start, so there is no "warm start
// applied" line; and every configuration warning is one Warn line "configuration
// warning" with its rule and path, before the Info line. No line is written at
// Debug.
//
// Requirement: FR-WLD-037, FR-OBS-010, FR-OBS-076
func TestNewWorld_EventsAndLogsAtConstruction(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	raw := loadPack(t, "base.toml", "one.toml")
	cfg := economysim.Canonical(raw)
	rec := &recorder{}
	h := newCaptureHandler()
	w, err := economysim.NewWorld(raw, economysim.WithSeed(7), economysim.WithObserver(rec), economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	rows := economysim.ConstructionRows(w)
	if len(rec.events) != 1+len(rows) {
		t.Fatalf("events = %d, want WorldCreated and %d AgentSpawned", len(rec.events), len(rows))
	}
	wc, ok := rec.events[0].(economysim.WorldCreated)
	if !ok {
		t.Fatalf("first event is %T, want WorldCreated", rec.events[0])
	}
	wantNames := economysim.Names{
		Regions: idsOf(len(cfg.Regions), func(i int) string { return cfg.Regions[i].ID }), Commodities: idsOf(len(cfg.Commodities), func(i int) string { return cfg.Commodities[i].ID }),
		Roles:   idsOf(len(cfg.Roles), func(i int) string { return cfg.Roles[i].ID }),
		Recipes: idsOf(len(cfg.Recipes), func(i int) string { return cfg.Recipes[i].ID }),
		Seasons: idsOf(len(cfg.Calendar.Seasons), func(i int) string { return cfg.Calendar.Seasons[i].ID }),
	}
	if wc.Tick != 0 || wc.Seed != 7 || wc.ConfigHash != w.ConfigHash() || wc.WarmStart || !reflect.DeepEqual(wc.Names, wantNames) {
		t.Errorf("WorldCreated = %+v, want Tick 0, seed 7, the world's hash, no warm start and names %+v", wc, wantNames)
	}
	for i, e := range rec.events[1:] {
		got, ok := e.(economysim.AgentSpawned)
		want := economysim.AgentSpawned{Agent: rows[i].ID, Region: rows[i].Region, Role: rows[i].Role, Weight: rows[i].Weight, StartMoney: economysim.Money(rows[i].Money)}
		if !ok || got != want {
			t.Fatalf("event %d = %+v, want %+v (agents in ascending ID, Tick 0)", i+1, e, want)
		}
	}

	lines := h.records()
	warnings := cfg.Warnings()
	if len(lines) != len(warnings)+1 {
		t.Fatalf("construction wrote %d lines, want %d warnings and the Info line: %v", len(lines), len(warnings), lines)
	}
	for i, wn := range warnings {
		l := lines[i]
		rule, _ := l.attr("rule")
		path, _ := l.attr("path")
		if l.level != slog.LevelWarn || l.msg != "configuration warning" || rule.String() != wn.Rule || path.String() != strings.Join(wn.Path, ".") {
			t.Errorf("line %d = %v, want a Warn \"configuration warning\" with rule %q and path %q", i, l, wn.Rule, strings.Join(wn.Path, "."))
		}
	}
	checkWorldCreatedLine(t, lines[len(lines)-1], &cfg, w, rows)
}

// idsOf collects n strings by index.
func idsOf(n int, f func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = f(i)
	}
	return out
}

// checkWorldCreatedLine checks the Info line "world created" against the
// configuration: the keys, the counts, the modules and the hash.
func checkWorldCreatedLine(t *testing.T, l logLine, cfg *economysim.Config, w *economysim.World, rows []economysim.AgentRow) {
	t.Helper()
	var households int64
	for _, r := range rows {
		households += r.Weight
	}
	hash := w.ConfigHash()
	want := []struct {
		key string
		got func(slog.Value) bool
	}{
		{"seed", func(v slog.Value) bool { return v.Uint64() == 7 }},
		{"config_hash", func(v slog.Value) bool { return v.String() == fmt.Sprintf("%x", hash[:8]) }},
		{"commodities", func(v slog.Value) bool { return v.Int64() == int64(len(cfg.Commodities)) }},
		{"recipes", func(v slog.Value) bool { return v.Int64() == int64(len(cfg.Recipes)) }},
		{"roles", func(v slog.Value) bool { return v.Int64() == int64(len(cfg.Roles)) }},
		{"regions", func(v slog.Value) bool { return v.Int64() == int64(len(cfg.Regions)) }},
		{"agents", func(v slog.Value) bool { return v.Int64() == int64(len(rows)) }},
		{"households", func(v slog.Value) bool { return v.Int64() == households }},
		{"modules", func(v slog.Value) bool { m, ok := v.Any().([]string); return ok && len(m) == 0 }},
		{"warm_start", func(v slog.Value) bool { return !v.Bool() }},
	}
	if l.level != slog.LevelInfo || l.msg != "world created" || len(l.attrs) != len(want) {
		t.Fatalf("last line = %v, want an Info \"world created\" with %d keys", l, len(want))
	}
	for i, k := range want {
		if l.attrs[i].Key != k.key || !k.got(l.attrs[i].Value) {
			t.Errorf("world created attribute %d = %v, want the key %q with the value of the configuration", i, l.attrs[i], k.key)
		}
	}
}

// TestNewWorld_EventNamesAreOwned verifies that the names table of WorldCreated
// is a copy owned by the event: two worlds of one configuration give two events
// that share no slice, and changing one does not change the world's own
// configuration (FR-OBS-002).
//
// Requirement: FR-OBS-002
func TestNewWorld_EventNamesAreOwned(t *testing.T) {
	t.Parallel()
	var events [2]*recorder
	var worlds [2]*economysim.World
	for i := range events {
		events[i] = &recorder{}
		worlds[i] = shellWorld(t, economysim.WithObserver(events[i]))
	}
	first := events[0].events[0].(economysim.WorldCreated)
	first.Names.Regions[0] = "changed"
	second := events[1].events[0].(economysim.WorldCreated)
	if second.Names.Regions[0] != "home" || worlds[1].Config().Regions[0].ID != "home" {
		t.Errorf("names of the second world = %v after changing the first's, want [home]: each event owns its slices", second.Names.Regions)
	}
}
