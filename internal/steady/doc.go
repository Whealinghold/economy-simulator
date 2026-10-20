// Package steady is the exact solver behind the viability rules and the
// warm-start table (spec-viability.md 5 and 6): a fraction-free simplex over
// unbounded integers, with Bland's rule, and a certificate that checks every
// answer from the original data before it is returned.
//
// The package holds no state and knows no commodity, region or role: a
// programme is rows and columns named by index, and a caller that needs the
// original object keeps the index (architecture.md 2.9). It imports the
// standard library only; math/big is allowed here and in no other non-test
// file of the module (architecture.md 3.5). No floating-point type or function
// is used. A result leaves the package as an exact fraction (Frac) or, through
// RoundHalfEven, Floor and Ceil, as an int64 once, checked against the caller's
// bound; those three (and the methods Floor, Ceil and Round of Frac, which call
// them) are the only functions that return an int64 from a big value. Frac is
// opaque to the root, which combines fractions with the exact methods of
// rational.go.
//
// The solver is a pure function of its arguments: it draws no random number,
// writes no log and starts no goroutine (FR-VIA-020). It takes a context and
// checks it between pivots. It is outside the tick path, so its allocation is
// not counted in the limits of observability-and-logging.md 8.2 (NFR-VIA-001).
//
// The order of the columns is part of the input: with alternative optima Bland's
// rule picks the vertex by index, so the caller builds the columns in canonical
// order (FR-VIA-022).
package steady
