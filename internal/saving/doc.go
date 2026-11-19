// Package saving holds the pure rules of the saving and investment module over
// primitive values (spec-saving-and-investment.md 4 and 5): the precaution
// buffer and the floor of the savings target, the value per tick of a recipe,
// the job value, the rate and service time of a facility, the benefit and cost
// of a project, the hurdle with its spread and commit band, the test, the
// choice between two projects, the outlay, the held value, the goal, the funds
// test, the reach test and the terms of a repair. It knows no class, recipe,
// role or commodity by name: a class is data the root resolves, and the root
// converts to and from these primitives inline at the call (architecture.md
// 2.9). It imports the standard library and internal/checked only, draws
// nothing, uses no map and no floating point (NFR-SAV-005).
//
// Every product is held in 128 bits and every quotient is truncated toward zero
// unless a doc comment says floor or ceil. A value that leaves int64 saturates
// at the nearest extreme and no function returns an ok, because the world
// cannot refuse inside a tick (FR-SAV-043); Benefit and Cost saturate at the
// money bound the caller passes, so a saturated benefit passes the test and a
// saturated cost fails it (FR-SAV-014). Arguments that the spec bounds below
// zero (a count, a duration, a stock) are taken as 0 when they are negative.
package saving
