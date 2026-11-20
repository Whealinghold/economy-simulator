package indicator_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// The window of spec-indicators.md 18.11, hired labour: at alpha the grower role is
// hireable and the baker role an employer. In tick 10 LabourRound has Offered 12, Hired 9,
// NoVacancy 3, HouseholdTicks 9 and Paid 270; in tick 11 Offered 10, Hired 8, NoVacancy 2,
// HouseholdTicks 8 and Paid 240.

// labourConfig is valleyConfig with the two roles of the labour window.
func labourConfig() economysim.Config {
	cfg := valleyConfig()
	cfg.Roles = []economysim.Role{
		{ID: "grower", Hireable: true},
		{ID: "baker", Employer: &economysim.Employer{PayrollBP: 1_000}},
	}
	return cfg
}

// labourSnap is a valley snapshot of the labour configuration.
func labourSnap(tick int64, hh [4]int64) economysim.Snapshot {
	s := valleySnap(tick, hh)
	s.ConfigHash = economysim.HashConfig(labourConfig())
	return s
}

// labourEvents are the two LabourRound events of 18.11, with the grower rows and one
// refusal of the formation in tick 10.
func labourEvents() (a, b economysim.LabourRound) {
	a = economysim.LabourRound{Tick: 10, Region: alpha, Offered: 12, Hired: 9, NoVacancy: 3, HouseholdTicks: 9, Paid: 270,
		EmployerFunds: 4, NoInput: 1, Workers: []economysim.LabourRow{{Role: grower, Offered: 12, Hired: 9, NoVacancy: 3}}}
	b = economysim.LabourRound{Tick: 11, Region: alpha, Offered: 10, Hired: 8, NoVacancy: 2, HouseholdTicks: 8, Paid: 240,
		EmployerFunds: 2, Workers: []economysim.LabourRow{{Role: grower, Offered: 10, Hired: 8, NoVacancy: 2}}}
	return a, b
}

// labourWindow runs the two ticks of 18.11 and returns the window and its snapshots. The
// running grower households at the start of the ticks are 40 and 42 (18.7): a weight
// change in tick 10 adds 2. At the close 8 of the 39 grower households are on a hired job
// and 5 hired households work for the bakers.
func labourWindow(t *testing.T, extra ...economysim.Event) (indicator.Window, economysim.Snapshot, economysim.Snapshot) {
	t.Helper()
	open, closing := labourSnap(10, [4]int64{40, 10, 0, 0}), labourSnap(12, [4]int64{39, 11, 0, 0})
	closing.Roles[0].Employed, closing.Roles[1].Employing = 8, 5
	tl := indicator.NewTally()
	if err := tl.Open(open); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	a, b := labourEvents()
	step(tl, 10, append([]economysim.Event{economysim.WeightChanged{Tick: 10, Region: alpha, Role: grower, From: 4, To: 6}, a}, extra...)...)
	step(tl, 11, b)
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	return w, open, closing
}

// labourGDP is the part of the GDP of 18.2 that WageBill reads: the value added 1 200
// and the cash form 1 300 at alpha.
func labourGDP(w indicator.Window) indicator.GDPResult {
	row := indicator.GDPRow{Region: alpha, ValueAdded: 1_200, Cash: 1_300}
	return indicator.GDPResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To,
		Regions: []indicator.GDPRow{row, {Region: beta}}, Realm: indicator.GDPRow{Region: -1, ValueAdded: 1_200, Cash: 1_300}}
}

// TestUnemployment_RateOfTheWorkedWindow verifies the worked row of 18.11 and the three
// counts beside it, by region and worker role and in all, and the refusals by reason.
//
// Arithmetic: NoVacancy 3 + 2 = 5 over Offered 12 + 10 = 22, so the rate is
// rdiv(10 000 * 5, 22) = rdiv(50 000, 22) = 2 273 (2 272.7); Hired 9 + 8 = 17. The refusals
// are employer_funds 4 + 2 = 6 and no_input 1, listed in the order of the checks (no_input
// is check 6, employer_funds check 12).
//
// Requirement: FR-IND-056
func TestUnemployment_RateOfTheWorkedWindow(t *testing.T) {
	t.Parallel()
	w, _, _ := labourWindow(t)
	got, err := indicator.Unemployment(w)
	if err != nil {
		t.Fatalf("Unemployment() error = %v, want nil", err)
	}
	rate := indicator.Index{Value: 2_273, OK: true}
	row := indicator.UnemploymentRow{Region: alpha, Role: grower, Offered: 22, Hired: 17, NoVacancy: 5, Rate: rate}
	region, total := row, row
	region.Role = -1
	total.Region, total.Role = -1, -1
	want := indicator.UnemploymentResult{
		ConfigHash: w.ConfigHash, From: 10, To: 12,
		Rows: []indicator.UnemploymentRow{row}, Regions: []indicator.UnemploymentRow{region}, Total: total,
		ByReason: []indicator.RefusalRow{{Reason: economysim.ReasonNoInput, Count: 1}, {Reason: economysim.ReasonEmployerFunds, Count: 6}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unemployment() = %+v, want %+v", got, want)
	}
}

// TestUnemployment_NothingOfferedIsUndefined verifies that a window with no LabourRound
// has no rows and an undefined rate, not 0, and that a region that only posted vacancies
// (no household offered) has a row whose rate is undefined.
//
// Requirement: FR-IND-056
func TestUnemployment_NothingOfferedIsUndefined(t *testing.T) {
	t.Parallel()
	empty := lossWindow(t)
	got, err := indicator.Unemployment(empty)
	if err != nil || got.Total.Rate.OK || got.Total.Offered != 0 {
		t.Errorf("Unemployment(no labour) = %+v, %v, want an undefined rate over 0 offered", got, err)
	}
	if len(got.Rows)+len(got.Regions)+len(got.ByReason) != 0 {
		t.Errorf("Unemployment(no labour) = %+v, want no rows", got)
	}
	vacancies := lossWindow(t, economysim.LabourRound{Tick: 10, Region: beta, Posted: 3, Unfilled: 3, EmployerFunds: 3})
	got, err = indicator.Unemployment(vacancies)
	if err != nil || len(got.Regions) != 1 || got.Regions[0].Rate.OK || got.Total.Rate.OK {
		t.Errorf("Unemployment(vacancies only) = %+v, %v, want one region row with an undefined rate", got, err)
	}
	if want := []indicator.RefusalRow{{Reason: economysim.ReasonEmployerFunds, Count: 3}}; !reflect.DeepEqual(got.ByReason, want) {
		t.Errorf("Unemployment(vacancies only).ByReason = %+v, want %+v", got.ByReason, want)
	}
}

// TestUnemployment_SplitsRegionsAndRoles verifies that the rows are by (region, worker
// role) in ascending order and the region rows sum their roles, so a rate of one region
// is not another's.
//
// Arithmetic: alpha grower 4 offered, 1 unmatched, 2 500; alpha baker 4 offered, 3
// unmatched, 7 500; alpha 8 offered, 4 unmatched, 5 000; beta grower 2 offered, none
// unmatched, 0; all 10 offered, 4 unmatched, 4 000.
//
// Requirement: FR-IND-056
func TestUnemployment_SplitsRegionsAndRoles(t *testing.T) {
	t.Parallel()
	w := lossWindow(t,
		economysim.LabourRound{Tick: 10, Region: beta, Offered: 2, Hired: 2, Workers: []economysim.LabourRow{{Role: grower, Offered: 2, Hired: 2}}},
		economysim.LabourRound{Tick: 10, Region: alpha, Offered: 8, Hired: 4, NoVacancy: 4, Workers: []economysim.LabourRow{
			{Role: grower, Offered: 4, Hired: 3, NoVacancy: 1}, {Role: baker, Offered: 4, Hired: 1, NoVacancy: 3}}})
	got, err := indicator.Unemployment(w)
	if err != nil {
		t.Fatalf("Unemployment() error = %v, want nil", err)
	}
	var rates []int64
	var keys [][2]int
	for _, r := range got.Rows {
		rates = append(rates, int64(r.Rate.Value))
		keys = append(keys, [2]int{int(r.Region), int(r.Role)})
	}
	if wantKeys := [][2]int{{0, 0}, {0, 1}, {1, 0}}; !reflect.DeepEqual(keys, wantKeys) || !reflect.DeepEqual(rates, []int64{2_500, 7_500, 0}) {
		t.Errorf("Unemployment().Rows = (region, role) %v rates %v, want %v and [2500 7500 0]", keys, rates, wantKeys)
	}
	if len(got.Regions) != 2 || got.Regions[0].Rate.Value != 5_000 || got.Regions[1].Rate.Value != 0 || !got.Regions[1].Rate.OK || got.Total.Rate.Value != 4_000 {
		t.Errorf("Unemployment().Regions = %+v, Total = %+v, want 5 000 and 0 by region, 4 000 in all", got.Regions, got.Total)
	}
}

// TestUnemployment_RejectsABrokenWindow verifies that a hand-built window with a row
// outside the tables is ErrRange and negative dimensions are refused, not indexed.
//
// Requirement: FR-IND-056
func TestUnemployment_RejectsABrokenWindow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		w    indicator.Window
	}{
		{"role row of another region", indicator.Window{Regions: 1, Roles: 1, LabourRoles: []indicator.LabourRoleTally{{Region: 1}}}},
		{"role row of another role", indicator.Window{Regions: 1, Roles: 1, LabourRoles: []indicator.LabourRoleTally{{Role: 1}}}},
		{"region row of another region", indicator.Window{Regions: 1, Roles: 1, Labour: []indicator.LabourTally{{Region: 3}}}},
		{"negative dimension", indicator.Window{Regions: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := indicator.Unemployment(tt.w); !errors.Is(err, indicator.ErrRange) {
				t.Errorf("Unemployment(%s) error = %v, want ErrRange", tt.name, err)
			}
		})
	}
}

// TestUnemployment_SumPastInt64IsOverflow verifies that region rows whose sum passes
// int64 give ErrOverflow and not a wrapped total (FR-IND-009).
//
// Requirement: FR-IND-056
func TestUnemployment_SumPastInt64IsOverflow(t *testing.T) {
	t.Parallel()
	half := int64(1) << 62
	w := indicator.Window{Regions: 2, Roles: 1, Labour: []indicator.LabourTally{{Region: 0, Offered: half}, {Region: 1, Offered: half}}}
	if _, err := indicator.Unemployment(w); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Unemployment(two rows of 2^62) error = %v, want ErrOverflow", err)
	}
}

// TestWageBill_SharesOfTheWorkedWindow verifies the worked row of 18.11: the money paid
// against the value added of 18.2 in its accrual and cash forms.
//
// Arithmetic: Paid = 270 + 240 = 510; Share = rdiv(10 000 * 510, 1 200) = 4 250 exactly;
// ShareCash = rdiv(5 100 000, 1 300) = 3 923 (3 923.08); nothing was refunded.
//
// Requirement: FR-IND-057
func TestWageBill_SharesOfTheWorkedWindow(t *testing.T) {
	t.Parallel()
	w, _, _ := labourWindow(t)
	got, err := indicator.WageBill(w, labourGDP(w))
	if err != nil {
		t.Fatalf("WageBill() error = %v, want nil", err)
	}
	row := indicator.WageBillRow{Region: alpha, Paid: 510, ValueAdded: 1_200, Cash: 1_300,
		Share: indicator.Index{Value: 4_250, OK: true}, ShareCash: indicator.Index{Value: 3_923, OK: true}}
	realm := row
	realm.Region = -1
	want := indicator.WageBillResult{ConfigHash: w.ConfigHash, From: 10, To: 12, Regions: []indicator.WageBillRow{row}, Realm: realm}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WageBill() = %+v, want %+v", got, want)
	}
}

// TestWageBill_AboveTheWholeIsAResult verifies that a wage bill above the value added
// is a share above 10 000 and not an error, that a cash form of 0 has no share and that the Refunded of a cut start is
// reported beside Paid.
//
// Arithmetic: 600 paid over a value added of 500 is rdiv(6 000 000, 500) = 12 000; over a
// cash form of 0 the cash share is undefined; Refunded 30 is summed as it is.
//
// Requirement: FR-IND-057
func TestWageBill_AboveTheWholeIsAResult(t *testing.T) {
	t.Parallel()
	w := lossWindow(t, economysim.LabourRound{Tick: 10, Region: alpha, Paid: 600, Refunded: 30})
	g := indicator.GDPResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To,
		Regions: []indicator.GDPRow{{Region: alpha, ValueAdded: 500}, {Region: beta}}, Realm: indicator.GDPRow{Region: -1, ValueAdded: 500}}
	got, err := indicator.WageBill(w, g)
	if err != nil {
		t.Fatalf("WageBill() error = %v, want nil", err)
	}
	r := got.Regions[0]
	if r.Paid != 600 || r.Refunded != 30 {
		t.Errorf("WageBill().Regions[0] = %+v, want Paid 600 and Refunded 30", r)
	}
	if r.Share.Value != 12_000 || !r.Share.OK || r.ShareCash.OK {
		t.Errorf("WageBill().Regions[0] shares = %+v and %+v, want 12 000 and an undefined cash share", r.Share, r.ShareCash)
	}
}

// TestWageBill_NoShareAtOrBelowZero verifies that a value added at or below 0, in a region
// or in the realm, gives an undefined share and keeps the money paid.
//
// Requirement: FR-IND-057
func TestWageBill_NoShareAtOrBelowZero(t *testing.T) {
	t.Parallel()
	w := lossWindow(t, economysim.LabourRound{Tick: 10, Region: alpha, Paid: 600, Refunded: 30})
	g := indicator.GDPResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To,
		Regions: []indicator.GDPRow{{Region: alpha, ValueAdded: -5, Cash: 1}, {Region: beta}}, Realm: indicator.GDPRow{Region: -1}}
	got, err := indicator.WageBill(w, g)
	if err != nil || got.Realm.Paid != 600 {
		t.Fatalf("WageBill(value added at or below 0) = %+v, %v, want Paid 600", got, err)
	}
	if got.Regions[0].Share.OK || got.Realm.Share.OK {
		t.Errorf("WageBill(value added at or below 0) shares = %+v and %+v, want both undefined", got.Regions[0].Share, got.Realm.Share)
	}
}

// TestWageBill_RejectsAnotherWindow verifies that a GDP of another configuration or
// window is refused with the sentinel of the mismatch, and that a region with a wage bill
// and no GDP row is ErrRange.
//
// Requirement: FR-IND-057
func TestWageBill_RejectsAnotherWindow(t *testing.T) {
	t.Parallel()
	w, _, _ := labourWindow(t)
	tests := []struct {
		name   string
		mutate func(g *indicator.GDPResult)
		want   error
	}{
		{"another configuration", func(g *indicator.GDPResult) { g.ConfigHash[0]++ }, indicator.ErrConfigMismatch},
		{"another window", func(g *indicator.GDPResult) { g.To++ }, indicator.ErrWindowOrder},
		{"no row for the region", func(g *indicator.GDPResult) { g.Regions = g.Regions[1:] }, indicator.ErrRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := labourGDP(w)
			tt.mutate(&g)
			if _, err := indicator.WageBill(w, g); !errors.Is(err, tt.want) {
				t.Errorf("WageBill(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestEmployment_WindowAndCloseRates verifies the worked row of 18.11: the paid
// household-ticks over the running households of the window, and the households on a
// hired job at the close.
//
// Arithmetic: paid household-ticks 9 + 8 = 17 over the running grower households 40 + 42 =
// 82 is rdiv(170 000, 82) = 2 073 (2 073.2); at the close 8 of 39 is rdiv(80 000, 39) =
// 2 051 (2 051.3); beta has a hireable role and no households, so its window rate is
// undefined. The baker role is an employer: its row carries Employing 5 and no rate.
//
// Requirement: FR-IND-058
func TestEmployment_WindowAndCloseRates(t *testing.T) {
	t.Parallel()
	w, _, closing := labourWindow(t)
	got, err := indicator.Employment(w, labourConfig(), closing)
	if err != nil {
		t.Fatalf("Employment() error = %v, want nil", err)
	}
	wantRows := []indicator.EmploymentRow{
		{Region: alpha, Role: grower, Hireable: true, Employed: 8, Households: 39, CloseRate: indicator.Index{Value: 2_051, OK: true}},
		{Region: alpha, Role: baker, Employing: 5, Households: 11},
		{Region: beta, Role: grower, Hireable: true},
		{Region: beta, Role: baker},
	}
	if !reflect.DeepEqual(got.Rows, wantRows) {
		t.Errorf("Employment().Rows = %+v, want %+v", got.Rows, wantRows)
	}
	wantRegions := []indicator.EmploymentRegion{
		{Region: alpha, HouseholdTicks: 17, Running: 82, Rate: indicator.Index{Value: 2_073, OK: true}},
		{Region: beta},
	}
	if !reflect.DeepEqual(got.Regions, wantRegions) {
		t.Errorf("Employment().Regions = %+v, want %+v", got.Regions, wantRegions)
	}
	if want := (indicator.EmploymentRegion{Region: -1, HouseholdTicks: 17, Running: 82, Rate: indicator.Index{Value: 2_073, OK: true}}); got.Total != want {
		t.Errorf("Employment().Total = %+v, want %+v", got.Total, want)
	}
}

// TestEmployment_RoleThatIsNeitherIsAbsent verifies that a role that is not hireable and
// has no employer block has no row, and that the households of such a role do not enter
// the running count of the window rate.
//
// Requirement: FR-IND-058
func TestEmployment_RoleThatIsNeitherIsAbsent(t *testing.T) {
	t.Parallel()
	cfg := labourConfig()
	cfg.Roles[1].Employer = nil
	open, closing := labourSnap(10, [4]int64{40, 10, 0, 0}), labourSnap(11, [4]int64{40, 10, 0, 0})
	open.ConfigHash, closing.ConfigHash = economysim.HashConfig(cfg), economysim.HashConfig(cfg)
	tl := indicator.NewTally()
	if err := tl.Open(open); err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	step(tl, 10, economysim.LabourRound{Tick: 10, Region: alpha, HouseholdTicks: 10})
	w, err := tl.Close(closing)
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	got, err := indicator.Employment(w, cfg, closing)
	if err != nil {
		t.Fatalf("Employment() error = %v, want nil", err)
	}
	for _, r := range got.Rows {
		if r.Role == baker {
			t.Errorf("Employment().Rows has the row %+v of a role that is neither hireable nor an employer, want it absent", r)
		}
	}
	if got.Total.Running != 40 || got.Total.HouseholdTicks != 10 || got.Total.Rate.Value != 2_500 {
		t.Errorf("Employment().Total = %+v, want 10 over the 40 running grower households, 2 500", got.Total)
	}
}

// TestEmployment_RejectsAMismatch verifies the sentinels of a configuration of another
// world, a snapshot of another tick and a window whose tables do not fit its dimensions.
//
// Requirement: FR-IND-058
func TestEmployment_RejectsAMismatch(t *testing.T) {
	t.Parallel()
	w, _, closing := labourWindow(t)
	other := labourConfig()
	other.Roles[0].Hireable = false
	short := labourConfig()
	short.Roles = short.Roles[:1]
	late := closing
	late.Tick++
	bad := closing
	bad.Roles = append([]economysim.RoleRow(nil), closing.Roles...)
	bad.Roles[0].Role = 9
	tests := []struct {
		name string
		cfg  economysim.Config
		s    economysim.Snapshot
		want error
	}{
		{"another configuration", other, closing, indicator.ErrConfigMismatch},
		{"fewer roles", short, closing, indicator.ErrConfigMismatch},
		{"another tick", labourConfig(), late, indicator.ErrWindowOrder},
		{"role row outside the table", labourConfig(), bad, indicator.ErrRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := indicator.Employment(w, tt.cfg, tt.s); !errors.Is(err, tt.want) {
				t.Errorf("Employment(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestTally_LabourRoundSumsByRegionAndRole verifies the rows the tally keeps: the sums of
// the region over ticks, the worker rows by (region, role) in ascending order, and the
// refusals, all over two ticks and two regions.
//
// Requirement: FR-IND-056, FR-IND-057
func TestTally_LabourRoundSumsByRegionAndRole(t *testing.T) {
	t.Parallel()
	w, _, _ := labourWindow(t, economysim.LabourRound{Tick: 10, Region: beta, Offered: 1, Hired: 1, HouseholdTicks: 1, Refunded: 7, NotStarted: 1,
		Workers: []economysim.LabourRow{{Role: baker, Offered: 1, Hired: 1}}, TooMany: 2, NoRoom: 3})
	wantLabour := []indicator.LabourTally{
		{Region: alpha, Offered: 22, Hired: 17, NoVacancy: 5, HouseholdTicks: 17, Paid: 510, Refusals: indicator.LabourRefusals{NoInput: 1, EmployerFunds: 6}},
		{Region: beta, Offered: 1, Hired: 1, NotStarted: 1, HouseholdTicks: 1, Refunded: 7, Refusals: indicator.LabourRefusals{TooMany: 2, NoRoom: 3}},
	}
	if !reflect.DeepEqual(w.Labour, wantLabour) {
		t.Errorf("Window.Labour = %+v, want %+v", w.Labour, wantLabour)
	}
	wantRoles := []indicator.LabourRoleTally{
		{Region: alpha, Role: grower, Offered: 22, Hired: 17, NoVacancy: 5},
		{Region: beta, Role: baker, Offered: 1, Hired: 1},
	}
	if !reflect.DeepEqual(w.LabourRoles, wantRoles) {
		t.Errorf("Window.LabourRoles = %+v, want %+v", w.LabourRoles, wantRoles)
	}
}

// TestTally_LabourRoundOutsideTheTablesIsStray verifies that a LabourRound with a region
// or role outside the names table, a negative count or a tick before the window marks
// the window stray, so Close refuses it, as for every other event.
//
// Requirement: FR-IND-056
func TestTally_LabourRoundOutsideTheTablesIsStray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		e    economysim.LabourRound
	}{
		{"region outside", economysim.LabourRound{Tick: 10, Region: 5}},
		{"negative region", economysim.LabourRound{Tick: 10, Region: -1}},
		{"role outside", economysim.LabourRound{Tick: 10, Workers: []economysim.LabourRow{{Role: 7}}}},
		{"negative offered", economysim.LabourRound{Tick: 10, Offered: -1}},
		{"negative money", economysim.LabourRound{Tick: 10, Paid: -1}},
		{"negative refusal", economysim.LabourRound{Tick: 10, NoRoom: -1}},
		{"negative role count", economysim.LabourRound{Tick: 10, Workers: []economysim.LabourRow{{Role: grower, Hired: -1}}}},
		{"before the window", economysim.LabourRound{Tick: 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tl := indicator.NewTally()
			openValley(t, tl, 10, [4]int64{})
			step(tl, 10, tt.e)
			if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrWindowOrder) {
				t.Errorf("Close() after %s error = %v, want ErrWindowOrder", tt.name, err)
			}
		})
	}
}

// TestTally_LabourRoundOverflowIsReported verifies that a sum of a count past int64
// makes Close return ErrOverflow, as for every other row.
//
// Requirement: FR-IND-056
func TestTally_LabourRoundOverflowIsReported(t *testing.T) {
	t.Parallel()
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	half := int64(1) << 62
	step(tl, 10, economysim.LabourRound{Tick: 10, Offered: half}, economysim.LabourRound{Tick: 10, Offered: half}, economysim.LabourRound{Tick: 10, Offered: half})
	if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Close() after three 2^62 offers error = %v, want ErrOverflow", err)
	}
}
