package economysim

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// removalWorld builds the credit world with the shield 2 and an observer at tick 1,
// where the claims of a removal test are planted with a due tick far away.
func removalWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return settleAt(t, 2, 0)
}

// removalEvents returns the events of a removal that concern claims and the
// bankruptcy, in the order they were emitted. The lots of goods the agent leaves are
// not the subject of these tests, so GoodsListed is left out.
func removalEvents(rec *evLog) []Event {
	var out []Event
	for _, e := range rec.events {
		switch ev := e.(type) {
		case LoanEnded, LoanPassed:
			out = append(out, e)
		case AgentBankrupted:
			ev.GoodsListed = 0
			out = append(out, ev)
		}
	}
	return out
}

// TestCredit_Waterfall verifies the removal of a debtor (spec-credit.md 9.1, with the
// worked case of 20.2): its whole balance pays its claims with no shield, pro rata
// by what each owes with the largest remainder and interest first, what is unpaid is
// written off with the cause debtor_removed, the estate receives only what is left,
// and the events are LoanEnded per claim in ascending claim ID before AgentBankrupted.
//
// Arithmetic: the claims owe 500 (50 of interest, 450 of principal) and 700 (50 and
// 650). With a balance of 800, pay = min(800, 1 200) = 800, and the shares are
// floor(800 * 500 / 1 200) = 333 and floor(800 * 700 / 1 200) = 466 plus the one unit
// left to the larger fraction (466.67 against 333.33): 333 and 467. They pay 50 of
// interest each and then 283 and 417 of principal, so 167 and 233 are lost and the
// estate receives nothing. With 1 500 both are paid in full and the estate receives
// 1 500 - 1 200 = 300 under estate_money. ShieldTicks is 2 and plays no part.
//
// Requirement: FR-CRD-040, FR-CRD-041, FR-CRD-043, FR-CRD-071
func TestCredit_Waterfall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		balance     int64
		shareX      [2]int64 // principal and interest paid on the first claim
		shareY      [2]int64
		lostX, lost int64
		toEstate    int64
	}{
		{"a balance short of the debts", 800, [2]int64{283, 50}, [2]int64{417, 50}, 167, 233, 0},
		{"a balance above the debts", 1_500, [2]int64{450, 50}, [2]int64{650, 50}, 0, 0, 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := removalWorld(t)
			setBalances(t, w, map[AgentID]int64{homeWeaver: 450, homeB: 650})
			x := plantLoan(t, w, homeWeaver, homeA, 450, 50, 90)
			y := plantLoan(t, w, homeB, homeA, 650, 50, 90)
			setBalances(t, w, map[AgentID]int64{homeA: tt.balance})
			pool, money := w.estates.pool(w.ledger, 0), moneyTotal(w)
			rec.events = nil
			if err := w.removeAgent(&w.agents[homeA], CauseConsumer); err != nil {
				t.Fatalf("removeAgent error = %v, want nil", err)
			}
			want := []Event{
				LoanEnded{Tick: 1, Claim: ClaimID(x), Origin: OriginLoan, Creditor: homeWeaver, Debtor: homeA, Principal: 450,
					PaidPrincipal: Money(tt.shareX[0]), PaidInterest: Money(tt.shareX[1]), LostPrincipal: Money(tt.lostX), Cause: CauseDebtorRemoved},
				LoanEnded{Tick: 1, Claim: ClaimID(y), Origin: OriginLoan, Creditor: homeB, Debtor: homeA, Principal: 650,
					PaidPrincipal: Money(tt.shareY[0]), PaidInterest: Money(tt.shareY[1]), LostPrincipal: Money(tt.lost), Cause: CauseDebtorRemoved},
				AgentBankrupted{Tick: 1, Agent: homeA, Region: 0, Role: 0, Weight: 1, Cause: CauseConsumer, Money: Money(tt.toEstate)},
			}
			if got := removalEvents(rec); !reflect.DeepEqual(got, want) {
				t.Errorf("events of the removal = %+v, want %+v", got, want)
			}
			if got := heldMoney(t, w, homeWeaver); got != tt.shareX[0]+tt.shareX[1] {
				t.Errorf("first creditor holds %d, want its share %d", got, tt.shareX[0]+tt.shareX[1])
			}
			if got := w.estates.pool(w.ledger, 0) - pool; int64(got) != tt.toEstate {
				t.Errorf("the estate received %d, want %d", got, tt.toEstate)
			}
			if got := moneyTotal(w); got != money {
				t.Errorf("money over the accounts = %d after the removal, want %d", got, money)
			}
			if _, _, open := claimOwed(t, w, x); open {
				t.Error("first claim is open after the removal, want ended")
			}
			auditNow(t, w)
		})
	}
}

// TestCredit_PassToEstate verifies the removal of an agent that is a debtor and a
// creditor (spec-credit.md 9.2.3): its debts are paid from its balance first (2d),
// its claims as creditor then pass to the estate of the region of its account in
// ascending claim ID with no money moved (2e), LoanPassed carries the estate's
// region, and the estate is then the creditor, paid at P1.7a under the pool.
//
// Arithmetic: agent 3 owes the weaver 300 (claim 0), lends agent 4 200 (claim 1) and
// lends agent 6, of the other region, 100 (claim 2), and holds 100 at its removal. Its
// 100 pay claim 0 (200 are lost), claims 1 and 2 pass to the home estate, region 0,
// whatever the region of their debtors, and nothing moves with them. Agent 4 then
// repays 200 at tick 2: the pool receives it.
//
// Requirement: FR-CRD-042, FR-CRD-043, FR-CRD-072, FR-CRD-073
func TestCredit_PassToEstate(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 300, homeB: 300})
	debt := plantLoan(t, w, homeWeaver, homeB, 300, 0, 90)
	toC := plantLoan(t, w, homeB, homeC, 200, 0, 2)
	toY := plantLoan(t, w, homeB, yonderA, 100, 0, 90)
	setBalances(t, w, map[AgentID]int64{homeB: 100, homeC: 200})
	pool, money := w.estates.pool(w.ledger, 0), moneyTotal(w)
	rec.events = nil
	if err := w.removeAgent(&w.agents[homeB], CauseConsumer); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	want := []Event{
		LoanEnded{Tick: 1, Claim: ClaimID(debt), Origin: OriginLoan, Creditor: homeWeaver, Debtor: homeB, Principal: 300,
			PaidPrincipal: 100, LostPrincipal: 200, Cause: CauseDebtorRemoved},
		LoanPassed{Tick: 1, Claim: ClaimID(toC), Creditor: homeB, Debtor: homeC, Region: 0, Owed: 200},
		LoanPassed{Tick: 1, Claim: ClaimID(toY), Creditor: homeB, Debtor: yonderA, Region: 0, Owed: 100},
		AgentBankrupted{Tick: 1, Agent: homeB, Region: 0, Role: 0, Weight: 1, Cause: CauseConsumer},
	}
	if got := removalEvents(rec); !reflect.DeepEqual(got, want) {
		t.Errorf("events of the removal = %+v, want %+v", got, want)
	}
	for _, id := range []int64{toC, toY} {
		if c, _ := w.ledger.Claim(id); c.Creditor != w.estates.account(0) || c.PrincipalOwed != c.Principal {
			t.Errorf("claim %d after the passing = %+v, want the home estate as creditor and the owed unchanged", id, c)
		}
	}
	if got := w.estates.pool(w.ledger, 0) - pool; got != 0 {
		t.Errorf("the pool received %d with the removal, want 0: the passing moves no money and the balance went to the creditor", got)
	}
	if got := moneyTotal(w); got != money {
		t.Errorf("money over the accounts = %d after the removal, want %d", got, money)
	}
	auditNow(t, w)

	step(t, w, 1) // tick 2: claim toC falls due; the estate is its creditor
	setBalances(t, w, map[AgentID]int64{homeC: 200})
	rec.events = nil
	pool = w.estates.pool(w.ledger, 0)
	w.p17aSettle()
	if got := w.estates.pool(w.ledger, 0) - pool; got != 200 {
		t.Errorf("the pool received %d at P1.7a, want the 200 homeC repaid", got)
	}
	wantEnd := LoanEnded{Tick: 2, Claim: ClaimID(toC), Origin: OriginLoan, Creditor: 0, Debtor: homeC, Principal: 200, ToEstate: true, Cause: CauseRepaid}
	wantOne(t, rec, wantEnd)
	wantOne(t, rec, CreditSettled{Tick: 2, Region: 0, Due: 1, Repaid: 1, PrincipalPaid: 200, ReceivedByEstate: 200})
}

// TestCredit_NoRegionBound verifies that a debtor and a creditor with open claims are
// not refused a relocation, so the module registers no region_bound obligation
// (spec-credit.md 9.4.2, N-090), and that dropClaims is called from one place only,
// the removal path of fanin.go, so a reduction of weight, a role change and a
// relocation run no step of the module (FR-CRD-043, FR-CRD-044).
//
// Requirement: FR-CRD-043, FR-CRD-044
//
// Guard: a second caller of dropClaims would run the waterfall at a change that
// leaves the agent in the world, and a region_bound row would stop a debtor from
// moving.
func TestCredit_NoRegionBound(t *testing.T) {
	t.Parallel()
	w, _ := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 100, homeA: 100})
	plantLoan(t, w, homeWeaver, homeA, 100, 0, 90)
	plantLoan(t, w, homeA, homeB, 100, 0, 90)
	for _, id := range []AgentID{homeA, homeWeaver} {
		if reason, err := w.RelocationRefusal(id, 1); reason != "" || err != nil {
			t.Errorf("RelocationRefusal(%d, 1) = %q, %v, want none: an open claim is no obligation", id, reason, err)
		}
	}
	if got := callersOf(t, "dropClaims"); !reflect.DeepEqual(got, []string{"fanin.go"}) {
		t.Errorf("dropClaims is called from %v, want from fanin.go alone", got)
	}
}

// callersOf returns the files of the package, tests left out, that call a method or
// a function of the given name, once for each call.
func callersOf(t *testing.T, name string) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob error = %v, want nil", err)
	}
	var callers []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
					callers = append(callers, file)
				}
			}
			return true
		})
	}
	return callers
}

// TestCredit_Audit_IndexesAreTheList verifies the audit of the module (spec-credit.md
// 3.6, FR-CRD-082): it passes over planted claims, fails naming the claim when an
// index no longer lists it, and changes no state (the bytes of the checkpoint are
// the same after it).
//
// Requirement: FR-CRD-082, FR-CRD-084, FR-CRD-085, FR-CRD-086
func TestCredit_Audit_IndexesAreTheList(t *testing.T) {
	t.Parallel()
	w, _ := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 200})
	plantLoan(t, w, homeWeaver, homeA, 100, 10, 90)
	plantPayout(t, w, homeB, homeA, 50, 90)
	auditNow(t, w)
	before, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	w.credit.idx.byDebtor[w.agents[homeA].account] = []int64{1}
	err = w.Audit()
	if !errors.Is(err, ErrAuditFailed) || !strings.Contains(err.Error(), "claim 0") {
		t.Errorf("Audit with claim 0 missing from its debtor's index = %v, want ErrAuditFailed naming claim 0", err)
	}
	w.credit.idx.byDebtor[w.agents[homeA].account] = []int64{0, 1}
	after, err := w.Checkpoint()
	if err != nil || !bytes.Equal(before.b, after.b) {
		t.Errorf("checkpoint after two audits = %v bytes, error %v, want the bytes of before: an audit changes no state", len(after.b), err)
	}
}
