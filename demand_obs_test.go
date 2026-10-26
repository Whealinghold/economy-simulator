package economysim_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// obsWorld builds a world of the reference chain with the demand module on, a
// recorder and a capturing logger, and returns the role of the cook (whose ladder
// is comfort, then treat).
func obsWorld(t *testing.T) (*economysim.World, *recorder, *captureHandler, economysim.RoleID) {
	t.Helper()
	return obsWorldOf(t, demandConfig())
}

// obsWorldOf is obsWorld for the configuration given.
func obsWorldOf(t *testing.T, cfg economysim.Config) (*economysim.World, *recorder, *captureHandler, economysim.RoleID) {
	t.Helper()
	rec, lg := &recorder{}, newCaptureHandler()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithObserver(rec), economysim.WithLogger(slog.New(lg)))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	rec.forget()
	lg.forget()
	for i, r := range cfg.Roles {
		if r.ID == "cook" {
			return w, rec, lg, economysim.RoleID(i)
		}
	}
	t.Fatal("no cook role")
	return nil, nil, nil, 0
}

// shortfalls returns the WantShortfall events of evs.
func shortfalls(evs []economysim.Event) []economysim.WantShortfall {
	var out []economysim.WantShortfall
	for _, e := range evs {
		if s, ok := e.(economysim.WantShortfall); ok {
			out = append(out, s)
		}
	}
	return out
}

// TestWantShortfall_EmittedOnlyWhenShort verifies that WantShortfall is delivered
// for a (region, role, want) cell if and only if at least one household was
// choked or the funded units are below the planned ones: never for a fully
// funded cell, never when no observer was supplied, and in a real run every event
// is a short one.
//
// Requirement: FR-DEM-035
//
// Guard: an event for every cell every tick would cost a healthy world and would
// hide the ones that matter.
func TestWantShortfall_EmittedOnlyWhenShort(t *testing.T) {
	t.Parallel()
	w, rec, _, cook := obsWorld(t)
	const comfort, treat = 0, 1
	economysim.SetWantTally(w, 0, cook, comfort, 10, 0, 7, 7) // funded in full
	economysim.SetWantTally(w, 0, cook, treat, 10, 0, 0, 0)   // nothing planned, nothing short
	economysim.P36WantsForTest(w)
	if got := shortfalls(rec.events); len(got) != 0 {
		t.Errorf("fully funded cells delivered %+v, want none", got)
	}
	economysim.SetWantTally(w, 0, cook, comfort, 10, 0, 7, 6) // unfunded by one unit
	economysim.SetWantTally(w, 0, cook, treat, 10, 3, 0, 0)   // choked, no unit planned
	economysim.P36WantsForTest(w)
	want := []economysim.WantShortfall{
		{Region: 0, Role: cook, Want: comfort, Households: 10, Planned: 7, Funded: 6},
		{Region: 0, Role: cook, Want: treat, Households: 10, Choked: 3},
	}
	if got := shortfalls(rec.events); !reflect.DeepEqual(got, want) {
		t.Errorf("short cells delivered %+v, want %+v", got, want)
	}

	// A run of the world: every event is short, and a world without an observer
	// reaches the same state.
	rec.forget()
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	evs := shortfalls(rec.events)
	if len(evs) == 0 {
		t.Fatalf("30 ticks of the demand world delivered no WantShortfall, want some")
	}
	for _, s := range evs {
		if s.Households < 1 || (s.Choked < 1 && s.Funded >= s.Planned) || s.Funded > s.Planned || s.Choked > s.Households {
			t.Errorf("event %+v is not a short cell", s)
		}
	}
}

// TestWantShortfall_OrderAndNoMaps verifies that the events of one tick come in
// ascending region, role and want, whatever order the cells were filled in, and
// that the event type holds no map, slice, pointer, function or channel.
//
// Requirement: FR-DEM-035
//
// Guard: a map walked to build the event would give an order that changes between
// runs of one seed (architecture.md 7.6).
func TestWantShortfall_OrderAndNoMaps(t *testing.T) {
	t.Parallel()
	// Rank order is the reverse of the want ID order, so the ladder position of a
	// want is not its ID and an emission in ladder order would be caught.
	cfg := demandConfig()
	wantOf(&cfg, "comfort").Rank, wantOf(&cfg, "treat").Rank = 2, 1
	w, rec, _, cook := obsWorldOf(t, cfg)
	if l := economysim.LadderOf(w, cook); !slices.Equal(l, []economysim.WantID{1, 0}) {
		t.Fatalf("ladder of the cook = %v, want treat before comfort", l)
	}
	// Fill the cells in descending order.
	for region := economysim.RegionID(len(demandConfig().Regions) - 1); region >= 0; region-- {
		for want := economysim.WantID(1); want >= 0; want-- {
			economysim.SetWantTally(w, region, cook, want, 4, 4, 2, 0)
		}
	}
	economysim.P36WantsForTest(w)
	evs := shortfalls(rec.events)
	if len(evs) != 2*len(demandConfig().Regions) {
		t.Fatalf("delivered %d events, want %d", len(evs), 2*len(demandConfig().Regions))
	}
	if !slices.IsSortedFunc(evs, func(a, b economysim.WantShortfall) int {
		return cmpKey(a) - cmpKey(b)
	}) {
		t.Errorf("events %+v are not in region, role, want order", evs)
	}
	typ := reflect.TypeFor[economysim.WantShortfall]()
	for i := range typ.NumField() {
		switch k := typ.Field(i).Type.Kind(); k {
		case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Func, reflect.Chan, reflect.Interface:
			t.Errorf("WantShortfall.%s is a %v, want a plain value", typ.Field(i).Name, k)
		}
	}
	// The tally is closed: a second call has nothing to say.
	rec.forget()
	economysim.P36WantsForTest(w)
	if got := shortfalls(rec.events); len(got) != 0 {
		t.Errorf("a second P3.6 delivered %+v, want none", got)
	}
}

// cmpKey orders a WantShortfall by region, role and want.
func cmpKey(s economysim.WantShortfall) int {
	return (int(s.Region)*1000+int(s.Role))*1000 + int(s.Want)
}

// TestWantsChokedWarn_TenthAndEveryTenth verifies the Warn "wants choked": it is
// emitted per region, role and want when all the cell's households were choked, at
// the tenth consecutive tick and every tenth after; a tick with one household not
// choked, or none, resets the run; another cell keeps its own run.
//
// Requirement: FR-DEM-036
func TestWantsChokedWarn_TenthAndEveryTenth(t *testing.T) {
	t.Parallel()
	w, _, lg, cook := obsWorld(t)
	const comfort, treat = 0, 1
	tick := func(allChoked bool) []logLine {
		lg.forget()
		economysim.SetWantTally(w, 0, cook, comfort, 5, 5, 3, 0)
		if !allChoked {
			economysim.SetWantTally(w, 0, cook, comfort, 5, 4, 3, 0)
		}
		economysim.SetWantTally(w, 0, cook, treat, 5, 1, 3, 0) // never all choked
		economysim.P36WantsForTest(w)
		var warns []logLine
		for _, l := range lg.records() {
			if l.msg == "wants choked" {
				warns = append(warns, l)
			}
		}
		return warns
	}
	for n := 1; n <= 25; n++ {
		warns := tick(true)
		if n%10 != 0 {
			if len(warns) != 0 {
				t.Fatalf("tick %d of the run warned %d times, want none before the tenth", n, len(warns))
			}
			continue
		}
		if len(warns) != 1 || warns[0].level != slog.LevelWarn {
			t.Fatalf("tick %d of the run gave %+v, want one Warn", n, warns)
		}
		for key, want := range map[string]int64{"region": 0, "role": int64(cook), "want": comfort, "households": 5} {
			if v, ok := warns[0].attr(key); !ok || v.Int64() != want {
				t.Errorf("tick %d: attribute %s = %v (present %v), want %d", n, key, v, ok, want)
			}
		}
	}
	if got := economysim.ChokedRun(w, 0, cook, comfort); got != 25 {
		t.Errorf("run after 25 ticks = %d, want 25", got)
	}
	if got := economysim.ChokedRun(w, 0, cook, treat); got != 0 {
		t.Errorf("run of the cell that was never all choked = %d, want 0", got)
	}
	if warns := tick(false); len(warns) != 0 || economysim.ChokedRun(w, 0, cook, comfort) != 0 {
		t.Errorf("a tick with one household not choked gave %d warns and run %d, want a reset", len(warns), economysim.ChokedRun(w, 0, cook, comfort))
	}
	for n := 1; n <= 10; n++ {
		if warns := tick(true); (len(warns) == 1) != (n == 10) {
			t.Fatalf("tick %d after the reset gave %d warns, want one at the tenth only", n, len(warns))
		}
	}
}

// TestDemandLogs_DebugAndGuarded verifies the lines of the module: "want skipped"
// carries agent, want and a reason from the closed set; "want consumed" carries
// agent, commodity and a positive quantity; both are Debug, there is no Info and no
// Error line of the module, and a logger that leaves Debug off receives none of
// them.
//
// Requirement: FR-DEM-037
//
// Guard: an Info or Error line per want, or a Debug line built behind no Enabled
// check, would put the module on the hot path of a healthy world.
func TestDemandLogs_DebugAndGuarded(t *testing.T) {
	t.Parallel()
	w, _, lg, _ := obsWorld(t)
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	reasons := map[string]bool{"choked": true, "over_budget": true, "no_budget": true, "no_stock": true}
	var skipped, consumed, quiet int
	for _, l := range lg.records() {
		switch l.msg {
		case "want skipped":
			skipped++
			r, _ := l.attr("reason")
			if _, ok := l.attr("agent"); !ok || !reasons[r.String()] || l.level != slog.LevelDebug {
				t.Errorf("want skipped line %+v, want agent, a closed reason, Debug", l)
			}
			if _, ok := l.attr("want"); !ok {
				t.Errorf("want skipped line %+v has no want", l)
			}
		case "want consumed":
			consumed++
			q, _ := l.attr("quantity")
			_, okA := l.attr("agent")
			_, okC := l.attr("commodity")
			if !okA || !okC || q.Int64() < 1 || l.level != slog.LevelDebug {
				t.Errorf("want consumed line %+v, want agent, commodity, quantity >= 1, Debug", l)
			}
		default:
			if l.level >= slog.LevelInfo {
				quiet++
			}
		}
	}
	if skipped == 0 || consumed == 0 {
		t.Errorf("30 ticks gave %d skipped and %d consumed lines, want some of each", skipped, consumed)
	}
	h := &levelHandler{min: slog.LevelInfo}
	w2, err := economysim.NewWorld(demandConfig(), economysim.WithSeed(1), economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	before := h.handled
	for range 30 {
		if err := w2.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	if h.handled-before != quiet {
		t.Errorf("a logger at Info received %d lines in 30 ticks, the Info and above lines of the capture are %d", h.handled-before, quiet)
	}
}

// TestWantShortReason verifies the definitions of the three reasons of a planned
// want that was funded short.
//
// Requirement: FR-DEM-037
func TestWantShortReason(t *testing.T) {
	t.Parallel()
	bid := economysim.Intent{Side: economysim.Bid, Commodity: 3, WantUnits: 2}
	cases := []struct {
		name    string
		intents []economysim.Intent
		funded  int64
		want    economysim.Reason
	}{
		{"part funded", []economysim.Intent{bid}, 1, economysim.ReasonOverBudget},
		{"bid placed, none funded", []economysim.Intent{bid}, 0, economysim.ReasonNoBudget},
		{"no bid at all", nil, 0, economysim.ReasonNoStock},
		{"bid for another commodity", []economysim.Intent{{Side: economysim.Bid, Commodity: 4, WantUnits: 2}}, 0, economysim.ReasonNoStock},
		{"bid carries no want units", []economysim.Intent{{Side: economysim.Bid, Commodity: 3}}, 0, economysim.ReasonNoStock},
		{"an ask is no bid", []economysim.Intent{{Side: economysim.Ask, Commodity: 3, WantUnits: 2}}, 0, economysim.ReasonNoStock},
	}
	for _, c := range cases {
		if got := economysim.WantShortReason(c.intents, []economysim.CommodityID{3}, c.funded); got != c.want {
			t.Errorf("%s: reason = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestDemand_ObserverAndLoggerDoNotChangeResults verifies that a world with an
// observer, a logger at Debug, both, or neither gives the same checkpoint bytes
// after 40 ticks.
//
// Requirement: FR-DEM-038
func TestDemand_ObserverAndLoggerDoNotChangeResults(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	opts := map[string]func() []economysim.Option{
		"bare":     func() []economysim.Option { return nil },
		"observer": func() []economysim.Option { return []economysim.Option{economysim.WithObserver(&recorder{})} },
		"logger": func() []economysim.Option {
			return []economysim.Option{economysim.WithLogger(slog.New(newCaptureHandler()))}
		},
		"both": func() []economysim.Option {
			return []economysim.Option{economysim.WithObserver(&recorder{}), economysim.WithLogger(slog.New(newCaptureHandler()))}
		},
	}
	var first []byte
	for _, name := range []string{"bare", "observer", "logger", "both"} {
		w, err := economysim.NewWorld(demandConfig(), append([]economysim.Option{economysim.WithSeed(1)}, opts[name]()...)...)
		if err != nil {
			t.Fatalf("%s: NewWorld error = %v, want nil", name, err)
		}
		for range 40 {
			if err := w.Step(); err != nil {
				t.Fatalf("%s: Step error = %v, want nil", name, err)
			}
		}
		b := mustMarshal(t, mustCheckpoint(t, w))
		if first == nil {
			first = b
		} else if !bytes.Equal(first, b) {
			t.Errorf("%s: the checkpoint after 40 ticks differs from the bare world's (%d and %d bytes)", name, len(b), len(first))
		}
	}
}

// demandAgentOf returns a live agent of the role, or -1.
func demandAgentOf(w *economysim.World, role economysim.RoleID) economysim.AgentID {
	for id := range economysim.AgentCount(w) {
		if r, weight := economysim.AgentRole(w, economysim.AgentID(id)); weight >= 1 && r == role {
			return economysim.AgentID(id)
		}
	}
	return -1
}

// TestFork_Demand verifies the fork-at-N of a world with the demand module on: at
// N in {0, inside a job, across a season change, just after one}, after a call
// between ticks that changed an agent's role and set a choke run, the original
// and two restores give the same events, snapshots, audit and checkpoint bytes,
// so the carries and the runs of tag 12 are all that the module needs.
//
// Requirement: FR-DEM-034
//
// Guard: a carry or a run missing from the checkpoint would make the restored
// world want a different number of units, one tick later.
func TestFork_Demand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	role := func(id string) economysim.RoleID {
		for i, r := range cfg.Roles {
			if r.ID == id {
				return economysim.RoleID(i)
			}
		}
		t.Fatalf("no role %q", id)
		return 0
	}
	cook, grower := role("cook"), role("grower")
	for _, c := range append(slices.Clone(forkCells), struct {
		name string
		n    int
	}{"after a role change", 20}) {
		t.Run(fmt.Sprintf("%s N %d", c.name, c.n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "one-demand", Cfg: cfg, Seed: 1, N: c.n, M: 15, Logs: true,
				Before: func(w *economysim.World) {
					if c.name != "after a role change" {
						return
					}
					if id := demandAgentOf(w, cook); id >= 0 {
						economysim.ChangeRoleForTest(w, id, grower)
					}
					// A run inside a choke: the Warn of the tenth tick must come from the
					// same tick in the restored world.
					economysim.SetChokedRun(w, 0, cook, economysim.LadderOf(w, cook)[0], int64(c.n))
				},
			})
		})
	}
}

// overclaim is the default strategy that then claims more than it was given: each
// bid's WantUnits is raised by 1000, each funded row is doubled and a row is added
// for a want outside every ladder.
type overclaim struct{ recStrategy }

func (o *overclaim) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	o.recStrategy.Trade(v, rng, p)
	for i := range p.Intents {
		if p.Intents[i].Side == economysim.Bid {
			p.Intents[i].WantUnits += 1000
		}
	}
	for i := range p.WantFunded {
		p.WantFunded[i].Units *= 2
	}
	p.WantFunded = append(p.WantFunded, economysim.WantFunding{Want: 99, Units: 7})
}

// TestDemandLogs_StrategyOverclaimIsLogged verifies that a strategy that sets
// WantUnits above what the world accepts, or reports funded units above the
// planned ones or for a want outside the ladder, gets a Debug line "want units
// cut" (agent, commodity, quantity cut) or "want funding cut" (agent, want,
// quantity not counted), so the gap between what it placed and WantShortfall
// is explained.
//
// Guard: the clamps are otherwise silent; the sibling clamps of the plan
// ("intent normalised", "discard refused") are each a Debug line.
func TestDemandLogs_StrategyOverclaimIsLogged(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lg := newCaptureHandler()
	cfg := demandConfig()
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "overclaim"
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithLogger(slog.New(lg)),
		economysim.WithStrategy("overclaim", &overclaim{recStrategy{wants: true}}))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	step(t, w, 10)
	var units, funding, bogus int
	for _, l := range lg.records() {
		q, _ := l.attr("quantity")
		switch l.msg {
		case "want units cut":
			units++
			if _, ok := l.attr("commodity"); !ok || l.level != slog.LevelDebug || q.Int64() < 1 {
				t.Errorf("want units cut line %+v, want commodity, quantity >= 1, Debug", l)
			}
		case "want funding cut":
			funding++
			wt, ok := l.attr("want")
			if !ok || l.level != slog.LevelDebug || q.Int64() < 1 {
				t.Errorf("want funding cut line %+v, want want, quantity >= 1, Debug", l)
			}
			if wt.Int64() == 99 {
				bogus++
			}
		}
	}
	if units == 0 || funding == 0 || bogus == 0 {
		t.Errorf("10 ticks gave %d want units cut, %d want funding cut (%d for the want outside the ladder), want some of each", units, funding, bogus)
	}
}
