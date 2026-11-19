package insure

// Failure is the reason a tentative fill does not form, the closed set of
// spec-insurance.md 7.4 and 7.2.2 in the order the checks run; the root maps each to
// its exported reason constant. None is 0.
type Failure int

const (
	// None says the fill passes every check.
	None Failure = iota
	// TooManyPolicies says MaxPolicies are open; it ends the round for the lane.
	TooManyPolicies
	// BelowMinimum says the sum is below MinSum or the premium is below 1.
	BelowMinimum
	// Funds says the premium exceeds the insured's available money.
	Funds
	// Overflow says the premium's quotient passes the money bound, or the
	// underwriter's balance plus the premium would.
	Overflow
	// NoRisk says the lane's expected loss is 0, so there is nothing to insure.
	NoRisk
	// NoOffer says no offer of the lane was at the price or had the room.
	NoOffer
)

// Form is the input of Check: the numbers of one tentative fill. PremiumOK is false
// when the premium's quotient did not fit the money bound.
type Form struct {
	// Open is the number of policies open now, those formed this round included.
	Open int64
	// MaxPolicies is the limit of open policies.
	MaxPolicies int64
	// Sum is the candidate's sum insured and MinSum the smallest sum formed.
	Sum, MinSum int64
	// Premium is the premium and PremiumOK whether it was computed within the bound.
	Premium   int64
	PremiumOK bool
	// Available is the insured's balance less what its orders reserved.
	Available int64
	// Balance is the underwriter's balance and Limit the money bound.
	Balance, Limit int64
}

// Check runs the checks of spec-insurance.md 7.4 in their order and returns the
// first failure, or None (FR-INS-036): too many policies, below the minimum (a sum
// below MinSum, or a premium below 1), funds, overflow.
func Check(f Form) Failure {
	switch {
	case f.Open >= f.MaxPolicies:
		return TooManyPolicies
	case f.Sum < f.MinSum || (f.PremiumOK && f.Premium < 1):
		return BelowMinimum
	case f.PremiumOK && f.Premium > f.Available:
		return Funds
	case !f.PremiumOK || f.Balance > f.Limit-f.Premium:
		return Overflow
	}
	return None
}
