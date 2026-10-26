package ledger

import (
	"fmt"
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// None is the Account or Commodity of an AuditError that names no account or
// no commodity (Commodity is Money for money, and Claim is -1 for none).
const None = -2

// Identity names one identity of the audit (spec-ledger.md 9.1, 9.3, 10;
// spec-credit.md 3.6), in the order Audit checks them.
type Identity uint8

// The identities of the audit.
const (
	// IdentityStructure is the shape of the state: accounts, the boundary
	// table, bounds and the flow totals.
	IdentityStructure Identity = iota
	// IdentityNonNegative is the non-negativity of 9.1 item 2, and that
	// reserved is at most the balance.
	IdentityNonNegative
	// IdentityCanonical is the canonical form of lots (5.1.3, 9.1 item 4).
	IdentityCanonical
	// IdentityHint is that no earliest-expiry hint is later than the truth
	// (5.3.2).
	IdentityHint
	// IdentityClaim is the claim list: order, ranges, parties (CRD 3.6 items
	// 3 and 4).
	IdentityClaim
	// IdentityZeroTotal is the zero total of 9.1 item 1.
	IdentityZeroTotal
	// IdentityConservation is the conservation by reason of 9.1 item 3,
	// the construction bucket included (FR-LED-046).
	IdentityConservation
	// IdentitySectoral is the sectoral identity of 9.3.
	IdentitySectoral
	// IdentityClaimTotals is the pair of CRD 3.6 item 1, with Awarded.
	IdentityClaimTotals
)

var identityNames = [...]string{
	IdentityStructure:    "structure",
	IdentityNonNegative:  "non-negativity",
	IdentityCanonical:    "canonical form",
	IdentityHint:         "expiry hint",
	IdentityClaim:        "claim",
	IdentityZeroTotal:    "zero total",
	IdentityConservation: "conservation by reason",
	IdentitySectoral:     "sectoral identity",
	IdentityClaimTotals:  "claim totals",
}

// String returns the name of the identity.
func (i Identity) String() string {
	if int(i) < len(identityNames) {
		return identityNames[i]
	}
	return fmt.Sprintf("identity %d", uint8(i))
}

// AuditError is the first identity Audit found false, with the account,
// commodity or claim it names. It matches ErrAuditFailed with errors.Is.
type AuditError struct {
	Identity Identity
	// Account is the account ID, or None. For conservation by reason it is
	// the boundary account of the reason.
	Account int
	// Commodity is Money for money, a commodity index, or None.
	Commodity int
	// Claim is the ID of the first claim that fails (FR-LED-064), or -1.
	Claim int64
	// Detail says what was found.
	Detail string
}

// Error implements error.
func (e *AuditError) Error() string {
	s := "ledger: audit failed: " + e.Identity.String()
	if e.Account != None {
		s += fmt.Sprintf(", account %d", e.Account)
	}
	switch e.Commodity {
	case None:
	case Money:
		s += ", money"
	default:
		s += fmt.Sprintf(", commodity %d", e.Commodity)
	}
	if e.Claim >= 0 {
		s += fmt.Sprintf(", claim %d", e.Claim)
	}
	return s + ": " + e.Detail
}

// Unwrap lets errors.Is match ErrAuditFailed.
func (e *AuditError) Unwrap() error { return ErrAuditFailed }

// auditFail builds the AuditError of a failed identity that names no claim.
func auditFail(id Identity, account, commodity int, format string, args ...any) error {
	return &AuditError{Identity: id, Account: account, Commodity: commodity, Claim: -1, Detail: fmt.Sprintf(format, args...)}
}

// auditClaimFail builds the AuditError of a failed claim identity.
func auditClaimFail(account int, claim int64, format string, args ...any) error {
	return &AuditError{Identity: IdentityClaim, Account: account, Commodity: None, Claim: claim, Detail: fmt.Sprintf(format, args...)}
}

// wide is a signed 128-bit sum, so that a total of int64 balances cannot wrap.
type wide struct {
	hi int64
	lo uint64
}

func (w wide) add(v int64) wide {
	lo, c := bits.Add64(w.lo, uint64(v), 0)
	return wide{w.hi + int64(c) + v>>63, lo}
}

func (w wide) plus(o wide) wide {
	lo, c := bits.Add64(w.lo, o.lo, 0)
	return wide{w.hi + o.hi + int64(c), lo}
}

func (w wide) neg() wide {
	lo, c := bits.Add64(^w.lo, 1, 0)
	return wide{^w.hi + int64(c), lo}
}

// invariantViolated panics for a violated internal invariant found outside
// an audit: a programmer error, not an input (spec-ledger.md 12.2,
// architecture.md 8.1).
func invariantViolated(what string) {
	panic("ledger: programmer error: violated internal invariant: " + what)
}

// HasReservation reports whether any account holds a reservation of money or
// an earmark of goods. The world asserts it is false at a tick boundary
// (FR-LED-020).
func (l *Ledger) HasReservation() bool {
	for i := range l.accounts {
		if a := &l.accounts[i]; a.reserved != 0 || a.earmarked() {
			return true
		}
	}
	return false
}

// Audit checks the identities of spec-ledger.md 9.1, 9.3 and 10 and the claim
// identities of spec-credit.md 3.6, and returns an *AuditError for the first
// that fails, or nil. It reads and changes nothing, and is meant for a tick
// boundary or a world between ticks. The order is: each account in ID order
// (structure, non-negativity, canonical form, hint), the flow totals, each
// open claim in ID order, the zero totals, the conservation of each boundary
// account by reason, the sectoral identity, and the claim totals. Exactness:
// every comparison is by equality, in 128-bit sums.
func (l *Ledger) Audit() error {
	for id := range l.accounts {
		if err := l.auditAccount(id, &l.accounts[id]); err != nil {
			return err
		}
	}
	if err := l.auditCounters(); err != nil {
		return err
	}
	if err := l.auditClaims(); err != nil {
		return err
	}
	if err := l.auditZeroTotals(); err != nil {
		return err
	}
	if err := l.auditByReason(); err != nil {
		return err
	}
	if err := l.auditSectors(); err != nil {
		return err
	}
	return l.auditClaimTotals()
}

// auditAccount checks one slot of the account table.
func (l *Ledger) auditAccount(id int, a *account) error {
	if a.kind >= numKinds {
		return auditFail(IdentityStructure, id, None, "kind %d is not a kind", a.kind)
	}
	if a.closed {
		if closedHoldsState(a) {
			return auditFail(IdentityStructure, id, None, "a closed account holds state, or is a boundary")
		}
		return nil
	}
	if a.kind == KindBoundary {
		return l.auditBoundary(id, a)
	}
	if err := l.auditMoney(id, a); err != nil {
		return err
	}
	return l.auditLots(id, a)
}

// closedHoldsState reports whether a closed account is not wholly zeroed.
func closedHoldsState(a *account) bool {
	return a.balance != 0 || a.reserved != 0 || len(a.lots) != 0 || a.hasHint || a.owner != 0 || a.region != 0 || a.kind == KindBoundary
}

// auditMoney checks the region and the money balance of a non-boundary account.
func (l *Ledger) auditMoney(id int, a *account) error {
	switch {
	case a.region < 0 || a.region >= l.regions:
		return auditFail(IdentityStructure, id, None, "region %d is outside the %d regions", a.region, l.regions)
	case a.balance < 0:
		return auditFail(IdentityNonNegative, id, Money, "balance %d is negative", a.balance)
	case a.reserved < 0 || a.reserved > a.balance:
		return auditFail(IdentityNonNegative, id, Money, "reserved %d is outside [0, balance %d]", a.reserved, a.balance)
	case a.balance > l.maxMoney:
		return auditFail(IdentityStructure, id, Money, "balance %d is above MaxMoney %d", a.balance, l.maxMoney)
	}
	return nil
}

// auditBoundary checks a boundary account against the boundary table.
func (l *Ledger) auditBoundary(id int, a *account) error {
	slot, err := l.slotOf(a.reason, a.thing)
	if err != nil || l.boundary[slot] != id {
		return auditFail(IdentityStructure, id, a.thing, "a boundary account is not the account of its reason and thing in the boundary table")
	}
	if rw, _ := rowOf(a.reason, a.thing != Money); rw.class == Internal {
		return auditFail(IdentityStructure, id, a.thing, "a boundary account of an internal reason")
	}
	if a.region != -1 || a.reserved != 0 || len(a.lots) != 0 || a.hasHint {
		return auditFail(IdentityStructure, id, a.thing, "a boundary account has a region, a reservation, lots or a hint")
	}
	return nil
}

// auditLots checks the lots of a non-boundary account: canonical form,
// non-negativity, the bounds of a key and the expiry hint.
func (l *Ledger) auditLots(id int, a *account) error {
	var (
		sum      int64
		earliest int64
		spoils   bool
	)
	for i, lt := range a.lots {
		if err := l.auditLotShape(id, i, a.lots); err != nil {
			return err
		}
		var err error
		if sum, err = l.auditLotSum(id, i, a.lots, sum); err != nil {
			return err
		}
		if s := l.comms[lt.Commodity].ShelfLife; s > 0 {
			earliest, spoils = earlier(earliest, spoils, expiry(lt.Born, s))
		}
	}
	return auditHint(id, a, spoils, earliest)
}

// auditLotSum adds lot i to the running quantity of its commodity (sum, which
// restarts at the first lot of a commodity) and checks it against MaxQuantity.
func (l *Ledger) auditLotSum(id, i int, lots []Lot, sum int64) (int64, error) {
	lt := lots[i]
	if i == 0 || lots[i-1].Commodity != lt.Commodity {
		sum = 0
	}
	sum, ok := checked.Add(sum, lt.Quantity)
	if !ok || sum > l.maxQty {
		return sum, auditFail(IdentityStructure, id, lt.Commodity, "quantity is above MaxQuantity %d", l.maxQty)
	}
	return sum, nil
}

// earlier folds expiry e into the earliest expiry seen so far.
func earlier(earliest int64, seen bool, e int64) (int64, bool) {
	if !seen || e < earliest {
		return e, true
	}
	return earliest, true
}

// auditLotShape checks lot i of lots: its commodity, quantities, order, born
// and durability.
func (l *Ledger) auditLotShape(id, i int, lots []Lot) error {
	lt := lots[i]
	if lt.Commodity < 0 || lt.Commodity >= len(l.comms) {
		return auditFail(IdentityCanonical, id, lt.Commodity, "lot %d names a commodity outside the table", i)
	}
	if err := auditLotQuantity(id, i, lt); err != nil {
		return err
	}
	if i > 0 && cmpLot(lots[i-1], lt.key()) >= 0 {
		return auditFail(IdentityCanonical, id, lt.Commodity, "lot %d is not after lot %d in (commodity, born, durability) order", i, i-1)
	}
	return l.auditLotRange(id, i, lt)
}

// auditLotQuantity checks the quantity and earmark of lot i.
func auditLotQuantity(id, i int, lt Lot) error {
	switch {
	case lt.Quantity < 0 || lt.Reserved < 0 || lt.Reserved > lt.Quantity:
		return auditFail(IdentityNonNegative, id, lt.Commodity, "lot %d has quantity %d and %d earmarked", i, lt.Quantity, lt.Reserved)
	case lt.Quantity == 0:
		return auditFail(IdentityCanonical, id, lt.Commodity, "lot %d has quantity 0", i)
	}
	return nil
}

// auditLotRange checks the born and durability of lot i against its commodity.
func (l *Ledger) auditLotRange(id, i int, lt Lot) error {
	c := l.comms[lt.Commodity]
	switch {
	case outsideRange(c.ShelfLife, lt.Born, -(c.ShelfLife - 1), l.tick):
		return auditFail(IdentityCanonical, id, lt.Commodity, "lot %d has born %d, outside its range", i, lt.Born)
	case outsideRange(c.Durability, lt.Durability, 1, c.Durability):
		return auditFail(IdentityCanonical, id, lt.Commodity, "lot %d has durability %d, outside its range", i, lt.Durability)
	}
	return nil
}

// outsideRange reports whether v breaks the range of a commodity attribute
// whose limit is given: v must be 0 when limit is 0, and within [lo, hi] when
// limit is positive.
func outsideRange(limit, v, lo, hi int64) bool {
	switch {
	case limit == 0:
		return v != 0
	case limit > 0:
		return v < lo || v > hi
	}
	return false
}

// auditHint checks the expiry hint of an account against its earliest expiry.
func auditHint(id int, a *account, spoils bool, earliest int64) error {
	switch {
	case spoils && !a.hasHint:
		return auditFail(IdentityHint, id, None, "a spoiling lot is held and the account has no hint")
	case spoils && a.hint > earliest:
		return auditFail(IdentityHint, id, None, "hint %d is later than the earliest expiry %d", a.hint, earliest)
	}
	return nil
}

// auditCounters checks that the flow totals, the wear units and the claim
// totals are never negative, and that the tables have their layout.
func (l *Ledger) auditCounters() error {
	if !l.tablesFit() {
		return auditFail(IdentityStructure, None, None, "a table has the wrong length")
	}
	for _, s := range [...]struct {
		name string
		v    []int64
	}{{"flow total", l.totals}, {"construction bucket", l.bucket}, {"wear units", l.wear}, {"construction wear units", l.wearBucket}} {
		for i, v := range s.v {
			if v < 0 {
				return auditFail(IdentityStructure, None, None, "%s %d is %d", s.name, i, v)
			}
		}
	}
	return l.auditClaimCounters()
}

// tablesFit reports whether the flow tables have the lengths of the layout.
func (l *Ledger) tablesFit() bool {
	return len(l.boundary) == l.stride && len(l.base) == l.stride &&
		len(l.totals) == l.stride*l.regions && len(l.bucket) == len(l.totals)
}

// auditClaimCounters checks that no claim total is negative.
func (l *Ledger) auditClaimCounters() error {
	t := l.claimTot
	for i, v := range [...]int64{t.Lent, t.Awarded, t.Repaid, t.InterestPaid, t.InterestFixed, t.LostPrincipal, t.LostInterest} {
		if v < 0 {
			return auditFail(IdentityStructure, None, None, "claim total %d is %d", i, v)
		}
	}
	return nil
}

// auditClaims checks the open claims, in ID order: the list (ascending IDs,
// below the next ID, at most MaxClaims), the ranges (CRD 3.6 item 4) and the
// parties (item 3). It names the first claim that fails (FR-LED-064).
func (l *Ledger) auditClaims() error {
	if len(l.claims) > l.maxClaims {
		return auditFail(IdentityClaim, None, None, "%d claims are open, MaxClaims is %d", len(l.claims), l.maxClaims)
	}
	prev := int64(-1)
	for i := range l.claims {
		c := &l.claims[i]
		if err := l.auditClaimShape(c, prev); err != nil {
			return err
		}
		prev = c.ID
		if err := l.auditClaimParties(c); err != nil {
			return err
		}
	}
	return nil
}

// auditClaimShape checks the ID order and the ranges of one open claim.
func (l *Ledger) auditClaimShape(c *Claim, prev int64) error {
	switch {
	case c.ID <= prev || c.ID >= l.nextClaim:
		return auditClaimFail(None, c.ID, "IDs are not ascending or not below the next ID %d", l.nextClaim)
	case c.Principal < 1 || c.Interest < 0 || c.Origin == OriginPayout && c.Interest != 0:
		return auditClaimFail(None, c.ID, "principal %d and interest %d are not valid", c.Principal, c.Interest)
	}
	return auditClaimOwed(c)
}

// auditClaimOwed checks the owed amounts and the parties of a claim differ.
func auditClaimOwed(c *Claim) error {
	switch {
	case c.PrincipalOwed < 0 || c.PrincipalOwed > c.Principal || c.InterestOwed < 0 || c.InterestOwed > c.Interest:
		return auditClaimFail(None, c.ID, "owed %d and %d are outside [0, %d] and [0, %d]", c.PrincipalOwed, c.InterestOwed, c.Principal, c.Interest)
	case c.PrincipalOwed == 0 && c.InterestOwed == 0:
		return auditClaimFail(None, c.ID, "an open claim owes nothing")
	case c.Creditor == c.Debtor:
		return auditClaimFail(c.Debtor, c.ID, "creditor and debtor are the same account")
	}
	return nil
}

// auditClaimParties checks that the creditor and the debtor of a claim are open
// accounts of the right kinds.
func (l *Ledger) auditClaimParties(c *Claim) error {
	cr, err := l.open(c.Creditor)
	if err != nil || !setAE.has(cr.kind) {
		return auditClaimFail(c.Creditor, c.ID, "the creditor is not an open Agent or Estate account")
	}
	if db, err := l.open(c.Debtor); err != nil || db.kind != KindAgent {
		return auditClaimFail(c.Debtor, c.ID, "the debtor is not an open Agent account")
	}
	return nil
}

// auditZeroTotals checks that money, and each commodity, sums to 0 over all
// accounts, boundaries included (9.1 item 1).
func (l *Ledger) auditZeroTotals() error {
	money, goods := l.holdings()
	if money != (wide{}) {
		return auditFail(IdentityZeroTotal, None, Money, "the total of money is not 0")
	}
	for c, g := range goods {
		if g != (wide{}) {
			return auditFail(IdentityZeroTotal, None, c, "the total quantity is not 0")
		}
	}
	return nil
}

// holdings sums money and each commodity over all accounts, boundaries
// included.
func (l *Ledger) holdings() (money wide, goods []wide) {
	goods = make([]wide, len(l.comms))
	for i := range l.accounts {
		a := &l.accounts[i]
		switch {
		case a.kind == KindBoundary && a.thing == Money:
			money = money.add(a.balance)
		case a.kind == KindBoundary:
			goods[a.thing] = goods[a.thing].add(a.balance)
		default:
			money = money.add(a.balance)
			for _, lt := range a.lots {
				goods[lt.Commodity] = goods[lt.Commodity].add(lt.Quantity)
			}
		}
	}
	return money, goods
}

// flowSum is the total moved under the flow slot over every region, from the
// totals of the current tick and, while the totals have not been cleared
// since construction, the construction bucket (7.4.1), whose share the
// baseline does not hold yet.
func (l *Ledger) flowSum(slot int) wide {
	var w wide
	for r := 0; r < l.regions; r++ {
		w = w.add(l.totals[r*l.stride+slot])
		if !l.cleared {
			w = w.add(l.bucket[r*l.stride+slot])
		}
	}
	return w
}

// slotVisitor receives a flow slot of a faucet or a sink: its reason, class,
// commodity (Money for money) and slot index.
type slotVisitor func(r Reason, class Class, commodity, slot int) error

// eachBoundarySlot calls f for every flow slot of a faucet or a sink: its
// reason, class, commodity (Money for money) and slot index.
func (l *Ledger) eachBoundarySlot(f slotVisitor) error {
	for r := Reason(0); r < NumReasons; r++ {
		for t := range table[r] {
			if err := l.eachThingSlot(r, t, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// eachThingSlot calls f for the slots of one reason and thing (money or
// goods), if they are a faucet or a sink.
func (l *Ledger) eachThingSlot(r Reason, t int, f slotVisitor) error {
	o, class := l.offOf(r, t), table[r][t].class
	if o < 0 || class == Internal {
		return nil
	}
	if t == thingMoney {
		return f(r, class, Money, o)
	}
	for c := range l.comms {
		if err := f(r, class, c, o+c); err != nil {
			return err
		}
	}
	return nil
}

// auditByReason checks the conservation by reason (9.1 item 3): the balance
// of each boundary account has moved since the totals were last cleared by
// exactly the faucet or sink total of its reason, summed over the regions.
func (l *Ledger) auditByReason() error {
	return l.eachBoundarySlot(func(r Reason, class Class, commodity, slot int) error {
		sum := l.flowSum(slot)
		if class == Faucet {
			sum = sum.neg()
		}
		id := l.boundary[slot]
		if id < 0 {
			if sum != (wide{}) {
				return auditFail(IdentityConservation, None, commodity, "reason %d has a flow total and no boundary account", r)
			}
			return nil
		}
		moved := wide{}.add(l.accounts[id].balance).plus(wide{}.add(l.base[slot]).neg())
		if moved != sum {
			return auditFail(IdentityConservation, id, commodity, "reason %d: the boundary balance moved differently from its flow total", r)
		}
		return nil
	})
}

// sector is the sector of a kind (9.3): households, public, contract,
// treasury, and the outside sector of the external module, which is zero at a
// tick boundary.
type sector uint8

const (
	sectorHouseholds sector = iota
	sectorPublic
	sectorContract
	sectorTreasury
	sectorOutside
)

// kindSector maps each non-boundary kind to its sector.
var kindSector = [numKinds]sector{
	KindAgent:    sectorHouseholds,
	KindEstate:   sectorPublic,
	KindEscrow:   sectorContract,
	KindTransit:  sectorContract,
	KindOutside:  sectorOutside,
	KindTreasury: sectorTreasury,
}

// auditSectors checks the sectoral identity (9.3): the outside sector is
// empty, and the money (and each commodity) held by the other four sectors
// has changed since the totals were last cleared by the faucets less the
// sinks of that period.
func (l *Ledger) auditSectors() error {
	nc := len(l.comms)
	held := l.sectorHeld(nc)
	out := int(sectorOutside) * (nc + 1)
	for j := 0; j <= nc; j++ {
		if held[out+j] != (wide{}) {
			return auditFail(IdentitySectoral, None, j-1, "the outside sector holds something")
		}
	}
	net, start := l.periodNet(nc)
	for j := 0; j <= nc; j++ {
		var now wide
		for s := sectorHouseholds; s < sectorOutside; s++ {
			now = now.plus(held[int(s)*(nc+1)+j])
		}
		// now - (-start) = net.
		if now.plus(start[j]) != net[j] {
			return auditFail(IdentitySectoral, None, j-1, "the four sectors changed by other than faucets less sinks")
		}
	}
	return nil
}

// sectorHeld sums what each sector holds: held[s*(nc+1)] is money and
// held[s*(nc+1)+1+c] commodity c.
func (l *Ledger) sectorHeld(nc int) []wide {
	held := make([]wide, int(sectorOutside+1)*(nc+1))
	for i := range l.accounts {
		a := &l.accounts[i]
		if a.kind == KindBoundary || a.closed {
			continue
		}
		base := int(kindSector[a.kind]) * (nc + 1)
		held[base] = held[base].add(a.balance)
		for _, lt := range a.lots {
			held[base+1+lt.Commodity] = held[base+1+lt.Commodity].add(lt.Quantity)
		}
	}
	return held
}

// periodNet returns net[j], faucets less sinks, and start[j], the boundary
// balances at the start of the period, for money (j 0) and each commodity
// (j 1+c).
func (l *Ledger) periodNet(nc int) (net, start []wide) {
	net, start = make([]wide, nc+1), make([]wide, nc+1)
	_ = l.eachBoundarySlot(func(_ Reason, class Class, commodity, slot int) error {
		sum := l.flowSum(slot)
		if class == Sink {
			sum = sum.neg()
		}
		net[commodity+1] = net[commodity+1].plus(sum)
		start[commodity+1] = start[commodity+1].add(l.base[slot])
		return nil
	})
	return net, start
}

// auditClaimTotals checks the pair of CRD 3.6 item 1: the principal owed over
// the open claims is Lent + Awarded - Repaid - LostPrincipal, and the interest
// owed plus InterestPaid plus LostInterest is InterestFixed.
func (l *Ledger) auditClaimTotals() error {
	var principal, interest wide
	for i := range l.claims {
		principal = principal.add(l.claims[i].PrincipalOwed)
		interest = interest.add(l.claims[i].InterestOwed)
	}
	t := l.claimTot
	want := wide{}.add(t.Lent).add(t.Awarded).add(-t.Repaid).add(-t.LostPrincipal)
	if principal != want {
		return auditFail(IdentityClaimTotals, None, Money, "the principal owed is not Lent + Awarded - Repaid - LostPrincipal")
	}
	paid := interest.add(t.InterestPaid).add(t.LostInterest)
	if paid != (wide{}.add(t.InterestFixed)) {
		return auditFail(IdentityClaimTotals, None, Money, "the interest owed, paid and lost is not InterestFixed")
	}
	return nil
}
