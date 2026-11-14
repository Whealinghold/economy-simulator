// Package technology holds the pure arithmetic of the technology module of
// spec-technology.md 4 as functions over primitive values: the fixed-point
// logarithm and the learned gain, the trend's carried remainder, the keyed
// discovery hash, the contact weight, the frontier and the catch-up, and the
// step of one tick composed from them (4.6). It imports the standard library,
// internal/checked and internal/ordering only, knows no region, recipe, field
// or agent identifier (architecture.md 9.1: a cell is an index), keeps no state
// between calls except what the caller passes in, takes no random source, logs
// and emits nothing, and uses no map and no floating point (architecture.md
// 2.9, 3.5).
//
// A function that multiplies reports ok false on an overflow, as
// internal/checked does, and never wraps. The one wrap that is allowed is the
// modular product of the discovery hash, which is the hash (spec-technology.md
// 4.4.2). A quotient is truncated toward zero unless its doc comment says
// otherwise.
package technology

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md
// 3.5).
const Basis int64 = 10_000
