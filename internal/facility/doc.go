// Package facility holds the pure arithmetic of the infrastructure module of
// spec-infrastructure.md as functions over primitive values: the effect
// fraction of a facility (3.3.2), the capped sum of gains and the transit
// layer (5.2.2), the most-worn-first assignment of use wear (5.1.3), the
// repair distribution (4.3.2) and the round robin of an initial row (3.4.2).
// It imports the standard library and internal/checked only, knows no region,
// class, recipe or agent identifier (architecture.md 9.1: a facility is a Ref,
// an index of the caller's slice), keeps no state between calls, takes no
// random source, logs and emits nothing, and uses no map and no floating point
// (architecture.md 2.9, 3.5).
//
// A function that multiplies reports ok false on an overflow or on an input
// outside its documented range, as internal/checked does, and never wraps. Each
// function states its rounding. Every order that affects a result ends in the
// caller's Ref, a unique key, so it is total (architecture.md 7.6).
package facility

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md
// 3.5).
const Basis int64 = 10_000
