package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
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

// coverPolicy is one cover (spec-insurance.md 3.3), the record of what was fixed at formation.
// A route policy is open from P3.3c of the tick of dispatch until its shipment arrives; a
// store policy lives inside one tick.
type coverPolicy struct {
	id                   int64
	kind                 CoverKind
	insured, underwriter AgentID
	// shipment is the shipment a route policy covers and NoShipment for a store.
	shipment ShipmentID
	// sum is the sum insured, maxPayout the most the policy pays, coverBP the cover share
	// and premium what was paid, all fixed at formation; formed is the tick of formation.
	sum, maxPayout, coverBP, premium, formed int64
}

// insurance is the state of the insurance module, nil when Config.Modules.Insurance is
// nil. The policies are the module's own state (spec-insurance.md 3.3, 16): the agent
// type holds none of it (N-128).
type insurance struct {
	// cfg is the module's configuration of the world's canonical Config.
	cfg *Insurance
	// next is the ID of the next policy, issued in order of formation and never reused.
	next int64
	// open are the route policies in ascending ID, the ones in flight between two ticks;
	// stores are the store policies of the tick, which P8.1b ends in the tick of formation.
	open, stores []coverPolicy
	// byShipment is the ID of the route policy of a shipment, read by lookup only.
	byShipment map[ShipmentID]int64
	// liability is, by agent ID, the sum of the maximum payouts of the open policies the
	// agent underwrites, and insured the number of open route policies it holds as the
	// insured. Both are derived from the policy lists and grow with the agent IDs.
	liability, insured []int64
	// tot are the cumulative totals of the module (spec-insurance.md 9.3).
	tot coverTotals
	// arrived are the arrivals of the tick that have a route policy and amounts the units
	// they lost, kept from P1.3 to P1.3a (hook H-11); both are emptied at P1.3a.
	arrived []coverArrival
	amounts []Amount
	// vs is the storage of the group of the view, round the requests, offers and counts of
	// the cover round and st the items of a settlement. All are scratch, never saved.
	vs    coverScratch
	round coverRound
	st    coverSettle
}

// newInsurance builds the state of the module for a canonical configuration, or nil when
// the module is off (a nil receiver is the off state everywhere it is read).
func newInsurance(c *Config) *insurance {
	if c.Modules.Insurance == nil {
		return nil
	}
	n := len(c.Regions)
	in := &insurance{cfg: c.Modules.Insurance, byShipment: map[ShipmentID]int64{}}
	in.round.lanes, in.round.counts = make([]coverLane, n), make([]coverCount, n)
	return in
}

// insuranceReasons returns the ledger reasons the insurance module declares: the premium
// an insured pays an underwriter and the payout an underwriter pays an insured
// (spec-insurance.md 9.1). Both are internal and money only (FR-INS-002), so neither opens
// a boundary account, and their pinned codes are rows 30 and 31 of the table of
// spec-ledger.md 7.2.
func insuranceReasons() []ledger.Reason {
	return []ledger.Reason{ledger.Premium, ledger.Payout}
}

// openCount is the number of policies open now: the route policies in flight and the
// store policies formed this tick (the count MaxPolicies limits).
func (in *insurance) openCount() int64 { return int64(len(in.open) + len(in.stores)) }

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

// covered reports whether the shipment already has a route policy.
func (in *insurance) covered(s ShipmentID) bool {
	_, ok := in.byShipment[s]
	return ok
}

// add appends a policy just formed to its list and the indexes. A route policy joins the
// list in ID order because the IDs rise with formation.
func (in *insurance) add(p coverPolicy) {
	in.liability = bump(in.liability, p.underwriter, p.maxPayout)
	if p.kind == CoverStore {
		in.stores = append(in.stores, p)
		return
	}
	in.insured = bump(in.insured, p.insured, 1)
	in.byShipment[p.shipment] = p.id
	in.open = append(in.open, p)
}

// bump adds d to the entry of an agent, growing the slice to hold it.
func bump(s []int64, id AgentID, d int64) []int64 {
	for int(id) >= len(s) {
		s = append(s, 0)
	}
	s[id] += d
	return s
}

// route returns the open route policy with an ID, found by binary search.
func (in *insurance) route(id int64) (*coverPolicy, bool) {
	i, found := slices.BinarySearchFunc(in.open, id, func(p coverPolicy, id int64) int { return cmp.Compare(p.id, id) })
	if !found {
		return nil, false
	}
	return &in.open[i], true
}

// end removes a policy from its list and the indexes, whatever the cause: its liability
// leaves the underwriter and a route policy leaves the shipment index and the insured's
// count. The lists keep their ascending order.
func (in *insurance) end(p *coverPolicy) {
	in.liability[p.underwriter] -= p.maxPayout
	list := &in.open
	if p.kind == CoverStore {
		list = &in.stores
	} else {
		in.insured[p.insured]--
		delete(in.byShipment, p.shipment)
	}
	if i, found := slices.BinarySearchFunc(*list, p.id, func(q coverPolicy, id int64) int { return cmp.Compare(q.id, id) }); found {
		*list = slices.Delete(*list, i, i+1)
	}
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

// openAgentAccount reports whether the ID is a live agent whose account is an open Agent
// account of the ledger.
func (w *World) openAgentAccount(id AgentID) bool {
	if !w.liveAgent(id) {
		return false
	}
	v, err := w.ledger.Account(w.agents[id].account)
	return err == nil && v.Kind == ledger.KindAgent && !v.Closed
}

// shipmentInFlight returns the shipment with the ID when it is in flight, by binary search of
// the ships, which are in ascending ID order. It reports false with the transport module off.
func (w *World) shipmentInFlight(id ShipmentID) (*shipment, bool) {
	if w.transport == nil {
		return nil, false
	}
	i, ok := slices.BinarySearchFunc(w.transport.ships, id, func(s shipment, id ShipmentID) int { return cmp.Compare(s.id, id) })
	if !ok {
		return nil, false
	}
	return &w.transport.ships[i], true
}

// checkOpenTotals checks the totals against the open route policies: the premiums paid are at
// least the premiums of the open policies and the policies formed at least their count.
func (in *insurance) checkOpenTotals() error {
	var premiums int64
	for i := range in.open {
		premiums = addSat(premiums, in.open[i].premium)
	}
	if in.tot.premiums < premiums || in.tot.formed < int64(len(in.open)) {
		return fmt.Errorf("%d premiums over %d policies formed, want at least the %d of the %d open policies",
			in.tot.premiums, in.tot.formed, premiums, len(in.open))
	}
	return nil
}

// audit checks the identities of spec-insurance.md 9.4 (FR-INS-081): the due is the sum of
// its parts over the run with the shortfall the ledger's Awarded, the policies formed are the
// next ID, every open route policy has a live shipment and open distinct accounts, and the
// indexes are the ones the lists give. The money identity is the ledger's (FR-LED-024).
func (in *insurance) audit(w *World) error {
	err := in.auditTotals(w)
	if err == nil {
		err = in.auditPolicies(w)
	}
	if err == nil {
		err = in.auditIndexes()
	}
	if err != nil {
		return fmt.Errorf("insurance: %w", err)
	}
	return nil
}

// auditTotals checks identity 1 and the counts of the totals. The identity is not checked
// once a total has reached MaxMoney, where the sums saturate.
func (in *insurance) auditTotals(w *World) error {
	t := &in.tot
	short := w.ledger.ClaimTotals().Awarded
	if t.formed != in.next {
		return fmt.Errorf("%d policies formed, want the next policy ID %d", t.formed, in.next)
	}
	if t.due < MaxMoney && t.due != t.paid+short+t.dropped+t.cut {
		return fmt.Errorf("due %d is not paid %d + short %d + dropped %d + cut %d", t.due, t.paid, short, t.dropped, t.cut)
	}
	return in.checkOpenTotals()
}

// auditPolicies checks identity 4 for each open route policy: ascending IDs below the next,
// and what policyFault reads.
func (in *insurance) auditPolicies(w *World) error {
	for i := range in.open {
		p := &in.open[i]
		if p.id >= in.next || (i > 0 && p.id <= in.open[i-1].id) {
			return fmt.Errorf("policy %d is out of order or not below the next ID %d", p.id, in.next)
		}
		if why := w.policyFault(p); why != "" {
			return fmt.Errorf("policy %d %s", p.id, why)
		}
	}
	return nil
}

// auditIndexes checks that the by-shipment index, the open liability of each underwriter and
// the count of each insured are what the lists give, store policies included.
func (in *insurance) auditIndexes() error {
	liability, insured, err := in.indexesOfLists()
	if err != nil {
		return err
	}
	if !slices.Equal(liability, in.liability) || !slices.Equal(insured, in.insured) || len(in.byShipment) != len(in.open) {
		return fmt.Errorf("the indexes (%d liabilities, %d insured, %d by shipment) are not the lists' (%d open policies)",
			len(in.liability), len(in.insured), len(in.byShipment), len(in.open))
	}
	for i := range in.open {
		if in.byShipment[in.open[i].shipment] != in.open[i].id {
			return fmt.Errorf("the by-shipment index does not name policy %d for shipment %d", in.open[i].id, in.open[i].shipment)
		}
	}
	return nil
}

// indexesOfLists recomputes the open liability of each underwriter and the count of route
// policies of each insured from the two lists, in slices of the length the indexes have.
func (in *insurance) indexesOfLists() (liability, insured []int64, err error) {
	liability, insured = make([]int64, len(in.liability)), make([]int64, len(in.insured))
	for _, list := range [...][]coverPolicy{in.open, in.stores} {
		for i := range list {
			p := &list[i]
			if p.underwriter < 0 || int(p.underwriter) >= len(liability) {
				return nil, nil, fmt.Errorf("policy %d: underwriter %d has no liability entry", p.id, p.underwriter)
			}
			sum, ok := checked.Add(liability[p.underwriter], p.maxPayout)
			if !ok {
				return nil, nil, fmt.Errorf("policy %d: the open liability of underwriter %d overflows", p.id, p.underwriter)
			}
			liability[p.underwriter] = sum
			if p.kind == CoverRoute {
				if p.insured < 0 || int(p.insured) >= len(insured) {
					return nil, nil, fmt.Errorf("policy %d: insured %d has no count entry", p.id, p.insured)
				}
				insured[p.insured]++
			}
		}
	}
	return liability, insured, nil
}
