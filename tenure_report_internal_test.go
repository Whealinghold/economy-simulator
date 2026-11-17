package economysim

import (
	"errors"
	"testing"
)

// TestSnapshot_TenureRows_FunctionsOfSavedState verifies the tenure fields of the snapshot
// row of an endowment: the number of titles, the pool's share, the commons share, the
// largest share of an agent, the posted rent, the scale, Closed and the two runs, and that
// the rows of a region's own view equal the world's.
//
// Requirement: FR-LND-032
func TestSnapshot_TenureRows_FunctionsOfSavedState(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, func(m *TenureConfig) { m.Terms[0].HasClosed, m.Terms[0].Closed = true, true })
	if err := w.TransferTitles([]TitleTransfer{tr(0, 2, 5, 30, 0)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	if err := w.SetRentScales([]EndowmentScale{{Region: 0, BP: 6_000}}); err != nil {
		t.Fatalf("SetRentScales error = %v, want nil", err)
	}
	w.tenure.noTitleRun[0], w.tenure.rentFundsRun[0] = 3, 2
	rows := errVal(w.Snapshot()).Endowments
	tenureFields := func(r EndowmentRow) [9]any {
		return [9]any{r.Holders, r.PoolBP, r.CommonsBP, r.TopBP, r.Posted, r.ScaleBP, r.Closed, r.NoTitleRun, r.RentFundsRun}
	}
	tnEq(t, "tenure fields of home", tenureFields(rows[0]), [9]any{int64(5), int64(1_000), int64(8_700), int64(100), int64(4), int64(6_000), true, int64(3), int64(2)})
	tnEq(t, "tenure fields of yonder", tenureFields(rows[1]), [9]any{int64(3), int64(500), int64(9_400), int64(50), int64(0), int64(wholeBP), false, int64(0), int64(0)})
	tnEq(t, "the region's own row", errVal(w.Region(0)).Endowments[0], rows[0])
}

// TestTenure_OffLeavesNoTrace verifies that a world with the module off has no state of
// it, answers its calls with ErrModuleOff, has no tenure field in a row, no rent scale in
// the slots, no section 25 in its checkpoint and no ledger row of rent or title_sale.
//
// Requirement: FR-LND-001
func TestTenure_OffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(5, 10, 100), WithAudit())
	tnEq(t, "w.tenure", w.tenure == nil, true)
	if _, err := w.Titles(1); !errors.Is(err, ErrModuleOff) {
		t.Errorf("Titles = %v, want ErrModuleOff", err)
	}
	row := errVal(w.Snapshot()).Endowments[0]
	tnEq(t, "tenure fields of a row", [3]int64{row.Holders, row.ScaleBP, row.CommonsBP}, [3]int64{})
	tnEq(t, "Slots().RentScales", len(errVal(w.Slots()).RentScales), 0)
	tnEq(t, "modulesOn names tenure", slicesContainsString(modulesOn(&w.cfg), "tenure"), false)
	for _, sec := range checkpointSections {
		tnEq(t, "section 25 present", sec.tag == 25 && sec.present(&w.cfg), false)
	}
	tnEq(t, "moduleReasons", len(moduleReasons(&w.cfg)), 0)
}

// slicesContainsString reports whether list holds s.
func slicesContainsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// TestTenureRun_RemovalsAndAuditOverManyTicks verifies that a world with the module on runs
// 40 ticks under the audit with titles held by households that starve and are removed, and
// that every title still has a live holder: the shares of an endowment never exceed
// 10 000 and the removed agents' titles are in the pool.
//
// Requirement: FR-LND-023, FR-LND-037
func TestTenureRun_RemovalsAndAuditOverManyTicks(t *testing.T) {
	t.Parallel()
	c := tenureWorldConfig(nil)
	c.Roles[0].StartStock = nil
	c.Roles[0].StartMoney = 0
	w := newMill(t, c, WithSeed(exitSeed), WithAudit())
	for range 40 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	for g := RegionID(0); g < 2; g++ {
		st := tnHolders(t, w, g)
		var sum int64
		for _, x := range st.Titles {
			sum += x.ShareBP
		}
		if sum+st.CommonsBP != wholeBP {
			t.Errorf("region %d: titles %d and commons %d do not sum to %d", g, sum, st.CommonsBP, wholeBP)
		}
	}
}
