// Package settlement holds the pure rules of the settlements module over
// primitive values (spec-settlements.md 3 and 4): the effective thresholds of a
// region, the review that gives a class from a household count, the dwell test,
// the class a world is built with, the dense tables of flags and limits, and the
// headroom of a limit. It knows no region, role, flag or class by name: a class
// is an index, a flag or a limit is an index into a table, and the root converts
// at the call (architecture.md 2.9). It imports the standard library and
// internal/checked only, uses no floating point and draws nothing.
package settlement
