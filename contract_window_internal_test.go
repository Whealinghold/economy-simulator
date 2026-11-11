package economysim

import (
	"errors"
	"slices"
	"testing"
)

// windowConfig is merchConfig with the broker's modes replaced by modes and every edge
// given the length dist, so the travel time of a self-carried mode of speed s is
// ceil(dist / s) (transit.Ticks at the factor 10 000). The extra modes are "cart"
// (self-carried, speed 4) and "wagon" (crew 2, which a merchant cannot carry itself).
func windowConfig(dist int64, footSpeed int64, modes ...string) Config {
	c := merchConfig()
	for i := range c.Edges {
		c.Edges[i].Distance = dist
	}
	for i := range c.Modules.Transport.Modes {
		if c.Modules.Transport.Modes[i].ID == "foot" {
			c.Modules.Transport.Modes[i].Speed = footSpeed
		}
	}
	c.Modules.Transport.Modes = append(c.Modules.Transport.Modes,
		Mode{ID: "cart", Speed: 4, UnitMass: 400, UnitBulk: 300, Crew: 1})
	i := slices.IndexFunc(c.Roles, func(r Role) bool { return r.ID == "broker" })
	c.Roles[i].Merchant.Modes = modes
	return c
}

func brokerOf(t *testing.T, c *Config) *Role {
	t.Helper()
	i := slices.IndexFunc(c.Roles, func(r Role) bool { return r.ID == "broker" })
	if i < 0 {
		t.Fatal("the configuration has no broker role")
	}
	return &c.Roles[i]
}

// TestMerchantTripTicks_Cases verifies the longest travel time of a merchant role's
// self-carried modes over the region graph: 0 without a block, with only hired modes, with
// a mode a merchant cannot carry itself (crew above 1), with no path, with a speed outside
// its bound or an invalid graph; the longest of several modes; and the cap at
// MaxTravelTicks.
//
// Arithmetic: the two regions are joined by edges of length d, a mode of speed s takes
// ceil(d / s) ticks, at least 1. foot at speed 1 over 40 takes 40; cart (speed 4) over 40
// takes ceil(40 / 4) = 10, so with both the longest is 40. A distance of MaxDistance at
// speed 1 would take 10^9 ticks and is capped at MaxTravelTicks = 100 000.
//
// Requirement: FR-WEL-009, FR-MER-003
func TestMerchantTripTicks_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(c *Config)
		cfg  Config
		want int64
	}{
		{"self-carried mode", nil, windowConfig(40, 1, "foot"), 40},
		{"rounds the time up", nil, windowConfig(41, 4, "foot"), 11},
		{"hired mode only", nil, windowConfig(40, 1, "coach"), 0},
		{"hired mode ignored beside a self-carried one", nil, windowConfig(40, 1, "coach", "foot"), 40},
		{"two modes, the longest wins", nil, windowConfig(40, 1, "cart", "foot"), 40},
		{"two modes, the other is longer", nil, windowConfig(40, 20, "cart", "foot"), 10},
		{"crew above one is not self-carried", nil, windowConfig(40, 1, "wagon"), 0},
		{"no path", func(c *Config) { c.Edges = nil }, windowConfig(40, 1, "foot"), 0},
		{"speed above its bound", nil, windowConfig(40, MaxSpeed+1, "foot"), 0},
		{"speed zero", nil, windowConfig(40, 0, "foot"), 0},
		{"graph invalid", func(c *Config) { c.Edges[0].To = "nowhere" }, windowConfig(40, 1, "foot"), 0},
		{"transport module off", func(c *Config) { c.Modules.Transport = nil }, windowConfig(40, 1, "foot"), 0},
		{"capped at the longest journey", nil, windowConfig(MaxDistance, 1, "foot"), MaxTravelTicks},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.cfg
			if tt.mut != nil {
				tt.mut(&c)
			}
			if got := merchantTripTicks(&c, newLookup(&c), brokerOf(t, &c)); got != tt.want {
				t.Errorf("merchantTripTicks(broker) = %d, want %d", got, tt.want)
			}
		})
	}
	t.Run("role without a block", func(t *testing.T) {
		t.Parallel()
		c := windowConfig(40, 1, "foot")
		if got := merchantTripTicks(&c, newLookup(&c), &c.Roles[0]); got != 0 {
			t.Errorf("merchantTripTicks(miller) = %d, want 0 (no merchant block)", got)
		}
	})
}

// TestEffectiveWindow_MerchantTrip verifies that the trip joins the derivation of the
// effective window: T = max(WelfareWindow, WindowCycles * L) with L the longest of the
// recipes' Duration and Kmax.
//
// Arithmetic: the defaults are WelfareWindow 30 and WindowCycles 2, so a trip is felt from
// Kmax above 30 / 2 = 15: Kmax 14 and 15 give max(30, 28) = max(30, 30) = 30, Kmax 16 gives
// 32. A role with no recipe has L = Kmax. A role without a block keeps 30 whatever the
// edges are.
//
// Requirement: FR-WEL-009
func TestEffectiveWindow_MerchantTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kmax int64
		want int64
	}{
		{"trip under the base window", 14, 30},
		{"trip equal to the base window", 15, 30},
		{"trip over the base window", 16, 32},
		{"long trip", 40, 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := windowConfig(tt.kmax, 1, "foot") // distance kmax at speed 1 takes kmax ticks
			if got, ok := effectiveWindow(&c, newLookup(&c), brokerOf(t, &c)); !ok || got != tt.want {
				t.Errorf("effectiveWindow(broker, Kmax %d) = %d, %v, want %d, true", tt.kmax, got, ok, tt.want)
			}
			if got, ok := effectiveWindow(&c, newLookup(&c), &c.Roles[0]); !ok || got != 30 {
				t.Errorf("effectiveWindow(miller, Kmax %d) = %d, %v, want 30, true (no block)", tt.kmax, got, ok)
			}
		})
	}
	t.Run("a longer recipe still wins", func(t *testing.T) {
		t.Parallel()
		c := windowConfig(20, 1, "foot")
		c.Recipes[0].Duration = 25 // mill, on the miller only
		b := brokerOf(t, &c)
		b.Recipes = []string{"mill"}
		// L = max(25, 20) = 25; T = max(30, 2 * 25) = 50.
		if got, ok := effectiveWindow(&c, newLookup(&c), b); !ok || got != 50 {
			t.Errorf("effectiveWindow(broker, Duration 25, Kmax 20) = %d, %v, want 50, true", got, ok)
		}
	})
}

// TestEffectiveWindow_MerchantTripOutOfBounds verifies that a trip long enough to push the
// window past MaxWelfareWindow is refused at the existing site, with ErrOutOfBounds, and
// that a hired-only broker with the same edges is not.
//
// Arithmetic: MaxWelfareWindow is 3 650, so with WindowCycles 2 a trip of 1 825 ticks
// still fits (2 * 1 825 = 3 650) and 1 826 does not (3 652).
//
// Requirement: FR-WEL-009
func TestEffectiveWindow_MerchantTripOutOfBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kmax    int64
		modes   []string
		wantErr bool
	}{
		{"trip at the bound", 1_825, []string{"foot"}, false},
		{"trip over the bound", 1_826, []string{"foot"}, true},
		{"same edges, hired only", 1_826, []string{"coach"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := windowConfig(tt.kmax, 1, tt.modes...)
			err := c.Validate()
			if got := errors.Is(err, ErrOutOfBounds); got != tt.wantErr {
				t.Errorf("Validate() with Kmax %d, modes %v = %v, want ErrOutOfBounds %v", tt.kmax, tt.modes, err, tt.wantErr)
			}
		})
	}
}

// TestMerchantTripTicks_Overflow verifies that the arithmetic is checked: the largest
// distance at the smallest speed and the largest speed over a long edge give a bounded
// result, never a wrapped one.
//
// Requirement: FR-WEL-009
//
// Guard: the window is WindowCycles times L; an uncapped Kmax near 2^63 would wrap that
// product negative and the window check would pass a role it must refuse.
func TestMerchantTripTicks_Overflow(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		dist, speed int64
	}{
		{"largest distance, speed 1", MaxDistance, 1},
		{"largest distance, largest speed", MaxDistance, MaxSpeed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := windowConfig(tt.dist, tt.speed, "foot")
			got := merchantTripTicks(&c, newLookup(&c), brokerOf(t, &c))
			if got < 1 || got > MaxTravelTicks {
				t.Errorf("merchantTripTicks(distance %d, speed %d) = %d, want 1 to %d", tt.dist, tt.speed, got, int64(MaxTravelTicks))
			}
			if w, ok := effectiveWindow(&c, newLookup(&c), brokerOf(t, &c)); !ok || w < 1 {
				t.Errorf("effectiveWindow = %d, %v, want a positive window", w, ok)
			}
		})
	}
}

// TestWindow_MerchantRoleOnSnapshot verifies the window on a world: the snapshot's row of
// the broker role carries the window with the trip in it, and the row of a role without a
// block does not.
//
// Arithmetic: foot at speed 1 over edges of 40 takes 40 ticks, so the broker's window is
// max(30, 2 * 40) = 80; the miller's stays 30.
//
// Requirement: FR-WEL-009, FR-WEL-020
func TestWindow_MerchantRoleOnSnapshot(t *testing.T) {
	t.Parallel()
	c := windowConfig(40, 1, "foot")
	w, _, _ := newMerchWorld(t, func(cc *Config) { *cc = c }, nil)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	got := map[string]int64{}
	for _, r := range s.Roles {
		got[s.Names.Roles[r.Role]] = r.Window
	}
	if got["broker"] != 80 || got["miller"] != 30 {
		t.Errorf("Snapshot().Roles windows = %v, want broker 80 and miller 30", got)
	}
}

// TestMerchantTripTicks_ZeroLookupAndCache verifies that a zero lookup (no cache) still
// answers, and that a lookup built once answers every role of the configuration from one
// pass over the graph: the second answer is the first, even after the edges change.
//
// Guard: the cache is the defence against MaxRoles merchant roles each rebuilding a graph
// of MaxRegions regions in the validator; a role asked twice must not walk the graph twice.
//
// Requirement: FR-WEL-009
func TestMerchantTripTicks_ZeroLookupAndCache(t *testing.T) {
	t.Parallel()
	c := windowConfig(40, 1, "foot")
	b := brokerOf(t, &c)
	if got := merchantTripTicks(&c, lookup{}, b); got != 40 {
		t.Errorf("merchantTripTicks(zero lookup) = %d, want 40", got)
	}
	lk := newLookup(&c)
	if got := merchantTripTicks(&c, lk, b); got != 40 {
		t.Fatalf("merchantTripTicks = %d, want 40", got)
	}
	c.Edges = nil // the lookup is valid while c is unchanged: the cached answer stands
	if got := merchantTripTicks(&c, lk, b); got != 40 {
		t.Errorf("merchantTripTicks (second ask) = %d, want the cached 40", got)
	}
}
