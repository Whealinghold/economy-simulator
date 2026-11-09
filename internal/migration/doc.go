// Package migration holds the rules of the migration module over primitive
// numbers (spec-migration.md): the journey time and the horizon discount, the
// review rotation, settling, the push gate, the margin of a move, the estimate of
// a destination from rumoured prices, the stages and the ranking of the
// candidates, and the admission of the ranked households against the caps and the
// provisioning. It knows no region, commodity, role or agent: the root converts
// at the call and keeps the references it needs (architecture.md 2.9). Every
// product is checked or held in 128 bits and an overflow is reported as ok false,
// never wrapped; there is no floating point and no random source.
package migration
