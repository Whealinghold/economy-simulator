"""Integer model of the counterparty cycle of spec-external-sector.md (sections 4 to 8).

A prototype of the rules of the spec on toy accounts, not of the library. It
shows (1) the arithmetic of the worked example of 18.1, (2) that the outside
counterparties hold nothing at a tick boundary and every unit and every minor unit
moves by a named reason, (3) the identity  M_t - M_0 = sum(budget - unspent -
receipts)  (the balance of trade, 8.4), and (4) a cumulative cap that ends a line.
Python's random.Random stands in for the world's seeded source; the numbers are
placeholders and the draws are not the library's.
"""
import random
import sys

MAX_PRICE = 10**9


def half_even_div(n, d):
    """n / d rounded half to even, n >= 0, d > 0."""
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q


def midpoint(a, b):
    """Midpoint of limits a <= b rounded half to even (spec-clearing.md 5.2)."""
    d = b - a
    m = a + d // 2
    if d % 2 == 1 and m % 2 == 1:
        m += 1
    return m


def trunc_div(n, d):
    """Division truncating toward zero (architecture.md 3.5)."""
    q = abs(n) // abs(d)
    return q if (n >= 0) == (d > 0) else -q


class Ledger:
    """Accounts that cannot go negative plus signed boundaries per (reason, thing)."""

    def __init__(self):
        self.acc = {}       # name -> {"m": money, "g": {commodity: qty}}
        self.bnd = {}       # (reason, thing) -> signed balance; thing is "money" or a commodity
        self.totals = {}    # (reason, thing) -> sum moved this tick (absolute)

    def open(self, name):
        self.acc[name] = {"m": 0, "g": {}}

    def _note(self, reason, thing, amount):
        k = (reason, thing)
        self.totals[k] = self.totals.get(k, 0) + amount

    def money(self, frm, to, amount, reason, kind):
        """kind: internal, faucet (frm is a boundary), sink (to is a boundary)."""
        assert amount >= 1
        if kind == "faucet":
            self.bnd[(reason, "money")] = self.bnd.get((reason, "money"), 0) - amount
        else:
            assert self.acc[frm]["m"] >= amount, (frm, amount)
            self.acc[frm]["m"] -= amount
        if kind == "sink":
            self.bnd[(reason, "money")] = self.bnd.get((reason, "money"), 0) + amount
        else:
            self.acc[to]["m"] += amount
        self._note(reason, "money", amount)

    def goods(self, frm, to, c, q, reason, kind):
        assert q >= 1
        if kind == "faucet":
            self.bnd[(reason, c)] = self.bnd.get((reason, c), 0) - q
        else:
            assert self.acc[frm]["g"].get(c, 0) >= q, (frm, c, q)
            self.acc[frm]["g"][c] -= q
        if kind == "sink":
            self.bnd[(reason, c)] = self.bnd.get((reason, c), 0) + q
        else:
            self.acc[to]["g"][c] = self.acc[to]["g"].get(c, 0) + q
        self._note(reason, c, q)

    def total(self, thing):
        if thing == "money":
            a = sum(x["m"] for x in self.acc.values())
        else:
            a = sum(x["g"].get(thing, 0) for x in self.acc.values())
        b = sum(v for (r, t), v in self.bnd.items() if t == thing)
        return a, b


def average_clear(bids, asks):
    """AverageClearing over (agent, limit, qty); ties by input position."""
    B = sorted(enumerate(bids), key=lambda x: (-x[1][1], x[0]))
    A = sorted(enumerate(asks), key=lambda x: (x[1][1], x[0]))
    B = [[i, o[0], o[1], o[2]] for i, o in B]
    A = [[i, o[0], o[1], o[2]] for i, o in A]
    trades = []
    while B and A and B[0][2] >= A[0][2]:
        q = min(B[0][3], A[0][3])
        p = midpoint(A[0][2], B[0][2])
        trades.append((B[0][1], A[0][1], p, q))
        B[0][3] -= q
        A[0][3] -= q
        if B[0][3] == 0:
            B.pop(0)
        if A[0][3] == 0:
            A.pop(0)
    return trades


class Line:
    """One (node, commodity) line with up to two sides (spec 3.3, 6, 7)."""

    def __init__(self, commodity, base, ask_margin=None, bid_margin=None, cap_sell=0, cap_buy=0,
                 price_season=None, window_sell=None, window_buy=None, trend_bp=0,
                 floor_bp=1000, ceil_bp=100000, walk_step=0, walk_revert=0, walk_bound=0,
                 total_units_sell=0, total_units_buy=0, total_value_sell=0, total_value_buy=0):
        self.c, self.base = commodity, base
        self.ask_margin, self.bid_margin = ask_margin, bid_margin
        self.cap = {"sell": cap_sell, "buy": cap_buy}
        self.price_season = price_season or {}
        self.window = {"sell": window_sell or {}, "buy": window_buy or {}}
        self.trend_bp, self.floor_bp, self.ceil_bp = trend_bp, floor_bp, ceil_bp
        self.walk_step, self.walk_revert, self.walk_bound = walk_step, walk_revert, walk_bound
        self.drift = 0
        self.carry = {"sell": 0, "buy": 0}
        self.units = {"sell": 0, "buy": 0}
        self.value = {"sell": 0, "buy": 0}      # list value: units traded x the quote price (spec 7.4)
        self.real = {"sell": 0, "buy": 0}       # realised money, for the report
        self.total_units = {"sell": total_units_sell, "buy": total_units_buy}
        self.total_value = {"sell": total_value_sell, "buy": total_value_buy}
        self.ended = {"sell": False, "buy": False}

    def update(self, rng):
        """Phase 1: one draw per line with a walk, in (node, commodity) order."""
        if self.walk_step > 0:
            d = rng.randrange(2 * self.walk_step + 1) - self.walk_step
            drift = self.drift - trunc_div(self.drift * self.walk_revert, 10000) + d
            self.drift = max(-self.walk_bound, min(self.walk_bound, drift))

    def mid(self, t, ticks_per_year, season):
        f_trend = 10000 + trunc_div(self.trend_bp * t, ticks_per_year)
        f = half_even_div(max(f_trend, 0) * (10000 + self.drift), 10000)
        f = max(self.floor_bp, min(self.ceil_bp, f))
        seas = self.price_season.get(season, 10000)
        m = half_even_div(self.base * seas * f, 10000 * 10000)
        return max(1, min(MAX_PRICE, m))

    def quote(self, side, mid):
        if side == "sell":
            return -((-mid * (10000 + self.ask_margin)) // 10000)      # ceiling
        return (mid * (10000 - self.bid_margin)) // 10000               # floor

    def wanted(self, side, season):
        avail = self.window[side].get(season, 10000)
        num = self.cap[side] * avail + self.carry[side]
        return num // 10000, num % 10000


def run_tick(L, t, rng, nodes_lines, realm_orders, season, tpy, log=None):
    """One tick of the counterparty cycle over one gate region and one market per line.

    realm_orders(c) -> (bids, asks) lists of (agent, limit, qty) from realm agents.
    """
    L.totals = {}
    for ln in nodes_lines:
        ln.update(rng)                                                    # phase 1
    out = []
    for ln in nodes_lines:                                                # phase 3
        mid = ln.mid(t, tpy, season)
        bids, asks = realm_orders(ln.c)
        placed = {}
        for side in ("buy", "sell"):
            if (side == "sell" and ln.ask_margin is None) or (side == "buy" and ln.bid_margin is None):
                continue
            if ln.ended[side]:
                continue
            price = ln.quote(side, mid)
            if price < 1:
                continue
            want, newcarry = ln.wanted(side, season)
            offered = want
            cut_units = cut_value = 0
            if ln.total_units[side]:
                rem = ln.total_units[side] - ln.units[side]
                if offered > rem:
                    cut_units, offered = offered - rem, rem
            if ln.total_value[side]:
                rem_v = ln.total_value[side] - ln.value[side]
                maxq = rem_v // price
                if offered > maxq:
                    cut_value, offered = offered - maxq, maxq
            ln.carry[side] = newcarry
            if offered < 1:
                continue
            name = "OB" if side == "buy" else "OS"
            if side == "buy":
                L.money(None, name, price * offered, "outside_budget", "faucet")
                bids = bids + [(name, price, offered)]
            else:
                L.goods(None, name, ln.c, offered, "outside_offer", "faucet")
                asks = asks + [(name, price, offered)]
            placed[side] = (price, offered, want, cut_units, cut_value)
        trades = average_clear(bids, asks)                                # phase 4
        traded = {"buy": 0, "sell": 0}
        value = {"buy": 0, "sell": 0}
        for buyer, seller, p, q in trades:
            L.money(buyer, seller, p * q, "trade", "internal")
            L.goods(seller, buyer, ln.c, q, "trade", "internal")
            if buyer == "OB":
                traded["buy"] += q
                value["buy"] += p * q
            if seller == "OS":
                traded["sell"] += q
                value["sell"] += p * q
        # sweep (phase 4, after every market has cleared)
        if L.acc["OB"]["m"]:
            L.money("OB", None, L.acc["OB"]["m"], "outside_unspent", "sink")
        if L.acc["OB"]["g"].get(ln.c, 0):
            L.goods("OB", None, ln.c, L.acc["OB"]["g"][ln.c], "outside_purchases", "sink")
        if L.acc["OS"]["g"].get(ln.c, 0):
            L.goods("OS", None, ln.c, L.acc["OS"]["g"][ln.c], "outside_unsold", "sink")
        if L.acc["OS"]["m"]:
            L.money("OS", None, L.acc["OS"]["m"], "outside_receipts", "sink")
        for side in ("buy", "sell"):
            ln.units[side] += traded[side]
            ln.value[side] += traded[side] * placed.get(side, (0,))[0]
            ln.real[side] += value[side]
            if ln.total_units[side] and ln.units[side] >= ln.total_units[side] and not ln.ended[side]:
                ln.ended[side] = True
                out.append(("ended", ln.c, side, "units", t))
            if ln.total_value[side] and ln.total_value[side] - ln.value[side] < (placed.get(side, (1,))[0]) and not ln.ended[side]:
                ln.ended[side] = True
                out.append(("ended", ln.c, side, "value", t))
        out.append(("line", ln.c, t, mid, placed, traded, value, trades))
    # tick-boundary invariants
    for name in ("OB", "OS"):
        assert L.acc[name]["m"] == 0 and all(v == 0 for v in L.acc[name]["g"].values()), name
    for thing in ["money"] + sorted({ln.c for ln in nodes_lines}):
        a, b = L.total(thing)
        assert a + b == 0, (thing, a, b)
    return out


def worked_example():
    print("== 18.1 worked tick (example A, one gate, tick 0) ==")
    L = Ledger()
    for n in ("OB", "OS", "g1", "g2", "g3", "g4", "s1", "s2", "s3"):
        L.open(n)
    # realm sellers of grain hold stock from a recipe (not part of the outside)
    for n, q in (("g1", 20), ("g2", 15), ("g3", 10), ("g4", 30)):
        L.goods(None, n, "grain", q, "recipe_output", "faucet")
    for n in ("s1", "s2", "s3"):
        L.money(None, n, 1000, "initial", "faucet")
    grain = Line("grain", 20, bid_margin=1000, cap_buy=50)
    salt = Line("salt", 40, ask_margin=500, cap_sell=30)
    m0 = L.total("money")[0]

    def realm(c):
        if c == "grain":
            return [], [("g1", 15, 20), ("g2", 17, 15), ("g3", 19, 10), ("g4", 21, 30)]
        return [("s1", 50, 10), ("s2", 45, 10), ("s3", 41, 10)], []

    out = run_tick(L, 0, random.Random(1), [grain, salt], realm, season=0, tpy=360)
    for o in out:
        if o[0] == "line":
            _, c, t, mid, placed, traded, value, trades = o
            print(f"{c}: mid {mid}, placed {placed}")
            for tr in trades:
                print(f"   trade buyer {tr[0]} seller {tr[1]} price {tr[2]} qty {tr[3]}")
            print(f"   traded {traded} value {value}")
    print("flows this tick (reason, thing): amount")
    for k in sorted(L.totals):
        print(f"   {k[0]:18s} {k[1]:6s} {L.totals[k]}")
    print("boundary balances:")
    for k in sorted(L.bnd):
        if k[0].startswith("outside"):
            print(f"   {k[0]:18s} {k[1]:6s} {L.bnd[k]}")
    m1 = L.total("money")[0]
    print(f"money in the realm {m0} -> {m1}, change {m1 - m0}")


def run_a(ticks=720):
    print("\n== 18.2 two years of example A with a walk on grain, 4 seasons of 90 ticks ==")
    tpy = 360
    rng = random.Random(7)
    gen = random.Random(11)
    L = Ledger()
    for n in ("OB", "OS"):
        L.open(n)
    sellers = [f"gs{i}" for i in range(6)]
    buyers = [f"sb{i}" for i in range(4)]
    for n in sellers + buyers:
        L.open(n)
    for n in buyers:
        L.money(None, n, 10**7, "initial", "faucet")
    grain = Line("grain", 20, bid_margin=1000, cap_buy=50,
                 window_buy={0: 10000, 1: 10000, 2: 10000, 3: 6000},
                 walk_step=100, walk_revert=500, walk_bound=2000)
    salt = Line("salt", 40, ask_margin=500, cap_sell=30, trend_bp=-300)
    lines = [grain, salt]
    m0 = L.total("money")[0]
    flows = {}
    ended = []
    drifts = []
    for t in range(ticks):
        season = (t % tpy) // 90
        for s in sellers:
            L.goods(None, s, "grain", 15, "recipe_output", "faucet")

        def realm(c):
            if c == "grain":
                asks = [(s, 13 + 2 * i + gen.randrange(4), 15) for i, s in enumerate(sellers)]
                return [], asks
            bids = [(b, 38 + 3 * i + gen.randrange(8), 10) for i, b in enumerate(buyers)]
            return bids, []

        out = run_tick(L, t, rng, lines, realm, season, tpy)
        for o in out:
            if o[0] == "ended":
                ended.append(o)
        for k, v in L.totals.items():
            flows[k] = flows.get(k, 0) + v
        drifts.append(grain.drift)
        # sellers' unsold grain spoils so that stock does not grow (a stand-in sink)
        for s in sellers:
            q = L.acc[s]["g"].get("grain", 0)
            if q:
                L.goods(s, None, "grain", q, "spoilage", "sink")
    print("cumulative flows by reason and thing:")
    for k in sorted(flows):
        if k[0].startswith("outside"):
            print(f"   {k[0]:18s} {k[1]:6s} {flows[k]}")
    budget = flows[("outside_budget", "money")]
    unspent = flows[("outside_unspent", "money")]
    receipts = flows[("outside_receipts", "money")]
    export_revenue = budget - unspent
    m1 = L.total("money")[0]
    print(f"exports (money in) {export_revenue}, imports (money out) {receipts}, balance {export_revenue - receipts}")
    print(f"money in the realm {m0} -> {m1}, change {m1 - m0}")
    assert m1 - m0 == export_revenue - receipts
    print(f"drift of the grain walk: min {min(drifts)}, max {max(drifts)} (bound 2000)")
    print("lines ended:", ended)


def run_b():
    print("\n== 18.3 a cumulative cap ends a line (example B) ==")
    rng = random.Random(3)
    L = Ledger()
    for n in ("OB", "OS", "m1", "m2"):
        L.open(n)
    L.goods(None, "m1", "ore", 10**6, "recipe_output", "faucet")
    L.money(None, "m2", 10**7, "initial", "faucet")
    ore = Line("ore", 120, bid_margin=2000, cap_buy=200, total_units_buy=1500)
    reagent = Line("reagent", 300, ask_margin=800, cap_sell=10, total_value_sell=12000, trend_bp=-200)
    ended = []
    cuts = []
    for t in range(40):
        def realm(c):
            if c == "ore":
                return [], [("m1", 90, 200)]
            return [("m2", 400, 10)], []
        out = run_tick(L, t, rng, [ore, reagent], realm, 0, 360)
        for o in out:
            if o[0] == "ended":
                ended.append(o)
            if o[0] == "line" and o[4]:
                for side, v in o[4].items():
                    if v[3] or v[4]:
                        cuts.append((t, o[1], side, v))
    print("ended:", ended)
    print("first cut records (tick, commodity, side, (price, offered, wanted, cut_units, cut_value)):", cuts[:3])
    print(f"ore units sold {ore.units['buy']} of cap 1500; reagent list value {reagent.value['sell']} of cap 12000 (realised money {reagent.real['sell']})")
    b = {k[0]: v for k, v in L.bnd.items() if k[1] == "money"}
    exp_v = -b["outside_budget"] - b["outside_unspent"]     # the budget balance is negative (a faucet)
    print(f"export value {exp_v}, import value {b['outside_receipts']}, balance of trade {exp_v - b['outside_receipts']}")


if __name__ == "__main__":
    worked_example()
    run_a()
    run_b()
    print("\nall identities held (zero total each thing, counterparties empty at every tick boundary)")
    sys.exit(0)
