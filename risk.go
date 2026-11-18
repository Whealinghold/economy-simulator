package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the risk module (spec-risk.md): the loss of cargo in transit and of
// stored goods as data. The pure rules are in internal/risk. The file holds, in
// the order of architecture.md 2.10.1: the ledger reason the module declares, then
// its state and its calls. The configuration validator and the warnings are in
// risk_validate.go.

// riskReasons returns the ledger reasons the risk module declares: the one goods
// sink, loss (spec-risk.md 7.1). Its boundary accounts are opened goods
// commodity-major in the order of the Modules pointers (spec-world.md 5.2 row 3a).
func riskReasons() []ledger.Reason {
	return []ledger.Reason{ledger.Loss}
}
