package ledger

import (
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// healthy builds the ledger the identity tests break one rule of: funded
// accounts with goods of all four shapes, a loan claim and a payout claim, one
// tick ended and one transfer in the current tick, so the baseline and the
// totals are both non-zero.
func healthy(tb testing.TB) (*rig, map[string]int) {
	tb.Helper()
	g := newRig(tb, scriptMaxMoney)
	g.fund(tb)
	l := g.l
	if err := l.Move(g.bnd[Initial][2], g.holders[2], 2, 3, Initial); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	if _, err := l.Lend(g.holders[0], g.holders[1], 50, 5, 500, 3); err != nil {
		tb.Fatalf("setup: Lend: %v", err)
	}
	if _, err := l.Obligate(g.holders[4], g.holders[1], 10, 4); err != nil {
		tb.Fatalf("setup: Obligate: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	if err := l.Transfer(g.holders[0], g.holders[1], 7, Trade); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	l.EndTick()
	if err := l.Transfer(g.holders[1], g.holders[0], 2, Trade); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	if err := l.Move(g.holders[2], g.bnd[Disposal][2], 2, 1, Disposal); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	if err := l.Audit(); err != nil {
		tb.Fatalf("setup: the base ledger fails its audit: %v", err)
	}
	return g, map[string]int{"a0": g.holders[0], "a1": g.holders[1], "a2": g.holders[2], "e0": g.holders[4], "mint": g.mint}
}

// anyThing in a want matches any account or commodity.
const anyThing = -99

// TestAudit_OneCasePerIdentity verifies that each identity of the audit is
// reported, with the account, commodity or claim it names, by a ledger that a
// test breaks directly in exactly that one way (the only way to reach an
// inconsistent state), and that the healthy ledger passes.
//
// Requirement: FR-LED-035, FR-LED-024, FR-LED-046, FR-LED-064, FR-LED-036
func TestAudit_OneCasePerIdentity(t *testing.T) {
	t.Parallel()
	slot := func(l *Ledger, r Reason, c int) int {
		i, err := l.slotOf(r, c)
		if err != nil {
			t.Fatalf("slotOf(%d, %d): %v", r, c, err)
		}
		return i
	}
	type want struct {
		identity  Identity
		account   int
		commodity int
		claim     int64
	}
	tests := []struct {
		name string
		// mut breaks the ledger; ids are the named accounts.
		mut  func(g *rig, ids map[string]int)
		want func(ids map[string]int, g *rig) want
	}{
		{
			name: "region outside the ledger",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a0"]].region = 5 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["a0"], None, -1} },
		},
		{
			name: "closed account holds money",
			mut: func(g *rig, ids map[string]int) {
				id, _ := g.l.Open(KindEstate, 0, 0)
				_ = g.l.Close(id)
				g.l.accounts[id].balance = 1
				ids["x"] = id
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["x"], None, -1} },
		},
		{
			name: "boundary table points elsewhere",
			mut:  func(g *rig, ids map[string]int) { g.l.boundary[slot(g.l, Initial, Money)] = ids["a0"] },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["mint"], Money, -1} },
		},
		{
			name: "boundary holds a reservation",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["mint"]].reserved = 1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["mint"], Money, -1} },
		},
		{
			name: "balance below zero",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a0"]].balance = -1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityNonNegative, ids["a0"], Money, -1} },
		},
		{
			name: "reserved above the balance",
			mut: func(g *rig, ids map[string]int) {
				a := &g.l.accounts[ids["a1"]]
				a.reserved = a.balance + 1
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityNonNegative, ids["a1"], Money, -1} },
		},
		{
			name: "balance above MaxMoney",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["e0"]].balance = scriptMaxMoney + 1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["e0"], Money, -1} },
		},
		{
			name: "lots out of order",
			mut: func(g *rig, ids map[string]int) {
				lots := g.l.accounts[ids["a1"]].lots
				lots[0], lots[1] = lots[1], lots[0]
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], anyThing, -1} },
		},
		{
			name: "lot of quantity zero",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].lots[0].Quantity = 0 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], 0, -1} },
		},
		{
			name: "equal keys not merged",
			mut: func(g *rig, ids map[string]int) {
				a := &g.l.accounts[ids["a1"]]
				a.lots = append(a.lots[:1], a.lots...)
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], anyThing, -1} },
		},
		{
			name: "earmark above the lot",
			mut: func(g *rig, ids map[string]int) {
				lt := &g.l.accounts[ids["a1"]].lots[0]
				lt.Reserved = lt.Quantity + 1
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityNonNegative, ids["a1"], 0, -1} },
		},
		{
			name: "negative quantity",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].lots[0].Quantity = -1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityNonNegative, ids["a1"], 0, -1} },
		},
		{
			name: "born in the future",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].lots[0].Born = g.l.tick + 1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], 0, -1} },
		},
		{
			name: "born older than a stated lot can be",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].lots[0].Born = -3 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], 0, -1} },
		},
		{
			name: "born on a commodity that does not spoil",
			mut: func(g *rig, ids map[string]int) {
				a := &g.l.accounts[ids["a2"]]
				for i := range a.lots {
					if a.lots[i].Commodity == 2 {
						a.lots[i].Born = 1
					}
				}
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a2"], 2, -1} },
		},
		{
			name: "durability above full",
			mut: func(g *rig, ids map[string]int) {
				a := &g.l.accounts[ids["a1"]]
				for i := range a.lots {
					if a.lots[i].Commodity == 1 {
						a.lots[i].Durability = 99
					}
				}
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityCanonical, ids["a1"], 1, -1} },
		},
		{
			name: "quantity above MaxQuantity",
			mut: func(g *rig, ids map[string]int) {
				a := &g.l.accounts[ids["a1"]]
				for i := range a.lots {
					if a.lots[i].Commodity == 2 {
						a.lots[i].Quantity = scriptMaxQty + 1
					}
				}
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, ids["a1"], 2, -1} },
		},
		{
			name: "no hint for a spoiling lot",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].hasHint = false },
			want: func(ids map[string]int, g *rig) want { return want{IdentityHint, ids["a1"], None, -1} },
		},
		{
			name: "hint later than the expiry",
			mut: func(g *rig, ids map[string]int) {
				g.l.accounts[ids["a1"]].hint = 1000
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityHint, ids["a1"], None, -1} },
		},
		{
			name: "negative flow total",
			mut:  func(g *rig, ids map[string]int) { g.l.totals[0] = -1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, None, None, -1} },
		},
		{
			name: "negative claim total",
			mut:  func(g *rig, ids map[string]int) { g.l.claimTot.Repaid = -1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityStructure, None, None, -1} },
		},
		{
			name: "claim IDs not ascending",
			mut:  func(g *rig, ids map[string]int) { g.l.claims[1].ID = 0 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, None, None, 0} },
		},
		{
			name: "more claims than MaxClaims",
			mut:  func(g *rig, ids map[string]int) { g.l.maxClaims = 1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, None, None, -1} },
		},
		{
			name: "owed above principal",
			mut:  func(g *rig, ids map[string]int) { g.l.claims[0].PrincipalOwed = g.l.claims[0].Principal + 1 },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, None, None, 0} },
		},
		{
			name: "open claim owes nothing",
			mut: func(g *rig, ids map[string]int) {
				g.l.claims[1].PrincipalOwed = 0
				g.l.claims[1].InterestOwed = 0
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, None, None, 1} },
		},
		{
			name: "payout claim with interest",
			mut: func(g *rig, ids map[string]int) {
				g.l.claims[1].Interest = 1
				g.l.claims[1].InterestOwed = 1
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, None, None, 1} },
		},
		{
			name: "creditor and debtor are one account",
			mut:  func(g *rig, ids map[string]int) { g.l.claims[0].Creditor = g.l.claims[0].Debtor },
			want: func(ids map[string]int, g *rig) want {
				return want{IdentityClaim, g.l.claims[0].Debtor, None, 0}
			},
		},
		{
			name: "debtor is not an agent",
			mut:  func(g *rig, ids map[string]int) { g.l.claims[0].Debtor = ids["e0"] },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, ids["e0"], None, 0} },
		},
		{
			name: "claim outlives its debtor",
			mut: func(g *rig, ids map[string]int) {
				id, _ := g.l.Open(KindAgent, 0, 0)
				_ = g.l.Close(id)
				g.l.claims[0].Debtor = id
				ids["x"] = id
			},
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaim, ids["x"], None, 0} },
		},
		{
			name: "money does not sum to zero",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a2"]].balance++ },
			want: func(ids map[string]int, g *rig) want { return want{IdentityZeroTotal, None, Money, -1} },
		},
		{
			name: "a commodity does not sum to zero",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a1"]].lots[0].Quantity++ },
			want: func(ids map[string]int, g *rig) want { return want{IdentityZeroTotal, None, 0, -1} },
		},
		{
			name: "money flow total moved without the boundary",
			mut:  func(g *rig, ids map[string]int) { g.l.totals[slot(g.l, Initial, Money)]++ },
			want: func(ids map[string]int, g *rig) want { return want{IdentityConservation, ids["mint"], Money, -1} },
		},
		{
			name: "goods flow total moved without the boundary",
			mut:  func(g *rig, ids map[string]int) { g.l.totals[slot(g.l, Disposal, 2)]++ },
			want: func(ids map[string]int, g *rig) want {
				return want{IdentityConservation, g.bnd[Disposal][2], 2, -1}
			},
		},
		{
			name: "baseline of a boundary disagrees",
			mut:  func(g *rig, ids map[string]int) { g.l.base[slot(g.l, Initial, 1)]++ },
			want: func(ids map[string]int, g *rig) want {
				return want{IdentityConservation, g.bnd[Initial][1], 1, -1}
			},
		},
		{
			name: "outside sector holds money",
			mut:  func(g *rig, ids map[string]int) { g.l.accounts[ids["a2"]].kind = KindOutside },
			want: func(ids map[string]int, g *rig) want { return want{IdentitySectoral, None, anyThing, -1} },
		},
		{
			name: "principal owed is not Lent + Awarded - Repaid - LostPrincipal",
			mut:  func(g *rig, ids map[string]int) { g.l.claimTot.Lent++ },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaimTotals, None, Money, -1} },
		},
		{
			name: "interest owed, paid and lost is not InterestFixed",
			mut:  func(g *rig, ids map[string]int) { g.l.claimTot.InterestFixed++ },
			want: func(ids map[string]int, g *rig) want { return want{IdentityClaimTotals, None, Money, -1} },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, ids := healthy(t)
			tt.mut(g, ids)
			before := fingerprint(g.l)
			err := g.l.Audit()
			var ae *AuditError
			if !errors.As(err, &ae) || !errors.Is(err, ErrAuditFailed) {
				t.Fatalf("Audit() = %v, want an *AuditError matching ErrAuditFailed", err)
			}
			w := tt.want(ids, g)
			if ae.Identity != w.identity || ae.Account != w.account || ae.Claim != w.claim ||
				(w.commodity != anyThing && ae.Commodity != w.commodity) {
				t.Errorf("Audit() = {%v account %d commodity %d claim %d: %s}, want {%v account %d commodity %d claim %d}",
					ae.Identity, ae.Account, ae.Commodity, ae.Claim, ae.Detail, w.identity, w.account, w.commodity, w.claim)
			}
			if !reflect.DeepEqual(before, fingerprint(g.l)) {
				t.Errorf("Audit changed the ledger")
			}
		})
	}
}

// TestAudit_FlowTotalWithoutBoundaryAccount verifies that a flow total under
// a reason whose boundary account was never opened is a conservation failure
// that names no account.
//
// Requirement: FR-LED-035
func TestAudit_FlowTotalWithoutBoundaryAccount(t *testing.T) {
	t.Parallel()
	l, err := New(Config{Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 10, MaxQuantity: 10})
	if err != nil {
		t.Fatal(err)
	}
	l.totals[l.off[Disposal][thingGoods]] = 1
	var ae *AuditError
	if err := l.Audit(); !errors.As(err, &ae) || ae.Identity != IdentityConservation || ae.Account != None {
		t.Errorf("Audit() = %v, want conservation naming no account", err)
	}
}

// TestAudit_ConstructionBucket verifies that the audit holds, with exact
// equality, in the construction phase and between EndConstruction and the
// first clear, where the bucket is part of what the boundaries moved, and that
// a tampered bucket is a conservation failure.
//
// Requirement: FR-LED-046
func TestAudit_ConstructionBucket(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	if err := l.Audit(); err != nil {
		t.Fatalf("Audit in construction = %v, want nil", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatal(err)
	}
	if err := l.Transfer(g.holders[0], g.holders[1], 4, Trade); err != nil {
		t.Fatal(err)
	}
	if err := l.Audit(); err != nil {
		t.Fatalf("Audit after EndConstruction = %v, want nil", err)
	}
	l.ClearTotals()
	if err := l.Audit(); err != nil {
		t.Fatalf("Audit after the first clear = %v, want nil", err)
	}
	// The bucket is part of what the baseline holds once cleared; a bucket
	// that is changed afterwards is not detected, and one changed before is.
	g2 := newRig(t, scriptMaxMoney)
	g2.fund(t)
	g2.l.bucket[g2.l.off[Initial][thingMoney]]++
	var ae *AuditError
	if err := g2.l.Audit(); !errors.As(err, &ae) || ae.Identity != IdentityConservation || ae.Account != g2.mint {
		t.Errorf("Audit with a tampered bucket = %v, want conservation naming the mint", err)
	}
}

// TestAuditError_Message verifies that the error names the identity, the
// account, the commodity and the claim, for the log of the caller.
func TestAuditError_Message(t *testing.T) {
	t.Parallel()
	e := &AuditError{Identity: IdentityConservation, Account: 3, Commodity: 2, Claim: 7, Detail: "x"}
	for _, s := range []string{"conservation by reason", "account 3", "commodity 2", "claim 7"} {
		if !strings.Contains(e.Error(), s) {
			t.Errorf("Error() = %q, want it to contain %q", e.Error(), s)
		}
	}
	m := &AuditError{Identity: IdentityZeroTotal, Account: None, Commodity: Money, Claim: -1}
	if got := m.Error(); !strings.Contains(got, "money") || strings.Contains(got, "account") || strings.Contains(got, "claim") {
		t.Errorf("Error() = %q, want money and no account or claim", got)
	}
	if got := Identity(200).String(); got != "identity 200" {
		t.Errorf("Identity(200).String() = %q", got)
	}
}

// TestHasReservation verifies that the read reports a reservation of money
// and an earmark of goods, and none once they are released.
//
// Requirement: FR-LED-020
func TestHasReservation(t *testing.T) {
	t.Parallel()
	g, ids := healthy(t)
	l := g.l
	if l.HasReservation() {
		t.Fatal("HasReservation() = true on a ledger with none")
	}
	if err := l.ReserveMoney(ids["a0"], 5); err != nil {
		t.Fatal(err)
	}
	if !l.HasReservation() {
		t.Error("HasReservation() = false with money reserved")
	}
	if err := l.ReleaseMoney(ids["a0"], 5); err != nil {
		t.Fatal(err)
	}
	if err := l.ReserveGoods(ids["a1"], 1, 2); err != nil {
		t.Fatal(err)
	}
	if !l.HasReservation() {
		t.Error("HasReservation() = false with goods earmarked")
	}
	if err := l.ReleaseGoods(ids["a1"], 1, 2); err != nil {
		t.Fatal(err)
	}
	if l.HasReservation() {
		t.Error("HasReservation() = true after both were released")
	}
}

// TestInvariantViolation_PanicsAsProgrammerError verifies that the two places
// where an operation finds, after its own checks, that an invariant is
// violated panic with a message that says it is a programmer error, and do
// not return an error.
//
// Requirement: FR-LED-032
func TestInvariantViolation_PanicsAsProgrammerError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(g *rig, ids map[string]int)
	}{
		{
			name: "a claim transfer refused after its checks",
			run: func(g *rig, ids map[string]int) {
				// A claim of an account with itself passes every check of
				// Settle and is refused by the transfer.
				g.l.claims[0].Creditor = g.l.claims[0].Debtor
				_ = g.l.Settle(g.l.claims[0].ID, 1, 0)
			},
		},
		{
			name: "wear finds fewer items than it counted",
			run: func(g *rig, ids map[string]int) {
				// A durable lot with durability 0 is counted as held and
				// skipped by the selection.
				a := &g.l.accounts[ids["a1"]]
				for i := range a.lots {
					if a.lots[i].Commodity == 1 {
						a.lots[i].Durability = 0
					}
				}
				_ = g.l.Wear(ids["a1"], 1, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, ids := healthy(t)
			defer func() {
				r := recover()
				msg, _ := r.(string)
				if !strings.Contains(msg, "programmer error") {
					t.Errorf("panic = %v, want a message that says programmer error", r)
				}
			}()
			tt.run(g, ids)
		})
	}
}

// TestInvariants_OverSeeds verifies, over a range of seeds and thousands of
// scripted operations each, that after every operation the audit passes (the
// zero totals, non-negativity, canonical form, the hint, conservation by
// reason, the sectoral identity and the claim identities), that every refused
// operation leaves the ledger equal to its copy, and that no reservation is
// held at a tick end once the script has released them. The failing seed is in
// the message.
//
// Requirement: FR-LED-001, FR-LED-002, FR-LED-003, FR-LED-007, FR-LED-011, FR-LED-024, FR-LED-025
func TestInvariants_OverSeeds(t *testing.T) {
	t.Parallel()
	const seeds, steps = 40, 3000
	ok, tried := map[string]int{}, map[string]int{}
	for seed := uint64(1); seed <= seeds; seed++ {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		s := newScripter(g, &randChooser{r: rand.New(rand.NewPCG(seed, 7)), left: steps})
		if err := s.run(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		for k, v := range s.ok {
			ok[k] += v
		}
		for k, v := range s.tried {
			tried[k] += v
		}
	}
	// Guard: a script that is only refused proves nothing. Every operation
	// is accepted at least once across the range.
	for name, n := range tried {
		if ok[name] == 0 {
			t.Errorf("operation %s was tried %d times and never accepted", name, n)
		}
	}
	for _, name := range []string{"Transfer", "Move", "Transform", "WearItems", "Lend", "Settle", "WriteOff", "Obligate", "EndTick", "ReserveGoods", "MoveLot", "MoveStated", "Close"} {
		if ok[name] == 0 {
			t.Errorf("operation %s was never accepted", name)
		}
	}
}

// FuzzOps decodes its input into scripted operations and asserts the
// properties of TestInvariants_OverSeeds: nothing panics, every refusal
// leaves the ledger unchanged, and the audit passes after every operation.
// The ledger parses no external input, so the fuzz test is not required by
// testing.md 7.4; it is here because atomicity is the property a hand-written
// table misses.
//
// Requirement: FR-LED-001, FR-LED-007, FR-LED-011
func FuzzOps(f *testing.F) {
	rng := rand.New(rand.NewPCG(11, 13))
	for _, n := range []int{0, 8, 64, 256, 1024} {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(rng.UintN(256))
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			data = data[:4096]
		}
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		if err := newScripter(g, &byteChooser{b: data}).run(); err != nil {
			t.Fatal(err)
		}
	})
}

// TestFuzzCorpus_SeedFilesExist verifies that the committed seed corpus of
// FuzzOps is present, so that go test runs it.
//
// Guard: the corpus is the permanent regression set of the fuzz test
// (testing.md 7.5); an empty directory would silently drop it.
func TestFuzzCorpus_SeedFilesExist(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob(filepath.Join("testdata", "fuzz", "FuzzOps", "*"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no corpus under testdata/fuzz/FuzzOps: %v", err)
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Error(err)
		}
	}
}

// TestAuditSectors_ChangeWithoutFlow verifies the sectoral identity on its own:
// money that leaves the four sectors for a boundary account without a flow
// total breaks it. The whole audit reports a different identity first (the
// by-reason one), so the check is called directly.
//
// Requirement: FR-LED-025, FR-LED-024
func TestAuditSectors_ChangeWithoutFlow(t *testing.T) {
	t.Parallel()
	g, ids := healthy(t)
	if err := g.l.auditSectors(); err != nil {
		t.Fatalf("healthy ledger: %v", err)
	}
	g.l.accounts[ids["a0"]].balance--
	g.l.accounts[g.mint].balance++
	var ae *AuditError
	err := g.l.auditSectors()
	if !errors.As(err, &ae) || ae.Identity != IdentitySectoral || ae.Commodity != Money {
		t.Fatalf("auditSectors = %v, want the sectoral identity for money", err)
	}
}
