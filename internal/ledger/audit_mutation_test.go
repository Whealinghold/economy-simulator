package ledger

import (
	"errors"
	"strings"
	"testing"
)

// auditErr unwraps err to an *AuditError, or fails the test.
func auditErr(t *testing.T, err error) *AuditError {
	t.Helper()
	var ae *AuditError
	if !errors.As(err, &ae) {
		t.Fatalf("error = %v, want an *AuditError", err)
	}
	return ae
}

// TestIdentity_StringAtTheEndOfTheTable verifies that the first number past
// the named identities has the numbered form and the last named one its name.
//
// Guard: String indexes a table; the number equal to its length must not index it.
func TestIdentity_StringAtTheEndOfTheTable(t *testing.T) {
	t.Parallel()
	if got := IdentityClaimTotals.String(); got != "claim totals" {
		t.Errorf("IdentityClaimTotals.String() = %q, want %q", got, "claim totals")
	}
	n := Identity(len(identityNames))
	if got, want := n.String(), "identity 9"; got != want {
		t.Errorf("Identity(%d).String() = %q, want %q", n, got, want)
	}
}

// TestAuditError_NamesClaimZero verifies that claim 0, the first claim ID, is
// named in the message, as the claim of FR-LED-064.
//
// Requirement: FR-LED-064
func TestAuditError_NamesClaimZero(t *testing.T) {
	t.Parallel()
	e := &AuditError{Identity: IdentityClaim, Account: None, Commodity: None, Claim: 0, Detail: "x"}
	if got := e.Error(); !strings.Contains(got, "claim 0") {
		t.Errorf("Error() = %q, want it to name claim 0", got)
	}
}

// TestAuditAccount_Structure verifies the structural checks of one account
// slot, each by an account broken in exactly one way, and that the account at
// the edge of each bound passes.
//
// Requirement: FR-LED-035, FR-LED-024
func TestAuditAccount_Structure(t *testing.T) {
	t.Parallel()
	g, _ := healthy(t)
	l := g.l
	tests := []struct {
		name string
		a    account
		ok   bool
	}{
		{"kind one past the last kind", account{kind: numKinds}, false},
		{"kind far past the last kind", account{kind: numKinds + 1}, false},
		{"region equal to the region count", account{kind: KindAgent, region: l.regions}, false},
		{"last region", account{kind: KindAgent, region: l.regions - 1}, true},
		{"balance equal to MaxMoney", account{kind: KindAgent, balance: l.maxMoney}, true},
		{"balance above MaxMoney", account{kind: KindAgent, balance: l.maxMoney + 1}, false},
		{"closed and zeroed", account{kind: KindAgent, closed: true}, true},
		{"closed with positive balance", account{kind: KindAgent, closed: true, balance: 1}, false},
		{"closed with negative balance", account{kind: KindAgent, closed: true, balance: -1}, false},
		{"closed with positive reservation", account{kind: KindAgent, closed: true, reserved: 1}, false},
		{"closed with negative reservation", account{kind: KindAgent, closed: true, reserved: -1}, false},
		{"closed with positive owner", account{kind: KindAgent, closed: true, owner: 1}, false},
		{"closed with negative owner", account{kind: KindAgent, closed: true, owner: -1}, false},
		{"closed with positive region", account{kind: KindAgent, closed: true, region: 1}, false},
		{"closed with negative region", account{kind: KindAgent, closed: true, region: -1}, false},
		{"closed with a lot", account{kind: KindAgent, closed: true, lots: []Lot{{Commodity: 2, Quantity: 1}}}, false},
		{"closed with a hint", account{kind: KindAgent, closed: true, hasHint: true}, false},
		// Region 0 and nothing else set: only the kind says it is a boundary.
		{"closed boundary", account{kind: KindBoundary, closed: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := tt.a
			err := l.auditAccount(len(l.accounts), &a)
			if tt.ok {
				if err != nil {
					t.Fatalf("auditAccount = %v, want nil", err)
				}
				return
			}
			if ae := auditErr(t, err); ae.Identity != IdentityStructure {
				t.Errorf("auditAccount identity = %v, want %v", ae.Identity, IdentityStructure)
			}
		})
	}
}

// TestAuditBoundary verifies each way a boundary account can be wrong, and
// that a boundary account that is right passes.
//
// Requirement: FR-LED-035
func TestAuditBoundary(t *testing.T) {
	t.Parallel()
	const id = 7 // an account ID the tables are pointed at; the account is a copy
	money := account{kind: KindBoundary, region: -1, reason: Initial, thing: Money}
	tests := []struct {
		name   string
		a      account
		point  func(l *Ledger, g *rig) // sets the boundary table
		id     int
		detail string // "" means the account passes
	}{
		{
			name:  "right",
			a:     money,
			point: func(l *Ledger, _ *rig) { l.boundary[l.off[Initial][thingMoney]] = id },
			id:    id,
		},
		{
			// The table names an account with a lower ID than the one audited.
			name: "table names a lower account",
			a:    account{kind: KindBoundary, region: -1, reason: Disposal, thing: 2},
			point: func(l *Ledger, g *rig) {
				l.boundary[l.off[Disposal][thingGoods]+2] = g.mint
			},
			id:     id,
			detail: "not the account of its reason",
		},
		{
			// Trade is internal: it has a row and a slot but no boundary.
			name:   "internal reason",
			a:      account{kind: KindBoundary, region: -1, reason: Trade, thing: Money},
			point:  func(l *Ledger, _ *rig) { l.boundary[l.off[Trade][thingMoney]] = id },
			id:     id,
			detail: "internal reason",
		},
		{
			name: "region zero",
			a:    account{kind: KindBoundary, region: 0, reason: Initial, thing: Money},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
		{
			name: "region below minus one",
			a:    account{kind: KindBoundary, region: -2, reason: Initial, thing: Money},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
		{
			name: "positive reservation",
			a:    account{kind: KindBoundary, region: -1, reason: Initial, thing: Money, reserved: 1},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
		{
			name: "negative reservation",
			a:    account{kind: KindBoundary, region: -1, reason: Initial, thing: Money, reserved: -1},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
		{
			name: "a lot",
			a: account{kind: KindBoundary, region: -1, reason: Initial, thing: Money,
				lots: []Lot{{Commodity: 2, Quantity: 1}}},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
		{
			name: "a hint",
			a:    account{kind: KindBoundary, region: -1, reason: Initial, thing: Money, hasHint: true},
			point: func(l *Ledger, _ *rig) {
				l.boundary[l.off[Initial][thingMoney]] = id
			},
			id:     id,
			detail: "region",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, _ := healthy(t)
			tt.point(g.l, g)
			a := tt.a
			err := g.l.auditBoundary(tt.id, &a)
			if tt.detail == "" {
				if err != nil {
					t.Fatalf("auditBoundary = %v, want nil", err)
				}
				return
			}
			ae := auditErr(t, err)
			if ae.Identity != IdentityStructure || !strings.Contains(ae.Detail, tt.detail) {
				t.Errorf("auditBoundary = {%v: %s}, want a structure failure containing %q", ae.Identity, ae.Detail, tt.detail)
			}
		})
	}
}

// TestAuditLots_Bounds verifies the edges of the lot checks: a quantity
// exactly at MaxQuantity passes, a commodity one past the table is a
// canonical-form failure rather than a crash, and the failure for an order
// violation names the two lots.
//
// Requirement: FR-LED-035, FR-LED-024
func TestAuditLots_Bounds(t *testing.T) {
	t.Parallel()
	g, _ := healthy(t)
	l := g.l
	nc := len(l.comms)

	// Commodity 2 neither spoils nor wears, so Born and Durability are 0.
	at := account{kind: KindAgent, lots: []Lot{{Commodity: 2, Quantity: l.maxQty}}}
	if err := l.auditLots(0, &at); err != nil {
		t.Errorf("auditLots with the quantity equal to MaxQuantity = %v, want nil", err)
	}

	past := account{kind: KindAgent, lots: []Lot{{Commodity: nc, Quantity: 1}}}
	if ae := auditErr(t, l.auditLots(0, &past)); ae.Identity != IdentityCanonical {
		t.Errorf("auditLots with commodity %d identity = %v, want %v", nc, ae.Identity, IdentityCanonical)
	}

	// Two lots of equal key: the second is not after the first (indices 1, 0).
	dup := account{kind: KindAgent, lots: []Lot{{Commodity: 2, Quantity: 1}, {Commodity: 2, Quantity: 1}}}
	ae := auditErr(t, l.auditLots(0, &dup))
	if want := "lot 1 is not after lot 0"; ae.Identity != IdentityCanonical || !strings.Contains(ae.Detail, want) {
		t.Errorf("auditLots = {%v: %s}, want a canonical-form failure containing %q", ae.Identity, ae.Detail, want)
	}
}

// TestAuditCounters_TableLengths verifies that a table one entry too short or
// one too long is a structure failure, for each of the four tables.
//
// Requirement: FR-LED-035
func TestAuditCounters_TableLengths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(l *Ledger)
	}{
		{"boundary short", func(l *Ledger) { l.boundary = l.boundary[:len(l.boundary)-1] }},
		{"boundary long", func(l *Ledger) { l.boundary = append(l.boundary, -1) }},
		{"base short", func(l *Ledger) { l.base = l.base[:len(l.base)-1] }},
		{"base long", func(l *Ledger) { l.base = append(l.base, 0) }},
		// totals and bucket change together, so the totals term is the one that fails.
		{"totals and bucket short", func(l *Ledger) {
			l.totals, l.bucket = l.totals[:len(l.totals)-1], l.bucket[:len(l.bucket)-1]
		}},
		{"totals and bucket long", func(l *Ledger) {
			l.totals, l.bucket = append(l.totals, 0), append(l.bucket, 0)
		}},
		{"bucket short", func(l *Ledger) { l.bucket = l.bucket[:len(l.bucket)-1] }},
		{"bucket long", func(l *Ledger) { l.bucket = append(l.bucket, 0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, _ := healthy(t)
			if err := g.l.auditCounters(); err != nil {
				t.Fatalf("auditCounters on the healthy ledger = %v, want nil", err)
			}
			tt.mut(g.l)
			ae := auditErr(t, g.l.auditCounters())
			if ae.Identity != IdentityStructure || !strings.Contains(ae.Detail, "wrong length") {
				t.Errorf("auditCounters = {%v: %s}, want a structure failure about the length", ae.Identity, ae.Detail)
			}
		})
	}
}

// TestAuditClaims_Shape verifies the order of claim IDs (descending, not just
// equal, and at or above the next ID) and the range of principal and interest,
// each naming the failing claim.
//
// Requirement: FR-LED-064
func TestAuditClaims_Shape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mut    func(l *Ledger)
		claim  int64
		detail string
	}{
		{"descending IDs", func(l *Ledger) { l.claims[0].ID, l.claims[1].ID = 1, 0 }, 0, "ascending"},
		{"ID equal to the next ID", func(l *Ledger) { l.claims[1].ID = l.nextClaim }, 2, "ascending"},
		{"ID above the next ID", func(l *Ledger) { l.claims[1].ID = l.nextClaim + 3 }, 5, "ascending"},
		{"principal zero", func(l *Ledger) { l.claims[0].Principal = 0 }, 0, "not valid"},
		{"interest negative", func(l *Ledger) { l.claims[0].Interest = -1 }, 0, "not valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, _ := healthy(t)
			tt.mut(g.l)
			ae := auditErr(t, g.l.auditClaims())
			if ae.Identity != IdentityClaim || ae.Claim != tt.claim || !strings.Contains(ae.Detail, tt.detail) {
				t.Errorf("auditClaims = {%v claim %d: %s}, want a claim failure on claim %d containing %q",
					ae.Identity, ae.Claim, ae.Detail, tt.claim, tt.detail)
			}
		})
	}
}

// TestAudit_FreshLedgerPasses verifies that a ledger with no boundary account
// opened and no flow passes: a slot without a boundary account is a failure
// only when it has a flow total.
//
// Requirement: FR-LED-035
func TestAudit_FreshLedgerPasses(t *testing.T) {
	t.Parallel()
	l, err := New(Config{Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 10, MaxQuantity: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Audit(); err != nil {
		t.Errorf("Audit() on a fresh ledger = %v, want nil", err)
	}
}

// TestAuditSectors_LastCommodity verifies the sectoral identity for the last
// commodity and for money: what the outside sector holds is named with the
// commodity it holds, and a change of the four sectors in the last commodity
// alone is reported.
//
// Requirement: FR-LED-025, FR-LED-024
func TestAuditSectors_LastCommodity(t *testing.T) {
	t.Parallel()
	t.Run("outside holds the last commodity", func(t *testing.T) {
		t.Parallel()
		g, _ := healthy(t)
		nc := len(g.l.comms)
		id, err := g.l.Open(KindOutside, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		g.l.accounts[id].lots = []Lot{{Commodity: nc - 1, Quantity: 1}}
		ae := auditErr(t, g.l.auditSectors())
		if ae.Identity != IdentitySectoral || ae.Commodity != nc-1 || !strings.Contains(ae.Detail, "outside") {
			t.Errorf("auditSectors = {%v commodity %d: %s}, want the outside sector, commodity %d",
				ae.Identity, ae.Commodity, ae.Detail, nc-1)
		}
	})
	t.Run("outside holds money", func(t *testing.T) {
		t.Parallel()
		g, _ := healthy(t)
		id, err := g.l.Open(KindOutside, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		g.l.accounts[id].balance = 1
		ae := auditErr(t, g.l.auditSectors())
		if ae.Identity != IdentitySectoral || ae.Commodity != Money || !strings.Contains(ae.Detail, "outside") {
			t.Errorf("auditSectors = {%v commodity %d: %s}, want the outside sector, money", ae.Identity, ae.Commodity, ae.Detail)
		}
	})
	t.Run("four sectors change in the last commodity", func(t *testing.T) {
		t.Parallel()
		g, ids := healthy(t)
		nc := len(g.l.comms)
		found := false
		for i := range g.l.accounts[ids["a0"]].lots {
			if lt := &g.l.accounts[ids["a0"]].lots[i]; lt.Commodity == nc-1 {
				lt.Quantity++
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("setup: a0 holds no lot of commodity %d", nc-1)
		}
		ae := auditErr(t, g.l.auditSectors())
		if ae.Identity != IdentitySectoral || ae.Commodity != nc-1 || !strings.Contains(ae.Detail, "four sectors") {
			t.Errorf("auditSectors = {%v commodity %d: %s}, want the sector change, commodity %d",
				ae.Identity, ae.Commodity, ae.Detail, nc-1)
		}
	})
}
