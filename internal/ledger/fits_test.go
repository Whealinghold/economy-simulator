package ledger_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestFundFits verifies that the probe answers exactly what the real creation
// answers: it fits when Transfer and Move under Initial into a new account of the
// region would succeed, and does not when one would be refused, and it changes
// nothing. Each case runs the probe on one fixture and the real creation on a
// second, identical one.
//
// Requirement: FR-INT-020
func TestFundFits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		maxMoney int64
		region   int
		money    int64
		goods    []ledger.Fund
		want     bool
	}{
		{"nothing", testMaxMoney, 0, 0, nil, true},
		{"money and goods inside", testMaxMoney, 1, 50, []ledger.Fund{{Commodity: 0, Qty: 5}}, true},
		{"money at the bound", testMaxMoney, 0, testMaxMoney, nil, true},
		{"money above the bound", testMaxMoney, 0, testMaxMoney + 1, nil, false},
		{"goods at the bound", testMaxMoney, 0, 0, []ledger.Fund{{Commodity: 0, Qty: testMaxQuantity}}, true},
		{"goods above the bound", testMaxMoney, 0, 0, []ledger.Fund{{Commodity: 0, Qty: testMaxQuantity + 1}}, false},
		{"boundary would pass int64", math.MaxInt64, 0, math.MaxInt64 - 50, nil, false},
		{"region below range", testMaxMoney, -1, 1, nil, false},
		{"region above range", testMaxMoney, 2, 1, nil, false},
		{"commodity out of range", testMaxMoney, 0, 0, []ledger.Fund{{Commodity: 7, Qty: 1}}, false},
		{"commodity with no boundary", testMaxMoney, 0, 0, []ledger.Fund{{Commodity: 1, Qty: 1}}, false},
		{"amounts below 1 are skipped", testMaxMoney, 0, -3, []ledger.Fund{{Commodity: 7, Qty: 0}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, real := newFix(t, tt.maxMoney), newFix(t, tt.maxMoney)
			before := snapshot(f.l)
			if got := f.l.FundFits(tt.region, tt.money, tt.goods); got != tt.want {
				t.Errorf("FundFits(%d, %d, %v) = %v, want %v", tt.region, tt.money, tt.goods, got, tt.want)
			}
			if !reflect.DeepEqual(snapshot(f.l), before) {
				t.Errorf("FundFits changed the ledger")
			}
			// The oracle: the creation the probe stands for.
			ok := tt.region >= 0 && tt.region < 2
			if ok {
				acct, err := real.l.Open(ledger.KindAgent, 99, tt.region)
				if err != nil {
					t.Fatalf("Open() = %v, want nil", err)
				}
				if tt.money >= 1 {
					ok = real.l.Transfer(real.mint, acct, tt.money, ledger.Initial) == nil
				}
				for _, g := range tt.goods {
					if g.Qty >= 1 && ok {
						ok = real.l.Move(real.goodsBoundary, acct, g.Commodity, g.Qty, ledger.Initial) == nil
					}
				}
			}
			if ok != tt.want {
				t.Errorf("the real creation succeeded = %v, want %v (the probe's answer)", ok, tt.want)
			}
		})
	}
}
