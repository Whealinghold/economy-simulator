// Package risk holds the pure rules of the risk module over primitive values
// (spec-risk.md 3.4, 4.3, 5.3, 5.4, 6.1 to 6.3 and 8.1): the effective
// probability of a row, the strike test, the fold of the kept share of a
// shipment, the loss of a line, the stored fraction of a store class and the
// share of a class, the expected loss of a path and the loss of a trip
// account. It knows no region, mode, store class or commodity: it receives
// numbers, and the root converts at the call (architecture.md 2.9).
//
// It imports the standard library and internal/checked only. It draws no
// word: the world's draw, already reduced to [0, 10 000), is a parameter.
// Every share is in basis points and every rounded quotient is rounded half to
// even, the rule constant of the module (spec-risk.md 12.4). Every product is
// checked: a function that cannot be applied reports ok false with a zero
// result (architecture.md 3.5).
package risk
