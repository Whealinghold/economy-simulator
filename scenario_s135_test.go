package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s135Doc is the document of FIS-S4 (spec-fiscal.md 18.5 and 19) on pack one: the crown governs the
// north and keeps a buffer store of the staple. It bids at 80% of the reference price up to a hold of
// 400 units, asks at 120% with 50 units never released, and pays the cooks one unit in kind a tick in
// ticks 80 to 99, in the middle of the shock. Every number is a labelled placeholder (N-018, N-027).
const s135Doc = `
[modules.polity.jurisdiction.crown]
seat = "north"
start_money = 20000

[modules.polity.member]
north = "crown"

[modules.fiscal.purchase.buy]
authority = "crown"
commodity = "staple"
quantity = 20
limit_bp = 8000
hold = 400

[modules.fiscal.release.sell]
authority = "crown"
commodity = "staple"
quantity = 20
limit_bp = 12000
reserve = 50

[modules.fiscal.payout.dole]
authority = "crown"
target = { kind = "households", region = "north", role = "cook" }
commodity = "staple"
amount = 1
window = { start = 80, until = 100 }
`

// bufferWatch keeps the units and the value the treasury's agent bought and sold, from the trades.
type bufferWatch struct {
	*fiscalKept
	agent                  economysim.AgentID
	boughtQ, soldQ         int64
	boughtV, soldV         int64
	minMoney, maxHeld      int64
	minMoneySet            bool
	heldStaple, tradesSelf int64
	lastMoney              int64
}

// Observe keeps the treasury's trades and forwards every event to the counters.
func (b *bufferWatch) Observe(e economysim.Event) {
	if tr, ok := e.(economysim.TradeExecuted); ok {
		v, q := int64(tr.Price)*int64(tr.Quantity), int64(tr.Quantity)
		switch b.agent {
		case tr.Buyer:
			b.boughtQ, b.boughtV = b.boughtQ+q, b.boughtV+v
		case tr.Seller:
			b.soldQ, b.soldV = b.soldQ+q, b.soldV+v
		}
		if tr.Buyer == tr.Seller {
			b.tradesSelf++
		}
	}
	b.fiscalKept.Observe(e)
}

// sample reads the treasury's money and staple stock after a tick.
func (b *bufferWatch) sample(t testing.TB, w *economysim.World, staple economysim.CommodityID) {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	m := int64(s.Jurisdictions[0].Money)
	b.lastMoney = m
	if !b.minMoneySet || m < b.minMoney {
		b.minMoney, b.minMoneySet = m, true
	}
	// One region, so the row of the staple is at its commodity's rank (Markets is by region, then commodity).
	b.heldStaple = int64(s.Markets[staple].HeldByTreasury)
	b.maxHeld = max(b.maxHeld, b.heldStaple)
}

// s135Run runs pack one with the buffer store and the shock of S79 on grow_staple (3 000 of 10 000 for the
// ticks 40 to 159: 120 ticks) for 200 ticks under the audit, sampling the treasury after every tick.
func s135Run(t testing.TB, seed uint64) (*bufferWatch, economysim.Money) {
	t.Helper()
	cfg := loadPackBytes(t, []byte(s135Doc), packOne...)
	b := &bufferWatch{fiscalKept: newFiscalKept()}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(b))
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	b.agent = s0.Jurisdictions[0].Agent
	staple := economysim.CommodityID(slices.Index(s0.Names.Commodities, "staple"))
	grow := economysim.RecipeID(slices.Index(s0.Names.Recipes, "grow_staple"))
	start := s0.Jurisdictions[0].Money
	for w.Tick() < 200 {
		switch w.Tick() {
		case 40, 160:
			bp := int64(3_000)
			if w.Tick() == 160 {
				bp = 10_000
			}
			if err := w.SetYieldShocks([]economysim.YieldShock{{Region: 0, Recipe: grow, BP: bp}}); err != nil {
				t.Fatalf("SetYieldShocks(%d) error = %v, want nil", bp, err)
			}
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
		b.sample(t, w, staple)
	}
	return b, start
}

// TestScenarioS135_Band reports FIS-S4 of spec-fiscal.md 19 (STB-L176, STB-L177) on pack one, seeds 1 to
// 10, 200 ticks, audit on: a buffer store of the staple under a harvest shortfall held for 120 ticks.
// The identities are exact: the treasury's money is never negative, its stock never above the hold and
// never negative, the units it bought less those it sold less those paid in kind are its stock at the
// end plus the units that spoiled (no event reports spoilage, so the spoiled units are the residual, and
// the check is that the residual is never negative: no unit is created; the warm start's old lots spoil
// early), its money is its start less what it bought plus what it sold, and it never traded with itself.
// The price path, the spoiled units and the units paid are reported, never asserted (the band is the
// reference model's, N-027).
//
// Requirement: FR-FIS-050, FR-FIS-051, FR-FIS-053, FR-FIS-056, FR-FIS-039, FR-FIS-063
//
// Guard: a purchase above its hold, a release of reserved stock or an in-kind payout that created
// units shows as a break of the stock identity.
func TestScenarioS135_Band(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	fiscalScenario(t, "one", "S135 one+fiscal buffer store under a held shortfall, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		c := fiscalChecker{t, rep, "S135", "one", seed}
		b, start := s135Run(t, seed)
		inKind := int64(b.applied["dole"])
		want := int64(start) - b.boughtV + b.soldV
		c.check("treasury money never negative", b.minMoney >= 0, "lowest balance %d", b.minMoney)
		c.check("stock never above the hold", b.maxHeld <= 400+20, "highest stock %d, hold 400 plus one order of 20", b.maxHeld)
		spoiled := b.boughtQ - b.soldQ - inKind - b.heldStaple
		c.check("stock + spoiled = bought - sold - paid in kind, spoiled >= 0", spoiled >= 0,
			"stock %d, bought %d, sold %d, paid %d: %d units created", b.heldStaple, b.boughtQ, b.soldQ, inKind, -spoiled)
		c.check("money = start - bought + sold", b.lastMoney == want, "money %d, want %d - %d + %d", b.lastMoney, int64(start), b.boughtV, b.soldV)
		c.check("never traded with itself", b.tradesSelf == 0, "%d self trades", b.tradesSelf)
		rep.num(seed, "units bought", b.boughtQ)
		rep.num(seed, "units sold", b.soldQ)
		rep.num(seed, "units paid in kind", inKind)
		rep.num(seed, "units spoiled", spoiled)
		rep.num(seed, "highest stock", b.maxHeld)
		rep.num(seed, "treasury money end", b.lastMoney)
	})
}
