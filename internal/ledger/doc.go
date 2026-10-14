// Package ledger holds all the money and goods of a world and is the only
// place that changes them (spec-ledger.md). Every change is a transfer
// between two accounts under a reason from a closed registry, so the total
// over all accounts, boundaries included, is always exactly zero.
//
// The package is standard library only and never imports the root: it sees
// accounts, commodity indexes, reasons and numbers (architecture.md 2.9), and
// receives what it needs about commodities as a table of primitives. All
// state is slices of structs; every addition is checked; an operation applies
// completely or changes nothing; and the package logs and emits nothing, so
// a refusal is a returned error that its one caller handles.
//
// Audit checks every identity of the ledger (zero totals, non-negativity,
// canonical form, conservation by reason, the sectoral identity and the claim
// identities) and returns an *AuditError for the first that fails. It reads
// and changes nothing, and is the package's one diagnostic: a returned error,
// never a log line.
//
// A Ledger is built in its construction phase, and the world ends it when the
// first step begins. It is not safe for concurrent use.
package ledger
