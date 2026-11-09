package denomination

import (
	"fmt"
	"slices"
	"strconv"
)

// options holds the settings an Option changes.
type options struct {
	precision int
}

// Option configures NewFormatter.
type Option func(*options) error

// WithPrecision sets the number of decimal places, 0 to MaxPrecision, with
// which the smallest Unit is written. The default is 0, which truncates the
// remainder. NewFormatter returns ErrPrecision for a value outside the range.
func WithPrecision(p int) Option {
	return func(o *options) error {
		if p < 0 || p > MaxPrecision {
			return fmt.Errorf("precision %d: %w", p, ErrPrecision)
		}
		o.precision = p
		return nil
	}
}

// Formatter writes an int64 count of minor units in the Units of a Ladder. It
// is built by NewFormatter; a nil *Formatter and a zero Formatter are not
// usable. Format is a pure function of the ladder, the precision and the
// amount, and a *Formatter is safe for concurrent use.
type Formatter struct {
	units     Ladder
	precision int
}

// NewFormatter validates the ladder and the options and returns a Formatter
// over its own copy of the ladder, so a later change of the caller's slice
// changes nothing. A nil Option is ignored. It returns the first fault found,
// as ErrNoUnits, ErrUnitName, ErrUnitDuplicate, ErrUnitSize, ErrUnitOrder or
// ErrPrecision.
func NewFormatter(l Ladder, opts ...Option) (*Formatter, error) {
	if err := l.validate(); err != nil {
		return nil, err
	}
	var o options
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&o); err != nil {
			return nil, err
		}
	}
	return &Formatter{units: slices.Clone(l), precision: o.precision}, nil
}

// Format writes amount, a count of minor units, as the counts of the Units from
// the largest down, each followed by its symbol and joined by one space. The
// largest Unit is unbounded, a Unit with a zero count is left out, the
// smallest Unit carries the precision's decimal places, and a sign is written
// only when a printed count or fraction is non-zero.
func (f *Formatter) Format(amount int64) string {
	mag, neg := magnitude(amount)
	var body []byte
	last := len(f.units) - 1
	nonZero := false
	for i, u := range f.units {
		size := uint64(u.Size)
		count := mag / size
		mag %= size
		if i == last {
			var frac uint64
			body, frac = f.appendSmallest(body, count, mag, size)
			nonZero = nonZero || count != 0 || frac != 0
			break
		}
		if count != 0 {
			body = appendUnit(body, count, u.Symbol)
			nonZero = true
		}
	}
	if neg && nonZero {
		return "-" + string(body)
	}
	return string(body)
}

// magnitude returns the absolute value of amount as a uint64 and whether the
// amount is negative. -uint64(amount) is exact for math.MinInt64.
func magnitude(amount int64) (uint64, bool) {
	if amount < 0 {
		return -uint64(amount), true
	}
	return uint64(amount), false
}

// appendUnit appends "<count> <symbol>" to dst, after a space if dst is not
// empty.
func appendUnit(dst []byte, count uint64, symbol string) []byte {
	if len(dst) > 0 {
		dst = append(dst, ' ')
	}
	dst = strconv.AppendUint(dst, count, 10)
	dst = append(dst, ' ')
	return append(dst, symbol...)
}

// appendSmallest appends the smallest Unit, count and the remainder r (which
// is below size) written with the precision's decimal places, and returns the
// fraction. It prints nothing when count and fraction are both zero and
// something was printed before; when dst is empty it always prints, so the
// output is never empty (the zero form: "0 <symbol>" or "0.00 <symbol>").
//
// The fraction is r * 10^p / size, truncated and zero-padded to p digits.
// r < size <= MaxSize = 10^12 and p <= MaxPrecision = 6, so r * 10^p < 10^18,
// which is below 2^64: the product cannot overflow.
func (f *Formatter) appendSmallest(dst []byte, count, r, size uint64) ([]byte, uint64) {
	sym := f.units[len(f.units)-1].Symbol
	var frac uint64
	if f.precision > 0 {
		frac = r * pow10(f.precision) / size
	}
	if count == 0 && frac == 0 && len(dst) > 0 {
		return dst, 0
	}
	if f.precision == 0 {
		return appendUnit(dst, count, sym), 0
	}
	if len(dst) > 0 {
		dst = append(dst, ' ')
	}
	dst = strconv.AppendUint(dst, count, 10)
	dst = append(dst, '.')
	digits := strconv.AppendUint(nil, frac, 10)
	for n := len(digits); n < f.precision; n++ {
		dst = append(dst, '0')
	}
	dst = append(dst, digits...)
	dst = append(dst, ' ')
	return append(dst, sym...), frac
}

// pow10 returns 10^p for p in 0 to MaxPrecision.
func pow10(p int) uint64 {
	v := uint64(1)
	for range p {
		v *= 10
	}
	return v
}
