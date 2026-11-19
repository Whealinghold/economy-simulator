package economysim

import (
	"log/slog"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the readers of the rights (spec-institutions.md 5): a start, an order, a
// delivery offer and take. They build a world from orgToy, run the membership round
// directly so that the members are known, and call the world's own admission of a request
// or of an order.

// orgRightsToy is orgToy with the rights the tests read. Grain is the one commodity and
// both roles farm it. Region b is the one with non-members: the guild has 5 places there
// for 8 smiths, so 5 are members and 3 are not; it holds the smiths of a too (4 of 4).
//
// The rights: farm_b reserves the start of farm in b to the guild; mine_b reserves mine,
// a recipe of the farmers only, in b to the guild; sell_b reserves the sale of grain in b
// to the guild; buy_c is a prohibition of the purchase of grain in c (no body). The
// recipe forage is on the own-use list of both roles (the subsistence module), not on any
// role's list of recipes, with no right; forage_b reserves it in b to the guild.
func orgRightsToy() Config {
	c := orgToy()
	mine := c.Recipes[0]
	mine.ID = "mine"
	forage := c.Recipes[0]
	forage.ID, forage.Sited = "forage", true
	for _, g := range []string{"a", "b", "c"} {
		c.Yields = append(c.Yields, YieldProfile{Region: g, Recipe: "forage", HasAll: true, All: 10_000})
	}
	c.Recipes = append(c.Recipes, forage, mine)
	c.Roles[orgFarmer].Recipes = []string{"farm", "mine"}
	own := DefaultSubsistence()
	own.Recipes = []string{"forage"}
	c.Modules.Subsistence = &own
	c.Modules.Institutions.Rights = []Right{
		{ID: "farm_b", Body: "guild", Activity: ActivityProduce, Subject: "farm", Region: "b"},
		{ID: "forage_b", Body: "guild", Activity: ActivityProduce, Subject: "forage", Region: "b"},
		{ID: "mine_b", Body: "guild", Activity: ActivityProduce, Subject: "mine", Region: "b"},
		{ID: "sell_b", Body: "guild", Activity: ActivitySell, Subject: "grain", Region: "b"},
		{ID: "buy_c", Activity: ActivityBuy, Subject: "grain", Region: "c"},
	}
	return c
}

// orgReady builds the world of c with an event log and a capturing logger, runs the
// membership round and the positions of a tick up to the work of P2.5, and returns them.
func orgReady(t *testing.T, c Config) (*World, *evLog, capture) {
	t.Helper()
	rec, lg := &evLog{}, newCapture()
	w := newMill(t, c, WithObserver(rec), WithLogger(slog.New(lg)))
	w.org.round(w)
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
	return w, rec, lg
}

// orgOutsider returns the first smith of region g that is not a member of body 0, and
// orgInsider the first that is.
func orgOutsider(t *testing.T, w *World, g RegionID) *agent {
	t.Helper()
	for _, i := range orgAgents(w, g, orgSmith) {
		if a := &w.agents[i]; !w.org.isMember(i, a.role, g, 0) {
			return a
		}
	}
	t.Fatalf("no smith of region %d is a non-member of the guild", g)
	return nil
}

func orgInsider(t *testing.T, w *World, g RegionID) *agent {
	t.Helper()
	for _, i := range orgAgents(w, g, orgSmith) {
		if a := &w.agents[i]; w.org.isMember(i, a.role, g, 0) {
			return a
		}
	}
	t.Fatalf("no smith of region %d is a member of the guild", g)
	return nil
}

// orgRecipe returns the RecipeID of the recipe with the ID.
func orgRecipe(t *testing.T, w *World, id string) RecipeID {
	t.Helper()
	for i := range w.cfg.Recipes {
		if w.cfg.Recipes[i].ID == id {
			return RecipeID(i)
		}
	}
	t.Fatalf("no recipe %q in the configuration", id)
	return 0
}

// orgReasonLines returns the "recipe not started" lines the logger captured, by reason.
func orgReasonLines(lg capture, reason Reason) int {
	n := 0
	for _, l := range *lg.lines {
		if l.msg == "recipe not started" && l.strs["reason"] == string(reason) {
			n++
		}
	}
	return n
}

// TestStart_NotPermittedAfterNotOnRoleBeforeK0 verifies the place of the refusal of a
// start (spec-institutions.md 5.2, 5.6.1): a recipe a right bars is refused with
// not_permitted, a recipe off the role's list is still refused as not_on_role first, the
// refusal starts nothing and a member of the holder is admitted as before.
//
// Arithmetic: in b the guild holds the 5 smiths admitted first and 3 smiths are not
// members. farm_b bars the 3 from farm and not the 5. A smith asking for mine, which is
// the farmers' recipe, is not_on_role whether or not it is a member, and mine_b does not
// show in its count.
//
// Requirement: FR-ORG-020, FR-PRO-048
func TestStart_NotPermittedAfterNotOnRoleBeforeK0(t *testing.T) {
	t.Parallel()
	w, rec, lg := orgReady(t, orgRightsToy())
	farm, mine := orgRecipe(t, w, "farm"), orgRecipe(t, w, "mine")
	out, in := orgOutsider(t, w, orgB), orgInsider(t, w, orgB)
	w.admitJob(out, orgB, JobRequest{Recipe: farm, Executions: 1})
	w.admitJob(in, orgB, JobRequest{Recipe: farm, Executions: 1})
	w.admitJob(out, orgB, JobRequest{Recipe: mine, Executions: 1})
	if w.pr.asks[out.id].set {
		t.Errorf("the non-member smith has an admitted request for farm in b, want it refused")
	}
	if !w.pr.asks[in.id].set {
		t.Errorf("the member smith has no admitted request for farm in b, want it admitted")
	}
	if n := orgReasonLines(lg, ReasonNotPermitted); n != 1 {
		t.Errorf("Debug lines with the reason not_permitted = %d, want 1 (the non-member's farm)", n)
	}
	if n := orgReasonLines(lg, ReasonNotOnRole); n != 1 {
		t.Errorf("Debug lines with the reason not_on_role = %d, want 1 (the smith's mine)", n)
	}
	w.p72bInstitutions()
	got := of[RightRefused](rec)
	if len(got) != 1 || got[0].Region != orgB || got[0].Produce != 1 || got[0].Sell != 0 {
		t.Errorf("RightRefused = %+v, want one event of b with Produce 1", got)
	}
	if got[0].Right != orgRight(t, w, "farm_b") {
		t.Errorf("RightRefused.Right = %d, want farm_b (%d)", got[0].Right, orgRight(t, w, "farm_b"))
	}
}

// orgPair returns a member and a non-member of the guild in region b, as the principal
// and the executor of a job whose Who the test sets.
func orgPair(t *testing.T, w *World) (member, outsider *agent) {
	t.Helper()
	return orgInsider(t, w, orgB), orgOutsider(t, w, orgB)
}

// TestStart_WhoPrincipalExecutorBoth verifies whose membership a produce right tests
// (spec-institutions.md 5.3.1): with principal the account whose inputs the job uses, with
// executor the household that works, with both each of them.
//
// Arithmetic: of the four pairs (principal, executor) of a member and a non-member, a
// right with Who principal bars the pair when the principal is the non-member (2 of 4:
// (out, in) and (out, out)), one with executor when the executor is (2 of 4: (in, out)
// and (out, out)), and one with both when either is (3 of 4: all but (in, in)).
//
// Requirement: FR-ORG-020
func TestStart_WhoPrincipalExecutorBoth(t *testing.T) {
	t.Parallel()
	w, _, _ := orgReady(t, orgRightsToy())
	in, out := orgPair(t, w)
	farm, id := orgRecipe(t, w, "farm"), orgRight(t, w, "farm_b")
	cases := []struct {
		name     string
		who      uint8
		pr, ex   *agent
		wantBarr bool
	}{
		{"principal: member principal, outsider executor", whoPrincipal, in, out, false},
		{"principal: outsider principal, member executor", whoPrincipal, out, in, true},
		{"executor: member principal, outsider executor", whoExecutor, in, out, true},
		{"executor: outsider principal, member executor", whoExecutor, out, in, false},
		{"both: both members", whoBoth, in, in, false},
		{"both: outsider principal", whoBoth, out, in, true},
		{"both: outsider executor", whoBoth, in, out, true},
	}
	for _, tc := range cases {
		w.org.rights.who[id] = tc.who
		if _, got := w.org.startBarred(w, tc.ex, tc.pr, false, orgB, farm); got != tc.wantBarr {
			t.Errorf("startBarred(%s) = %v, want %v", tc.name, got, tc.wantBarr)
		}
	}
}

// TestStart_TraineeExemptFromExecutor verifies that the executor test is not applied to a
// trainee, whose principal, the trainer, is tested as for principal whatever Who says
// (spec-institutions.md 5.3.2).
//
// Arithmetic: a trainee that is not a member, under a member trainer, is permitted by each
// of the three values of Who; under a trainer that is not a member it is barred by each,
// because the trainer is tested even for executor.
//
// Requirement: FR-ORG-021
func TestStart_TraineeExemptFromExecutor(t *testing.T) {
	t.Parallel()
	w, _, _ := orgReady(t, orgRightsToy())
	in, out := orgPair(t, w)
	farm, id := orgRecipe(t, w, "farm"), orgRight(t, w, "farm_b")
	for _, who := range []uint8{whoPrincipal, whoExecutor, whoBoth} {
		w.org.rights.who[id] = who
		if _, got := w.org.startBarred(w, out, in, true, orgB, farm); got {
			t.Errorf("startBarred(trainee outsider, member trainer, who %d) = barred, want permitted", who)
		}
		if _, got := w.org.startBarred(w, in, out, true, orgB, farm); !got {
			t.Errorf("startBarred(trainee member, outsider trainer, who %d) = permitted, want barred", who)
		}
	}
}

// TestStart_OwnUseExemptUnlessBound verifies that a job of an own-use recipe is not read
// against a produce right unless OwnUseBound is set, and that a bound own-use refusal
// counts the household as blocked (spec-institutions.md 5.5.2; spec-subsistence.md 14.1).
//
// Arithmetic: forage is on the own-use list and forage_b reserves it in b to the guild. A
// non-member smith asking for one execution is admitted with OwnUseBound false and refused
// with it true; the household is counted blocked once in its cell, in the slot of any other
// reason.
//
// Requirement: FR-ORG-022, FR-SUB-023
func TestStart_OwnUseExemptUnlessBound(t *testing.T) {
	t.Parallel()
	for _, bound := range []bool{false, true} {
		c := orgRightsToy()
		c.Modules.Institutions.OwnUseBound = bound
		w, _, _ := orgReady(t, c)
		out, forage := orgOutsider(t, w, orgB), orgRecipe(t, w, "forage")
		w.admitJob(out, orgB, JobRequest{Recipe: forage, Executions: 1})
		cell := int(orgB)*len(w.cfg.Roles) + int(out.role)
		if got := w.pr.asks[out.id].set; got == bound {
			t.Errorf("OwnUseBound %v: own-use request admitted = %v, want %v", bound, got, !bound)
		}
		if got, want := w.sub.blocked[cell] > 0, bound && w.sub.on(out); got != want {
			t.Errorf("OwnUseBound %v: own-use household blocked = %v, want %v", bound, got, want)
		}
	}
}

// TestSubsistence_OwnUseBoundBlocks verifies the reason that blocks an own-use start:
// not_permitted is one, with no slot of its own (the hashed shape of the state is
// unchanged).
//
// Requirement: FR-SUB-023
func TestSubsistence_OwnUseBoundBlocks(t *testing.T) {
	t.Parallel()
	if !blocksOwnUse(ReasonNotPermitted) {
		t.Errorf("blocksOwnUse(not_permitted) = false, want true")
	}
	if got, want := blockReasonAt(ReasonNotPermitted), len(blockReasons)-1; got != want {
		t.Errorf("blockReasonAt(not_permitted) = %d, want the any-other slot %d", got, want)
	}
}

// TestRight_NotReadAtContinuationCompletionHolding verifies that a right is read at a
// start and never again: a household whose membership lapses while its job runs finishes
// the job, and the output is its own (spec-institutions.md 5.5.1).
//
// Arithmetic: farm lasts 3 ticks. The member starts it in b (tick 0 is the first step of
// the job), the membership is cleared, and two more steps of P2.1 complete it: one
// RecipeExecuted of the household, and no RightRefused.
//
// Requirement: FR-ORG-018
func TestRight_NotReadAtContinuationCompletionHolding(t *testing.T) {
	t.Parallel()
	c := orgRightsToy()
	c.Recipes[0].Duration = 3
	w, rec, _ := orgReady(t, c)
	in := orgInsider(t, w, orgB)
	w.admitJob(in, orgB, JobRequest{Recipe: orgRecipe(t, w, "farm"), Executions: 1})
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start() error = %v, want nil", err)
	}
	if !in.hasJob() {
		t.Fatalf("the member has no job after P2.7, want the recipe started")
	}
	w.org.member[in.id], w.org.memberRegion[in.id] = 0, 0
	for range 2 {
		w.welfare.reset(int(in.id)) // one job a tick: the scratch of the start's tick is not the next one's
		w.p21Complete()
	}
	if got := len(of[RecipeExecuted](rec)); got != 1 {
		t.Errorf("RecipeExecuted events = %d, want 1 (the job of the lapsed member completes)", got)
	}
	w.p72bInstitutions()
	if got := of[RightRefused](rec); len(got) != 0 {
		t.Errorf("RightRefused = %+v, want none: the right is not read at a completion", got)
	}
}

// TestStart_NothingConsumed verifies that a refused start takes nothing: the ledger of a
// household refused for not_permitted is as before, and no job is open.
//
// Requirement: FR-ORG-020
func TestStart_NothingConsumed(t *testing.T) {
	t.Parallel()
	w, _, _ := orgReady(t, orgRightsToy())
	out := orgOutsider(t, w, orgB)
	before, err := w.ledger.Account(out.account)
	if err != nil {
		t.Fatalf("Account() error = %v, want nil", err)
	}
	w.admitJob(out, orgB, JobRequest{Recipe: orgRecipe(t, w, "farm"), Executions: 1})
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start() error = %v, want nil", err)
	}
	after, _ := w.ledger.Account(out.account)
	if before != after || out.hasJob() {
		t.Errorf("account %+v, job %v after a refused start, want %+v and no job", after, out.hasJob(), before)
	}
}

// orgIntent is an intent of one unit of grain at a limit the agent can pay, on the side.
func orgIntent(side Side) Intent {
	return Intent{Commodity: 0, Side: side, Quantity: 1, Limit: 100}
}

// orgRejected returns the reasons of the OrderRejected events of the agent, in order.
func orgRejected(rec *evLog, a *agent) []Reason {
	var out []Reason
	for _, e := range of[OrderRejected](rec) {
		if e.Agent == a.id {
			out = append(out, e.Reason)
		}
	}
	return out
}

// TestOrder_NotPermittedAfterNotTradable verifies that a right bars an order of a
// household that is not a member of the holder, with the reason not_permitted, and not one
// of a member (spec-institutions.md 5.2, 5.6.2).
//
// Arithmetic: sell_b reserves the sale of grain in b to the guild. A member's ask of one
// unit passes the right and is refused for what it does not hold (insufficient_stock, a
// later row), the non-member smith's and the farmer's (a role on no body's list) are refused
// not_permitted first. A bid is not an ask: the purchase of grain in b has no right.
//
// Requirement: FR-ORG-024, FR-AGT-085
func TestOrder_NotPermittedAfterNotTradable(t *testing.T) {
	t.Parallel()
	w, rec, _ := orgReady(t, orgRightsToy())
	in, out := orgPair(t, w)
	farmer := &w.agents[orgAgents(w, orgB, orgFarmer)[0]]
	for _, a := range []*agent{in, out, farmer} {
		w.submit(a, orgB, orgIntent(Ask))
	}
	w.submit(out, orgB, orgIntent(Bid))
	if got := orgRejected(rec, out); len(got) != 1 || got[0] != ReasonNotPermitted {
		t.Errorf("reasons of the non-member's orders = %v, want [not_permitted] (the ask only: the bid is open)", got)
	}
	if got := orgRejected(rec, farmer); len(got) != 1 || got[0] != ReasonNotPermitted {
		t.Errorf("reasons of the farmer's orders = %v, want [not_permitted]", got)
	}
	if got := orgRejected(rec, in); len(got) != 1 || got[0] != ReasonInsufficientStock {
		t.Errorf("reasons of the member's orders = %v, want [insufficient_stock]: the right is held", got)
	}
}

// TestOrder_RefusedHasNoOutcome verifies that an order refused as not permitted is a
// rejection with no outcome: the market keeps no record of it, so the household's beliefs
// are not revised for it (spec-institutions.md 5.6.2, spec-clearing.md 7.3).
//
// Requirement: FR-ORG-024
func TestOrder_RefusedHasNoOutcome(t *testing.T) {
	t.Parallel()
	w, _, _ := orgReady(t, orgRightsToy())
	mi := w.markets.index(orgB, 0)
	before := len(w.markets.intents[mi])
	w.submit(orgOutsider(t, w, orgB), orgB, orgIntent(Ask))
	if got := len(w.markets.intents[mi]); got != before {
		t.Errorf("records of the market after a refused order = %d, want %d: a rejection has no outcome", got, before)
	}
}

// TestRight_ProhibitionBarsAllButPseudoAgents verifies a right with no body: buy_c bars
// every household of c from buying grain, whatever its role, and counts each refusal under
// the right and the activity (spec-institutions.md 5.1.3, 5.6.3).
//
// Arithmetic: c holds 5 smiths, in no body; each of two bids is refused, so RightRefused
// of (c, buy_c) has Buy 2 and nothing else.
//
// Requirement: FR-ORG-016, FR-ORG-040
func TestRight_ProhibitionBarsAllButPseudoAgents(t *testing.T) {
	t.Parallel()
	w, rec, _ := orgReady(t, orgRightsToy())
	ids := orgAgents(w, orgC, orgSmith)
	for _, i := range ids[:2] {
		w.submit(&w.agents[i], orgC, orgIntent(Bid))
	}
	w.p72bInstitutions()
	got := of[RightRefused](rec)
	want := RightRefused{Tick: w.Tick(), Region: orgC, Right: orgRight(t, w, "buy_c"), Buy: 2}
	if len(got) != 1 || got[0] != want {
		t.Errorf("RightRefused = %+v, want [%+v]", got, want)
	}
}

// TestRight_EstateAndCounterpartyExempt verifies that a pseudo-agent is never refused:
// the estate of b sells a lot of grain although sell_b reserves the sale to the guild, so
// it can liquidate what it inherits (spec-institutions.md 5.5.2).
//
// Requirement: FR-ORG-023
func TestRight_EstateAndCounterpartyExempt(t *testing.T) {
	t.Parallel()
	w, rec, _ := orgReady(t, orgRightsToy())
	src, ok := w.ledger.Boundary(ledger.Initial, 0)
	if !ok {
		t.Fatalf("no Initial boundary for grain")
	}
	if err := w.ledger.Move(src, w.estates.account(orgB), 0, 3, ledger.Initial); err != nil {
		t.Fatalf("Move() error = %v, want nil", err)
	}
	if err := w.p34Estates(); err != nil {
		t.Fatalf("p34Estates() error = %v, want nil", err)
	}
	if got := len(w.markets.books[w.markets.index(orgB, 0)].entries); got != 1 {
		t.Errorf("orders in the book of grain in b after the estate's ask = %d, want 1", got)
	}
	if got := of[OrderRejected](rec); len(got) != 0 {
		t.Errorf("OrderRejected = %+v, want none for the estate", got)
	}
}
