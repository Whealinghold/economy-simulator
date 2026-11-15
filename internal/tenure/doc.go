// Package tenure holds the pure rules of the tenure module over primitive
// values (spec-land.md 4, 5.2 to 5.5 and 6.1.2): the apportionment of a stock
// among holders and the commons, the four tiers of the ration, the supply of
// the lessors and the attribution of let units to tenants, the rent in force,
// the largest number of executions a principal can pay for, the step of the
// posted rent and the cap on a holder's share. It knows no region, resource or
// agent: it receives numbers, and the root converts at the call
// (architecture.md 2.9).
//
// It imports the standard library, internal/checked and internal/endowment
// only. It takes no draw: every tie goes by the stateless key of
// spec-resources.md 5.3.3 that the caller passes in, then by the ID. Every
// product and sum is checked, and a function that cannot be applied reports ok
// false with every output zero.
package tenure
