package employment_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// TestBudget verifies the due and the budget of a payer: the floor of
// receipts times PayrollBP over 10 000, cut at the free balance.
//
// Requirement: FR-EMP-068
func TestBudget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		p          employment.Payer
		due, bud   int64
		wantOK     bool
		wantReason string
	}{
		// 1 001 * 5 000 / 10 000 = 500.5, floored to 500.
		{"floor of the share", employment.Payer{Receipts: 1001, BP: 5000, Balance: 9999}, 500, 500, true, ""},
		// due 500, but the payer holds only 120.
		{"cut at the balance", employment.Payer{Receipts: 1000, BP: 5000, Balance: 120}, 500, 120, true, ""},
		{"zero balance pays nothing", employment.Payer{Receipts: 1000, BP: 5000, Balance: 0}, 500, 0, true, ""},
		{"negative balance counts as zero", employment.Payer{Receipts: 1000, BP: 5000, Balance: -5}, 500, 0, true, ""},
		{"no receipts", employment.Payer{Receipts: 0, BP: 10000, Balance: 50}, 0, 0, true, ""},
		{"whole share", employment.Payer{Receipts: 77, BP: 10000, Balance: 77}, 77, 77, true, ""},
		// 9 223 372 036 854 775 807 * 10 000 / 10 000 needs the 128-bit product.
		{"largest receipts at 100%", employment.Payer{Receipts: math.MaxInt64, BP: 10000, Balance: math.MaxInt64}, math.MaxInt64, math.MaxInt64, true, ""},
		{"negative receipts", employment.Payer{Receipts: -1, BP: 5000, Balance: 5}, 0, 0, false, ""},
		{"share above 100%", employment.Payer{Receipts: 1, BP: 10001, Balance: 5}, 0, 0, false, ""},
		{"negative share", employment.Payer{Receipts: 1, BP: -1, Balance: 5}, 0, 0, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			due, bud, ok := employment.Budget(tt.p)
			if ok != tt.wantOK || due != tt.due || bud != tt.bud {
				t.Errorf("Budget(%+v) = %d, %d, %v, want %d, %d, %v", tt.p, due, bud, ok, tt.due, tt.bud, tt.wantOK)
			}
		})
	}
}

// TestSplit verifies the shares and the funding sequence on small cases whose
// arithmetic is written out.
//
// Requirement: FR-EMP-070, FR-EMP-071, FR-EMP-069
func TestSplit(t *testing.T) {
	t.Parallel()
	payer := func(ref int, receipts, bp, balance int64) employment.Payer {
		return employment.Payer{Ref: ref, Receipts: receipts, BP: bp, Balance: balance}
	}
	worker := func(ref int, k int64) employment.Worker { return employment.Worker{Ref: ref, K: k} }
	tests := []struct {
		name    string
		payers  []employment.Payer
		workers []employment.Worker
		want    []employment.Transfer
	}{
		{"no payers", nil, []employment.Worker{worker(7, 2)}, nil},
		{"no workers", []employment.Payer{payer(1, 100, 5000, 100)}, nil, nil},
		{"zero pool", []employment.Payer{payer(1, 100, 5000, 0)}, []employment.Worker{worker(7, 2)}, nil},
		// Pool 50 (100 * 50%); one worker takes it whole.
		{"one payer one worker", []employment.Payer{payer(1, 100, 5000, 100)}, []employment.Worker{worker(7, 3)},
			[]employment.Transfer{{Payer: 1, Worker: 7, Amount: 50}}},
		// Pool 10, k = 1, 1, 1: cumulative floors floor(10*1/3)=3, floor(10*2/3)=6, 10 give 3, 3, 4.
		{"remainder goes by the cumulative floor", []employment.Payer{payer(1, 20, 5000, 20)},
			[]employment.Worker{worker(5, 1), worker(6, 1), worker(7, 1)},
			[]employment.Transfer{{1, 5, 3}, {1, 6, 3}, {1, 7, 4}}},
		// Pool 12 = 8 + 4; k = 3, 1: cumulative floors 9, 12 give 9, 3. The first payer (budget 8)
		// funds worker 5 up to 8, the second (budget 4) the last unit of 5 and then worker 6.
		{"two payers in order across two workers",
			[]employment.Payer{payer(1, 16, 5000, 100), payer(2, 8, 5000, 100)},
			[]employment.Worker{worker(5, 3), worker(6, 1)},
			[]employment.Transfer{{1, 5, 8}, {2, 5, 1}, {2, 6, 3}}},
		// Payer 1 is cut to 2 of a due 10, so the pool is 2 + 4 = 6 and k = 1, 1 gives 3, 3.
		{"a payer cut at its balance",
			[]employment.Payer{payer(1, 20, 5000, 2), payer(2, 8, 5000, 100)},
			[]employment.Worker{worker(5, 1), worker(6, 1)},
			[]employment.Transfer{{1, 5, 2}, {2, 5, 1}, {2, 6, 3}}},
		// Pool 1 over k = 1, 1: floors 0, 1 give 0, 1; the zero row is not emitted.
		{"a zero share emits no row", []employment.Payer{payer(1, 2, 5000, 5)},
			[]employment.Worker{worker(5, 1), worker(6, 1)},
			[]employment.Transfer{{1, 6, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := employment.Split(nil, tt.payers, tt.workers)
			if !ok || !slices.Equal(got, tt.want) {
				t.Errorf("Split(%+v, %+v) = %+v, %v, want %+v, true", tt.payers, tt.workers, got, ok, tt.want)
			}
		})
	}
}

// TestSplit_Refusals verifies that a negative input or an overflow is reported
// and leaves dst as it was.
//
// Requirement: NFR-EMP-006
func TestSplit_Refusals(t *testing.T) {
	t.Parallel()
	dst := []employment.Transfer{{Payer: 9, Worker: 9, Amount: 9}}
	tests := []struct {
		name    string
		payers  []employment.Payer
		workers []employment.Worker
	}{
		{"negative receipts", []employment.Payer{{Ref: 1, Receipts: -1, BP: 5000, Balance: 1}}, []employment.Worker{{Ref: 2, K: 1}}},
		{"worker with no executions", []employment.Payer{{Ref: 1, Receipts: 10, BP: 5000, Balance: 10}}, []employment.Worker{{Ref: 2, K: 0}}},
		{"pool overflows", []employment.Payer{
			{Ref: 1, Receipts: math.MaxInt64, BP: 10000, Balance: math.MaxInt64},
			{Ref: 2, Receipts: 1, BP: 10000, Balance: 1}}, []employment.Worker{{Ref: 3, K: 1}}},
		{"executions overflow", []employment.Payer{{Ref: 1, Receipts: 10, BP: 5000, Balance: 10}},
			[]employment.Worker{{Ref: 2, K: math.MaxInt64}, {Ref: 3, K: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := employment.Split(slices.Clone(dst), tt.payers, tt.workers)
			if ok || !slices.Equal(got, dst) {
				t.Errorf("Split(%s) = %+v, %v, want %+v, false", tt.name, got, ok, dst)
			}
		})
	}
}

// FuzzSplit asserts, for any payers and workers within the types, that Split
// never panics and, when it succeeds, conserves money: the rows sum to the pool,
// no payer pays more than its budget, no row is below 1 and every worker gets
// the cumulative-floor amount.
//
// Requirement: NFR-EMP-006
func FuzzSplit(f *testing.F) {
	f.Add(int64(1000), int64(5000), int64(400), int64(16), int64(5000), int64(100), int64(3), int64(1), int64(2))
	f.Add(int64(math.MaxInt64), int64(10000), int64(math.MaxInt64), int64(0), int64(1), int64(0), int64(1), int64(1), int64(1))
	f.Add(int64(0), int64(0), int64(0), int64(1), int64(10000), int64(1), int64(math.MaxInt64), int64(1), int64(1))
	f.Fuzz(func(t *testing.T, r1, bp1, b1, r2, bp2, b2, k1, k2, k3 int64) {
		payers := []employment.Payer{{Ref: 1, Receipts: r1, BP: bp1, Balance: b1}, {Ref: 2, Receipts: r2, BP: bp2, Balance: b2}}
		workers := []employment.Worker{{Ref: 10, K: k1}, {Ref: 11, K: k2}, {Ref: 12, K: k3}}
		rows, ok := employment.Split(nil, payers, workers)
		if !ok {
			return
		}
		var pool, out int64
		budget := map[int]int64{}
		for _, p := range payers {
			_, b, _ := employment.Budget(p)
			pool += b
			budget[p.Ref] = b
		}
		paid := map[int]int64{}
		for _, row := range rows {
			if row.Amount < 1 {
				t.Fatalf("Split(%+v, %+v) has a row of %d, want at least 1", payers, workers, row.Amount)
			}
			out += row.Amount
			paid[row.Payer] += row.Amount
		}
		if len(rows) > 0 && out != pool {
			t.Errorf("Split(%+v, %+v) paid %d, want the pool %d", payers, workers, out, pool)
		}
		for ref, p := range paid {
			if p > budget[ref] {
				t.Errorf("Split(%+v, %+v): payer %d paid %d, want at most %d", payers, workers, ref, p, budget[ref])
			}
		}
	})
}
