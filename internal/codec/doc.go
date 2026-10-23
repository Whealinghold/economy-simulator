// Package codec holds the primitives of the checkpoint encoding of
// spec-checkpoint.md 5.2: the unsigned varint U, the zigzag I, the boolean B,
// raw and length-prefixed bytes, and the counts, indexes and orders that the
// rules R1 to R13 check. Every value has exactly one spelling, so equal states
// give equal bytes and an accepted input re-encodes to the same bytes.
//
// A Writer cannot fail. A Reader keeps its first error (a *Error that matches
// ErrMalformed) and returns zero values after it, so a section decoder reads
// straight down and checks Err at the end of a record. The package imports the
// standard library only and none of the module's packages (architecture.md
// 2.6, 2.9), keeps no state and uses no randomness, no clock and no
// floating point (NFR-CHK-005).
package codec
