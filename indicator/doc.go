// Package indicator derives measures of a run from snapshots and events, as
// pure functions in fixed point (spec-indicators.md).
//
// An indicator is a function of a snapshot (or two), of a Tally an Observer
// kept, and of a price table and a class table passed as arguments. It takes no
// *World, calls nothing back into one and draws no random number, so nothing in
// the world reads an indicator and a run with every preset computed every tick
// is byte for byte the run with none. The package imports the root package
// (imported by economysim "github.com/Whealinghold/economy-sim") and
// internal/measure, and nothing in the module imports it.
//
// The three generic aggregators are the value of flows at a price table, a
// weighted statistic over household rows (Stat) and a window statistic over a
// series (Stats); every preset is a named, fixed call of them.
//
// Arithmetic is integer: sums in 128 bits, one rounding rule (half to even,
// applied at the last division), ratios and indices in basis points (BP) of a
// base of 10 000. A measure that is undefined (a zero divisor, an empty set) is
// a result with OK false, never an error and never a panic; the sentinels in
// errors.go are for a misuse of the inputs.
//
// A series, a base and a tally belong to the consumer: the package keeps no
// history and no package-level state.
package indicator
