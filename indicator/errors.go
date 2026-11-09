package indicator

import (
	"errors"
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

var (
	// ErrConfigMismatch is returned for two snapshots, or a snapshot and a price
	// table, from different configuration hashes.
	ErrConfigMismatch = errors.New("indicator: configuration hash mismatch")
	// ErrWindowOrder is returned when the open tick is not below the close tick,
	// or a tally's events do not cover the window exactly once.
	ErrWindowOrder = errors.New("indicator: window out of order")
	// ErrUnknownClass is returned for a class name no commodity carries.
	ErrUnknownClass = errors.New("indicator: unknown class")
	// ErrRange is returned for an ID outside its slice of the names table, a
	// negative weight or value where it is not allowed, or a share above 10 000.
	ErrRange = errors.New("indicator: value out of range")
	// ErrNegativeValue is returned for the Gini coefficient of a row set with a
	// negative value.
	ErrNegativeValue = errors.New("indicator: negative value")
	// ErrOverflow is returned for a result that does not fit int64, or for a
	// product past the guard of spec-indicators.md 5.6.
	ErrOverflow = errors.New("indicator: overflow")
)

// wrap returns err wrapped with the name of the preset and the field, so a
// caller can branch with errors.Is and read where it came from
// (architecture.md 8.2).
func wrap(preset, field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", preset, field, err)
}

// fromMeasure maps an error of internal/measure to the sentinel of this package.
func fromMeasure(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, measure.ErrNegative):
		return ErrNegativeValue
	case errors.Is(err, measure.ErrRange):
		return ErrRange
	}
	return ErrOverflow
}
