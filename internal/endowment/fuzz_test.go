package endowment_test

import (
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// FuzzRegen verifies one tick of Regen never panics, keeps the stock in
// [0, C] and the carry in [0, D), reports Added as the change of the stock, and,
// when the stock is not clamped, satisfies the identity
// Added * D + Carry' = Carry + n against the math/big oracle.
//
// Requirement: NFR-RES-003, FR-RES-010
func FuzzRegen(f *testing.F) {
	f.Add(uint8(1), int64(10_000), int64(0), int64(1_000_000), int64(10_000), int64(365), int64(1000), int64(5), int64(10_000))
	f.Add(uint8(2), int64(3000), int64(20), int64(50_000), int64(10_000), int64(640), int64(20_000), int64(0), int64(20_000))
	f.Add(uint8(0), int64(365), int64(0), int64(100), int64(1_000_000), int64(100_000), int64(0), int64(99), int64(100_000))
	f.Fuzz(func(t *testing.T, shape uint8, rate, refuge, capacity, scale, tpy, stock, carry, season int64) {
		p := endowment.Params{Shape: endowment.Shape(shape % 3), Rate: rate, Refuge: refuge, Capacity: capacity, ScaleBP: scale, TicksPerYear: tpy}
		got, ok := endowment.Regen(p, endowment.Step{Stock: stock, Carry: carry, SeasonBP: season})
		if !ok {
			return
		}
		d := denom(tpy)
		if got.Stock < 0 || got.Stock > capacity || got.Carry < 0 || big.NewInt(got.Carry).Cmp(d) >= 0 {
			t.Fatalf("Regen(%+v, stock %d carry %d season %d) = %+v out of range", p, stock, carry, season, got)
		}
		if got.Added != got.Stock-stock {
			t.Errorf("Regen(%+v, stock %d) = %+v, Added is not the change of the stock", p, stock, got)
		}
		if got.Stock == capacity {
			return
		}
		lhs := new(big.Int).Mul(big.NewInt(got.Added), d)
		lhs.Add(lhs, big.NewInt(got.Carry))
		rhs := new(big.Int).Add(big.NewInt(carry), bigNumerator(p, stock, season))
		if capacity > 0 && lhs.Cmp(rhs) != 0 {
			t.Errorf("Regen(%+v, stock %d carry %d season %d): Added*D+carry' = %v, carry+n = %v", p, stock, carry, season, lhs, rhs)
		}
	})
}

// FuzzRation verifies Ration never panics on up to eight requests built from the
// input, never grants above a request or above the stock, and gives the same
// grants for the reversed listing.
//
// Requirement: NFR-RES-003, FR-RES-020
func FuzzRation(f *testing.F) {
	f.Add(int64(10), []byte{4, 1, 3, 1, 3, 1, 2, 1, 1, 1})
	f.Add(int64(7), []byte{1, 5, 1, 4, 1, 3})
	f.Add(int64(0), []byte{})
	f.Fuzz(func(t *testing.T, available int64, raw []byte) {
		var reqs []endowment.Request
		for i := 0; i+1 < len(raw) && len(reqs) < 8; i += 2 {
			reqs = append(reqs, endowment.Request{Ref: len(reqs), Executions: int64(raw[i]), Draw: int64(raw[i+1]), Key: uint64(raw[i]) % 3})
		}
		grants := make([]int64, len(reqs))
		if !endowment.Ration(available, slices.Clone(reqs), grants) {
			return
		}
		var used int64
		for i, r := range reqs {
			if grants[i] < 0 || grants[i] > r.Executions {
				t.Fatalf("Ration(%d, %+v) granted %d to %+v", available, reqs, grants[i], r)
			}
			used += grants[i] * r.Draw
		}
		if used > available {
			t.Errorf("Ration(%d, %+v) granted %d units", available, reqs, used)
		}
		rev := slices.Clone(reqs)
		slices.Reverse(rev)
		again := make([]int64, len(reqs))
		endowment.Ration(available, rev, again)
		if !slices.Equal(grants, again) {
			t.Errorf("Ration(%d, %+v) = %v, reversed %v", available, reqs, grants, again)
		}
	})
}

// FuzzAbundance verifies Abundance never panics and, for a curve Check accepts,
// lies between the least and greatest factor of the knots.
//
// Requirement: NFR-RES-003, FR-RES-013
func FuzzAbundance(f *testing.F) {
	f.Add(int64(0), int64(2000), int64(2000), int64(8000), int64(10_000), int64(10_000), int64(1000), int64(2000))
	f.Fuzz(func(t *testing.T, x0, y0, x1, y1, x2, y2, stock, capacity int64) {
		knots := []curve.Knot{{X: x0, Y: y0}, {X: x1, Y: y1}, {X: x2, Y: y2}}
		got, ok := endowment.Abundance(knots, stock, capacity)
		if !ok || curve.Check(knots, 10_000, 100_000) >= 0 || capacity == 0 {
			return
		}
		lo, hi := min(y0, y1, y2), max(y0, y1, y2)
		if got < lo || got > hi {
			t.Errorf("Abundance(%v, %d, %d) = %d, outside [%d, %d]", knots, stock, capacity, got, lo, hi)
		}
	})
}
