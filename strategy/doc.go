// Package strategy holds additional strategies for economysim: the first is
// Merchant, the strategy of a merchant role (spec-merchants.md). The default
// strategy of each extension interface lives in the root package, which cannot
// import this one (architecture.md 2.7), so a consumer registers the standard
// names of this package with Options:
//
//	economysim.NewWorld(cfg, strategy.Options()...)
//
// The package keeps no state and no random source: every strategy in it takes its
// randomness from the world, as architecture.md 7.7 requires, and a strategy that
// holds no field is not Stateful (architecture.md 4.3).
package strategy
