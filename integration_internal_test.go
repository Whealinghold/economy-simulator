package economysim

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestEnter_Order verifies that a call's guard runs in the order of
// spec-integration.md 3.2: a running step before a failed world before a module
// that is off, and that no module named means none needed.
//
// Requirement: FR-INT-001, FR-INT-006
func TestEnter_Order(t *testing.T) {
	t.Parallel()
	failed := errors.New("boom")
	tests := []struct {
		name     string
		stepping bool
		failed   error
		module   string
		on       bool
		want     error
	}{
		{"all clear", false, nil, "", false, nil},
		{"module on", false, nil, "transport", true, nil},
		{"module off", false, nil, "transport", false, ErrModuleOff},
		{"failed beats module off", false, failed, "transport", false, ErrWorldFailed},
		{"step beats failed", true, failed, "transport", false, ErrInStep},
		{"step alone", true, nil, "", false, ErrInStep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := internalShell(t)
			w.stepping, w.failed = tt.stepping, tt.failed
			err := w.enter("Call", tt.module, tt.on)
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Fatalf("enter(Call, %q, %v) = %v, want %v", tt.module, tt.on, err, tt.want)
			}
			if err != nil && !strings.HasPrefix(err.Error(), "Call: ") {
				t.Errorf("enter error %q does not start with the call's name", err)
			}
			if tt.want == ErrWorldFailed && !errors.Is(err, failed) {
				t.Errorf("enter error %v does not wrap the cause %v", err, failed)
			}
		})
	}
}

// TestInStep_Enter verifies that the guard also fires for a call made from an
// observer of a running step.
//
// Requirement: FR-INT-001
func TestInStep_Enter(t *testing.T) {
	t.Parallel()
	var got error
	var w *World
	obs := ObserverFunc(func(e Event) {
		if _, ok := e.(TickStarted); ok && got == nil {
			got = w.enter("Call", "", false)
		}
	})
	w = internalShell(t, WithObserver(obs))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if !errors.Is(got, ErrInStep) {
		t.Errorf("enter inside a step = %v, want ErrInStep", got)
	}
}

// TestSentinels_IntegrationDistinct verifies that the sentinels this plan adds
// have messages of their own and that ErrAgentRemoved is also an unknown agent.
//
// Guard: two sentinels with one message cannot be told apart in a log, and a
// removed agent that stopped matching ErrUnknownAgent would break a caller that
// tests for the general case (spec-integration.md 7.1.1).
func TestSentinels_IntegrationDistinct(t *testing.T) {
	t.Parallel()
	all := []error{ErrModuleOff, ErrAgentRemoved, ErrNotExternal, ErrNotDirected, ErrNotOnRole, ErrNotTradable, ErrUnknownLine, ErrUnknownAgent}
	seen := map[string]bool{}
	for _, s := range all {
		if seen[s.Error()] {
			t.Errorf("sentinel message %q is used twice, want a distinct message", s.Error())
		}
		seen[s.Error()] = true
		for _, o := range all {
			want := o == s || (s == ErrAgentRemoved && o == ErrUnknownAgent)
			if got := errors.Is(s, o); got != want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", s, o, got, want)
			}
		}
	}
}

// TestArgError_Path verifies the spelling of the path of an argument error: the
// slice entry is one segment, the field the next, and errors.Is and errors.As
// find the rule and the typed error.
//
// Requirement: FR-INT-005
func TestArgError_Path(t *testing.T) {
	t.Parallel()
	err := argError(ErrOutOfBounds, "20000", "MaxShockBP", entry("changes", 2), "bp")
	var ve *ValidationError
	if !errors.As(err, &ve) || !slices.Equal(ve.Path, []string{"changes[2]", "bp"}) || ve.Value != "20000" || ve.Bound != "MaxShockBP" {
		t.Fatalf("argError = %+v, want path changes[2], bp, value 20000, bound MaxShockBP", ve)
	}
	if !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("errors.Is(%v, ErrOutOfBounds) = false, want true", err)
	}
	if want := `"changes[2]"."bp": `; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("argError text = %q, want it to start with %q", err.Error(), want)
	}
}

// TestKeyedOnce verifies that the duplicate check returns the index of the first
// key that repeats an earlier one, leaves the slice unchanged and finds nothing in
// a slice of distinct keys.
//
// Requirement: FR-INT-003
func TestKeyedOnce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		keys []int64
		want int
	}{
		{"nil", nil, -1},
		{"one", []int64{5}, -1},
		{"distinct", []int64{3, 1, 2}, -1},
		{"adjacent pair", []int64{1, 1}, 1},
		{"far pair", []int64{4, 2, 9, 4}, 3},
		{"first repeat wins", []int64{7, 1, 1, 7}, 2},
		{"triple", []int64{2, 2, 2}, 1},
		{"extremes", []int64{-1 << 63, 1<<63 - 1, -1 << 63}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(tt.keys)
			if got := keyedOnce(tt.keys); got != tt.want {
				t.Errorf("keyedOnce(%v) = %d, want %d", tt.keys, got, tt.want)
			}
			if !slices.Equal(in, tt.keys) {
				t.Errorf("keyedOnce changed its argument: %v, want %v", tt.keys, in)
			}
		})
	}
}

// TestQueries_FailedWorld verifies that the queries with an error result return
// ErrWorldFailed on a failed world and that the readers with none return their value
// (spec-integration.md 6.1.2).
//
// Requirement: FR-INT-060
func TestQueries_FailedWorld(t *testing.T) {
	t.Parallel()
	w := internalShell(t)
	cause := errors.New("boom")
	w.failed = cause
	_, agentErr := w.Agent(0)
	_, agentsErr := w.Agents(0)
	_, marketErr := w.Market(0, 0)
	_, regionErr := w.Region(0)
	for name, err := range map[string]error{"Agent": agentErr, "Agents": agentsErr, "Market": marketErr, "Region": regionErr} {
		if !errors.Is(err, ErrWorldFailed) || !errors.Is(err, cause) {
			t.Errorf("%s on a failed world = %v, want ErrWorldFailed wrapping the cause", name, err)
		}
	}
	if w.Err() != cause || w.Calendar().HoursPerDay == 0 || w.Tick() != 0 {
		t.Errorf("the readers on a failed world gave Err %v, Calendar %+v, Tick %d, want their values", w.Err(), w.Calendar(), w.Tick())
	}
}
