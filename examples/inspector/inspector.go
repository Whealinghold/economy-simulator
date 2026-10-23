package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// freezeTOML turns off role change and the death rung, so the households of
// each role stay as the pack gives them: it isolates the markets from the
// spawner.
const freezeTOML = "[world]\nentry_share = 0\nexit_share = 0\n"

// list is a repeatable string flag.
type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

// options are the parsed flags.
type options struct {
	chains, pack              string
	layers                    string
	cold, freeze, asJSON, tui bool
	extraFiles, inline        list
	seed                      uint64
	ticks, every, win         int64
	show, events, trace       string
}

// run is the whole program, with its streams injected so a test can call it.
// It returns the exit code.
func run(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("inspector", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.chains, "chains", "testdata/chains", "directory of the pack's TOML layers")
	fs.StringVar(&o.pack, "pack", "one", "pack name: loads base, the pack (with lines for five) and warm-<pack>")
	fs.StringVar(&o.layers, "layers", "", "comma-separated layer files, replacing the layers the pack name gives")
	fs.BoolVar(&o.cold, "cold", false, "leave out the warm-start layer")
	fs.BoolVar(&o.freeze, "freeze", false, "no role change and no deaths (entry_share = 0, exit_share = 0)")
	fs.Var(&o.extraFiles, "layer", "extra TOML file layered on top, may repeat")
	fs.Var(&o.inline, "toml", "extra inline TOML layered on top, may repeat, e.g. '[world]\\nentry_share = 0'")
	fs.Uint64Var(&o.seed, "seed", 1, "random seed")
	fs.Int64Var(&o.ticks, "ticks", 600, "ticks to run")
	fs.Int64Var(&o.every, "every", 100, "print a snapshot every this many ticks")
	fs.Int64Var(&o.win, "win", 100, "event window in ticks")
	fs.StringVar(&o.show, "show", "roles,markets,money,events", "any of roles, markets, money, events")
	fs.StringVar(&o.events, "events", "", "comma-separated key prefixes of the window counters to print (default all), e.g. dead,move,short")
	fs.StringVar(&o.trace, "trace", "", "comma-separated commodities whose clearing is printed at each snapshot")
	fs.BoolVar(&o.tui, "tui", false, "open the interactive terminal UI instead of printing a report")
	fs.BoolVar(&o.asJSON, "json", false, "print one JSON object per line instead of text")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if o.every < 1 || o.win < 1 || o.ticks < 0 {
		fmt.Fprintln(stderr, "inspector: -every and -win must be at least 1 and -ticks at least 0")
		return 2
	}
	exec := execute
	if o.tui {
		exec = func(o options, _ io.Writer) error { return runTUI(o) }
	}
	if err := exec(o, stdout); err != nil {
		fmt.Fprintln(stderr, "inspector:", err)
		return 1
	}
	return 0
}

// layerNames returns the files of the pack in layering order.
func (o options) layerNames() []string {
	if o.layers != "" {
		return strings.Split(o.layers, ",")
	}
	names := []string{"base.toml"}
	if o.pack == "five" {
		names = append(names, "lines.toml")
	}
	names = append(names, o.pack+".toml")
	if !o.cold {
		names = append(names, "warm-"+o.pack+".toml")
	}
	return names
}

// load reads and merges the layers: the pack's, then -layer files, then the
// inline documents, then the freeze.
func (o options) load() (economysim.Config, error) {
	var docs [][]byte
	read := func(name string) error {
		p := name
		if !strings.ContainsRune(name, filepath.Separator) {
			p = filepath.Join(o.chains, name)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		docs = append(docs, b)
		return nil
	}
	for _, n := range append(o.layerNames(), o.extraFiles...) {
		if err := read(n); err != nil {
			return economysim.Config{}, err
		}
	}
	for _, s := range o.inline {
		docs = append(docs, []byte(strings.ReplaceAll(s, `\n`, "\n")))
	}
	if o.freeze {
		docs = append(docs, []byte(freezeTOML))
	}
	return config.Parse(docs...)
}

// execute runs the world and prints the report.
func execute(o options, out io.Writer) error {
	cfg, err := o.load()
	if err != nil {
		return err
	}
	tr := newTracker()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(o.seed), economysim.WithObserver(tr))
	if err != nil {
		return err
	}
	r := newReporter(out, o, tr)
	for t := int64(0); t <= o.ticks; t++ {
		if t%o.every == 0 || t == o.ticks {
			s, err := w.Snapshot()
			if err != nil {
				return err
			}
			r.snapshot(s)
		}
		if t%o.win == 0 && t > 0 {
			r.window(t)
		}
		if t == o.ticks {
			break
		}
		if _, err := w.Run(context.Background(), 1); err != nil {
			return fmt.Errorf("tick %d: %w", t, err)
		}
	}
	return r.err
}
