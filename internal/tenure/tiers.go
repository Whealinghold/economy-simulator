package tenure

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// Holder is one holder of an endowment with its entitlement (spec-land.md 4.2).
// ID is the agent ID (the pool's ID is its region ID) and must be distinct.
// Unused is written by Tiers: the entitlement the holder's own requests left,
// which is let to tenants (4.3 step 1).
type Holder struct {
	ID       int64
	Entitled int64
	Unused   int64
}

// Request is one request on the endowment (spec-land.md 4.3): Executions wanted
// of Draw units each. ID is the executor's agent ID (distinct), Principal the
// account whose inputs the job uses, Key the stateless key of RES 5.3.3 and
// Balance the principal's money at the ration. Exempt requests pay no rent
// (5.3.4). Ref is the caller's slot and is not read.
//
// Own, Commons and Let are written by Tiers: the executions granted in each
// tier. PricedOut is the units the clip of the let demand removed (4.3 step 3).
type Request struct {
	Ref        int
	ID         int64
	Principal  int64
	Executions int64
	Draw       int64
	Key        uint64
	Balance    int64
	Exempt     bool

	Own       int64
	Commons   int64
	Let       int64
	PricedOut int64

	asked int64 // the let demand after the clip, in executions
}

// Params is what the tiers of one endowment share. Stock is the snapshot stock
// the entitlements were apportioned from, Commons the commons entitlement, Rent
// the rent in force per unit let, and Closed whether the endowment lets nothing.
type Params struct {
	Stock   int64
	Commons int64
	Rent    int64
	Closed  bool
}

// Result is what the tiers took, in units. Supply is the let supply L, the sum
// of the unused entitlements, and Held that same sum when the endowment is
// Closed (the supply is then 0). PricedOut is the units the clip removed and
// Unserved the units asked of the let tier at the rent in force and not granted
// (the X of spec-land.md 5.5.1).
type Result struct {
	OwnUnits     int64
	CommonsUnits int64
	LetUnits     int64
	Supply       int64
	Held         int64
	PricedOut    int64
	Unserved     int64
}

// Scratch holds the working slices of Tiers, so a call allocates nothing once
// they have grown. The zero value is ready to use.
type Scratch struct {
	reqs   []endowment.Request
	grants []int64
	ti, li []int   // the indices of Attribute, in ascending ID
	left   []int64 // the lessors' remaining supplies in Attribute
}

// Tiers rations an endowment among its requests in the four steps of
// spec-land.md 4.3: own, commons, let. Each ration is the rule of
// endowment.Ration (RES 5.3.2) on a different supply.
//
// It reorders holders and reqs in place, so the caller reads the results by Ref
// and ID, and the result does not depend on how they were listed. Unit sums are
// checked. ok is false, and every output is 0, for a negative number, a Draw
// below 1, a repeated holder or executor ID, entitlements that sum beyond
// p.Stock, or a sum beyond int64.
func (s *Scratch) Tiers(p Params, holders []Holder, reqs []Request) (Result, bool) {
	res, ok := s.run(p, holders, reqs)
	if !ok {
		clearOutputs(holders, reqs)
		return Result{}, false
	}
	return res, true
}

// run is Tiers before the clearing of a failure.
func (s *Scratch) run(p Params, holders []Holder, reqs []Request) (Result, bool) {
	if !s.prepare(p, holders, reqs) {
		return Result{}, false
	}
	var res Result
	var ok bool
	if res.OwnUnits, ok = s.own(holders, reqs); !ok {
		return Result{}, false
	}
	slices.SortFunc(reqs, func(a, b Request) int { return cmp.Compare(a.ID, b.ID) })
	if res.CommonsUnits, ok = s.commons(p.Commons, reqs); !ok {
		return Result{}, false
	}
	return res, s.let(p, holders, reqs, &res)
}

// prepare validates the input, zeroes the outputs and sizes the scratch.
func (s *Scratch) prepare(p Params, holders []Holder, reqs []Request) bool {
	if p.Stock < 0 || p.Commons < 0 || p.Rent < 0 {
		return false
	}
	slices.SortFunc(holders, func(a, b Holder) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(reqs, func(a, b Request) int { return cmp.Compare(a.ID, b.ID) })
	if !prepareHolders(p, holders) || !prepareRequests(reqs) {
		return false
	}
	s.grants = slices.Grow(s.grants[:0], len(reqs))[:len(reqs)]
	return true
}

// prepareHolders checks the holders, sorted by ID, are distinct with entitlements
// that with the commons do not pass the stock, and sets each Unused to its
// entitlement.
func prepareHolders(p Params, holders []Holder) bool {
	entitled := p.Commons
	for i := range holders {
		h := &holders[i]
		var ok bool
		if h.Entitled < 0 || (i > 0 && holders[i-1].ID == h.ID) {
			return false
		}
		if entitled, ok = checked.Add(entitled, h.Entitled); !ok {
			return false
		}
		h.Unused = h.Entitled
	}
	return entitled <= p.Stock
}

// prepareRequests checks the requests, sorted by ID, are valid and distinct, and
// zeroes their outputs.
func prepareRequests(reqs []Request) bool {
	for i := range reqs {
		r := &reqs[i]
		if r.Executions < 0 || r.Draw < 1 || (i > 0 && reqs[i-1].ID == r.ID) {
			return false
		}
		r.Own, r.Commons, r.Let, r.PricedOut, r.asked = 0, 0, 0, 0, 0
	}
	return true
}

// clearOutputs zeroes every output the tiers write.
func clearOutputs(holders []Holder, reqs []Request) {
	for i := range holders {
		holders[i].Unused = 0
	}
	for i := range reqs {
		r := &reqs[i]
		r.Own, r.Commons, r.Let, r.PricedOut = 0, 0, 0, 0
	}
}

// own serves the requests of each holder from its own entitlement (4.3 step 1)
// and returns the units taken. The requests are sorted by principal, then by
// ID, so a group is contiguous and its position order is its ID order.
func (s *Scratch) own(holders []Holder, reqs []Request) (int64, bool) {
	slices.SortFunc(reqs, func(a, b Request) int {
		return cmp.Or(cmp.Compare(a.Principal, b.Principal), cmp.Compare(a.ID, b.ID))
	})
	var total int64
	for start := 0; start < len(reqs); {
		end := start + 1
		for end < len(reqs) && reqs[end].Principal == reqs[start].Principal {
			end++
		}
		hi, found := slices.BinarySearchFunc(holders, reqs[start].Principal, func(h Holder, id int64) int {
			return cmp.Compare(h.ID, id)
		})
		if found {
			used, ok := s.ownGroup(&holders[hi], reqs[start:end])
			if !ok {
				return 0, false
			}
			if total, ok = checked.Add(total, used); !ok {
				return 0, false
			}
		}
		start = end
	}
	return total, true
}

// ownGroup rations the entitlement of h among the requests of its principal,
// writes their Own grants and the Unused of h, and returns the units used.
func (s *Scratch) ownGroup(h *Holder, group []Request) (int64, bool) {
	s.reqs = s.reqs[:0]
	for i, r := range group {
		s.reqs = append(s.reqs, endowment.Request{Ref: i, Executions: r.Executions, Draw: r.Draw, Key: r.Key})
	}
	if !endowment.Ration(h.Entitled, s.reqs, s.grants[:len(group)]) {
		return 0, false
	}
	var used int64
	for i := range group {
		u, ok := checked.Mul(s.grants[i], group[i].Draw)
		if !ok {
			return 0, false
		}
		if used, ok = checked.Add(used, u); !ok {
			return 0, false
		}
		group[i].Own = s.grants[i]
	}
	h.Unused = h.Entitled - used // used <= Entitled: the ration never exceeds its supply
	return used, true
}

// commons serves the residual demands from the commons entitlement (4.3 step 2)
// and returns the units taken. The requests are sorted by ID.
func (s *Scratch) commons(entitled int64, reqs []Request) (int64, bool) {
	s.reqs = s.reqs[:0]
	for i, r := range reqs {
		s.reqs = append(s.reqs, endowment.Request{Ref: i, Executions: r.Executions - r.Own, Draw: r.Draw, Key: r.Key})
	}
	if !endowment.Ration(entitled, s.reqs, s.grants) {
		return 0, false
	}
	var total int64
	for i := range reqs {
		u, ok := checked.Mul(s.grants[i], reqs[i].Draw)
		if !ok {
			return 0, false
		}
		if total, ok = checked.Add(total, u); !ok {
			return 0, false
		}
		reqs[i].Commons = s.grants[i]
	}
	return total, true
}

// let serves what remains, clipped to what each principal can pay, from the sum
// of the unused entitlements (4.3 step 3), and fills the let fields of res. The
// requests are sorted by ID.
func (s *Scratch) let(p Params, holders []Holder, reqs []Request, res *Result) bool {
	var supply int64
	for _, h := range holders {
		var ok bool
		if supply, ok = checked.Add(supply, h.Unused); !ok {
			return false
		}
	}
	if p.Closed {
		res.Held = supply
		return true
	}
	res.Supply = supply
	s.reqs = s.reqs[:0]
	for i := range reqs {
		r := &reqs[i]
		resid := r.Executions - r.Own - r.Commons
		clipped := clip(resid, r.Draw, r.Balance, p.Rent, r.Exempt)
		priced, ok := checked.Mul(resid-clipped, r.Draw)
		if !ok {
			return false
		}
		r.PricedOut, r.asked = priced, clipped
		s.reqs = append(s.reqs, endowment.Request{Ref: i, Executions: clipped, Draw: r.Draw, Key: r.Key})
	}
	if !endowment.Ration(supply, s.reqs, s.grants) {
		return false
	}
	return s.tally(reqs, res)
}

// tally writes the Let grants and the unit counts of the let tier.
func (s *Scratch) tally(reqs []Request, res *Result) bool {
	for i := range reqs {
		r := &reqs[i]
		r.Let = s.grants[i]
		taken, ok1 := checked.Mul(r.Let, r.Draw)
		unserved, ok2 := checked.Mul(r.asked-r.Let, r.Draw) // the ration never exceeds the demand
		priced, ok3 := checked.Add(res.PricedOut, r.PricedOut)
		let, ok4 := checked.Add(res.LetUnits, taken)
		short, ok5 := checked.Add(res.Unserved, unserved)
		if !(ok1 && ok2 && ok3 && ok4 && ok5) {
			return false
		}
		res.PricedOut, res.LetUnits, res.Unserved = priced, let, short
	}
	return true
}

// clip returns the executions of a residual demand whose rent the principal can
// pay at rent per unit: the whole demand when no rent is due, else at most
// floor(balance / (rent * draw)). A product beyond int64 is unaffordable.
func clip(resid, draw, balance, rent int64, exempt bool) int64 {
	if rent == 0 || exempt {
		return resid
	}
	unit, ok := checked.Mul(rent, draw)
	if !ok || balance < unit {
		return 0
	}
	return min(resid, balance/unit)
}
