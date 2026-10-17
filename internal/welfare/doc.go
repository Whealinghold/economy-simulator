// Package welfare holds the pure arithmetic of spec-welfare-and-wages.md as
// functions over primitive values: the ratio with its padding and its earned
// form (2.1, 2.2), the calibration formulas (3.1, 5), the work share and the
// work floor (5, 5.3), the range of a drawn target (5.1), and the wear cost,
// the cost and the income of a tick (2.1, 2.1.4); the Record of a tick and the
// Ring of the last records of one agent with running sums (13); and the
// aggregates over a set (4.3, 11.2). It imports the standard library and
// internal/checked only, knows no role, region, commodity or agent identifier,
// holds no state except the Ring a caller owns, takes no random source, logs
// and emits nothing, and uses no map and no floating point (architecture.md
// 2.9, 3.5).
//
// Every quotient is truncated toward zero unless its doc comment says
// otherwise. The functions the world calls in a tick saturate: a value that
// leaves int64 becomes the nearest extreme and they return no ok, because the
// world cannot refuse at P6.3 (spec-welfare-and-wages.md 2.3.2). The
// calibration formulas run outside the tick and report ok false on an overflow
// or a zero divisor, as internal/checked does.
package welfare
