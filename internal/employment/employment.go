// Package employment is the arithmetic of the payroll form of the employment
// module (spec-employment.md 24): the budget of a payer and the split of the
// pool among the workers. It works on primitive structs and knows no role, agent
// or region; the root converts at the call (architecture.md 2.9). It takes no
// random source and keeps no state.
package employment

import "github.com/Whealinghold/economy-sim/internal/checked"

// WholeBP is 100% in basis points.
const WholeBP = 10_000

// Payer is an agent of an employer role: its receipts of the tick from trades,
// its PayrollBP and its free balance. Ref is returned in each Transfer.
type Payer struct {
	Ref      int
	Receipts int64
	BP       int64
	Balance  int64
}

// Worker is an agent of a Hireable role that worked in the tick, with the
// executions of its job (K at least 1). Ref is returned in each Transfer.
type Worker struct {
	Ref int
	K   int64
}

// Transfer is one row of the payroll: Amount (at least 1) from the payer with
// the Ref Payer to the worker with the Ref Worker.
type Transfer struct {
	Payer, Worker int
	Amount        int64
}

// Budget returns the due and the budget of a payer (spec-employment.md 24.1.3):
// due = floor(Receipts * BP / 10 000) and budget = min(due, Balance), a balance
// below 0 counting as 0. It reports false when an input is negative or BP is
// above 10 000, so a caller that validated its configuration never sees it.
func Budget(p Payer) (due, budget int64, ok bool) {
	if p.Receipts < 0 || p.BP < 0 || p.BP > WholeBP {
		return 0, 0, false
	}
	due, ok = checked.MulDivFloor(p.Receipts, p.BP, WholeBP)
	if !ok {
		return 0, 0, false
	}
	return due, min(due, max(p.Balance, 0)), true
}

// Split appends to dst the transfers of the payroll of one region
// (spec-employment.md 24.1.4, 24.1.5) and returns it. The payers and the
// workers must be in ascending agent ID, which the caller guarantees: the order
// is the tie-break, so Split sorts nothing.
//
// With T the sum of the budgets and K the sum of the executions, nothing is
// paid when either is 0. Otherwise worker j receives
// floor(T * C_j / K) - floor(T * C_{j-1} / K), C_j the running sum of the
// executions, so the amounts sum to T exactly; they are funded by the payers in
// order, each used up before the next. It reports false, with dst unchanged, on
// a negative input or an overflow.
func Split(dst []Transfer, payers []Payer, workers []Worker) ([]Transfer, bool) {
	pool, execs, ok := totals(payers, workers)
	if !ok {
		return dst, false
	}
	if pool == 0 || execs == 0 {
		return dst, true
	}
	n := len(dst)
	f := funder{payers: payers}
	var run, paidBefore int64
	for _, w := range workers {
		run += w.K // cannot overflow: the sum of the K was checked by totals
		upTo, ok := checked.MulDivFloor(pool, run, execs)
		if !ok {
			return dst[:n], false
		}
		if amount := upTo - paidBefore; amount > 0 {
			if dst, ok = f.pay(dst, w.Ref, amount); !ok {
				return dst[:n], false
			}
		}
		paidBefore = upTo
	}
	return dst, true
}

// totals returns the pool (the sum of the budgets) and the sum of the
// executions, and false on a negative input or an overflow.
func totals(payers []Payer, workers []Worker) (pool, execs int64, ok bool) {
	for _, p := range payers {
		_, b, ok := Budget(p)
		if !ok {
			return 0, 0, false
		}
		if pool, ok = checked.Add(pool, b); !ok {
			return 0, 0, false
		}
	}
	for _, w := range workers {
		if w.K < 1 {
			return 0, 0, false
		}
		var ok bool
		if execs, ok = checked.Add(execs, w.K); !ok {
			return 0, 0, false
		}
	}
	return pool, execs, true
}

// funder walks the payers in order, each used up before the next.
type funder struct {
	payers []Payer
	next   int   // the payer being drawn on
	used   int64 // what it has already paid
}

// pay appends the rows that fund amount to worker, drawing on the payers from
// where the last call stopped. It reports false if the budgets run out, which
// cannot happen when the amounts sum to the pool.
func (f *funder) pay(dst []Transfer, worker int, amount int64) ([]Transfer, bool) {
	for amount > 0 {
		for f.next < len(f.payers) {
			_, b, _ := Budget(f.payers[f.next]) // validated by Split
			if f.used < b {
				break
			}
			f.next, f.used = f.next+1, 0
		}
		if f.next == len(f.payers) {
			return dst, false
		}
		_, b, _ := Budget(f.payers[f.next])
		take := min(amount, b-f.used)
		dst = append(dst, Transfer{Payer: f.payers[f.next].Ref, Worker: worker, Amount: take})
		f.used += take
		amount -= take
	}
	return dst, true
}
