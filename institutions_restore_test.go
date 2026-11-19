package economysim_test

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of what a restore does with the module's saved state beyond the refusals of
// institutions_report_test.go, and the invariants of the membership over seeds (plan 3-org
// PR B, spec-institutions.md 6, 11, 12).

// orgUvarints returns the unsigned varints of the values, the encoding of tag 31.
func orgUvarints(vs ...uint64) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.AppendUvarint(b, v)
	}
	return b
}

// TestCheckpoint_InstitutionsLevelsAndSuspensionsRestore verifies the accepting side of the
// decoder of tag 31: a body that carries a level for a price control, one for a rate limit
// and a suspended right restores; the snapshot of the restored world shows the levels and
// the suspension; the world writes the same body again; and it steps on with the Warn run
// it was given.
//
// Arithmetic: the controls sort together by ID, assize 0 and usury 1; the configured levels
// are (12 000, 8 000) and (1 200, 0), so (15 000, 9 000) and (500, 0) differ from them and
// fit (the floor is not above the cap; a rate limit has no floor); the rights sort by ID,
// make_tools 0 and sell_tools 1, so the suspended right is 1; the one run row is region 5,
// body 0, run 3, the run of the world after 3 ticks.
//
// Requirement: FR-ORG-033, FR-ORG-038, FR-ORG-044
//
// Guard: slot calls that set a level or a suspension are not built yet, so no other test
// reaches the accepting side of the two readers; a decoder that kept a level in the wrong
// row or dropped a suspension would pass every refusal test and fail here.
func TestCheckpoint_InstitutionsLevelsAndSuspensionsRestore(t *testing.T) {
	t.Parallel()
	cfg := orgWaitingConfig()
	w := newPackWorld(t, cfg, 1)
	step(t, w, 3)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	body := slices.Concat(
		orgUvarints(2, 0, 15_000, 9_000, 1, 500, 0), // two levels
		orgUvarints(1, 1),       // one suspended right
		orgUvarints(1, 5, 0, 3), // one run row
	)
	c := reframe(t, raw, func(s []section) []section {
		return editSection(s, 31, func([]byte) []byte { return body })
	})
	restored, err := economysim.RestoreWorld(cfg, c)
	if err != nil {
		t.Fatalf("RestoreWorld(a body with two levels and a suspension) error = %v, want nil", err)
	}
	s := snapshotOf(t, restored)
	wantLevels := []economysim.LegalRow{{Control: 0, CapBP: 15_000, FloorBP: 9_000}, {Control: 1, CapBP: 500, FloorBP: 0}}
	if !slices.Equal(s.LegalLevels, wantLevels) {
		t.Errorf("Snapshot.LegalLevels = %+v, want %+v", s.LegalLevels, wantLevels)
	}
	if want := []economysim.RightID{1}; !slices.Equal(s.Suspended, want) {
		t.Errorf("Snapshot.Suspended = %v, want %v", s.Suspended, want)
	}
	if got := orgSection(t, mustMarshal(t, mustCheckpoint(t, restored))); !slices.Equal(got, body) {
		t.Errorf("tag 31 of the restored world = %v, want the body it was given, %v", got, body)
	}
	rec := &recorder{}
	again, err := economysim.RestoreWorld(cfg, c, economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("RestoreWorld(the same body, with an observer) error = %v, want nil", err)
	}
	step(t, again, 1)
	if got := membershipEvents(rec); len(got) != 1 || got[0].Waiting != 15 || got[0].Members != 40 {
		t.Errorf("MembershipChanged after the first step of the restored world = %+v, want one event with 40 members and 15 waiting", got)
	}
}

// TestInvariants_InstitutionsMembership verifies, over seeds and after every tick of a
// world built with fewer places than eligible households, what the round promises: the
// members of a body never exceed its places or its eligible households, the waiting
// households are the eligible less the members in every event, and the household count is
// the one of the world with the bodies alone, no right and no legal price (nobody leaves for a place). The event is not
// compared with the snapshot: it reports the round of P1.9a, the snapshot the end of the
// tick, after the exits and entries of phase 7 (spec-institutions.md 11.1).
//
// Arithmetic: north has 55 makers and the guild 40 places, so from the first round on there
// are 40 members and 15 waiting at a round; the figures come from the events and the
// snapshot, not from constants, so a change in the pack moves the expectation with it.
//
// Requirement: FR-ORG-006, FR-ORG-043, FR-ORG-053
//
// Guard: the unit tests of internal/institution prove the bound over small populations; this
// is the same promise over the whole world, where the weights, the lapses and the order of
// the regions meet.
func TestInvariants_InstitutionsMembership(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const ticks = 30
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			// The world of the bodies alone: the rights and the legal prices bind the
			// economy since PR C, so the household count is compared with the world
			// without the module only where the round is the module's one effect.
			bodies := orgWaitingConfig()
			bodies.Modules.Institutions.Rights, bodies.Modules.Institutions.Controls, bodies.Modules.Institutions.RateLimits = nil, nil, nil
			base := bodies
			base.Modules.Institutions = nil
			off := newPackWorld(t, base, seed)
			rec := &recorder{}
			on := newPackWorld(t, bodies, seed, economysim.WithObserver(rec))
			for tick := 1; tick <= ticks; tick++ {
				step(t, off, 1)
				step(t, on, 1)
				orgCheckMembership(t, seed, snapshotOf(t, on), snapshotOf(t, off), rec.events)
				rec.events = rec.events[:0]
			}
		})
	}
}

// orgCheckMembership checks the invariants of TestInvariants_InstitutionsMembership for one
// tick: s is the snapshot of the world with the module, ref the snapshot of the same world
// without it, and events the events of the tick.
func orgCheckMembership(t *testing.T, seed uint64, s, ref economysim.Snapshot, events []economysim.Event) {
	t.Helper()
	if len(s.Bodies) == 0 {
		t.Fatalf("seed %d tick %d: Snapshot.Bodies is empty, want a row for the one body in its scope", seed, s.Tick)
	}
	orgCheckRows(t, seed, s)
	if got, want := orgHouseholds(s), orgHouseholds(ref); got != want {
		t.Errorf("seed %d tick %d: households = %d, want %d, those of the world without the module", seed, s.Tick, got, want)
	}
	for _, ev := range events {
		if e, ok := ev.(economysim.MembershipChanged); ok {
			orgCheckEvent(t, seed, e)
		}
	}
}

// orgCheckRows checks that no row of the snapshot has more members than places or eligible
// households.
func orgCheckRows(t *testing.T, seed uint64, s economysim.Snapshot) {
	t.Helper()
	for _, row := range s.Bodies {
		if row.Members > row.Places || row.Members > row.Eligible || row.Members < 0 {
			t.Errorf("seed %d tick %d: row %+v, want 0 <= members <= min(places, eligible)", seed, s.Tick, row)
		}
	}
}

// orgCheckEvent checks that an event has no more members than places and that its waiting
// households are its eligible less its members.
func orgCheckEvent(t *testing.T, seed uint64, e economysim.MembershipChanged) {
	t.Helper()
	if e.Members > e.Places || e.Waiting != e.Eligible-e.Members {
		t.Errorf("seed %d: event %+v, want members <= places and waiting = eligible - members", seed, e)
	}
}

// orgHouseholds returns the households of the snapshot, the sum of the weights of its roles.
func orgHouseholds(s economysim.Snapshot) int64 {
	var n int64
	for _, r := range s.Roles {
		n += r.Households
	}
	return n
}
