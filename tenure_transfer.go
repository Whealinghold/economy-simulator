package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// This file is the movement of titles and the calls and queries of the tenure module
// (spec-land.md 6.3, 6.4, 9): the step of a removal, which moves the titles of the
// removed agent to the pool of each endowment's region, TransferTitles, SetRentScales and
// the queries Titles, Holders and Rent.

// dropAgent is step 2b of a removal (spec-agents.md 3.6, spec-land.md 6.4.1): each title
// of the removed agent moves to the pool of the title's endowment region, in ascending
// (RegionID, ResourceID), with one TitleTransferred of cause removal each. No money
// moves. It runs after the contracts of step 2a are settled and before the agent's
// account is emptied, so the event precedes AgentBankrupted.
func (t *tenure) dropAgent(a *agent) {
	w := t.world
	for cell, list := range t.holds {
		share := shareIn(list, a.id)
		if share == 0 {
			continue
		}
		region := t.regionOf(cell)
		pool := AgentID(region)
		// The pool's share is read first: setShare deletes in place and leaves the old
		// list header over a shifted, zeroed tail.
		pooled := shareIn(list, pool) + share
		t.holds[cell] = setShare(setShare(list, a.id, 0), pool, pooled)
		if w.obs.supplied {
			w.obs.deliver(TitleTransferred{
				Tick: w.Tick(), Region: region, Resource: t.resourceOf(cell), From: a.id, To: pool, ShareBP: share,
				Cause: CauseRemoval, FromAfter: 0, ToAfter: t.shareOf(cell, pool),
				FromRegion: w.regionOf(a), ToRegion: region, ToPool: true,
			})
		}
	}
}

// TransferTitles moves shares of endowments between holders now (the tenure module,
// spec-land.md 6.3, 9.2). Each entry moves ShareBP basis points from From to To, either
// of which may be NoAgent for the commons (an enclosure or a release), and a positive
// Price is paid by To to From with the reason title_sale, in the same step. The call
// first checks each entry that needs no state of the titles, in the order of the slice,
// then checks every entry against the titles in the canonical key order (region,
// resource, From, To) on a working copy, and applies none if any fails; the entries are
// applied in canonical order. It
// returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the first entry that fails
// ErrUnknownRegion, ErrOutsideNode, ErrUnknownResource, ErrUnknownAgent,
// ErrAgentRemoved, ErrNotAHolder, ErrOutOfBounds (a share or a price out of its bound,
// or From equal to To), ErrNoTitle (From does not hold the share, or the commons do
// not), ErrTitleCap (To above MaxHolderShareBP, the pool excepted), an error that wraps
// the ledger's ErrInsufficientFunds (To cannot pay) or ErrOverflow (the payee's balance
// would pass MaxMoney), and ErrDuplicateID for a key that appears twice. A refused call
// changes nothing. It emits TitleTransferred, cause consumer, for each entry in
// canonical order.
func (w *World) TransferTitles(changes []TitleTransfer) error {
	if err := w.enter("TransferTitles", "tenure", w.tenure != nil); err != nil {
		return err
	}
	if len(changes) > MaxTitleTransfers {
		return fmt.Errorf("TransferTitles: %w", argError(ErrOutOfBounds, itoa(len(changes)), "MaxTitleTransfers", "changes"))
	}
	t := w.tenure
	for i := range changes {
		if err := t.checkEntry(i, &changes[i]); err != nil {
			return fmt.Errorf("TransferTitles: %w", err)
		}
	}
	order := t.canonicalOrder(changes)
	if d := t.firstDuplicate(changes, order); d >= 0 {
		return fmt.Errorf("TransferTitles: %w", argError(ErrDuplicateID, fmt.Sprintf("%+v", changes[d]), "", entry("changes", d), "region"))
	}
	plan, err := t.simulate(changes, order)
	if err != nil {
		return fmt.Errorf("TransferTitles: %w", err)
	}
	events := t.apply(plan)
	slotNote(w, "titles transferred", events)
	return nil
}

// checkEntry is the check of one entry that needs no state of the titles: the region,
// the resource, the two holders and the bounds of the share and the price.
func (t *tenure) checkEntry(i int, c *TitleTransfer) error {
	w := t.world
	if err := w.slotRegion(i, c.Region, "region", true); err != nil {
		return err
	}
	if c.Resource < 0 || int(c.Resource) >= t.resources {
		return argError(ErrUnknownResource, itoa(int(c.Resource)), "", entry("changes", i), "resource")
	}
	if err := t.checkHolder(i, c.Region, c.From, "from"); err != nil {
		return err
	}
	if err := t.checkHolder(i, c.Region, c.To, "to"); err != nil {
		return err
	}
	return checkBounds(i, c)
}

// checkBounds checks that the two sides differ and that the share and the price are in
// their bounds.
func checkBounds(i int, c *TitleTransfer) error {
	switch {
	case c.From == c.To:
		return argError(ErrOutOfBounds, itoa(int(c.From)), "From and To differ", entry("changes", i), "to")
	case c.ShareBP < 1 || c.ShareBP > wholeBP:
		return argError(ErrOutOfBounds, itoa(int(c.ShareBP)), "10000", entry("changes", i), "share_bp")
	case c.Price < 0 || c.Price > MaxMoney:
		return argError(ErrOutOfBounds, itoa(int(c.Price)), "MaxMoney", entry("changes", i), "price")
	}
	return nil
}

// checkHolder checks that id can hold a title of an endowment of region: NoAgent for
// the commons, the estate pseudo-agent of region for the pool, or a live agent that is
// not a pseudo-agent. The estate of another region and a counterparty or a treasury
// cannot hold one.
func (t *tenure) checkHolder(i int, region RegionID, id AgentID, field string) error {
	w := t.world
	at := func(rule error) error { return argError(rule, itoa(int(id)), "", entry("changes", i), field) }
	switch {
	case id == NoAgent:
		return nil
	case id < 0 || id >= w.state.nextAgent:
		return at(ErrUnknownAgent)
	case id < AgentID(t.regions):
		if id != AgentID(region) {
			return at(ErrNotAHolder)
		}
	case int(id) < w.firstHousehold:
		return at(ErrNotAHolder)
	case w.agents[id].weight < 1:
		return at(ErrAgentRemoved)
	}
	return nil
}

// canonicalOrder returns the indexes of the entries in ascending (region, resource,
// From, To), the order the entries are checked on the working copy and applied in.
func (t *tenure) canonicalOrder(changes []TitleTransfer) []int {
	order := make([]int, len(changes))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return compareTransfer(&changes[a], &changes[b]) })
	return order
}

// compareTransfer orders two entries by their key.
func compareTransfer(a, b *TitleTransfer) int {
	return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Resource, b.Resource),
		cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
}

// firstDuplicate returns the index of the first entry, in input order, that has the key
// of an entry before it, or -1 when every key appears once. order is stable, so of two
// entries with one key the later one follows the earlier in it.
func (t *tenure) firstDuplicate(changes []TitleTransfer, order []int) int {
	dup := -1
	for j := 1; j < len(order); j++ {
		if compareTransfer(&changes[order[j]], &changes[order[j-1]]) == 0 && (dup < 0 || order[j] < dup) {
			dup = order[j]
		}
	}
	return dup
}

// transferStep is one entry after its check on the working copy: the shares after and
// the accounts of the price.
type transferStep struct {
	change             TitleTransfer
	cell               int
	fromAfter, toAfter int64
	payer, payee       int // ledger accounts of the price; -1 when no money moves
}

// transferPlan is the result of the working copy: the steps in canonical order and the
// titles of each endowment the call changes.
type transferPlan struct {
	steps []transferStep
	cells []int
	holds [][]title
}

// delta is the money the entries before one moved on an account, kept to check the
// balances of the entries one after the other.
type delta struct {
	account int
	amount  int64
}

// simulate applies the entries to a working copy of the titles and of the balances in
// canonical order and returns the plan, or the first entry that fails (spec-land.md
// 6.3.4): From holds the share, To stays within the cap, To can pay the price and the
// payee's balance stays within MaxMoney.
func (t *tenure) simulate(changes []TitleTransfer, order []int) (transferPlan, error) {
	var plan transferPlan
	var moves []delta
	for _, i := range order {
		c := &changes[i]
		cell := int(c.Region)*t.resources + int(c.Resource)
		// order is ascending by (region, resource), so the endowment is the last one made or new.
		k := len(plan.cells) - 1
		if k < 0 || plan.cells[k] != cell {
			k = len(plan.cells)
			plan.cells, plan.holds = append(plan.cells, cell), append(plan.holds, slices.Clone(t.holds[cell]))
		}
		step, list, err := t.step(i, c, cell, plan.holds[k], &moves)
		if err != nil {
			return transferPlan{}, err
		}
		plan.holds[k] = list
		plan.steps = append(plan.steps, step)
	}
	return plan, nil
}

// step checks one entry against the working copy of its endowment's titles and returns
// it with the shares after and the titles after it.
func (t *tenure) step(i int, c *TitleTransfer, cell int, list []title, moves *[]delta) (transferStep, []title, error) {
	if err := t.checkMove(i, c, list); err != nil {
		return transferStep{}, nil, err
	}
	st := transferStep{change: *c, cell: cell, payer: -1, payee: -1}
	if c.From != NoAgent {
		list = setShare(list, c.From, shareIn(list, c.From)-c.ShareBP)
		st.fromAfter = shareIn(list, c.From)
	}
	if c.To != NoAgent {
		list = setShare(list, c.To, shareIn(list, c.To)+c.ShareBP)
		st.toAfter = shareIn(list, c.To)
	}
	commons := wholeBP - sumTitles(list)
	if c.From == NoAgent {
		st.fromAfter = commons
	}
	if c.To == NoAgent {
		st.toAfter = commons
	}
	return st, list, t.pay(i, &st, moves)
}

// checkMove checks that From holds the share (or the commons do) and that To stays within
// the cap, the pool excepted.
func (t *tenure) checkMove(i int, c *TitleTransfer, list []title) error {
	at := func(rule error, field string) error {
		return argError(rule, itoa(int(c.ShareBP)), "", entry("changes", i), field)
	}
	commons := wholeBP - sumTitles(list)
	switch {
	case c.From == NoAgent && commons < c.ShareBP, c.From != NoAgent && shareIn(list, c.From) < c.ShareBP:
		return at(ErrNoTitle, "from")
	case c.To != NoAgent && c.To >= AgentID(t.regions) && tnr.OverCap(shareIn(list, c.To)+c.ShareBP, t.capBP):
		return at(ErrTitleCap, "to")
	}
	return nil
}

// sumTitles is the sum of the shares of a list of titles.
func sumTitles(list []title) int64 {
	var s int64
	for _, x := range list {
		s += x.share
	}
	return s
}

// account is the ledger account that pays or is paid the price on the side of a holder:
// the estate account of the endowment's region for the pool and for NoAgent, the
// agent's own account otherwise.
func (t *tenure) account(id AgentID, region RegionID) int {
	if id == NoAgent || id < AgentID(t.regions) {
		return t.world.estates.account(region)
	}
	return t.world.agents[id].account
}

// pay checks the price of a step: the payer, which is To, holds it and the payee, which
// is From, ends within MaxMoney, counting what the entries before moved. A price of 0,
// and a price between one account and itself, moves nothing.
func (t *tenure) pay(i int, st *transferStep, moves *[]delta) error {
	c := &st.change
	payer, payee := t.account(c.To, c.Region), t.account(c.From, c.Region)
	if c.Price == 0 || payer == payee {
		return nil
	}
	price := int64(c.Price)
	pv, err := t.world.ledger.Account(payer)
	if err != nil {
		invariant("the account of a payer of a title is unreadable: " + err.Error())
	}
	qv, err := t.world.ledger.Account(payee)
	if err != nil {
		invariant("the account of a payee of a title is unreadable: " + err.Error())
	}
	if pv.Balance-pv.Reserved+moved(*moves, payer) < price {
		return argError(ledger.ErrInsufficientFunds, itoa(int(price)), "", entry("changes", i), "price")
	}
	if after, ok := checked.Add(qv.Balance+moved(*moves, payee), price); !ok || after > MaxMoney {
		return argError(ErrOverflow, itoa(int(price)), "MaxMoney", entry("changes", i), "price")
	}
	*moves = append(*moves, delta{payer, -price}, delta{payee, price})
	st.payer, st.payee = payer, payee
	return nil
}

// moved is the sum of what the entries before moved on an account.
func moved(moves []delta, account int) int64 {
	var s int64
	for _, m := range moves {
		if m.account == account {
			s += m.amount
		}
	}
	return s
}

// apply makes the plan real: the titles, the prices (reason title_sale) in canonical
// order, and the events, which it returns. The plan was checked on the same state, so a
// refusal of the ledger is a violated invariant.
func (t *tenure) apply(p transferPlan) []TitleTransferred {
	w := t.world
	for k, cell := range p.cells {
		t.holds[cell] = p.holds[k]
	}
	var events []TitleTransferred
	for i := range p.steps {
		st := &p.steps[i]
		if st.payer >= 0 {
			if err := w.ledger.Transfer(st.payer, st.payee, int64(st.change.Price), ledger.TitleSale); err != nil {
				invariant("the ledger refused the price of a title that was checked: " + err.Error())
			}
		}
		if w.obs.supplied {
			events = append(events, t.callEvent(st))
		}
	}
	return events
}

// callEvent is the TitleTransferred of one entry of a call.
func (t *tenure) callEvent(st *transferStep) TitleTransferred {
	c := &st.change
	from, fromPool := t.side(c.From, c.Region)
	to, toPool := t.side(c.To, c.Region)
	return TitleTransferred{
		Tick: t.world.Tick(), Region: c.Region, Resource: c.Resource, From: c.From, To: c.To, ShareBP: c.ShareBP,
		Price: c.Price, Cause: CauseConsumer, FromAfter: st.fromAfter, ToAfter: st.toAfter,
		FromRegion: from, ToRegion: to, FromPool: fromPool, ToPool: toPool,
	}
}

// side is the region of the account that pays or is paid the price on one side of an
// entry, and whether that account is the pool: the pool and NoAgent are the pool of the
// endowment's region, an agent is its account's region.
func (t *tenure) side(id AgentID, region RegionID) (RegionID, bool) {
	if id == NoAgent || id < AgentID(t.regions) {
		return region, true
	}
	return t.world.regionOf(&t.world.agents[id]), false
}

// SetRentScales sets the rent scale of endowments (the tenure module, spec-land.md
// 9.1): the rent in force is the posted rent times the scale, read from the next tick, a
// scale of 0 makes the rent free and a scale of 10 000 clears the key. It never edits
// the configured rent. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the
// first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrUnknownResource,
// ErrOutOfBounds or ErrDuplicateID; a refused call changes nothing. It emits RentScaled
// for each changed key in (region, resource) order, after the whole call is applied.
func (w *World) SetRentScales(changes []EndowmentScale) error {
	if err := w.enter("SetRentScales", "tenure", w.tenure != nil); err != nil {
		return err
	}
	keys, err := slotKeys(changes, func(i int, c EndowmentScale) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		key, err := w.slotEndowment(i, c.Region, c.Resource)
		if err != nil {
			return 0, err
		}
		return key, slotBP(i, c.BP, 0, MaxShockBP, "MaxShockBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("SetRentScales: %w", err)
	}
	t := w.tenure
	var events []RentScaled
	for _, i := range inKeyOrder(keys) {
		c := changes[i]
		cell := w.res.cellOf(c.Region, c.Resource)
		if before := t.scale[cell]; before != c.BP {
			t.scale[cell] = c.BP
			events = append(events, RentScaled{Tick: w.Tick(), Region: c.Region, Resource: c.Resource, From: before, To: c.BP})
		}
	}
	slotNote(w, "rent scales set", events)
	return nil
}

// tenureSlots adds the rent scales that are not the default to the slot list, in
// (region, resource) order; it is empty with the module off.
func (w *World) tenureSlots(s *SlotState) {
	t := w.tenure
	if t == nil {
		return
	}
	for cell, bp := range t.scale {
		if bp != wholeBP {
			s.RentScales = append(s.RentScales, EndowmentScale{Region: t.regionOf(cell), Resource: t.resourceOf(cell), BP: bp})
		}
	}
}

// Titles returns the titles an agent holds, or the estate pseudo-agent's titles in its
// own region, in ascending (region, resource) (the tenure module, spec-land.md 9.3). It
// returns ErrInStep, ErrWorldFailed, ErrModuleOff, ErrUnknownAgent for an ID never
// issued, ErrAgentRemoved for a removed agent and ErrNotAHolder for the estate of an
// outside node, a counterparty or a treasury, as TransferTitles does (6.3.4). It draws nothing, emits nothing and
// returns a copy.
func (w *World) Titles(a AgentID) ([]TitleState, error) {
	if err := w.enter("Titles", "tenure", w.tenure != nil); err != nil {
		return nil, err
	}
	t := w.tenure
	if err := t.queriedHolder(a); err != nil {
		return nil, fmt.Errorf("Titles: agent %d: %w", a, err)
	}
	var out []TitleState
	for cell := range t.holds {
		if share := t.shareOf(cell, a); share > 0 {
			out = append(out, TitleState{Region: t.regionOf(cell), Resource: t.resourceOf(cell), Holder: a, ShareBP: share})
		}
	}
	return out, nil
}

// queriedHolder is the check of the agent of the query Titles: nil for the estate of an
// ordinary region and for a live agent.
func (t *tenure) queriedHolder(a AgentID) error {
	w := t.world
	switch {
	case a < 0 || a >= w.state.nextAgent:
		return ErrUnknownAgent
	case a >= AgentID(t.regions) && int(a) < w.firstHousehold:
		return ErrNotAHolder
	case a < AgentID(t.regions) && w.isOutsideNode(RegionID(a)):
		return ErrNotAHolder
	case a >= AgentID(t.regions) && w.agents[a].weight < 1:
		return ErrAgentRemoved
	}
	return nil
}

// Holders returns the tenure of one endowment: the commons share, whether it is closed,
// the posted rent, the scale and the titles in ascending holder ID (the tenure module,
// spec-land.md 9.3). It returns ErrInStep, ErrWorldFailed, ErrModuleOff,
// ErrUnknownRegion, ErrOutsideNode or ErrUnknownResource. It draws nothing, emits
// nothing and returns a copy.
func (w *World) Holders(region RegionID, resource ResourceID) (HoldingsState, error) {
	cell, err := w.tenureCell("Holders", region, resource)
	if err != nil {
		return HoldingsState{}, err
	}
	t := w.tenure
	st := HoldingsState{CommonsBP: wholeBP - sumTitles(t.holds[cell]), Closed: t.closed[cell], Posted: t.posted[cell], ScaleBP: t.scale[cell]}
	for _, x := range t.holds[cell] {
		st.Titles = append(st.Titles, TitleState{Region: region, Resource: resource, Holder: x.holder, ShareBP: x.share})
	}
	return st, nil
}

// Rent returns the rent in force of one endowment: the posted rent times the scale, in
// money per unit drawn from a let unit (the tenure module, spec-land.md 5.2.2, 9.3). Its
// errors are those of Holders.
func (w *World) Rent(region RegionID, resource ResourceID) (int64, error) {
	cell, err := w.tenureCell("Rent", region, resource)
	if err != nil {
		return 0, err
	}
	return w.tenure.rentInForce(cell), nil
}

// tenureCell checks the arguments of a query of one endowment and returns its dense
// index.
func (w *World) tenureCell(call string, region RegionID, resource ResourceID) (int, error) {
	if err := w.enter(call, "tenure", w.tenure != nil); err != nil {
		return 0, err
	}
	t := w.tenure
	switch {
	case region < 0 || int(region) >= t.regions:
		return 0, fmt.Errorf("%s: region %d: %w", call, region, ErrUnknownRegion)
	case w.isOutsideNode(region):
		return 0, fmt.Errorf("%s: region %d: %w", call, region, ErrOutsideNode)
	case resource < 0 || int(resource) >= t.resources:
		return 0, fmt.Errorf("%s: resource %d: %w", call, resource, ErrUnknownResource)
	}
	return int(region)*t.resources + int(resource), nil
}
