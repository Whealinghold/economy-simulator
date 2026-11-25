package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s134Rate is the placeholder rate of the sales duty of S134, 3% of the value of a trade.
const s134Rate = 300

// s134Doc is the document of FIS-S3 (spec-fiscal.md 19): a sales duty on every trade of the root's
// regions, paid by the seller, and a payout of payback a tick to the cooks of the centre (0 for none).
func s134Doc(payback int64) []byte {
	doc := fmt.Sprintf(`
[modules.fiscal.duty.sales]
authority = "root"
payer = "seller"
rate_bp = %d
`, s134Rate)
	if payback > 0 {
		doc += fmt.Sprintf(`
[modules.fiscal.payout.back]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = %d
`, payback)
	}
	return []byte(doc)
}

// dutyWatch keeps what the exact rebuild of a duty needs: the value of the trades of each household
// seller in each tick, and the duty's own events (a pseudo-agent never pays, so a seller below
// firstHousehold is left out).
type dutyWatch struct {
	*fiscalKept
	firstHousehold economysim.AgentID
	tick           int64
	cur            map[economysim.AgentID]int64
	ticks          []map[economysim.AgentID]int64
}

// Observe keeps the trades of household sellers by tick and forwards every event to the counters.
func (d *dutyWatch) Observe(e economysim.Event) {
	if tr, ok := e.(economysim.TradeExecuted); ok && tr.Seller >= d.firstHousehold {
		if tr.Tick != d.tick {
			d.flush()
			d.tick = tr.Tick
		}
		d.cur[tr.Seller] += int64(tr.Price) * int64(tr.Quantity)
	}
	d.fiscalKept.Observe(e)
}

// flush closes the tick being kept.
func (d *dutyWatch) flush() {
	if len(d.cur) > 0 {
		d.ticks = append(d.ticks, d.cur)
		d.cur = map[economysim.AgentID]int64{}
	}
}

// expectedDue is the sum of the dues of the duty rebuilt from the trades alone: per seller, num = base *
// rate * 10 000 + carry, due = num / 10^8, the carry the remainder (spec-fiscal.md 6.3), at the full scale.
func (d *dutyWatch) expectedDue() int64 {
	d.flush()
	const den = 100_000_000
	carry := map[economysim.AgentID]int64{}
	var due int64
	for _, tick := range d.ticks {
		for seller, base := range tick {
			num := base*s134Rate*10_000 + carry[seller]
			due += num / den
			carry[seller] = num % den
		}
	}
	return due
}

// s134Out is what one run of S134 keeps.
type s134Out struct {
	due, applied, paid economysim.Money
	start, end         economysim.Money
	householdMoney     economysim.Money
	households         int64
	exact              bool
}

// runS134 runs the duty with the payback for ticks under the audit.
func runS134(t testing.TB, seed uint64, payback, ticks int64) s134Out {
	t.Helper()
	cfg := loadPackBytes(t, s134Doc(payback), fivePolityGoverned...)
	watch := &dutyWatch{fiscalKept: newFiscalKept(), cur: map[economysim.AgentID]int64{}}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(watch))
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if len(s0.Jurisdictions) == 0 {
		t.Fatalf("seed %d: no governed jurisdiction, want the root's treasury", seed)
	}
	for _, j := range s0.Jurisdictions {
		watch.firstHousehold = max(watch.firstHousehold, j.Agent+1)
	}
	start := treasuryMoneyRows(t, w)[0]
	fiscalStepTo(t, w, ticks)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var hh economysim.Money
	for _, r := range s.Regions {
		hh += r.MoneyHouseholds
	}
	return s134Out{
		due: watch.due["sales"], applied: watch.applied["sales"], paid: watch.applied["back"],
		start: start, end: s.Jurisdictions[0].Money, householdMoney: hh, households: s.Households,
		exact: int64(watch.due["sales"]) == watch.expectedDue(),
	}
}

// TestScenarioS134_DutyCash reports FIS-S3 of spec-fiscal.md 19 (STB-L175) on the governed warm five,
// seeds 1 to 10, 200 ticks, audit on: a 3% sales duty on every trade of the root's regions, with no
// payout, and with a payout back of 1 and of 4 a cook a tick. The accounting is exact: the dues of the
// duty equal, to the carry, what a rebuild from the TradeExecuted events alone gives (per household
// seller, num = base * rate * 10 000 + carry over 10^8), the collections are never above the dues, and
// the root's treasury ends at its start plus the duty less the payouts. Revenue per tick, the money per
// household and the treasury are reported, never asserted (N-027).
//
// Requirement: FR-FIS-030, FR-FIS-031, FR-FIS-032, FR-FIS-038, FR-FIS-002, FR-FIS-003
//
// Guard: a duty that counted a trade twice, missed a market or took a pseudo-agent's trade would
// disagree with the rebuild; a payout that paid more than the treasury held breaks the identity.
func TestScenarioS134_DutyCash(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const ticks = 200
	fiscalScenario(t, "five", "S134 five+fiscal sales duty and the cash channel, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		c := fiscalChecker{t, rep, "S134", "five", seed}
		for _, p := range []int64{0, 1, 4} {
			o := runS134(t, seed, p, ticks)
			tag := fmt.Sprintf("payback %d", p)
			c.check(tag+": dues equal the rebuild from the trades", o.exact, "event dues %d differ from the rebuild", o.due)
			c.check(tag+": collected within the due", o.applied <= o.due, "applied %d, due %d", o.applied, o.due)
			c.check(tag+": treasury = start + duty - payout", o.end == o.start+o.applied-o.paid,
				"end %d, want %d + %d - %d", o.end, o.start, o.applied, o.paid)
			c.check(tag+": the duty collects", o.applied > 0, "collected %d", o.applied)
			c.check(tag+": the payout pays iff it is configured", (o.paid > 0) == (p > 0), "paid %d with a payback of %d", o.paid, p)
			rep.num(seed, tag+" revenue per tick", int64(o.applied)/ticks)
			rep.num(seed, tag+" money per household", int64(o.householdMoney)/max(o.households, 1))
			rep.num(seed, tag+" treasury end", int64(o.end))
		}
	})
}
