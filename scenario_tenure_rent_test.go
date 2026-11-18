package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tenureRentTicks is the length of a run of TestScenarioTenure_BindingEndowmentWithTitles.
const tenureRentTicks = 40

// TestScenarioTenure_BindingEndowmentWithTitles runs the tenure module as a consumer
// would, through the exported API alone, on pack one with the binding resources overlay
// and the tenure overlay, seeds 1 to 2, with the audit on, and checks after every tick
// what a consumer can read: the rent query reports the posted rent, the holders query
// reports titles whose shares and the commons sum to 10 000 basis points, and the
// snapshot's endowment row of the titled land agrees with both and with the stock, which
// never passes its capacity. The audit of P8.4 holds money and goods every tick, so rent,
// which moves money between accounts, creates none. What the economy draws is reported,
// not asserted (N-352).
//
// Requirement: FR-LND-007, FR-LND-017, FR-LND-032, FR-LND-037
func TestScenarioTenure_BindingEndowmentWithTitles(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range seedList(invariantSeeds(2)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/resources-binding.toml", "overlay/tenure.toml")...)
			w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit())
			north, arable := regionNamed(t, cfg, "north"), arableOf(t, cfg)
			for tick := range tenureRentTicks {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d, tick %d: Step error = %v, want nil (the audit failed or the step did)", seed, tick, err)
				}
				checkTenureConsumerView(t, w, seed, tick, north, arable)
			}
		})
	}
}

// checkTenureConsumerView asserts what the consumer reads of the titled endowment after a
// tick: the queries and the snapshot row agree.
func checkTenureConsumerView(t *testing.T, w *economysim.World, seed uint64, tick int, region economysim.RegionID, resource economysim.ResourceID) {
	t.Helper()
	rent, err := w.Rent(region, resource)
	if err != nil || rent != 2 {
		t.Fatalf("seed %d, tick %d: Rent() = %d, %v, want 2, nil (the overlay's posted rent)", seed, tick, rent, err)
	}
	st, err := w.Holders(region, resource)
	if err != nil {
		t.Fatalf("seed %d, tick %d: Holders() error = %v, want nil", seed, tick, err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d, tick %d: Snapshot() error = %v, want nil", seed, tick, err)
	}
	rows := 0
	for _, row := range snap.Endowments {
		if row.Region == region && row.Resource == resource {
			rows++
			checkTenureRow(t, row, st, rent, fmt.Sprintf("seed %d, tick %d", seed, tick))
		}
	}
	if rows != 1 {
		t.Errorf("seed %d, tick %d: snapshot rows of the titled endowment = %d, want 1", seed, tick, rows)
	}
}

// checkTenureRow asserts that the snapshot row of the titled endowment agrees with the
// holders query and the rent query.
func checkTenureRow(t *testing.T, row economysim.EndowmentRow, st economysim.HoldingsState, rent int64, at string) {
	t.Helper()
	var held int64
	for _, x := range st.Titles {
		held += x.ShareBP
	}
	if row.Holders != int64(len(st.Titles)) || row.Posted != rent {
		t.Errorf("%s: row holders %d, posted %d, want %d and %d", at, row.Holders, row.Posted, len(st.Titles), rent)
	}
	if held > 10_000 || row.CommonsBP != 10_000-held {
		t.Errorf("%s: shares held %d, commons %d, want the commons to be the rest of 10 000", at, held, row.CommonsBP)
	}
	if row.Stock < 0 || row.Stock > row.Capacity {
		t.Errorf("%s: stock %d outside [0, capacity %d]", at, row.Stock, row.Capacity)
	}
}

// arableOf is the ID of the resource "arable" of the configuration.
func arableOf(t testing.TB, cfg economysim.Config) economysim.ResourceID {
	t.Helper()
	for i, r := range cfg.Modules.Resources.Resources {
		if r.ID == "arable" {
			return economysim.ResourceID(i)
		}
	}
	t.Fatalf("no resource arable in the configuration")
	return 0
}
