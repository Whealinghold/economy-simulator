package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/insure"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the cover round of the insurance module (spec-insurance.md 5.2, 7; the
// continuation of insurance.go, architecture.md 2.6): the requests and offers of the
// plans, kept for P3.3c, the admission, the lanes of the round, the order and the matching
// through internal/insure. It takes no word of the stream: every tie is the stateless key
// K(t, agent) (FR-INS-033). The formation of a policy is in insurance_form.go and its
// candidates in insurance_candidates.go.

// coverRequest is a request as a plan returned it, with the agent and its number in the
// plan, kept for the round (the plan is reused by the next agent). store says it is the
// plan's store request and sc is read; otherwise lane is.
type coverRequest struct {
	agent AgentID
	seq   int
	store bool
	lane  LaneCover
	sc    StoreCover
}

// coverOffer is an offer as a plan returned it, with the agent.
type coverOffer struct {
	agent AgentID
	o     CoverOffer
}

// admittedRequest and admittedOffer are an admitted request and offer, found by the Ref of
// the row of the lane that holds its limit, cover share, rates and key.
type admittedRequest struct {
	req    coverRequest
	region RegionID
}

type admittedOffer struct {
	agent  AgentID
	region RegionID
	o      CoverOffer
}

// coverLane is the admitted requests and offers of one region, the lane of the round
// (spec-insurance.md 7.2.1): the route requests, the store requests and the offers, which
// both passes share and whose room the matching lowers.
type coverLane struct {
	routes, stores []insure.Request
	offers         []insure.Offer
}

// coverCandidate is one policy asked for: the admitted request it belongs to, the shipment
// of a route candidate (NoShipment for a store), the sum insured, the lane's expected
// loss, and what became of it (candidateOpen, candidateCovered or candidateFailed).
type coverCandidate struct {
	req   int
	ship  ShipmentID
	sum   int64
	loss  int64
	state int8
}

const (
	candidateOpen int8 = iota
	candidateCovered
	candidateFailed
)

// coverCount is what CoverRound reports for one region.
type coverCount struct {
	requests, offers, requestsRefused, offersRefused int64
	inArrears, exposure                              int64
	candidates, formed, sum, premium, unfilled       int64
	noShipment, noRisk, noOffer, belowMinimum        int64
	funds, tooMany, overflow, valueCapped            int64
}

// quiet reports whether every count that CoverRound carries is 0.
func (c coverCount) quiet() bool { return c == coverCount{} }

// coverRound is the storage of the round: the submitted requests and offers, the admitted
// ones, the lanes and the counts per region, the candidates and the unit values of the
// pass, reused every tick.
type coverRound struct {
	reqs   []coverRequest
	offs   []coverOffer
	rq     []admittedRequest
	of     []admittedOffer
	lanes  []coverLane
	counts []coverCount
	cands  []coverCandidate
	rows   []insure.Candidate
	// values and capped are the unit values of the lane's region, by commodity, and
	// whether the cap bound; qty and held the units of one candidate by commodity.
	values, qty, held []int64
	capped            []bool
	// haltAt is the index of the candidate at which the policies limit ended the pass of
	// the lane, or the number of candidates when it did not.
	haltAt int
	form   func(insure.Fill) insure.Outcome
}

// reset empties the round for the next tick, keeping the storage.
func (rd *coverRound) reset() {
	rd.reqs, rd.offs, rd.rq, rd.of = rd.reqs[:0], rd.offs[:0], rd.rq[:0], rd.of[:0]
	for i := range rd.lanes {
		l := &rd.lanes[i]
		l.routes, l.stores, l.offers = l.routes[:0], l.stores[:0], l.offers[:0]
	}
	clear(rd.counts)
}

// p33cCoverRound is P3.3c (spec-insurance.md 7): the admission of the requests and offers
// of the plans in agent order, the lanes in ascending region with the order and the
// matching of each, the formation of each fill, then CoverRound per region. It runs after
// the credit round and every plan and takes no word. The store policies of the last tick
// end first: no position ends them in their own tick yet (plan 3-ins PR D, P8.1b).
func (w *World) p33cCoverRound() {
	in := w.insurance
	rd := &in.round
	in.dropStores()
	if len(rd.reqs)+len(rd.offs) > 0 {
		w.admitCoverRequests()
		w.admitCoverOffers()
		w.matchCoverLanes()
		w.reportCoverRound()
	}
	rd.reset()
}

// admitCoverRequests admits or refuses each request in agent order, a refusal whole with its
// reason (spec-insurance.md 5.2), and puts an admitted one in the lane of its agent with its
// limit, its lowest cover share and its key.
func (w *World) admitCoverRequests() {
	in := w.insurance
	rd, t := &in.round, w.Tick()
	for i := range rd.reqs {
		s := &rd.reqs[i]
		a := &w.agents[s.agent]
		g := w.regionOf(a)
		c := &rd.counts[g]
		if reason := w.coverRequestRefusal(a, s); reason != "" {
			c.requestsRefused++
			w.logCoverRefused(s.agent, reason)
			continue
		}
		c.requests++
		limit, minCover := s.lane.MaxLoadingBP, s.lane.MinCoverBP
		if s.store {
			limit, minCover = s.sc.MaxStoreRateBP, s.sc.MinCoverBP
		}
		row := insure.Request{Ref: len(rd.rq), Party: int64(s.agent), Limit: limit, MinCover: minCover, Key: ordering.Key(t, int64(s.agent)), Seq: s.seq}
		l := &rd.lanes[g]
		if s.store {
			l.stores = append(l.stores, row)
		} else {
			l.routes = append(l.routes, row)
		}
		rd.rq = append(rd.rq, admittedRequest{req: *s, region: g})
	}
}

// coverRequestRefusal is the first failing check of 5.2 for a request, in the order of the
// table, or "" when it is admitted. The module is on, so insurance_off is not one.
func (w *World) coverRequestRefusal(a *agent, s *coverRequest) Reason {
	t := w.cfg.Roles[a.role].Cover
	switch {
	case t == nil || t.Insure == nil:
		return ReasonNotInsuredRole
	case int64(s.seq) >= w.insurance.cfg.MaxCoverRequests:
		return ReasonTooManyRequests
	case !validRequest(s):
		return ReasonInvalidCover
	}
	if reason := w.coverTargetRefusal(a, s); reason != "" {
		return reason
	}
	if w.coverRequested(s) {
		return ReasonDuplicate
	}
	return ""
}

// validRequest is row 4 of the table of 5.2: the cover share, the limit and the share of a
// request inside their bounds.
func validRequest(s *coverRequest) bool {
	if s.store {
		return s.sc.ShareBP >= 1 && s.sc.ShareBP <= wholeBP && s.sc.MinCoverBP >= 1 && s.sc.MinCoverBP <= wholeBP &&
			s.sc.MaxStoreRateBP >= 0 && s.sc.MaxStoreRateBP <= MaxStoreRateBound
	}
	return s.lane.MinCoverBP >= 1 && s.lane.MinCoverBP <= wholeBP && s.lane.MaxLoadingBP >= 0 && s.lane.MaxLoadingBP <= MaxLoadingBound
}

// coverTargetRefusal is row 5 of the table: a lane that is a configured mode and a region
// other than the agent's own and not an outside node, or a store request of an agent that
// owns a store.
func (w *World) coverTargetRefusal(a *agent, s *coverRequest) Reason {
	if !s.store {
		return w.laneRefusal(a, &s.lane)
	}
	if w.sto == nil || w.sto.owned(a.id) == 0 {
		return ReasonNoStore
	}
	return ""
}

// laneRefusal is unknown_lane for a lane request: the mode is not configured, or the
// destination is not a region, is the agent's own or is an outside node.
func (w *World) laneRefusal(a *agent, l *LaneCover) Reason {
	if w.transport == nil || l.Mode < 0 || int(l.Mode) >= len(w.transport.modes) {
		return ReasonUnknownLane
	}
	if l.To < 0 || int(l.To) >= len(w.cfg.Regions) || l.To == w.regionOf(a) || w.isOutsideNode(l.To) {
		return ReasonUnknownLane
	}
	return ""
}

// coverRequested says the agent already has an admitted request of the same lane, or its
// store request, this tick. The requests are in agent order, so the agent's admitted ones
// are the last rows.
func (w *World) coverRequested(s *coverRequest) bool {
	rq := w.insurance.round.rq
	for j := len(rq) - 1; j >= 0 && rq[j].req.agent == s.agent; j-- {
		r := &rq[j].req
		if r.store && s.store || !r.store && !s.store && r.lane.Mode == s.lane.Mode && r.lane.To == s.lane.To {
			return true
		}
	}
	return false
}

// admitCoverOffers admits or refuses each offer in agent order and puts an admitted one in
// its lane with its terms, its room (the liability it offers) and its key.
func (w *World) admitCoverOffers() {
	in := w.insurance
	rd, t := &in.round, w.Tick()
	for i := range rd.offs {
		s := &rd.offs[i]
		a := &w.agents[s.agent]
		g := w.regionOf(a)
		c := &rd.counts[g]
		if reason := w.coverOfferRefusal(a, s); reason != "" {
			c.offersRefused++
			switch reason {
			case ReasonInArrears:
				c.inArrears++
			case ReasonExposure:
				c.exposure++
			}
			w.logCoverRefused(s.agent, reason)
			continue
		}
		c.offers++
		l := &rd.lanes[g]
		l.offers = append(l.offers, insure.Offer{
			Ref: len(rd.of), Party: int64(s.agent), Loading: s.o.LoadingBP, StoreRate: s.o.StoreRateBP, CoverBP: s.o.CoverBP,
			Room: s.o.MaxLiability, Key: ordering.Key(t, int64(s.agent)),
		})
		rd.of = append(rd.of, admittedOffer{agent: s.agent, region: g, o: s.o})
	}
}

// coverOfferRefusal is the first failing check of 5.2 for an offer, or "" when it is
// admitted: the underwriter's available money is its balance less what its orders of the
// tick reserved (5.2.1), and an offer above its room is refused whole, never cut (5.2.2).
func (w *World) coverOfferRefusal(a *agent, s *coverOffer) Reason {
	t := w.cfg.Roles[a.role].Cover
	if t == nil || t.Underwrite == nil {
		return ReasonNotUnderwriter
	}
	if !validOffer(&s.o, w.insurance.cfg.MinSum) {
		return ReasonInvalidOffer
	}
	if of := w.insurance.round.of; len(of) > 0 && of[len(of)-1].agent == s.agent {
		return ReasonDuplicate
	}
	if w.figuresOf(a.account).Overdue > 0 {
		return ReasonInArrears
	}
	v := w.accountView(a.account)
	free, ok := insure.Free(t.Underwrite.CapBP, v.Balance-v.Reserved, w.insurance.liabilityOf(a.id), MaxMoney)
	if !ok {
		invariant("a negative figure in the exposure room of an underwriter")
	}
	if s.o.MaxLiability > free {
		return ReasonExposure
	}
	return ""
}

// validOffer is row 4 of the table of 5.2 for an offer: a liability from MinSum to
// MaxMoney, a cover share and the two rates inside their bounds.
func validOffer(o *CoverOffer, minSum int64) bool {
	return o.MaxLiability >= minSum && o.MaxLiability <= MaxMoney && o.CoverBP >= 1 && o.CoverBP <= wholeBP &&
		o.LoadingBP >= 0 && o.LoadingBP <= MaxLoadingBound && o.StoreRateBP >= 0 && o.StoreRateBP <= MaxStoreRateBound
}

// matchCoverLanes runs the two passes of each lane in ascending region (7.3.2): the route
// requests first, all of them, then the store requests. The limit of open policies ends a
// pass for its lane only (7.4 step 1).
func (w *World) matchCoverLanes() {
	in := w.insurance
	rd := &in.round
	if rd.form == nil {
		rd.form = w.formCover
	}
	for g := range rd.lanes {
		l := &rd.lanes[g]
		if len(l.routes) > 0 {
			insure.SortLoadings(l.offers)
			w.coverPass(RegionID(g), l.routes, l.offers, false)
		}
		if len(l.stores) > 0 {
			insure.SortStoreRates(l.offers)
			w.coverPass(RegionID(g), l.stores, l.offers, true)
		}
	}
}

// coverPass orders the requests of one kind of a lane, finds and values their candidates,
// matches them against the offers and counts what no offer covered.
func (w *World) coverPass(g RegionID, reqs []insure.Request, offers []insure.Offer, store bool) {
	rd := &w.insurance.round
	insure.SortRequests(reqs)
	w.coverValues(g)
	rd.cands, rd.rows = rd.cands[:0], rd.rows[:0]
	for i := range reqs {
		w.coverCandidates(g, &reqs[i], store)
	}
	rd.haltAt = len(rd.cands)
	insure.Match(rd.rows, offers, store, rd.form)
	c := &rd.counts[g]
	for i := range rd.cands {
		cd := &rd.cands[i]
		if cd.state == candidateCovered {
			continue
		}
		c.unfilled = addSat(c.unfilled, cd.sum)
		if cd.state == candidateOpen && i < rd.haltAt { // a candidate the scan did not reach is not a missing offer
			c.noOffer++
		}
	}
}

// coverValues sets the unit values of region g, by commodity, at the prices published
// before the round (spec-insurance.md 6.2.1), and whether the cap bound for each.
func (w *World) coverValues(g RegionID) {
	rd := &w.insurance.round
	n := len(w.cfg.Commodities)
	rd.capped = coverBools(rd.capped, n)
	rd.values = w.unitValues(g, rd.values[:0], rd.capped)
	rd.qty, rd.held = coverInts(rd.qty, n), coverInts(rd.held, n)
}

// coverInts returns s with length n and every entry 0, reusing its storage.
func coverInts(s []int64, n int) []int64 {
	if cap(s) < n {
		return make([]int64, n)
	}
	s = s[:n]
	clear(s)
	return s
}

// coverBools returns s with length n, reusing its storage.
func coverBools(s []bool, n int) []bool {
	if cap(s) < n {
		return make([]bool, n)
	}
	return s[:n]
}

// coverCandidates appends the candidates of an admitted request, in ascending shipment ID for
// a lane request and one for a store request, to the round's candidates and rows
// (spec-insurance.md 7.2.2). A request that finds none is counted NoShipment, a lane with an
// expected loss of 0 NoRisk, and a sum that passes the money bound Overflow; none of the
// three is scanned against offers.
func (w *World) coverCandidates(g RegionID, r *insure.Request, store bool) {
	rd := &w.insurance.round
	rq := &rd.rq[r.Ref]
	c := &rd.counts[g]
	if store {
		w.storeCandidate(g, r, rq)
		return
	}
	// The shipments of the tick are the tail of the list in ascending ID: walk it forward.
	ships := w.transport.ships
	start := len(ships)
	for start > 0 && ships[start-1].dispatched == w.Tick() {
		start--
	}
	found := false
	for i := start; i < len(ships); i++ {
		if s := &ships[i]; w.asked(s, g, &rq.req) {
			found = true
			w.routeCandidate(g, r, s)
		}
	}
	if !found {
		c.noShipment++
	}
}

// asked says the shipment is one a lane request asks to cover: the agent's own, on the lane
// of the request from the region of the round, and with no policy yet (7.2.2).
func (w *World) asked(s *shipment, g RegionID, req *coverRequest) bool {
	return s.shipper == req.agent && s.mode == req.lane.Mode && s.to == req.lane.To && s.from == g && !w.insurance.covered(s.id)
}

// routeCandidate values one shipment and appends its candidate, or counts why it is not
// one.
func (w *World) routeCandidate(g RegionID, r *insure.Request, s *shipment) {
	rd := &w.insurance.round
	c := &rd.counts[g]
	c.candidates++
	loss, _ := w.RouteLoss(s.from, s.to, s.mode)
	if loss == 0 {
		c.noRisk++
		return
	}
	sum, ok := w.routeSum(s)
	if !ok {
		c.overflow++
		return
	}
	w.addCandidate(r, s.id, sum, loss)
}

// routeSum is the sum insured of a shipment by its shape (spec-insurance.md 6.3.2): the
// lines term for a hired shipment, the account term for a trip to oneself, both for a
// self-carried shipment with lines. The units valued at the cap are counted. ok is false
// when a product or the sum passes MaxMoney.
func (w *World) routeSum(s *shipment) (sum int64, ok bool) {
	rd := &w.insurance.round
	var total int64
	if len(s.lines) > 0 {
		clear(rd.qty)
		for _, ln := range s.lines {
			rd.qty[ln.Commodity] = int64(ln.Quantity)
		}
		lines, ok := insure.SumLines(rd.qty, rd.values, MaxMoney)
		if !ok {
			return 0, false
		}
		total = lines
		w.countCapped(s.from)
	}
	if s.hired {
		return total, true
	}
	acct, vehicle := w.agents[s.traveller].account, int(w.transport.modes[s.mode].vehicle)
	w.coverHoldings(acct)
	held, ok := insure.SumAccount(rd.held, rd.values, vehicle, MaxMoney)
	if !ok {
		return 0, false
	}
	w.countCappedHeld(s.from, vehicle)
	if total, ok = addBounded(total, held); !ok {
		return 0, false
	}
	return total, true
}

// addBounded is a plus b and whether it is within MaxMoney.
func addBounded(a, b int64) (int64, bool) {
	s, ok := checked.Add(a, b)
	return s, ok && s <= MaxMoney
}

// coverHoldings fills the round's held units by commodity from an account.
func (w *World) coverHoldings(acct int) {
	rd := &w.insurance.round
	for c := range rd.held {
		held, _, err := w.ledger.Holding(acct, c)
		if err != nil {
			invariant("the ledger refused the holding of an insured account: " + err.Error())
		}
		rd.held[c] = held
	}
}

// countCapped adds to ValueCapped of region g the units of qty valued at the cap.
func (w *World) countCapped(g RegionID) {
	rd := &w.insurance.round
	c := &rd.counts[g]
	for i, q := range rd.qty {
		if rd.capped[i] {
			c.valueCapped = addSat(c.valueCapped, q)
		}
	}
}

// countCappedHeld adds the held units valued at the cap, but the vehicle's.
func (w *World) countCappedHeld(g RegionID, vehicle int) {
	rd := &w.insurance.round
	c := &rd.counts[g]
	for i, q := range rd.held {
		if rd.capped[i] && i != vehicle {
			c.valueCapped = addSat(c.valueCapped, q)
		}
	}
}

// storeCandidate appends the one candidate of a store request: the share of the value of
// the goods the agent holds in its region (spec-insurance.md 6.3.3).
func (w *World) storeCandidate(g RegionID, r *insure.Request, rq *admittedRequest) {
	rd := &w.insurance.round
	c := &rd.counts[g]
	c.candidates++
	w.coverHoldings(w.agents[rq.req.agent].account)
	sum, ok := insure.SumStore(rd.held, rd.values, rq.req.sc.ShareBP, MaxMoney)
	if !ok {
		c.overflow++
		return
	}
	for i, q := range rd.held {
		if rd.capped[i] {
			c.valueCapped = addSat(c.valueCapped, q)
		}
	}
	w.addCandidate(r, NoShipment, sum, 0)
}

// addCandidate appends a candidate and its row for the matching.
func (w *World) addCandidate(r *insure.Request, ship ShipmentID, sum, loss int64) {
	rd := &w.insurance.round
	rd.rows = append(rd.rows, insure.Candidate{Ref: len(rd.cands), Party: r.Party, Limit: r.Limit, MinCover: r.MinCover, Sum: sum})
	rd.cands = append(rd.cands, coverCandidate{req: r.Ref, ship: ship, sum: sum, loss: loss})
}

// formCover forms the policy of one tentative fill, in the order of 7.4: the policies limit,
// the minimum, the insured's funds, the overflow, then the premium and the policy. The
// ledger changes only for a policy that forms. It returns the outcome the matching steers by:
// Halted ends the pass of the lane, Failed ends the scan for the candidate and Dropped goes
// on to the next offer.
func (w *World) formCover(f insure.Fill) insure.Outcome {
	in := w.insurance
	rd := &in.round
	cd := &rd.cands[f.Candidate]
	rq, of := &rd.rq[cd.req], &rd.of[f.Offer]
	c := &rd.counts[rq.region]
	premium, rate, ok := w.coverPremium(cd, &of.o)
	insured, under := &w.agents[rq.req.agent], &w.agents[of.agent]
	iv, uv := w.accountView(insured.account), w.accountView(under.account)
	failure := insure.Check(insure.Form{
		Open: in.openCount(), MaxPolicies: in.cfg.MaxPolicies, Sum: cd.sum, MinSum: in.cfg.MinSum,
		Premium: premium, PremiumOK: ok, Available: iv.Balance - iv.Reserved, Balance: uv.Balance, Limit: MaxMoney,
	})
	switch failure {
	case insure.TooManyPolicies:
		c.tooMany++
		rd.haltAt = f.Candidate
		return insure.Halted
	case insure.BelowMinimum:
		c.belowMinimum++
		cd.state = candidateFailed
		return insure.Failed
	case insure.Funds:
		c.funds++
		cd.state = candidateFailed
		return insure.Failed
	case insure.Overflow:
		c.overflow++
		return insure.Dropped
	}
	// Check ruled out the refusals the ledger has for this transfer (funds, balance bound).
	mustClaim(w.ledger.Transfer(insured.account, under.account, premium, ledger.Premium), "a premium")
	p := coverPolicy{
		id: in.next, insured: insured.id, underwriter: under.id, shipment: cd.ship, sum: cd.sum,
		maxPayout: f.MaxPayout, coverBP: of.o.CoverBP, premium: premium, formed: w.Tick(),
	}
	p.kind = CoverRoute
	if rq.req.store {
		p.kind = CoverStore
	}
	in.next++
	in.add(p)
	cd.state = candidateCovered
	c.formed++
	c.sum, c.premium = addSat(c.sum, cd.sum), addSat(c.premium, premium)
	w.reportCoverFormed(&p, rate, rq, w.regionOf(under))
	return insure.Formed
}

// coverPremium is the premium and the rate of a fill (spec-insurance.md 6.4): a route
// policy's from the expected loss, the cover share and the loading, a store policy's from
// the sum and the store rate. ok is false when the quotient passes MaxMoney.
func (w *World) coverPremium(cd *coverCandidate, o *CoverOffer) (premium, rate int64, ok bool) {
	if cd.ship == NoShipment {
		premium, ok = insure.StorePremium(cd.sum, o.StoreRateBP, MaxMoney)
		return premium, o.StoreRateBP, ok
	}
	return insure.Premium(cd.sum, cd.loss, o.CoverBP, o.LoadingBP, MaxMoney)
}
