package room_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// TestOrder_Properties verifies, over 5 000 random instances of the order of
// 5.5.2.1, what any correct order satisfies: no step takes more than the lot
// holds, a met order leaves no shortfall, a discard (by unit) takes the least
// that meets the dimension it works on, and an excess order takes only whole
// lots.
//
// Requirement: FR-STO-018, FR-STO-024
func TestOrder_Properties(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 4))
	for n := range 5000 {
		var cands []room.Candidate
		var load room.Pair
		for c := range 1 + rng.IntN(4) {
			cd := room.Candidate{Commodity: c, Size: room.Pair{Mass: rng.Int64N(4), Bulk: rng.Int64N(4)}}
			for range 1 + rng.IntN(3) {
				q := rng.Int64N(6) // a lot may be empty
				cd.Lots = append(cd.Lots, q)
				load.Mass += q * cd.Size.Mass
				load.Bulk += q * cd.Size.Bulk
			}
			cands = append(cands, cd)
		}
		rm := room.Pair{Mass: rng.Int64N(40), Bulk: rng.Int64N(40)}
		for _, byUnit := range []bool{false, true} {
			var steps []room.Step
			var met bool
			if byUnit {
				steps, met = room.DiscardOrder(room.Pair{Mass: load.Mass - rm.Mass, Bulk: load.Bulk - rm.Bulk}, rm, cands)
			} else {
				steps, met = room.ExcessOrder(load, rm, cands)
			}
			left := make([][]int64, len(cands))
			for i, c := range cands {
				left[i] = slices.Clone(c.Lots)
			}
			after := load
			for _, s := range steps {
				if s.Units < 1 || s.Units > left[s.Cand][s.Lot] {
					t.Fatalf("case %d byUnit=%v: step %+v takes more than the lot holds (%d)", n, byUnit, s, left[s.Cand][s.Lot])
				}
				if !byUnit && s.Units != cands[s.Cand].Lots[s.Lot] {
					t.Fatalf("case %d: excess step %+v is not a whole lot", n, s)
				}
				left[s.Cand][s.Lot] -= s.Units
				after.Mass -= s.Units * cands[s.Cand].Size.Mass
				after.Bulk -= s.Units * cands[s.Cand].Size.Bulk
			}
			fits := after.Mass <= rm.Mass && after.Bulk <= rm.Bulk
			if met != fits {
				t.Fatalf("case %d byUnit=%v: met = %v but the load after is %v in room %v", n, byUnit, met, after, rm)
			}
			if !met {
				// Not met only when a dimension is over and nothing with a size in it is left.
				for d, over := range []bool{after.Mass > rm.Mass, after.Bulk > rm.Bulk} {
					if !over {
						continue
					}
					for i, c := range cands {
						size := []int64{c.Size.Mass, c.Size.Bulk}[d]
						if size > 0 && slices.Max(left[i]) > 0 {
							t.Fatalf("case %d byUnit=%v: gave up with dimension %d over while candidate %d has units", n, byUnit, d, i)
						}
					}
				}
			}
			if byUnit && met && len(steps) > 0 {
				// The last step took the least: one unit less would leave its dimension short.
				last := steps[len(steps)-1]
				back := room.Pair{Mass: after.Mass + (last.Units-1)*0, Bulk: after.Bulk}
				_ = back
				if last.Units > 1 {
					c := cands[last.Cand]
					m := after.Mass + c.Size.Mass
					b := after.Bulk + c.Size.Bulk
					if m <= rm.Mass && b <= rm.Bulk {
						t.Fatalf("case %d: the last discard step %+v took a unit more than needed", n, last)
					}
				}
			}
		}
	}
}

// TestOrder_Selection verifies the choices the properties do not pin: empty
// lots are skipped, a candidate with no size in the binding dimension is not
// picked, a candidate with no units is not picked, equal loads go to the lower
// ID whichever way the slice is ordered, and a shortfall in one dimension only
// binds that dimension (including a negative one in the other).
//
// Requirement: FR-STO-018, FR-STO-024
func TestOrder_Selection(t *testing.T) {
	t.Parallel()
	one := room.Pair{Mass: 1, Bulk: 1}
	for _, tt := range []struct {
		name      string
		shortfall room.Pair
		rm        room.Pair
		cands     []room.Candidate
		want      []room.Step
	}{
		{"empty lot skipped", room.Pair{Mass: 3}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: one, Lots: []int64{0, 5}}}, []room.Step{{Cand: 0, Lot: 1, Units: 3}}},
		{"no units is not picked", room.Pair{Mass: 3}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 9}, Lots: []int64{0}}, {Commodity: 2, Size: one, Lots: []int64{5}}}, []room.Step{{Cand: 1, Lot: 0, Units: 3}}},
		{"no size in the dimension is not picked", room.Pair{Mass: 3}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Bulk: 9}, Lots: []int64{50}}, {Commodity: 2, Size: one, Lots: []int64{5}}}, []room.Step{{Cand: 1, Lot: 0, Units: 3}}},
		{"bulk only", room.Pair{Mass: -5, Bulk: 2}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 5, Bulk: 1}, Lots: []int64{9}}}, []room.Step{{Cand: 0, Lot: 0, Units: 2}}},
		{"mass only", room.Pair{Mass: 2, Bulk: -5}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 1, Bulk: 5}, Lots: []int64{9}}}, []room.Step{{Cand: 0, Lot: 0, Units: 2}}},
		{"zero shortfall", room.Pair{}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 1, Size: one, Lots: []int64{9}}}, nil},
		{"larger relative bulk binds", room.Pair{Mass: 1, Bulk: 1}, room.Pair{Mass: 10, Bulk: 2},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 1}, Lots: []int64{9}}, {Commodity: 2, Size: room.Pair{Bulk: 1}, Lots: []int64{9}}}, []room.Step{{Cand: 1, Lot: 0, Units: 1}, {Cand: 0, Lot: 0, Units: 1}}},
		{"equal relative shortfall binds mass", room.Pair{Mass: 2, Bulk: 4}, room.Pair{Mass: 5, Bulk: 10},
			[]room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 1}, Lots: []int64{9}}, {Commodity: 2, Size: room.Pair{Bulk: 1}, Lots: []int64{9}}}, []room.Step{{Cand: 0, Lot: 0, Units: 2}, {Cand: 1, Lot: 0, Units: 4}}},
		{"equal load, lower ID first", room.Pair{Mass: 4}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 8, Size: one, Lots: []int64{4}}, {Commodity: 2, Size: one, Lots: []int64{4}}}, []room.Step{{Cand: 1, Lot: 0, Units: 4}}},
		{"equal load, lower ID first (other order)", room.Pair{Mass: 4}, room.Pair{Mass: 9, Bulk: 9},
			[]room.Candidate{{Commodity: 2, Size: one, Lots: []int64{4}}, {Commodity: 8, Size: one, Lots: []int64{4}}}, []room.Step{{Cand: 0, Lot: 0, Units: 4}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, met := room.DiscardOrder(tt.shortfall, tt.rm, tt.cands)
			if !slices.Equal(got, tt.want) || !met {
				t.Errorf("DiscardOrder(%v, %v, %v) = %v, %v, want %v, true", tt.shortfall, tt.rm, tt.cands, got, met, tt.want)
			}
		})
	}
}

// TestOrder_ExtremeLoad verifies that a candidate whose load overflows int64
// still ranks (as the greatest) and a discard of whole units uses ceil: 7
// needed at size 2 is 4 units.
//
// Requirement: FR-STO-024
func TestOrder_ExtremeLoad(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{
		{Commodity: 1, Size: room.Pair{Mass: 2}, Lots: []int64{math.MaxInt64 / 2}},
		{Commodity: 2, Size: room.Pair{Mass: 2}, Lots: []int64{10}},
	}
	got, _ := room.DiscardOrder(room.Pair{Mass: 7}, room.Pair{Mass: 9, Bulk: 9}, cands)
	if want := []room.Step{{Cand: 0, Lot: 0, Units: 4}}; !slices.Equal(got, want) {
		t.Errorf("DiscardOrder = %v, want %v", got, want)
	}
}

// TestScale_Boundaries verifies the edges of 6.2.2: targets exactly filling
// the avail do not scale, one unit over scales, a zero avail gives 0, and
// ScaleQty of zero or of f = 0 is 0 while f = 1 floors.
//
// Requirement: FR-STO-022
func TestScale_Boundaries(t *testing.T) {
	t.Parallel()
	tg := []room.Item{{Qty: 10, Size: room.Pair{Mass: 1}}}
	rm := room.Pair{Mass: 10, Bulk: 0}
	for _, tt := range []struct {
		name string
		rm   room.Pair
		want int64
	}{
		{"exactly fills", room.Pair{Mass: 10}, 10_000},
		{"one over", room.Pair{Mass: 9}, 9_000},
		{"avail zero", room.Pair{Mass: 0}, 0},
		{"tiny avail floors to 1000", room.Pair{Mass: 1}, 1_000},
	} {
		if got := room.Scale(tg, tt.rm, room.Pair{}, room.Pair{}); got != tt.want {
			t.Errorf("%s: Scale = %d, want %d", tt.name, got, tt.want)
		}
	}
	_ = rm
	if got := room.Scale(tg, room.Pair{Mass: 10}, room.Pair{Mass: 10}, room.Pair{}); got != 0 {
		t.Errorf("Scale with pledge = room = %d, want 0", got)
	}
	if got := room.Scale(nil, room.Pair{}, room.Pair{}, room.Pair{}); got != 10_000 {
		t.Errorf("Scale of no target = %d, want 10000", got)
	}
	if got := room.ScaleQty(1, 1); got != 0 {
		t.Errorf("ScaleQty(1, 1) = %d, want 0", got)
	}
	if got := room.ScaleQty(10_000, 1); got != 1 {
		t.Errorf("ScaleQty(10000, 1) = %d, want 1", got)
	}
	if got := room.ScaleQty(0, 5); got != 0 {
		t.Errorf("ScaleQty(0, 5) = %d, want 0", got)
	}
}

// TestKRoom_Boundaries verifies the edges of 5.3.2 that the random search
// meets rarely: k0 of exactly 1, a free room of exactly the load, one unit
// less, a residue that is only the reason for a refusal, the lower bound at its
// exact quotient and just above it, and an overflow in the output, the input
// and the residue term.
//
// Requirement: FR-STO-012
func TestKRoom_Boundaries(t *testing.T) {
	t.Parallel()
	one := room.Pair{Mass: 1}
	out := func(q, res int64) []room.Output {
		return []room.Output{{Item: room.Item{Qty: q, Size: one}, Residue: res}}
	}
	in := func(q int64) []room.Item { return []room.Item{{Qty: q, Size: one}} }
	for _, tt := range []struct {
		name string
		ins  []room.Item
		outs []room.Output
		e    int64
		free room.Pair
		k0   int64
		want int64
	}{
		{"k0 one fits exactly", nil, out(4, 0), 10_000, room.Pair{Mass: 4, Bulk: 9}, 1, 1},
		{"one unit short", nil, out(4, 0), 10_000, room.Pair{Mass: 3, Bulk: 9}, 1, 0},
		{"residue tips a half unit", nil, out(1, 5_000), 5_000, room.Pair{Mass: 1, Bulk: 9}, 4, 1},
		{"residue exactly fits", nil, out(1, 5_000), 5_000, room.Pair{Mass: 2, Bulk: 9}, 1, 1},
		{"residue large", nil, out(1, 9_999), 5_000, room.Pair{Mass: 1, Bulk: 9}, 2, 0},
		{"lower bound exact", in(3), out(1, 0), 10_000, room.Pair{Mass: -6, Bulk: 9}, 9, 9},
		{"lower bound above k0", in(3), out(1, 0), 10_000, room.Pair{Mass: -6, Bulk: 9}, 2, 0},
		{"lower bound just fits", in(3), out(1, 0), 10_000, room.Pair{Mass: -2, Bulk: 9}, 2, 2},
		{"R zero with c positive", nil, out(1, 0), 10_000, room.Pair{Mass: 0, Bulk: 9}, 3, 0},
		{"c zero with R zero", in(1), out(1, 0), 10_000, room.Pair{Mass: 0, Bulk: 9}, 3, 3},
		{"c zero with R negative", in(1), out(1, 0), 10_000, room.Pair{Mass: -1, Bulk: 9}, 3, 0},
		{"residue term overflows", nil, []room.Output{{Item: room.Item{Qty: 1, Size: room.Pair{Mass: math.MaxInt64}}, Residue: 5}}, 5_000, room.Pair{Mass: 9, Bulk: 9}, 3, 0},
		{"yield overflows", nil, out(math.MaxInt64, 0), 10_000, room.Pair{Mass: 9, Bulk: 9}, 3, 0},
		{"scaled input overflows", in(math.MaxInt64 / 5_000), out(1, 0), 10_000, room.Pair{Mass: 9, Bulk: 9}, 3, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.KRoom(tt.ins, tt.outs, tt.e, tt.free, room.Pair{}, tt.k0); got != tt.want {
				t.Errorf("KRoom(%v, %v, E=%d, free %v, k0 %d) = %d, want %d", tt.ins, tt.outs, tt.e, tt.free, tt.k0, got, tt.want)
			}
		})
	}
}
