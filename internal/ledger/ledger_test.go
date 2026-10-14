package ledger_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestNew_RejectsUnusableTables verifies that New refuses a configuration it
// cannot use and accepts the smallest one.
//
// Requirement: FR-LED-007
func TestNew_RejectsUnusableTables(t *testing.T) {
	t.Parallel()
	ok := ledger.Config{Regions: 1, MaxMoney: 1, MaxQuantity: 1}
	tests := []struct {
		name string
		cfg  func(c *ledger.Config)
		want error
	}{
		{name: "smallest valid", cfg: func(c *ledger.Config) {}},
		{name: "no regions", cfg: func(c *ledger.Config) { c.Regions = 0 }, want: ledger.ErrInvalidConfig},
		{name: "no money bound", cfg: func(c *ledger.Config) { c.MaxMoney = 0 }, want: ledger.ErrInvalidConfig},
		{name: "no quantity bound", cfg: func(c *ledger.Config) { c.MaxQuantity = 0 }, want: ledger.ErrInvalidConfig},
		{name: "negative shelf life", cfg: func(c *ledger.Config) { c.Commodities = []ledger.Commodity{{ShelfLife: -1}} }, want: ledger.ErrInvalidConfig},
		{name: "negative durability", cfg: func(c *ledger.Config) { c.Commodities = []ledger.Commodity{{Durability: -1}} }, want: ledger.ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ok
			tt.cfg(&cfg)
			l, err := ledger.New(cfg)
			if !errors.Is(err, tt.want) {
				t.Fatalf("New(%+v) error = %v, want %v", cfg, err, tt.want)
			}
			if err == nil && (l == nil || !l.InConstruction() || l.Len() != 0) {
				t.Errorf("New(%+v) = %+v, want an empty ledger in construction", cfg, l)
			}
		})
	}
}

// TestAccounts_IDsAndClose verifies that IDs are assigned in creation order
// and never reused after a close, that Close checks unknown, closed, a
// reservation and money in that order, and that every operation naming a
// closed account is refused with ErrClosedAccount.
//
// Requirement: FR-LED-008, FR-LED-009
func TestAccounts_IDsAndClose(t *testing.T) {
	t.Parallel()
	f := newFix(t, testMaxMoney)
	// The fixture opened 2 boundaries and 3 accounts.
	if got := f.l.Len(); got != 5 {
		t.Fatalf("Len = %d, want 5", got)
	}
	for want, id := range []int{f.mint, f.goodsBoundary, f.a0, f.a1, f.e0} {
		if id != want {
			t.Errorf("account %d has ID %d, want creation order", want, id)
		}
	}

	if err := f.l.Close(99); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Close(99) = %v, want ErrUnknownAccount", err)
	}
	if err := f.l.Close(f.mint); !errors.Is(err, ledger.ErrInvalidKind) {
		t.Errorf("Close(boundary) = %v, want ErrInvalidKind", err)
	}
	// a0 holds money with an earmark: the reservation is named first, which
	// is why a holding with only an earmark says what is wrong with it.
	if err := f.l.ReserveMoney(f.a0, 10); err != nil {
		t.Fatalf("ReserveMoney: %v", err)
	}
	before := snapshot(f.l)
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrOpenReservation) {
		t.Errorf("Close(reserved) = %v, want ErrOpenReservation", err)
	}
	if err := f.l.ReleaseMoney(f.a0, 10); err != nil {
		t.Fatalf("ReleaseMoney: %v", err)
	}
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrNotEmpty) {
		t.Errorf("Close(money) = %v, want ErrNotEmpty", err)
	}
	_ = before
	if err := f.l.Transfer(f.a0, f.e0, 100, ledger.EstateMoney); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	pre := snapshot(f.l)
	if err := f.l.Close(f.a0); err != nil {
		t.Fatalf("Close(empty) = %v, want nil", err)
	}
	if reflect.DeepEqual(pre, snapshot(f.l)) {
		t.Errorf("Close changed nothing")
	}
	v, err := f.l.Account(f.a0)
	if err != nil || !v.Closed || v.Kind != ledger.KindAgent {
		t.Errorf("Account(closed) = %+v, %v, want a closed agent", v, err)
	}

	closedOps := []struct {
		name string
		do   func() error
	}{
		{"close again", func() error { return f.l.Close(f.a0) }},
		{"debit", func() error { return f.l.Transfer(f.a0, f.a1, 1, ledger.Trade) }},
		{"credit", func() error { return f.l.Transfer(f.e0, f.a0, 1, ledger.StartMoney) }},
		{"reserve", func() error { return f.l.ReserveMoney(f.a0, 1) }},
		{"release", func() error { return f.l.ReleaseMoney(f.a0, 1) }},
	}
	for _, c := range closedOps {
		if err := c.do(); !errors.Is(err, ledger.ErrClosedAccount) {
			t.Errorf("%s on a closed account = %v, want ErrClosedAccount", c.name, err)
		}
	}

	// The ID of a closed account is never reused.
	id, err := f.l.Open(ledger.KindAgent, 1, 0)
	if err != nil || id != 5 {
		t.Errorf("Open after a close = %d, %v, want 5 (no reuse)", id, err)
	}
	if got := sum(f.l); got != 0 {
		t.Errorf("total of money = %d, want 0", got)
	}
}

// TestOpen_Refusals verifies the refusals of Open and OpenBoundary and the
// boundary lookup.
//
// Requirement: FR-LED-008, FR-LED-052
func TestOpen_Refusals(t *testing.T) {
	t.Parallel()
	f := newFix(t, testMaxMoney)
	tests := []struct {
		name string
		do   func() error
		want error
	}{
		{"open a boundary kind", func() error { _, err := f.l.Open(ledger.KindBoundary, 0, 0); return err }, ledger.ErrInvalidKind},
		{"open a number that is no kind", func() error { _, err := f.l.Open(ledger.Kind(40), 0, 0); return err }, ledger.ErrInvalidKind},
		{"negative region", func() error { _, err := f.l.Open(ledger.KindAgent, 0, -1); return err }, ledger.ErrUnknownRegion},
		{"region too large", func() error { _, err := f.l.Open(ledger.KindAgent, 0, 2); return err }, ledger.ErrUnknownRegion},
		{"duplicate boundary", func() error { _, err := f.l.OpenBoundary(ledger.Initial, ledger.Money); return err }, ledger.ErrDuplicateBoundary},
		{"internal reason has no boundary", func() error { _, err := f.l.OpenBoundary(ledger.Trade, ledger.Money); return err }, ledger.ErrReasonMismatch},
		{"goods reason has no money boundary", func() error { _, err := f.l.OpenBoundary(ledger.Consumption, ledger.Money); return err }, ledger.ErrReasonMismatch},
		{"reason out of range", func() error { _, err := f.l.OpenBoundary(ledger.NumReasons, ledger.Money); return err }, ledger.ErrReasonMismatch},
		{"commodity too large", func() error { _, err := f.l.OpenBoundary(ledger.Consumption, 2); return err }, ledger.ErrUnknownCommodity},
		{"commodity below money", func() error { _, err := f.l.OpenBoundary(ledger.Consumption, -2); return err }, ledger.ErrUnknownCommodity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := snapshot(f.l)
			if err := tt.do(); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
			if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
				t.Errorf("a refused open changed the ledger")
			}
		})
	}

	id, err := f.l.OpenBoundary(ledger.Consumption, 1)
	if err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	if got, ok := f.l.Boundary(ledger.Consumption, 1); !ok || got != id {
		t.Errorf("Boundary(Consumption, 1) = %d, %v, want %d, true", got, ok, id)
	}
	if _, ok := f.l.Boundary(ledger.Consumption, 0); ok {
		t.Errorf("Boundary(Consumption, 0) found an account that was never opened")
	}
	if _, ok := f.l.Boundary(ledger.Trade, ledger.Money); ok {
		t.Errorf("Boundary(Trade, Money) found an account for an internal reason")
	}
	v, _ := f.l.Account(id)
	if v.Kind != ledger.KindBoundary || v.Region != -1 {
		t.Errorf("boundary view = %+v, want kind boundary and region -1", v)
	}
}

// TestFlowTotals_Attribution verifies that a flow is attributed to the
// region of the non-boundary side for a faucet and of the debited account for
// an internal flow, that totals add over transfers, and that AppendFlows
// lists them in ascending region, reason and money-before-goods order with
// zero totals left out.
//
// Requirement: FR-LED-023, FR-LED-053
func TestFlowTotals_Attribution(t *testing.T) {
	t.Parallel()
	f := newFix(t, testMaxMoney)
	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	// Faucet into region 1 (a1), internal region 0 -> region 1 (debits a0),
	// internal region 1 -> region 0 (debits a1), and a second faucet to a1.
	steps := []struct {
		from, to int
		amount   int64
		reason   ledger.Reason
	}{
		{f.mint, f.a1, 7, ledger.Initial},
		{f.a0, f.a1, 20, ledger.Trade},
		{f.a1, f.e0, 5, ledger.EstateMoney},
		{f.mint, f.a1, 3, ledger.Initial},
		{f.a0, f.a1, 10, ledger.Trade},
	}
	for _, s := range steps {
		if err := f.l.Transfer(s.from, s.to, s.amount, s.reason); err != nil {
			t.Fatalf("Transfer: %v", err)
		}
	}
	// The fixture's own funding of a0 (100, Initial, region 0) happened in
	// construction and is in the bucket, not here.
	want := []ledger.Flow{
		// region 0: trades debited a0, 20 + 10.
		{Region: 0, Reason: ledger.Trade, Commodity: ledger.Money, Total: 30},
		// region 1: faucet 7 + 3 into a1, and the estate_money debit of a1.
		{Region: 1, Reason: ledger.Initial, Commodity: ledger.Money, Total: 10},
		{Region: 1, Reason: ledger.EstateMoney, Commodity: ledger.Money, Total: 5},
	}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, want) {
		t.Errorf("AppendFlows(totals) = %+v, want %+v", got, want)
	}
	wantBucket := []ledger.Flow{{Region: 0, Reason: ledger.Initial, Commodity: ledger.Money, Total: 100}}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, wantBucket) {
		t.Errorf("AppendFlows(bucket) = %+v, want %+v", got, wantBucket)
	}
	if got := sum(f.l); got != 0 {
		t.Errorf("total of money = %d, want 0", got)
	}
}

// TestConstruction_BucketAndClear verifies that a move in construction is in
// the bucket and not in the totals, that EndConstruction is once only, that a
// clear zeroes the totals and leaves the bucket, and that a call between
// ticks is in the totals of the next tick.
//
// Requirement: FR-LED-045, FR-LED-023
func TestConstruction_BucketAndClear(t *testing.T) {
	t.Parallel()
	f := newFix(t, testMaxMoney)
	if !f.l.InConstruction() {
		t.Fatalf("InConstruction = false after New")
	}
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("totals in construction = %+v, want none", got)
	}
	bucket := []ledger.Flow{{Region: 0, Reason: ledger.Initial, Commodity: ledger.Money, Total: 100}}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, bucket) {
		t.Fatalf("bucket = %+v, want %+v", got, bucket)
	}

	if err := f.l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	if f.l.InConstruction() {
		t.Errorf("InConstruction = true after EndConstruction")
	}
	if err := f.l.EndConstruction(); !errors.Is(err, ledger.ErrNotConstruction) {
		t.Errorf("second EndConstruction = %v, want ErrNotConstruction", err)
	}

	if err := f.l.Transfer(f.a0, f.a1, 4, ledger.Trade); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	totals := []ledger.Flow{{Region: 0, Reason: ledger.Trade, Commodity: ledger.Money, Total: 4}}
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, totals) {
		t.Errorf("totals after a step's move = %+v, want %+v", got, totals)
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, bucket) {
		t.Errorf("bucket after a move = %+v, want it unchanged", got)
	}

	f.l.ClearTotals()
	if got := f.l.AppendFlows(nil, false); len(got) != 0 {
		t.Errorf("totals after ClearTotals = %+v, want none", got)
	}
	if got := f.l.AppendFlows(nil, true); !reflect.DeepEqual(got, bucket) {
		t.Errorf("bucket after ClearTotals = %+v, want it kept", got)
	}
	// A call between ticks is totalled in the next tick.
	if err := f.l.Transfer(f.a0, f.a1, 6, ledger.Trade); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	totals[0].Total = 6
	if got := f.l.AppendFlows(nil, false); !reflect.DeepEqual(got, totals) {
		t.Errorf("totals after the clear and a move = %+v, want %+v", got, totals)
	}
}

// TestAppendFlows_ReusesDestination verifies that AppendFlows appends to the
// slice it is given and allocates nothing when that slice has room.
//
// Requirement: FR-LED-023
//
// AllocsPerRun cannot run in a parallel test, so this one is serial.
func TestAppendFlows_ReusesDestination(t *testing.T) {
	f := newFix(t, testMaxMoney)
	dst := make([]ledger.Flow, 0, 8)
	got := f.l.AppendFlows(dst, true)
	if len(got) != 1 || &got[0] != &dst[:1][0] {
		t.Errorf("AppendFlows did not write into the slice it was given: %+v", got)
	}
	allocs := testing.AllocsPerRun(100, func() { dst = f.l.AppendFlows(dst[:0], true) })
	if allocs != 0 {
		t.Errorf("AppendFlows allocated %v times with room in the slice, want 0", allocs)
	}
}
