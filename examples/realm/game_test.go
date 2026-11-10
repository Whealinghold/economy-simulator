package main

import (
	"bytes"
	"errors"
	"reflect"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOnce loads the pack once for the tests of the package: Load runs the
// start-up solve, and every game of a test shares its configuration.
var packOnce = sync.OnceValues(Load)

// newTestGame builds a game over the shared configuration, with the audit on.
func newTestGame(t testing.TB, seed uint64) *Game {
	t.Helper()
	cfg, err := packOnce()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	g, err := NewGameFrom(cfg, seed, economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewGameFrom(seed %d) = %v, want nil", seed, err)
	}
	return g
}

// gameTicks is the length of a run of the tests: a 5 000-household world is about
// ten times slower under the race detector, so it steps a fifth of the ticks there.
func gameTicks() int64 {
	if raceEnabled {
		return 20
	}
	return 100
}

// player is the player of the tests: a traveller in the crownlands with some money
// and grain.
var player = PlayerSpec{Region: "crownlands", Role: "traveller", Money: 5_000, Goods: map[string]int64{"grain": 10, "bread": 2}}

// TestGame_RunsAndKeepsTheInvariants verifies that the realm runs with the audit on:
// every ten ticks Audit() is nil (goods and money conserved, nothing negative), and
// every region still has households.
//
// Requirement: FR-REF-028
// Guard: testing.md 6.3. A rename that broke a recipe or a role would show as an
// audit failure or an emptied region within the run.
func TestGame_RunsAndKeepsTheInvariants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	g := newTestGame(t, 1)
	for done := int64(0); done < gameTicks(); done += 10 {
		if _, err := g.Advance(10); err != nil {
			t.Fatalf("Advance(10) at tick %d = %v, want nil", g.Tick(), err)
		}
		if err := g.Audit(); err != nil {
			t.Fatalf("Audit() at tick %d = %v, want nil", g.Tick(), err)
		}
	}
	s, err := g.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if len(s.Regions) != 6 {
		t.Fatalf("Snapshot().Regions = %d rows, want 6 (five regions and the overseas node)", len(s.Regions))
	}
	for _, r := range s.Regions {
		if !r.Outside && r.Households <= 0 {
			t.Errorf("region %d has %d households at tick %d, want above zero", r.Region, r.Households, s.Tick)
		}
	}
}

// TestGame_PlayerRoundTrip verifies AddPlayer and Player: the player is the agent the
// world reports, with the money and goods given, and the market of a commodity is
// readable by name.
//
// Requirement: FR-INT-004
func TestGame_PlayerRoundTrip(t *testing.T) {
	t.Parallel()
	g := newTestGame(t, 1)
	if _, err := g.Player(); !errors.Is(err, ErrNoPlayer) {
		t.Errorf("Player() before AddPlayer = %v, want ErrNoPlayer", err)
	}
	id, err := g.AddPlayer(player)
	if err != nil {
		t.Fatalf("AddPlayer() = %v, want nil", err)
	}
	s, err := g.Player()
	if err != nil {
		t.Fatalf("Player() = %v, want nil", err)
	}
	if s.ID != id || !s.External || int64(s.Money) != player.Money {
		t.Errorf("Player() = {ID %d, External %t, Money %d}, want {ID %d, External true, Money %d}", s.ID, s.External, s.Money, id, player.Money)
	}
	if _, err := g.AddPlayer(player); !errors.Is(err, ErrPlayerExists) {
		t.Errorf("second AddPlayer() = %v, want ErrPlayerExists", err)
	}
	if _, err := g.Advance(2); err != nil {
		t.Fatalf("Advance(2) = %v, want nil", err)
	}
	m, err := g.Market("crownlands", "grain")
	if err != nil {
		t.Fatalf("Market(crownlands, grain) = %v, want nil", err)
	}
	if m.Price <= 0 {
		t.Errorf("Market(crownlands, grain).Price = %d, want above zero after two ticks", m.Price)
	}
}

// TestGame_RefusesBadInput verifies that a name the pack does not define, a
// negative amount and a negative tick count are refused with the sentinel and change
// nothing.
//
// Requirement: FR-INT-004
func TestGame_RefusesBadInput(t *testing.T) {
	t.Parallel()
	g := newTestGame(t, 1)
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"unknown region", func() error { _, err := g.AddPlayer(PlayerSpec{Region: "atlantis", Role: "traveller"}); return err }, ErrUnknownName},
		{"unknown role", func() error { _, err := g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "wizard"}); return err }, ErrUnknownName},
		{"unknown commodity", func() error {
			_, err := g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "traveller", Goods: map[string]int64{"mithril": 1}})
			return err
		}, ErrUnknownName},
		{"negative money", func() error {
			_, err := g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "traveller", Money: -1})
			return err
		}, ErrBadInput},
		{"zero quantity", func() error {
			_, err := g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "traveller", Goods: map[string]int64{"grain": 0}})
			return err
		}, ErrBadInput},
		{"negative ticks", func() error { _, err := g.Advance(-1); return err }, ErrBadInput},
		{"market of an unknown region", func() error { _, err := g.Market("atlantis", "grain"); return err }, ErrUnknownName},
		{"market of an unknown commodity", func() error { _, err := g.Market("crownlands", "mithril"); return err }, ErrUnknownName},
		{"an agent of the overseas node", func() error { _, err := g.AddPlayer(PlayerSpec{Region: "overseas", Role: "traveller"}); return err }, economysim.ErrOutsideNode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, tt.want) {
				t.Errorf("%s: error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
	if g.Tick() != 0 {
		t.Errorf("Tick() after the refused calls = %d, want 0", g.Tick())
	}
	if _, err := g.Player(); !errors.Is(err, ErrNoPlayer) {
		t.Errorf("Player() after the refused calls = %v, want ErrNoPlayer", err)
	}
}

// TestGame_CheckpointResumesBitIdentical verifies that a checkpoint taken mid-run and
// restored into a second game continues exactly as the first: the same snapshot and
// the same checkpoint bytes after the same further ticks, and the player survives.
//
// Requirement: FR-CHK-057, FR-REF-032
func TestGame_CheckpointResumesBitIdentical(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	a := newTestGame(t, 7)
	if _, err := a.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer() = %v, want nil", err)
	}
	if _, err := a.Advance(10); err != nil {
		t.Fatalf("Advance(10) = %v, want nil", err)
	}
	saved, err := a.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	if _, err := a.Advance(10); err != nil {
		t.Fatalf("Advance(10) after the checkpoint = %v, want nil", err)
	}
	b := newTestGame(t, 7)
	if err := b.Restore(saved); err != nil {
		t.Fatalf("Restore() = %v, want nil", err)
	}
	if _, err := b.Player(); err != nil {
		t.Errorf("Player() after the restore = %v, want the player kept", err)
	}
	if _, err := b.Advance(10); err != nil {
		t.Fatalf("Advance(10) after the restore = %v, want nil", err)
	}
	sa, _ := a.Snapshot()
	sb, _ := b.Snapshot()
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("snapshot at tick %d after the restore differs from the uninterrupted run", sa.Tick)
	}
	ca, _ := a.Checkpoint()
	cb, _ := b.Checkpoint()
	if !bytes.Equal(ca, cb) {
		t.Errorf("checkpoint at tick %d after the restore differs from the uninterrupted run (%d and %d bytes)", sa.Tick, len(cb), len(ca))
	}
}

// TestGame_RestoreRefusesGarbageAndDropsAMissingPlayer verifies that a restore of
// bytes that are no checkpoint is refused and leaves the game as it was, and that a
// checkpoint taken before the player existed leaves the game without one.
//
// Requirement: FR-CHK-016
func TestGame_RestoreRefusesGarbageAndDropsAMissingPlayer(t *testing.T) {
	t.Parallel()
	g := newTestGame(t, 3)
	before, err := g.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	if _, err := g.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer() = %v, want nil", err)
	}
	if err := g.Restore([]byte{1, 2}); !errors.Is(err, ErrBadInput) {
		t.Errorf("Restore(2 bytes) = %v, want ErrBadInput", err)
	}
	if err := g.Restore(append(make([]byte, headerLen), "not a checkpoint"...)); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
		t.Errorf("Restore(garbage) = %v, want ErrCheckpointCorrupt", err)
	}
	if _, err := g.Player(); err != nil {
		t.Errorf("Player() after a refused restore = %v, want the player kept", err)
	}
	if err := g.Restore(before); err != nil {
		t.Fatalf("Restore(before the player) = %v, want nil", err)
	}
	if _, err := g.Player(); !errors.Is(err, ErrNoPlayer) {
		t.Errorf("Player() after restoring a checkpoint without it = %v, want ErrNoPlayer", err)
	}
}
