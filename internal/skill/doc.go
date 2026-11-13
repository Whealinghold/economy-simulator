// Package skill holds the pure arithmetic of the traineeship of
// spec-skill-barriers.md 4 and of the premium checks of its section 3 as
// functions over primitive values: the trainer's gain, the teaching
// accumulator, the places a region offers, the order of the queue and of a
// requeue, the expected wait, the training cost, the completion test, the run
// counter of the "no places open" warning, the break-even premium and its
// tolerance. It imports the standard library and internal/checked only, knows
// no role, region, recipe or agent identifier (architecture.md 9.1), keeps no
// state, takes no random source, logs and emits nothing, and uses no map and no
// floating point (architecture.md 2.9, 3.5, FR-SKL-028).
//
// A function that multiplies reports ok false on an overflow, as
// internal/checked does, and never wraps. A quotient is truncated toward zero
// unless its doc comment says otherwise.
package skill

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md
// 3.5).
const Basis int64 = 10_000
