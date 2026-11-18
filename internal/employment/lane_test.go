package employment_test

import (
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// employer is a test employer: the plan its positions come from and the
// vacancy it posts with them.
type employer struct {
	plan    employment.Plan
	vacancy employment.Vacancy
}

// post turns an employer's plan into its vacancy through Positions, the way the
// default strategy does, and reports false when it posts nothing.
func post(t *testing.T, ref int, p employment.Plan, share int64, key uint64) (employer, bool) {
	t.Helper()
	n, ok := employment.Positions(p)
	if !ok {
		t.Fatalf("Positions(%+v) = %d, %v, want ok", p, n, ok)
	}
	if n == 0 {
		return employer{}, false
	}
	return employer{plan: p, vacancy: employment.Vacancy{Ref: ref, Positions: n, Bid: p.Bid, Duration: p.Duration, ShareBP: share, Key: key}}, true
}

// fundsCheck is the employer-funds formation check of the world (spec-employment.md
// 5.5) as the root builds it: the bill of the pair, with the bills the
// vacancy's earlier hires already hold, must fit the employer's cash. It
// closes by fit, as a check that depends on the size of the block does.
func fundsCheck(cash map[int]int64, failures *int) employment.Check {
	return func(o employment.Offer, v employment.Vacancy, wage int64, formed []employment.Hire) (int, employment.Close) {
		bill, ok := employment.Bill(wage, v.Duration, o.Households)
		if !ok {
			*failures++
			return employment.ReasonCheck, employment.CloseFit
		}
		for _, h := range formed {
			if h.Vacancy != v.Ref {
				continue
			}
			held, ok := employment.Bill(h.Wage, v.Duration, h.Households)
			if !ok || bill+held < bill {
				*failures++
				return employment.ReasonCheck, employment.CloseFit
			}
			bill += held
		}
		if bill > cash[v.Ref] {
			*failures++
			return employment.ReasonCheck, employment.CloseFit
		}
		return employment.ReasonNone, employment.CloseFit
	}
}

// TestLane_PostRoundBill_WorkedCycle verifies the hire arithmetic as one lane
// uses it: Positions sizes two employers' vacancies from their stock, tools
// and cash, Round matches five offers to them, the wage of each hire is the
// half-even share of its margin and Bill prices each hire within the
// employer's cash.
//
// Requirement: FR-EMP-015, FR-EMP-019, FR-EMP-020, FR-EMP-021, FR-EMP-042
func TestLane_PostRoundBill_WorkedCycle(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	a, b := workedEmployers(t)
	cash := map[int]int64{10: 90, 11: 50}
	offers := []employment.Offer{
		{Ref: 4, Households: 1, Reserve: 7}, {Ref: 3, Households: 1, Reserve: 5},
		{Ref: 5, Households: 2, Reserve: 3}, {Ref: 2, Households: 1, Reserve: 2}, {Ref: 1, Households: 1, Reserve: 1},
	}
	var failures int
	var r employment.Result
	employment.Round(&r, offers, []employment.Vacancy{b.vacancy, a.vacancy}, employment.Lane{MaxOpen: 10}, fundsCheck(cash, &failures))

	// Offers in order of reserve: 1 (r1), 2 (r2), 5 (r3, two households), 3 (r5), 4 (r7).
	// Offer 1 at A: 1 + halfEven(5 000 * 5 / 10 000 = 2.5) = 1 + 2 = 3, bill 3 * 2 * 1 = 6.
	// Offer 2 at A: 2 + 5 000 * 4 / 10 000 = 2 + 2 = 4, bill 4 * 2 * 1 = 8.
	// Offer 5 needs 2 and A has 1 left, so it goes to B: 3 + halfEven(0.5) = 3 + 0 = 3, bill 3 * 3 * 2 = 18.
	// Offer 3 at A: 5 + halfEven(0.5) = 5, bill 5 * 2 * 1 = 10; A is full.
	// Offer 4 at r7 is above A's bid of 6 and B's of 4: unmatched.
	want := []employment.Hire{
		{Offer: 1, Vacancy: 10, Households: 1, Wage: 3, Reserve: 1, Bid: 6},
		{Offer: 2, Vacancy: 10, Households: 1, Wage: 4, Reserve: 2, Bid: 6},
		{Offer: 5, Vacancy: 11, Households: 2, Wage: 3, Reserve: 3, Bid: 4},
		{Offer: 3, Vacancy: 10, Households: 1, Wage: 5, Reserve: 5, Bid: 6},
	}
	if !slices.Equal(r.Hires, want) {
		t.Errorf("Round hires = %+v, want %+v", r.Hires, want)
	}
	if failures != 0 {
		t.Errorf("the funds check refused %d pairs, want 0: Positions bounds the vacancy by the cash", failures)
	}
	assertWorkedOutcome(t, &r, map[int]employment.Vacancy{10: a.vacancy, 11: b.vacancy}, cash)
}

// assertWorkedOutcome checks the bills, the offers and the vacancies of the worked cycle.
func assertWorkedOutcome(t *testing.T, r *employment.Result, vacancies map[int]employment.Vacancy, cash map[int]int64) {
	t.Helper()
	billed := billsByVacancy(r.Hires, vacancies)
	if billed[10] != 24 || billed[11] != 18 {
		t.Errorf("Bill totals per employer = %v, want map[10:24 11:18] (6 + 8 + 10 and 18)", billed)
	}
	hired := make([]bool, len(r.Offers))
	for i, o := range r.Offers {
		hired[i] = o.Hired
	}
	if !slices.Equal(hired, []bool{true, true, true, true, false}) {
		t.Errorf("Round offers hired = %v, want [true true true true false]", hired)
	}
	if got, want := []int64{r.Vacancies[0].Left, r.Vacancies[1].Left}, []int64{0, 0}; !slices.Equal(got, want) || r.WageBelowUnit != 0 {
		t.Errorf("Round positions left = %v, WageBelowUnit = %d, want %v, 0", got, r.WageBelowUnit, want)
	}
}

// billsByVacancy sums Bill over the hires of each vacancy.
func billsByVacancy(hires []employment.Hire, vacancies map[int]employment.Vacancy) map[int]int64 {
	billed := map[int]int64{}
	for _, h := range hires {
		bill, _ := employment.Bill(h.Wage, vacancies[h.Vacancy].Duration, h.Households)
		billed[h.Vacancy] += bill
	}
	return billed
}

// workedEmployers posts the two employers of the worked cycle.
func workedEmployers(t *testing.T) (a, b employer) {
	t.Helper()
	// Employer A (Ref 10): MaxHires 5 less 1 hired is 4; stock (14 - 2) / 3 is
	// 4; 3 tools; cash (100 - 10) / (6 * 2) = 90 / 12 is 7. min is 3 positions.
	a, ok := post(t, 10, employment.Plan{MaxHires: 5, Hired: 1, Inputs: []employment.Input{{Stock: 14, Keep: 2, Qty: 3}},
		Tools: []int64{3}, Money: 100, UpkeepCost: 10, Bid: 6, Duration: 2}, 5000, 0)
	if !ok || a.vacancy.Positions != 3 {
		t.Fatalf("Positions of employer A = %+v, want 3 positions", a.vacancy)
	}
	// Employer B (Ref 11): MaxHires 2, no inputs or tools, cash 50 / (4 * 3) is
	// 4. min is 2 positions.
	b, ok = post(t, 11, employment.Plan{MaxHires: 2, Money: 50, Bid: 4, Duration: 3}, 5000, 0)
	if !ok || b.vacancy.Positions != 2 {
		t.Fatalf("Positions of employer B = %+v, want 2 positions", b.vacancy)
	}
	return a, b
}

// lane is a random lane built from a seed: the employers' plans become
// vacancies through Positions, the offers are random.
type lane struct {
	offers    []employment.Offer
	vacancies []employment.Vacancy
	cash      map[int]int64
	maxOpen   int64
}

// randomLane draws a lane from seed. The keys are drawn from a small range so
// that ties are common and the key and then the Ref decide them.
func randomLane(t *testing.T, seed uint64) lane {
	t.Helper()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	l := lane{cash: map[int]int64{}, maxOpen: rng.Int64N(12) + 1}
	for i := range rng.IntN(8) + 1 {
		p := employment.Plan{
			MaxHires: rng.Int64N(6), Hired: rng.Int64N(3), KOwn: rng.Int64N(2),
			Inputs: []employment.Input{{Stock: rng.Int64N(40), Keep: rng.Int64N(4), Qty: rng.Int64N(4) + 1}},
			Tools:  []int64{rng.Int64N(8)},
			Money:  rng.Int64N(400), UpkeepCost: rng.Int64N(20), Bid: rng.Int64N(14), Duration: rng.Int64N(4) + 1,
		}
		e, ok := post(t, 1000+i, p, rng.Int64N(employment.WholeBP+1), uint64(rng.IntN(3)))
		if !ok {
			continue
		}
		l.vacancies = append(l.vacancies, e.vacancy)
		l.cash[e.vacancy.Ref] = max(p.Money-p.UpkeepCost, 0)
	}
	for i := range rng.IntN(20) + 1 {
		l.offers = append(l.offers, employment.Offer{Ref: i, Households: rng.Int64N(3) + 1, Reserve: rng.Int64N(14), Key: uint64(rng.IntN(3))})
	}
	return l
}

// TestLane_Invariants_RandomLanesOverSeeds verifies, over seeds 1 to 300, the
// invariants of a lane built from Positions, Round, Wage and Bill: an offer is
// hired at most once and whole, the households hired equal the positions
// filled, positions are conserved, every wage lies in [reserve, bid], no
// employer's bills exceed its cash (the bound of Positions makes the funds
// check pass for every wage at or below the bid), the hires stay under MaxOpen,
// and the result does not depend on the order the offers and vacancies arrive
// in. Exact equality throughout; a failure names the seed.
//
// Requirement: FR-EMP-015, FR-EMP-019, FR-EMP-021, FR-EMP-042, FR-EMP-063, NFR-EMP-003
//
// Guard: Round and Positions are tested apart; this fixes the contract between
// them. A change that let a vacancy outgrow the employer's cash, or made the
// order of the inputs matter, would pass every unit test of one of them.
func TestLane_Invariants_RandomLanesOverSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for seed := uint64(1); seed <= 300; seed++ {
		l := randomLane(t, seed)
		var failures int
		check := fundsCheck(l.cash, &failures)
		var first employment.Result
		employment.Round(&first, slices.Clone(l.offers), slices.Clone(l.vacancies), employment.Lane{MaxOpen: l.maxOpen}, check)
		assertLaneInvariants(t, seed, l, &first, failures)

		slices.Reverse(l.offers)
		slices.Reverse(l.vacancies)
		var second employment.Result
		employment.Round(&second, l.offers, l.vacancies, employment.Lane{MaxOpen: l.maxOpen}, check)
		if !reflect.DeepEqual(first, second) {
			t.Errorf("seed %d: Round depends on the input order:\n%+v\n%+v", seed, first, second)
		}
	}
}

// assertLaneInvariants checks one round's result against its lane.
func assertLaneInvariants(t *testing.T, seed uint64, l lane, r *employment.Result, failures int) {
	t.Helper()
	if failures != 0 {
		t.Errorf("seed %d: the funds check refused %d pairs, want 0", seed, failures)
	}
	if int64(len(r.Hires)) > l.maxOpen {
		t.Errorf("seed %d: %d hires under MaxOpen %d", seed, len(r.Hires), l.maxOpen)
	}
	vacancies := map[int]employment.Vacancy{}
	for _, v := range l.vacancies {
		vacancies[v.Ref] = v
	}
	assertHires(t, seed, l, r)
	for ref, total := range billsByVacancy(r.Hires, vacancies) {
		if total > l.cash[ref] {
			t.Errorf("seed %d: vacancy %d billed %d, more than its cash %d", seed, ref, total, l.cash[ref])
		}
	}
	assertFilled(t, seed, vacancies, r)
}

// assertHires checks that each hire takes a whole offer once, at a wage within
// its limits, and that the offer results agree with the hires.
func assertHires(t *testing.T, seed uint64, l lane, r *employment.Result) {
	t.Helper()
	block := map[int]int64{}
	for _, o := range l.offers {
		block[o.Ref] = o.Households
	}
	hired := map[int]bool{}
	for _, h := range r.Hires {
		if hired[h.Offer] || h.Households != block[h.Offer] {
			t.Errorf("seed %d: hire %+v hires offer %d twice or not whole (block %d)", seed, h, h.Offer, block[h.Offer])
		}
		hired[h.Offer] = true
		if h.Wage < max(h.Reserve, 1) || h.Wage > h.Bid {
			t.Errorf("seed %d: hire %+v has a wage outside [max(reserve, 1), bid]", seed, h)
		}
	}
	for _, o := range r.Offers {
		if o.Hired != hired[o.Ref] {
			t.Errorf("seed %d: offer %+v disagrees with the hires", seed, o)
		}
	}
}

// assertFilled checks that each vacancy's filled positions equal the
// households of its hires and that positions are conserved.
func assertFilled(t *testing.T, seed uint64, vacancies map[int]employment.Vacancy, r *employment.Result) {
	t.Helper()
	took := map[int]int64{}
	for _, h := range r.Hires {
		took[h.Vacancy] += h.Households
	}
	for _, v := range r.Vacancies {
		if v.Filled != took[v.Ref] || v.Left < 0 || v.Filled+v.Left != vacancies[v.Ref].Positions {
			t.Errorf("seed %d: vacancy %+v: filled %d by the hires, positions %d", seed, v, took[v.Ref], vacancies[v.Ref].Positions)
		}
	}
}

// TestCircuit_InvalidOverflows verifies that every checked product of the
// closed form reports false instead of wrapping.
//
// Requirement: FR-EMP-062, NFR-EMP-003
func TestCircuit_InvalidOverflows(t *testing.T) {
	t.Parallel()
	good := employment.Circuit{Employers: 10, Upkeep: 3, Units: 6, PhiBP: 5000, Places: 100}
	tests := []struct {
		name   string
		change func(c *employment.Circuit)
	}{
		// MaxInt64 * 1 fits, and 10 000 + MaxInt64 does not.
		{"gain past int64", func(c *employment.Circuit) { c.SpendMaxShareBP = math.MaxInt64; c.SpendHalfTicks = 1 }},
		// (10 000 - 0) * MaxInt64 does not fit.
		{"denominator past int64", func(c *employment.Circuit) { c.PhiBP = 0; c.Units = math.MaxInt64 }},
		// 10^17 * 1 fits and 10^17 * 1 000 = 10^20 does not.
		{"supply in thousandths past int64", func(c *employment.Circuit) { c.Employers = 1e17; c.Upkeep = 1 }},
		// supply 10^9 * gain about 10^18 / den 10 000 * 6 is about 10^22, past int64.
		{"quotient past int64", func(c *employment.Circuit) {
			c.Employers, c.Upkeep, c.PhiBP = 1e6, 1, 0
			c.SpendMaxShareBP, c.SpendHalfTicks = 1e9, 1e9
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := good
			tt.change(&c)
			if got, ok := employment.HiredMilli(c); ok {
				t.Errorf("HiredMilli(%+v) = %d, true, want false", c, got)
			}
		})
	}
}
