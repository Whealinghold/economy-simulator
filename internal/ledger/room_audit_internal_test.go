package ledger

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// limitedAgentWithLots returns the ID of an agent account of the healthy ledger
// that holds lots, with a table of sizes of 1 and a room that holds them.
func limitedAgentWithLots(t *testing.T, g *rig, ids map[string]int) int {
	t.Helper()
	g.l.sizes = make([]Size, len(g.l.comms))
	for i := range g.l.sizes {
		g.l.sizes[i] = Size{Mass: 1, Bulk: 1}
	}
	for _, id := range ids {
		a := &g.l.accounts[id]
		if a.kind != KindAgent || len(a.lots) == 0 {
			continue
		}
		if err := g.l.SetRoom(id, true, math.MaxInt64, math.MaxInt64); err != nil {
			t.Fatalf("setup: SetRoom(%d): %v", id, err)
		}
		return id
	}
	t.Fatal("setup: no agent account holds lots")
	return -1
}

// TestAuditRoom_LotsThatOverflowAreNamedAsSuch verifies that the audit of a
// room says the lots weigh more than int64 holds, instead of printing a sum
// that was never computed, when the load of the lots does not fit.
//
// Fix: DP3-STO-2. The overflow of loadOfLots was reported as a mismatch of the
// load with (0, 0), which sent the reader looking for a lost lot.
//
// Requirement: FR-STO-040
func TestAuditRoom_LotsThatOverflowAreNamedAsSuch(t *testing.T) {
	t.Parallel()
	g, ids := healthy(t)
	id := limitedAgentWithLots(t, g, ids)
	a := &g.l.accounts[id]
	a.lots[0].Quantity = 2
	g.l.sizes[a.lots[0].Commodity] = Size{Mass: math.MaxInt64, Bulk: 1}
	err := g.l.Audit()
	var ae *AuditError
	if !errors.As(err, &ae) || ae.Identity != IdentityRoom || ae.Account != id {
		t.Fatalf("Audit() = %v, want the room identity of account %d", err, id)
	}
	if !strings.Contains(ae.Detail, "more than int64") {
		t.Errorf("Detail = %q, want it to name the overflow of the lots", ae.Detail)
	}
}

// TestAudit_ClosedAccountHoldsNoRoom verifies that a closed account that still
// has a room is a structure failure: a closed account is wholly zeroed.
//
// Fix: DP3-STO-2. closedHoldsState did not read the room, so a closed account
// with a room passed the audit.
//
// Requirement: FR-STO-040
func TestAudit_ClosedAccountHoldsNoRoom(t *testing.T) {
	t.Parallel()
	g, _ := healthy(t)
	id, err := g.l.Open(KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("setup: Open: %v", err)
	}
	if err := g.l.SetRoom(id, true, 5, 5); err != nil {
		t.Fatalf("setup: SetRoom: %v", err)
	}
	if err := g.l.Close(id); err != nil {
		t.Fatalf("setup: Close: %v", err)
	}
	if err := g.l.Audit(); err != nil {
		t.Fatalf("Audit() after Close = %v, want nil", err)
	}
	g.l.accounts[id].room = &Room{Limited: true, Mass: 5, Bulk: 5}
	var ae *AuditError
	if err := g.l.Audit(); !errors.As(err, &ae) || ae.Identity != IdentityStructure || ae.Account != id {
		t.Errorf("Audit() = %v, want the structure identity of account %d", err, id)
	}
}
