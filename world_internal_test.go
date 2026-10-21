package economysim

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// shellConfig is the smallest valid configuration: one raw commodity, one
// region, one role and ten households. The shell of the world needs no agent.
func shellConfig() Config {
	world := DefaultWorldParams()
	world.Upkeep = []Item{{Commodity: "grain", Quantity: 1}}
	grain := DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := DefaultRegion()
	region.ID = "home"
	role := DefaultRole()
	role.ID = "farmer"
	return Config{
		World:       world,
		Calendar:    DefaultCalendar(),
		Commodities: []Commodity{grain},
		Roles:       []Role{role},
		Regions:     []Region{region},
		Groups:      []Group{{Region: "home", Role: "farmer", Households: 10}},
	}
}

// internalShell builds a world of shellConfig.
func internalShell(t *testing.T, opts ...Option) *World {
	t.Helper()
	w, err := NewWorld(shellConfig(), opts...)
	if err != nil {
		t.Fatalf("NewWorld(shell) error = %v, want nil", err)
	}
	return w
}

// moveSomeMoney makes the totals of the current tick non-zero: it ends the
// construction phase, opens an agent account and mints 5 minor units into it.
func moveSomeMoney(t *testing.T, w *World) {
	t.Helper()
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	acct, err := w.ledger.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open(agent) = %v, want nil", err)
	}
	src, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatalf("Boundary(initial, money) not found")
	}
	if err := w.ledger.Transfer(src, acct, 5, ledger.Initial); err != nil {
		t.Fatalf("Transfer() = %v, want nil", err)
	}
	if n := len(w.ledger.AppendFlows(nil, false)); n == 0 {
		t.Fatalf("no flow total after a transfer, want one")
	}
}

// TestNewWorld_SeedLastWins verifies that the last WithSeed given wins, that the
// default seed is 0, and that the seed reaches the stream and the saved state.
//
// Requirement: FR-WLD-001, FR-WLD-003
func TestNewWorld_SeedLastWins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []Option
		want uint64
	}{
		{"no option", nil, 0},
		{"one seed", []Option{WithSeed(7)}, 7},
		{"last seed wins", []Option{WithSeed(1), WithSeed(2), WithSeed(3)}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := internalShell(t, tt.opts...)
			if w.state.seed != tt.want || w.rnd.seed != tt.want {
				t.Errorf("seed = %d in the state and %d in the stream, want %d", w.state.seed, w.rnd.seed, tt.want)
			}
		})
	}
}

// TestEndStep_FailureBracketsTheStep verifies the bracketing of a failed step:
// the world is failed before TickEnded is emitted, TickEnded carries Failed,
// Tick is unchanged and the flow totals are not cleared; a step that ends well
// advances the tick and clears them, and clears the in-step flag.
//
// Requirement: FR-WLD-006, FR-WLD-007, FR-WLD-012
func TestEndStep_FailureBracketsTheStep(t *testing.T) {
	t.Parallel()
	synthetic := errors.New("synthetic failure")
	tests := []struct {
		name       string
		err        error
		wantTick   int64
		wantTotals bool
	}{
		{"failed step", synthetic, 0, true},
		{"successful step", nil, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var w *World
			var failedAtEvent error
			var ended []TickEnded
			w = internalShell(t, WithObserver(ObserverFunc(func(e Event) {
				if te, ok := e.(TickEnded); ok {
					failedAtEvent = w.Err()
					ended = append(ended, te)
				}
			})))
			moveSomeMoney(t, w)
			if err := w.beginStep(); err != nil {
				t.Fatalf("beginStep() = %v, want nil", err)
			}
			if !w.stepping {
				t.Errorf("stepping = false after beginStep, want true")
			}
			if got := w.endStep(tt.err); got != tt.err {
				t.Errorf("endStep(%v) = %v, want the error back", tt.err, got)
			}
			if len(ended) != 1 || ended[0].Failed != (tt.err != nil) || ended[0].Tick != 0 {
				t.Errorf("TickEnded events = %+v, want one for tick 0 with Failed %t", ended, tt.err != nil)
			}
			if failedAtEvent != tt.err {
				t.Errorf("Err() at the TickEnded event = %v, want %v: the world is failed before the event", failedAtEvent, tt.err)
			}
			if w.Tick() != tt.wantTick || w.Err() != tt.err || w.stepping {
				t.Errorf("after endStep Tick = %d, Err = %v, stepping = %t, want %d, %v, false", w.Tick(), w.Err(), w.stepping, tt.wantTick, tt.err)
			}
			if has := len(w.ledger.AppendFlows(nil, false)) > 0; has != tt.wantTotals {
				t.Errorf("flow totals present after endStep = %t, want %t", has, tt.wantTotals)
			}
		})
	}
}

// TestStep_FlagOnEveryPath verifies that the in-step flag is clear after every
// way a step can end: a replay-style beginStep and endStep(nil) with no Step
// around them leaves Audit working, and a panic in an observer, recovered by the
// caller, leaves the world failed with ErrWorldFailed and not ErrInStep.
//
// Requirement: FR-WLD-006, FR-WLD-007
func TestStep_FlagOnEveryPath(t *testing.T) {
	t.Parallel()
	t.Run("replay without Step", func(t *testing.T) {
		t.Parallel()
		w := internalShell(t)
		if err := w.beginStep(); err != nil {
			t.Fatalf("beginStep() = %v, want nil", err)
		}
		if err := w.endStep(nil); err != nil {
			t.Fatalf("endStep(nil) = %v, want nil", err)
		}
		if w.stepping || w.Audit() != nil || w.Tick() != 1 {
			t.Errorf("after a replayed step stepping = %t, Audit = %v, Tick = %d, want false, nil, 1", w.stepping, w.Audit(), w.Tick())
		}
	})
	t.Run("panic in an observer", func(t *testing.T) {
		t.Parallel()
		w := internalShell(t, WithObserver(ObserverFunc(func(e Event) {
			if _, ok := e.(TickStarted); ok {
				panic("observer panic")
			}
		})))
		func() {
			defer func() { _ = recover() }()
			_ = w.Step()
		}()
		if w.stepping {
			t.Errorf("stepping = true after a recovered panic, want false")
		}
		for i, err := range []error{w.Step(), w.Audit()} {
			if !errors.Is(err, ErrWorldFailed) || errors.Is(err, ErrInStep) {
				t.Errorf("%s() after a recovered panic = %v, want ErrWorldFailed and not ErrInStep", []string{"Step", "Audit"}[i], err)
			}
		}
		if got := w.Err(); got == nil || !strings.Contains(got.Error(), "tick 0") || !strings.Contains(got.Error(), "panicked") {
			t.Errorf("Err() = %v, want an error that names tick 0 and the panic", got)
		}
	})
}

// TestStep_RefusedCallKeepsTheCause verifies that a call refused on a failed
// world keeps the real cause: after a step that failed with a synthetic error,
// every later Step returns ErrWorldFailed wrapping that same error, Err returns
// the same value, and the refusal neither sets nor clears the in-step flag.
//
// Requirement: FR-WLD-006
//
// Guard: if a refused call ran the deferred panic path of Step it would replace
// the cause by a "panicked" error and the consumer would lose the real failure.
func TestStep_RefusedCallKeepsTheCause(t *testing.T) {
	t.Parallel()
	w := internalShell(t)
	synthetic := errors.New("synthetic failure")
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() = %v, want nil", err)
	}
	_ = w.endStep(synthetic)
	for call := 1; call <= 3; call++ {
		err := w.Step()
		if !errors.Is(err, ErrWorldFailed) || !errors.Is(err, synthetic) {
			t.Errorf("call %d: Step() = %v, want ErrWorldFailed wrapping the synthetic error", call, err)
		}
		if w.Err() != synthetic || w.stepping {
			t.Errorf("call %d: Err() = %v, stepping = %t, want the synthetic error and false", call, w.Err(), w.stepping)
		}
	}
}

// TestRun_FailedStepEndsTheRun verifies the row "step k + 1 failed" of
// spec-world.md 3.4.2: completed is k, the error wraps the step's, RunEnded has
// Failed and not Cancelled, and the world is failed. No position can fail in the
// shell, so an observer fails the world after the second tick (the test is
// inside the package for this); the third Step is then refused by the guard and
// Run ends with that error.
//
// Requirement: FR-WLD-008, FR-WLD-012
func TestRun_FailedStepEndsTheRun(t *testing.T) {
	t.Parallel()
	var w *World
	var ended []RunEnded
	synthetic := errors.New("synthetic failure")
	w = internalShell(t, WithObserver(ObserverFunc(func(e Event) {
		switch e := e.(type) {
		case TickEnded:
			if e.Tick == 1 {
				w.failed = synthetic
			}
		case RunEnded:
			ended = append(ended, e)
		}
	})))
	got, err := w.Run(context.Background(), 5)
	if got != 2 || !errors.Is(err, ErrWorldFailed) || !errors.Is(err, synthetic) || !strings.Contains(err.Error(), "after 2 ticks") {
		t.Errorf("Run(5) = (%d, %v), want (2, ErrWorldFailed wrapping the cause, naming 2 ticks)", got, err)
	}
	if want := []RunEnded{{Tick: 2, Completed: 2, Failed: true}}; !slices.Equal(ended, want) {
		t.Errorf("RunEnded events = %+v, want %+v", ended, want)
	}
}

// TestAuditFailure verifies the pure mapping of the ledger's audit result: nil
// to nil, any other error to one that wraps ErrAuditFailed and the ledger's
// error and names the tick (D190). The real ledger cannot be made to fail from
// the root, so the mapping is tested alone.
//
// Requirement: FR-WLD-006
func TestAuditFailure(t *testing.T) {
	t.Parallel()
	if got := auditFailure(3, nil); got != nil {
		t.Errorf("auditFailure(3, nil) = %v, want nil", got)
	}
	cause := errors.New("account 4 negative")
	got := auditFailure(7, cause)
	if !errors.Is(got, ErrAuditFailed) || !errors.Is(got, cause) || !strings.Contains(got.Error(), "tick 7") {
		t.Errorf("auditFailure(7, %v) = %v, want ErrAuditFailed wrapping the cause and naming tick 7", cause, got)
	}
}

// TestWorld_AuditOnAHealthyWorld verifies that Audit returns nil on a world
// that holds its invariants, before and after a step.
//
// Requirement: FR-WLD-007
func TestWorld_AuditOnAHealthyWorld(t *testing.T) {
	t.Parallel()
	w := internalShell(t)
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() at tick 0 = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() after a step = %v, want nil", err)
	}
}
