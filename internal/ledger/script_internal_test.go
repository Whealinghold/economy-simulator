package ledger

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
)

// chooser supplies the choices of a script: n(k) is in [0, k). The invariant
// test backs it with a seeded math/rand/v2 source and FuzzOps with the bytes
// of its input, so the two drive the same operations.
type chooser interface {
	n(k int) int
	// more reports whether the script should go on.
	more() bool
}

type randChooser struct {
	r    *rand.Rand
	left int
}

func (c *randChooser) n(k int) int { c.left--; return c.r.IntN(k) }
func (c *randChooser) more() bool  { return c.left > 0 }

type byteChooser struct {
	b []byte
	i int
}

func (c *byteChooser) n(k int) int {
	if c.i >= len(c.b) {
		c.i++
		return 0
	}
	v := int(c.b[c.i])
	c.i++
	return v % k
}
func (c *byteChooser) more() bool { return c.i < len(c.b) }

// Sizes of the scripted world. maxMoney and maxQty are small so that the
// bounds are reached.
const (
	scriptMaxMoney  = 1000
	scriptMaxQty    = 100
	scriptRegions   = 2
	scriptMaxClaims = 4
)

// rig is a ledger with the accounts the script uses, and the IDs of the
// accounts it picks from.
type rig struct {
	l *Ledger
	// mint is the money boundary of Initial, and bnd[r][c] the goods boundary
	// of reason r for commodity c.
	mint int
	bnd  map[Reason][]int
	// holders are the agents and estates; the boundaries are not in it.
	holders []int
}

// scriptReasons are the reasons whose goods boundaries the rig opens.
var scriptReasons = []Reason{Initial, RecipeInput, RecipeOutput, Disposal, Consumption, Spoilage, Wear}

// rigConfig is the configuration of the scripted ledger.
func rigConfig(maxMoney int64) Config {
	return Config{
		Commodities: []Commodity{{ShelfLife: 3}, {Durability: 5}, {}, {ShelfLife: 4, Durability: 6}},
		Regions:     scriptRegions,
		MaxMoney:    maxMoney,
		MaxQuantity: scriptMaxQty,
		Declared:    []Reason{LoanOut, LoanRepay, LoanInterest},
		MaxClaims:   scriptMaxClaims,
	}
}

// newRig builds the scripted ledger in its construction phase: four
// commodities (spoils; wears; neither; both), two regions, four agents, two
// estates, and every boundary account the operations use.
func newRig(tb testing.TB, maxMoney int64) *rig {
	tb.Helper()
	l, err := New(rigConfig(maxMoney))
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	g := &rig{l: l, bnd: map[Reason][]int{}}
	must := func(id int, err error) int {
		if err != nil {
			tb.Fatalf("setup: %v", err)
		}
		return id
	}
	g.mint = must(l.OpenBoundary(Initial, Money))
	for _, r := range scriptReasons {
		for c := range l.comms {
			g.bnd[r] = append(g.bnd[r], must(l.OpenBoundary(r, c)))
		}
	}
	for i := 0; i < 4; i++ {
		g.holders = append(g.holders, must(l.Open(KindAgent, i, i%scriptRegions)))
	}
	for i := 0; i < 2; i++ {
		g.holders = append(g.holders, must(l.Open(KindEstate, 100+i, i)))
	}
	return g
}

// fund gives every agent money and goods, in construction.
func (g *rig) fund(tb testing.TB) {
	tb.Helper()
	for _, a := range g.holders[:4] {
		if err := g.l.Transfer(g.mint, a, 300, Initial); err != nil {
			tb.Fatalf("fund money: %v", err)
		}
		for c := range g.l.comms {
			if err := g.l.Move(g.bnd[Initial][c], a, c, 12, Initial); err != nil {
				tb.Fatalf("fund goods: %v", err)
			}
		}
	}
	// A stated lot of a spoiling commodity, born before construction.
	key := Key{Commodity: 0, Born: -1}
	if err := g.l.MoveStated(g.bnd[Initial][0], g.holders[0], key, 3, Initial); err != nil {
		tb.Fatalf("fund stated: %v", err)
	}
}

// fingerprint is a deep copy of everything a ledger holds, with the scratch
// that is empty between calls normalised, so that two fingerprints are
// reflect.DeepEqual exactly when the ledgers are in the same state.
func fingerprint(l *Ledger) *Ledger {
	c := *l
	c.accounts = make([]account, len(l.accounts))
	for i, a := range l.accounts {
		a.lots = slices.Clone(a.lots)
		c.accounts[i] = a
	}
	c.comms = slices.Clone(l.comms)
	c.boundary = slices.Clone(l.boundary)
	c.totals = slices.Clone(l.totals)
	c.bucket = slices.Clone(l.bucket)
	c.base = slices.Clone(l.base)
	c.claims = slices.Clone(l.claims)
	c.wear = slices.Clone(l.wear)
	c.wearBucket = slices.Clone(l.wearBucket)
	c.spoil = slices.Clone(l.spoil)
	c.sel = nil
	return &c
}

// scripter drives a rig with a chooser and checks the properties after every
// operation.
type scripter struct {
	g  *rig
	ch chooser
	// ok counts the operations of each name that were accepted, and tried
	// those that were attempted.
	ok, tried map[string]int
}

func newScripter(g *rig, ch chooser) *scripter {
	return &scripter{g: g, ch: ch, ok: map[string]int{}, tried: map[string]int{}}
}

var (
	moneyAmounts = []int64{0, 1, 2, 5, 13, 40, 120, 999, 1001, -1}
	goodsAmounts = []int64{0, 1, 2, 3, 5, 12, 30, 60, 101, -1}
	wearUnits    = []int64{1, 2, 3, 5, 9}
)

func (s *scripter) account() int { return s.ch.n(s.g.l.Len()+3) - 1 }

// mostlyAgent is an agent, usually, else any account.
func (s *scripter) mostlyAgent() int {
	if s.ch.n(4) > 0 {
		return s.agent()
	}
	return s.account()
}

func (s *scripter) holder() int { return s.g.holders[s.ch.n(len(s.g.holders))] }

func (s *scripter) agent() int { return s.g.holders[s.ch.n(4)] }

func (s *scripter) commodity() int { return s.ch.n(len(s.g.l.comms)+2) - 1 }

func (s *scripter) reason() Reason { return Reason(s.ch.n(int(NumReasons) + 1)) }

// moneyPair picks the two accounts and the reason of a money transfer,
// biased to the shapes that can succeed.
func (s *scripter) moneyPair() (from, to int, r Reason) {
	switch s.ch.n(8) {
	case 0:
		return s.g.mint, s.holder(), Initial
	case 1:
		return s.agent(), s.agent(), Trade
	case 2:
		return s.agent(), s.g.holders[4+s.ch.n(2)], EstateMoney
	case 3:
		return s.g.holders[4+s.ch.n(2)], s.agent(), StartMoney
	case 4:
		return s.agent(), s.agent(), []Reason{LoanOut, LoanRepay, LoanInterest}[s.ch.n(3)]
	}
	return s.account(), s.account(), s.reason()
}

// goodsPair picks the two accounts and the reason of a goods move.
func (s *scripter) goodsPair(c int) (from, to int, r Reason) {
	at := func(r Reason) int {
		if c >= 0 && c < len(s.g.l.comms) {
			return s.g.bnd[r][c]
		}
		return s.account()
	}
	switch s.ch.n(8) {
	case 0:
		return at(Initial), s.holder(), Initial
	case 1:
		return s.agent(), at(Disposal), Disposal
	case 2:
		return s.agent(), at(Consumption), Consumption
	case 3, 4:
		return s.agent(), s.agent(), Trade
	case 5:
		return s.agent(), s.g.holders[4+s.ch.n(2)], EstateGoods
	}
	return s.account(), s.account(), s.reason()
}

// key picks the key of a named lot, usually one the account holds.
func (s *scripter) key(id int) Key {
	if lots, err := s.g.l.AppendLots(nil, id); err == nil && len(lots) > 0 && s.ch.n(4) > 0 {
		return lots[s.ch.n(len(lots))].key()
	}
	return Key{Commodity: s.ch.n(4), Born: int64(s.ch.n(5)) - 3, Durability: int64(s.ch.n(7))}
}

func (s *scripter) lines() []Line {
	out := make([]Line, s.ch.n(3))
	for i := range out {
		out[i] = Line{Commodity: s.commodity(), Quantity: goodsAmounts[s.ch.n(len(goodsAmounts))]}
	}
	return out
}

func (s *scripter) claimID() int64 {
	cl := s.g.l.AppendClaims(nil)
	if len(cl) == 0 || s.ch.n(8) == 0 {
		return int64(s.ch.n(6)) - 1
	}
	return cl[s.ch.n(len(cl))].ID
}

// reservedMoney picks an account with money reserved, usually, else any.
func (s *scripter) reservedMoney() int {
	var ids []int
	for id := range s.g.l.accounts {
		if s.g.l.accounts[id].reserved > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || s.ch.n(4) == 0 {
		return s.account()
	}
	return ids[s.ch.n(len(ids))]
}

// reservedLot picks a lot with units earmarked, usually, else any account and
// key.
func (s *scripter) reservedLot() (int, Key) {
	type pick struct {
		id  int
		key Key
	}
	var picks []pick
	for id := range s.g.l.accounts {
		for _, lt := range s.g.l.accounts[id].lots {
			if lt.Reserved > 0 {
				picks = append(picks, pick{id, lt.key()})
			}
		}
	}
	if len(picks) == 0 || s.ch.n(4) == 0 {
		id := s.account()
		return id, s.key(id)
	}
	p := picks[s.ch.n(len(picks))]
	return p.id, p.key
}

// releaseAll releases every reservation and earmark in the ledger.
func (s *scripter) releaseAll() error {
	l := s.g.l
	for id := 0; id < l.Len(); id++ {
		a := &l.accounts[id]
		if a.closed || a.kind == KindBoundary {
			continue
		}
		if a.reserved > 0 {
			if err := l.ReleaseMoney(id, a.reserved); err != nil {
				return err
			}
		}
		for c := range l.comms {
			if n := a.earmarkedOf(c); n > 0 {
				if err := l.ReleaseGoods(id, c, n); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// step runs one operation chosen by the chooser and returns its name and the
// error it returned. A property failure is the second error return.
func (s *scripter) step() (name string, opErr, failure error) {
	l := s.g.l
	before := fingerprint(l)
	name, opErr = s.do()
	s.tried[name]++
	if opErr == nil {
		s.ok[name]++
	} else if !reflect.DeepEqual(before, fingerprint(l)) {
		return name, opErr, fmt.Errorf("%s was refused with %v and changed the ledger", name, opErr)
	}
	if err := l.Audit(); err != nil {
		return name, opErr, fmt.Errorf("after %s: %w", name, err)
	}
	return name, opErr, nil
}

func (s *scripter) do() (string, error) {
	l, ch := s.g.l, s.ch
	switch ch.n(30) {
	case 0, 1, 2:
		f, t, r := s.moneyPair()
		return "Transfer", l.Transfer(f, t, moneyAmounts[ch.n(len(moneyAmounts))], r)
	case 3:
		_, t, r := s.moneyPair()
		return "TransferReserved", l.TransferReserved(s.reservedMoney(), t, moneyAmounts[ch.n(len(moneyAmounts))], r)
	case 4:
		return "ReserveMoney", l.ReserveMoney(s.mostlyAgent(), moneyAmounts[ch.n(len(moneyAmounts))])
	case 5:
		return "ReleaseMoney", l.ReleaseMoney(s.reservedMoney(), moneyAmounts[ch.n(len(moneyAmounts))])
	case 6, 7, 8:
		c := s.commodity()
		f, t, r := s.goodsPair(c)
		return "Move", l.Move(f, t, c, goodsAmounts[ch.n(len(goodsAmounts))], r)
	case 9:
		f, key := s.reservedLot()
		_, t, r := s.goodsPair(key.Commodity)
		return "MoveReserved", l.MoveReserved(f, t, key.Commodity, goodsAmounts[ch.n(len(goodsAmounts))], r)
	case 10:
		f := s.account()
		c := s.commodity()
		_, t, r := s.goodsPair(c)
		return "MoveLot", l.MoveLot(f, t, s.key(f), goodsAmounts[ch.n(len(goodsAmounts))], r)
	case 11:
		f, key := s.reservedLot()
		_, t, r := s.goodsPair(key.Commodity)
		return "MoveLotReserved", l.MoveLotReserved(f, t, key, goodsAmounts[ch.n(len(goodsAmounts))], r)
	case 12:
		c := s.commodity()
		from, to := s.g.mint, s.holder()
		if c >= 0 && c < len(l.comms) {
			from = s.g.bnd[Initial][c]
		}
		k := Key{Commodity: c, Born: -int64(ch.n(5)), Durability: int64(ch.n(8))}
		return "MoveStated", l.MoveStated(from, to, k, goodsAmounts[ch.n(len(goodsAmounts))], Initial)
	case 13:
		return "ReserveGoods", l.ReserveGoods(s.mostlyAgent(), s.commodity(), goodsAmounts[ch.n(len(goodsAmounts))])
	case 14:
		id, key := s.reservedLot()
		return "ReleaseGoods", l.ReleaseGoods(id, key.Commodity, goodsAmounts[ch.n(len(goodsAmounts))])
	case 15:
		id := s.mostlyAgent()
		return "ReserveLot", l.ReserveLot(id, s.key(id), goodsAmounts[ch.n(len(goodsAmounts))])
	case 16:
		id, key := s.reservedLot()
		return "ReleaseLot", l.ReleaseLot(id, key, goodsAmounts[ch.n(len(goodsAmounts))])
	case 17, 18:
		return "Transform", l.Transform(s.agent(), s.lines(), s.lines())
	case 19:
		return "WearItems", l.WearItems(s.agent(), s.commodity(), goodsAmounts[ch.n(len(goodsAmounts))], wearUnits[ch.n(len(wearUnits))])
	case 20:
		return "Wear", l.Wear(s.agent(), s.commodity(), wearUnits[ch.n(len(wearUnits))])
	case 21, 22:
		return "EndTick", s.endTick()
	case 23:
		due := l.Tick() + int64(ch.n(5))
		_, err := l.Lend(s.agent(), s.agent(), moneyAmounts[ch.n(len(moneyAmounts))], int64(ch.n(6))-1, 500, due)
		return "Lend", err
	case 24:
		id := s.claimID()
		owed := int64(20)
		if cl := l.AppendClaims(nil); len(cl) > 0 {
			owed = cl[0].PrincipalOwed
		}
		parts := []int64{0, 1, 2, owed, owed + 1, -1}
		return "Settle", l.Settle(id, parts[ch.n(len(parts))], parts[ch.n(len(parts))])
	case 25:
		if ch.n(2) == 0 {
			_, _, err := l.WriteOff(s.claimID())
			return "WriteOff", err
		}
		return "Reassign", l.Reassign(s.claimID(), s.account())
	case 26:
		if ch.n(2) == 0 {
			_, err := l.Obligate(s.holder(), s.agent(), moneyAmounts[ch.n(len(moneyAmounts))], l.Tick()+int64(ch.n(5)))
			return "Obligate", err
		}
		return "MarkOverdue", l.MarkOverdue(s.claimID(), l.Tick()+int64(ch.n(4)))
	case 27:
		if ch.n(3) == 0 {
			return "Close", l.Close(s.account())
		}
		_, err := l.Open(Kind(ch.n(int(numKinds))), 0, ch.n(scriptRegions+1))
		return "Open", err
	case 28:
		prices := []int64{3, 0, 7, 11}
		if ch.n(8) == 0 {
			prices[ch.n(4)] = int64(1) << 62
		}
		_, err := l.BalanceSheet(s.account(), prices)
		return "BalanceSheet", err
	}
	if _, err := l.Expiring(s.account(), s.commodity(), l.Tick()+int64(ch.n(5))-1); err != nil {
		return "Expiring", err
	}
	if ch.n(6) == 0 {
		return "EndConstruction", l.EndConstruction()
	}
	return "Expiring", nil
}

// endTick is what the world does at the end of a tick: the reservations are
// released, and none is held (FR-LED-020), the goods age, and the tick ends.
// Construction ends at the first one, as the first Step ends it.
func (s *scripter) endTick() error {
	l := s.g.l
	if err := s.releaseAll(); err != nil {
		return fmt.Errorf("releaseAll: %v", err)
	}
	if l.HasReservation() {
		return fmt.Errorf("a reservation is held after releaseAll")
	}
	if l.InConstruction() {
		if err := l.EndConstruction(); err != nil {
			return err
		}
	}
	if err := l.Age(); err != nil {
		return err
	}
	l.EndTick()
	return nil
}

// run drives the script until the chooser is exhausted and returns the first
// property failure.
func (s *scripter) run() error {
	for i := 0; s.ch.more(); i++ {
		name, _, err := s.step()
		if err != nil {
			return fmt.Errorf("operation %d (%s): %w", i, name, err)
		}
	}
	return nil
}
