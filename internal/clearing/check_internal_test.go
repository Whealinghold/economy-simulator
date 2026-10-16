package clearing

import "testing"

// TestLotLess_OrdersEachFieldInItsOwnDirection verifies the lot order that
// Check sorts and searches by: unnamed before named, then older Born first,
// then lower Durability first, and equal lots compare 0. The order is the
// same in both arguments' roles (antisymmetric).
//
// Guard: sort and search share lotLess, so a reversed or constant branch
// still resolves every fill through Check; only the comparator itself shows
// it. A branch that returned the same sign for (a, b) and (b, a) would
// break the binary search on some books.
func TestLotLess_OrdersEachFieldInItsOwnDirection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b Lot
		want int
	}{
		{"unnamed before named", Lot{}, Lot{Named: true}, -1},
		{"named after unnamed", Lot{Named: true}, Lot{}, 1},
		{"named decides before born", Lot{Born: 9}, Lot{Named: true, Born: 1}, -1},
		{"older born first", Lot{Named: true, Born: 1}, Lot{Named: true, Born: 2}, -1},
		{"newer born after", Lot{Named: true, Born: 2}, Lot{Named: true, Born: 1}, 1},
		{"born decides before durability", Lot{Named: true, Born: 1, Durability: 9}, Lot{Named: true, Born: 2, Durability: 1}, -1},
		{"lower durability first", Lot{Named: true, Born: 1, Durability: 1}, Lot{Named: true, Born: 1, Durability: 2}, -1},
		{"higher durability after", Lot{Named: true, Born: 1, Durability: 2}, Lot{Named: true, Born: 1, Durability: 1}, 1},
		{"equal lots", Lot{Named: true, Born: 1, Durability: 1}, Lot{Named: true, Born: 1, Durability: 1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := lotLess(tt.a, tt.b); got != tt.want {
				t.Errorf("lotLess(%+v, %+v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
