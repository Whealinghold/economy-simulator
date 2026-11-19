// Package policy holds the pure rules of the policy module (spec-policy.md 6.2,
// 7.2, 8.2 and 8.3) as functions over primitive values: the due test of an
// actor's rotation, the value function of a fixed schedule, the step of a
// reactive rule with its 128-bit product, the four gauges over primitive rows,
// and the checks and the two cuts that a returned list of settings goes through.
//
// It imports the standard library, internal/checked and internal/ordering only,
// knows no region, flow, edge, role, commodity or jurisdiction name (a lever is
// a row of numbers and flags, architecture.md 9.1, 2.9), keeps no state beyond the
// scratch a caller passes in, takes no random source, logs and emits nothing, and
// uses no map, no time, no goroutine and no floating point (FR-PLC-091,
// architecture.md 3.5).
//
// A function that multiplies or sums reports a flag on an input outside its
// domain, as internal/checked does, and never wraps: a sum that leaves int64 is
// cut to the bound of int64 in its sign, which the checks then cut to the
// lever's own bounds. A quotient is truncated toward zero unless its doc comment
// says it is rounded half to even.
package policy

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md 3.5).
const Basis int64 = 10_000
