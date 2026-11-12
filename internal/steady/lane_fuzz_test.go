package steady

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// laneCase decodes a small model of 2 or 3 regions and 1 or 2 commodities with a
// farm in each region and up to 4 lanes, valid by construction.
func laneCase(b *byteReader) Model {
	nr, nc := 2+int(b.next()%2), 1+int(b.next()%2)
	m := Model{Commodities: nc, Year: 365, Phi: Ratio{Num: 1, Den: 1}}
	for r := 0; r < nr; r++ {
		up := make([]int64, nc)
		for c := range up {
			up[c] = int64(b.next() % 4)
		}
		m.Regions = append(m.Regions, Region{Households: 1 + int64(b.next()%20), Upkeep: up})
		m.Activities = append(m.Activities, Activity{
			Region: r, Duration: 1, Starts: 365, FactorSum: 365 * (1 + int64(b.next())*40),
			Outputs: []Item{{Commodity: int(b.next()) % nc, Qty: 1 + int64(b.next()%5)}},
		})
	}
	for n := int(b.next() % 5); n > 0; n-- {
		from, to := int(b.next())%nr, int(b.next())%nr
		if from == to {
			continue
		}
		m.Lanes = append(m.Lanes, Lane{From: from, To: to, Commodity: int(b.next()) % nc, Labour: int64(b.next()) * 40, LossBP: int64(b.next()) * 39})
	}
	return m
}

// FuzzSteadyLanes verifies, over small random models with lanes, that building
// and solving never panics, that the only failures are an unbounded or an
// infeasible programme, that a result passes the certificate (Solve returns
// ErrInconsistent otherwise) and that solving again gives the same result.
//
// Requirement: NFR-VIA-004, FR-VIA-043
func FuzzSteadyLanes(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 0, 3, 5, 9, 4, 20, 7, 2, 30, 4, 1, 2, 0, 10, 100, 50})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		m := laneCase(&byteReader{data: data})
		o, err := SolveModel(context.Background(), m, 100_000)
		switch {
		case errors.Is(err, ErrUnbounded), errors.Is(err, ErrInfeasible):
			return
		case err != nil:
			t.Fatalf("SolveModel(%+v) error = %v, want nil, ErrUnbounded or ErrInfeasible", m, err)
		}
		again, err := SolveModel(context.Background(), m, 100_000)
		if err != nil {
			t.Fatalf("SolveModel(%+v) again error = %v, want nil", m, err)
		}
		if a, b := fmt.Sprint(o.Lambda, o.Price, o.Wage, o.Flow), fmt.Sprint(again.Lambda, again.Price, again.Wage, again.Flow); a != b {
			t.Fatalf("SolveModel(%+v) twice = %s then %s", m, a, b)
		}
	})
}
