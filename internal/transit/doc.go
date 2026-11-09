// Package transit holds the rules of the transport module over primitive
// numbers (spec-transport.md): the travel time of a distance at a speed and a
// transit factor, the carrying units a load needs and the pair test of a fit,
// the haul units a distance costs, the labour charge of a lane of the viability
// solver, and the free list of Transit accounts. It knows no region, mode,
// commodity or agent: the root converts at the call (architecture.md 2.9). Every
// product is checked or held in 128 bits and an overflow is reported as ok
// false, never wrapped; there is no floating point and no random source.
package transit
