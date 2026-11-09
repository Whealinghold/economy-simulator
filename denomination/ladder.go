package denomination

import (
	"errors"
	"fmt"
)

// MaxSize is the largest size of a Unit, in minor units.
const MaxSize = 1_000_000_000_000

// MaxPrecision is the largest number of decimal places of the smallest Unit.
const MaxPrecision = 6

// ErrNoUnits is returned when a Ladder has no Unit.
var ErrNoUnits = errors.New("ladder has no units")

// ErrUnitName is returned when a Unit has an empty name or an empty symbol.
var ErrUnitName = errors.New("unit has an empty name or symbol")

// ErrUnitDuplicate is returned when a name or a symbol is used by two Units.
var ErrUnitDuplicate = errors.New("unit name or symbol is used twice")

// ErrUnitSize is returned when a Unit's size is outside 1 to MaxSize.
var ErrUnitSize = errors.New("unit size is outside 1 to MaxSize")

// ErrUnitOrder is returned when the sizes of a Ladder do not strictly decrease.
var ErrUnitOrder = errors.New("unit sizes are not strictly decreasing")

// ErrPrecision is returned when a precision is outside 0 to MaxPrecision.
var ErrPrecision = errors.New("precision is outside 0 to MaxPrecision")

// Unit is one rung of a Ladder: a name, a symbol and a size in minor units.
// Format writes the symbol; the name is for the consumer's own use.
type Unit struct {
	Name   string
	Symbol string
	Size   int64
}

// Ladder is an ordered list of Units, largest first, with strictly decreasing
// sizes. A size need not divide the next larger one.
type Ladder []Unit

// validate returns the first fault of the ladder: no units; then, unit by
// unit, name and symbol, size, duplicates, order. Each is wrapped with the
// index of the unit.
func (l Ladder) validate() error {
	if len(l) == 0 {
		return ErrNoUnits
	}
	for i := range l {
		if err := l.validateUnit(i); err != nil {
			return fmt.Errorf("unit %d: %w", i, err)
		}
		if err := l.validateOrder(i); err != nil {
			return fmt.Errorf("unit %d: %w", i, err)
		}
	}
	return nil
}

// validateUnit checks the name, symbol and size of unit i, and that neither
// name nor symbol was used by an earlier unit.
func (l Ladder) validateUnit(i int) error {
	u := l[i]
	if u.Name == "" || u.Symbol == "" {
		return ErrUnitName
	}
	if u.Size < 1 || u.Size > MaxSize {
		return ErrUnitSize
	}
	for _, e := range l[:i] {
		if e.Name == u.Name || e.Symbol == u.Symbol {
			return ErrUnitDuplicate
		}
	}
	return nil
}

// validateOrder checks that unit i is strictly smaller than the one before it.
func (l Ladder) validateOrder(i int) error {
	if i > 0 && l[i].Size >= l[i-1].Size {
		return ErrUnitOrder
	}
	return nil
}
