package ledger

import (
	"reflect"
	"testing"
)

// fiscalRig builds a one-region ledger with one faucet flow and two fiscal flows (money,
// then goods) and opens the three boundary accounts the setup draws on.
func fiscalRig(t *testing.T, fiscal []FiscalFlow) (l *Ledger, money, goods int) {
	t.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 1000, MaxQuantity: 100,
		Flows: []Class{Faucet}, Fiscal: fiscal,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := l.OpenBoundary(FlowReason(0), Money); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if money, err = l.OpenBoundary(Initial, Money); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if goods, err = l.OpenBoundary(Initial, 0); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return l, money, goods
}

// fiscalRun funds an agent, moves money and goods under the two fiscal reasons in
// construction, ends construction and moves both again in tick 1.
func fiscalRun(t *testing.T, l *Ledger, money, goods int) {
	t.Helper()
	agent, err := l.Open(KindAgent, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	treasury, err := l.Open(KindTreasury, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	steps := []func() error{
		func() error { return l.Transfer(money, agent, 100, Initial) },
		func() error { return l.Move(goods, agent, 0, 5, Initial) },
		func() error { return l.Transfer(agent, treasury, 30, FlowReason(1)) },
		func() error { return l.Move(agent, treasury, 0, 2, FlowReason(2)) },
		l.EndConstruction,
		func() error { l.AdvanceTick(); return l.Transfer(agent, treasury, 7, FlowReason(1)) },
		func() error { return l.Move(agent, treasury, 0, 1, FlowReason(2)) },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("setup step %d: %v", i, err)
		}
	}
}

// TestFiscalFlows_CheckpointByPosition verifies that the totals of a fiscal flow reason
// are saved under the position of the reason after the money flows and restored into a
// ledger of the same configuration, with the money total of a tax and the goods total of
// an in-kind payout.
//
// Requirement: FR-FIS-002, FR-FIS-060
//
// Guard: a fiscal reason saved by its raw number would break the restore of a
// configuration that gains a built-in reason; saved by position it does not.
func TestFiscalFlows_CheckpointByPosition(t *testing.T) {
	t.Parallel()
	fiscal := []FiscalFlow{{}, {Goods: true}}
	l, money, goods := fiscalRig(t, fiscal)
	fiscalRun(t, l, money, goods)
	fresh, _, _ := fiscalRig(t, fiscal) // a restore opens the accounts itself
	if err := read(fresh, saved(l), l.Tick(), false); err != nil {
		t.Fatalf("ReadState(same configuration) error = %v, want nil", err)
	}
	if want, got := l.AppendFlows(nil, false), fresh.AppendFlows(nil, false); len(want) != 2 || !reflect.DeepEqual(got, want) {
		t.Errorf("AppendFlows after the restore = %v, want %v (the tax's 7 and the goods' 1)", got, want)
	}
}

// TestFiscalFlows_RestoreRefusesFewerFlows verifies that a ledger configured with fewer
// fiscal flows than the saved one refuses the section: the goods flow's position does not
// exist there.
//
// Requirement: FR-FIS-060
func TestFiscalFlows_RestoreRefusesFewerFlows(t *testing.T) {
	t.Parallel()
	l, money, goods := fiscalRig(t, []FiscalFlow{{}, {Goods: true}})
	fiscalRun(t, l, money, goods)
	other, _, _ := fiscalRig(t, []FiscalFlow{{}})
	if err := read(other, saved(l), l.Tick(), false); err == nil {
		t.Error("ReadState(a ledger with fewer fiscal flows) error = nil, want a refusal")
	}
}
