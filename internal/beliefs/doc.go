// Package beliefs holds the price-belief rule of spec-beliefs.md as pure
// functions over primitive structs: the bounds of a belief, the clamp that
// keeps it inside them and the draw of a limit price from it. It imports the
// standard library and internal/checked only, holds no state, takes no random
// source (the caller passes the one uint64 each draw consumes), logs and emits
// nothing, and uses no map and no floating point (architecture.md 2.9, 3.5).
//
// Every ratio is in basis points and goes through checked.MulDiv, truncated
// toward zero. A product that does not fit int64 never wraps and never panics:
// it saturates as the doc comment of each function says.
package beliefs
