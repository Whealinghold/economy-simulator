package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the insurance module (spec-insurance.md): single-event cover of a cargo in
// transit and of goods in store against the losses of the risk module. The pure rules are
// in internal/insure. The file holds, in the order of architecture.md 2.10.1, what the
// module owns so far: its state and the indexes of the open policies, and the ledger
// reasons it declares. The configuration validator and the warnings are in
// insurance_validate.go, the plan surface (the view, the plan check and the default rules)
// in insurance_strategy.go, the cover round of P3.3c in insurance_round.go and its report
// in insurance_report.go. The settlement of a policy (P1.3a, P8.1b, step 2f) and the
// checkpoint section come with the later changes of the plan.

// insurance is the state of the insurance module, nil when Config.Modules.Insurance is
// nil. The policies are the module's own state (spec-insurance.md 3.3, 16): the agent
// type holds none of it (N-128).
type insurance struct {
	// cfg is the module's configuration of the world's canonical Config.
	cfg *Insurance
	// liability is, by agent ID, the sum of the maximum payouts of the open policies the
	// agent underwrites, and insured the number of open route policies it holds as the
	// insured. Both are derived from the policy lists and grow with the agent IDs; no
	// policy is formed yet, so both are empty.
	liability, insured []int64
	// vs is the storage of the group of the view and round the requests, offers and counts
	// of the cover round. Both are scratch, never saved.
	vs    coverScratch
	round coverRound
}

// newInsurance builds the state of the module for a canonical configuration, or nil when
// the module is off (a nil receiver is the off state everywhere it is read).
func newInsurance(c *Config) *insurance {
	if c.Modules.Insurance == nil {
		return nil
	}
	return &insurance{cfg: c.Modules.Insurance}
}

// insuranceReasons returns the ledger reasons the insurance module declares: the premium
// an insured pays an underwriter and the payout an underwriter pays an insured
// (spec-insurance.md 9.1). Both are internal and money only (FR-INS-002), so neither opens
// a boundary account, and their pinned codes are rows 30 and 31 of the table of
// spec-ledger.md 7.2.
func insuranceReasons() []ledger.Reason {
	return []ledger.Reason{ledger.Premium, ledger.Payout}
}

// policiesOf is the number of open route policies an agent holds as the insured.
func (in *insurance) policiesOf(id AgentID) int64 {
	if in == nil || id < 0 || int(id) >= len(in.insured) {
		return 0
	}
	return in.insured[id]
}

// liabilityOf is the open liability of an underwriter: the sum of the maximum payouts of
// the policies it underwrites (spec-insurance.md 5.2.1). It is nil-safe.
func (in *insurance) liabilityOf(id AgentID) int64 {
	if in == nil || id < 0 || int(id) >= len(in.liability) {
		return 0
	}
	return in.liability[id]
}

// payoutOwed is the principal an agent owes as a debtor on claims of origin payout
// (spec-insurance.md 10.1, Owed): 0 when it owes none.
func (w *World) payoutOwed(account int) int64 {
	if w.credit == nil || account < 0 || account >= len(w.credit.idx.byDebtor) {
		return 0
	}
	var owed int64
	for _, id := range w.credit.idx.byDebtor[account] {
		if c := w.indexed(id); c.Origin == ledger.OriginPayout {
			owed = addSat(owed, c.PrincipalOwed)
		}
	}
	return owed
}
