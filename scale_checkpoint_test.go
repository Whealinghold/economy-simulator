package economysim_test

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// ckptShape is what a checkpoint must keep equal whatever the weights: the tags
// of its sections in order, and the counts of the state they hold.
type ckptShape struct {
	tags  []uint64
	state economysim.ScaleState
	lots  int64
}

// shapeOf reads the shape of w's checkpoint at its boundary.
func shapeOf(t *testing.T, w *economysim.World) ckptShape {
	t.Helper()
	var s ckptShape
	for _, sec := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sections {
		s.tags = append(s.tags, sec.tag)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	for i := range snap.Markets {
		s.lots += snap.Markets[i].Lots
	}
	s.state = economysim.ScaleStateOf(w)
	return s
}

// TestScaleCheckpoint_SizeAndRoundTripDoNotDependOnTheWeights verifies that two
// worlds with the same number of agents and weights 1 and 40 have checkpoints of
// the same shape at tick 0 (the same sections in order and the same agent slots,
// ledger accounts, lots, belief entries, residues and welfare ring slots), that
// after 20 ticks they have the same sections, and that each restores from its
// checkpoint and checkpoints byte-equal again. The byte length is not compared:
// every number is a minimal LEB128 varint, and the holdings and balances at weight
// 40 need longer ones (spec-checkpoint.md 5.2). The counts are not compared after
// the ticks: the two worlds are different economies (at tick 20 they hold 1 201 and
// 1 202 agents), so only the structure that no tick can change is.
//
// Arithmetic: 200 households per role at scale 1 and 8 000 at scale 40 are both
// 200 agents per role, 1 200 in all, of weight 1 and of weight 40.
//
// Requirement: NFR-SCL-004, NFR-CHK-002, FR-CHK-005
//
// Fix: plan 2-4. After a reduction by the failure ladder an agent kept its last
// unfed count above its new weight, and a checkpoint of that world was refused as
// corrupt ("unfed 36 is above 32"); a reduction now keeps the count within the weight.
func TestScaleCheckpoint_SizeAndRoundTripDoNotDependOnTheWeights(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfgs := map[int64]economysim.Config{1: ckptConfigOf(t, 1), 40: ckptConfigOf(t, 40)}
	worlds := map[int64]*economysim.World{}
	for scale, cfg := range cfgs {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("NewWorld(scale %d) error = %v, want nil", scale, err)
		}
		worlds[scale] = w
	}
	if a, b := shapeOf(t, worlds[1]), shapeOf(t, worlds[40]); !a.sameState(b) {
		t.Errorf("checkpoint shape at tick 0, weight 1 = %+v, weight 40 = %+v, want equal", a, b)
	}
	for scale, w := range worlds {
		stepN(t, scale, w, 20)
		checkRoundTrip(t, scale, cfgs[scale], w)
	}
	if a, b := shapeOf(t, worlds[1]), shapeOf(t, worlds[40]); !slices.Equal(a.tags, b.tags) {
		t.Errorf("section tags at tick 20, weight 1 = %v, weight 40 = %v, want equal", a.tags, b.tags)
	}
}

// sameState reports whether two shapes have the same sections and the same counts.
func (s ckptShape) sameState(o ckptShape) bool {
	return slices.Equal(s.tags, o.tags) && s.state == o.state && s.lots == o.lots
}

// stepN steps w n times.
func stepN(t *testing.T, scale int64, w *economysim.World, n int) {
	t.Helper()
	for i := range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d (scale %d) error = %v, want nil", i, scale, err)
		}
	}
}

// ckptConfigOf is `one` with 200 households per role times scale, at that scale and
// floor 1: 200 agents per role of weight scale.
func ckptConfigOf(t *testing.T, scale int64) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, scalePacks[0].layers...)
	cfg.World.Scale, cfg.World.MinAgents = scale, 1
	for i := range cfg.Groups {
		cfg.Groups[i].Households *= scale
	}
	return cfg
}

// checkRoundTrip restores w's checkpoint over cfg and asserts that checkpointing the
// restored world gives the same bytes.
func checkRoundTrip(t *testing.T, scale int64, cfg economysim.Config, w *economysim.World) {
	t.Helper()
	ck := mustCheckpoint(t, w)
	r, err := economysim.RestoreWorld(cfg, ck)
	if err != nil {
		t.Fatalf("RestoreWorld(weight %d) error = %v, want nil", scale, err)
	}
	if got, want := mustMarshal(t, mustCheckpoint(t, r)), mustMarshal(t, ck); !bytes.Equal(got, want) {
		t.Errorf("checkpoint of the restored world of weight %d differs from the one it was restored from (%d and %d bytes)", scale, len(got), len(want))
	}
}

// TestFork_OneScaled runs the fork-at-N of plan 2-2 on the pack one at world.scale 4
// and floor 10 (cooks 136 households are 34 agents of weight 4, the line gatherers
// of 30 are 10 agents of weight 3), at N in {0, 100, 150}: the original and two
// restores give the same events, snapshot, audit and checkpoint bytes for 30 ticks,
// so a cohort of weight above 1 survives a checkpoint.
//
// Requirement: NFR-SCL-004, FR-CHK-050, FR-CHK-052
func TestFork_OneScaled(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	cfg.World.Scale, cfg.World.MinAgents = 4, 10
	for _, n := range []int{0, 100, 150} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{Name: "one scale 4", Cfg: cfg, Seed: 1, N: n, M: forkM})
		})
	}
}
