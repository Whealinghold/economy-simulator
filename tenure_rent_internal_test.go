package economysim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRent_LedgerReasonRent_NoBoundaryFlow_SupplyUnchanged verifies that rent moves
// existing money between accounts: the total of the accounts is the same before and
// after the starts, and the tenant that paid is poorer by exactly what the pool gained
// (spec-land.md 5.1, 5.8).
//
// Requirement: FR-LND-017, FR-LND-045
func TestRent_LedgerReasonRent_NoBoundaryFlow_SupplyUnchanged(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	total, pool := tnTotalMoney(t, w), tnMoney(t, w, 0)
	toStart(t, w)
	paid := w.tenure.rt.counts[0].paid
	if paid == 0 {
		t.Fatalf("rent paid = 0, want a tenant that paid")
	}
	if got := tnTotalMoney(t, w); got != total {
		t.Errorf("total money after the rent = %d, want %d unchanged (no faucet, no sink)", got, total)
	}
	if got := tnMoney(t, w, 0) - pool; got != paid {
		t.Errorf("pool gained %d, want the rent paid %d", got, paid)
	}
}

// TestTenure_NoBranchOnIdentifier is a text guard over the tenure files of the root:
// none names a commodity, recipe, role, region or resource of a pack, and none takes a
// draw or reads the clock (spec-land.md FR-LND-036, FR-LND-042).
//
// Requirement: FR-LND-036, FR-LND-042
func TestTenure_NoBranchOnIdentifier(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("tenure*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("Glob(tenure*.go) = %v, %v, want the module files", files, err)
	}
	banned := []string{`== "`, `case "`, "rand.", "time.Now", "w.rng", ".Draw(", "crypto/"}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v, want nil", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			code, _, _ := strings.Cut(line, "//")
			code = strings.ReplaceAll(code, `!= ""`, "")
			for _, b := range banned {
				if strings.Contains(code, b) {
					t.Errorf("%s:%d contains %q: a tenure file must not branch on an identifier or draw", f, i+1, b)
				}
			}
		}
	}
}
