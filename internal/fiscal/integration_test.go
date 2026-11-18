package fiscal_test

import (
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// payers is the state of the levy half of a round: the payers' weights,
// balances, carries and running total due, and the cash held by the treasury.
type payers struct {
	rule     fiscal.Rule
	weight   []int64
	bal      []int64
	carry    []int64
	totals   []int64
	treasury int64
}

func newPayers(rule fiscal.Rule, weight, bal []int64) *payers {
	p := &payers{rule: rule, weight: weight, bal: bal, carry: make([]int64, len(bal)), totals: make([]int64, len(bal))}
	for i := range p.carry {
		p.carry[i] = rule.StartCarry(int64(i))
	}
	return p
}

// levy collects one tick from every payer into the treasury.
func (p *payers) levy(t *testing.T, tick int) {
	t.Helper()
	for i := range p.bal {
		res, ok := p.rule.Due(p.weight[i], p.bal[i], p.carry[i])
		if !ok {
			t.Fatalf("tick %d: Due(%d, %d, %d) not ok", tick, p.weight[i], p.bal[i], p.carry[i])
		}
		p.bal[i] -= res.Collected
		p.treasury += res.Collected
		p.carry[i], p.totals[i] = res.Carry, p.totals[i]+res.Due
	}
}

// held is all the money of the payers and the treasury.
func (p *payers) held() (sum int64, nonNegative bool) {
	sum, nonNegative = p.treasury, p.treasury >= 0
	for _, b := range p.bal {
		sum += b
		nonNegative = nonNegative && b >= 0
	}
	return sum, nonNegative
}

// TestRound_TaxThenPayoutConservesMoney runs the levy and payout of one
// jurisdiction for 40 ticks with the real Rule.Due and Cut: six payers pay a
// fixed levy, the collected cash goes to a treasury, and the treasury is cut
// pro rata among three payees who are due more than it holds. It verifies that
// the money in payers, treasury and payees is the same after every tick, that
// no balance goes negative, that a short treasury is paid out in full, that
// each payer's total due equals the exact closed form
// floor((n*w*amount*scale + carry0) / (period*10^4)), and that a second run
// gives the same payees' balances.
//
// Setup: period 4 with StartCarry spreads the first dues of the payers over
// the period; payer 5 holds only 90, so its due is cut by its balance and
// its carry still advances by the due, not by what it paid (FR-FIS-024).
//
// Requirement: FR-FIS-021, FR-FIS-024, FR-FIS-038, NFR-FIS-004
func TestRound_TaxThenPayoutConservesMoney(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rule := fiscal.Rule{Shape: fiscal.Fixed, Amount: 25, Scale: 10_000, Period: 4, MaxMoney: maxMoney}
	run := func() (paid [3]int64, p *payers) {
		p = newPayers(rule, []int64{1, 2, 3, 1, 2, 4}, []int64{500, 800, 1_200, 700, 900, 90})
		initial, _ := p.held()
		for tick := 1; tick <= 40; tick++ {
			p.levy(t, tick)
			short := p.treasury < 450 // the payees are due 300 + 100 + 50
			rs := []fiscal.Recipient{{Due: 300, Key: 7, ID: 10}, {Due: 100, Key: 3, ID: 11}, {Due: 50, Key: 9, ID: 12}}
			var s fiscal.Scratch
			applied, ok := s.Cut(nil, p.treasury, rs)
			if !ok {
				t.Fatalf("tick %d: Cut(%d) not ok", tick, p.treasury)
			}
			for i, a := range applied {
				paid[i] += a
				p.treasury -= a
			}
			held, nonNeg := p.held()
			held += paid[0] + paid[1] + paid[2]
			if !nonNeg || held != initial || (short && p.treasury != 0) {
				t.Fatalf("tick %d: money %d, want %d; treasury %d (0 if short); balances %v (none negative)", tick, held, initial, p.treasury, p.bal)
			}
		}
		return paid, p
	}
	paid, p := run()
	for i, got := range p.totals {
		want := new(big.Int).Mul(big.NewInt(40*p.weight[i]*rule.Amount), big.NewInt(rule.Scale))
		want.Add(want, big.NewInt(rule.StartCarry(int64(i))))
		want.Div(want, big.NewInt(rule.Period*10_000))
		if big.NewInt(got).Cmp(want) != 0 {
			t.Errorf("payer %d total due over 40 ticks = %d, want exact %v", i, got, want)
		}
	}
	if again, _ := run(); again != paid {
		t.Errorf("two runs paid the payees %v and %v, want the same", paid, again)
	}
}

// TestRound_DutyReachesTheTreasuryExactly verifies that the sales duty over
// four ticks, built from Base and DutyDue together, collects exactly what the
// closed form says: the sum of the dues times 10^8 plus the final carry
// equals the base times rate and scale plus the first carry.
//
// Setup: 300 bp at scale 10 000 on the sales 530, 820, 208 and 100 of four
// ticks, with a first carry of 5 * 10^7 (the 18.3 example of spec-fiscal.md).
//
// Requirement: FR-FIS-032
func TestRound_DutyReachesTheTreasuryExactly(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	sales := [][]int64{{530, 820}, {208}, {}, {100}}
	var sum, exact int64
	carry := int64(50_000_000)
	for tick, vs := range sales {
		var b fiscal.Base
		for _, v := range vs {
			if !b.Add(v) {
				t.Fatalf("tick %d: Add(%d) refused", tick, v)
			}
			exact += v
		}
		due, next, ok := fiscal.DutyDue(b, 300, 10_000, carry, maxMoney)
		if !ok {
			t.Fatalf("tick %d: DutyDue not ok", tick)
		}
		sum, carry = sum+due, next
	}
	if got, want := sum*100_000_000+carry, exact*300*10_000+50_000_000; got != want {
		t.Errorf("duty %d with carry %d: due*10^8 + carry = %d, want %d", sum, carry, got, want)
	}
}

// TestRound_TollLanesAddUp verifies that the tolls of several lanes, built
// with Toll and added with Total, equal the sum of the lanes.
//
// Setup: lanes of 40, 0, 17 and 3 crossed by 1, 2, 3 and 4 carriers at
// 12 500 bp: 40 * 1.25 = 50; 0; 17 * 1.25 * 3 = 63.75, floored to 63; and
// 3 * 1.25 * 4 = 15. The total is 128.
//
// Requirement: FR-FIS-034, FR-FIS-035
func TestRound_TollLanesAddUp(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	var tolls []int64
	for i, amount := range []int64{40, 0, 17, 3} {
		toll, ok := fiscal.Toll(amount, 12_500, int64(i+1))
		if !ok {
			t.Fatalf("Toll(%d, 12500, %d) not ok", amount, i+1)
		}
		tolls = append(tolls, toll)
	}
	want := []int64{50, 0, 63, 15}
	if got, ok := fiscal.Total(tolls); !ok || got != 128 || tolls[0] != want[0] || tolls[2] != want[2] || tolls[3] != want[3] {
		t.Errorf("Total(%v) = %d, %v; want tolls %v and total 128", tolls, got, ok, want)
	}
}

// buyer is a standing purchase order with a hold: its cash, the stock it
// holds, the units it has bought and the carry of its wish.
type buyer struct {
	cash, held, bought, carry, wishes int64
}

// tick places one purchase of 7 units at 12 500 bp against a limit and a hold
// of 10, releases above a reserve on every tenth tick n, and reports what is wrong with the result, if anything.
func (b *buyer) tick(n int, limit int64) string {
	wish, next, ok := fiscal.Wish(7, 12_500, b.carry)
	if !ok {
		return "Wish not ok"
	}
	b.carry, b.wishes = next, b.wishes+wish
	c := fiscal.CapPurchase(wish, b.cash, limit, 10, b.held)
	b.cash -= c.Qty * limit
	b.held += c.Qty
	b.bought += c.Qty
	if b.cash < 0 || b.held > 10 {
		return "cash below 0 or stock above the hold"
	}
	if n%10 == 0 && !b.release() {
		return "CapRelease placed more than the unreserved stock"
	}
	return ""
}

// release places a release of the stock above a reserve of 4 and reports
// whether it stayed within the unreserved stock.
func (b *buyer) release() bool {
	r := fiscal.CapRelease(b.wishes, b.held, 4)
	b.held -= r.Qty
	return r.Qty <= max(0, b.held+r.Qty-4)
}

// TestRound_OrderPipelineKeepsCashAndHold runs a standing purchase order for
// 30 ticks through Limit, Wish and CapPurchase and then a release through
// CapRelease: the cash never goes negative, the stock held never passes the
// hold, the cash spent is the units bought at the limit, and the wishes plus
// the final carry equal the exact closed form, so a cut order does not lose
// the carry.
//
// Setup: reference 50 at 12 000 bp gives a limit of 60; 7 units at 12 500 bp
// wish 8 units on most ticks (8.75 per tick with the carry); cash 400 buys 6
// units at 60, so the cash cuts the early ticks and the hold of 10 the later
// ones.
//
// Requirement: FR-FIS-050, FR-FIS-051
func TestRound_OrderPipelineKeepsCashAndHold(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	limit, ok := fiscal.Limit(50, 12_000, 1_000)
	if !ok || limit != 60 {
		t.Fatalf("Limit(50, 12000, 1000) = %d, %v; want 60, true", limit, ok)
	}
	b := &buyer{cash: 400}
	for tick := 1; tick <= 30; tick++ {
		if msg := b.tick(tick, limit); msg != "" {
			t.Fatalf("tick %d: %s: %+v", tick, msg, *b)
		}
	}
	if b.cash != 400-b.bought*limit || b.bought != 6 {
		t.Errorf("cash %d after buying %d units at %d, want 400 - 6*60 = 40", b.cash, b.bought, limit)
	}
	// The carry is in [0, 10^4), so this equality is the exact floor.
	if b.wishes*fiscal.Basis+b.carry != 30*7*12_500 {
		t.Errorf("30 wishes sum to %d with carry %d, want wishes*10^4 + carry = %d", b.wishes, b.carry, 30*7*12_500)
	}
}
