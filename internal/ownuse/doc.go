// Package ownuse holds the rules of the subsistence module over primitive
// numbers (spec-subsistence.md): the cover, cost, afford and secure ticks of a
// household and the enter and leave rule of the own-use mode (5.2, 5.3), the
// coverage of a commodity by a recipe and the floor figure of a region (4.1,
// 4.2), the choice among own-use recipes, the ramp of a cohort and the retry
// test (5.5). It knows no commodity, recipe, role or agent: the root passes
// stocks, upkeeps, prices and opaque references and converts the answer back
// (architecture.md 2.9).
//
// Every product goes through internal/checked, so nothing wraps. A function
// that can overflow returns an ok result that is false then; the root reads it
// as the saturated value its doc comment names. The package uses integers
// only, draws no random number and holds no state.
package ownuse
