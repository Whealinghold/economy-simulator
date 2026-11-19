package ledger_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// fiscalLedger builds a ledger with one money flow (a faucet), a money fiscal flow
// and a goods fiscal flow, and opens the accounts the tests use: an agent with 100
// money and 5 units of commodity 0, an estate and a treasury.
func fiscalLedger(t *testing.T) (l *ledger.Ledger, agent, estate, treasury int) {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}, {}},
		Regions:     2,
		MaxMoney:    testMaxMoney,
		MaxQuantity: testMaxQuantity,
		Flows:       []ledger.Class{ledger.Faucet},
		Fiscal:      []ledger.FiscalFlow{{}, {Goods: true}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mint, err := l.OpenBoundary(ledger.Initial, ledger.Money)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	goods, err := l.OpenBoundary(ledger.Initial, 0)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	ids := make([]int, 3)
	for i, k := range []ledger.Kind{ledger.KindAgent, ledger.KindEstate, ledger.KindTreasury} {
		if ids[i], err = l.Open(k, i, 0); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	if err := l.Transfer(mint, ids[0], 100, ledger.Initial); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := l.Move(goods, ids[0], 0, 5, ledger.Initial); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return l, ids[0], ids[1], ids[2]
}

// TestFiscalFlows_InternalRows verifies that a fiscal flow is an Internal reason
// numbered after the money flows, that a money flow moves money between agent, estate
// and treasury accounts and a goods flow moves goods, that neither moves the other
// thing nor touches a boundary account, and that the totals reach AppendFlows under
// the flow's reason.
//
// Requirement: FR-FIS-002
func TestFiscalFlows_InternalRows(t *testing.T) {
	t.Parallel()
	l, agent, estate, treasury := fiscalLedger(t)
	tax, kind := ledger.FlowReason(1), ledger.FlowReason(2)
	other, err := l.Open(ledger.KindTreasury, 3, 0)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, tt := range []struct {
		name   string
		move   func() error
		wantOK bool
	}{
		{"money tax agent to treasury", func() error { return l.Transfer(agent, treasury, 30, tax) }, true},
		{"money remittance treasury to treasury", func() error { return l.Transfer(treasury, other, 1, tax) }, true},
		{"money payout treasury to estate", func() error { return l.Transfer(treasury, estate, 10, tax) }, true},
		{"money flow moves no goods", func() error { return l.Move(agent, treasury, 0, 1, tax) }, false},
		{"goods in kind treasury", func() error { return l.Move(agent, treasury, 0, 2, kind) }, true},
		{"goods flow moves no money", func() error { return l.Transfer(agent, treasury, 1, kind) }, false},
		{"the money flow of the money supply stays a faucet", func() error { return l.Transfer(agent, treasury, 1, ledger.FlowReason(0)) }, false},
	} {
		err := tt.move()
		if tt.wantOK && err != nil || !tt.wantOK && !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("%s: error = %v, want ok = %v (or ErrReasonMismatch)", tt.name, err, tt.wantOK)
		}
	}
	got := map[ledger.Reason][2]int64{}
	for _, f := range l.AppendFlows(nil, true) {
		if f.Reason >= ledger.NumReasons {
			v := got[f.Reason]
			if f.Commodity == ledger.Money {
				v[0] += f.Total
			} else {
				v[1] += f.Total
			}
			got[f.Reason] = v
		}
	}
	if want := (map[ledger.Reason][2]int64{tax: {41, 0}, kind: {0, 2}}); len(got) != len(want) || got[tax] != want[tax] || got[kind] != want[kind] {
		t.Errorf("AppendFlows totals by flow reason = %v, want %v (tax 30+1+10 is 41 only in its regions, counted per region of the debited account)", got, want)
	}
}

// TestFiscalFlows_TooMany verifies that the money flows and the fiscal flows together
// must fit the reason numbers.
//
// Requirement: FR-FIS-004
func TestFiscalFlows_TooMany(t *testing.T) {
	t.Parallel()
	_, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 10, MaxQuantity: 10,
		Flows:  make([]ledger.Class, 100),
		Fiscal: make([]ledger.FiscalFlow, 200),
	})
	if !errors.Is(err, ledger.ErrInvalidConfig) {
		t.Errorf("New with 300 flows error = %v, want ErrInvalidConfig", err)
	}
}
