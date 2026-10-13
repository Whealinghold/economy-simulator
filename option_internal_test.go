package economysim

import (
	"errors"
	"log/slog"
	"testing"
)

// optionsCase is one case of the option tests: the options given and the
// effect each field of options must show.
type optionsCase struct {
	name string
	opts []Option
	want func(o options) string // returns "" when the options are as expected
}

// TestApplyOptions_Effects verifies the effect of each option on the unexported
// options, including the defaults of a world built with none.
//
// Requirement: FR-WLD-003, FR-WLD-001
func TestApplyOptions_Effects(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.DiscardHandler)
	var seen []string
	obs := ObserverFunc(func(Event) { seen = append(seen, "x") })
	tests := []optionsCase{
		{"defaults", nil, func(o options) string {
			switch {
			case o.seed != 0:
				return "seed is not 0"
			case o.logger == nil:
				return "logger is nil"
			case o.observers.supplied || len(o.observers.list) != 0:
				return "observers supplied"
			case o.audit:
				return "audit on"
			case o.targetSpreadGiven || o.targetSpread != 0:
				return "target spread given"
			}
			return ""
		}},
		{"logger given", []Option{WithLogger(logger)}, func(o options) string {
			if o.logger != logger {
				return "logger is not the one given"
			}
			return ""
		}},
		{"nil logger means the discard default", []Option{WithLogger(nil)}, func(o options) string {
			if o.logger == nil || o.logger.Enabled(t.Context(), slog.LevelError) {
				return "logger is nil or enabled"
			}
			return ""
		}},
		{"observer sets the flag", []Option{WithObserver(obs)}, func(o options) string {
			if !o.observers.supplied || len(o.observers.list) != 1 {
				return "flag not set or list wrong"
			}
			return ""
		}},
		{"nil observer is ignored", []Option{WithObserver(nil), WithObserver(ObserverFunc(nil))}, func(o options) string {
			if o.observers.supplied || len(o.observers.list) != 0 {
				return "a nil observer was kept"
			}
			return ""
		}},
		{"observers are additive", []Option{WithObserver(obs), WithObserver(nil), WithObserver(obs)}, func(o options) string {
			if !o.observers.supplied || len(o.observers.list) != 2 {
				return "want two observers"
			}
			return ""
		}},
		{"audit is a flag", []Option{WithAudit()}, func(o options) string {
			if !o.audit {
				return "audit off"
			}
			return ""
		}},
		{"seed given", []Option{WithSeed(42)}, func(o options) string {
			if o.seed != 42 {
				return "seed is not 42"
			}
			return ""
		}},
		{"last seed wins", []Option{WithSeed(1), WithSeed(2), WithSeed(3)}, func(o options) string {
			if o.seed != 3 {
				return "seed is not 3"
			}
			return ""
		}},
		{"target spread 0 is given", []Option{WithTargetSpread(0)}, func(o options) string {
			if !o.targetSpreadGiven || o.targetSpread != 0 {
				return "0 not recorded as given"
			}
			return ""
		}},
		{"target spread at the top bound", []Option{WithTargetSpread(9_999)}, func(o options) string {
			if !o.targetSpreadGiven || o.targetSpread != 9_999 {
				return "9999 not recorded"
			}
			return ""
		}},
		{"last target spread wins", []Option{WithTargetSpread(500), WithTargetSpread(0)}, func(o options) string {
			if !o.targetSpreadGiven || o.targetSpread != 0 {
				return "last value not kept"
			}
			return ""
		}},
		{"nil option is skipped", []Option{nil, WithSeed(9)}, func(o options) string {
			if o.seed != 9 {
				return "seed is not 9"
			}
			return ""
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := applyOptions(tt.opts)
			if err != nil {
				t.Fatalf("applyOptions(%s) error = %v, want nil", tt.name, err)
			}
			if msg := tt.want(o); msg != "" {
				t.Errorf("applyOptions(%s) = %+v: %s", tt.name, o, msg)
			}
		})
	}
}

// TestWithTargetSpread_Bounds verifies that a spread outside 0 to 9 999 is an
// error that wraps ErrOutOfBounds, that nothing panics, and that a refused
// value leaves the options unchanged for the caller (applyOptions returns no
// options).
//
// Requirement: FR-WEL-026
func TestWithTargetSpread_Bounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		bp      int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"one", 1, false},
		{"top of the range", 9_999, false},
		{"whole", 10_000, true},
		{"negative", -1, true},
		{"far above", 1 << 62, true},
		{"most negative", -1 << 63, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := applyOptions([]Option{WithTargetSpread(tt.bp)})
			if tt.wantErr {
				if !errors.Is(err, ErrOutOfBounds) {
					t.Errorf("WithTargetSpread(%d) error = %v, want ErrOutOfBounds", tt.bp, err)
				}
				return
			}
			if err != nil || !o.targetSpreadGiven || o.targetSpread != tt.bp {
				t.Errorf("WithTargetSpread(%d) = %+v, %v, want given with %d and no error", tt.bp, o, err, tt.bp)
			}
		})
	}
}

// TestApplyOptions_StopsAtFirstError verifies that options are applied in
// order and the first failure is returned, so NewWorld builds no world from a
// bad option (FR-WLD-001).
//
// Requirement: FR-WLD-001
func TestApplyOptions_StopsAtFirstError(t *testing.T) {
	t.Parallel()
	_, err := applyOptions([]Option{WithSeed(1), WithTargetSpread(-5), WithTargetSpread(10_000)})
	if !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("applyOptions(bad spreads) error = %v, want ErrOutOfBounds", err)
	}
	if got, want := err.Error(), "target spread -5 outside 0 to 9999: value out of bounds"; got != want {
		t.Errorf("applyOptions(bad spreads) error text = %q, want %q (the first failure)", got, want)
	}
}
