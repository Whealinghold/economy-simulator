package steady

import (
	"context"
	"errors"
	"math/big"
	"testing"
)

// fuzzEndow decodes a resource row of small values from a fuzz input.
func fuzzEndow(b *byteReader) Endow {
	return Endow{
		Resource: 0, Region: 0,
		Shape:       int(b.next() % 3),
		Capacity:    int64(b.next()) * 10,
		Stock:       int64(b.next()) * 10,
		Rate:        int64(b.next()) * 100,
		PlanShareBP: int64(b.next()) * 10000 / 255,
		ScaleBP:     int64(b.next()) * 100,
		Horizon:     1 + int64(b.next()%50),
		Seasons:     []Run{{Length: 1 + int64(b.next()%5), FactorBP: int64(b.next()) * 100}, {Length: 1 + int64(b.next()%5), FactorBP: int64(b.next()) * 100}},
	}
}

// FuzzSustain verifies, for any resource row of small values, that the sustainable
// draw is exact and never negative, that doubling the regeneration scale doubles
// the regeneration and not the reserve, and that a model that draws on the row
// solves, with a certificate that holds, to an outcome whose draw at lambda*
// stays within the row and equals it exactly when the row is tight.
//
// Requirement: FR-RES-039, NFR-RES-003
func FuzzSustain(f *testing.F) {
	f.Add([]byte{1, 100, 50, 60, 128, 100, 9, 3, 100, 4, 200})
	f.Add([]byte{2, 255, 255, 255, 255, 255, 49, 1, 0, 1, 0})
	f.Add([]byte{0, 0, 200, 0, 0, 100, 14, 5, 255, 1, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		e := fuzzEndow(&byteReader{data: data})
		year := int64(10)
		phi := e.Phi(year)
		if phi.Sign() < 0 {
			t.Fatalf("Phi(%d) of %+v = %v, want >= 0", year, e, phi)
		}
		doubled := e
		doubled.ScaleBP *= 2
		want := new(big.Rat).Mul(phi, big.NewRat(2, 1))
		if e.Shape == Constant {
			reserve := big.NewRat(e.Stock, e.Horizon*year)
			want.Sub(want, reserve)
		}
		if got := doubled.Phi(year); got.Cmp(want) != 0 {
			t.Fatalf("Phi(%d) at twice the scale of %+v = %v, want %v", year, e, got, want)
		}
		m := fuzzModel(e)
		out, err := SolveModel(context.Background(), m, 10000)
		if errors.Is(err, ErrInconsistent) {
			t.Fatalf("SolveModel(%+v) error = %v, want a certified outcome or a data error", e, err)
		}
		// The draw at lambda* is the draw at lambda 1 times lambda*, and it is the
		// sustainable draw exactly when the row is tight.
		if err == nil && out.Lambda.Num.Sign() > 0 {
			atOptimum := out.ResDraw[0].Times(out.Lambda)
			if tight := atOptimum.Cmp(out.ResPhi[0]) == 0; tight != out.ResTight[0] || atOptimum.Cmp(out.ResPhi[0]) > 0 {
				t.Fatalf("draw at lambda* = %v, Phi = %v, tight = %v for %+v", atOptimum, out.ResPhi[0], out.ResTight[0], e)
			}
		}
	})
}

// fuzzModel is one region of 10 households needing 5 of commodity 0 a tick, with
// the activity of the resource models: duration 1, J 10, one unit out and one unit
// drawn on e.
func fuzzModel(e Endow) Model {
	return Model{
		Commodities: 1,
		Regions:     []Region{{Households: 10, Upkeep: []int64{5}}},
		Year:        10,
		Phi:         Ratio{Num: 1, Den: 1},
		Activities: []Activity{{
			Duration: 1, Starts: 10, FactorSum: 100000, Outputs: []Item{{Commodity: 0, Qty: 1}},
			Draws: []Draw{{Resource: 0, Qty: 1}}, AbundanceBP: 10000,
		}},
		Resources: []Endow{e},
	}
}
