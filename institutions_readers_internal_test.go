package economysim

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// rightReaders are the calls through which a right is read and the files that may make
// them: the module's own file, and the hook sites of spec-institutions.md 7.3.2, each one
// guarded call (hooks H-4, H-5, H-7 and H-12 of plan 3-org). The training right is read when
// a place opens, by canTrain and by the entry step's count of the places it closes.
var rightReaders = map[string][]string{
	"rightIn(":         {"institutions_rights.go"},
	"barredAt(":        {"institutions_rights.go"},
	"startBarred(":     {"institutions_rights.go"},
	"refuseStart(":     {"institutions_rights.go", "phase_production.go", "skills.go"},
	"orderBarred(":     {"institutions_rights.go", "phase_trade.go"},
	"refusedOrder(":    {"institutions_rights.go", "phase_trade.go"},
	"contractRefusal(": {"institutions_rights.go", "contract.go"},
	"NotPermitted =":   {"phase_trade.go"},
	"mayTrain(":        {"institutions_rights.go", "skills_queue.go"},
	"noteTrain(":       {"institutions_rights.go", "skills_queue.go"},
}

// TestRightReaders_StaticCheck verifies that a right is read only at a start (P2.5 and a
// trainee's request), an order (P3.2), an offer or a take of the contract round (P3.3) and a
// place for a trainee (P7.1(a)), by the calls of the hook list and nowhere else: not at a continuation, a completion, a
// holding or a shipment (spec-institutions.md 5.5.1, FR-ORG-018).
//
// Requirement: FR-ORG-018, FR-ORG-054
//
// Guard: a right read at P2.1, at a completion or in the transport files would refuse work
// that a household began legally; the text search catches the first read outside the sites.
func TestRightReaders_StaticCheck(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
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
			for call, allowed := range rightReaders {
				if strings.Contains(line, call) && !slices.Contains(allowed, f) {
					t.Errorf("%s:%d reads a right with %q, want only %v to (spec-institutions.md 5.5.1)", f, n+1, call, allowed)
				}
			}
		}
	}
}
