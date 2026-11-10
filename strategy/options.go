package strategy

import economysim "github.com/Whealinghold/economy-sim"

// MerchantName is the name under which Options registers Merchant, the value of
// Role.Strategy that selects it (spec-merchants.md 3.1.1).
const MerchantName = "merchant"

// Options returns the options that register the standard strategies of this
// package under their names, in the order of the names: today "merchant". A
// consumer passes them to NewWorld or RestoreWorld; a configuration that names
// "merchant" without them fails with ErrUnknownStrategy.
func Options() []economysim.Option {
	return []economysim.Option{economysim.WithStrategy(MerchantName, Merchant{})}
}
