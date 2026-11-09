package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// This file is the second half of the merchants module (spec-merchants.md 9.7 to
// 9.8, 15.4, 16): the settlement of the contracts whose shipment arrived (P1.4), the
// step of a removal, the open_contract refusal, the audit, the snapshot rows and
// the checkpoint section of tag 21. The formation is contract.go.

// openContract is the name of the region_bound refusal of the module
// (spec-regions.md 6.5.2, spec-merchants.md 9.8.1). Its instance key is the
// contract ID.
const openContract = "open_contract"

// ContractRow is a contract in flight in the snapshot (spec-merchants.md 15.4,
// spec-observability.md 5.3.8): the record of section 16 with the time left. Region
// is the origin, the deliverer's region at the dispatch, and Escrowed is Price times
// Quantity.
type ContractRow struct {
	Contract         ContractID
	Payer, Deliverer AgentID
	Region, To       RegionID
	Commodity        CommodityID
	Quantity         Quantity
	Price            Price
	Escrowed         Money
	Shipment         ShipmentID
	Formed, Arrives  int64
	TicksLeft        int64
}

// holds reports whether the agent is the payer or the deliverer of an open
// contract that skip does not name by instance (spec-merchants.md 9.8.1).
func (m *merchants) holds(a *agent, skip []string) bool {
	for i := range m.open {
		c := &m.open[i]
		if (c.payer == a.id || c.deliverer == a.id) && !skipHas(skip, openContract, int64(c.id)) {
			return true
		}
	}
	return false
}

// arrivalSkips returns the instance keys the guard query of an arrival skips besides
// the arriving shipment's own: the contract formed on that shipment (spec-transport.md
// 5.5.3, ruling C16), so a deliverer that arrives at P1.3 while its contract settles
// at P1.4 is not held by it. Every other contract of the agent still refuses.
func (w *World) arrivalSkips(s *shipment, _ RegionID) []string {
	if w.mer == nil {
		return nil
	}
	for i := range w.mer.open {
		if c := &w.mer.open[i]; c.shipment == s.id {
			return []string{openContract + ":" + strconv.FormatInt(int64(c.id), 10)}
		}
	}
	return nil
}

// p14Settle is P1.4 (spec-merchants.md 9.7): the contracts whose shipment arrived in
// P1.3 are settled in ascending contract ID, after the ShipmentArrived events. The
// arrivals are in ascending shipment ID, so each contract finds its own by search.
// The error is the overflow of a balance, which fails the step like a trade's.
func (w *World) p14Settle() error {
	m, t := w.mer, w.transport
	if len(m.open) == 0 || len(t.arrived) == 0 {
		return nil
	}
	keep := m.open[:0]
	for _, ct := range m.open {
		i, ok := slices.BinarySearchFunc(t.arrived, ct.shipment, func(a arrival, id ShipmentID) int { return cmp.Compare(a.shipment, id) })
		if !ok {
			keep = append(keep, ct)
			continue
		}
		if err := w.settleArrived(&ct, t.arrivedLines[t.arrived[i].first]); err != nil {
			return err // the step fails, and a failed world is never observed or saved
		}
	}
	m.open = keep
	return nil
}

// settleArrived settles a contract at the arrival of its shipment, whose one line
// is ln: the units that reached the destination, delivered to the payer or diverted
// to the estate, are paid to the deliverer, and the units spoiled are refunded to
// the payer or, when it was removed, to the estate of the destination (9.7). The
// shipment record has no loss yet, so Lost is 0 (the risk module is plan 2-18's).
func (w *World) settleArrived(ct *contract, ln ArrivedLine) error {
	cause := CauseDelivered
	switch {
	case !w.liveAgent(ct.payer):
		cause = CausePayerRemoved
	case ln.Spoiled > 0:
		cause = CauseShort
	}
	return w.endContract(ct, ln.Delivered+ln.Diverted, ln.Spoiled, cause)
}

// endContract pays the deliverer price * reached out of the escrow, returns the rest
// to the payer or the estate of the destination, releases the Escrow account and
// reports the end of the contract. The deliverer is live: a contract ends at the
// removal of its deliverer, before its account closes.
func (w *World) endContract(ct *contract, reached, spoiled Quantity, cause Cause) error {
	escrowed := int64(ct.price) * int64(ct.qty)
	paid := int64(0) // a removal of the deliverer refunds the whole escrow
	if cause != CauseDelivererRemoved {
		var ok bool
		if paid, _, ok = merchant.Settle(int64(ct.price), int64(ct.qty), int64(reached), int64(spoiled), 0); !ok {
			invariant("settle a contract: the units reached and spoiled do not add up to the quantity")
		}
	}
	if paid > 0 {
		if err := w.ledger.Transfer(ct.escrow, w.agents[ct.deliverer].account, paid, ledger.EscrowOut); err != nil {
			return refused("pay a contract", ct.to, ct.escrow, err)
		}
	}
	if back := escrowed - paid; back > 0 {
		to := w.estates.accounts[ct.to]
		if w.liveAgent(ct.payer) {
			to = w.agents[ct.payer].account
		}
		if err := w.ledger.Transfer(ct.escrow, to, back, ledger.EscrowRefund); err != nil {
			return refused("refund a contract", ct.to, ct.escrow, err)
		}
	}
	w.mer.free[ct.to].Put(ct.escrow)
	w.announceSettled(ct, reached, spoiled, Money(paid), Money(escrowed-paid), cause)
	return nil
}

// announceSettled emits ContractSettled and writes the Debug line of the end of a
// contract. The event's Region is the deliverer's account region at the event.
func (w *World) announceSettled(ct *contract, reached, spoiled Quantity, paid, refunded Money, cause Cause) {
	if w.obs.supplied {
		w.obs.deliver(ContractSettled{
			Tick: w.Tick(), Contract: ct.id, Payer: ct.payer, Deliverer: ct.deliverer, Region: w.regionOf(&w.agents[ct.deliverer]),
			To: ct.to, Commodity: ct.commodity, Quantity: ct.qty, Reached: reached, Spoiled: spoiled,
			Paid: paid, Refunded: refunded, Cause: cause,
		})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "contract settled",
			slog.Int64(logkeys.Contract, int64(ct.id)), slog.String(logkeys.Reason, string(cause)))
	}
}

// dropContracts is the step of a removal (spec-merchants.md 9.8.2, spec-agents.md 3.6):
// each open contract of which the agent is the deliverer ends, in ascending contract
// ID, with the whole escrow returned to the payer, or to the estate of the destination
// when the payer is gone; the shipment is not touched. A payer's contracts stand.
func (w *World) dropContracts(a *agent) error {
	m := w.mer
	if m == nil || !m.on {
		return nil
	}
	keep := m.open[:0]
	for _, ct := range m.open {
		if ct.deliverer != a.id {
			keep = append(keep, ct)
			continue
		}
		if err := w.endContract(&ct, 0, 0, CauseDelivererRemoved); err != nil {
			return err
		}
	}
	m.open = keep
	return nil
}

// auditContracts is the contracts' part of the audit (spec-merchants.md FR-MER-045):
// the IDs are unique, ascending and below the next; each contract's Escrow account is
// an open Escrow account of its destination region that no other contract uses and
// holds exactly price * quantity; its shipment is in flight; and every other open
// Escrow account is empty. The error names the check and wraps ErrAuditFailed.
func (w *World) auditContracts() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("audit at tick %d: %w: contracts: %s", w.Tick(), ErrAuditFailed, fmt.Sprintf(format, args...))
	}
	used := make(map[int]bool, len(w.mer.open))
	for i := range w.mer.open {
		if msg := w.contractFault(i, used); msg != "" {
			return fail("contract %d: %s", w.mer.open[i].id, msg)
		}
		if msg := w.shipmentFault(&w.mer.open[i]); msg != "" {
			return fail("contract %d: %s", w.mer.open[i].id, msg)
		}
	}
	if msg := w.freeEscrowFault(used); msg != "" {
		return fail("%s", msg)
	}
	return nil
}

// contractFault returns what is wrong with the order of the open contract at index
// i or with its Escrow account, or "". It marks the account in used.
func (w *World) contractFault(i int, used map[int]bool) string {
	m := w.mer
	ct := &m.open[i]
	switch {
	case i > 0 && ct.id <= m.open[i-1].id:
		return fmt.Sprintf("it is not above contract %d", m.open[i-1].id)
	case ct.id >= m.next:
		return fmt.Sprintf("it is not below the next contract %d", m.next)
	case used[ct.escrow]:
		return fmt.Sprintf("account %d is used by two contracts", ct.escrow)
	}
	used[ct.escrow] = true
	v, err := w.ledger.Account(ct.escrow)
	switch {
	case err != nil || v.Kind != ledger.KindEscrow || v.Closed || v.Region != int(ct.to):
		return fmt.Sprintf("account %d is not an open Escrow account of region %d", ct.escrow, ct.to)
	case v.Balance != int64(ct.price)*int64(ct.qty):
		return fmt.Sprintf("account %d holds %d, want %d", ct.escrow, v.Balance, int64(ct.price)*int64(ct.qty))
	}
	return ""
}

// shipmentFault returns what is wrong with the shipment of an open contract: it is
// in flight, and is the shipment the contract was formed on (the deliverer ships,
// the payer receives the one line to the destination, in the tick of the formation).
func (w *World) shipmentFault(ct *contract) string {
	j, ok := slices.BinarySearchFunc(w.transport.ships, ct.shipment, func(s shipment, id ShipmentID) int { return cmp.Compare(s.id, id) })
	if !ok {
		return fmt.Sprintf("shipment %d is not in flight", ct.shipment)
	}
	if s := &w.transport.ships[j]; s.shipper != ct.deliverer || s.consignee != ct.payer || s.to != ct.to || s.dispatched != ct.formed ||
		len(s.lines) != 1 || s.lines[0] != (Line{Commodity: ct.commodity, Quantity: ct.qty}) {
		return fmt.Sprintf("shipment %d is not the shipment it was formed on", ct.shipment)
	}
	return ""
}

// freeEscrowFault returns the first open Escrow account that no contract uses and
// that is not empty, or "".
func (w *World) freeEscrowFault(used map[int]bool) string {
	for id := range w.ledger.Len() {
		v, err := w.ledger.Account(id)
		if err != nil {
			return fmt.Sprintf("the ledger refused account %d: %v", id, err)
		}
		if v.Kind == ledger.KindEscrow && !v.Closed && !used[id] && v.Balance != 0 {
			return fmt.Sprintf("free Escrow account %d holds %d", id, v.Balance)
		}
	}
	return ""
}

// snapshotContracts fills the contract rows, in ascending contract ID, the money in
// escrow in total and by destination region, and adds it to the money of the world
// (spec-merchants.md 15.4, 9.6.3). It leaves the snapshot alone with the module off.
func (w *World) snapshotContracts(s *Snapshot) {
	if w.mer == nil || !w.mer.on {
		return
	}
	for i := range w.mer.open {
		ct := &w.mer.open[i]
		sh := w.shipmentOf(ct.shipment)
		held := Money(int64(ct.price) * int64(ct.qty))
		s.Contracts = append(s.Contracts, ContractRow{
			Contract: ct.id, Payer: ct.payer, Deliverer: ct.deliverer, Region: sh.from, To: ct.to, Commodity: ct.commodity,
			Quantity: ct.qty, Price: ct.price, Escrowed: held, Shipment: ct.shipment, Formed: ct.formed,
			Arrives: sh.arrives, TicksLeft: sh.arrives - w.Tick(),
		})
		s.Escrow = satMoney(s.Escrow, held, 0)
		s.Regions[ct.to].MoneyContract = satMoney(s.Regions[ct.to].MoneyContract, held, 0)
		s.Money = satMoney(s.Money, held, 0)
	}
}

// saveMerchants is tag 21 (spec-checkpoint.md 4.2, spec-merchants.md 16): the next
// contract ID, then the contracts in flight in ID order, each with its payer,
// deliverer, shipment, commodity, quantity, price, Escrow account, the tick of its
// formation and its destination. The free list of Escrow accounts, the fee shares
// and the scratch are derived and not saved.
func (w *World) saveMerchants(cw *codec.Writer) error {
	m := w.mer
	cw.U(unsigned(m.next))
	cw.Count(len(m.open))
	for i := range m.open {
		c := &m.open[i]
		cw.U(unsigned(c.id))
		cw.U(unsigned(c.payer))
		cw.U(unsigned(c.deliverer))
		cw.U(unsigned(c.shipment))
		cw.U(unsigned(c.commodity))
		cw.U(unsigned(c.qty))
		cw.U(unsigned(c.price))
		cw.U(unsigned(c.escrow))
		cw.U(unsigned(c.formed))
		cw.U(unsigned(c.to))
	}
	return nil
}

// loadMerchants reads tag 21: the next contract ID and the contracts with strictly
// ascending IDs below it, at most MaxOpen of them, each with bounded fields. That
// the shipment, the Escrow account and the amount it holds are what the record says
// is the cross-check merchantsCross and the audit of the restore.
func (w *World) loadMerchants(rs *restoreCtx, r *codec.Reader) error {
	m := w.mer
	m.next = ContractID(r.Bounded("next contract", maxID))
	for i, n := 0, r.Count(10); i < n && r.Err() == nil; i++ {
		c := w.readContract(rs, r)
		if r.Err() != nil {
			return nil
		}
		if msg := w.contractShape(&c, i); msg != "" {
			r.Failf("contract %d %s", c.id, msg)
		}
		m.open = append(m.open, c)
	}
	return nil
}

// contractShape returns what is wrong with the record of the i-th contract read, or
// "": its ID is not above the one before it and below the next, there are more than
// MaxOpen, a party is no agent or both are one agent, or it has no goods or price.
func (w *World) contractShape(c *contract, i int) string {
	m := w.mer
	switch {
	case c.id >= m.next || (i > 0 && c.id <= m.open[i-1].id):
		return fmt.Sprintf("is not in strictly ascending order below the next contract %d", m.next)
	case int64(i) >= m.maxOpen:
		return fmt.Sprintf("is more than MaxOpen %d contracts", m.maxOpen)
	case int(c.payer) < len(w.cfg.Regions) || int(c.deliverer) < len(w.cfg.Regions) || c.payer == c.deliverer || c.qty < 1 || c.price < 1:
		return "has a party that is no agent, one agent for both, or no goods or price"
	}
	return ""
}

// readContract reads one contract record of tag 21 with the bound of each field.
func (w *World) readContract(rs *restoreCtx, r *codec.Reader) contract {
	next := int64(w.state.nextAgent)
	c := contract{id: ContractID(r.Bounded("contract", maxID))}
	c.payer = AgentID(r.Bounded("payer", next-1))
	c.deliverer = AgentID(r.Bounded("deliverer", next-1))
	c.shipment = ShipmentID(r.Bounded("shipment", maxID))
	c.commodity = CommodityID(r.Index(uint64(len(w.cfg.Commodities))))
	c.qty = Quantity(r.Bounded("quantity", MaxQuantity))
	c.price = Price(r.Bounded("price", MaxPrice))
	c.escrow = int(r.Bounded("escrow account", int64(w.ledger.Len())-1))
	c.formed = r.Bounded("formation tick", rs.tick-1)
	c.to = RegionID(r.Index(uint64(len(w.cfg.Regions))))
	return c
}

// merchantsCross is the module's part of step 7: each contract's shipment is in
// flight and is the one it was formed on, its Escrow account is an open Escrow
// account of the destination that holds the price of the goods and no account serves
// two contracts; and the free list of each region is every open Escrow account of the
// region that no contract names.
func (w *World) merchantsCross() error {
	used := make(map[int]bool, len(w.mer.open))
	for i := range w.mer.open {
		ct := &w.mer.open[i]
		msg := w.contractFault(i, used)
		if msg == "" {
			msg = w.shipmentFault(ct)
		}
		if msg != "" {
			return fmt.Errorf("%w: contract %d: %s", ErrCheckpointCorrupt, ct.id, msg)
		}
	}
	free := make([][]int, len(w.mer.free))
	for id := range w.ledger.Len() {
		if v, err := w.ledger.Account(id); err == nil && v.Kind == ledger.KindEscrow && !v.Closed && !used[id] {
			free[v.Region] = append(free[v.Region], id)
		}
	}
	for g := range w.mer.free {
		w.mer.free[g].Rebuild(free[g])
	}
	return nil
}
