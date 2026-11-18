package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tenureSteppedPack is the golden pack of the tenure module with the step of the rent on, so
// the posted rent is state (saved) and moves: 2 500 basis points of the rent of 2 is a step
// of 1, inside the band 0 to 4 of the defaults. On the five at seed 1 the posted rent of the
// north falls to 0 at tick 60 and a let unit is drawn there (tick 60 of the probe of the
// golden test).
func tenureSteppedPack(t *testing.T) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packFiveTenure...)
	m := *cfg.Modules.Tenure
	m.RentStepBP = 2_500
	cfg.Modules.Tenure = &m
	return cfg
}

// firstTitleHolders returns the first two household holders of the endowment, in ascending
// agent ID (the pool is below every agent and skipped).
func firstTitleHolders(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID) (economysim.AgentID, economysim.AgentID) {
	t.Helper()
	st, err := w.Holders(g, r)
	if err != nil {
		t.Fatalf("Holders(%d, %d) error = %v, want nil", g, r, err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	var out []economysim.AgentID
	for _, x := range st.Titles {
		if x.Holder >= economysim.AgentID(len(snap.Names.Regions)) {
			out = append(out, x.Holder)
		}
	}
	if len(out) < 2 {
		t.Fatalf("the endowment has %d household holders, want at least 2", len(out))
	}
	return out[0], out[1]
}

// TestFork_Tenure verifies FR-LND-034 and the fork at N of CHK 8.1: the original and two
// restores give the same events, snapshot, audit and checkpoint bytes after N ticks of
// {0, 61 (the tick after a start that drew a let unit and after a rent step: the posted rent
// has moved), a call that moves a title and sets a rent scale, a removal of a title holder}.
// The posted rent, the scale, the titles and the runs are saved state.
//
// Requirement: FR-LND-034, FR-CHK-048
//
// Guard: a restore that lost the posted rent, the scale or a title, or kept the entitlements
// or the reservations of the old tick, would differ at the first rent step after it.
func TestFork_Tenure(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	cfg := tenureSteppedPack(t)
	move := func(w *economysim.World) {
		g, r := s111Endowment(t, w)
		from, to := firstTitleHolders(t, w, g, r)
		if err := w.TransferTitles([]economysim.TitleTransfer{{Region: g, Resource: r, From: from, To: to, ShareBP: 5}, {Region: g, Resource: r, From: to, To: economysim.NoAgent, ShareBP: 2}}); err != nil {
			t.Errorf("TransferTitles error = %v, want nil", err)
		}
		if err := w.SetRentScales([]economysim.EndowmentScale{{Region: g, Resource: r, BP: 7_500}}); err != nil {
			t.Errorf("SetRentScales error = %v, want nil", err)
		}
	}
	remove := func(w *economysim.World) {
		g, r := s111Endowment(t, w)
		from, _ := firstTitleHolders(t, w, g, r)
		id, err := w.AddAgent(economysim.NewAgent{Region: g, Money: 100})
		if err != nil {
			t.Fatalf("AddAgent error = %v, want nil", err)
		}
		gift := economysim.TitleTransfer{Region: g, Resource: r, From: from, To: id, ShareBP: 5}
		if err := w.TransferTitles([]economysim.TitleTransfer{gift}); err != nil {
			t.Fatalf("TransferTitles(a gift to the added agent) error = %v, want nil", err)
		}
		if err := w.RemoveAgent(id); err != nil {
			t.Errorf("RemoveAgent(%d) error = %v, want nil", id, err)
		}
	}
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"before the first step", 0, nil},
		{"after a let unit and a rent step", 61, nil},
		{"after a transfer and a scale", 30, move},
		{"after a removal that moved a title to the pool", 30, remove},
	} {
		forkAtN(t, forkCase{Name: fmt.Sprintf("tenure %s", c.name), Cfg: cfg, Seed: 1, N: c.n, M: forkM, Before: c.before,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }})
	}
}

// TestDeterminism_Tenure verifies FR-LND-036 and FR-LND-043 on the golden pack with the step
// of the rent on, seed 1, 400 ticks: a run split into Run calls, with a logger, an observer
// or the audit, ends in the snapshot and the state hash of the bare Step run, and its events
// equal the reference's. The module takes no draw, so the stream of the world is the same
// with the module on or off.
//
// Requirement: FR-LND-036, FR-LND-043, FR-OBS-023, FR-WLD-063
//
// Guard: the posted rent and the runs are state; a logger, an observer or the chunking of Run
// that moved either would fail here.
func TestDeterminism_Tenure(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	cfg := tenureSteppedPack(t)
	seed := uint64(1)
	base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
	ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
	refEvents := renderEvents(withoutRunBrackets(ref))
	if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
		t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
	}
	for _, tc := range detCases(true) {
		spec := tc.spec()
		spec.Seed, spec.Ticks = seed, detTicks
		got := runPack(t, cfg, spec)
		if !reflect.DeepEqual(got.Final, base.Final) {
			t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
		}
		if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
			t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
		}
		if spec.Observe {
			if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
				t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
			}
		}
	}
}
