package otelobserver

import (
	"errors"
	"testing"
)

// TestNew_OptionErrorIsReturned verifies that New returns the error of an option
// that fails, wrapped so that errors.Is finds it, and no adapter.
//
// Requirement: FR-OBS-050
// Guard: no built-in option fails today, so this branch is reachable only through
// an Option of the package; without the test a future validating option could
// have its error dropped.
func TestNew_OptionErrorIsReturned(t *testing.T) {
	errOpt := errors.New("option refused")
	obs, err := New(nil, nil, func(*options) error { return errOpt })
	if !errors.Is(err, errOpt) || obs != nil {
		t.Errorf("New(failing option) = %v, %v, want nil and %v", obs, err, errOpt)
	}
}
