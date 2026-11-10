package experiment_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/internal/design"
)

// FuzzOverlayThroughTheLoader verifies that any path and value, turned into the
// overlay document of a change and layered over a pack, never panic the loader
// and give the same outcome twice: the loader accepts the document or rejects
// it with an error, never both and never differently on a second call. A path
// that is not of the syntax makes no document at all.
//
// Requirement: NFR-EXP-005
func FuzzOverlayThroughTheLoader(f *testing.F) {
	base := onePack(f)
	f.Add("world.belief_step", int64(150))
	f.Add("world.cost_flor", int64(1))
	f.Add("region.north", int64(-1))
	f.Add("a.b.c.d.e.f.g.h", int64(1<<62))
	f.Add("World.x", int64(0))
	f.Add("", int64(0))
	f.Fuzz(func(t *testing.T, path string, value int64) {
		doc, err := design.Overlay(path, value)
		if err != nil {
			if design.ValidPath(path) || !errors.Is(err, design.ErrPath) {
				t.Fatalf("Overlay(%q, %d) error = %v for a path with ValidPath %v", path, value, err, design.ValidPath(path))
			}
			return
		}
		docs := append(append([][]byte{}, base...), doc)
		_, e1 := config.Parse(docs...)
		_, e2 := config.Parse(docs...)
		if (e1 == nil) != (e2 == nil) {
			t.Fatalf("Parse of %q gave %v then %v", doc, e1, e2)
		}
	})
}
