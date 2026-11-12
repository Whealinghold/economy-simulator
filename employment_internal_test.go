package economysim

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

// payrollWorld is a world of one commodity ("grain") and two roles that both tend
// grain: "payer", with an employer block of payrollBP, and "worker", which is
// Hireable. Each group of households is one agent of weight 1 per household
// (the default scale), so the slots of the payers come first, then the workers'.
// Every payer starts with payerMoney and every worker with none. regions is 1 or 2;
// the second region has the same two groups.
func payrollWorld(t *testing.T, payers, workers int, payerMoney Money, payrollBP int64, regions int, obs Observer, extra ...Option) *World {
	t.Helper()
	world := DefaultWorldParams()
	world.Upkeep = []Item{{Commodity: "grain", Quantity: 2}}
	world.OutputCoverTicks = 0
	grain := DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	payer, worker := DefaultRole(), DefaultRole()
	payer.ID, payer.Recipes, payer.StartMoney = "payer", []string{"tend"}, payerMoney
	payer.Employer = &Employer{PayrollBP: payrollBP}
	worker.ID, worker.Recipes, worker.StartMoney, worker.Hireable = "worker", []string{"tend"}, 0, true
	cfg := Config{
		World: world, Calendar: DefaultCalendar(),
		Commodities: []Commodity{grain}, Roles: []Role{payer, worker},
		Recipes: []Recipe{{ID: "tend", Duration: 1, TraineeYield: 5_000, Outputs: []Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}}},
	}
	for g := range regions {
		r := DefaultRegion()
		r.ID, r.ReferenceBasket = fmt.Sprintf("r%d", g), 2_000
		cfg.Regions = append(cfg.Regions, r)
		cfg.Groups = append(cfg.Groups, Group{Region: r.ID, Role: "payer", Households: int64(payers)}, Group{Region: r.ID, Role: "worker", Households: int64(workers)})
	}
	cfg.Modules.Employment = &Employment{}
	opts := append([]Option{WithSeed(1), WithAudit()}, extra...)
	if obs != nil {
		opts = append(opts, WithObserver(obs))
	}
	w, err := NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w
}

// slotsOf returns the slots of the agents of the role named id, in ascending order.
func slotsOf(w *World, role string) []int {
	var out []int
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if w.cfg.Roles[w.agents[i].role].ID == role {
			out = append(out, i)
		}
	}
	return out
}

// balanceOf returns the balance of the account of a slot.
func balanceOf(t *testing.T, w *World, slot int) Money {
	t.Helper()
	v, err := w.ledger.Account(w.agents[slot].account)
	if err != nil {
		t.Fatalf("Account(slot %d) error = %v, want nil", slot, err)
	}
	return Money(v.Balance)
}

// totalMoney is the sum of the balances of the household agents.
func totalMoney(t *testing.T, w *World) Money {
	t.Helper()
	var sum Money
	for i := w.firstHousehold; i < len(w.agents); i++ {
		sum += balanceOf(t, w, i)
	}
	return sum
}

// TestPayroll_PaysByExecutionsAndConservesMoney verifies the rule of P4.1b on a
// hand-built world: the payer pays floor(receipts * 5 000 / 10 000) and the workers
// share it by the cumulative floor of their executions, every unit paid is received,
// and the sum of the balances is the same before and after.
//
// Arithmetic: receipts 1 001 give due floor(1 001 * 5 000 / 10 000) = 500; the payer
// holds more, so the budget is 500. Workers with k = 1, 2, 4 (K = 7) are paid
// floor(500 * 1 / 7) = 71, floor(500 * 3 / 7) - 71 = 214 - 71 = 143 and
// floor(500 * 7 / 7) - 214 = 286, which sum to 500.
//
// Requirement: FR-EMP-068, FR-EMP-070, FR-EMP-071, FR-EMP-072, FR-EMP-073
func TestPayroll_PaysByExecutionsAndConservesMoney(t *testing.T) {
	t.Parallel()
	var events []PayrollPaid
	w := payrollWorld(t, 1, 3, 10_000, 5_000, 1, ObserverFunc(func(e Event) {
		if p, ok := e.(PayrollPaid); ok {
			events = append(events, p)
		}
	}))
	payer, workers := slotsOf(w, "payer")[0], slotsOf(w, "worker")
	w.welfare.addReceipt(payer, 1_001)
	for i, k := range []int64{1, 2, 4} {
		w.welfare.markWorking(workers[i], k, false)
	}
	before, payerBefore := totalMoney(t, w), balanceOf(t, w, payer)
	if err := w.p41bPayroll(); err != nil {
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	if got := totalMoney(t, w); got != before {
		t.Errorf("sum of balances after P4.1b = %d, want %d (no money created or destroyed)", got, before)
	}
	if got := balanceOf(t, w, payer); got != payerBefore-500 {
		t.Errorf("payer balance = %d, want %d", got, payerBefore-500)
	}
	for i, want := range []Money{71, 143, 286} {
		if got := balanceOf(t, w, workers[i]); got != want {
			t.Errorf("worker %d balance = %d, want %d", i, got, want)
		}
	}
	want := PayrollPaid{Tick: 0, Region: 0, Role: 0, Payers: 1, Workers: 7, Due: 500, Paid: 500, Cut: 0}
	if len(events) != 1 || events[0] != want {
		t.Errorf("PayrollPaid events = %+v, want [%+v]", events, want)
	}
	if f := w.welfare.at(payer); f.wagePaid != 500 || f.earned != 0 {
		t.Errorf("payer V_t, Y_t = %d, %d, want 500, 0", f.wagePaid, f.earned)
	}
	if f := w.welfare.at(workers[2]); f.earned != 286 || f.wagePaid != 0 {
		t.Errorf("worker Y_t, V_t = %d, %d, want 286, 0", f.earned, f.wagePaid)
	}
}

// TestPayroll_Boundaries verifies the cases the rule names: a payer cut at its
// balance pays what it holds and the event counts the cut; nothing is paid when no
// worker worked, when the only worker is on an own-use job, or when the payer had
// no receipts; a payer's budget is funded in ascending agent ID.
//
// Arithmetic: receipts 1 000 at 5 000 bp are due 500. A balance of 120 cuts the
// budget to 120 (Cut 380). With no worker the pool is not paid and Cut is all of Due.
// With two payers of budgets 300 and 200 and one worker, the worker is paid 500 and
// both budgets are paid whole, the first payer first.
//
// Requirement: FR-EMP-068, FR-EMP-069, FR-EMP-071
func TestPayroll_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		payers    int
		balance   Money
		receipts  []Money // by payer
		workerK   []int64 // by worker; a negative k is an own-use job of -k executions
		wantPaid  Money
		wantEvent *PayrollPaid
	}{
		{"cut at the balance", 1, 120, []Money{1_000}, []int64{1}, 120,
			&PayrollPaid{Payers: 1, Workers: 1, Due: 500, Paid: 120, Cut: 380}},
		{"no worker worked", 1, 10_000, []Money{1_000}, []int64{0}, 0,
			&PayrollPaid{Due: 500, Cut: 500, Payers: 1}},
		{"own-use job is not work for hire", 1, 10_000, []Money{1_000}, []int64{-3}, 0,
			&PayrollPaid{Due: 500, Cut: 500, Payers: 1}},
		{"no receipts", 1, 10_000, []Money{0}, []int64{2}, 0, nil},
		{"payers are used up in order", 2, 10_000, []Money{600, 400}, []int64{1}, 500,
			&PayrollPaid{Payers: 2, Workers: 1, Due: 500, Paid: 500}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var events []PayrollPaid
			w := payrollWorld(t, tt.payers, len(tt.workerK), tt.balance, 5_000, 1, ObserverFunc(func(e Event) {
				if p, ok := e.(PayrollPaid); ok {
					events = append(events, p)
				}
			}))
			payers, workers := slotsOf(w, "payer"), slotsOf(w, "worker")
			for i, r := range tt.receipts {
				w.welfare.addReceipt(payers[i], int64(r))
			}
			for i, k := range tt.workerK {
				switch {
				case k < 0:
					w.welfare.markWorking(workers[i], -k, true)
				case k > 0:
					w.welfare.markWorking(workers[i], k, false)
				}
			}
			before := totalMoney(t, w)
			if err := w.p41bPayroll(); err != nil {
				t.Fatalf("p41bPayroll() error = %v, want nil", err)
			}
			var paid Money
			for _, s := range workers {
				paid += balanceOf(t, w, s)
			}
			if paid != tt.wantPaid {
				t.Errorf("workers hold %d, want %d", paid, tt.wantPaid)
			}
			if got := totalMoney(t, w); got != before {
				t.Errorf("sum of balances = %d, want %d", got, before)
			}
			switch {
			case tt.wantEvent == nil && len(events) != 0:
				t.Errorf("PayrollPaid events = %+v, want none", events)
			case tt.wantEvent != nil && (len(events) != 1 || events[0] != *tt.wantEvent):
				t.Errorf("PayrollPaid events = %+v, want [%+v]", events, *tt.wantEvent)
			}
			if tt.payers == 2 {
				// budgets 300 and 200: each payer pays its whole budget.
				for i, want := range []Money{10_000 - 300, 10_000 - 200} {
					if b := balanceOf(t, w, payers[i]); b != want {
						t.Errorf("payer %d balance = %d, want %d", i, b, want)
					}
				}
			}
		})
	}
}

// TestPayroll_RegionsAreSeparate verifies that a wage never crosses a region: the
// payer of region 0 pays only the workers of region 0, and the events come in
// ascending region order.
//
// Requirement: FR-EMP-069, FR-EMP-076
func TestPayroll_RegionsAreSeparate(t *testing.T) {
	t.Parallel()
	var events []PayrollPaid
	w := payrollWorld(t, 1, 1, 10_000, 5_000, 2, ObserverFunc(func(e Event) {
		if p, ok := e.(PayrollPaid); ok {
			events = append(events, p)
		}
	}))
	payers, workers := slotsOf(w, "payer"), slotsOf(w, "worker")
	// Slots: region 0 payer, region 0 worker, region 1 payer, region 1 worker.
	if len(payers) != 2 || len(workers) != 2 {
		t.Fatalf("payers %v, workers %v, want two of each", payers, workers)
	}
	w.welfare.addReceipt(payers[0], 1_000)
	w.welfare.addReceipt(payers[1], 2_000)
	w.welfare.markWorking(workers[0], 1, false) // only region 0's worker worked
	if err := w.p41bPayroll(); err != nil {
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	if got := balanceOf(t, w, workers[0]); got != 500 {
		t.Errorf("worker of region 0 holds %d, want 500", got)
	}
	if got := balanceOf(t, w, workers[1]); got != 0 {
		t.Errorf("worker of region 1 holds %d, want 0 (it did not work, and a wage never crosses a region)", got)
	}
	want := []PayrollPaid{
		{Region: 0, Payers: 1, Workers: 1, Due: 500, Paid: 500},
		{Region: 1, Payers: 1, Due: 1_000, Cut: 1_000},
	}
	if !slices.Equal(events, want) {
		t.Errorf("PayrollPaid events = %+v, want %+v", events, want)
	}
}

// TestPayroll_WelfareFeeds verifies the welfare terms of 24.2.1: a worker's wage is
// earned income Y_t (in the income of the tick, not in the relief or the unearned
// terms) and a payer's payment is V_t netted from its income.
//
// Arithmetic: the payer received 1 000 and paid 500, so its income is 1 000 - 500 =
// 500; the single worker received 500 and nothing else, so its income is 500 and its
// relief is 0.
//
// Requirement: FR-EMP-073
func TestPayroll_WelfareFeeds(t *testing.T) {
	t.Parallel()
	w := payrollWorld(t, 1, 1, 10_000, 5_000, 1, nil)
	payer, worker := slotsOf(w, "payer")[0], slotsOf(w, "worker")[0]
	w.welfare.addReceipt(payer, 1_000)
	w.welfare.markWorking(worker, 1, false)
	if err := w.p41bPayroll(); err != nil {
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	prices := make([]int64, len(w.cfg.Commodities))
	got := w.welfare.record(payer, 1, true, 1, 0, prices, nil)
	if got.Income != 500 {
		t.Errorf("payer Income = %d, want 500 (1 000 received less 500 paid)", got.Income)
	}
	got = w.welfare.record(worker, 1, true, 1, 0, prices, nil)
	if got.Income != 500 || got.Relief != 0 || got.Unearned != 0 || got.OwnReceipt != 0 {
		t.Errorf("worker Income, Relief, Unearned, OwnReceipt = %d, %d, %d, %d, want 500, 0, 0, 0", got.Income, got.Relief, got.Unearned, got.OwnReceipt)
	}
}

// TestPayroll_PooledWageExcludesPayers verifies 24.2.3: the rings the cost anchor
// pools into the regional wage hold the households of the roles without an employer
// block only.
//
// Requirement: FR-EMP-073
func TestPayroll_PooledWageExcludesPayers(t *testing.T) {
	t.Parallel()
	w := payrollWorld(t, 3, 2, 10_000, 5_000, 1, nil)
	w.p12Anchors()
	if got, want := len(w.tk.rings[0]), 2; got != want {
		t.Errorf("rings pooled into the wage of region 0 = %d, want %d (the 2 workers, not the 3 payers)", got, want)
	}
}

// TestPayroll_NoAllocationOnTheNoOpPath verifies NFR-EMP-007: with the default
// observer and logger, once the scratch has grown, P4.1b allocates nothing.
//
// Guard: a position of the tick loop that allocated per tick would show up in every
// step of every run with the module on (observability-and-logging.md 8.2).
//
// Requirement: NFR-EMP-007
func TestPayroll_NoAllocationOnTheNoOpPath(t *testing.T) {
	// not parallel: AllocsPerRun is unreliable when other tests allocate.
	w := payrollWorld(t, 2, 3, 10_000, 5_000, 1, nil)
	payers, workers := slotsOf(w, "payer"), slotsOf(w, "worker")
	feed := func() {
		for _, p := range payers {
			w.welfare.addReceipt(p, 1_000)
		}
		for _, s := range workers {
			w.welfare.at(s).jobSet = false
			w.welfare.markWorking(s, 2, false)
		}
	}
	feed()
	if err := w.p41bPayroll(); err != nil { // grows the scratch
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	if n := testing.AllocsPerRun(50, func() {
		feed()
		if err := w.p41bPayroll(); err != nil {
			t.Fatalf("p41bPayroll() error = %v, want nil", err)
		}
	}); n != 0 {
		t.Errorf("allocations per P4.1b = %v, want 0", n)
	}
}

// TestPayroll_CutIsLoggedAtDebugOnly verifies 24.6.3: a payer cut at its balance is
// one Debug line with a constant message and the keys agent, role, due and applied,
// and nothing is logged above Debug.
//
// Requirement: FR-EMP-078
func TestPayroll_CutIsLoggedAtDebugOnly(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}})
	w := payrollWorld(t, 1, 1, 120, 5_000, 1, nil, WithLogger(slog.New(h)))
	payer, worker := slotsOf(w, "payer")[0], slotsOf(w, "worker")[0]
	buf.Reset() // the construction lines
	w.welfare.addReceipt(payer, 1_000)
	w.welfare.markWorking(worker, 1, false)
	if err := w.p41bPayroll(); err != nil {
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	want := fmt.Sprintf("level=DEBUG msg=\"payroll cut at balance\" agent=%d role=0 due=500 applied=120", payer)
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("log = %q, want %q", got, want)
	}
}

// TestPayroll_ConservationProperty verifies, over 300 pseudo-random states of one
// world (receipts, executions and balances change every round; a fixed generator,
// so the run is reproducible), that every unit paid is received and none is created
// or destroyed: the payers lose exactly what the workers gain, the sum of the
// balances is unchanged, the events add up to the same figure, and no balance goes
// negative.
//
// Requirement: FR-EMP-068, FR-EMP-070, FR-EMP-072, NFR-EMP-006
func TestPayroll_ConservationProperty(t *testing.T) {
	t.Parallel()
	var paidByEvent Money
	w := payrollWorld(t, 5, 8, 50_000, 7_300, 1, ObserverFunc(func(e Event) {
		if p, ok := e.(PayrollPaid); ok {
			paidByEvent += p.Paid
		}
	}))
	payers, workers := slotsOf(w, "payer"), slotsOf(w, "worker")
	x := uint64(88172645463325252) // xorshift64 state: a fixed seed
	next := func(n uint64) int64 {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		return int64(x % n)
	}
	for round := range 300 {
		for _, s := range append(slices.Clone(payers), workers...) {
			*w.welfare.at(s) = flows{}
		}
		for _, p := range payers {
			w.welfare.addReceipt(p, next(200_000)) // some payers cannot cover the due and are cut
		}
		for _, s := range workers {
			if k := next(5); k > 0 {
				w.welfare.markWorking(s, k, next(7) == 0)
			}
		}
		beforeAll, beforeP, beforeW := totalMoney(t, w), sumBalances(t, w, payers), sumBalances(t, w, workers)
		beforeEv := paidByEvent
		if err := w.p41bPayroll(); err != nil {
			t.Fatalf("round %d: p41bPayroll() error = %v, want nil", round, err)
		}
		lost := beforeP - sumBalances(t, w, payers)
		if got := totalMoney(t, w); got != beforeAll {
			t.Fatalf("round %d: sum of balances = %d, want %d", round, got, beforeAll)
		}
		if gained := sumBalances(t, w, workers) - beforeW; gained != lost {
			t.Fatalf("round %d: workers gained %d, payers lost %d, want equal", round, gained, lost)
		}
		if got := paidByEvent - beforeEv; got != lost {
			t.Fatalf("round %d: events paid %d, payers lost %d, want equal", round, got, lost)
		}
		for _, s := range append(slices.Clone(payers), workers...) {
			if b := balanceOf(t, w, s); b < 0 {
				t.Fatalf("round %d: balance of slot %d = %d, want at least 0", round, s, b)
			}
		}
	}
}

// sumBalances is the sum of the balances of the accounts of the slots.
func sumBalances(t *testing.T, w *World, slots []int) Money {
	t.Helper()
	var sum Money
	for _, s := range slots {
		sum += balanceOf(t, w, s)
	}
	return sum
}

// TestPayroll_PayerOutOfRangeIsAnInvariant verifies that a payer whose receipts are
// negative, which no settled trade produces, stops the step with the programmer
// invariant and not with a silent zero or a wrapped budget.
//
// Requirement: FR-EMP-068
func TestPayroll_PayerOutOfRangeIsAnInvariant(t *testing.T) {
	t.Parallel()
	w := payrollWorld(t, 1, 1, 10_000, 5_000, 1, nil)
	payer := slotsOf(w, "payer")[0]
	w.welfare.at(payer).receipts = -1
	mustPanicInvariant(t, "the payroll inputs of a payer are out of range", func() { _ = w.p41bPayroll() })
}
