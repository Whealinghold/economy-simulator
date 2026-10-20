package economysim

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// FuzzSolveSteadyState verifies, over the toy configuration with up to four
// edits from the menu of FuzzCheck and the options from the bytes, that
// SolveSteadyState never panics; that a configuration Validate rejects is an
// error of the same class; that every other result is an error from the closed
// set or a table that passes Validate with the lists in canonical order; that
// the call is deterministic and leaves its argument alone; and that it is never
// ErrSolveInconsistent, which no input can cause.
//
// Requirement: NFR-VIA-004, FR-VIA-014, FR-VIA-020, FR-VIA-030
func FuzzSolveSteadyState(f *testing.F) {
	for _, s := range [][]byte{
		nil, {2, 1}, {2, 1, 3, 1}, {5, 0, 0, 2}, {6, 50}, {7, 0}, {7, 1}, {10, 3}, {1, 0, 1, 0}, {3, 25, 2, 1},
		{8, 200, 7, 0}, {11, 0, 5, 0}, {11, 1, 2, 3}, {4, 255, 9, 5}, {0, 99, 1, 200}, {5, 0, 255, 255, 255}, {5, 0, 1, 0, 255, 0, 0, 3},
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		ops := data
		var opts []SolveOption
		if len(data) >= 5 { // a fifth byte onwards sets the options; the first four are edits
			ops = data[:4]
			opts = []SolveOption{WithStockCoverTicks(int64(data[4]) % 30), WithMoneyBufferTicks(int64(data[4]) / 3)}
			if len(data) > 5 && data[5]%2 == 0 {
				opts = append(opts, WithFreeTrade())
			}
			if len(data) > 6 {
				opts = append(opts, WithMaxPivots(1+int64(data[6])%40))
			}
		}
		build := func() Config {
			c := viaToy()
			for i := 0; i+1 < len(ops); i += 2 {
				viaPerturb(&c, ops[i], ops[i+1])
			}
			return c
		}
		c := build()
		verr := c.Validate()
		st, err := SolveSteadyState(context.Background(), c, opts...)
		if !reflect.DeepEqual(c, build()) {
			t.Fatalf("SolveSteadyState(%v) changed its argument", data)
		}
		switch {
		case verr != nil:
			if err == nil || (errors.Is(verr, ErrNotViable) && !errors.Is(err, ErrNotViable)) {
				t.Fatalf("SolveSteadyState(%v) = %v, want an error like Validate's %v", data, err, verr)
			}
		case err != nil:
			if errors.Is(err, ErrSolveInconsistent) || !(errors.Is(err, ErrNoUpkeep) || errors.Is(err, ErrSolveLimit) || errors.Is(err, ErrOverflow) ||
				errors.Is(err, ErrWarmStartPrice) || errors.Is(err, ErrWarmStartJob)) {
				t.Fatalf("SolveSteadyState(%v) = %v, want a sentinel of the closed set", data, err)
			}
			if !reflect.DeepEqual(st, SteadyState{}) {
				t.Fatalf("SolveSteadyState(%v) returned a result with %v", data, err)
			}
		default:
			c.WarmStart = &st.Table
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%v with its table) = %v, want nil (FR-VIA-030)", data, err)
			}
			if again, err := SolveSteadyState(context.Background(), build(), opts...); err != nil || !reflect.DeepEqual(again, st) {
				t.Fatalf("SolveSteadyState(%v) is not deterministic: %v", data, err)
			}
		}
	})
}
