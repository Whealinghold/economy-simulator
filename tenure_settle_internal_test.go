package economysim

import (
	"math"
	"testing"

	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// The unit tests of the rent step of pass 2 (settleCell, payRent) and of the small
// readers of the tenure scratch (entitled, leastRent, fillRecipe). They call the methods
// on a built world with planted scratch, so each rule is checked without a tick around it.

// payerOf builds the rentStart of a household that took units let units of the endowment 0
// at a rent of rent per unit, paying from its own account.
func payerOf(w *World, id AgentID, units, rent int64) rentStart {
	return rentStart{cell: 0, tenant: id, payer: id, account: w.agents[id].account, units: units, rent: rent}
}

// toSnapshot replays one tick of w up to and including the snapshot of P2.3, which builds
// the entitlements, and stops there.
func toSnapshot(t *testing.T, w *World) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	w.p21Complete()
	w.p23Snapshot()
}

// TestPayRent_SkipsExemptFreeAndSelf verifies spec-land.md 5.3.3: a triple moves no money
// and counts none when the request is exempt, when the rent per unit is 0 and when the
// payer is the lessor itself (the units still count as let, which the caller counts).
//
// Guard: each of the three early returns guards a transfer the ledger would refuse or a
// payment to oneself; without the test a reordered condition would pay a free or an own
// unit.
//
// Requirement: FR-LND-016, FR-LND-018
func TestPayRent_SkipsExemptFreeAndSelf(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		mutate func(x *rentStart)
		lessor AgentID
	}{
		{"exempt", func(x *rentStart) { x.exempt = true }, 0},
		{"rent of 0", func(x *rentStart) { x.rent = 0 }, 0},
		{"payer is the lessor", func(*rentStart) {}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
			x := payerOf(w, 4, 3, 7)
			tt.mutate(&x)
			before := tnTotalMoney(t, w)
			var c tenureCount
			w.tenure.payRent(&c, x, tt.lessor, 3)
			if c != (tenureCount{}) {
				t.Errorf("payRent(%+v, lessor %d) counted %+v, want nothing", x, tt.lessor, c)
			}
			if got := tnMoney(t, w, 4); got != 10_000 {
				t.Errorf("payer holds %d after payRent, want the start money 10 000", got)
			}
			if got := tnTotalMoney(t, w); got != before {
				t.Errorf("total money = %d, want %d", got, before)
			}
		})
	}
}

// TestPayRent_PayeeAtMaxMoneyForfeitsTheRest verifies spec-land.md 5.3.3 and FR-LND-016: a
// lessor that cannot receive the whole rent receives what fits under MaxMoney, the rest
// is forfeited and not charged, and a lessor already at MaxMoney receives nothing and the
// transfer is not attempted.
//
// Arithmetic: the rent due is 7 per unit on 3 units, 21. A lessor holding MaxMoney - 5
// receives 5 and 16 are forfeited; one holding MaxMoney receives 0 and 21 are forfeited.
// A lessor with room receives all 21.
//
// Requirement: FR-LND-016
func TestPayRent_PayeeAtMaxMoneyForfeitsTheRest(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name            string
		room            int64 // MaxMoney less the lessor's balance
		paid, forfeited int64
	}{{"room for all", 1_000, 21, 0}, {"room for 5", 5, 5, 16}, {"no room", 0, 0, 21}} {
		t.Run(tt.name, func(t *testing.T) {
			w := tenureWorld(t, nil)
			lessor, err := w.AddAgent(NewAgent{Region: 0, Money: Money(MaxMoney - tt.room)})
			if err != nil {
				t.Fatalf("AddAgent error = %v, want nil", err)
			}
			x := payerOf(w, 2, 3, 7)
			start := tnMoney(t, w, 2)
			var c tenureCount
			w.tenure.payRent(&c, x, lessor, 3)
			if c.paid != tt.paid || c.forfeited != tt.forfeited {
				t.Errorf("payRent counted paid %d, forfeited %d, want %d and %d", c.paid, c.forfeited, tt.paid, tt.forfeited)
			}
			if got, want := tnMoney(t, w, lessor), MaxMoney-tt.room+tt.paid; got != want {
				t.Errorf("lessor holds %d, want %d", got, want)
			}
			if got, want := tnMoney(t, w, 2), start-tt.paid; got != want {
				t.Errorf("payer holds %d, want %d (only what was paid is charged)", got, want)
			}
		})
	}
}

// TestSettleCell_SupplyByUnusedWeight_AttributionAscending verifies pass 2 for one
// endowment on planted entitlements: three lessors with unused entitlements 5, 3 and 2
// (pool, agent 2, agent 3) are apportioned the 4 let units the starts took by the largest
// remainder, and the units are attributed to tenants in ascending ID against lessors in
// ascending ID, each triple paying its own rent.
//
// Arithmetic: the weights sum to 10, so the shares of 4 are 2.0, 1.2 and 0.8: floors 2, 1,
// 0 and the one unit left goes to the largest remainder, 0.8, the agent 3: supplies 2, 1, 1.
// Tenant 4 took 3 units and tenant 5 took 1. In ascending order tenant 4 takes the pool's 2
// and agent 2's 1, then tenant 5 takes agent 3's 1. At a rent of 7 per unit the pool
// receives 14, agent 2 receives 7 and agent 3 receives 7; tenant 4 pays 21 and tenant 5 7.
//
// Requirement: FR-LND-013
func TestSettleCell_SupplyByUnusedWeight_AttributionAscending(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	toStart(t, w) // grows the scratch; the counts are cleared below
	tn := w.tenure
	tn.rt.ents[0] = []tnr.Holder{{ID: 0, Unused: 5}, {ID: 2, Unused: 3}, {ID: 3, Unused: 2}}
	tn.rt.counts[0] = tenureCount{}
	before := map[AgentID]int64{}
	for _, id := range []AgentID{0, 2, 3, 4, 5} {
		before[id] = tnMoney(t, w, id)
	}
	total := tnTotalMoney(t, w)
	tn.settleCell(0, []rentStart{payerOf(w, 4, 3, 7), payerOf(w, 5, 1, 7)})
	for _, tt := range []struct {
		id   AgentID
		want int64
	}{{0, 14}, {2, 7}, {3, 7}, {4, -21}, {5, -7}} {
		if got := tnMoney(t, w, tt.id) - before[tt.id]; got != tt.want {
			t.Errorf("balance change of agent %d = %d, want %d", tt.id, got, tt.want)
		}
	}
	c := tn.rt.counts[0]
	if c.paid != 28 || c.tenants != 2 || c.lessors != 3 || c.forfeited != 0 {
		t.Errorf("counts = %+v, want paid 28, 2 tenants, 3 lessors and nothing forfeited", c)
	}
	if got := tnTotalMoney(t, w); got != total {
		t.Errorf("total money = %d, want %d (rent moves money and creates none)", got, total)
	}
}

// TestEntitled_ZeroForNonHolderAndBeforeTheFirstSnapshot verifies that the entitlement of
// an agent that holds no title reads 0, that an endowment beyond the scratch the first
// snapshot has not built reads 0, and that a holder reads its entitlement.
//
// Requirement: FR-LND-008
func TestEntitled_ZeroForNonHolderAndBeforeTheFirstSnapshot(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	tn := w.tenure
	if got := tn.entitled(0, 0); got != 0 {
		t.Errorf("entitled(0, pool) before any snapshot = %d, want 0", got)
	}
	toSnapshot(t, w)
	if got := tn.entitled(0, 0); got != 5 {
		t.Errorf("entitled(0, pool) = %d, want 5 (5 000 basis points of 10)", got)
	}
	if got := tn.entitled(0, 3); got != 0 {
		t.Errorf("entitled(0, household 3) = %d, want 0 (it holds no title)", got)
	}
	if got := tn.entitled(1, 0); got != 0 {
		t.Errorf("entitled(1, pool) = %d, want 0 (no such endowment)", got)
	}
}

// TestLeastRent_CommonsReadBeforeTheFirstSnapshot verifies spec-land.md 4.7.2: before the
// first snapshot the commons entitlement reads 0 and the least rent is that of every unit
// beyond the own entitlement; after it the commons are deducted.
//
// Arithmetic: 2 executions of 3 units are 6 units; the pool's title is not the asker's, so
// own 0; at a rent of 2 and with no commons the least rent is 6 * 2 = 12; with the commons
// entitlement of 5 it is (6 - 5) * 2 = 2.
//
// Requirement: FR-LND-012
func TestLeastRent_CommonsReadBeforeTheFirstSnapshot(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	if got, ok := w.tenure.leastRent(3, 0, 2, 3); !ok || got != 12 {
		t.Errorf("leastRent before the snapshot = %d, %v, want 12, true", got, ok)
	}
	toSnapshot(t, w)
	if got, ok := w.tenure.leastRent(3, 0, 2, 3); !ok || got != 2 {
		t.Errorf("leastRent after the snapshot = %d, %v, want 2, true", got, ok)
	}
}

// TestFillRecipe_RentSaturatesWhenTheProductOverflows verifies that the rent of a recipe
// in a view is the largest int64 when the rent of a draw does not fit, so a strategy reads
// an unaffordable recipe and the arithmetic never wraps (architecture.md 3.5).
//
// Guard: a wrapped sum would show a huge rent as a small or negative one, and the default
// strategy would take the recipe as cheap.
func TestFillRecipe_RentSaturatesWhenTheProductOverflows(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	w.tenure.posted[0] = math.MaxInt64 / 2 // 3 units of it overflow
	var r RecipeView
	r.MaxExecutions = 1
	w.tenure.fillRecipe(&r, 0, &w.agents[1])
	if r.Rent != math.MaxInt64 {
		t.Errorf("fillRecipe rent = %d, want MaxInt64", r.Rent)
	}
}
