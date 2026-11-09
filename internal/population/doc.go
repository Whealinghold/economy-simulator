// Package population holds the rules of the population module over primitive
// numbers (spec-population.md): the evaluation of a rate schedule with its
// region scale, the exact carried remainder that turns a rate into whole
// households and the Weyl start of a death carry, the floor and cap cuts, the
// replacement ratio and the gain test of the Malthusian fixed point, and the
// household-weighted mean rates of the snapshot. It knows no region, role or
// agent: the root converts at the call (architecture.md 2.9). Every product is
// checked and an overflow is reported, never wrapped; there is no floating
// point and no random source.
package population
