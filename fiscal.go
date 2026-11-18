package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the fiscal module (spec-fiscal.md, FIS; architecture.md 2.10.1): the
// treasury's taxes, duties, tolls, payouts and standing orders. The configuration
// types are in config.go, the validator in fiscal_validate.go and its class rules in
// fiscal_class.go, the warnings in fiscal_warnings.go and the hash in fiscal_hash.go.
// The module is off unless Config.Modules.Fiscal is present, and while it is off it
// leaves no trace (architecture.md 2.10.5).

// fiscalReasons returns the ledger's reason of every flow that moves money or goods
// between accounts, in the canonical order: the taxes, the duties, the tolls and the
// payouts, each list by ID (spec-fiscal.md 3.4.1, plan 3-fis D5). A purchase and a
// release use the reason trade and declare none. The reasons are Internal and open no
// boundary account, so no account ID shifts; fiscal flow j is the ledger's reason
// FlowReason(len(money flows) + j). A payout in kind moves goods and no money. It
// returns nil for a configuration with the module off.
func fiscalReasons(c *Config) []ledger.FiscalFlow {
	m := c.Modules.Fiscal
	if m == nil {
		return nil
	}
	out := make([]ledger.FiscalFlow, 0, len(m.Taxes)+len(m.Duties)+len(m.Tolls)+len(m.Payouts))
	out = append(out, make([]ledger.FiscalFlow, len(m.Taxes)+len(m.Duties)+len(m.Tolls))...)
	for _, p := range m.Payouts {
		out = append(out, ledger.FiscalFlow{Goods: p.Commodity != ""})
	}
	return out
}
