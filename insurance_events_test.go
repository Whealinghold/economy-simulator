package economysim_test

import (
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// coverEventTypes are the insurance events built so far (PR C); CoverEnded and CoverSettled
// join the list with PR D.
func coverEventTypes() []reflect.Type {
	return []reflect.Type{reflect.TypeFor[economysim.CoverFormed](), reflect.TypeFor[economysim.CoverRound]()}
}

// TestEvent_CoverFormedFields verifies the fields of CoverFormed over a run of the cover
// pack (FR-INS-070): a route policy names a shipment, a mode and a destination other than
// the insured's region; the sum, maximum payout and premium are positive with the maximum
// payout at most the sum; the rate and the cover share are in basis points; and the policy
// IDs run from 0 in formation order with the tick not going back.
//
// Requirement: FR-INS-070
//
// Guard: a route policy told with NoShipment, NoMode or its own region as the destination
// would mislead a consumer that joins the event to a shipment.
func TestEvent_CoverFormedFields(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg, _ := coverPack(t, coverKnobsDefault())
	regions, modes := len(cfg.Regions), len(cfg.Modules.Transport.Modes)
	formed, _ := coverOf(coverRun(t, coverKnobsDefault(), 1, 20))
	if len(formed) == 0 {
		t.Fatal("no CoverFormed in 20 ticks, want policies formed")
	}
	var lastTick int64
	for i, f := range formed {
		if f.Policy != int64(i) || f.Tick < lastTick {
			t.Errorf("formed[%d]: Policy %d Tick %d after tick %d, want policy %d and a tick that does not go back", i, f.Policy, f.Tick, lastTick, i)
		}
		lastTick = f.Tick
		checkCoverLane(t, f, regions, modes)
		checkCoverFigures(t, f)
	}
}

// checkCoverLane checks the kind, shipment, mode and destination of a route CoverFormed.
func checkCoverLane(t *testing.T, f economysim.CoverFormed, regions, modes int) {
	t.Helper()
	if f.Kind != economysim.CoverRoute {
		t.Errorf("policy %d: Kind %q, want %q (the pack has no store request)", f.Policy, f.Kind, economysim.CoverRoute)
	}
	if f.Shipment < 0 || int(f.Mode) < 0 || int(f.Mode) >= modes {
		t.Errorf("policy %d: Shipment %d Mode %d, want a shipment and one of %d modes", f.Policy, f.Shipment, f.Mode, modes)
	}
	if int(f.To) < 0 || int(f.To) >= regions || f.To == f.Region {
		t.Errorf("policy %d: To %d from region %d, want another of %d regions", f.Policy, f.To, f.Region, regions)
	}
}

// checkCoverFigures checks the money and basis-point figures of a CoverFormed.
func checkCoverFigures(t *testing.T, f economysim.CoverFormed) {
	t.Helper()
	if f.Sum <= 0 || f.MaxPayout <= 0 || f.MaxPayout > f.Sum || f.Premium <= 0 {
		t.Errorf("policy %d: Sum %d MaxPayout %d Premium %d, want 0 < MaxPayout <= Sum and Premium > 0", f.Policy, f.Sum, f.MaxPayout, f.Premium)
	}
	if f.RateBP <= 0 || f.CoverBP <= 0 || f.CoverBP > 10_000 {
		t.Errorf("policy %d: RateBP %d CoverBP %d, want a positive rate and a cover share of at most 10000", f.Policy, f.RateBP, f.CoverBP)
	}
}

// TestEvent_CoverRoundNeverAllZero verifies that CoverRound is never emitted with all its
// counts at 0 (FR-INS-072): each round of the run has a request or an offer admitted or
// refused, and so says something.
//
// Requirement: FR-INS-072
//
// Guard: an empty round per region and tick would put an event on every tick of every
// world with the module on, which spec-insurance.md 15.1 forbids.
func TestEvent_CoverRoundNeverAllZero(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	_, rounds := coverOf(coverRun(t, coverKnobsDefault(), 1, 20))
	if len(rounds) == 0 {
		t.Fatal("no CoverRound in 20 ticks, want rounds")
	}
	for _, r := range rounds {
		zero := r
		zero.Tick, zero.Region = 0, 0
		if zero == (economysim.CoverRound{}) {
			t.Errorf("CoverRound tick %d region %d has every count at 0, want none emitted", r.Tick, r.Region)
		}
	}
}

// TestEvent_InsuranceEventsPlainData verifies by reflection that the insurance events
// hold plain data (FR-INS-070): no map, pointer, function, channel or interface, and a
// Tick field, so a consumer can marshal them on another goroutine.
//
// Requirement: FR-INS-070, FR-INS-072
//
// Guard: a map or pointer in an event breaks the golden rendering and the copy rule of
// observability-and-logging.md 4.1.
func TestEvent_InsuranceEventsPlainData(t *testing.T) {
	t.Parallel()
	for _, typ := range coverEventTypes() {
		t.Run(typ.Name(), func(t *testing.T) {
			t.Parallel()
			if f, ok := typ.FieldByName("Tick"); !ok || f.Type.Kind() != reflect.Int64 {
				t.Errorf("%s has no int64 Tick field, want one", typ.Name())
			}
			for i := range typ.NumField() {
				f := typ.Field(i)
				switch f.Type.Kind() {
				case reflect.Map, reflect.Pointer, reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer, reflect.Slice:
					t.Errorf("%s.%s has kind %v, want plain data", typ.Name(), f.Name, f.Type.Kind())
				}
				if f.Anonymous {
					t.Errorf("%s embeds %s, want a named field (architecture.md 4.7)", typ.Name(), f.Name)
				}
			}
		})
	}
}

// TestCardinality_InsuranceAttributes verifies that the attributes of the insurance
// metrics are closed sets of the configuration (FR-INS-073): the rule file lists the
// values of cover_kind, cover_loss and cover_limit, CoverFormed carries a cover kind of
// the closed set and regions, a mode of the configuration, and CoverRound has one count for
// each value of cover_limit that the round feeds.
//
// Requirement: FR-INS-073
//
// Guard: a value of cover_limit with no count on the event, or a kind outside the set,
// would make the attribute unbounded or the instrument empty.
func TestCardinality_InsuranceAttributes(t *testing.T) {
	t.Parallel()
	checkCoverClosedSets(t)
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg, _ := coverPack(t, coverKnobsDefault())
	regions := len(cfg.Regions)
	formed, rounds := coverOf(coverRun(t, coverKnobsDefault(), 1, 20))
	if len(formed) == 0 || len(rounds) == 0 {
		t.Fatalf("%d CoverFormed and %d CoverRound, want both: the run exercised nothing", len(formed), len(rounds))
	}
	for _, f := range formed {
		if f.Kind != economysim.CoverRoute && f.Kind != economysim.CoverStore {
			t.Errorf("policy %d: Kind %q, want a cover kind", f.Policy, f.Kind)
		}
		checkCoverRegions(t, f, regions)
	}
	for _, r := range rounds {
		if int(r.Region) < 0 || int(r.Region) >= regions {
			t.Errorf("CoverRound tick %d: Region %d, want below %d", r.Tick, r.Region, regions)
		}
	}
}

// checkCoverRegions checks that both regions of a CoverFormed are regions of the configuration.
func checkCoverRegions(t *testing.T, f economysim.CoverFormed, regions int) {
	t.Helper()
	if int(f.Region) >= regions || int(f.UnderwriterRegion) >= regions {
		t.Errorf("policy %d: regions %d and %d, want below %d", f.Policy, f.Region, f.UnderwriterRegion, regions)
	}
}

// checkCoverClosedSets checks that the rule file lists the closed sets of the insurance
// attributes and that CoverRound has one count for each value of cover_limit.
func checkCoverClosedSets(t *testing.T) {
	t.Helper()
	rules := readFile(t, "doc/rules/observability-and-logging.md")
	for _, set := range []string{
		"`economysim.cover_kind` (`route`, `store`)",
		"`economysim.cover_loss` (`dropped`, `cut`)",
		"`economysim.cover_limit` (`in_arrears`, `exposure`, `no_risk`, `no_offer`, `below_minimum`, `funds`, `too_many`, `overflow`)",
	} {
		if !strings.Contains(rules, set) {
			t.Errorf("observability-and-logging.md 5.4.1 does not list %s", set)
		}
	}
	round := reflect.TypeFor[economysim.CoverRound]()
	for _, f := range []string{"InArrears", "Exposure", "NoRisk", "NoOffer", "BelowMinimum", "Funds", "TooMany", "Overflow"} {
		if _, ok := round.FieldByName(f); !ok {
			t.Errorf("CoverRound has no field %s, want one count for each value of cover_limit", f)
		}
	}
}
