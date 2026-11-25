package economysim_test

import (
	"context"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the end-to-end test of the cover round through the public API alone (plan
// 3-ins Phase 5; FR-INS-012 to 016, 030, 034 to 036, 070, 072, NFR-INS-003): the shipper
// pack with the risk, credit and insurance modules on, a strategy that asks for cover on
// the lane of each shipment it sends, and one that offers capacity. Nothing here reads an
// unexported name; the policies are told by CoverFormed and CoverRound, and the world runs
// under the audit.

// coverOfferLiability is an offered liability above the maximum payout of one shipment of
// the pack (about 13 700) and below the underwriters' exposure cap (refused from about
// 19 500), so an offer is admitted and covers one policy.
const coverOfferLiability = 14_000

// coverKnobs are what a case of the end-to-end test changes.
type coverKnobs struct {
	liability  int64 // MaxLiability of each offer
	onlyFirst  bool  // only the first household of each region offers
	noInsured  bool  // the carrier role has no insured block
	ownRegion  bool  // the requests name the agent's own region as the destination
	moduleOff  bool  // the module and the blocks are not configured
	insureTerm economysim.InsureTerms
}

// coverScript is the strategy of the pack: the shipper, then the cover the knobs ask for.
type coverScript struct {
	shipper
	k       coverKnobs
	backer  economysim.RoleID
	insured economysim.RoleID
}

func (s coverScript) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.shipper.Trade(v, rng, p)
	// The default strategy asked for cover of its own; the knobs are the only ask.
	p.Cover, p.Offer = p.Cover[:0], p.Offer[:0]
	if v.Role != s.insured {
		// Only the underwriter role holds a block; the other roles' offers would all be refused.
		if v.Role == s.backer && (!s.k.onlyFirst || v.Agent == s.first[v.Region]) {
			p.Offer = append(p.Offer, economysim.CoverOffer{MaxLiability: s.k.liability, LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000})
		}
		return
	}
	for _, sh := range p.Shipments {
		l := economysim.LaneCover{Mode: sh.Mode, To: sh.To, MinCoverBP: 5_000, MaxLoadingBP: 20_000}
		if s.k.ownRegion {
			l.To = v.Region
		}
		p.Cover = append(p.Cover, l)
	}
}

// coverPack returns the shipper pack with the three modules on, the carrier an insured and
// the other role an underwriter, and the strategy of the knobs.
func coverPack(t testing.TB, k coverKnobs) (economysim.Config, []economysim.Option) {
	t.Helper()
	cfg, _ := riskShipperPack(t)
	carrier := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
	backer := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID != "carrier" })
	if carrier < 0 || backer < 0 {
		t.Fatalf("the shipper pack has no carrier or no other role: %d, %d", carrier, backer)
	}
	if !k.moduleOff {
		cfg.Modules.Credit = &economysim.Credit{MinPrincipal: 1, ShieldTicks: 1, WriteOffTicks: 3, MaxClaims: 1_000}
		cfg.Modules.Insurance = &economysim.Insurance{ValueCapBP: 20_000, PayoutTicks: 1, MinSum: 1, MaxPolicies: 1_000, MaxCoverRequests: 4}
		if !k.noInsured {
			it := k.insureTerm
			cfg.Roles[carrier].Cover = &economysim.CoverTerms{Insure: &it}
		}
		cfg.Roles[backer].Cover = &economysim.CoverTerms{Underwrite: &economysim.UnderwriteTerms{
			LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000, CapBP: 20_000, ShareBP: 5_000}}
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(cover pack) error = %v, want nil", err)
	}
	// shipperFor builds a probe world with no "ship" strategy registered, so the roles
	// carry the built-in one until it puts "ship" on them.
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "default"
	}
	cfg, sh := shipperFor(t, cfg)
	cs := coverScript{shipper: sh, k: k, backer: economysim.RoleID(backer), insured: economysim.RoleID(carrier)}
	return cfg, []economysim.Option{economysim.WithStrategy("ship", cs)}
}

// coverRun runs the pack for ticks ticks under the audit and returns the events after
// the construction.
func coverRun(t testing.TB, k coverKnobs, seed uint64, ticks int64) []economysim.Event {
	t.Helper()
	cfg, opts := coverPack(t, k)
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, append(opts, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithAudit())...)
	if err != nil {
		t.Fatalf("NewWorld(cover pack) error = %v, want nil", err)
	}
	rec.forget()
	if _, err := w.Run(context.Background(), ticks); err != nil {
		t.Fatalf("Run(%d) error = %v, want nil (seed %d)", ticks, err, seed)
	}
	return rec.events
}

// coverOf returns the cover events of a run in the order they were told.
func coverOf(events []economysim.Event) (formed []economysim.CoverFormed, rounds []economysim.CoverRound) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.CoverFormed:
			formed = append(formed, e)
		case economysim.CoverRound:
			rounds = append(rounds, e)
		}
	}
	return formed, rounds
}

// coverKnobsDefault are the knobs of the case in which every offer covers one policy.
func coverKnobsDefault() coverKnobs {
	return coverKnobs{liability: coverOfferLiability,
		insureTerm: economysim.InsureTerms{MinLossBP: 1, MaxLoadingBP: 20_000, MinCoverBP: 5_000}}
}

// checkFormedPolicy checks the figures of the i-th CoverFormed of a run.
func checkFormedPolicy(t *testing.T, i int64, f economysim.CoverFormed) {
	t.Helper()
	if f.Policy != i {
		t.Errorf("formed[%d].Policy = %d, want %d (policies are numbered in order of formation)", i, f.Policy, i)
	}
	if f.Kind != economysim.CoverRoute || f.Shipment == economysim.NoShipment {
		t.Errorf("policy %d: Kind %v Shipment %d, want a route policy of a shipment", f.Policy, f.Kind, f.Shipment)
	}
	if f.Insured == f.Underwriter || f.Region != f.UnderwriterRegion {
		t.Errorf("policy %d: insured %d region %d, underwriter %d region %d, want two agents of one region", f.Policy, f.Insured, f.Region, f.Underwriter, f.UnderwriterRegion)
	}
	checkPolicyFigures(t, f)
}

// checkPolicyFigures checks the money figures of a CoverFormed.
func checkPolicyFigures(t *testing.T, f economysim.CoverFormed) {
	t.Helper()
	if f.Sum <= 0 || f.Premium <= 0 || f.RateBP <= 0 || f.CoverBP != 8_000 {
		t.Errorf("policy %d: sum %d premium %d rate %d cover %d, want positive figures at cover 8000", f.Policy, f.Sum, f.Premium, f.RateBP, f.CoverBP)
	}
	// The maximum payout is the cover share of the sum, rounded half to even.
	if d := int64(f.MaxPayout)*10_000 - f.CoverBP*int64(f.Sum); d < -5_000 || d > 5_000 {
		t.Errorf("policy %d: MaxPayout %d for sum %d at cover %d, want the share rounded to the unit", f.Policy, f.MaxPayout, f.Sum, f.CoverBP)
	}
}

// checkRoundsAreTotals checks that each CoverRound is the total of the CoverFormed of its
// region and tick, and that every region and tick with a policy has a round.
func checkRoundsAreTotals(t *testing.T, formed []economysim.CoverFormed, rounds []economysim.CoverRound) {
	t.Helper()
	type key struct {
		tick   int64
		region economysim.RegionID
	}
	var sums = map[key][3]int64{}
	for _, f := range formed {
		s := sums[key{f.Tick, f.Region}]
		sums[key{f.Tick, f.Region}] = [3]int64{s[0] + 1, s[1] + int64(f.Sum), s[2] + int64(f.Premium)}
	}
	for _, r := range rounds {
		s := sums[key{r.Tick, r.Region}]
		if r.Formed != s[0] || int64(r.Sum) != s[1] || int64(r.Premium) != s[2] {
			t.Errorf("CoverRound tick %d region %d: formed %d sum %d premium %d, want the totals of its CoverFormed %v", r.Tick, r.Region, r.Formed, r.Sum, r.Premium, s)
		}
		if r.Formed+r.NoOffer+r.BelowMinimum+r.Funds+r.Overflow > r.Candidates {
			t.Errorf("CoverRound tick %d region %d: outcomes exceed the %d candidates: %+v", r.Tick, r.Region, r.Candidates, r)
		}
		delete(sums, key{r.Tick, r.Region})
	}
	if len(sums) != 0 {
		t.Errorf("CoverFormed of %d (tick, region) pairs have no CoverRound, want one round for each", len(sums))
	}
}

// TestCoverE2E_FormsAndTellsPolicies verifies that a strategy asking for cover on the
// lane of its shipment and underwriters offering capacity form route policies at P3.3c
// (FR-INS-030, 034 to 036, 070, 072): each CoverFormed is a route policy of the carrier
// with an underwriter of its own region, a maximum payout of the cover share of the sum,
// a premium and a rate; the policies are numbered in order; each CoverRound is the total
// of the CoverFormed events of its region and tick and comes after them; and the audit,
// which runs every tick, holds.
//
// Requirement: FR-INS-030, FR-INS-034, FR-INS-035, FR-INS-036, FR-INS-070, FR-INS-072
//
// Guard: the round is the first step of the module that moves money; a policy formed but
// not told, or told with a figure that is not the one the round booked, would hide a
// premium from every consumer.
func TestCoverE2E_FormsAndTellsPolicies(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	events := coverRun(t, coverKnobsDefault(), 1, 20)
	formed, rounds := coverOf(events)
	if len(formed) == 0 {
		t.Fatal("no CoverFormed in 20 ticks, want policies formed on the lane of the shipments")
	}
	for i, f := range formed {
		checkFormedPolicy(t, int64(i), f)
	}
	checkRoundsAreTotals(t, formed, rounds)
	lastFormed, firstRound := -1, len(events)
	for i, e := range events {
		switch e.(type) {
		case economysim.CoverFormed:
			lastFormed = i
		case economysim.CoverRound:
			firstRound = min(firstRound, i)
		}
	}
	if lastFormed < 0 || firstRound > len(events)-1 {
		t.Fatalf("event positions: last CoverFormed %d, first CoverRound %d", lastFormed, firstRound)
	}
}

// TestCoverE2E_RationedAtTheFreeCapacity verifies that a single underwriter offering room
// for one policy forms one policy a region and tick (FR-INS-034, 035): the other
// requests of the tick find no offer, are counted as unfilled and form nothing, and the
// maximum payout of a policy never exceeds the liability offered.
//
// Requirement: FR-INS-034, FR-INS-035, FR-INS-072
//
// Guard: if the room of an offer were not spent by the policy it covers, one underwriter
// would cover every request of the tick past its cap.
func TestCoverE2E_RationedAtTheFreeCapacity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	k := coverKnobsDefault()
	k.onlyFirst = true
	formed, rounds := coverOf(coverRun(t, k, 1, 20))
	if len(formed) == 0 {
		t.Fatal("no CoverFormed in 20 ticks, want one policy a region on the tick of the shipments")
	}
	for _, f := range formed {
		if int64(f.MaxPayout) > coverOfferLiability {
			t.Errorf("policy %d: MaxPayout %d above the offered liability %d", f.Policy, f.MaxPayout, coverOfferLiability)
		}
	}
	rationed := false
	for _, r := range rounds {
		rationed = checkRationedRound(t, r) || rationed
	}
	if !rationed {
		t.Error("no round had more than one candidate, the case rations nothing")
	}
}

// checkRationedRound checks a round of the single-underwriter case and reports whether it
// had more than one candidate, so that some of them went unfilled.
func checkRationedRound(t *testing.T, r economysim.CoverRound) bool {
	t.Helper()
	if r.Candidates == 0 {
		return false
	}
	if r.Offers != 1 || r.Formed != 1 {
		t.Errorf("tick %d region %d: offers %d formed %d, want one offer and one policy", r.Tick, r.Region, r.Offers, r.Formed)
	}
	if r.Candidates > 1 && (r.NoOffer != r.Candidates-1 || r.Unfilled <= 0) {
		t.Errorf("tick %d region %d: %d candidates, NoOffer %d, Unfilled %d, want every candidate but one unfilled", r.Tick, r.Region, r.Candidates, r.NoOffer, r.Unfilled)
	}
	return r.Candidates > 1
}

// TestCoverE2E_Refusals verifies the refusals a strategy meets through the public API
// (FR-INS-012 to 016, 072): an offer above the underwriter's room is refused whole as
// exposure and nothing forms; a request naming the agent's own region is refused; a
// request from a role with no insured block is refused; and no refusal moves money or
// ends the run.
//
// Requirement: FR-INS-012, FR-INS-013, FR-INS-014, FR-INS-016, FR-INS-072
//
// Guard: a refused ask is counted and logged, never an error of the run: a strategy that
// asks for the impossible must not stop the world.
func TestCoverE2E_Refusals(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	tests := []struct {
		name  string
		set   func(k *coverKnobs)
		check func(t *testing.T, r economysim.CoverRound)
	}{
		{"an offer above the room is refused whole", func(k *coverKnobs) { k.liability = 100_000 }, func(t *testing.T, r economysim.CoverRound) {
			if r.Offers != 0 || r.OffersRefused == 0 || r.Exposure != r.OffersRefused {
				t.Errorf("offers %d refused %d exposure %d, want every offer refused as exposure", r.Offers, r.OffersRefused, r.Exposure)
			}
		}},
		{"a request to the agent's own region", func(k *coverKnobs) { k.ownRegion = true }, func(t *testing.T, r economysim.CoverRound) {
			if r.Requests != 0 || r.RequestsRefused == 0 {
				t.Errorf("requests %d refused %d, want every request refused", r.Requests, r.RequestsRefused)
			}
		}},
		{"a request from a role with no insured block", func(k *coverKnobs) { k.noInsured = true }, func(t *testing.T, r economysim.CoverRound) {
			if r.Requests != 0 || r.RequestsRefused == 0 {
				t.Errorf("requests %d refused %d, want every request refused", r.Requests, r.RequestsRefused)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := coverKnobsDefault()
			tt.set(&k)
			formed, rounds := coverOf(coverRun(t, k, 1, 20))
			if len(formed) != 0 {
				t.Errorf("%d policies formed, want none", len(formed))
			}
			seen := false
			for _, r := range rounds {
				if r.Requests+r.RequestsRefused == 0 {
					continue
				}
				seen = true
				if r.Formed != 0 || r.Premium != 0 {
					t.Errorf("tick %d region %d: formed %d premium %d, want nothing", r.Tick, r.Region, r.Formed, r.Premium)
				}
				tt.check(t, r)
			}
			if !seen {
				t.Error("no round saw a request, the case refuses nothing")
			}
		})
	}
}

// TestCoverE2E_ModuleOffIsSilent verifies that a strategy asking for cover in a world with
// the module off is refused without a trace (FR-INS-016): no cover event, the run
// completes and the audit holds.
//
// Requirement: FR-INS-001, FR-INS-016
//
// Guard: the plan fields exist whether or not the module does; a consumer's strategy must
// not need to know the configuration.
func TestCoverE2E_ModuleOffIsSilent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	k := coverKnobsDefault()
	k.moduleOff = true
	events := coverRun(t, k, 1, 20)
	formed, rounds := coverOf(events)
	if len(formed) != 0 || len(rounds) != 0 {
		t.Errorf("%d CoverFormed and %d CoverRound with the module off, want none", len(formed), len(rounds))
	}
	if !slices.ContainsFunc(events, func(e economysim.Event) bool { _, ok := e.(economysim.ShipmentDispatched); return ok }) {
		t.Error("no shipment was dispatched, the case asks for cover on nothing")
	}
}

// TestCoverE2E_Deterministic verifies NFR-INS-003 and 004 through the public API: two runs
// of one seed tell the same events, whose cover events are a function of the seed alone,
// and two seeds each run to the end under the audit.
//
// Requirement: FR-INS-030, NFR-INS-003
//
// Guard: the round takes no random word; a tie broken by map order or a draw would show
// as two runs of one seed that differ.
func TestCoverE2E_Deterministic(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range []uint64{1, 7} {
		a := coverRun(t, coverKnobsDefault(), seed, 20)
		b := coverRun(t, coverKnobsDefault(), seed, 20)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("seed %d: two runs differ (%d and %d events)", seed, len(a), len(b))
		}
		if f, _ := coverOf(a); len(f) == 0 {
			t.Errorf("seed %d: no policy formed, the comparison would pass for nothing", seed)
		}
	}
}
