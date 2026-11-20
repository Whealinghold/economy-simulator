package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the insurance module (spec-insurance.md): single-event cover of a
// cargo in transit and of goods in store against the losses of the risk module.
// The pure rules are in internal/insure. The file holds, in the order of
// architecture.md 2.10.1, what the module owns so far: the ledger reasons it
// declares. Its state, round and settlement come with the later changes of the
// plan, and its configuration validator and warnings are in insurance_validate.go.

// insuranceReasons returns the ledger reasons the insurance module declares: the
// premium an insured pays an underwriter and the payout an underwriter pays an
// insured (spec-insurance.md 9.1). Both are internal and money only (FR-INS-002), so
// neither opens a boundary account, and their pinned codes are rows 30 and 31 of the
// table of spec-ledger.md 7.2.
func insuranceReasons() []ledger.Reason {
	return []ledger.Reason{ledger.Premium, ledger.Payout}
}
