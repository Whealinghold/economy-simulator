package economysim

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The guards of the institutions module: what it must not touch (spec-institutions.md 7.3.4,
// 11.4.2, FR-ORG-017, FR-ORG-029, FR-ORG-052) and the order of its events.

// orgWithoutOutputs returns the snapshot without the outputs that belong to the module: the
// hash, the list of modules, the names of its entities and its rows.
func orgWithoutOutputs(s Snapshot) Snapshot {
	s.ConfigHash, s.Modules = [32]byte{}, nil
	s.Names.Bodies, s.Names.Rights, s.Names.Controls = nil, nil, nil
	s.Bodies, s.LegalLevels, s.Suspended = nil, nil, nil
	return s
}

// TestInstitutions_LeaveOtherResultsUnchanged verifies that the module moves nothing else:
// after 10 ticks the world with the module on and the world without it have the same snapshot
// apart from the module's own outputs, so no money, good or account is created or moved by it
// (it has no ledger reason) and a world created names only what the module declares.
//
// Requirement: FR-ORG-029, FR-ORG-052, NFR-ORG-003
//
// Guard: a draw, a ledger move or a changed order of agents in the round would move the
// economy before any reader of the module exists; this fails first.
func TestInstitutions_LeaveOtherResultsUnchanged(t *testing.T) {
	t.Parallel()
	offCfg := orgToy()
	offCfg.Modules.Institutions = nil
	on, off := newMill(t, orgToy(), WithSeed(5)), newMill(t, offCfg, WithSeed(5))
	step(t, on, 10)
	step(t, off, 10)
	a, errA := on.Snapshot()
	b, errB := off.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot() errors = %v, %v, want nil", errA, errB)
	}
	if got, want := orgWithoutOutputs(a), orgWithoutOutputs(b); !reflect.DeepEqual(got, want) {
		t.Errorf("the snapshot with the module on differs from the one without it after 10 ticks, apart from its own outputs")
	}
	if len(a.Bodies) != 3 {
		t.Errorf("Snapshot.Bodies = %v, want one row for each of the three (region, body) pairs", a.Bodies)
	}
}

// TestInstitutions_TakesNoDraw verifies by a text search that the files of the module take
// no random number: no call on the world's source, no import of a random package and no
// clock (spec-institutions.md 4.3.2, 7.3.4; FR-ORG-017).
//
// Requirement: FR-ORG-017
//
// Guard: a draw in the round would move the stream of every later site only if it used the
// world's keyed source; the text search catches the first use of any source at all.
func TestInstitutions_TakesNoDraw(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("institutions*.go")
	if err != nil {
		t.Fatalf("Glob() = %v, want nil", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) = %v, want nil", f, err)
		}
		for n, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, bad := range []string{"rnd.", "math/rand", "crypto/rand", "time.Now", "keyed("} {
				if strings.Contains(line, bad) {
					t.Errorf("%s:%d uses %q, want the module to take no draw and read no clock", f, n+1, bad)
				}
			}
		}
	}
}

// TestEvents_P72bAfterSettlement verifies the order of the events of a tick: the
// SettlementClassChanged of P7.2a comes before every MembershipChanged of P7.2b, and both
// before the end of the step.
//
// Arithmetic: region c has 5 households under the town's 10 enter; six smiths added make 11,
// so the review of tick 0 changes its class from hamlet to town before the round reports.
//
// Requirement: FR-ORG-043
func TestEvents_P72bAfterSettlement(t *testing.T) {
	t.Parallel()
	c := orgSettlementToy()
	c.Modules.Settlements.Dwell = 0 // the change applies at the first review, with no hold
	w, rec := orgWorld(t, c)
	for range 6 {
		if _, err := w.AddAgent(NewAgent{Region: orgC, Role: orgSmith, Money: 100}); err != nil {
			t.Fatalf("AddAgent() error = %v, want nil", err)
		}
	}
	rec.events = nil
	step(t, w, 1)
	changed, firstMember, ended := -1, -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case SettlementClassChanged:
			changed = i
		case MembershipChanged:
			if firstMember < 0 {
				firstMember = i
			}
		case TickEnded:
			ended = i
		}
	}
	if !(changed >= 0 && changed < firstMember && firstMember < ended) {
		t.Errorf("positions: SettlementClassChanged %d, first MembershipChanged %d, TickEnded %d, want them in that order", changed, firstMember, ended)
	}
}
