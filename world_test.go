package economysim_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// shellWorld builds a world of minimalConfig, which has one region, one role
// and ten households and needs no agent: the shell of Phase 2 steps it with
// nothing in the straight line but the empty P0.2 (A4 of plan-phase-1-11.md).
func shellWorld(t *testing.T, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(minimalConfig(), opts...)
	if err != nil {
		t.Fatalf("NewWorld(minimal) error = %v, want nil", err)
	}
	return w
}

// nopClearing is a consumer-style Clearing with no trade, to register by name.
type nopClearing struct{}

func (nopClearing) Clear(_, _ []economysim.Order, _ *rand.Rand) []economysim.Trade { return nil }

// render names the events of the lifecycle in a compact text, so a table can
// state the expected sequence in one line. An event of another type, such as the
// construction bucket's BoundaryFlow or the aggregates of the production phase,
// is left out: the lifecycle tests are about the bracketing of a step and a run,
// and the other events are tested where they are built.
func render(events []economysim.Event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		switch e := e.(type) {
		case economysim.RunStarted:
			out = append(out, fmt.Sprintf("RunStarted ticks=%d", e.Ticks))
		case economysim.RunEnded:
			out = append(out, fmt.Sprintf("RunEnded completed=%d cancelled=%t failed=%t", e.Completed, e.Cancelled, e.Failed))
		case economysim.TickStarted:
			out = append(out, fmt.Sprintf("TickStarted %d", e.Tick))
		case economysim.TickEnded:
			out = append(out, fmt.Sprintf("TickEnded %d failed=%t", e.Tick, e.Failed))
		}
	}
	return out
}

// prefixed renders one event of the lifecycle with a prefix, and nothing for any
// other event.
func prefixed(prefix string, e economysim.Event) []string {
	r := render([]economysim.Event{e})
	if len(r) == 0 {
		return nil
	}
	return []string{prefix + r[0]}
}

// recoverPanic runs f and returns the value it panicked with, or nil.
func recoverPanic(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

// panicOnTickStarted is an Observer that panics when a step begins.
func panicOnTickStarted() economysim.Observer {
	return economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.TickStarted); ok {
			panic("observer panic")
		}
	})
}

// TestNewWorld_OptionsAndNames verifies the options half of NewWorld (the
// options apply in order and a bad one fails the call with no world; the name
// checks run after Validate and name the field): a nil logger and observer are
// the default, WithTargetSpread outside 0 to 9 999 is ErrOutOfBounds, an
// unregistered clearing, strategy or spawner is a *ValidationError at the path
// of its field, a name registered twice is ErrDuplicateID, an invalid name is
// ErrInvalidID and a nil extension is ErrNilExtension.
//
// Requirement: FR-WLD-001, FR-WLD-002, FR-WLD-003, FR-CFG-038
func TestNewWorld_OptionsAndNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mutate   func(c *economysim.Config)
		opts     []economysim.Option
		wantErr  error  // nil means NewWorld succeeds
		wantPath string // the path of the *ValidationError, "" if none
	}{
		{name: "no option"},
		{name: "nil logger is the default", opts: []economysim.Option{economysim.WithLogger(nil)}},
		{name: "nil observer is the default", opts: []economysim.Option{economysim.WithObserver(nil)}},
		{name: "nil option is skipped", opts: []economysim.Option{nil}},
		{name: "target spread at 0", opts: []economysim.Option{economysim.WithTargetSpread(0)}},
		{name: "target spread at the role bound 3333", opts: []economysim.Option{economysim.WithTargetSpread(3_333)}},
		// 9 999 is the option's bound; the role's own bound is 3 333, which the
		// validator applies to the overridden configuration.
		{name: "target spread over the role bound", opts: []economysim.Option{economysim.WithTargetSpread(9_999)}, wantErr: economysim.ErrOutOfBounds, wantPath: "roles.farmer.target_spread"},
		{name: "target spread at 10000", opts: []economysim.Option{economysim.WithTargetSpread(10_000)}, wantErr: economysim.ErrOutOfBounds},
		{name: "target spread below 0", opts: []economysim.Option{economysim.WithTargetSpread(-1)}, wantErr: economysim.ErrOutOfBounds},
		{name: "a later bad option fails after a good one", opts: []economysim.Option{economysim.WithSeed(1), economysim.WithTargetSpread(-5)}, wantErr: economysim.ErrOutOfBounds},
		{
			name: "unregistered clearing", wantErr: economysim.ErrUnknownClearing, wantPath: "world.clearing",
			mutate: func(c *economysim.Config) { c.World.Clearing = "mine" },
		},
		{
			name: "registered clearing", mutate: func(c *economysim.Config) { c.World.Clearing = "mine" },
			opts: []economysim.Option{economysim.WithClearing("mine", nopClearing{})},
		},
		{name: "clearing registered twice", opts: []economysim.Option{economysim.WithClearing("mine", nopClearing{}), economysim.WithClearing("mine", nopClearing{})}, wantErr: economysim.ErrDuplicateID},
		{name: "clearing named like the built-in", opts: []economysim.Option{economysim.WithClearing("average", nopClearing{})}, wantErr: economysim.ErrDuplicateID},
		{name: "invalid clearing name", opts: []economysim.Option{economysim.WithClearing("Not An ID", nopClearing{})}, wantErr: economysim.ErrInvalidID},
		{name: "nil clearing", opts: []economysim.Option{economysim.WithClearing("mine", nil)}, wantErr: economysim.ErrNilExtension},
		{
			name: "role naming an unknown strategy", wantErr: economysim.ErrUnknownStrategy, wantPath: "roles.farmer.strategy",
			mutate: func(c *economysim.Config) { c.Roles[0].Strategy = "ghost" },
		},
		{
			name: "role naming a registered strategy", mutate: func(c *economysim.Config) { c.Roles[0].Strategy = "mine" },
			opts: []economysim.Option{economysim.WithStrategy("mine", economysim.DefaultStrategy{})},
		},
		{
			name: "role naming the directed built-in", mutate: func(c *economysim.Config) { c.Roles[0].Strategy = "directed" },
		},
		{
			name: "unknown spawner", wantErr: economysim.ErrUnknownSpawner, wantPath: "world.spawner",
			mutate: func(c *economysim.Config) { c.World.Spawner = "mine" },
		},
		{
			name: "registered spawner", mutate: func(c *economysim.Config) { c.World.Spawner = "mine" },
			opts: []economysim.Option{economysim.WithSpawner("mine", economysim.DefaultSpawner{})},
		},
		{
			name: "validation comes before the name checks", wantErr: economysim.ErrOutOfBounds, wantPath: "world.scale",
			mutate: func(c *economysim.Config) { c.World.Clearing = "mine"; c.World.Scale = -1 },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := minimalConfig()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			w, err := economysim.NewWorld(cfg, tt.opts...)
			if tt.wantErr == nil {
				if err != nil || w == nil {
					t.Fatalf("NewWorld() = (%v, %v), want a world and nil", w, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) || w != nil {
				t.Fatalf("NewWorld() = (%v, %v), want (nil, an error that wraps %v)", w, err, tt.wantErr)
			}
			if tt.wantPath == "" {
				return
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("NewWorld() error = %v, want a *ValidationError", err)
			}
			if got := strings.Join(ve.Path, "."); got != tt.wantPath {
				t.Errorf("ValidationError.Path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

// TestNewWorld_ObserversAreAdditiveAndOrdered verifies that two WithObserver
// options both receive every event, A before B for each event, in the order the
// options were given, and that a world with no observer needs none.
//
// Requirement: FR-WLD-003, FR-OBS-090
func TestNewWorld_ObserversAreAdditiveAndOrdered(t *testing.T) {
	t.Parallel()
	var order []string
	a := economysim.ObserverFunc(func(e economysim.Event) { order = append(order, prefixed("A ", e)...) })
	b := economysim.ObserverFunc(func(e economysim.Event) { order = append(order, prefixed("B ", e)...) })
	w := shellWorld(t, economysim.WithObserver(a), economysim.WithObserver(nil), economysim.WithObserver(b))
	order = nil // the construction events are Phase 3's tests
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	// One step is two events, each delivered to A and then B.
	want := []string{"A TickStarted 0", "B TickStarted 0", "A TickEnded 0 failed=false", "B TickEnded 0 failed=false"}
	if !slices.Equal(order, want) {
		t.Errorf("deliveries = %q, want %q", order, want)
	}
}

// optionFuncs lists the exported With functions of option.go by a parse of the
// file, in source order.
func optionFuncs(t *testing.T) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "option.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse option.go: %v", err)
	}
	var names []string
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && strings.HasPrefix(fd.Name.Name, "With") && fd.Name.IsExported() {
			names = append(names, fd.Name.Name)
		}
	}
	return names
}

// TestNewWorld_HashAndOptions verifies, for each option of option.go, whether
// it moves ConfigHash: WithSeed, WithLogger, WithObserver and WithAudit leave
// it equal to HashConfig of the same configuration; WithTargetSpread changes it,
// because the override is a datum of the effective configuration; a registered
// name changes it only through the configuration that names it. The table is
// checked against a parse of option.go, so a new option has to be classified.
//
// Requirement: FR-WLD-020, FR-WLD-021
//
// Guard: a result-changing option that is neither a datum of the effective
// configuration nor a registered name would let a checkpoint restore into a
// world that behaves differently (spec-world.md 4.4).
func TestNewWorld_HashAndOptions(t *testing.T) {
	t.Parallel()
	base := minimalConfig()
	plain := economysim.HashConfig(base)
	named := minimalConfig()
	named.World.Clearing = "mine"
	tests := []struct {
		option string
		cfg    economysim.Config
		opts   []economysim.Option
		moves  bool // whether ConfigHash differs from HashConfig(base)
	}{
		{"WithSeed", base, []economysim.Option{economysim.WithSeed(99)}, false},
		{"WithLogger", base, []economysim.Option{economysim.WithLogger(slog.New(slog.DiscardHandler))}, false},
		{"WithObserver", base, []economysim.Option{economysim.WithObserver(&recorder{})}, false},
		{"WithAudit", base, []economysim.Option{economysim.WithAudit()}, false},
		{"WithTargetSpread", base, []economysim.Option{economysim.WithTargetSpread(1)}, true},
		{"WithClearing", base, []economysim.Option{economysim.WithClearing("mine", nopClearing{})}, false},
		{"WithClearing", named, []economysim.Option{economysim.WithClearing("mine", nopClearing{})}, true},
		{"WithStrategy", base, []economysim.Option{economysim.WithStrategy("mine", economysim.DefaultStrategy{})}, false},
		{"WithSpawner", base, []economysim.Option{economysim.WithSpawner("mine", economysim.DefaultSpawner{})}, false},
	}
	var covered []string
	for _, tt := range tests {
		covered = append(covered, tt.option)
		w, err := economysim.NewWorld(tt.cfg, tt.opts...)
		if err != nil {
			t.Errorf("NewWorld(%s) error = %v, want nil", tt.option, err)
			continue
		}
		if got := w.ConfigHash() != plain; got != tt.moves {
			t.Errorf("%s: ConfigHash differs from the default = %t, want %t", tt.option, got, tt.moves)
		}
		if !tt.moves && tt.cfg.World.Clearing == base.World.Clearing && w.ConfigHash() != economysim.HashConfig(tt.cfg) {
			t.Errorf("%s: ConfigHash = %x, want HashConfig of the same configuration %x", tt.option, w.ConfigHash(), economysim.HashConfig(tt.cfg))
		}
	}
	// WithTargetSpread: the hash is that of the configuration with every role's
	// spread overridden, as HashConfig sees it.
	over := minimalConfig()
	over.Roles[0].TargetSpread = 1
	w := shellWorld(t, economysim.WithTargetSpread(1))
	if w.ConfigHash() != economysim.HashConfig(over) {
		t.Errorf("WithTargetSpread(1): ConfigHash = %x, want HashConfig of the overridden configuration %x", w.ConfigHash(), economysim.HashConfig(over))
	}
	if got := w.Config().Roles[0].TargetSpread; got != 1 {
		t.Errorf("WithTargetSpread(1): Config().Roles[0].TargetSpread = %d, want 1", got)
	}
	for _, name := range optionFuncs(t) {
		if !slices.Contains(covered, name) {
			t.Errorf("option %s of option.go has no row in the table: classify it as a datum, a registered name or neither", name)
		}
	}
}

// TestWorld_ConfigIsACopy verifies that NewWorld keeps no part of the caller's
// configuration and that Config returns a deep copy: changing either changes
// nothing in the world.
//
// Requirement: FR-WLD-002, FR-WLD-110
func TestWorld_ConfigIsACopy(t *testing.T) {
	t.Parallel()
	cfg := minimalConfig()
	w, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	before := w.ConfigHash()
	cfg.World.Upkeep[0].Quantity = 7
	cfg.Roles[0].ID = "changed"
	got := w.Config()
	got.Commodities[0].ID = "changed"
	got.World.Upkeep[0].Quantity = 9
	if h := economysim.HashConfig(w.Config()); h != before || w.Config().Commodities[0].ID != "grain" || w.Config().World.Upkeep[0].Quantity != 1 {
		t.Errorf("a change to the caller's or the returned copy reached the world: hash %x, want %x", h, before)
	}
}

// TestWorld_GuardsInStep verifies that an observer that calls Step, Run or
// Audit from inside a step gets ErrInStep and changes nothing, that a refused
// inner call does not clear the outer step's flag (a second inner call in the
// same step is refused too), and that the readers still work from inside.
//
// Requirement: FR-WLD-007, FR-WLD-110
func TestWorld_GuardsInStep(t *testing.T) {
	t.Parallel()
	for _, at := range []string{"TickStarted", "TickEnded"} {
		t.Run(at, func(t *testing.T) {
			t.Parallel()
			var w *economysim.World
			var inner []error
			rec := &recorder{}
			probe := economysim.ObserverFunc(func(e economysim.Event) {
				if r := render([]economysim.Event{e}); len(r) == 0 || !strings.HasPrefix(r[0], at) {
					return
				}
				_, runErr := w.Run(context.Background(), 1)
				inner = append(inner, w.Step(), w.Step(), runErr, w.Audit())
				if w.Tick() != 0 || w.Err() != nil || w.ConfigHash() == [32]byte{} || w.Config().Roles == nil {
					t.Errorf("a reader from inside a step gave Tick %d, Err %v, want 0 and nil and the configuration", w.Tick(), w.Err())
				}
			})
			w = shellWorld(t, economysim.WithObserver(rec), economysim.WithObserver(probe))
			rec.forget()
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
			if len(inner) != 4 {
				t.Fatalf("the observer made %d inner calls, want 4", len(inner))
			}
			for i, err := range inner {
				if !errors.Is(err, economysim.ErrInStep) {
					t.Errorf("inner call %d = %v, want ErrInStep", i, err)
				}
			}
			want := []string{"TickStarted 0", "TickEnded 0 failed=false"}
			if got := render(rec.events); !slices.Equal(got, want) {
				t.Errorf("events = %q, want %q: an inner call changed nothing", got, want)
			}
			if w.Tick() != 1 || w.Audit() != nil {
				t.Errorf("after the step Tick = %d and Audit = %v, want 1 and nil", w.Tick(), w.Audit())
			}
		})
	}
}

// TestWorld_PanicFailsTheWorld verifies that an observer that panics in a step
// leaves the world failed, that the panic propagates, that no TickEnded is
// emitted, that the failure is kept as it was on every later call (the same
// error value, not a new one), and that the readers still work.
//
// Requirement: FR-WLD-006, FR-WLD-012, FR-OBS-013, FR-OBS-024
func TestWorld_PanicFailsTheWorld(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := shellWorld(t, economysim.WithObserver(rec), economysim.WithObserver(panicOnTickStarted()))
	rec.forget()
	if v := recoverPanic(func() { _ = w.Step() }); v != "observer panic" {
		t.Fatalf("Step() panicked with %v, want the observer's panic", v)
	}
	if got, want := render(rec.events), []string{"TickStarted 0"}; !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q: a panic emits no TickEnded", got, want)
	}
	cause := w.Err()
	if cause == nil {
		t.Fatalf("Err() = nil after a panic, want the failure")
	}
	for call := 1; call <= 3; call++ {
		_, runErr := w.Run(context.Background(), 1)
		for i, err := range []error{w.Step(), runErr, w.Audit()} {
			if !errors.Is(err, economysim.ErrWorldFailed) || !errors.Is(err, cause) {
				t.Errorf("call %d, %s() = %v, want ErrWorldFailed wrapping %v", call, []string{"Step", "Run", "Audit"}[i], err, cause)
			}
		}
		if w.Err() != cause {
			t.Errorf("call %d: Err() = %v, want the original failure %v", call, w.Err(), cause)
		}
	}
	if w.Tick() != 0 || w.ConfigHash() == [32]byte{} || len(w.Config().Roles) != 1 {
		t.Errorf("readers on a failed world: Tick %d, hash %x, roles %d, want 0, set, 1", w.Tick(), w.ConfigHash(), len(w.Config().Roles))
	}
	if got := len(render(rec.events)); got != 1 {
		t.Errorf("a refused call emitted lifecycle events: %d in all, want 1", got)
	}
}

// TestWorld_Run verifies the table of spec-world.md 3.4.2: a Run that completes,
// one cancelled before tick k by an observer on the k-th TickEnded, one with a
// context done before the first tick, ticks 0, ticks negative (ErrOutOfBounds,
// no event), and a Run on a world that is in a step (ErrInStep, no event).
//
// Arithmetic: a Run of n completed ticks emits RunStarted, n pairs of
// TickStarted and TickEnded, and RunEnded, which is 2n + 2 events; the world
// ends at tick n.
//
// Requirement: FR-WLD-008, FR-WLD-009, FR-WLD-012
func TestWorld_Run(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		ticks         int64
		cancelOnEnd   int64 // cancel the context on the TickEnded of this tick; -1 for none
		preCancel     bool
		wantCompleted int64
		wantErr       error
		wantEvents    []string
	}{
		{name: "completes", ticks: 2, cancelOnEnd: -1, wantCompleted: 2, wantEvents: []string{
			"RunStarted ticks=2", "TickStarted 0", "TickEnded 0 failed=false", "TickStarted 1", "TickEnded 1 failed=false",
			"RunEnded completed=2 cancelled=false failed=false"}},
		{name: "cancelled before tick 2", ticks: 5, cancelOnEnd: 1, wantCompleted: 2, wantErr: context.Canceled, wantEvents: []string{
			"RunStarted ticks=5", "TickStarted 0", "TickEnded 0 failed=false", "TickStarted 1", "TickEnded 1 failed=false",
			"RunEnded completed=2 cancelled=true failed=false"}},
		{name: "cancelled before the first tick", ticks: 3, cancelOnEnd: -1, preCancel: true, wantErr: context.Canceled, wantEvents: []string{
			"RunStarted ticks=3", "RunEnded completed=0 cancelled=true failed=false"}},
		{name: "zero ticks", ticks: 0, cancelOnEnd: -1, wantEvents: []string{
			"RunStarted ticks=0", "RunEnded completed=0 cancelled=false failed=false"}},
		{name: "negative ticks", ticks: -1, cancelOnEnd: -1, wantErr: economysim.ErrOutOfBounds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.preCancel {
				cancel()
			}
			rec := &recorder{}
			stop := economysim.ObserverFunc(func(e economysim.Event) {
				if te, ok := e.(economysim.TickEnded); ok && te.Tick == tt.cancelOnEnd {
					cancel()
				}
			})
			w := shellWorld(t, economysim.WithObserver(rec), economysim.WithObserver(stop))
			rec.forget()
			got, err := w.Run(ctx, tt.ticks)
			if got != tt.wantCompleted || !errors.Is(err, tt.wantErr) {
				t.Errorf("Run(%d) = (%d, %v), want (%d, wrapping %v)", tt.ticks, got, err, tt.wantCompleted, tt.wantErr)
			}
			if errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), fmt.Sprintf("after %d ticks", tt.wantCompleted)) {
				t.Errorf("Run(%d) error = %q, want it to name the %d ticks completed", tt.ticks, err, tt.wantCompleted)
			}
			if events := render(rec.events); !slices.Equal(events, tt.wantEvents) {
				t.Errorf("Run(%d) events = %q, want %q", tt.ticks, events, tt.wantEvents)
			}
			if w.Tick() != tt.wantCompleted || w.Err() != nil {
				t.Errorf("after Run(%d) Tick = %d and Err = %v, want %d and nil: a cancelled Run leaves a usable world", tt.ticks, w.Tick(), w.Err(), tt.wantCompleted)
			}
			if tt.wantErr != nil && w.Step() != nil {
				t.Errorf("Step() after Run(%d) = error, want a usable world", tt.ticks)
			}
		})
	}

	t.Run("from inside a step", func(t *testing.T) {
		t.Parallel()
		var w *economysim.World
		var inner error
		rec := &recorder{}
		probe := economysim.ObserverFunc(func(e economysim.Event) {
			if _, ok := e.(economysim.TickStarted); ok {
				_, inner = w.Run(context.Background(), 2)
			}
		})
		w = shellWorld(t, economysim.WithObserver(rec), economysim.WithObserver(probe))
		rec.forget()
		if err := w.Step(); err != nil || !errors.Is(inner, economysim.ErrInStep) {
			t.Fatalf("Step() = %v and the inner Run = %v, want nil and ErrInStep", err, inner)
		}
		if want := []string{"TickStarted 0", "TickEnded 0 failed=false"}; !slices.Equal(render(rec.events), want) {
			t.Errorf("events = %q, want %q: the refused Run emits no RunStarted", render(rec.events), want)
		}
	})
}

// TestWorld_RunEndingInAPanic verifies that a Run whose step panics emits
// RunStarted and the step's TickStarted and no RunEnded, writes "run started"
// and no "run ended", and leaves the world failed.
//
// Requirement: FR-WLD-012, FR-OBS-013, FR-OBS-024
func TestWorld_RunEndingInAPanic(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	h := newCaptureHandler()
	w := shellWorld(t, economysim.WithObserver(rec), economysim.WithObserver(panicOnTickStarted()), economysim.WithLogger(slog.New(h)))
	rec.forget()
	h.forget()
	if v := recoverPanic(func() { _, _ = w.Run(context.Background(), 3) }); v == nil {
		t.Fatalf("Run(3) did not panic, want the observer's panic to propagate")
	}
	if got, want := render(rec.events), []string{"RunStarted ticks=3", "TickStarted 0"}; !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
	lines := h.records()
	if len(lines) != 1 || lines[0].msg != "run started" {
		t.Errorf("log lines = %v, want exactly \"run started\": a panic ends no run", lines)
	}
	if w.Err() == nil {
		t.Errorf("Err() = nil after a panic in a Run, want the failure")
	}
}

// TestWorld_RunLogs verifies that Run writes "run started" (ticks) and "run
// ended" (ticks completed, reason) at Info with a logger and no observer, that
// reason is completed or cancelled, that a refused Run writes nothing, and that
// a Step writes no line at all (no Info in the tick).
//
// Requirement: FR-OBS-012, FR-OBS-013
func TestWorld_RunLogs(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name      string
		ctx       context.Context
		ticks     int64
		wantTicks int64
		wantWhy   string // "" when no line is written
	}{
		{"completed", context.Background(), 3, 3, "completed"},
		{"cancelled", ctx, 3, 0, "cancelled"},
		{"negative ticks", context.Background(), -1, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newCaptureHandler()
			w := shellWorld(t, economysim.WithLogger(slog.New(h)))
			h.forget()
			_, _ = w.Run(tt.ctx, tt.ticks)
			// The Debug lines of the positions (a market cleared) are not Run's.
			lines := infoAndAbove(h.records())
			if tt.wantWhy == "" {
				if len(lines) != 0 {
					t.Errorf("Run(%d) wrote %v, want no line", tt.ticks, lines)
				}
				return
			}
			if len(lines) != 2 || lines[0].msg != "run started" || lines[1].msg != "run ended" {
				t.Fatalf("Run(%d) lines = %v, want \"run started\" then \"run ended\"", tt.ticks, lines)
			}
			if v, _ := lines[0].attr("ticks"); v.Int64() != tt.ticks {
				t.Errorf("run started ticks = %v, want %d", v, tt.ticks)
			}
			if v, _ := lines[1].attr("ticks"); v.Int64() != tt.wantTicks {
				t.Errorf("run ended ticks = %v, want %d completed", v, tt.wantTicks)
			}
			if v, _ := lines[1].attr("reason"); v.String() != tt.wantWhy {
				t.Errorf("run ended reason = %v, want %q", v, tt.wantWhy)
			}
			for _, l := range lines {
				if l.level != slog.LevelInfo {
					t.Errorf("line %q at level %v, want Info", l.msg, l.level)
				}
			}
		})
	}
	t.Run("a step writes nothing", func(t *testing.T) {
		t.Parallel()
		h := newCaptureHandler()
		w := shellWorld(t, economysim.WithLogger(slog.New(h)))
		h.forget()
		for range 3 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
		if lines := infoAndAbove(h.records()); len(lines) != 0 {
			t.Errorf("3 steps wrote %v at Info or above, want no line: no Info in the tick", lines)
		}
	})
}

// infoAndAbove returns the lines at Info or above: the lifecycle lines and the
// Warn and Error lines. The Debug lines of the positions are per market and per
// order and have tests of their own.
func infoAndAbove(lines []logLine) []logLine {
	var out []logLine
	for _, l := range lines {
		if l.level >= slog.LevelInfo {
			out = append(out, l)
		}
	}
	return out
}

// TestWorld_TwoWorldsShareNothing verifies that two worlds built from one
// configuration value and stepped on two goroutines share nothing: under -race
// a shared slice or table would be reported.
//
// Requirement: FR-WLD-011, NFR-WLD-003
func TestWorld_TwoWorldsShareNothing(t *testing.T) {
	t.Parallel()
	cfg := minimalConfig()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	ticks := make([]int64, 2)
	for i := range errs {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(uint64(i)))
		if err != nil {
			t.Fatalf("NewWorld() = %v, want nil", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = w.Run(context.Background(), 50)
			ticks[i] = w.Tick()
		}()
	}
	wg.Wait()
	for i := range errs {
		if errs[i] != nil || ticks[i] != 50 {
			t.Errorf("world %d: Run(50) error = %v, Tick = %d, want nil and 50", i, errs[i], ticks[i])
		}
	}
}

// TestWorld_RunLogsAFailedRun verifies that a Run whose step fails writes "run
// started" and "run ended" with reason failed and ticks 0 completed, at Info,
// and that the error is returned and not logged (observability rule 2.2).
//
// Arithmetic: the "faulty clearing" scenario returns an invalid trade list at the
// first clearing, so tick 0 fails and no tick completes.
//
// Requirement: FR-OBS-012, FR-OBS-013
func TestWorld_RunLogsAFailedRun(t *testing.T) {
	t.Parallel()
	var sc scenario
	for _, s := range scenarios() {
		if s.name == "faulty clearing" {
			sc = s
		}
	}
	w, _, h := build(t, sc, who{logger: true})
	h.forget()
	n, err := w.Run(context.Background(), 3)
	if n != 0 || !errors.Is(err, economysim.ErrInvalidTrade) {
		t.Fatalf("Run(3) = (%d, %v), want (0, ErrInvalidTrade)", n, err)
	}
	var lines []logLine
	for _, l := range h.records() {
		if l.level >= slog.LevelInfo {
			lines = append(lines, l)
		}
	}
	if len(lines) != 2 || lines[0].msg != "run started" || lines[1].msg != "run ended" {
		t.Fatalf("lines = %v, want \"run started\" then \"run ended\" and nothing else at Info or above", lines)
	}
	if v, _ := lines[1].attr("reason"); v.String() != "failed" {
		t.Errorf("run ended reason = %v, want \"failed\"", v)
	}
	if v, _ := lines[1].attr("ticks"); v.Int64() != 0 {
		t.Errorf("run ended ticks = %v, want 0", v)
	}
}

// TestWorld_RunOfZeroTicks verifies the boundary of Run's argument: 0 ticks is
// not an error, completes 0 ticks, emits RunStarted and RunEnded with nothing
// between, and leaves the world healthy; -1 is the first refused value.
//
// Requirement: FR-WLD-008, FR-WLD-009
// Guard: a comparison written as "ticks != 0" would refuse the legal empty Run.
func TestWorld_RunOfZeroTicks(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := shellWorld(t, economysim.WithObserver(rec))
	rec.forget()
	n, err := w.Run(context.Background(), 0)
	if n != 0 || err != nil {
		t.Fatalf("Run(0) = (%d, %v), want (0, nil)", n, err)
	}
	if got, want := render(rec.events), []string{"RunStarted ticks=0", "RunEnded completed=0 cancelled=false failed=false"}; !slices.Equal(got, want) {
		t.Errorf("Run(0) events = %q, want %q", got, want)
	}
	if w.Err() != nil || w.Tick() != 0 {
		t.Errorf("after Run(0): Err() = %v, Tick() = %d, want nil and 0", w.Err(), w.Tick())
	}
	if _, err := w.Run(context.Background(), -1); !errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("Run(-1) error = %v, want ErrOutOfBounds", err)
	}
}

// TestWorld_RunRefusedByAGuardCompletesNothing verifies that a Run refused by the
// guard of a failed world returns 0 ticks completed with ErrWorldFailed.
//
// Requirement: FR-WLD-007, FR-WLD-012
func TestWorld_RunRefusedByAGuardCompletesNothing(t *testing.T) {
	t.Parallel()
	var sc scenario
	for _, s := range scenarios() {
		if s.name == "faulty clearing" {
			sc = s
		}
	}
	f, _, _ := build(t, sc, who{})
	if err := f.Step(); !errors.Is(err, economysim.ErrInvalidTrade) {
		t.Fatalf("Step() of the faulty world error = %v, want ErrInvalidTrade", err)
	}
	if n, err := f.Run(context.Background(), 3); n != 0 || !errors.Is(err, economysim.ErrWorldFailed) {
		t.Errorf("Run(3) on a failed world = (%d, %v), want (0, ErrWorldFailed)", n, err)
	}
}

// TestWorld_APanicAfterAFailedStepKeepsTheFirstCause verifies that when a step
// fails and an observer then panics while TickEnded is delivered, the world's
// error is still the step's failure and not the generic "step panicked" one.
//
// Requirement: FR-WLD-006, FR-WLD-012
// Guard: the first cause of a failure is what a consumer debugs from; replacing it
// with the panic notice would hide the invalid trade list.
func TestWorld_APanicAfterAFailedStepKeepsTheFirstCause(t *testing.T) {
	t.Parallel()
	var sc scenario
	for _, s := range scenarios() {
		if s.name == "faulty clearing" {
			sc = s
		}
	}
	panicOnEnd := economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.TickEnded); ok {
			panic("observer panic at TickEnded")
		}
	})
	w, _, _ := build(t, sc, who{}, economysim.WithObserver(panicOnEnd))
	if v := recoverPanic(func() { _ = w.Step() }); v == nil {
		t.Fatalf("Step() did not panic, want the observer's panic to propagate")
	}
	if !errors.Is(w.Err(), economysim.ErrInvalidTrade) {
		t.Errorf("Err() = %v, want it to keep ErrInvalidTrade, the cause of the failed step", w.Err())
	}
	if err := w.Step(); !errors.Is(err, economysim.ErrWorldFailed) {
		t.Errorf("Step() on the failed world error = %v, want ErrWorldFailed", err)
	}
}
