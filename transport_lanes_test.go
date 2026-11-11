package economysim_test

import (
	"cmp"
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// transportTwo is the pack `two` with its edges and the transport overlay: one
// self-carried mode (foot) and one hired mode (coach: unit mass 1200, unit bulk
// 900, haul span 10, the recipe run_haul of 4 ticks for 4 haul units).
var transportTwo = append(slices.Clone(packLayers["two"]), "overlay/trade-two.toml", "overlay/transport-two.toml")

// withLoad returns cfg with the mass and bulk of one unit of a commodity set.
func withLoad(cfg economysim.Config, id string, mass, bulk int64) economysim.Config {
	cfg.Commodities = slices.Clone(cfg.Commodities)
	for i := range cfg.Commodities {
		if cfg.Commodities[i].ID == id {
			cfg.Commodities[i].Mass, cfg.Commodities[i].Bulk = mass, bulk
		}
	}
	return cfg
}

// TestTransportLanes verifies the lanes of `two` with the transport overlay: one
// per ordered pair of regions and commodity for the hired mode and none for the
// self-carried one, sorted, with the labour of spec-transport.md 9.4. By hand, for
// staple with mass 300 and bulk 90 on the coach: the binding fill is
// max(300/1200, 90/900) = 1/4, the edge is 100 long so s = ceil(100/10) = 10, and
// the cheapest recipe that makes haul is run_haul (4 ticks, 4 units): labour =
// ceil(10 000 * 1/4 * 10 * 4 / 4) = 25 000. The other commodities keep the default
// mass and bulk of 1: f = max(1/1200, 1/900) = 1/900 and labour = ceil(100 000 /
// 900) = 112. A commodity with mass and bulk 0 (waste here) costs 0; the transport
// module off gives no lane.
//
// Requirement: FR-TRN-041
func TestTransportLanes(t *testing.T) {
	t.Parallel()
	cfg := withLoad(withLoad(loadPack(t, transportTwo...), "staple", 300, 90), "waste", 0, 0)
	lanes, err := cfg.TransportLanes()
	if err != nil {
		t.Fatalf("TransportLanes() error = %v, want nil", err)
	}
	if want := 2 * len(cfg.Commodities); len(lanes) != want {
		t.Fatalf("len(lanes) = %d, want %d (two ordered pairs, every commodity, the hired mode only)", len(lanes), want)
	}
	sorted := slices.IsSortedFunc(lanes, func(a, b economysim.Lane) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To), cmp.Compare(a.Commodity, b.Commodity))
	})
	if !sorted {
		t.Errorf("lanes are not sorted by (From, To, Commodity): %v", lanes)
	}
	for _, ln := range lanes {
		want := int64(112)
		switch ln.Commodity {
		case "staple":
			want = 25_000
		case "waste":
			want = 0
		}
		if ln.Labour != want || ln.LossBP != 0 {
			t.Errorf("lane %+v: Labour, LossBP = %d, %d, want %d, 0", ln, ln.Labour, ln.LossBP, want)
		}
	}

	off := cfg
	off.Modules.Transport = nil
	if got, err := off.TransportLanes(); got != nil || err != nil {
		t.Errorf("transport off: TransportLanes() = %v, %v, want nil, nil", got, err)
	}
	foot := cfg
	foot.Modules.Transport = &economysim.Transport{Modes: slices.DeleteFunc(slices.Clone(cfg.Modules.Transport.Modes), func(m economysim.Mode) bool { return m.Haul != "" })}
	if got, err := foot.TransportLanes(); len(got) != 0 || err != nil {
		t.Errorf("self-carried modes only: TransportLanes() = %v, %v, want no lane, nil", got, err)
	}
}

// TestTransportLanes_Deterministic verifies that two calls return the same list.
//
// Requirement: FR-TRN-041
func TestTransportLanes_Deterministic(t *testing.T) {
	t.Parallel()
	cfg := withLoad(loadPack(t, transportTwo...), "staple", 300, 90)
	a, errA := cfg.TransportLanes()
	b, errB := cfg.TransportLanes()
	if errA != nil || errB != nil || !slices.Equal(a, b) {
		t.Errorf("TransportLanes twice = %v, %v and %v, %v, want equal lists and nil errors", a, errA, b, errB)
	}
}

// TestTransportLanes_Errors verifies the two refusals that return no partial
// list: a haul commodity no recipe makes (ErrUnproducible) and a labour that does
// not fit in int64 (ErrOverflow).
//
// Requirement: FR-TRN-041
func TestTransportLanes_Errors(t *testing.T) {
	t.Parallel()
	base := loadPack(t, transportTwo...)

	noRecipe := base
	noRecipe.Recipes = slices.DeleteFunc(slices.Clone(base.Recipes), func(r economysim.Recipe) bool { return r.ID == "run_haul" })
	overflow := withLoad(base, "staple", 1<<60, 0)
	for _, tc := range []struct {
		name string
		cfg  economysim.Config
		want error
	}{
		{"no recipe makes the haul commodity", noRecipe, economysim.ErrUnproducible},
		{"labour overflows int64", overflow, economysim.ErrOverflow},
	} {
		got, err := tc.cfg.TransportLanes()
		if !errors.Is(err, tc.want) || got != nil {
			t.Errorf("%s: TransportLanes() = %v, %v, want nil list and %v", tc.name, got, err, tc.want)
		}
	}
}

// TestTransportLanes_FeedsSolver verifies that the derived list is accepted by
// WithLanes and that the table it produces passes Validate on the same
// configuration. No world is built.
//
// Requirement: FR-TRN-041, FR-VIA-043
func TestTransportLanes_FeedsSolver(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := withLoad(loadPack(t, transportTwo...), "staple", 300, 90)
	lanes, err := cfg.TransportLanes()
	if err != nil {
		t.Fatalf("TransportLanes() error = %v, want nil", err)
	}
	st := solveLanes(t, cfg, economysim.WithLanes(lanes))
	cfg.WarmStart = &st.Table
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate(with the lane table) = %v, want nil", err)
	}
}
