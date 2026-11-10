// Package merchant holds the rules of the merchants module over primitive numbers
// (spec-merchants.md): the fixed spread of the fee share, the cautious revenue
// of a rumour, the perishable test, the cost of carriage, the test of the fee
// share and the prices that follow from it, the sizing of a lot, the take of a
// consignee, the matching of delivery offers with takes, the midpoint of a pair
// and the settlement of a contract. It knows no region, commodity, role, mode or
// agent: the root converts at the call and keeps the references it needs
// (architecture.md 2.9). Every product is checked or held in 128 bits and an
// overflow is reported as ok false, never wrapped, and rounds against the
// merchant (spec-merchants.md 5.8); there is no floating point and no random
// source.
package merchant

// wholeBP is 10 000 basis points, a ratio of 1.
const wholeBP = 10_000
