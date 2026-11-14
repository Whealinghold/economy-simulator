// Package credit holds the pure arithmetic and the matching of the credit
// module (spec-credit.md 6 to 9) as functions over primitive values: the interest
// of a loan, the pot of a debtor, the allocation of a payment among claims by the
// largest remainder, the cut of a payment to a creditor's headroom, the order and
// the matching of the requests and offers of a round, and the waterfall of a
// removed debtor. It imports the standard library and internal/checked only,
// knows no role, region, agent or claim identifier (architecture.md 9.1), keeps
// no state beyond the scratch a caller passes in, takes no random source (a tie is
// broken by a key the caller computes and passes in), logs and emits nothing, and
// uses no map and no floating point (architecture.md 2.9, 3.5, FR-CRD-062).
//
// A function that multiplies or sums reports ok false on an overflow or on an
// input outside its domain, as internal/checked does, and never wraps. A quotient
// is truncated toward zero unless its doc comment says otherwise.
package credit

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md 3.5).
const Basis int64 = 10_000
