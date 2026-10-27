package economysim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// Commodity indexes of packOneCommodities used below: prepared spoils (shelf
// life 4), staple keeps (shelf life 700), tool_a wears (durability 10).
const (
	poolStaple = 5
	poolTool   = 6
)

// poolFixture is a two-region ledger of the pack-one commodity table with the
// estates open, still in construction, and two plain agent accounts.
type poolFixture struct {
	l      *ledger.Ledger
	e      *estates
	a, b   int
	minted [9]int64 // goods created by initial, per commodity
	money  int64    // money created by initial
}

func newPoolFixture(t *testing.T) *poolFixture {
	t.Helper()
	cfg := Config{Commodities: packOneCommodities(), Regions: []Region{{ID: "north"}, {ID: "south"}}}
	f := &poolFixture{l: mustLedger(t, cfg)}
	e, err := openEstates(f.l, 2)
	if err != nil {
		t.Fatalf("openEstates: %v", err)
	}
	f.e = e
	for _, p := range []*int{&f.a, &f.b} {
		id, err := f.l.Open(ledger.KindAgent, 100, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		*p = id
	}
	return f
}

// fund creates money for account to by initial.
func (f *poolFixture) fund(t *testing.T, to int, m int64) {
	t.Helper()
	src, _ := f.l.Boundary(ledger.Initial, ledger.Money)
	if err := f.l.Transfer(src, to, m, ledger.Initial); err != nil {
		t.Fatalf("fund: %v", err)
	}
	f.money += m
}

// stock creates qty units of commodity c for account to by initial.
func (f *poolFixture) stock(t *testing.T, to, c int, qty int64) {
	t.Helper()
	src, _ := f.l.Boundary(ledger.Initial, c)
	if err := f.l.Move(src, to, c, qty, ledger.Initial); err != nil {
		t.Fatalf("stock: %v", err)
	}
	f.minted[c] += qty
}

func (f *poolFixture) lots(t *testing.T, acct int) []ledger.Lot {
	t.Helper()
	lots, err := f.l.AppendLots(nil, acct)
	if err != nil {
		t.Fatalf("AppendLots(%d): %v", acct, err)
	}
	return lots
}

func (f *poolFixture) balance(t *testing.T, acct int) int64 {
	t.Helper()
	v, err := f.l.Account(acct)
	if err != nil {
		t.Fatalf("Account(%d): %v", acct, err)
	}
	return v.Balance
}

// audit checks the ledger audit and the exact conservation of the money and
// goods created by initial over the non-boundary accounts.
func (f *poolFixture) audit(t *testing.T, ctx string) {
	t.Helper()
	if err := f.l.Audit(); err != nil {
		t.Fatalf("%s: Audit() = %v, want nil", ctx, err)
	}
	var money int64
	var goods [9]int64
	for id := range f.l.Len() {
		if v, _ := f.l.Account(id); v.Kind == ledger.KindBoundary || v.Closed {
			continue
		}
		money += f.balance(t, id)
		for _, lt := range f.lots(t, id) {
			goods[lt.Commodity] += lt.Quantity
		}
	}
	if money != f.money || goods != f.minted {
		t.Fatalf("%s: held money %d goods %v, created money %d goods %v", ctx, money, goods, f.money, f.minted)
	}
}

// TestOpenEstates_AccountsFollowTheBoundary verifies that the estates open in
// region order straight after the boundary accounts, with the IDs B + r, and
// that a ledger with a non-boundary account already open is an invariant
// while one with fewer boundary accounts than the built-in group still opens.
//
// Arithmetic: nine commodities give B = 1 + 7 * 9 = 64 boundary accounts, so
// the estates are 64 and 65 and the first agent is 66.
//
// Requirement: FR-EST-004, FR-EST-005
func TestOpenEstates_AccountsFollowTheBoundary(t *testing.T) {
	t.Parallel()
	t.Run("after the built-in boundary", func(t *testing.T) {
		t.Parallel()
		f := newPoolFixture(t)
		if !slices.Equal(f.e.accounts, []int{64, 65}) || f.a != 66 {
			t.Fatalf("estate accounts %v, first agent %d, want [64 65], 66", f.e.accounts, f.a)
		}
		for r := range 2 {
			v, err := f.l.Account(f.e.account(RegionID(r)))
			if err != nil || v.Kind != ledger.KindEstate || v.Region != r {
				t.Errorf("Account(estate %d) = %+v, %v, want an estate of region %d", r, v, err, r)
			}
		}
	})
	t.Run("an agent account first is an invariant", func(t *testing.T) {
		t.Parallel()
		l := mustLedger(t, Config{Commodities: packOneCommodities(), Regions: []Region{{ID: "north"}}})
		if _, err := l.Open(ledger.KindAgent, 0, 0); err != nil {
			t.Fatal(err)
		}
		mustPanicInvariant(t, "estate", func() { _, _ = openEstates(l, 1) })
	})
	t.Run("a smaller boundary group still opens", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := l.OpenBoundary(ledger.Initial, ledger.Money); err != nil {
			t.Fatal(err)
		}
		e, err := openEstates(l, 1)
		if err != nil || e.accounts[0] != 1 {
			t.Fatalf("openEstates = %v, %v, want account 1", e, err)
		}
	})
}

// TestSeedPool verifies that the pool is created by the reason initial during
// construction (it shows in the construction bucket), that 0 is skipped and
// that outside construction it is ledger.ErrNotConstruction.
//
// Arithmetic: seeds 300 and 0 leave pools 300 and 0, and the bucket holds one
// initial money flow of 300 in region 0.
//
// Requirement: FR-EST-015, FR-EST-013
func TestSeedPool(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	if err := f.e.seedPool(f.l, 0, 300); err != nil {
		t.Fatalf("seedPool(0, 300) = %v", err)
	}
	if err := f.e.seedPool(f.l, 1, 0); err != nil {
		t.Fatalf("seedPool(1, 0) = %v", err)
	}
	f.money = 300
	if got0, got1 := f.e.pool(f.l, 0), f.e.pool(f.l, 1); got0 != 300 || got1 != 0 {
		t.Errorf("pool = %d, %d, want 300, 0", got0, got1)
	}
	var flow int64
	for _, fl := range f.l.AppendFlows(nil, true) {
		if fl.Reason == ledger.Initial && fl.Commodity == ledger.Money && fl.Region == 0 {
			flow += fl.Total
		}
	}
	if flow != 300 {
		t.Errorf("construction bucket initial money = %d, want 300", flow)
	}
	f.audit(t, "after seed")
	if err := f.l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	if err := f.e.seedPool(f.l, 0, 5); !errors.Is(err, ledger.ErrNotConstruction) {
		t.Errorf("seedPool outside construction = %v, want ErrNotConstruction", err)
	}
}

// TestGrant verifies that a draw is min(want, pool), is nothing for an amount
// below 1, never leaves the pool negative and uses the reason it is given.
//
// Arithmetic: pool 100; want 40 gives 40 (pool 60); want 90 gives the 60 left
// (pool 0); want 5 on the empty pool gives 0; want -3 and 0 give 0.
//
// Requirement: FR-EST-014, FR-EST-022
func TestGrant(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	if err := f.e.seedPool(f.l, 0, 100); err != nil {
		t.Fatal(err)
	}
	f.money = 100
	tests := []struct {
		name       string
		want       Money
		reason     ledger.Reason
		given, bal int64
	}{
		{"within the pool", 40, ledger.StartMoney, 40, 60},
		{"above the pool gives the rest", 90, ledger.Births, 60, 0},
		{"empty pool", 5, ledger.StartMoney, 0, 0},
		{"zero", 0, ledger.StartMoney, 0, 0},
		{"negative", -3, ledger.StartMoney, 0, 0},
	}
	var recv int64
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := f.e.grant(f.l, 0, f.a, tc.want, tc.reason)
			if err != nil || int64(got) != tc.given {
				t.Fatalf("grant(%d) = %d, %v, want %d", tc.want, got, err, tc.given)
			}
			recv += tc.given
			if p := f.e.pool(f.l, 0); int64(p) != tc.bal {
				t.Errorf("pool = %d, want %d", p, tc.bal)
			}
			if b := f.balance(t, f.a); b != recv {
				t.Errorf("recipient balance = %d, want %d", b, recv)
			}
			f.audit(t, tc.name)
		})
	}
}

// TestPoolMoney_Errors verifies the refusals the ledger reports through the
// pool functions: a wrong reason is ErrReasonMismatch and an estate balance
// over the bound is ErrBalanceOverflow, both with errors.Is.
//
// Arithmetic: with MaxMoney 100, a pool of 60 plus 50 paid in is 110 > 100.
//
// Requirement: FR-EST-016, FR-EST-022
func TestPoolMoney_Errors(t *testing.T) {
	t.Parallel()
	t.Run("grant with a wrong reason", func(t *testing.T) {
		t.Parallel()
		f := newPoolFixture(t)
		_ = f.e.seedPool(f.l, 0, 10)
		if _, err := f.e.grant(f.l, 0, f.a, 5, ledger.EstateMoney); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("grant with estate_money = %v, want ErrReasonMismatch", err)
		}
	})
	t.Run("acceptMoney with a wrong reason", func(t *testing.T) {
		t.Parallel()
		f := newPoolFixture(t)
		f.fund(t, f.a, 10)
		if err := f.e.acceptMoney(f.l, f.a, 0, 5, ledger.StartMoney); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("acceptMoney with start_money = %v, want ErrReasonMismatch", err)
		}
	})
	t.Run("estate balance overflow", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100})
		if err != nil {
			t.Fatal(err)
		}
		src, _ := l.OpenBoundary(ledger.Initial, ledger.Money)
		e, err := openEstates(l, 1)
		if err != nil {
			t.Fatal(err)
		}
		ag, _ := l.Open(ledger.KindAgent, 0, 0)
		if err := l.Transfer(src, ag, 50, ledger.Initial); err != nil {
			t.Fatal(err)
		}
		if err := e.seedPool(l, 0, 60); err != nil {
			t.Fatal(err)
		}
		if err := e.acceptMoney(l, ag, 0, 50, ledger.EstateMoney); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("acceptMoney past MaxMoney = %v, want ErrBalanceOverflow", err)
		}
		if _, _, err := e.receive(l, ag, 0); !errors.Is(err, ledger.ErrBalanceOverflow) {
			t.Errorf("receive past MaxMoney = %v, want ErrBalanceOverflow", err)
		}
	})
}

// TestAcceptMoney verifies that money moves from an agent to the estate by the
// reason of a weight reduction and that an amount below 1 moves nothing.
//
// Requirement: FR-EST-001, FR-EST-016
func TestAcceptMoney(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	f.fund(t, f.a, 50)
	for _, tc := range []struct {
		name   string
		amount Money
		reason ledger.Reason
		pool   Money
	}{
		{"split", 20, ledger.Split, 20},
		{"merge", 5, ledger.Merge, 25},
		{"starvation", 3, ledger.Starvation, 28},
		{"zero moves nothing", 0, ledger.Split, 28},
		{"negative moves nothing", -4, ledger.Split, 28},
	} {
		if err := f.e.acceptMoney(f.l, f.a, 0, tc.amount, tc.reason); err != nil {
			t.Fatalf("%s: acceptMoney = %v", tc.name, err)
		}
		if got := f.e.pool(f.l, 0); got != tc.pool {
			t.Errorf("%s: pool = %d, want %d", tc.name, got, tc.pool)
		}
		f.audit(t, tc.name)
	}
}

// TestReceive verifies the intake of a removal: the whole balance by
// estate_money and every lot by estate_goods with the keys kept, the counts
// the events carry, no payment to anyone for the goods (N-167), and nothing
// moved nor counted for an agent that holds nothing.
//
// Boundary: a balance of exactly 1 minor unit moves in full and strands nothing.
//
// Arithmetic: agent a holds 70 money, 4 staple and 3 + 2 tool_a; the two
// tool_a stocks share one key, so they merge into one lot of 5 and the agent
// holds 2 lots, which is the count GoodsListed carries.
//
// Requirement: FR-EST-001, FR-EST-021, FR-EST-013
func TestReceive(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	f.fund(t, f.a, 70)
	f.stock(t, f.a, poolStaple, 4)
	f.stock(t, f.a, poolTool, 3)
	f.stock(t, f.a, poolTool, 2) // same key: merges into one lot of 5
	want := f.lots(t, f.a)
	if len(want) != 2 {
		t.Fatalf("fixture lots = %v, want 2 lots", want)
	}
	money, lots, err := f.e.receive(f.l, f.a, 1)
	if err != nil || money != 70 || lots != 2 {
		t.Fatalf("receive = %d, %d, %v, want 70, 2, nil", money, lots, err)
	}
	if got := f.lots(t, f.e.account(1)); !slices.Equal(got, want) {
		t.Errorf("estate lots = %v, want %v (keys kept)", got, want)
	}
	if got := f.e.pool(f.l, 1); got != 70 {
		t.Errorf("pool = %d, want 70", got)
	}
	if left := f.lots(t, f.a); len(left) != 0 || f.balance(t, f.a) != 0 {
		t.Errorf("agent keeps %v and %d money, want nothing", left, f.balance(t, f.a))
	}
	f.audit(t, "after receive")
	// Nothing to take.
	if money, lots, err := f.e.receive(f.l, f.b, 0); err != nil || money != 0 || lots != 0 {
		t.Errorf("receive of an empty agent = %d, %d, %v, want 0, 0, nil", money, lots, err)
	}
	f.audit(t, "after empty receive")
	// Boundary: the smallest positive balance still moves (money > 0, not > 1).
	g := newPoolFixture(t)
	g.fund(t, g.a, 1)
	if money, lots, err := g.e.receive(g.l, g.a, 1); err != nil || money != 1 || lots != 0 {
		t.Errorf("receive of 1 minor unit = %d, %d, %v, want 1, 0, nil", money, lots, err)
	}
	if got := g.e.pool(g.l, 1); got != 1 {
		t.Errorf("pool after receiving 1 = %d, want 1", got)
	}
	if got := g.balance(t, g.a); got != 0 {
		t.Errorf("agent keeps %d money, want 0 (none stranded)", got)
	}
	g.audit(t, "after receive of 1")
}

// TestAcceptGoods verifies that a selector moves only the whole lots of the
// commodities it selects and counts them, and that a nil selector moves all.
//
// Requirement: FR-EST-001, FR-EST-021
func TestAcceptGoods(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	f.stock(t, f.a, poolStaple, 4)
	f.stock(t, f.a, poolTool, 3)
	if n, err := f.e.acceptGoods(f.l, f.a, 0, ledger.EstateGoods, func(c int) bool { return c == poolTool }); err != nil || n != 1 {
		t.Fatalf("acceptGoods(tool) = %d, %v, want 1, nil", n, err)
	}
	if got := f.lots(t, f.a); len(got) != 1 || got[0].Commodity != poolStaple {
		t.Errorf("agent lots = %v, want only staple", got)
	}
	if got := f.lots(t, f.e.account(0)); len(got) != 1 || got[0].Commodity != poolTool || got[0].Quantity != 3 {
		t.Errorf("estate lots = %v, want 3 tool_a", got)
	}
	if n, err := f.e.acceptGoods(f.l, f.a, 0, ledger.EstateGoods, nil); err != nil || n != 1 {
		t.Fatalf("acceptGoods(nil) = %d, %v, want 1, nil", n, err)
	}
	f.audit(t, "after goods")
	t.Run("a reason of the wrong kind is an invariant", func(t *testing.T) {
		g := newPoolFixture(t)
		g.stock(t, g.a, poolStaple, 1)
		mustPanicInvariant(t, "estate", func() { _, _ = g.e.acceptGoods(g.l, g.a, 0, ledger.StartMoney, nil) })
	})
}

// TestAcceptUnits verifies that a share of one commodity moves oldest lot
// first with the keys kept, that the units moved are returned, and that qty
// below 1 is a no-op returning 0.
//
// Arithmetic: staple lot A (5 units, tick 0) and lot B (4 units, born at a
// later tick) give qty 7 = all 5 of A and 2 of B, leaving B with 2.
//
// Requirement: FR-EST-001, FR-EST-016
func TestAcceptUnits(t *testing.T) {
	t.Parallel()
	f := newPoolFixture(t)
	f.stock(t, f.a, poolStaple, 5)
	if err := f.l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	f.l.AdvanceTick()
	f.stock(t, f.a, poolStaple, 4)
	before := f.lots(t, f.a)
	if len(before) != 2 || before[0].Born >= before[1].Born {
		t.Fatalf("fixture lots = %v, want two lots, oldest first", before)
	}
	for _, q := range []int64{0, -2} {
		if n, err := f.e.acceptUnits(f.l, f.a, 0, poolStaple, q, ledger.Split); n != 0 || err != nil {
			t.Errorf("acceptUnits(%d) = %d, %v, want 0, nil", q, n, err)
		}
	}
	n, err := f.e.acceptUnits(f.l, f.a, 0, poolStaple, 7, ledger.Split)
	if n != 7 || err != nil {
		t.Fatalf("acceptUnits(7) = %d, %v, want 7, nil", n, err)
	}
	est, left := f.lots(t, f.e.account(0)), f.lots(t, f.a)
	if len(est) != 2 || est[0].Quantity != 5 || est[1].Quantity != 2 || est[0].Born != before[0].Born || est[1].Born != before[1].Born {
		t.Errorf("estate lots = %v, want 5 of the old key and 2 of the new", est)
	}
	if len(left) != 1 || left[0].Quantity != 2 || left[0].Born != before[1].Born {
		t.Errorf("agent lots = %v, want 2 of the new key", left)
	}
	f.audit(t, "after units")
	mustPanicInvariant(t, "estate", func() { _, _ = f.e.acceptUnits(f.l, f.a, 0, poolStaple, 99, ledger.Split) })
}

// TestEstates_ConservationOverSeeds runs a random script of grants, payments,
// partial and whole-lot moves and removals over seeds 1 to 50 and checks the
// audit and the exact conservation of money and goods after every step.
//
// Guard: every transfer of the estate goes through these functions; a cap or a
// reason mistake would break conservation only for some orders of operations.
//
// Requirement: FR-EST-016, FR-EST-022
func TestEstates_ConservationOverSeeds(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 50; seed++ {
		f := newPoolFixture(t)
		rng := rand.New(rand.NewPCG(seed, 0))
		accts := []int{f.a, f.b}
		for r := range RegionID(2) {
			if err := f.e.seedPool(f.l, r, Money(rng.IntN(500))); err != nil {
				t.Fatalf("seed %d: seedPool: %v", seed, err)
			}
		}
		f.money = int64(f.e.pool(f.l, 0) + f.e.pool(f.l, 1))
		for _, a := range accts {
			f.fund(t, a, int64(rng.IntN(300)))
			for _, c := range []int{poolStaple, poolTool} {
				if q := int64(rng.IntN(20)); q > 0 {
					f.stock(t, a, c, q)
				}
			}
		}
		if err := f.l.EndConstruction(); err != nil {
			t.Fatal(err)
		}
		for step := range 40 {
			a, r := accts[rng.IntN(2)], RegionID(rng.IntN(2))
			var err error
			switch op := rng.IntN(5); op {
			case 0:
				_, err = f.e.grant(f.l, r, a, Money(rng.IntN(400)), ledger.Births)
			case 1:
				err = f.e.acceptMoney(f.l, a, r, Money(min(int64(rng.IntN(100)), f.balance(t, a))), ledger.Merge)
			case 2:
				var held int64
				for _, lt := range f.lots(t, a) {
					if lt.Commodity == poolStaple {
						held += lt.Quantity
					}
				}
				_, err = f.e.acceptUnits(f.l, a, r, poolStaple, min(held, int64(rng.IntN(8))), ledger.Starvation)
			case 3:
				_, err = f.e.acceptGoods(f.l, a, r, ledger.EstateGoods, func(c int) bool { return c == poolTool })
			default:
				_, _, err = f.e.receive(f.l, a, r)
			}
			if err != nil {
				t.Fatalf("seed %d step %d: %v", seed, step, err)
			}
			f.audit(t, fmt.Sprintf("seed %d step %d", seed, step))
			if p := f.e.pool(f.l, r); p < 0 {
				t.Fatalf("seed %d step %d: pool %d negative", seed, step, p)
			}
		}
	}
}

// TestPoolNoOutlet_Table verifies the condition of grant_to_pool_no_outlet on its
// own, without a validator: only a pool target, with the estates module on, in a
// region (or, with no region named, any region) whose PoolRelease is 0.
//
// Arithmetic: regions "a" (PoolRelease 0) and "b" (PoolRelease 100). A target
// that names "b" has an outlet; one that names "a" has none; one that names no
// region has none because "a" is among every region.
//
// Requirement: FR-MON-039
func TestPoolNoOutlet_Table(t *testing.T) {
	t.Parallel()
	regions := []Region{{ID: "a", PoolRelease: 0}, {ID: "b", PoolRelease: 100}}
	tests := []struct {
		name   string
		target FlowTarget
		module *Estates
		want   bool
	}{
		{"pool of a region with no release", FlowTarget{Kind: TargetPool, Region: "a"}, &Estates{}, true},
		{"pool of a region that releases", FlowTarget{Kind: TargetPool, Region: "b"}, &Estates{}, false},
		{"pool of every region, one releases nothing", FlowTarget{Kind: TargetPool}, &Estates{}, true},
		{"pool of an unknown region", FlowTarget{Kind: TargetPool, Region: "z"}, &Estates{}, false},
		{"households target", FlowTarget{Kind: TargetHouseholds, Region: "a"}, &Estates{}, false},
		{"module off", FlowTarget{Kind: TargetPool, Region: "a"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{Regions: regions, Modules: Modules{Estates: tt.module}}
			if got := poolNoOutlet(c, tt.target); got != tt.want {
				t.Errorf("poolNoOutlet(%+v, estates=%v) = %v, want %v", tt.target, tt.module != nil, got, tt.want)
			}
		})
	}
}
