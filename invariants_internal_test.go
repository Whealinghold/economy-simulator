package economysim

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the one place the invariants test (phase 5 of
// doc/plans/plan-phase-1-12.md, D179) reads unexported state: the snapshot holds
// sums, not accounts, so the scan for a negative balance or stock needs the
// ledger's accessors.

// accountScan is the part of *ledger.Ledger the scan reads.
type accountScan interface {
	Len() int
	Account(id int) (ledger.View, error)
	Holding(id, commodity int) (held, reserved int64, err error)
}

// negativeAccounts lists every open non-boundary account of the world that
// holds a negative balance, reservation or stock. Boundary accounts are
// skipped: a faucet's balance is legitimately negative.
func negativeAccounts(w *World) []string { return scanNegative(w.ledger, len(w.cfg.Commodities)) }

// scanNegative is negativeAccounts on the accessors, so it can be tested on a
// fake that holds a state the real ledger refuses to reach.
func scanNegative(l accountScan, commodities int) []string {
	var bad []string
	for id := 0; id < l.Len(); id++ {
		v, err := l.Account(id)
		if err != nil {
			bad = append(bad, fmt.Sprintf("account %d unreadable: %v", id, err))
			continue
		}
		if v.Closed || v.Kind == ledger.KindBoundary {
			continue
		}
		if v.Balance < 0 || v.Reserved < 0 {
			bad = append(bad, fmt.Sprintf("account %d: money balance %d, reserved %d", id, v.Balance, v.Reserved))
		}
		for c := 0; c < commodities; c++ {
			held, reserved, err := l.Holding(id, c)
			if err != nil {
				bad = append(bad, fmt.Sprintf("account %d commodity %d unreadable: %v", id, c, err))
				continue
			}
			if held < 0 || reserved < 0 || reserved > held {
				bad = append(bad, fmt.Sprintf("account %d commodity %d: held %d, earmarked %d", id, c, held, reserved))
			}
		}
	}
	return bad
}

// fakeScan is a ledger with the accounts and holdings given.
type fakeScan struct {
	views []ledger.View
	held  map[[2]int][2]int64
	err   error
}

func (f fakeScan) Len() int { return len(f.views) }
func (f fakeScan) Account(id int) (ledger.View, error) {
	return f.views[id], f.err
}
func (f fakeScan) Holding(id, c int) (int64, int64, error) {
	h := f.held[[2]int{id, c}]
	return h[0], h[1], nil
}

// TestScanNegative_Accounts verifies the scan reports a negative balance, a
// negative or over-large earmark and an unreadable account, and skips boundary
// and closed accounts.
//
// Requirement: FR-REF-028, FR-STB-008, FR-LED-035
// Guard: a scan that tested boundary accounts would fail every faucet.
func TestScanNegative_Accounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		f    fakeScan
		want int
	}{
		{"clean", fakeScan{views: []ledger.View{{Kind: 0, Balance: 5}}}, 0},
		{"boundary negative skipped", fakeScan{views: []ledger.View{{Kind: ledger.KindBoundary, Balance: -9}}}, 0},
		{"closed negative skipped", fakeScan{views: []ledger.View{{Kind: 0, Closed: true, Balance: -1}}}, 0},
		{"negative money", fakeScan{views: []ledger.View{{Kind: 0, Balance: -1}}}, 1},
		{"negative reservation", fakeScan{views: []ledger.View{{Kind: 0, Reserved: -1}}}, 1},
		{"negative stock", fakeScan{views: []ledger.View{{Kind: 0}}, held: map[[2]int][2]int64{{0, 1}: {-1, 0}}}, 1},
		{"earmark above held", fakeScan{views: []ledger.View{{Kind: 0}}, held: map[[2]int][2]int64{{0, 0}: {1, 2}}}, 1},
		{"unreadable", fakeScan{views: []ledger.View{{Kind: 0}}, err: errors.New("x")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := scanNegative(tt.f, 2); len(got) != tt.want {
				t.Errorf("scanNegative = %q, want %d line(s)", got, tt.want)
			}
		})
	}
}
