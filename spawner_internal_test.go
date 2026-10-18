package economysim

import (
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// namedSpawner is a hand-written Spawner distinguishable by its tag.
type namedSpawner struct{ tag string }

func (namedSpawner) Decide(_ *RegionView, _ *rand.Rand, _ *Decisions) {}

// TestWithSpawner_Rules verifies each rule of the registration: a name that is
// not an ID is ErrInvalidID, "default" or a repeated name is ErrDuplicateID and
// a nil extension is ErrNilExtension, checked in that order.
//
// Requirement: FR-WLD-002, FR-SPN-001
func TestWithSpawner_Rules(t *testing.T) {
	t.Parallel()
	s := namedSpawner{"a"}
	tests := []struct {
		name string
		opts []Option
		want error
	}{
		{"valid", []Option{WithSpawner("custom", s)}, nil},
		{"built-in name", []Option{WithSpawner("default", s)}, ErrDuplicateID},
		{"given twice", []Option{WithSpawner("custom", s), WithSpawner("custom", s)}, ErrDuplicateID},
		{"nil extension", []Option{WithSpawner("custom", nil)}, ErrNilExtension},
		{"empty name", []Option{WithSpawner("", s)}, ErrInvalidID},
		{"upper case name", []Option{WithSpawner("Custom", s)}, ErrInvalidID},
		{"invalid name wins over nil", []Option{WithSpawner("Bad", nil)}, ErrInvalidID},
		{"built-in name wins over nil", []Option{WithSpawner("default", nil)}, ErrDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := applyOptions(tt.opts)
			if tt.want == nil && err != nil || tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("applyOptions(%s) = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestResolveSpawner verifies the name check: "default" is the built-in, a
// registered name finds its extension in the order given, and an unknown or
// empty name is a *ValidationError at the path with rule ErrUnknownSpawner and
// the name as its value.
//
// Requirement: FR-SPN-040, FR-CFG-038
func TestResolveSpawner(t *testing.T) {
	t.Parallel()
	o, err := applyOptions([]Option{WithSpawner("zeta", namedSpawner{"z"}), WithSpawner("alpha", namedSpawner{"a"})})
	if err != nil {
		t.Fatalf("applyOptions: %v", err)
	}
	if want := []string{"zeta", "alpha"}; !slices.Equal(o.spawners.names, want) {
		t.Errorf("registered names = %v, want %v (order given)", o.spawners.names, want)
	}
	path := []string{"world", "spawner"}
	tests := []struct {
		name string
		in   string
		want Spawner
		fail bool
	}{
		{"built-in", "default", DefaultSpawner{}, false},
		{"first registered", "zeta", namedSpawner{"z"}, false},
		{"second registered", "alpha", namedSpawner{"a"}, false},
		{"unknown", "other", nil, true},
		{"empty", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveSpawner(&o, tt.in, path)
			if !tt.fail {
				if err != nil || !reflect.DeepEqual(got, tt.want) {
					t.Errorf("resolveSpawner(%q) = %v, %v, want %v, nil", tt.in, got, err, tt.want)
				}
				return
			}
			var ve *ValidationError
			if !errors.Is(err, ErrUnknownSpawner) || !errors.As(err, &ve) {
				t.Fatalf("resolveSpawner(%q) error = %v, want a *ValidationError of ErrUnknownSpawner", tt.in, err)
			}
			if !slices.Equal(ve.Path, path) || ve.Value != tt.in || got != nil {
				t.Errorf("resolveSpawner(%q) = %v, path %v, value %q, want nil, path %v, value %q", tt.in, got, ve.Path, ve.Value, path, tt.in)
			}
		})
	}
}

// TestRefusalReason_Total verifies that every refusal of internal/occupation
// has its own non-empty Reason, in the order of the checks, and that "no
// refusal" has none. The loop runs to RefusalEnd, so a refusal appended without
// a mapping fails here.
//
// Requirement: FR-SPN-010
func TestRefusalReason_Total(t *testing.T) {
	t.Parallel()
	want := []Reason{
		"", ReasonUnknownAgent, ReasonUnknownRole, ReasonExternalAgent,
		ReasonSameRole, ReasonBadHouseholds, ReasonUnsupportedCause,
	}
	if int(occupation.RefusalEnd) != len(want) {
		t.Fatalf("RefusalEnd = %d, want %d: map the new refusal and extend want", occupation.RefusalEnd, len(want))
	}
	for r := occupation.RefusalNone; r < occupation.RefusalEnd; r++ {
		if got := refusalReason(r); got != want[r] {
			t.Errorf("refusalReason(%d) = %q, want %q", r, got, want[r])
		}
	}
}

// TestDecisions_Reset verifies that reset empties the plan, keeps its backing
// array and zeroes every count, so a consumer's spawner starts each region at 0.
//
// Guard: a count left over from the previous region would add to the next
// region's EntryLimited.
func TestDecisions_Reset(t *testing.T) {
	t.Parallel()
	d := Decisions{
		List:      []Decision{{Kind: DecideEnter, Households: 1}},
		regionCap: 1, roleCap: 2, cooldown: 3, hysteresis: 4, exitCap: 5,
		noViableRole: 6, stranded: 9, noViableByRole: []int64{7, 8},
	}
	d.reset()
	if len(d.List) != 0 || cap(d.List) != 1 {
		t.Errorf("after reset len/cap of List = %d/%d, want 0/1", len(d.List), cap(d.List))
	}
	if d.regionCap|d.roleCap|d.cooldown|d.hysteresis|d.exitCap|d.noViableRole|d.stranded != 0 || d.noViableByRole[0]|d.noViableByRole[1] != 0 {
		t.Errorf("after reset counts = %+v, want all 0", d)
	}
}

// TestSatAllowance verifies that the allowance is the ceiling of the share and
// saturates to MaxInt64 where the product leaves int64.
//
// Guard: a wrapped allowance would be negative and silently stop all exits.
//
// Requirement: FR-SPN-016
func TestSatAllowance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		households, share int64
		want              int64
	}{
		{"ceiling of 1000 at 50 basis points: 1000*50/10000 = 5", 1000, 50, 5},
		{"product beyond int64 saturates", math.MaxInt64, 20_000, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satAllowance(tt.households, tt.share); got != tt.want {
				t.Errorf("satAllowance(%d, %d) = %d, want %d", tt.households, tt.share, got, tt.want)
			}
		})
	}
}
