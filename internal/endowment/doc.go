// Package endowment holds the rules of the resources module over primitive
// numbers (spec-resources.md): the regeneration of the three shapes with its
// exact carry, the effective capacity and scale, the abundance of a stock, the
// ration of a scarce stock, and the two integer conditions of the stability
// check. It knows no region, resource or agent: the root converts at the call
// (architecture.md 2.9).
//
// The regeneration numerator reaches 10^31, beyond the int64 helpers of
// internal/checked, so wide.go holds a local unsigned 128-bit value.
package endowment
