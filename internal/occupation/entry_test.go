package occupation_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestRenewal_CarryAcrossTicks verifies the carry of the worked case, with the
// carry fed back each tick.
//
// Renewal 5 000, one household removed a tick: total 5 000 (0 entries, carry 5
// 000), then 10 000 (1 entry, carry 0), then 5 000, then 10 000: entries 0, 1,
// 0, 1.
//
// Requirement: FR-SPN-024
func TestRenewal_CarryAcrossTicks(t *testing.T) {
	t.Parallel()
	carry := int64(0)
	wantN := []int64{0, 1, 0, 1}
	wantCarry := []int64{5000, 0, 5000, 0}
	for i := range wantN {
		n, next, ok := occupation.Renewal(carry, 1, 5000)
		if !ok || n != wantN[i] || next != wantCarry[i] {
			t.Fatalf("Renewal(%d, 1, 5000) at tick %d = %d, %d, %v, want %d, %d, true", carry, i, n, next, ok, wantN[i], wantCarry[i])
		}
		carry = next
	}
}

// TestRenewal_Cases verifies the carry rule at its edges and the refused
// arguments.
//
// 9 999 + 1 * 10 000 = 19 999: 1 entry, carry 9 999. 0 + 3 * 25 000 = 75 000: 7
// entries, carry 5 000. A rate of 0 or nothing removed keeps the carry.
//
// Requirement: FR-SPN-024
func TestRenewal_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		carry, removed, rate int64
		wantN, wantCarry     int64
		wantOK               bool
	}{
		{"carry plus a full household", 9999, 1, 10000, 1, 9999, true},
		{"rate above full", 0, 3, 25000, 7, 5000, true},
		{"rate 0", 1234, 5, 0, 0, 1234, true},
		{"nothing removed", 1234, 0, 5000, 0, 1234, true},
		{"negative carry", -1, 1, 5000, 0, 0, false},
		{"negative removed", 0, -1, 5000, 0, 0, false},
		{"negative rate", 0, 1, -1, 0, 0, false},
		{"product overflows", 0, math.MaxInt64, 2, 0, 0, false},
		{"sum overflows", math.MaxInt64, 1, 1, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, c, ok := occupation.Renewal(tt.carry, tt.removed, tt.rate)
			if n != tt.wantN || c != tt.wantCarry || ok != tt.wantOK {
				t.Errorf("Renewal(%d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.carry, tt.removed, tt.rate, n, c, ok, tt.wantN, tt.wantCarry, tt.wantOK)
			}
		})
	}
}

// TestFunded_Cases verifies the funded number against the worked case and its
// edges.
//
// Budget 15 000 at 10 000 each buys floor(1.5) = 1 of 5 due. Budget 100 000
// buys 10, more than the 2 due. An amount of 0 or below is not limited.
//
// Requirement: FR-SPN-008
func TestFunded_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		n, budget, amount, want int64
	}{
		{"worked case", 5, 15000, 10000, 1},
		{"budget exceeds demand", 2, 100000, 10000, 2},
		{"exact", 3, 30000, 10000, 3},
		{"one short", 3, 29999, 10000, 2},
		{"no budget", 3, 0, 10000, 0},
		{"negative budget", 3, -1, 10000, 0},
		{"free entrants", 3, 0, 0, 3},
		{"negative amount", 3, 0, -1, 3},
		{"none due", 0, 15000, 10000, 0},
		{"negative due", -2, 15000, 10000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Funded(tt.n, tt.budget, tt.amount); got != tt.want {
				t.Errorf("Funded(%d, %d, %d) = %d, want %d", tt.n, tt.budget, tt.amount, got, tt.want)
			}
		})
	}
}

// TestPlace_Cases verifies the host of an entrant: the smallest weight below the
// scale, ties to the earlier host, a full host skipped, and none.
//
// Hosts weigh 3, 1, 1, 5 at scale 4: weights 1 are smallest, the first (Ref 11)
// wins and has room 4 - 1 = 3; the 5 is above the scale. At scale 1 no host is
// below it, so an entrant is a new agent.
//
// FR-SPN-007 is covered in part: the placement only; the pool draw and the stores
// are the executor's (row 11).
//
// Requirement: FR-SPN-007
func TestPlace_Cases(t *testing.T) {
	t.Parallel()
	hosts := []occupation.Host{{Ref: 10, Weight: 3}, {Ref: 11, Weight: 1}, {Ref: 12, Weight: 1}, {Ref: 13, Weight: 5}}
	tests := []struct {
		name     string
		hosts    []occupation.Host
		scale    int64
		wantRef  int
		wantRoom int64
		wantOK   bool
	}{
		{"smallest, first of the tie", hosts, 4, 11, 3, true},
		{"host at the scale is full", []occupation.Host{{Ref: 7, Weight: 4}}, 4, 0, 0, false},
		{"host just below the scale", []occupation.Host{{Ref: 7, Weight: 3}}, 4, 7, 1, true},
		{"scale 1", hosts, 1, 0, 0, false},
		{"empty weight host", []occupation.Host{{Ref: 7, Weight: 0}}, 2, 7, 2, true},
		{"no hosts", nil, 4, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref, room, ok := occupation.Place(tt.hosts, tt.scale)
			if ref != tt.wantRef || room != tt.wantRoom || ok != tt.wantOK {
				t.Errorf("Place(%+v, %d) = %d, %d, %v, want %d, %d, %v", tt.hosts, tt.scale, ref, room, ok, tt.wantRef, tt.wantRoom, tt.wantOK)
			}
		})
	}
}

// TestRetarget_Cases verifies the target carried over a role change: the offset
// from the old role is kept, both clamps cut it, and overflow is refused.
//
// Old role 10 000, new 12 000: a target of 9 000 (offset -1 000) becomes 11 000,
// inside [10 800, 13 200]; 5 000 gives 7 000, cut to 10 800; 15 000 gives
// 17 000, cut to 13 200.
//
// FR-SPN-006 is covered in part: the carried target only; the account, goods, job
// and history steps are the executor's (row 11).
//
// Requirement: FR-SPN-006
func TestRetarget_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                             string
		target, oldRole, newRole, lo, hi int64
		want                             int64
		wantOK                           bool
	}{
		{"keeps the offset", 9000, 10000, 12000, 10800, 13200, 11000, true},
		{"same as the old centre", 10000, 10000, 12000, 10800, 13200, 12000, true},
		{"lower clamp", 5000, 10000, 12000, 10800, 13200, 10800, true},
		{"upper clamp", 15000, 10000, 12000, 10800, 13200, 13200, true},
		{"offset overflows", math.MaxInt64, math.MinInt64, 0, 0, 1, 0, false},
		{"sum overflows", math.MaxInt64, 0, 1, 0, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := occupation.Retarget(tt.target, tt.oldRole, tt.newRole, tt.lo, tt.hi)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Retarget(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.target, tt.oldRole, tt.newRole, tt.lo, tt.hi, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
