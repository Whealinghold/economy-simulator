package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of what a consumer reads and calls of the risk module through the public
// API of a world built from the shipped file layers (pack two, its trade and transport
// overlays and overlay/risk.toml), with no step taken: the expected loss of a route and
// the two between-ticks scale calls. The internal tests of risk_dispatch_internal_test.go
// and risk_scale_internal_test.go check the same rules against a hand-built world and the
// unexported state; these show that the rules hold for what a consumer holds.

// riskPublicWorld builds the world of pack two with the risk overlay, or of the same
// layers with the module taken out when off is true, and returns it with its configuration.
func riskPublicWorld(t testing.TB, off bool) (*economysim.World, economysim.Config) {
	t.Helper()
	cfg := loadValidPack(t, packTwoRisk...)
	if off {
		cfg.Modules.Risk = nil
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(risk off = %v) error = %v, want nil", off, err)
	}
	return w, cfg
}

// routeLossRows are the four route rows of overlay/risk.toml with the expected loss of
// each over its single edge at the scale 10 000: with one edge the expected loss is the
// probability times the severity, rhe(strike * severity / 10 000) (spec-risk.md 8.1.1).
var routeLossRows = []struct {
	name         string
	from, to     string
	mode         string
	strike, sev  int64 // the row of overlay/risk.toml
	wantLossBP   int64
	scaledTo5000 int64 // the loss at a scale of 5 000: the probability is halved first
}{
	// 4 000 * 8 000 / 10 000 = 3 200; halved, P = 2 000 and 2 000 * 8 000 / 10 000 = 1 600.
	{"foot north to south, a unit row", "north", "south", "foot", 4_000, 8_000, 3_200, 1_600},
	// 3 000 * 8 000 / 10 000 = 2 400; halved, P = 1 500 and 1 500 * 8 000 / 10 000 = 1 200.
	{"foot south to north, the common row", "south", "north", "foot", 3_000, 8_000, 2_400, 1_200},
	// 1 000 * 5 000 / 10 000 = 500; halved, P = 500 and 500 * 5 000 / 10 000 = 250.
	{"coach north to south", "north", "south", "coach", 1_000, 5_000, 500, 250},
	{"coach south to north", "south", "north", "coach", 1_000, 5_000, 500, 250},
}

// TestRisk_RouteLossThroughThePublicAPI verifies that World.RouteLoss answers, for a
// world built from the files, the expected loss of each configured route from the rows
// (and for the common row, the same figure as for a unit row: the expected loss does not
// tell them apart), and that it answers 0 with ok true for the same routes when the
// module is off.
//
// Requirement: FR-RSK-024, FR-RSK-001
func TestRisk_RouteLossThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, off := range []bool{false, true} {
		w, cfg := riskPublicWorld(t, off)
		for _, tt := range routeLossRows {
			from, to, mode := regionID(t, cfg, tt.from), regionID(t, cfg, tt.to), modeIDOf(t, cfg, tt.mode)
			want := tt.wantLossBP
			if off {
				want = 0
			}
			if got, ok := w.RouteLoss(from, to, mode); !ok || got != want {
				t.Errorf("RouteLoss(%s) with the module off = %v: (%d, %v), want (%d, true)", tt.name, off, got, ok, want)
			}
		}
	}
}

// TestRisk_RouteLossFollowsScales verifies that the expected loss a consumer reads follows
// the scale calls it makes between ticks: a scale of 5 000 halves the probability and
// lowers the loss, a scale of 0 takes it to 0 and a scale of 10 000 clears the key and
// gives the configured loss back. The table is rebuilt at each call, so no step is taken
// in between.
//
// Requirement: FR-RSK-024, FR-RSK-025, FR-RSK-028, FR-RSK-029
func TestRisk_RouteLossFollowsScales(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, cfg := riskPublicWorld(t, false)
	for _, tt := range routeLossRows {
		from, to, mode := regionID(t, cfg, tt.from), regionID(t, cfg, tt.to), modeIDOf(t, cfg, tt.mode)
		scale := func(bp int64) {
			t.Helper()
			if err := w.SetRouteRiskScales([]economysim.RouteRiskScale{{From: from, To: to, Mode: mode, BP: bp}}); err != nil {
				t.Fatalf("SetRouteRiskScales(%s, %d) error = %v, want nil", tt.name, bp, err)
			}
		}
		for _, step := range []struct {
			bp, want int64
		}{{5_000, tt.scaledTo5000}, {0, 0}, {10_000, tt.wantLossBP}} {
			scale(step.bp)
			if got, ok := w.RouteLoss(from, to, mode); !ok || got != step.want {
				t.Errorf("RouteLoss(%s) at a scale of %d = (%d, %v), want (%d, true)", tt.name, step.bp, got, ok, step.want)
			}
		}
	}
}

// TestRisk_RouteLossFollowsEdges verifies that a closed edge leaves the route with no path,
// (0, false), until it opens again, and that the reopened route gives its configured loss:
// the table of expected losses is rebuilt at the edge call, not only at a scale call.
//
// Requirement: FR-RSK-024, FR-RSK-025
func TestRisk_RouteLossFollowsEdges(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, cfg := riskPublicWorld(t, false)
	from, to, mode := regionID(t, cfg, "north"), regionID(t, cfg, "south"), modeIDOf(t, cfg, "foot")
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: from, To: to, Closed: true, FactorBP: 10_000}}); err != nil {
		t.Fatalf("SetEdgeStatuses(close north to south) error = %v, want nil", err)
	}
	if got, ok := w.RouteLoss(from, to, mode); ok || got != 0 {
		t.Errorf("RouteLoss(north to south) with the edge closed = (%d, %v), want (0, false)", got, ok)
	}
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: from, To: to, Closed: false, FactorBP: 10_000}}); err != nil {
		t.Fatalf("SetEdgeStatuses(open north to south) error = %v, want nil", err)
	}
	if got, ok := w.RouteLoss(from, to, mode); !ok || got != 3_200 {
		t.Errorf("RouteLoss(north to south) with the edge open again = (%d, %v), want (3200, true)", got, ok)
	}
}

// TestRisk_ScaleCallsRefuseWithTheirSentinels verifies, through the public calls and
// errors.Is, that SetRouteRiskScales and SetStoreRiskScales refuse a call the module
// cannot take with the sentinel the doc comment names and that a refused call changes
// nothing: a call whose second entry is bad leaves the first entry's route at its
// configured loss. A key with no row is ErrUnknownRiskRow, a scale above MaxShockBP is
// ErrOutOfBounds, a key given twice is ErrDuplicateID, and a world with the module off
// answers ErrModuleOff.
//
// Requirement: FR-RSK-028, FR-RSK-029
func TestRisk_ScaleCallsRefuseWithTheirSentinels(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, cfg := riskPublicWorld(t, false)
	north, south := regionID(t, cfg, "north"), regionID(t, cfg, "south")
	foot, coach := modeIDOf(t, cfg, "foot"), modeIDOf(t, cfg, "coach")
	good := economysim.RouteRiskScale{From: north, To: south, Mode: foot, BP: 5_000}
	tests := []struct {
		name   string
		scales []economysim.RouteRiskScale
		want   error
	}{
		{"a key with no row", []economysim.RouteRiskScale{good, {From: north, To: north, Mode: coach, BP: 5_000}}, economysim.ErrUnknownRiskRow},
		{"a scale above the bound", []economysim.RouteRiskScale{good, {From: south, To: north, Mode: foot, BP: economysim.MaxShockBP + 1}}, economysim.ErrOutOfBounds},
		{"a negative scale", []economysim.RouteRiskScale{good, {From: south, To: north, Mode: foot, BP: -1}}, economysim.ErrOutOfBounds},
		{"a key given twice", []economysim.RouteRiskScale{good, good}, economysim.ErrDuplicateID},
		{"an unknown mode", []economysim.RouteRiskScale{good, {From: south, To: north, Mode: economysim.ModeID(99), BP: 5_000}}, economysim.ErrUnknownMode},
		{"an unknown region", []economysim.RouteRiskScale{good, {From: economysim.RegionID(99), To: north, Mode: foot, BP: 5_000}}, economysim.ErrUnknownRegion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := w.SetRouteRiskScales(tt.scales); !errors.Is(err, tt.want) {
				t.Errorf("SetRouteRiskScales(%+v) error = %v, want %v", tt.scales, err, tt.want)
			}
			if got, ok := w.RouteLoss(north, south, foot); !ok || got != 3_200 {
				t.Errorf("RouteLoss(north to south) after the refused call = (%d, %v), want (3200, true): a refused call changes nothing", got, ok)
			}
		})
	}
	off, offCfg := riskPublicWorld(t, true)
	routeCall := off.SetRouteRiskScales([]economysim.RouteRiskScale{{From: regionID(t, offCfg, "north"), To: regionID(t, offCfg, "south"), Mode: foot, BP: 5_000}})
	if !errors.Is(routeCall, economysim.ErrModuleOff) {
		t.Errorf("SetRouteRiskScales with the module off error = %v, want %v", routeCall, economysim.ErrModuleOff)
	}
	if storeCall := off.SetStoreRiskScales(nil); !errors.Is(storeCall, economysim.ErrModuleOff) {
		t.Errorf("SetStoreRiskScales with the module off error = %v, want %v", storeCall, economysim.ErrModuleOff)
	}
}
