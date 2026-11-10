package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/strategy"
)

// packFS holds the layers: pack/cold is the modules-off chain, pack/modules the
// all-on layers, both renamed to the realm's words by pack/rename.sh (D682).
//
//go:embed pack/cold/*.toml pack/modules/*.toml
var packFS embed.FS

// coldLayers are the modules-off layers, in merge order (lines before five).
var coldLayers = []string{"cold/base.toml", "cold/calendar.toml", "cold/lines.toml", "cold/five.toml"}

// sub returns the embedded pack rooted at pack/, the file system config.LoadFS reads.
func sub() (fs.FS, error) { return fs.Sub(packFS, "pack") }

// moduleLayers returns the all-on layers in merge order: the file names carry a
// two-digit position and fs.Glob returns them sorted.
func moduleLayers(fsys fs.FS) ([]string, error) { return fs.Glob(fsys, "modules/*.toml") }

// Cold returns the modules-off configuration of the realm: the programme the warm
// start is solved from (D684). It has no warm start.
func Cold() (economysim.Config, error) {
	fsys, err := sub()
	if err != nil {
		return economysim.Config{}, fmt.Errorf("open the pack: %w", err)
	}
	cfg, err := config.LoadFS(fsys, coldLayers...)
	if err != nil {
		return economysim.Config{}, fmt.Errorf("load the cold layers: %w", err)
	}
	return cfg, nil
}

// Load returns the realm configuration: the cold layers solved once for the warm
// start (SolveSteadyState with its default options, the call warm-five.toml was
// produced by), then merged with the module layers, with the table set as
// WarmStart.
func Load() (economysim.Config, error) {
	cfg, _, err := LoadTimed()
	return cfg, err
}

// LoadTimed is Load that also returns the time the solve took. The solve is
// outside the tick budget (spec-scale.md 6.1); the caller reports the time. The
// clock is read here, in example code: the library has none.
func LoadTimed() (economysim.Config, time.Duration, error) {
	cold, err := Cold()
	if err != nil {
		return economysim.Config{}, 0, err
	}
	start := time.Now()
	st, err := economysim.SolveSteadyState(context.Background(), cold)
	took := time.Since(start)
	if err != nil {
		return economysim.Config{}, 0, fmt.Errorf("solve the warm start: %w", err)
	}
	fsys, err := sub()
	if err != nil {
		return economysim.Config{}, 0, fmt.Errorf("open the pack: %w", err)
	}
	mods, err := moduleLayers(fsys)
	if err != nil {
		return economysim.Config{}, 0, fmt.Errorf("list the module layers: %w", err)
	}
	cfg, err := config.LoadFS(fsys, append(append([]string{}, coldLayers...), mods...)...)
	if err != nil {
		return economysim.Config{}, 0, fmt.Errorf("load the module layers: %w", err)
	}
	cfg.WarmStart = &st.Table
	if err := cfg.Validate(); err != nil {
		return economysim.Config{}, 0, fmt.Errorf("validate the realm with its warm start: %w", err)
	}
	return cfg, took, nil
}

// RestoreOptions are the options a restore takes, and a new world takes besides
// its seed: the registry of the strategies the pack names (the merchants), the
// observer if there is one, and the audit when asked.
func RestoreOptions(obs economysim.Observer, audit bool) []economysim.Option {
	opts := strategy.Options()
	if obs != nil {
		opts = append(opts, economysim.WithObserver(obs))
	}
	if audit {
		opts = append(opts, economysim.WithAudit())
	}
	return opts
}

// Options are the options of a new world: WithSeed and RestoreOptions.
func Options(seed uint64, obs economysim.Observer, audit bool) []economysim.Option {
	return append([]economysim.Option{economysim.WithSeed(seed)}, RestoreOptions(obs, audit)...)
}
