// Package production holds the yield and output arithmetic of
// spec-production.md as pure functions over primitive values: the site factor
// of the yield table (6.2), the effective factor (6.3), the output of a job
// with its carried residue (6.6) and the rescale of a job whose executor's
// weight falls (5.4). It imports the standard library and internal/checked
// only, holds no state, takes no random source, logs and emits nothing, and
// uses no map and no floating point (architecture.md 2.9, 3.5).
//
// Every factor is in basis points and is applied as v * bp / 10 000 through
// checked.MulDiv, truncated toward zero. Arithmetic that does not fit int64
// reports ok false and never wraps. The functions do not read the root's
// bounds; the caller validates its inputs.
package production
