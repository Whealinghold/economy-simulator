package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestTenureLedger_ReasonsDeclaredOnlyWithTheModule verifies that rent and title_sale
// are declared module rows of the ledger exactly when the tenure module is on, that each
// has a root name, and that neither opens a boundary account: both are internal.
//
// Requirement: FR-LND-017, FR-LND-045
func TestTenureLedger_ReasonsDeclaredOnlyWithTheModule(t *testing.T) {
	t.Parallel()
	if got := flatReasons(&Config{}); slices.Contains(got, ledger.Rent) || slices.Contains(got, ledger.TitleSale) {
		t.Errorf("moduleReasons with the module off = %v, want neither rent nor title_sale", got)
	}
	on := &Config{}
	on.Modules.Tenure = &TenureConfig{}
	got := flatReasons(on)
	for r, want := range map[ledger.Reason]Reason{ledger.Rent: ReasonRent, ledger.TitleSale: ReasonTitleSale} {
		if !slices.Contains(got, r) {
			t.Errorf("moduleReasons with the module on = %v, want reason %d", got, r)
		}
		if class, ok := ledger.ClassOf(r, false); !ok || class != ledger.Internal {
			t.Errorf("ClassOf(%d, money) = %v, %v, want an internal row", r, class, ok)
		}
		checkReasonName(t, r, want)
	}
	if string(ReasonRent) != "rent" || string(ReasonTitleSale) != "title_sale" {
		t.Errorf("reason names = %q, %q, want rent and title_sale", ReasonRent, ReasonTitleSale)
	}
}

// flatReasons is the declared module reasons of a configuration in one list.
func flatReasons(c *Config) []ledger.Reason {
	var all []ledger.Reason
	for _, m := range moduleReasons(c) {
		all = append(all, m...)
	}
	return all
}

// checkReasonName checks that the ledger reason r and the root reason want name each other.
func checkReasonName(t *testing.T, r ledger.Reason, want Reason) {
	t.Helper()
	if name, ok := reasonName(r); !ok || name != want {
		t.Errorf("reasonName(%d) = %q, %v, want %q", r, name, ok, want)
	}
	if back, ok := ledgerReason(want); !ok || back != r {
		t.Errorf("ledgerReason(%q) = %d, %v, want %d", want, back, ok, r)
	}
}
