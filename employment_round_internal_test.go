package economysim

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestAdmission_RefusesByReason verifies, for each admission reason of the offer
// and of the vacancy, that the first failing reason refuses the request whole,
// counts it in the tally of the region and writes a Debug line with the reason, and
// that nothing is hired. The order of the reasons is that of spec-employment.md 5.2.
//
// Requirement: FR-EMP-010, FR-EMP-011, FR-EMP-058
func TestAdmission_RefusesByReason(t *testing.T) {
	t.Parallel()
	farmHire := HireRequest{Recipe: 0, Positions: 1, Bid: 100}
	handOffer := LabourOffer{Households: 1, Reserve: 10}
	cases := []struct {
		name   string
		o      hireOpts
		plan   func(v *AgentView) JobRequest
		reason Reason
		offer  bool // the refusal is an offer's
	}{
		{"offer of a role that is not hireable", hireOpts{}, byRole(JobRequest{NoJob: true, Offer: handOffer}, JobRequest{NoJob: true}), ReasonNotHireable, true},
		{"offer of more households than the weight", hireOpts{}, byRole(JobRequest{NoJob: true}, JobRequest{NoJob: true, Offer: LabourOffer{Households: 2}}), ReasonInvalidHouseholds, true},
		{"offer of a negative number of households", hireOpts{}, byRole(JobRequest{NoJob: true}, JobRequest{NoJob: true, Offer: LabourOffer{Households: -1}}), ReasonInvalidHouseholds, true},
		{"offer at a negative reserve", hireOpts{}, byRole(JobRequest{NoJob: true}, JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: -1}}), ReasonInvalidPrice, true},
		{"offer above the largest price", hireOpts{}, byRole(JobRequest{NoJob: true}, JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: MaxPrice + 1}}), ReasonInvalidPrice, true},
		{"vacancy of a role with no employer block", hireOpts{}, byRole(JobRequest{NoJob: true}, JobRequest{NoJob: true, Hire: farmHire}), ReasonNotEmployer, false},
		{"vacancy of a recipe off the hire list", hireOpts{}, byRole(JobRequest{NoJob: true, Hire: HireRequest{Recipe: 7, Positions: 1, Bid: 100}}, JobRequest{NoJob: true}), ReasonNotOnHireList, false},
		{"vacancy of a negative number of positions", hireOpts{}, byRole(JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: -1, Bid: 100}}, JobRequest{NoJob: true}), ReasonInvalidPositions, false},
		{"vacancy at a bid of 0", hireOpts{}, byRole(JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 1}}, JobRequest{NoJob: true}), ReasonInvalidPrice, false},
		{"vacancy above the largest price", hireOpts{}, byRole(JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 1, Bid: MaxPrice + 1}}, JobRequest{NoJob: true}), ReasonInvalidPrice, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			lg := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			log := &hireLog{}
			w := hireWorld(t, defaultHire(tt.o), tt.plan, WithObserver(log), WithLogger(lg))
			stepN(t, w, 1)
			c := w.employment.tallyOf(0, w.Tick()-1)
			got, want := c.vacanciesRefused, int64(1)
			if tt.offer {
				got = c.offersRefused
			}
			if got != want {
				t.Errorf("refusals counted in the tally = %d, want %d", got, want)
			}
			if len(log.formed) != 0 || len(w.employment.hires) != 0 {
				t.Errorf("hires formed = %d, open = %d, want none after a refused request", len(log.formed), len(w.employment.hires))
			}
			if line := "reason=" + string(tt.reason); !strings.Contains(buf.String(), line) {
				t.Errorf("Debug output %q does not hold %q", buf.String(), line)
			}
		})
	}
}

// TestAdmission_EmployerFullCutsAndRefuses verifies that a vacancy larger than the
// places the establishment has left is cut to them and admitted, and that the
// vacancy of an employer with no place left is refused employer_full: with
// MaxHires 2 and a first tick that fills both, the next tick's vacancy is refused.
//
// Requirement: FR-EMP-011
func TestAdmission_EmployerFullCutsAndRefuses(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 3, maxHires: 2, duration: 5}), farmWants(5, 100, 10))
	stepN(t, w, 1)
	c := w.employment.tallyOf(0, 0)
	if c.posted != 2 || c.filled != 2 || c.unfilled != 0 || c.matched != 2 || c.vacanciesRefused != 0 {
		t.Errorf("tick 0: posted %d filled %d unfilled %d matched %d refused %d, want 2, 2, 0, 2 and 0 (5 positions cut to the establishment 2)",
			c.posted, c.filled, c.unfilled, c.matched, c.vacanciesRefused)
	}
	stepN(t, w, 1)
	if c := w.employment.tallyOf(0, 1); c.vacanciesRefused != 1 || c.matched != 0 {
		t.Errorf("tick 1: vacancies refused %d matched %d, want 1 (employer_full) and 0", c.vacanciesRefused, c.matched)
	}
}

// TestRound_FirstFitByReserveLeavesTheDearerOfferUnmatched verifies the matching on
// a hand-built world with one position and two offers: the lower reserve is matched
// first and gets the wage of its own pair, the dearer offer finds no vacancy and
// its household is counted NoVacancy.
//
// Arithmetic: reserves 30 and 50 at a bid of 100 and a share of 5 000: the first
// pair's wage is 30 + (5 000 * 70 / 10 000) = 65.
//
// Requirement: FR-EMP-019, FR-EMP-020, FR-EMP-021, FR-EMP-060
func TestRound_FirstFitByReserveLeavesTheDearerOfferUnmatched(t *testing.T) {
	t.Parallel()
	reserves := map[AgentID]Price{}
	log := &hireLog{}
	plan := func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 1, Bid: 100}}
		}
		return JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: reserves[v.Agent]}}
	}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2}), plan, WithObserver(log))
	hands := slotsOf(w, "hand")
	reserves[AgentID(hands[0])], reserves[AgentID(hands[1])] = 50, 30
	stepN(t, w, 1)
	if len(log.formed) != 1 || log.formed[0].Worker != AgentID(hands[1]) || log.formed[0].Wage != 65 {
		t.Fatalf("hires formed = %+v, want one, of the hand with the reserve 30, at the wage 65", log.formed)
	}
	if c := w.employment.tallyOf(0, 0); c.offered != 2 || c.hired != 1 || c.noVacancy != 1 {
		t.Errorf("offered %d hired %d noVacancy %d, want 2, 1 and 1", c.offered, c.hired, c.noVacancy)
	}
}

// TestRound_LimitsByRegionAndMaxOpenIsPerRegion verifies that a lane is a region
// and MaxOpen is per region: with MaxOpen 1, each of two regions forms its one hire,
// and a second pair in a region is refused too_many_hires (the lane closes) while
// the other region is not affected.
//
// Requirement: FR-EMP-023, FR-EMP-063
func TestRound_LimitsByRegionAndMaxOpenIsPerRegion(t *testing.T) {
	t.Parallel()
	log := &hireLog{}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, maxOpen: 1, regions: 2}), farmWants(2, 100, 10), WithObserver(log))
	stepN(t, w, 1)
	if len(log.formed) != 2 {
		t.Fatalf("hires formed = %d, want 2 (one in each region)", len(log.formed))
	}
	for g := range 2 {
		c := w.employment.tallyOf(RegionID(g), 0)
		if c.matched != 1 || c.refused[1] != 1 || c.noVacancy != 1 {
			t.Errorf("region %d: matched %d, positions closed too_many_hires %d, noVacancy %d, want 1, 1 and 1", g, c.matched, c.refused[1], c.noVacancy)
		}
	}
	if log.formed[0].Region != 0 || log.formed[1].Region != 1 {
		t.Errorf("regions of the hires = %d, %d, want ascending 0, 1", log.formed[0].Region, log.formed[1].Region)
	}
}

// TestFormation_RefusalsCloseByKind verifies the formation checks that a hand-built
// world can reach, each with the reason it counts: employer_funds (the farm holds
// less than the bill), no_input and no_tool (the stock is below the block) and
// no_site (a sited recipe with no yield row). Nothing moves for a refused pair.
//
// Arithmetic: wage 70 (reserve 40, bid 100), 3 ticks, 1 household: bill 210.
//
// Requirement: FR-EMP-023, FR-EMP-042
func TestFormation_RefusalsCloseByKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		o    hireOpts
		code int
	}{
		{"employer funds below the bill", hireOpts{farmMoney: 209}, lrEmployerFunds},
		{"no input in stock", hireOpts{input: true, farmGrain: 0}, lrNoInput},
		{"no tool in stock", hireOpts{tool: true, farmTools: 0}, lrNoTool},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := hireWorld(t, defaultHire(tt.o), farmWants(1, 100, 40))
			total := totalMoney(t, w)
			stepN(t, w, 1)
			c := w.employment.tallyOf(0, 0)
			if c.refused[tt.code] != 1 || c.matched != 0 || c.unfilled != 1 || c.noVacancy != 1 {
				t.Errorf("refused[%d] = %d, matched %d, unfilled %d, noVacancy %d, want 1, 0, 1 and 1", tt.code, c.refused[tt.code], c.matched, c.unfilled, c.noVacancy)
			}
			if got := totalMoney(t, w); got != total || escrowHeld(t, w) != 0 {
				t.Errorf("household money = %d (want %d), escrow = %d (want 0): a refused formation moves nothing", got, total, escrowHeld(t, w))
			}
		})
	}
}

// TestFormation_ZeroYieldClosesTheVacancy verifies check 3: with the yield shock of the
// recipe at 0 the effective factor of a hired execution is 0, the pair is refused
// zero_yield and the vacancy is closed, so its positions are counted once.
//
// Requirement: FR-EMP-023
func TestFormation_ZeroYieldClosesTheVacancy(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 2}), farmWants(2, 100, 10))
	if err := w.shocks.set(0, 0, 0); err != nil {
		t.Fatalf("shocks.set() = %v, want nil", err)
	}
	stepN(t, w, 1)
	c := w.employment.tallyOf(0, 0)
	if c.refused[lrZeroYield] != 2 || c.matched != 0 || c.noVacancy != 2 {
		t.Errorf("positions closed zero_yield = %d, matched %d, noVacancy %d, want 2, 0 and 2", c.refused[lrZeroYield], c.matched, c.noVacancy)
	}
}

// TestFormation_StockCountsEarlierHiresOfTheRound verifies that the executions
// hired earlier in the round count against the employer's stock: a farm holding 2
// grain, for a recipe that takes 1 grain an execution, can hire two households of
// three offered, and the third offer is refused no_input.
//
// Requirement: FR-EMP-025
func TestFormation_StockCountsEarlierHiresOfTheRound(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 3, input: true, farmGrain: 2}), farmWants(3, 100, 10))
	stepN(t, w, 1)
	c := w.employment.tallyOf(0, 0)
	if c.matched != 2 || c.refused[lrNoInput] != 1 || c.noVacancy != 1 {
		t.Errorf("matched %d, positions closed no_input %d, noVacancy %d, want 2, 1 and 1", c.matched, c.refused[lrNoInput], c.noVacancy)
	}
}

// TestAdmission_NoEndowmentRefusesTheFormation verifies check 7 of the round
// (spec-employment.md 5.5): a vacancy whose recipe draws 3 ore an execution forms no
// hire while the region's stock of ore is below one execution, and counts the
// refusal as no_endowment; with a stock of 3 the same vacancy forms its hire.
//
// Requirement: FR-EMP-023
func TestAdmission_NoEndowmentRefusesTheFormation(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		ore   int64
		hires int
	}{{"stock below one execution", 2, 0}, {"stock of one execution", 3, 1}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			lg := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			w := hireWorld(t, defaultHire(hireOpts{ore: tt.ore}), farmWants(1, 100, 10), WithLogger(lg))
			stepN(t, w, 1)
			if got := len(w.employment.hires); got != tt.hires {
				t.Errorf("hires open = %d, want %d", got, tt.hires)
			}
			if refused := strings.Contains(buf.String(), "reason="+string(ReasonNoEndowment)); refused != (tt.hires == 0) {
				t.Errorf("no_endowment in the Debug output = %v, want %v: %q", refused, tt.hires == 0, buf.String())
			}
		})
	}
}
