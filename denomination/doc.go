// Package denomination is the optional display of money: a Ladder of Units, a
// Formatter that writes an int64 count of minor units in them, and the presets
// RPG and Single.
//
// The core never imports this package, and removing it changes no simulation
// result; a consumer formats at its own edge, for example
// f.Format(int64(price)). The package imports the standard library only, holds
// no state after construction, has no logger, observer, clock or random source,
// and returns errors only from NewFormatter.
//
// The output is the parts of the amount, largest Unit first, joined by one
// space, each as "<count> <symbol>": "1 gp 2 sp 3 cp". A Unit whose count is
// zero is left out; the smallest Unit is written with the Precision decimal
// places the Formatter was built with, "3.45 cp". A negative amount has a
// leading "-" with no space, and a zero amount is "0 <smallest symbol>".
//
// A *Formatter is safe for concurrent use: Format reads only the Formatter's
// own copy of the Ladder.
package denomination
