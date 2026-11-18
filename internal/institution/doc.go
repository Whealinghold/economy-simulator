// Package institution holds the pure rules of the institutions module over
// primitive values (spec-institutions.md 3 to 6): the resolution of a table keyed
// by a subject and a scope key over a chain, the overlap of two scopes, the legal
// range of a price in minor units, the clamp of a limit price, the lapse and the
// admission of a membership round, and the headroom of a place count and the
// least of two limits. It knows no body, right, role, region or commodity by name:
// a list, a scope and a subject are indexes, and the root converts at the call
// (architecture.md 2.9). It imports the standard library, internal/checked and
// internal/ordering only, uses no floating point, draws nothing and keeps no
// state: every function is a pure function of its arguments.
package institution
