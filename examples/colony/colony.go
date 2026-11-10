package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"strings"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/denomination"
)

// packFS holds the pack: byte copies of testdata/chains/second/ in the root module,
// guarded there by TestExamples_PackCopiesAgree (D687).
//
//go:embed pack/second.toml pack/warm-second.toml
var packFS embed.FS

// packLayers are the layers of the pack, in merge order.
var packLayers = []string{"pack/second.toml", "pack/warm-second.toml"}

// Load returns the configuration of the colony: the chain and its warm start.
func Load() (economysim.Config, error) {
	cfg, err := config.LoadFS(packFS, packLayers...)
	if err != nil {
		return economysim.Config{}, fmt.Errorf("load the colony pack: %w", err)
	}
	return cfg, nil
}

// Run is the command: it parses args, runs the colony and prints, every -every
// ticks, the tick, the households and the price of each commodity. It returns the
// exit code: 0, 1 for a run that failed and 2 for bad arguments.
func Run(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("colony", flag.ContinueOnError)
	fl.SetOutput(stderr)
	seed := fl.Uint64("seed", 1, "the seed of the world")
	ticks := fl.Int64("ticks", 400, "ticks to run")
	every := fl.Int64("every", 100, "ticks between two lines")
	audit := fl.Bool("audit", false, "audit the books after every step")
	if err := fl.Parse(args); err != nil {
		return 2
	}
	if *ticks < 0 || *every < 1 {
		fmt.Fprintf(stderr, "colony: -ticks %d and -every %d must be at least 0 and 1\n", *ticks, *every)
		return 2
	}
	if err := report(stdout, *seed, *ticks, *every, *audit); err != nil {
		fmt.Fprintf(stderr, "colony: %v\n", err)
		return 1
	}
	return 0
}

// report runs the world and prints a line each window.
func report(out io.Writer, seed uint64, ticks, every int64, audit bool) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	opts := []economysim.Option{economysim.WithSeed(seed)}
	if audit {
		opts = append(opts, economysim.WithAudit())
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		return fmt.Errorf("build the world: %w", err)
	}
	// One denomination: no ladder structure is required (spec-currency.md 4.3).
	f, err := denomination.NewFormatter(denomination.Single("credit", "cr"))
	if err != nil {
		return fmt.Errorf("build the formatter: %w", err)
	}
	if err := line(out, w, f); err != nil {
		return err
	}
	for done := int64(0); done < ticks; {
		n := min(every, ticks-done)
		if _, err := w.Run(context.Background(), n); err != nil {
			return fmt.Errorf("run: %w", err)
		}
		done += n
		if err := line(out, w, f); err != nil {
			return err
		}
	}
	return nil
}

// line prints the tick, the households and the price of each commodity, in
// commodity order. A commodity with no price yet prints as "-".
func line(out io.Writer, w *economysim.World, f *denomination.Formatter) error {
	s, err := w.Snapshot()
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	names := w.Config().Commodities
	var prices []string
	for _, m := range s.Markets {
		p := "-"
		if m.State.Price > 0 {
			p = f.Format(int64(m.State.Price))
		}
		prices = append(prices, names[m.State.Commodity].ID+" "+p)
	}
	_, err = fmt.Fprintf(out, "tick %d households %d prices %s\n", s.Tick, s.Households, strings.Join(prices, ", "))
	return err
}
