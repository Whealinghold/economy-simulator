package economysim

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// plantedWorld is a credit world at tick 4 with the three kinds of claim a section
// must keep: a loan that went overdue (due at 2, kept unpaid by the shield), a loan
// not yet due and a payout claim, a rate scale that is not the default, and a run
// of each Warn.
func plantedWorld(t *testing.T) *World {
	t.Helper()
	w := creditWorld(t, 1_000, 0, WithAudit())
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 600})
	plantLoan(t, w, homeWeaver, homeA, 500, 50, 2)
	plantLoan(t, w, homeWeaver, yonderA, 100, 0, 30)
	plantPayout(t, w, homeB, homeC, 70, 40)
	for range 3 {
		stepSet(t, w, map[AgentID]int64{homeA: 5_000})
	}
	w.credit.scale[1] = 7_500
	w.credit.overdueRun[0] = 3
	w.credit.capRun = 2
	return w
}

// TestCheckpoint_Credit verifies tag 26 over a world with open claims, one of them
// overdue, a non-default rate scale and the runs of both Warns: the restore gives the
// bytes of the original, the claims, the totals, the scales, the runs and the
// derived indexes, and the two settle the same claims in the ticks that follow; a
// world with the module off has no such section; and a restore with another credit
// configuration is ErrCheckpointConfig.
//
// Requirement: FR-CRD-087, FR-CRD-088
func TestCheckpoint_Credit(t *testing.T) {
	t.Parallel()
	w := plantedWorld(t)
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	restored, err := RestoreWorld(w.Config(), cp, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if again, _ := restored.Checkpoint(); !bytes.Equal(again.b, cp.b) {
		t.Errorf("checkpoint of the restored world has %d bytes and differs from the original's %d", len(again.b), len(cp.b))
	}
	wantSameCredit(t, restored, w)
	for range 3 {
		stepSet(t, w, map[AgentID]int64{homeA: 5_000})
		stepSet(t, restored, map[AgentID]int64{homeA: 5_000})
	}
	a, _ := w.Checkpoint()
	b, _ := restored.Checkpoint()
	if !bytes.Equal(a.b, b.b) {
		t.Error("checkpoints of the original and the restored world differ after three more ticks, want the same bytes")
	}

	other := w.Config()
	other.Modules.Credit.ShieldTicks++
	if _, err := RestoreWorld(other, cp); !errors.Is(err, ErrCheckpointConfig) {
		t.Errorf("RestoreWorld with another shield = %v, want ErrCheckpointConfig", err)
	}
}

// wantSameCredit fails the test unless the credit state of got is that of want: the
// scales, the runs, the claims and the totals, and indexes that are the claim list.
func wantSameCredit(t *testing.T, got, want *World) {
	t.Helper()
	g, w := got.credit, want.credit
	if !slices.Equal(g.scale, w.scale) || !slices.Equal(g.overdueRun, w.overdueRun) || g.capRun != w.capRun {
		t.Errorf("restored scales %v, runs %v and cap run %d, want %v, %v and %d", g.scale, g.overdueRun, g.capRun, w.scale, w.overdueRun, w.capRun)
	}
	claims := got.ledger.AppendClaims(nil)
	if !slices.Equal(claims, want.ledger.AppendClaims(nil)) || got.ledger.ClaimTotals() != want.ledger.ClaimTotals() {
		t.Errorf("restored claims %+v and totals %+v, want the original's", claims, got.ledger.ClaimTotals())
	}
	if err := g.idx.check(claims); err != nil {
		t.Errorf("indexes of the restored world: %v, want them to equal the claim list", err)
	}
}

// TestCheckpoint_CreditSectionIsAbsentWithTheModuleOff verifies that a world with the
// module off writes no section 26, so its bytes are those of format 1 (N-318).
//
// Requirement: FR-CRD-001, FR-CRD-088
func TestCheckpoint_CreditSectionIsAbsentWithTheModuleOff(t *testing.T) {
	t.Parallel()
	var row *section
	for i := range checkpointSections {
		if checkpointSections[i].tag == 26 {
			row = &checkpointSections[i]
		}
	}
	if row == nil || row.present == nil {
		t.Fatalf("no row for tag 26 with a condition: %+v", row)
	}
	off, on := creditConfig(0, 0), creditConfig(0, 0)
	off.Modules.Credit = nil
	if row.present(&off) || !row.present(&on) {
		t.Errorf("tag 26 present with the module off %v and on %v, want false and true", row.present(&off), row.present(&on))
	}
}

// TestLoadCredit_RefusesABadSection verifies that the section is refused, as a
// corrupt checkpoint, for a scale at its default, a scale out of order, a scale above
// MaxShockBP, a region that does not exist, a run of 0 or above the tick, and a run
// of the cap Warn above the tick.
//
// Requirement: FR-CRD-087
func TestLoadCredit_RefusesABadSection(t *testing.T) {
	t.Parallel()
	body := func(scales [][2]uint64, runs [][2]uint64, capRun uint64) []byte {
		w := codec.NewWriter(nil)
		w.Count(len(scales))
		for _, s := range scales {
			w.U(s[0])
			w.U(s[1])
		}
		w.Count(len(runs))
		for _, r := range runs {
			w.U(r[0])
			w.U(r[1])
		}
		w.U(capRun)
		return w.Out()
	}
	tests := []struct {
		name string
		body []byte
		ok   bool
	}{
		{"a good section", body([][2]uint64{{0, 0}, {1, 7_500}}, [][2]uint64{{1, 3}}, 4), true},
		{"a scale at its default", body([][2]uint64{{0, wholeBP}}, nil, 0), false},
		{"scales out of order", body([][2]uint64{{1, 7_500}, {0, 7_500}}, nil, 0), false},
		{"a scale above the bound", body([][2]uint64{{0, MaxShockBP + 1}}, nil, 0), false},
		{"a region that does not exist", body([][2]uint64{{2, 7_500}}, nil, 0), false},
		{"a run of 0", body(nil, [][2]uint64{{0, 0}}, 0), false},
		{"a run above the tick", body(nil, [][2]uint64{{0, 5}}, 0), false},
		{"a cap run above the tick", body(nil, nil, 5), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := creditWorld(t, 0, 0)
			step(t, w, 4)
			r := codec.NewReader(tt.body)
			err := w.loadCredit(&restoreCtx{tick: w.Tick()}, r)
			if err == nil {
				err = r.Done()
			}
			if (err == nil) != tt.ok {
				t.Errorf("loadCredit(%s) = %v, want ok %v", tt.name, err, tt.ok)
			}
		})
	}
}

// TestClaimTimesFault verifies the checks of spec-credit.md 17.5 that are the
// module's: the due tick after the formation, the formation not after the tick and an
// overdue tick of -1 or in [Due, tick), and that each failure names the claim.
//
// Requirement: FR-CRD-087
func TestClaimTimesFault(t *testing.T) {
	t.Parallel()
	good := ledger.Claim{ID: 4, Formed: 2, Due: 5, Overdue: -1}
	tests := []struct {
		name  string
		claim ledger.Claim
		tick  int64
		ok    bool
	}{
		{"an open claim not yet due", good, 3, true},
		{"overdue from the due tick", ledger.Claim{ID: 4, Formed: 2, Due: 5, Overdue: 5}, 6, true},
		{"due at its formation", ledger.Claim{ID: 4, Formed: 5, Due: 5, Overdue: -1}, 6, false},
		{"formed after the tick", good, 1, false},
		{"overdue before it was due", ledger.Claim{ID: 4, Formed: 2, Due: 5, Overdue: 4}, 6, false},
		{"overdue at the tick", ledger.Claim{ID: 4, Formed: 2, Due: 5, Overdue: 6}, 6, false},
		{"overdue below -1", ledger.Claim{ID: 4, Formed: 2, Due: 5, Overdue: -2}, 6, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := claimTimesFault(tt.claim, tt.tick)
			if (err == nil) != tt.ok {
				t.Errorf("claimTimesFault(%+v, %d) = %v, want ok %v", tt.claim, tt.tick, err, tt.ok)
			}
		})
	}
}

// TestCreditCross_RefusesAClaimOverdueAtTheTick verifies that the cross-check of a
// restore rebuilds the indexes of a good list and refuses, as ErrCheckpointCorrupt
// naming the claim, a list with an overdue tick that is not below the tick of the
// boundary.
//
// Arithmetic: at tick 2 the pass marks the unpaid claim overdue from 2, which is below
// the tick of the next boundary, 3, and not below 2: the same list is good at 3 and
// corrupt if the boundary is 2.
//
// Requirement: FR-CRD-087
func TestCreditCross_RefusesAClaimOverdueAtTheTick(t *testing.T) {
	t.Parallel()
	w, _ := settleAt(t, 1_000, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 100})
	id := plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	step(t, w, 1)
	w.p17aSettle()
	if c, _ := w.ledger.Claim(id); c.Overdue != 2 {
		t.Fatalf("setup: claim overdue from %d, want 2", c.Overdue)
	}
	err := w.creditCross()
	if !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("creditCross at tick 2 with a claim overdue from 2 = %v, want ErrCheckpointCorrupt", err)
	}
	w.ledger.AdvanceTick()
	if err := w.creditCross(); err != nil {
		t.Errorf("creditCross at tick 3 = %v, want nil", err)
	}
	if err := w.credit.idx.check(w.ledger.AppendClaims(nil)); err != nil {
		t.Errorf("indexes after the cross-check: %v, want them rebuilt", err)
	}
}
