package economysim

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestQueries_DeepCopy_NoDraw_NoEvent_Ascending verifies the three queries of spec-land.md
// 9.3: Titles lists an agent's titles in ascending (region, resource), a pool's titles are
// those of its own region, Holders lists the titles in ascending holder ID with the commons
// share, the posted rent, the scale and Closed, Rent is the rent in force, each returns a copy,
// and none draws, emits or changes anything.
//
// Requirement: FR-LND-027
func TestQueries_DeepCopy_NoDraw_NoEvent_Ascending(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, func(m *TenureConfig) { m.Terms[0].HasClosed, m.Terms[0].Closed = true, true }, WithObserver(obs))
	if err := w.TransferTitles([]TitleTransfer{tr(1, 1, 2, 30, 0)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	obs.events = nil
	before := errVal(w.Checkpoint())
	tnEq(t, "Titles(2)", errVal(w.Titles(2)), []TitleState{{Region: 0, Holder: 2, ShareBP: 100}, {Region: 1, Holder: 2, ShareBP: 30}})
	tnEq(t, "Titles(pool of yonder)", errVal(w.Titles(1)), []TitleState{{Region: 1, Holder: 1, ShareBP: 470}})
	tnEq(t, "Titles of a household with no title", len(errVal(w.Titles(5))), 0)
	st := tnHolders(t, w, 0)
	tnEq(t, "Holders(home)", [4]any{st.Closed, st.Posted, st.ScaleBP, st.CommonsBP}, [4]any{true, int64(4), int64(wholeBP), int64(8_700)})
	tnEq(t, "the holders in ascending order", slices.IsSortedFunc(st.Titles, func(a, b TitleState) int { return int(a.Holder - b.Holder) }), true)
	st.Titles[0].ShareBP = 1
	tnEq(t, "the pool's share after a change of the copy", tnHolders(t, w, 0).Titles[0].ShareBP, int64(1_000))
	tnEq(t, "Rent", errVal(w.Rent(0, 0)), int64(4))
	tnEq(t, "the checkpoint after the queries", errVal(w.Checkpoint()), before)
	tnEq(t, "events of the queries", len(obs.events), 0)
}

// TestQueries_Errors verifies the errors of the three queries, and that each is legal in
// the construction phase of a world that has not stepped.
//
// Requirement: FR-LND-027
func TestQueries_Errors(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	off := newMill(t, oreMill(5, 10, 100))
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"titles module off", errOf(off.Titles(1)), ErrModuleOff},
		{"holders module off", errOf(off.Holders(0, 0)), ErrModuleOff},
		{"rent module off", errOf(off.Rent(0, 0)), ErrModuleOff},
		{"titles of an ID never issued", errOf(w.Titles(99)), ErrUnknownAgent},
		{"titles of a negative ID", errOf(w.Titles(NoAgent)), ErrUnknownAgent},
		{"holders of an unknown region", errOf(w.Holders(7, 0)), ErrUnknownRegion},
		{"holders of an unknown resource", errOf(w.Holders(0, 4)), ErrUnknownResource},
		{"rent of an unknown region", errOf(w.Rent(-1, 0)), ErrUnknownRegion},
		{"rent of an unknown resource", errOf(w.Rent(0, -1)), ErrUnknownResource},
	}
	for _, tc := range cases {
		if !errors.Is(tc.err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, tc.err, tc.want)
		}
	}
	if _, err := w.Titles(2); err != nil {
		t.Errorf("Titles in the construction phase = %v, want nil", err)
	}
}

// errOf returns the error of a call whose first result the test does not need.
func errOf[T any](_ T, err error) error { return err }

// TestAudit_SharesSumAndHoldersLive verifies the identities of the audit of spec-land.md 14.2
// and FR-LND-037 (first two parts): a title that is not live, above the cap, out of order,
// of a share outside 1 to 10 000 or beyond the sum of 10 000 fails auditTenure, and the
// state of a fresh world passes.
//
// Requirement: FR-LND-037
func TestAudit_SharesSumAndHoldersLive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mut  func(t *tenure)
		want string
	}{
		{"a fresh world", func(*tenure) {}, ""},
		{"share above the sum", func(t *tenure) { t.holds[0] = append(t.holds[0], title{holder: 5, share: 9_000}) }, "sum"},
		{"dead holder", func(t *tenure) { t.holds[0] = append(t.holds[0], title{holder: 9, share: 1}) }, "not a live holder"},
		{"unordered", func(t *tenure) { t.holds[0][1], t.holds[0][2] = t.holds[0][2], t.holds[0][1] }, "not above"},
		{"zero share", func(t *tenure) { t.holds[0][1].share = 0 }, "outside 1 to"},
		{"share above 10000", func(t *tenure) { t.holds[0][1].share = 10_001 }, "outside 1 to"},
		{"pool of another region", func(t *tenure) { t.holds[0] = append([]title{{holder: 1, share: 1}}, t.holds[0]...) }, "not a live holder"},
		{"agent above the cap", func(t *tenure) { t.capBP = 50 }, "above the cap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := tenureWorld(t, nil)
			tc.mut(w.tenure)
			err := w.auditTenure()
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("auditTenure of a fresh world = %v, want nil", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Errorf("auditTenure = %v, want an error naming %q", err, tc.want)
			}
		})
	}
}

// TestAudit_RunsInAuditModules verifies that the audit of the world reaches the tenure
// audit: a corrupt title fails World.Audit with ErrAuditFailed.
//
// Requirement: FR-LND-037
func TestAudit_RunsInAuditModules(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit of a fresh world = %v, want nil", err)
	}
	w.tenure.holds[1][0].share = 20_000
	if err := w.Audit(); !errors.Is(err, ErrAuditFailed) {
		t.Errorf("Audit with a corrupt title = %v, want ErrAuditFailed", err)
	}
}

// TestEvents_RebuildMoneyFromTitleTransferred verifies FR-OBS-041 for a priced transfer:
// the household money of the payer's region falls and that of the payee's region rises by
// Price, as the event fields say, and, for an enclosure, the pool of Region gains it.
//
// Requirement: FR-OBS-041, FR-LND-029
func TestEvents_RebuildMoneyFromTitleTransferred(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	buyer, err := w.AddAgent(NewAgent{Region: 1, Money: 500})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	snap := func() (house, pool [2]int64) {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot error = %v, want nil", err)
		}
		for g := range 2 {
			house[g], pool[g] = int64(s.Regions[g].MoneyHouseholds), int64(s.Regions[g].MoneyPool)
		}
		return house, pool
	}
	h0, p0 := snap()
	obs.events = nil
	if err := w.TransferTitles([]TitleTransfer{tr(0, 2, buyer, 40, 60), tr(0, NoAgent, buyer, 10, 15)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	h1, p1 := snap()
	wantH, wantP := h0, p0
	for _, e := range of[TitleTransferred](obs) {
		pay := int64(e.Price)
		if e.ToPool {
			wantP[e.ToRegion] -= pay
		} else {
			wantH[e.ToRegion] -= pay
		}
		if e.FromPool {
			wantP[e.FromRegion] += pay
		} else {
			wantH[e.FromRegion] += pay
		}
	}
	if h1 != wantH || p1 != wantP {
		t.Errorf("money by region after = %v and pool %v, rebuilt from the events %v and %v", h1, p1, wantH, wantP)
	}
}
