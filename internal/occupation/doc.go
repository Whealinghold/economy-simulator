// Package occupation holds the pure arithmetic of the spawner as functions over
// primitive values (spec-spawner.md 4 to 7): the signals E and V of a role and
// the comparisons made with them, the rung-3 test, the review rotation, the
// exit allowance and the reduction of an agent, the scaling of a reduction, the
// renewal carry, the funded number, the placement of an entrant and the target
// carried over a role change. It imports the standard library and
// internal/checked and internal/ordering only, knows no role, region,
// commodity or agent identifier (architecture.md 9.1), keeps no state, takes no
// random source, logs and emits nothing, and uses no map and no floating point
// (architecture.md 2.9, 3.5).
//
// A value that names an agent or a role is an index into the caller's slice
// (Ref), which the result returns. A quotient is truncated toward zero unless
// its doc comment says otherwise. A function that multiplies and cannot
// saturate reports ok false on an overflow, as internal/checked does; the
// signals saturate by the sign of the wage instead, because the world cannot
// refuse at the point where it reads them (spec-spawner.md 6.1.2).
package occupation

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md
// 3.5).
const Basis int64 = 10_000
