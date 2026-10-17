// Package household holds the pure rules of the default strategy and of the
// agent's consumption, as spec-agents.md 5.4, 7.1, 7.3, 7.4 and 7.5 state them:
// the fed count and the hunger and privation steps, the upkeep cost and the
// reserve money, the targets of a commodity, the tiers of a bid, the spend cap,
// the buy share and the quantity one tier can afford. The recipe choice, the
// production halt and the keep-the-food cap are in the same package.
//
// The functions are scalar: one takes the numbers of one commodity or one
// recipe and the caller loops, so the tick converts nothing and allocates
// nothing (spec-agents.md NFR-AGT-001; architecture.md 2.9). The package
// imports the standard library and internal/checked only, knows no role,
// region, commodity or agent identifier (architecture.md 9.1), keeps no state,
// takes no random source, logs and emits nothing, and uses no map and no
// floating point. A function that multiplies reports ok false on an overflow,
// as internal/checked does; the others cannot overflow for any input. A
// quotient is truncated toward zero unless its doc comment says otherwise.
package household

// Basis is 10 000, the denominator of a basis-point ratio (architecture.md
// 3.5).
const Basis int64 = 10_000

// EmptyPos is the position in the price window, in basis points, that
// spec-agents.md 7.5.2 gives when the window is empty: the middle, 5 000. The
// caller passes it to BuyShare then. It is a rule constant, not a tunable
// (architecture.md 9.6).
const EmptyPos int64 = 5_000

// Params holds the eight tunables the rules read (spec-agents.md 12), copied by
// the caller from the view's Tunables group.
type Params struct {
	// CoverTicks is the ticks of use a household holds at the start of the next
	// tick.
	CoverTicks int64
	// InputJobs is the number of jobs of inputs a household holds.
	InputJobs int64
	// OutputCoverTicks is the ticks of output a maker holds to sell.
	OutputCoverTicks int64
	// FavourFloor is the least share of a tier-3 want that is bought, in basis
	// points.
	FavourFloor int64
	// ReserveTicks is the ticks of upkeep cost kept as the reserve money.
	ReserveTicks int64
	// SpendMaxShare is the largest share of the excess the spend cap allows, in
	// basis points.
	SpendMaxShare int64
	// SpendHalfTicks is the ticks of upkeep cost at which the cap share is half.
	SpendHalfTicks int64
	// ProduceHaltRatio is the share of its output target at which a recipe
	// halts, in basis points.
	ProduceHaltRatio int64
}
