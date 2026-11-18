package economysim

import (
	"log/slog"
	"testing"
)

// The worlds of the ration tests are oreMill: ten households of weight 1, each asking one
// execution of a recipe that draws 3 units of "ore" in home, with the tenure module on and
// the posted rent 2. The pool of home holds the title the test names, so no household holds
// one and no household is both a tenant and a lessor.

// rationConfig is oreMill(5, 10, stock) with the tenure module: rent 2 per unit let, the
// pool of home holding poolBP basis points (the rest is the commons) and the endowment
// closed when closed is set.
func rationConfig(stock, poolBP int64, closed bool) Config {
	c := oreMill(5, 10, stock)
	c.Modules.Estates = &Estates{}
	c.Modules.Tenure = &TenureConfig{
		Terms:    []TenureTerm{{Region: "home", Resource: "ore", HasRent: true, Rent: 2, HasClosed: true, Closed: closed}},
		Holdings: []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindPool}, ShareBP: poolBP}},
	}
	return c
}

// toStart replays one tick of w up to and including the starts of P2.7 and stops there,
// so the state of the production phase can be read before the trades. The world is not
// stepped again.
func toStart(t *testing.T, w *World) {
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
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work() error = %v, want nil", err)
	}
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start() error = %v, want nil", err)
	}
}

// toStartForced is toStart with every household asking one execution of the recipe at
// P2.5 whatever its view says. The default strategy reads the title bound of the view
// and does not ask for what the world would refuse, so the refusal of a start is
// exercised through the request the world admits itself.
func toStartForced(t *testing.T, w *World) {
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
	w.eachLive(func(a *agent) { w.admitJob(a, 0, JobRequest{Recipe: 0, Executions: 1}) })
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start() error = %v, want nil", err)
	}
}

// TestRation_TiersServeOwnCommonsLet verifies the four tiers at the world level on the
// example of spec-land.md 4.3: the pool holds 5 000 basis points of a stock of 10, the
// commons the other 5 000, and ten households each want one execution of 3 units.
//
// Arithmetic: the entitlements are 5 units for the pool and 5 for the commons, with no
// remainder. Own serves nothing (no household holds a title). The commons, 5 units
// against 30 wanted, gives floor(5 / 30) = 0 executions each and the 5 units left go, 3
// at a time, to the first in the order of the key: 1 execution (3 units), 2 left that
// are less than a draw. The let supply is the pool's 5 unused units; the 9 requests left
// ask 27 units, so again 1 execution (3 units) is granted. Offered 5, let 3, idle 2,
// unserved 8 requests of 3 units (24). The rent, 2 per unit, is 6, paid by the one
// tenant to the pool, which holds the only supply.
//
// Requirement: FR-LND-008, FR-LND-009, FR-LND-010, FR-LND-013
func TestRation_TiersServeOwnCommonsLet(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	pool := w.estates.account(0)
	before, ok := w.ledger.Account(pool)
	if ok != nil {
		t.Fatalf("Account(pool) error = %v, want nil", ok)
	}
	toStart(t, w)
	c := w.tenure.rt.counts[0]
	want := tenureCount{commonsUnits: 3, letUnits: 3, offered: 5, unserved: 24, tenants: 1, lessors: 1, paid: 6,
		rentInForce: 2, postedBefore: 2, entitle: 5}
	if c != want {
		t.Errorf("counts of the endowment = %+v, want %+v", c, want)
	}
	if got := w.tenure.rt.commons[0]; got != 5 {
		t.Errorf("commons entitlement = %d, want 5", got)
	}
	if got, want := w.res.stock[0], int64(10-6); got != want {
		t.Errorf("stock after the starts = %d, want %d (two executions of 3 units)", got, want)
	}
	after, _ := w.ledger.Account(pool)
	if got := after.Balance - before.Balance; got != 6 {
		t.Errorf("rent received by the pool = %d, want 6", got)
	}
	if got := c.idleUnits(); got != 2 {
		t.Errorf("idle units = %d, want 2 (offered 5 less let 3)", got)
	}
	var reserved int64
	for _, id := range w.tenure.rt.touched {
		reserved += w.tenure.reservedOf(id)
	}
	if reserved != 6 {
		t.Errorf("rent reserved against the principals = %d, want 6 (the start of the one tenant)", reserved)
	}
}

// TestRation_OwnTierServesTheHolderFirst verifies the own tier: each of ten households
// holds 1 000 basis points, so each is entitled to 10 of a stock of 100 and its request
// of 3 units is served from its own entitlement; no unit is let and no rent is paid.
//
// Requirement: FR-LND-008, FR-LND-017
func TestRation_OwnTierServesTheHolderFirst(t *testing.T) {
	t.Parallel()
	c := rationConfig(100, 0, false)
	c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 1_000}}
	w := newMill(t, c, WithSeed(1), WithAudit())
	toStart(t, w)
	got := w.tenure.rt.counts[0]
	if got.ownUnits != 30 || got.commonsUnits != 0 || got.letUnits != 0 || got.paid != 0 {
		t.Errorf("counts = %+v, want 30 own units and no commons, let or rent", got)
	}
	if got.offered != 70 {
		t.Errorf("let supply = %d, want 70 (10 unused units for each of ten holders less the 3 drawn: 10 * 7)", got.offered)
	}
}

// TestRefusal_NoTitleOnlyWhenClosedLeavesNoOpenUnit verifies the reason no_title of
// spec-land.md 4.5.1: the pool holds every unit of a closed endowment with a stock of 100,
// so the stock could serve an execution of 3 units but no unit is open to a household. Each
// of the ten requests is refused no_title, and nothing is drawn.
//
// Requirement: FR-LND-014
func TestRefusal_NoTitleOnlyWhenClosedLeavesNoOpenUnit(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(100, 10_000, true), WithSeed(1), WithAudit())
	toStartForced(t, w)
	if got := w.tenure.rt.counts[0].noTitle; got != 10 {
		t.Errorf("requests refused no_title = %d, want 10", got)
	}
	if got := startedBy(w); got != 0 {
		t.Errorf("households with a job = %d, want 0", got)
	}
	if got := w.res.stock[0]; got != 100 {
		t.Errorf("stock = %d, want 100 (nothing drawn)", got)
	}
	if got := w.tenure.rt.counts[0].held; got != 100 {
		t.Errorf("held units = %d, want 100 (the unused entitlement a closure keeps)", got)
	}
}

// TestRefusal_NoEndowmentWhenTheStockIsTooThin verifies that no_title is not the reason
// of a refusal that a thin stock caused: a closed endowment holding 2 units, below one
// execution of 3, refuses with no_endowment, which is RES's reason and counts no no_title.
//
// Requirement: FR-LND-014
func TestRefusal_NoEndowmentWhenTheStockIsTooThin(t *testing.T) {
	t.Parallel()
	cap := newCapture()
	w := newMill(t, rationConfig(2, 10_000, true), WithSeed(1), WithAudit(), WithLogger(slog.New(cap)))
	toStartForced(t, w)
	if got := w.tenure.rt.counts[0].noTitle; got != 0 {
		t.Errorf("requests refused no_title = %d, want 0 (floor(A / d) is 0)", got)
	}
	for _, l := range cap.take("recipe not started") {
		if l.strs["reason"] != string(ReasonNoEndowment) {
			t.Errorf("reason of a refusal = %q, want %q", l.strs["reason"], ReasonNoEndowment)
		}
	}
}

// TestStart_RentCapAndTitleCapKeepSeparateReasons verifies that a refusal for tenure is
// reported as no_title or rent_funds and never as no_endowment (spec-land.md 4.5.1): ten
// households are refused a closed endowment no_title, and nine priced out of the let tier
// are refused rent_funds while the tenth is served from the commons.
//
// Requirement: FR-LND-014, FR-LND-016
func TestStart_RentCapAndTitleCapKeepSeparateReasons(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		cfg    func() Config
		reason Reason
		lines  int
	}{
		{"closed", func() Config { return rationConfig(100, 10_000, true) }, ReasonNoTitle, 10},
		{"priced out", func() Config {
			c := rationConfig(10, 5_000, false)
			c.Roles[0].StartMoney = 0
			return c
		}, ReasonRentFunds, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cap := newCapture()
			w := newMill(t, tt.cfg(), WithSeed(1), WithAudit(), WithLogger(slog.New(cap)))
			toStartForced(t, w)
			lines := cap.take("recipe not started")
			if len(lines) != tt.lines {
				t.Fatalf("refusals = %d, want %d", len(lines), tt.lines)
			}
			for _, l := range lines {
				if l.strs["reason"] != string(tt.reason) {
					t.Errorf("reason of a refusal = %q, want %q", l.strs["reason"], tt.reason)
				}
			}
		})
	}
}

// TestView_BoundsMaxExecutionsByTheTitle verifies spec-land.md 4.5.2: the view of a
// household whose principal has no open unit of a closed endowment, though the stock
// could serve an execution, reports MaxExecutions 0 with the reason no_title, so a
// strategy asks nothing the world would refuse.
//
// Requirement: FR-LND-014, FR-LND-022
func TestView_BoundsMaxExecutionsByTheTitle(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(100, 10_000, true), WithSeed(1))
	step(t, w, 1)
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[1], viewWork); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if r := v.Recipes[0]; r.MaxExecutions != 0 || r.Reason != ReasonNoTitle {
		t.Errorf("MaxExecutions, Reason = %d, %q, want 0 and %q", r.MaxExecutions, r.Reason, ReasonNoTitle)
	}
}

// TestRefusal_RentFundsWhenPricedOut verifies the reason rent_funds: households with no
// money ask the let tier at a rent they cannot pay. The commons serves one household (the
// first by the key), and the other nine are clipped to 0 executions: 27 units priced out,
// nine requests refused rent_funds, and the let tier serves nothing.
//
// Requirement: FR-LND-014, FR-LND-016
func TestRefusal_RentFundsWhenPricedOut(t *testing.T) {
	t.Parallel()
	c := rationConfig(10, 5_000, false)
	c.Roles[0].StartMoney = 0
	w := newMill(t, c, WithSeed(1), WithAudit())
	toStart(t, w)
	got := w.tenure.rt.counts[0]
	if got.pricedOut != 27 || got.rentFunds != 9 || got.letUnits != 0 || got.paid != 0 {
		t.Errorf("counts = %+v, want 27 units priced out, 9 requests refused rent_funds and no let", got)
	}
	if n := startedBy(w); n != 1 {
		t.Errorf("households with a job = %d, want 1 (the commons grant)", n)
	}
}

// TestRation_NoTitleEqualsOpenAccess verifies FR-LND-006: with the module on and no title
// the endowment is rationed by RES 5.3, so a world with a Terms row and no holding takes
// the same draws, tick by tick, as the world without the module.
//
// Requirement: FR-LND-006
func TestRation_NoTitleEqualsOpenAccess(t *testing.T) {
	t.Parallel()
	off := newMill(t, oreMill(2, 10, 40), WithSeed(1), WithAudit())
	c := oreMill(2, 10, 40)
	c.Modules.Tenure = &TenureConfig{Terms: []TenureTerm{{Region: "home", Resource: "ore", HasRent: true, Rent: 2}}}
	on := newMill(t, c, WithSeed(1), WithAudit())
	for i := range 12 {
		step(t, off, 1)
		step(t, on, 1)
		if a, b := off.Endowment(0, 0), on.Endowment(0, 0); a != b {
			t.Fatalf("stock after tick %d = %d with the module, want %d without", i, b, a)
		}
		if a, b := tnTotalMoney(t, off), tnTotalMoney(t, on); a != b {
			t.Fatalf("money after tick %d = %d with the module, want %d without", i, b, a)
		}
	}
}

// TestInvariants_Tenure_UnitsNeverExceedStock verifies, over several seeds and the three
// shapes of tenure (a pool title, a closed pool title and a household title), that the
// entitlements sum to the stock, the units drawn in the three tiers equal the units the
// endowment lost to draws and never pass the snapshot stock, and money is conserved.
//
// Requirement: FR-LND-007, FR-LND-011, FR-LND-017, FR-LND-037
func TestInvariants_Tenure_UnitsNeverExceedStock(t *testing.T) {
	t.Parallel()
	shapes := []struct {
		name   string
		pool   int64
		closed bool
		own    int64
	}{{"pool", 3_000, false, 0}, {"closed pool", 3_000, true, 0}, {"households", 0, false, 400}}
	for _, sh := range shapes {
		for seed := uint64(1); seed <= 3; seed++ {
			t.Run(sh.name, func(t *testing.T) {
				c := rationConfig(60, sh.pool, sh.closed)
				if sh.own > 0 {
					c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: sh.own}}
				}
				w := newMill(t, c, WithSeed(seed), WithAudit())
				for range 20 {
					money := tnTotalMoney(t, w)
					tickInspecting(t, w, func() { checkTenureTick(t, w, seed) })
					if got := tnTotalMoney(t, w); got != money {
						t.Fatalf("seed %d: money %d after the tick, want %d (rent moves money and creates none)", seed, got, money)
					}
				}
			})
		}
	}
}

// checkTenureTick asserts the identities of one tick of a titled endowment.
func checkTenureTick(t *testing.T, w *World, seed uint64) {
	t.Helper()
	tn, r := w.tenure, w.res
	var sum int64
	for _, h := range tn.rt.ents[0] {
		sum += h.Entitled
	}
	sum += tn.rt.commons[0]
	if sum != r.snapStock[0] {
		t.Errorf("seed %d: entitlements sum to %d, want the snapshot stock %d", seed, sum, r.snapStock[0])
	}
	c := tn.rt.counts[0]
	if drawn := c.ownUnits + c.commonsUnits + c.letUnits; drawn != r.drawn[0] || drawn > r.snapStock[0] {
		t.Errorf("seed %d: units by tier = %d, want the %d drawn and at most the stock %d", seed, drawn, r.drawn[0], r.snapStock[0])
	}
	if r.granted[0] > r.requested[0] {
		t.Errorf("seed %d: granted %d units, above the %d requested", seed, r.granted[0], r.requested[0])
	}
}
