package economysim

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// fiscalLedgerConfig is a canonical configuration with two commodities, two regions, one
// money flow and the fiscal module with a tax, a duty, a toll, a money payout, an
// in-kind payout and a purchase. The module is not validated here: the ledger reads the
// flow lists alone.
func fiscalLedgerConfig(on bool) Config {
	c := Config{
		World:       DefaultWorldParams(),
		Calendar:    DefaultCalendar(),
		Commodities: []Commodity{{ID: "grain", ReferencePrice: 10}, {ID: "tool", ReferencePrice: 20}},
		Regions:     []Region{{ID: "a"}, {ID: "b"}},
	}
	c.Modules.MoneySupply = &MoneySupply{Flows: []MoneyFlow{{ID: "mint", Kind: FlowGrant}}}
	if on {
		c.Modules.Fiscal = &Fiscal{
			Taxes:     []Tax{{ID: "poll"}},
			Duties:    []Duty{{ID: "sales"}},
			Tolls:     []Toll{{ID: "bridge"}},
			Payouts:   []Payout{{ID: "alms"}, {ID: "dole", Commodity: "grain"}},
			Purchases: []Purchase{{ID: "buy"}},
		}
	}
	return c.canonical()
}

// TestFiscal_ReasonsInternal verifies that the fiscal module declares one Internal reason
// for each tax, duty, toll and payout (a purchase and a release use trade), in the order
// taxes, duties, tolls, payouts, with an in-kind payout moving goods; and that the
// reasons open no boundary account, so no account ID shifts.
//
// Requirement: FR-FIS-002
func TestFiscal_ReasonsInternal(t *testing.T) {
	t.Parallel()
	off, err := newLedger(fiscalLedgerConfig(false))
	if err != nil {
		t.Fatalf("newLedger(module off) error = %v, want nil", err)
	}
	cfg := fiscalLedgerConfig(true)
	on, err := newLedger(cfg)
	if err != nil {
		t.Fatalf("newLedger(module on) error = %v, want nil", err)
	}
	if on.Len() != off.Len() {
		t.Errorf("accounts with the module on = %d, off = %d, want equal: an internal reason opens no boundary account", on.Len(), off.Len())
	}
	if got := fiscalReasons(&cfg); len(got) != 5 || got[3].Goods || !got[4].Goods {
		t.Errorf("fiscalReasons = %+v, want 5 flows: tax, duty, toll, money payout, in-kind payout", got)
	}
	if fiscalReasons(&Config{}) != nil {
		t.Error("fiscalReasons(module off) is not nil")
	}
}

// TestFiscal_ReasonsMoveBetweenSectors verifies that a money reason moves money between
// agent and treasury accounts, that an in-kind payout moves no money, that a money flow
// of the money supply stays a faucet, that trade runs between a treasury and an agent
// (FR-FIS-052) and that the audit passes after the moves.
//
// Requirement: FR-FIS-002, FR-FIS-052
func TestFiscal_ReasonsMoveBetweenSectors(t *testing.T) {
	t.Parallel()
	on, err := newLedger(fiscalLedgerConfig(true))
	if err != nil {
		t.Fatalf("newLedger(module on) error = %v, want nil", err)
	}
	agent, err := on.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open(agent) error = %v", err)
	}
	treasury, err := on.Open(ledger.KindTreasury, 1, 1)
	if err != nil {
		t.Fatalf("Open(treasury) error = %v", err)
	}
	mint, ok := on.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("Boundary(initial, money) is not open")
	}
	if err := on.Transfer(mint, agent, 100, ledger.Initial); err != nil {
		t.Fatalf("Transfer(initial) error = %v", err)
	}
	// The money flow "mint" is reason 0 of the flows; the fiscal flows follow it.
	tax, payout, inKind := ledger.FlowReason(1), ledger.FlowReason(4), ledger.FlowReason(5)
	// In this order: the payout needs what the tax collected.
	for _, m := range []struct {
		name     string
		from, to int
		amount   int64
		reason   ledger.Reason
		wantErr  error
	}{
		{"tax", agent, treasury, 30, tax, nil},
		{"payout", treasury, agent, 10, payout, nil},
		{"trade", agent, treasury, 1, ledger.Trade, nil},
		{"money under the in-kind payout", treasury, agent, 1, inKind, ledger.ErrReasonMismatch},
		{"money under the money flow", agent, treasury, 1, ledger.FlowReason(0), ledger.ErrReasonMismatch},
	} {
		if err := on.Transfer(m.from, m.to, m.amount, m.reason); !errors.Is(err, m.wantErr) {
			t.Errorf("Transfer of %s error = %v, want %v", m.name, err, m.wantErr)
		}
	}
	if err := on.Audit(); err != nil {
		t.Errorf("Audit() error = %v, want nil: the moves conserve money", err)
	}
}
