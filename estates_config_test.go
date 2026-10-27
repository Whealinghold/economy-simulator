package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// oneRegionEstates returns the one-region configuration of the worked examples
// of spec-config.md 8.2.4 with the region at the given pool fields.
func oneRegionEstates(on bool, release, below int64) economysim.Config {
	c := economysim.Config{Regions: []economysim.Region{{ID: "north", PoolRelease: release, ReliefBelow: below}}}
	if on {
		c.Modules.Estates = &economysim.Estates{}
	}
	return c
}

// TestHash_EstatesModule verifies the Estates row of spec-config.md 8.2.4: a
// module that is on and has no field costs exactly the bytes that name it, and
// a region whose pool fields are at zero adds no pair for them even though the
// module is on.
//
// Requirement: FR-CFG-099, FR-CFG-104, FR-CFG-105
func TestHash_EstatesModule(t *testing.T) {
	t.Parallel()
	const want = "0848fee29c503bda99a17acb2c44c42fbba8b730eadc2079e3d8123b2d3317ad"
	if got := hexHash(oneRegionEstates(true, 0, 0)); got != want {
		t.Errorf("HashConfig(one region, Estates on, pool fields 0) = %s, want %s", got, want)
	}
}

// TestHash_RegionPoolFieldsOffLeaveNoTrace verifies that the defaults of the
// pool fields (100 and 10 000) leave the hash of a region as it was while the
// module is off, so the all-off reference configuration does not move.
//
// Requirement: FR-CFG-104, FR-CFG-105, FR-CFG-099
//
// Guard: the defaults are not zero, which architecture.md 2.10.5 allows only
// because the fields are written while the module is on; a pair written with the
// module off would move the pinned hash of every configuration.
func TestHash_RegionPoolFieldsOffLeaveNoTrace(t *testing.T) {
	t.Parallel()
	const want = "ed652dfd86904c18fd76feb17175e7efeef6d4afe94bec1e3b4f5a2e2409f013"
	if got := hexHash(oneRegionEstates(false, 100, 10_000)); got != want {
		t.Errorf("HashConfig(one region, Estates off, defaults) = %s, want %s", got, want)
	}
}

// TestHash_RegionPoolFieldsOn verifies the new row of spec-config.md 8.2.4
// (the module on, the region at 100 and 10 000), that each field moves the hash
// while the module is on and neither does while it is off, and that either at 0
// with the module on is the bytes of the Estates row without it.
//
// Requirement: FR-CFG-026, FR-CFG-099
func TestHash_RegionPoolFieldsOn(t *testing.T) {
	t.Parallel()
	const row = "75e8cca7e5d584f612e87b572b7d4113cda12ba8dc5f1456bddcca95949d869d"
	if got := hexHash(oneRegionEstates(true, 100, 10_000)); got != row {
		t.Errorf("HashConfig(one region, Estates on, 100 and 10 000) = %s, want %s", got, row)
	}
	tests := []struct {
		name         string
		on           bool
		release      int64
		below        int64
		sameAsRow    bool // equal to the row above
		sameAsBase   bool // equal to the Estates row with the fields at 0
		sameAsNoMods bool // equal to the one-region row, module off
	}{
		{"release changed", true, 101, 10_000, false, false, false},
		{"relief line changed", true, 100, 10_001, false, false, false},
		{"release at zero", true, 0, 10_000, false, false, false},
		{"both at zero equal the Estates row", true, 0, 0, false, true, false},
		{"off ignores any value", false, 7, 9, false, false, true},
	}
	base := hexHash(oneRegionEstates(true, 0, 0))
	plain := hexHash(oneRegionEstates(false, 0, 0))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hexHash(oneRegionEstates(tt.on, tt.release, tt.below))
			if (got == row) != tt.sameAsRow || (got == base) != tt.sameAsBase || (got == plain) != tt.sameAsNoMods {
				t.Errorf("HashConfig(on=%v, %d, %d) = %s: equal to row %v, Estates row %v, plain row %v; want %v, %v, %v",
					tt.on, tt.release, tt.below, got, got == row, got == base, got == plain,
					tt.sameAsRow, tt.sameAsBase, tt.sameAsNoMods)
			}
		})
	}
}

// TestRegionPoolFields_Bounds verifies that a pool field outside its bound is
// ErrOutOfBounds at its path while the module is on, and that the same value is
// no error while it is off (spec-config.md 8.2.1), and that the bounds are
// inclusive.
//
// Requirement: FR-EST-023
func TestRegionPoolFields_Bounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		release int64
		below   int64
		field   string
		bad     bool
	}{
		{"release below zero", -1, 10_000, "pool_release", true},
		{"release above 10 000", 10_001, 10_000, "pool_release", true},
		{"release at 10 000", 10_000, 10_000, "", false},
		{"release at zero", 0, 10_000, "", false},
		{"relief line below zero", 100, -1, "relief_below", true},
		{"relief line above the bound", 100, economysim.MaxBasisPoints + 1, "relief_below", true},
		{"relief line at the bound", 100, economysim.MaxBasisPoints, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			on := minimalConfig()
			on.Modules.Estates = &economysim.Estates{}
			on.Regions[0].PoolRelease, on.Regions[0].ReliefBelow = tt.release, tt.below
			err := on.Validate()
			if tt.bad {
				vs, _ := violations(err)
				if !errors.Is(err, economysim.ErrOutOfBounds) || len(vs) != 1 || len(pathsOf(vs)[0]) != 3 || pathsOf(vs)[0][2] != tt.field {
					t.Errorf("Validate(estates on, %s) = %v, want ErrOutOfBounds at regions.home.%s", tt.name, err, tt.field)
				}
			} else if err != nil {
				t.Errorf("Validate(estates on, %s) = %v, want nil", tt.name, err)
			}
			off := on
			off.Modules.Estates = nil
			if err := off.Validate(); err != nil {
				t.Errorf("Validate(estates off, %s) = %v, want nil: the fields are not validated while the module is off", tt.name, err)
			}
		})
	}
}

// TestDefaultRegion_PoolFields verifies the defaults of the pool fields.
//
// Requirement: FR-EST-023
func TestDefaultRegion_PoolFields(t *testing.T) {
	t.Parallel()
	r := economysim.DefaultRegion()
	if r.PoolRelease != 100 || r.ReliefBelow != 10_000 {
		t.Errorf("DefaultRegion() pool fields = %d and %d, want 100 and 10 000", r.PoolRelease, r.ReliefBelow)
	}
}
