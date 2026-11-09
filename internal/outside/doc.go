// Package outside holds the rules of the external module over primitive numbers
// (spec-external-sector.md 6 and 7): the mid of a line, the walk of its drift,
// the quote of a side at a gate, the units a side wants in a tick with the carry
// of the fraction, and the cumulative caps. It knows no map, no market and no
// agent: the root package converts at the call (architecture.md 2.9). Every
// product is checked and an overflow is reported as ok false, never wrapped;
// there is no floating point and no random source (the root draws the step of
// the walk and passes it in).
package outside
