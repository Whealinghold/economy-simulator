package economysim

import (
	"errors"
	"testing"
)

// TestRestore_SettlementsSlotsAreWired verifies that a restored world with the settlements
// module holds the slots the module wires into the skills and storage modules, so a
// completion held for the class limit stays held and a locked store class stays locked,
// as in the world that was not interrupted.
//
// Arithmetic: the class small has 0 places and the weaver holds 1 household, so the
// headroom is max(0, 0 - 1) = 0 and the trainee is held at its 4 ticks; the barn needs the
// flag market, which the hamlet lacks.
//
// Requirement: FR-SET-021, FR-SET-023, FR-CHK-001
//
// Fix: N-462. The slots were assigned only by construction, so a restored world completed
// a held traineeship and let a locked store class be added.
func TestRestore_SettlementsSlotsAreWired(t *testing.T) {
	t.Parallel()
	t.Run("held completion", func(t *testing.T) {
		t.Parallel()
		w, c := seamWorld(t, 0)
		w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 4, 1)
		r, err := restoreOf(t, w)
		if err != nil {
			t.Fatalf("RestoreWorld() error = %v, want nil", err)
		}
		for name, x := range map[string]*World{"original": w, "restored": r} {
			seamEntry(t, x)
			if !x.skills.isTrainee(c.homeMillers[0]) {
				t.Errorf("%s world: the trainee completed with no headroom, want held", name)
			}
		}
	})
	t.Run("locked store class", func(t *testing.T) {
		t.Parallel()
		w, _ := storeSeamWorld(t)
		step(t, w, 1) // a world that has not stepped has no checkpoint boundary to restore
		ck, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("Checkpoint() error = %v, want nil", err)
		}
		r, err := RestoreWorld(w.Config(), ck, WithStrategy("script", &shipScript{}))
		if err != nil {
			t.Fatalf("RestoreWorld() error = %v, want nil", err)
		}
		for name, x := range map[string]*World{"original": w, "restored": r} {
			if err := x.AddStore(shipHome, 0); !errors.Is(err, ErrStoreLocked) {
				t.Errorf("%s world: AddStore(barn) in a hamlet = %v, want ErrStoreLocked", name, err)
			}
		}
	})
}
