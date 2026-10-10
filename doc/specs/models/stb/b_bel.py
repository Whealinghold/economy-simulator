"""Model (b): a stylised model of the RULES of spec-beliefs.md (not of code).
One commodity, one region. NS seller agents and NB buyer agents (weight w each). Integer arithmetic for
the rule; Python's random is used only as the model's source of uint64 draws.
Sellers produce Q per household per tick (fixed), buyers consume D per household per tick (fixed).
One order per agent per tick. Clearing: spec-clearing average clearing (greedy best-to-best, crossing check, midpoint half-to-even).
Beliefs: spec-beliefs 3-5 with the section-11 placeholder defaults.
Buyers have an optional willingness-to-pay cap WTP (money constraint stand-in)."""
import random, statistics
DEFAULTS = dict(REV=False, EvShare=5000, DIR=False, BeliefStep=200, BeliefAdjustProb=5000, BeliefSignificance=500, BeliefStockLow=5000,
                BeliefStockHigh=15000, BeliefMaxAge=3, BeliefWidthInit=2000, BeliefWidthMin=200,
                BeliefWidthMax=5000, BeliefWidthStep=250, CostFloor=10000, CostCeiling=30000,
                RefBandLow=1000, RefBandHigh=100000, PriceWindow=15)
MAXP = 10**9
def hte(num, den):  # half to even division, non-negative
    q, r = divmod(num, den)
    if 2*r > den or (2*r == den and q % 2 == 1): q += 1
    return q
def bounds(ref, cost, P):
    refLow = max(1, ref*P['RefBandLow']//10000)
    refHigh = min(MAXP, ref*P['RefBandHigh']//10000)
    costLow = cost*P['CostFloor']//10000 if cost else 0
    costHigh = cost*P['CostCeiling']//10000 if cost else MAXP
    fl = min(MAXP, max(refLow, costLow))
    ce = max(fl, min(refHigh, costHigh))
    return fl, ce
class Agent:
    __slots__ = ('bid','w','centre','width','stock','cost','wtp','target_cover','money_short','unmet','sold','lots')
def draw_limit(a, fl, ce, u):
    c = min(max(a.centre, fl), ce)
    half = c*a.width//10000
    lo = max(fl, c-half); hi = min(ce, c+half)
    n = hi-lo+1
    return lo + ((u*n) >> 64)
def update(a, fl, ce, o_limit, o_qty, o_fill, mp, age, bestbid, bestask, target, u, P):
    a.centre = min(max(a.centre, fl), ce)               # re-clamp
    hi, _ = divmod(u*10000, 1 << 64)
    if hi >= P['BeliefAdjustProb']:
        return 'closed'
    delta = max(1, a.centre*P['BeliefStep']//10000)
    fresh = age <= P['BeliefMaxAge']
    T = None
    unfilled = o_qty - o_fill
    sig = a.centre*P['BeliefSignificance']//10000
    if a.bid:
        if unfilled > 0:
            cands = []
            if bestask is not None: cands.append(bestask)
            if fresh and mp > o_limit: cands.append(mp)
            if cands: T = min(cands)
        else:
            if fresh and o_limit - mp > sig: T = mp
    else:
        if unfilled > 0:
            cands = []
            if bestbid is not None: cands.append(bestbid)
            if fresh and mp < o_limit: cands.append(mp)
            if cands: T = max(cands)
        else:
            if fresh and mp - o_limit > sig: T = mp
    ev = 0 if T is None else max(-delta, min(delta, T-a.centre))
    if P.get('REV'):
        de = max(1, delta*P['EvShare']//10000)
        gL = 0 if T is None else T - o_limit
        gC = 0 if T is None else T - a.centre
        if T is None or gL == 0 or gC == 0 or (gL > 0) != (gC > 0): ev = 0
        else: ev = (1 if gC > 0 else -1) * min(abs(gL), abs(gC), de)
    if P.get('DIR') and ev:
        if a.bid and unfilled > 0 and ev < 0: ev = 0
        if (not a.bid) and unfilled > 0 and ev > 0: ev = 0
        if a.bid and unfilled == 0 and ev > 0: ev = 0
        if (not a.bid) and unfilled == 0 and ev < 0: ev = 0
    v = 0
    if target > 0:
        ratio = a.stock*10000//target
        v = 1 if ratio < P['BeliefStockLow'] else (-1 if ratio > P['BeliefStockHigh'] else 0)
    a.centre = min(max(a.centre + ev + v*delta, fl), ce)
    share = o_fill*10000//o_qty
    a.width = a.width - P['BeliefWidthStep'] if share >= 5000 else a.width + P['BeliefWidthStep']
    a.width = min(max(a.width, P['BeliefWidthMin']), P['BeliefWidthMax'])
    return 'moved'
def simulate(seed=0, ticks=600, NS=20, NB=20, w=1, Q=10, D=10, ref=1000, seller_cost=0, wtp=None,
             P=None, cover_b=3, cover_s=2, shocks=None, init_s=None, init_b=None, shelf=None,
             rec_beliefs=False, cost_by_seller=None, prod_stop=None, ed=0.0, es=0.0):
    """shocks: dict tick-> (D_mult_bp, Q_mult_bp) applied from that tick. returns dict of series."""
    P = dict(DEFAULTS, **(P or {}))
    rng = random.Random(seed)
    agents = []
    fl_s, ce_s = bounds(ref, seller_cost, P)
    fl_b, ce_b = bounds(ref, 0, P)
    if wtp: ce_b = min(ce_b, wtp); ce_b = max(ce_b, fl_b)
    def mk(bid):
        a = Agent(); a.bid = bid; a.w = w; a.width = P['BeliefWidthInit']
        start = (init_b if bid else init_s) or ref
        fl, ce = (fl_b, ce_b) if bid else (fl_s, ce_s)
        a.centre = min(max(start, fl), ce); a.stock = 0; a.unmet = 0; a.sold = 0
        a.lots = [] ; return a
    sellers = [mk(False) for _ in range(NS)]
    buyers = [mk(True) for _ in range(NB)]
    for b in buyers: b.stock = cover_b*D*w
    mp, age = ref, 0
    dm, qm = 10000, 10000
    ps, vol, short, atb, waste, last = [], [], [], [], [], None; cs = []; stk = []
    spoiled_total = 0; produced_total = 0
    for t in range(ticks):
        if shocks and t in shocks: dm, qm = shocks[t]
        Dt = int(D*dm//10000*(ref/mp)**ed); Qt = int(Q*qm//10000*(mp/ref)**es)
        for s in sellers:
            if prod_stop and s.stock*10000 >= prod_stop*cover_s*Q*w: continue
            s.stock += Qt*w; produced_total += Qt*w
            if shelf: s.lots.append([t, Qt*w])
        orders = []
        for b in buyers:
            qty = max(0, cover_b*Dt*w - b.stock)
            u = rng.getrandbits(64)
            if qty > 0:
                orders.append([b, True, draw_limit(b, fl_b, ce_b, u), qty, 0, rng.random()])
        for s in sellers:
            u = rng.getrandbits(64)
            if s.stock > 0:
                orders.append([s, False, draw_limit(s, fl_s, ce_s, u), s.stock, 0, rng.random()])
        bids = sorted([o for o in orders if o[1]], key=lambda o: (-o[2], o[5]))
        asks = sorted([o for o in orders if not o[1]], key=lambda o: (o[2], o[5]))
        i = j = 0; vwn = 0; vwd = 0
        while i < len(bids) and j < len(asks):
            bo, ao = bids[i], asks[j]
            if bo[2] < ao[2]: break
            q = min(bo[3]-bo[4], ao[3]-ao[4])
            p = ao[2] + (bo[2]-ao[2])//2 if (bo[2]-ao[2]) % 2 == 0 else ao[2] + (bo[2]-ao[2]) // 2 + (1 if (ao[2] + (bo[2]-ao[2])//2) % 2 == 1 else 0)
            vwn += p*q; vwd += q
            bo[4] += q; ao[4] += q
            bo[0].stock += q; ao[0].stock -= q
            if bo[4] == bo[3]: i += 1
            if ao[4] == ao[3]: j += 1
        if vwd: mp, age = hte(vwn, vwd), 0
        else: age += 1
        bb = max((o[2] for o in bids if o[4] < o[3]), default=None)
        ba = min((o[2] for o in asks if o[4] < o[3]), default=None)
        # consume the lots sold (oldest first)
        if shelf:
            for ao in asks:
                s = ao[0]; need = ao[4]
                while need > 0 and s.lots:
                    take = min(need, s.lots[0][1]); s.lots[0][1] -= take; need -= take
                    if s.lots[0][1] == 0: s.lots.pop(0)
        # belief updates, in admission order (bids then asks), one u per outcome
        nb_floor = 0
        for o in bids + asks:
            a = o[0]
            fl, ce = (fl_b, ce_b) if a.bid else (fl_s, ce_s)
            target = cover_b*Dt*w if a.bid else cover_s*Qt*w
            u = rng.getrandbits(64)
            update(a, fl, ce, o[2], o[3], o[4], mp, age, bb, ba, target, u, P)
            if a.centre == fl and not a.bid and fl > 1: nb_floor += 1
        # consumption and spoilage
        un = 0
        for b in buyers:
            c = Dt*w
            use = min(c, b.stock); b.stock -= use; un += c-use
        sp = 0
        if shelf:
            for s in sellers:
                keep = []
                for lot in s.lots:
                    if t - lot[0] + 1 >= shelf: sp += lot[1]; s.stock -= lot[1]
                    else: keep.append(lot)
                s.lots = keep
        spoiled_total += sp
        cs.append((sum(s.centre for s in sellers)//NS, sum(b.centre for b in buyers)//NB, sum(s.width for s in sellers)//NS)); stk.append((sum(s.stock for s in sellers), sum(b.stock for b in buyers))); ps.append(mp); vol.append(vwd); short.append(un); atb.append(nb_floor)
    return dict(stk=stk, cs=cs, p=ps, vol=vol, short=short, atfloor=atb, spoiled=spoiled_total, produced=produced_total)
def stats(res, burn=200):
    p = res['p'][burn:]
    m = statistics.mean(p)
    sd = statistics.pstdev(p)
    ch = statistics.mean(abs(p[i]-p[i-1]) for i in range(1, len(p)))
    notrade = sum(1 for v in res['vol'][burn:] if v == 0)/len(p)
    sh = sum(res['short'][burn:])/len(p)
    return dict(mean=m, relsd=sd/m, relchg=ch/m, notrade=notrade, short=sh)
if __name__ == "__main__":
    r = simulate(seed=1)
    print(stats(r)); print(r['p'][:20], r['p'][-5:])
