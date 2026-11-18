// Package fiscal holds the pure arithmetic and the matching of the fiscal module
// (spec-fiscal.md 5 to 9) as functions over primitive values: the due of a tax
// payer, the base and due of a duty in 128 bits, the toll of a crossing, the pro
// rata cut of a treasury that holds less than is due, and the limit price, the
// wish, the caps and the pair rule of a standing order. It imports the standard
// library, internal/checked and internal/moneyflow only, knows no role, region,
// jurisdiction, commodity or agent identifier (architecture.md 9.1, 2.9), keeps no
// state beyond the scratch a caller passes in, takes no random source (a tie is
// broken by a key the caller computes with the stateless key K and passes in),
// logs and emits nothing, and uses no map and no floating point (FR-FIS-062,
// architecture.md 3.5).
//
// A function that multiplies or sums reports ok false on an overflow or on an
// input outside its domain, as internal/checked does, and never wraps. A quotient
// is truncated toward zero unless its doc comment says otherwise (spec-fiscal.md
// 11.3).
package fiscal

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md 3.5).
const Basis int64 = 10_000
