// Package rumour holds the rules of the information module over primitive
// numbers (spec-information.md): the lag of a rumour, the decision whether a
// candidate gets one, the multiplicative error of a rumour, and the ring that
// records what each market published. It knows no region, commodity, role or
// agent: the root converts at the call (architecture.md 2.9). Every product is
// checked and an overflow is reported as ok false, never wrapped; there is no
// floating point and no random source (the root draws the error and passes it
// in).
package rumour
