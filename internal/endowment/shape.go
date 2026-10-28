package endowment

// Shape is the rule by which an endowment regenerates (spec-resources.md 4.2).
// The set is closed: adding a shape is a change of the spec.
type Shape int

// The three shapes. Constant is the zero value, as the spec's default.
const (
	// Constant is a flow that does not depend on the stock; with a rate of 0 it
	// is a finite deposit.
	Constant Shape = iota
	// Refill closes the gap to the capacity at a rate.
	Refill
	// Logistic grows in proportion to the stock and to the room left.
	Logistic
)

// BP is one whole in basis points: the neutral factor of a curve with no knots,
// and the divisor of every basis-point scale (architecture.md 3.5).
const BP = 10_000

// ticksDenominator is the 10^12 of D = 10^12 * TicksPerYear (spec-resources.md
// 4.2): three basis-point divisions of 10^4 each, the fourth being the year.
const ticksDenominator = 1_000_000_000_000

// Params are the constants of one endowment for a tick. Rate is units a year for
// Constant and basis points a year for Refill and Logistic. Capacity and
// ScaleBP are the effective values of spec-resources.md 3.4.
type Params struct {
	Shape        Shape
	Rate         int64
	Refuge       int64
	Capacity     int64
	ScaleBP      int64
	TicksPerYear int64
}

// Step is the state going into a tick: the stock, the carry (in units of 1/D)
// and the regeneration factor of the tick's season in basis points.
type Step struct {
	Stock    int64
	Carry    int64
	SeasonBP int64
}

// Result is the state after a tick. Added is the units actually kept, which is
// less than the units computed when the stock reached the capacity.
type Result struct {
	Stock int64
	Carry int64
	Added int64
}
