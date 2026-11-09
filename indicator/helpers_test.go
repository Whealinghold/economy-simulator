package indicator_test

import (
	"os"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// loadOne returns the configuration of pack one, warm, the reference chain the
// indicator tests read (testdata/chains), as World.Config returns it.
func loadOne(t *testing.T) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS("../testdata/chains"), "base.toml", "one.toml", "warm-one.toml")
	if err != nil {
		t.Fatalf("LoadFS(base, one, warm-one) error = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(pack one) error = %v, want nil", err)
	}
	return w.Config()
}

// newOne returns a world of pack one, seed 1.
func newOne(t *testing.T) *economysim.World {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS("../testdata/chains"), "base.toml", "one.toml", "warm-one.toml")
	if err != nil {
		t.Fatalf("LoadFS(base, one, warm-one) error = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(pack one) error = %v, want nil", err)
	}
	return w
}
