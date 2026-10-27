package economysim

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// declaredCfg returns a two-region configuration of c commodities with the demand
// module switched on, for the tests that build a ledger and nothing else (they
// do not validate).
func declaredCfg(c int, demand bool) Config {
	cfg := Config{Commodities: make([]Commodity, c), Regions: []Region{{ID: "r0"}, {ID: "r1"}}}
	if demand {
		cfg.Modules.Demand = &Demand{}
	}
	return cfg
}

// stage1Accounts is the creation order of the boundary accounts of the built-in
// group for c commodities (spec-ledger.md 3.4.2): initial money, then for each
// commodity the seven goods reasons.
func stage1Accounts(c int) (out []struct {
	r     ledger.Reason
	thing int
}) {
	type acct = struct {
		r     ledger.Reason
		thing int
	}
	out = append(out, acct{ledger.Initial, ledger.Money})
	for thing := 0; thing < c; thing++ {
		for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeInput, ledger.RecipeOutput, ledger.Disposal, ledger.Consumption, ledger.Spoilage, ledger.Wear} {
			out = append(out, acct{r, thing})
		}
	}
	return out
}

// TestNewLedger_OffHasNoWantAccount verifies that with the demand module off the
// ledger has the boundary accounts of Stage 1 and no others: 1 + 7C of them, at
// the IDs of spec-ledger.md 3.4.2, no want_consumption account, and the first
// account opened after them has ID 1 + 7C.
//
// Requirement: FR-DEM-001, FR-LED-050
func TestNewLedger_OffHasNoWantAccount(t *testing.T) {
	t.Parallel()
	for _, c := range []int{0, 1, 3, 9} {
		l := mustLedger(t, declaredCfg(c, false))
		want := stage1Accounts(c)
		if l.Len() != len(want) || len(want) != 1+7*c {
			t.Fatalf("newLedger(%d commodities, demand off) has %d accounts, want %d = 1 + 7*%d", c, l.Len(), len(want), c)
		}
		for id, a := range want {
			if got, ok := l.Boundary(a.r, a.thing); !ok || got != id {
				t.Errorf("Boundary(%d, %d) = %d, %v, want ID %d", a.r, a.thing, got, ok, id)
			}
		}
		for thing := ledger.Money; thing < c; thing++ {
			if id, ok := l.Boundary(ledger.WantConsumption, thing); ok {
				t.Errorf("Boundary(WantConsumption, %d) = %d with the module off, want none", thing, id)
			}
		}
		if id, err := l.Open(ledger.KindEstate, 0, 0); err != nil || id != len(want) {
			t.Errorf("first Open after the boundary accounts = %d, %v, want %d", id, err, len(want))
		}
	}
}

// TestNewLedger_WantAccountsAfterBuiltIn verifies that with the demand module on
// the want_consumption accounts, one per commodity in ascending order and none
// for money, are opened right after the 1 + 7C built-in ones and before the first
// estate or agent, and that the built-in accounts keep the IDs they have with the
// module off.
//
// Requirement: FR-LED-050, FR-LED-052
func TestNewLedger_WantAccountsAfterBuiltIn(t *testing.T) {
	t.Parallel()
	const c = 4
	on, off := mustLedger(t, declaredCfg(c, true)), mustLedger(t, declaredCfg(c, false))
	built := 1 + 7*c
	if on.Len() != built+c {
		t.Fatalf("newLedger(demand on) has %d accounts, want %d + %d", on.Len(), built, c)
	}
	for _, a := range stage1Accounts(c) {
		gotOn, _ := on.Boundary(a.r, a.thing)
		gotOff, _ := off.Boundary(a.r, a.thing)
		if gotOn != gotOff {
			t.Errorf("Boundary(%d, %d) = %d with the module on, %d off, want equal", a.r, a.thing, gotOn, gotOff)
		}
	}
	if id, ok := on.Boundary(ledger.WantConsumption, ledger.Money); ok {
		t.Errorf("Boundary(WantConsumption, Money) = %d, want none: the row is goods only", id)
	}
	for thing := 0; thing < c; thing++ {
		if id, ok := on.Boundary(ledger.WantConsumption, thing); !ok || id != built+thing {
			t.Errorf("Boundary(WantConsumption, %d) = %d, %v, want ID %d", thing, id, ok, built+thing)
		}
	}
	if id, err := on.Open(ledger.KindEstate, 0, 0); err != nil || id != built+c {
		t.Errorf("first Open after the boundary accounts = %d, %v, want %d", id, err, built+c)
	}
}

// TestNewLedger_BoundaryGroupOrder verifies the four groups of D311 in order
// (built-in, money-flow, outside, module), that opening them one after the other
// gives the same accounts as newLedger, and that the two empty hooks open nothing
// and sit between the built-in group and the module group.
//
// Guard: plans 6 and 16 fill groups 2 and 3 and must land them before the module
// rows; with the hooks empty only the order of the list pins that, and a test
// that opens group 4 straight after group 1 would pass if they were dropped.
func TestNewLedger_BoundaryGroupOrder(t *testing.T) {
	t.Parallel()
	const c = 2
	modules := moduleReasons(&Config{Modules: Modules{Demand: &Demand{}}})
	groups := boundaryGroups(c, modules, 0)
	names := []string{groups[0].name, groups[1].name, groups[2].name, groups[3].name}
	if want := []string{"built-in", "money-flow", "outside", "module"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("boundaryGroups names = %v, want %v", names, want)
	}
	cfg := declaredCfg(c, true)
	l, err := ledger.New(ledger.Config{Commodities: make([]ledger.Commodity, c), Regions: 2, MaxMoney: MaxMoney, MaxQuantity: MaxQuantity, Declared: modules[0]})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	wantLen := []int{1 + 7*c, 1 + 7*c, 1 + 7*c, 1 + 8*c}
	for i, g := range groups {
		if err := g.open(l); err != nil {
			t.Fatalf("group %q: %v", g.name, err)
		}
		if l.Len() != wantLen[i] {
			t.Errorf("after group %q the ledger has %d accounts, want %d", g.name, l.Len(), wantLen[i])
		}
	}
	ref := mustLedger(t, cfg)
	for id := 0; id < ref.Len(); id++ {
		a, _ := l.Account(id)
		b, _ := ref.Account(id)
		if a != b {
			t.Errorf("account %d from the groups = %+v, from newLedger %+v", id, a, b)
		}
	}
}

// TestBoundaryFlow_RowOrder verifies that the BoundaryFlow rows of a region list
// the built-in reasons first and the want_consumption rows after them, whatever
// the order of the totals it is given, region by region, and that the module
// rows are ordered by the rank of their registry row.
//
// Requirement: FR-LED-050, FR-OBS-014
func TestBoundaryFlow_RowOrder(t *testing.T) {
	t.Parallel()
	// The totals are in enum order, as Ledger.AppendFlows gives them, in the
	// first case; in the second the want rows come first, which the order of the
	// rows must undo.
	built := []ledger.Flow{
		{Region: 0, Reason: ledger.Initial, Commodity: ledger.Money, Total: 5},
		{Region: 0, Reason: ledger.Consumption, Commodity: 1, Total: 2},
		{Region: 0, Reason: ledger.Spoilage, Commodity: 0, Total: 1},
	}
	want0 := []ledger.Flow{
		{Region: 0, Reason: ledger.WantConsumption, Commodity: 0, Total: 4},
		{Region: 0, Reason: ledger.WantConsumption, Commodity: 1, Total: 3},
	}
	region1 := []ledger.Flow{
		{Region: 1, Reason: ledger.Wear, Commodity: 1, Total: 6},
		{Region: 1, Reason: ledger.WantConsumption, Commodity: 0, Total: 7},
	}
	tests := []struct {
		name  string
		flows []ledger.Flow
		want  []BoundaryFlow
	}{
		{
			name:  "enum order",
			flows: concat(built, want0, region1),
		},
		{
			name:  "module rows first in the input",
			flows: concat(want0, built, region1),
		},
	}
	wantRows := []BoundaryFlow{
		{Tick: 9, Region: 0, Reason: ReasonInitial, Kind: KindFaucet, Commodity: NoCommodity, Total: 5},
		{Tick: 9, Region: 0, Reason: ReasonConsumption, Kind: KindSink, Commodity: 1, Total: 2},
		{Tick: 9, Region: 0, Reason: ReasonSpoilage, Kind: KindSink, Commodity: 0, Total: 1},
		{Tick: 9, Region: 0, Reason: ReasonWantConsumption, Kind: KindSink, Commodity: 0, Total: 4},
		{Tick: 9, Region: 0, Reason: ReasonWantConsumption, Kind: KindSink, Commodity: 1, Total: 3},
		{Tick: 9, Region: 1, Reason: ReasonWear, Kind: KindSink, Commodity: 1, Total: 6},
		{Tick: 9, Region: 1, Reason: ReasonWantConsumption, Kind: KindSink, Commodity: 0, Total: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := boundaryFlows(nil, tt.flows, 9, nil); !reflect.DeepEqual(got, wantRows) {
				t.Errorf("boundaryFlows(%+v, 9, nil) =\n %+v\nwant\n %+v", tt.flows, got, wantRows)
			}
		})
	}
}

// concat joins flow lists into a new one.
func concat(parts ...[]ledger.Flow) []ledger.Flow {
	var out []ledger.Flow
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
