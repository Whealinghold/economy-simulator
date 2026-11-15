package economysim

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestSettle_EventsInTheOrderOfThePass verifies the events of P1.7a over two regions:
// the LoanEnded events in ascending claim ID, whatever the region and the debtor
// (the debtors are visited in ascending agent ID, so the order of the visits is not
// that of the claims), then one CreditSettled per region in ascending RegionID.
//
// Arithmetic: claim 0 (yonder weaver to agent 6, 100) and claim 1 (home weaver to
// agent 2, 100) are paid in full, and claim 2 (home weaver to agent 3, 100) is not.
// Agent 2 is visited before agent 6, but the events are LoanEnded 0 and LoanEnded 1,
// then CreditSettled of region 0 (claims 1 and 2 due, one repaid, one defaulted)
// and of region 1 (claim 0 due and repaid).
//
// Requirement: FR-CRD-071, FR-CRD-073
func TestSettle_EventsInTheOrderOfThePass(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 200, yonderWeaver: 100})
	plantLoan(t, w, yonderWeaver, yonderA, 100, 0, 2)
	plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	plantLoan(t, w, homeWeaver, homeB, 100, 0, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 100, yonderA: 100})
	rec.events = nil
	w.p17aSettle()
	want := []Event{
		LoanEnded{Tick: 2, Claim: 0, Origin: OriginLoan, Creditor: yonderWeaver, Debtor: yonderA, Region: 1, CreditorRegion: 1, Principal: 100, Cause: CauseRepaid},
		LoanEnded{Tick: 2, Claim: 1, Origin: OriginLoan, Creditor: homeWeaver, Debtor: homeA, Principal: 100, Cause: CauseRepaid},
		CreditSettled{Tick: 2, Region: 0, Due: 2, Repaid: 1, Defaulted: 1, PrincipalPaid: 100, Received: 100, Arrears: 100},
		CreditSettled{Tick: 2, Region: 1, Due: 1, Repaid: 1, PrincipalPaid: 100, Received: 100},
	}
	if !reflect.DeepEqual(rec.events, want) {
		t.Errorf("events of the pass = %+v, want %+v", rec.events, want)
	}
}

// TestSettle_TiesBreakByTheKeyOfTheClaim verifies that the unit left over by the
// largest remainder goes, between two equal fractional parts, to the claim with the
// smaller K(t, claim ID), not to the smaller ID (spec-credit.md 8.3, FR-CRD-032).
//
// Arithmetic: two claims owe 100 each and the pot is 101: each share is 50 and the
// fractional parts are both 0.5, so one claim gets the 51st unit. The IDs are chosen
// (by planting small loans first) so that the claim with the larger ID has the smaller
// key at tick 2.
//
// Requirement: FR-CRD-032, FR-CRD-022
func TestSettle_TiesBreakByTheKeyOfTheClaim(t *testing.T) {
	t.Parallel()
	first := int64(0)
	for ordering.Key(2, first) <= ordering.Key(2, first+1) {
		first++
	}
	w, _ := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 200 + first})
	for range first {
		plantLoan(t, w, homeWeaver, yonderB, 1, 0, 99)
	}
	a := plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	b := plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 101})
	w.p17aSettle()
	pa, _, _ := claimOwed(t, w, a)
	pb, _, _ := claimOwed(t, w, b)
	if pa != 50 || pb != 49 {
		t.Errorf("claims %d and %d owe %d and %d, want 50 and 49: the larger ID has the smaller key", a, b, pa, pb)
	}
}

// TestCredit_WelfareInterest verifies that interest paid is a cost of the debtor
// (netted from its income like an input) and interest received is unearned income of
// an agent creditor, and that the principal is neither.
//
// Arithmetic: the claim is 3 000 of principal and 300 of interest, paid in full. The
// debtor's record gains an input spend of 300 and the creditor's an unearned 300.
//
// Requirement: FR-CRD-054
func TestCredit_WelfareInterest(t *testing.T) {
	t.Parallel()
	w, _ := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 3_000})
	plantLoan(t, w, homeWeaver, homeA, 3_000, 300, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 3_300})
	w.p17aSettle()
	d, c := w.welfare.at(int(homeA)), w.welfare.at(int(homeWeaver))
	if d.inputSpend != 300 || d.unearned != 0 || d.receipts != 0 {
		t.Errorf("debtor flows = input %d, unearned %d, receipts %d, want 300, 0, 0", d.inputSpend, d.unearned, d.receipts)
	}
	if c.unearned != 300 || c.inputSpend != 0 || c.receipts != 0 {
		t.Errorf("creditor flows = unearned %d, input %d, receipts %d, want 300, 0, 0", c.unearned, c.inputSpend, c.receipts)
	}
}

// TestCredit_ClaimsOverdueWarnsAtTheTenthTick verifies the Warn "claims overdue":
// not before the debtors of a region have owed overdue claims at the end of P1.7a in
// ten consecutive ticks, once at the tenth with the region, the count and the owed,
// never per claim; and the Debug lines "claim overdue" and "loan ended".
//
// Arithmetic: the loan is due at tick 2 and unpaid by a shielded debtor, so it is
// overdue at the end of ticks 2 to 11, ten ticks, and the tenth is tick 11.
//
// Requirement: FR-CRD-076, FR-CRD-075
func TestCredit_ClaimsOverdueWarnsAtTheTenthTick(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := creditWorld(t, 1_000, 0, WithLogger(slog.New(lg)), WithAudit())
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 1_000})
	plantLoan(t, w, homeWeaver, homeA, 1_000, 100, 2)
	step(t, w, 1)
	lg.take("") // forget the lines of construction
	for tick := 2; tick <= 11; tick++ {
		stepSet(t, w, map[AgentID]int64{homeA: 5_000, homeWeaver: 5_000})
		warns := lg.take("claims overdue")
		if tick < 11 && len(warns) != 0 {
			t.Fatalf("tick %d: %d Warn lines, want none before the tenth tick", tick, len(warns))
		}
		if tick == 11 {
			wantLine(t, warns, slog.LevelWarn, map[string]int64{"region": 0, "claims": 1, "arrears": 1_100})
		}
	}
}

// TestCredit_DebugLines verifies the Debug lines of a claim that goes overdue and of
// one that ends, with their keys.
//
// Requirement: FR-CRD-075
func TestCredit_DebugLines(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := creditWorld(t, 0, 0, WithLogger(slog.New(lg)))
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 200})
	late := plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	paid := plantLoan(t, w, homeWeaver, homeB, 100, 0, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeB: 100})
	*lg.lines = nil
	w.p17aSettle()
	lines := func(msg string) []logRec {
		var out []logRec
		for _, l := range *lg.lines {
			if l.msg == msg {
				out = append(out, l)
			}
		}
		return out
	}
	wantLine(t, lines("claim overdue"), slog.LevelDebug, map[string]int64{"claim": late, "agent": int64(homeA), "owed": 100})
	wantLine(t, lines("loan ended"), slog.LevelDebug, map[string]int64{"claim": paid, "lost": 0})
}

// wantLine fails the test unless the lines are one, at the level, with the integer
// attributes.
func wantLine(t *testing.T, lines []logRec, level slog.Level, attrs map[string]int64) {
	t.Helper()
	if len(lines) != 1 || lines[0].level != level {
		t.Fatalf("lines = %+v, want one at %v", lines, level)
	}
	for k, v := range attrs {
		if lines[0].attrs[k] != v {
			t.Errorf("line %q has %s = %d, want %d", lines[0].msg, k, lines[0].attrs[k], v)
		}
	}
}

// TestCredit_NoDraws verifies that the credit module takes no word of the world's
// stream: no non-test file of the module names the stream or a source of randomness
// (spec-credit.md 12.2), so the words of a tick are the same with the module on and
// off, whatever its parameters and claims.
//
// Requirement: FR-CRD-062
//
// Guard: a draw added to the settlement or the waterfall would shift every later
// word of the tick, and with it every golden of every pack.
func TestCredit_NoDraws(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("credit*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("Glob(credit*.go) = %v, %v, want the module's files", files, err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "rand") {
				t.Errorf("%s imports %s, want no source of randomness", name, imp.Path.Value)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "rnd" || sel.Sel.Name == "keyed") {
				t.Errorf("%s reads %s, want no draw from the world's stream", name, sel.Sel.Name)
			}
			return true
		})
	}
}

// TestCredit_AllocsPerRun verifies that the settlement allocates nothing once its
// scratch has grown, with the default logger and observer, on the path that pays:
// each run tops the debtor's balance up to the shield plus 1 and settles, so each
// pays 1 unit of interest.
//
// Requirement: NFR-CRD-001, FR-CRD-075
func TestCredit_AllocsPerRun(t *testing.T) {
	w := creditWorld(t, 2, 0)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 1_000})
	plantLoan(t, w, homeWeaver, homeA, 1_000, 100, 2)
	step(t, w, 1)
	w.valuePrices()
	shield := 2 * w.creditUpkeep(&w.agents[homeA])
	setBalances(t, w, map[AgentID]int64{homeA: shield + 1})
	pool := w.estates.account(0)
	run := func() {
		if have := heldMoney(t, w, homeA); have < shield+1 {
			mustClaim(w.ledger.Transfer(pool, w.agents[homeA].account, shield+1-have, ledger.StartMoney), "the top-up of the debtor")
		}
		w.p17aSettle()
	}
	run() // grows the scratch
	if got := testing.AllocsPerRun(5, run); got != 0 {
		t.Errorf("AllocsPerRun(p17aSettle) = %v, want 0", got)
	}
}

// TestCredit_Observation verifies that a run with a logger at Debug, an observer and
// the audit on gives the same state as one with none of them: the same bytes of the
// checkpoint after a scripted run with claims that fall due, go overdue, are paid and
// are written off, and a removal that runs the waterfall.
//
// Requirement: FR-CRD-075
func TestCredit_Observation(t *testing.T) {
	t.Parallel()
	run := func(opts ...Option) []byte {
		w := creditWorld(t, 1, 3, opts...)
		step(t, w, 1)
		setBalances(t, w, map[AgentID]int64{homeWeaver: 600})
		plantLoan(t, w, homeWeaver, homeA, 500, 50, 2)
		plantLoan(t, w, homeWeaver, homeB, 100, 10, 3)
		for range 5 {
			stepSet(t, w, map[AgentID]int64{homeA: 700, homeB: 20})
		}
		if err := w.removeAgent(&w.agents[homeA], CauseConsumer); err != nil {
			t.Fatalf("removeAgent error = %v, want nil", err)
		}
		cp, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("Checkpoint error = %v, want nil", err)
		}
		return cp.b
	}
	plain := run()
	watched := run(WithObserver(&evLog{}), WithLogger(slog.New(newCapture())), WithAudit())
	if !bytes.Equal(plain, watched) {
		t.Errorf("the checkpoint with a logger, an observer and the audit has %d bytes and differs from the one without (%d)", len(watched), len(plain))
	}
}

// TestCredit_SummaryNamesTheModule verifies that the Info line "world created" lists
// credit among the modules that are on and carries the counts of the agents with a
// Lend block and with a Borrow block (spec-credit.md 16.2).
//
// Arithmetic: in the credit world the five millers have a Borrow block and the
// two weavers a Lend block.
//
// Requirement: FR-CRD-001
func TestCredit_SummaryNamesTheModule(t *testing.T) {
	t.Parallel()
	c := creditConfig(0, 0)
	c.Roles[0].Credit = &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1_000, MaxDebtTicks: 10}}
	c.Roles[1].Credit = &CreditTerms{Lend: &LendTerms{RateBP: 500, MaxTermTicks: 30, ShareBP: 5_000}}
	lg := newCapture()
	newMill(t, c, WithSeed(exitSeed), WithLogger(slog.New(lg)))
	lines := lg.take("world created")
	if len(lines) != 1 || lines[0].attrs["lenders"] != 2 || lines[0].attrs["borrowers"] != 5 {
		t.Fatalf(`"world created" lines = %+v, want one with lenders 2 and borrowers 5`, lines)
	}
}

// TestSettle_PayoutClaimEndsWithItsOwnEvent verifies that a payout claim paid in full
// ends with a LoanEnded of origin payout, cause repaid, and that CreditSettled, which
// counts loans only, is not emitted for it.
//
// Requirement: FR-CRD-071, FR-CRD-073
func TestSettle_PayoutClaimEndsWithItsOwnEvent(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 0)
	id := plantPayout(t, w, homeB, homeA, 500, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 500})
	rec.events = nil
	w.p17aSettle()
	wantOne(t, rec, LoanEnded{Tick: 2, Claim: ClaimID(id), Origin: OriginPayout, Creditor: homeB, Debtor: homeA, Principal: 500, Cause: CauseRepaid})
	if got := of[CreditSettled](rec); len(got) != 0 {
		t.Errorf("CreditSettled = %+v for a payout claim alone, want none", got)
	}
	if got := heldMoney(t, w, homeB); got != 500 {
		t.Errorf("creditor holds %d, want the 500 paid", got)
	}
}
