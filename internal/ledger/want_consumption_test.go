package ledger_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestLedger_WantConsumptionRow verifies that with the reason declared,
// want_consumption is a goods sink from an agent to a boundary (class, kinds and
// no money row), that moving goods to its boundary account is totalled under it
// by region and commodity, and that every other pairing is refused; and that
// without the declaration the ledger has no row, no account and no flow for it.
//
// Requirement: FR-LED-050
func TestLedger_WantConsumptionRow(t *testing.T) {
	t.Parallel()
	build := func(t *testing.T, declared ...ledger.Reason) (l *ledger.Ledger, agent, estate, sink int) {
		t.Helper()
		l, err := ledger.New(ledger.Config{
			Commodities: []ledger.Commodity{{}, {}}, Regions: 1, MaxMoney: 1000, MaxQuantity: 100, Declared: declared,
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		initial, err := l.OpenBoundary(ledger.Initial, 1)
		if err != nil {
			t.Fatalf("OpenBoundary(Initial): %v", err)
		}
		sink = -1
		if len(declared) > 0 {
			if sink, err = l.OpenBoundary(ledger.WantConsumption, 1); err != nil {
				t.Fatalf("OpenBoundary(WantConsumption): %v", err)
			}
		}
		if agent, err = l.Open(ledger.KindAgent, 0, 0); err != nil {
			t.Fatalf("Open agent: %v", err)
		}
		if estate, err = l.Open(ledger.KindEstate, 0, 0); err != nil {
			t.Fatalf("Open estate: %v", err)
		}
		if err := l.Move(initial, agent, 1, 5, ledger.Initial); err != nil {
			t.Fatalf("fund: %v", err)
		}
		if err := l.EndConstruction(); err != nil {
			t.Fatalf("EndConstruction: %v", err)
		}
		return l, agent, estate, sink
	}

	t.Run("declared registry row", func(t *testing.T) {
		t.Parallel()
		if class, ok := ledger.ClassOf(ledger.WantConsumption, true); !ok || class != ledger.Sink {
			t.Errorf("ClassOf(WantConsumption, goods) = %v, %v, want Sink", class, ok)
		}
		if _, ok := ledger.ClassOf(ledger.WantConsumption, false); ok {
			t.Errorf("ClassOf(WantConsumption, money) has a row, want none")
		}
		for from := ledger.KindAgent; from <= ledger.KindBoundary; from++ {
			for to := ledger.KindAgent; to <= ledger.KindBoundary; to++ {
				want := from == ledger.KindAgent && to == ledger.KindBoundary
				if got := ledger.Allowed(ledger.WantConsumption, true, from, to); got != want {
					t.Errorf("Allowed(WantConsumption, goods, %d, %d) = %v, want %v", from, to, got, want)
				}
				if ledger.Allowed(ledger.WantConsumption, false, from, to) {
					t.Errorf("Allowed(WantConsumption, money, %d, %d) = true, want false", from, to)
				}
			}
		}
	})

	t.Run("goods sink to the boundary", func(t *testing.T) {
		t.Parallel()
		l, agent, estate, sink := build(t, ledger.WantConsumption)
		if err := l.Move(agent, sink, 1, 3, ledger.WantConsumption); err != nil {
			t.Fatalf("Move(agent, sink, 3, WantConsumption): %v", err)
		}
		got := l.AppendFlows(nil, false)
		want := []ledger.Flow{{Region: 0, Reason: ledger.WantConsumption, Commodity: 1, Total: 3}}
		if !slices.Equal(got, want) {
			t.Errorf("AppendFlows after the move = %+v, want %+v", got, want)
		}
		if v, _ := l.Account(sink); v.Balance != 3 {
			t.Errorf("balance of the want_consumption boundary = %d, want 3", v.Balance)
		}
		// An estate is not an agent, and the sink is not a source.
		if err := l.Move(estate, sink, 1, 1, ledger.WantConsumption); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Move(estate, sink) error = %v, want ErrReasonMismatch", err)
		}
		if err := l.Move(sink, agent, 1, 1, ledger.WantConsumption); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Move(sink, agent) error = %v, want ErrReasonMismatch", err)
		}
		if err := l.Transfer(agent, estate, 1, ledger.WantConsumption); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Transfer(WantConsumption) of money error = %v, want ErrReasonMismatch", err)
		}
	})

	t.Run("undeclared leaves no trace", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 10, MaxQuantity: 10})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, err := l.OpenBoundary(ledger.WantConsumption, 0); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("OpenBoundary(WantConsumption) undeclared error = %v, want ErrReasonMismatch", err)
		}
		if _, ok := l.Boundary(ledger.WantConsumption, 0); ok {
			t.Errorf("Boundary(WantConsumption, 0) undeclared = true, want false")
		}
		if l.Len() != 0 {
			t.Errorf("Len after the refused open = %d, want 0", l.Len())
		}
		if got := l.AppendFlows(nil, false); len(got) != 0 {
			t.Errorf("AppendFlows undeclared = %+v, want none", got)
		}
	})
}
