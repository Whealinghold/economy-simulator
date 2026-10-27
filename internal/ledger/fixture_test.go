package ledger_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// fix is a small ledger of the tests: two regions, two commodities, and one
// account of each shape the money rules distinguish.
type fix struct {
	l *ledger.Ledger
	// mint is the boundary account of Initial money.
	mint int
	// a0 and a1 are agents of region 0 and 1, e0 an estate of region 0, and
	// goodsBoundary the boundary account of Initial for commodity 0.
	a0, a1, e0, goodsBoundary int
}

const (
	testMaxMoney    = 1000
	testMaxQuantity = 100
)

// newFix builds the fixture with the given money bound. a0 holds 100 and the
// rest nothing, so the boundary is at -100.
func newFix(t *testing.T, maxMoney int64) *fix {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{ShelfLife: 3}, {Durability: 5}},
		Regions:     2,
		MaxMoney:    maxMoney,
		MaxQuantity: testMaxQuantity,
		Declared:    []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest, ledger.Relief},
		MaxClaims:   4,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f := &fix{l: l}
	open := func(id int, err error) int {
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		return id
	}
	f.mint = open(l.OpenBoundary(ledger.Initial, ledger.Money))
	f.goodsBoundary = open(l.OpenBoundary(ledger.Initial, 0))
	f.a0 = open(l.Open(ledger.KindAgent, 7, 0))
	f.a1 = open(l.Open(ledger.KindAgent, 8, 1))
	f.e0 = open(l.Open(ledger.KindEstate, 0, 0))
	if err := l.Transfer(f.mint, f.a0, 100, ledger.Initial); err != nil {
		t.Fatalf("setup: fund a0: %v", err)
	}
	return f
}

// state is everything observable about a ledger, for comparing a ledger with
// its earlier self with reflect.DeepEqual.
type state struct {
	Accounts       []ledger.View
	Totals, Bucket []ledger.Flow
	Construction   bool
	Claims         []ledger.Claim
	ClaimTotals    ledger.ClaimTotals
}

func snapshot(l *ledger.Ledger) state {
	s := state{
		Totals:       l.AppendFlows(nil, false),
		Bucket:       l.AppendFlows(nil, true),
		Construction: l.InConstruction(),
		Claims:       l.AppendClaims(nil),
		ClaimTotals:  l.ClaimTotals(),
	}
	for id := 0; id < l.Len(); id++ {
		v, err := l.Account(id)
		if err != nil {
			panic(err)
		}
		s.Accounts = append(s.Accounts, v)
	}
	return s
}

// sum returns the money total over every account, boundaries included.
func sum(l *ledger.Ledger) int64 {
	var total int64
	for id := 0; id < l.Len(); id++ {
		v, err := l.Account(id)
		if err != nil {
			panic(err)
		}
		total += v.Balance
	}
	return total
}
