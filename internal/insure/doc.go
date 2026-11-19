// Package insure holds the pure arithmetic and the matching of the insurance
// module (spec-insurance.md 6, 7.3, 7.4 and 8.2) as functions over primitive
// values: the unit value, the sum insured by shape, the premium and the maximum
// payout, the loss value and the due, the order and the matching of the requests
// and offers of a round, and the checks of a tentative fill. It imports the
// standard library and internal/checked only, knows no place, transport kind, good
// or party identifier (architecture.md 9.1), keeps no state beyond the rows a caller
// passes in, takes no random source (a tie is broken by a key the caller computes and
// passes in), logs and emits nothing, and uses no map and no floating point
// (architecture.md 2.9, 3.5, NFR-INS-003, NFR-INS-005).
//
// A money bound is a parameter: the root passes its MaxMoney as limit, because an
// internal package never imports the root (architecture.md 2.9). A function that
// multiplies or sums reports ok false on an overflow, on a result above limit or on
// an input outside its domain, and never wraps; the one exception is LossValue,
// which saturates at limit as spec-insurance.md 6.3.5 states. A quotient is rounded
// half to even where its doc comment says so (spec-insurance.md 6.4.1).
package insure

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md 3.5).
const Basis int64 = 10_000
