"""A cobweb with and without an outside, spec-external-sector.md 17 (EXT-A2, EXT-A3).

A stylised model of the spec's rules, not of the library. It uses a *uniform* price
(the smallest integer price at which demand does not exceed supply) as a stand-in
for the belief and clearing dynamics: the library's midpoint rule and belief steps
are NOT modelled, so the numbers show the shape of the effect of a corridor, not
its size in the library.

Realm demand: Qd(P) = max(0, (140 - P) // 2 + 1) units. Realm supply is decided one
tick ahead: Q_t = clamp(P_{t-1} - 19, 0, 100) units (the unit j has cost 20 + j and is
made if the last price covers it), all offered at any price. Supply slope 1, demand
slope 1/2: the cobweb multiplier is 2 in magnitude, so it is unstable and ends on the
bounds of the price (here the reference band 20 to 120 of beliefs).

The outside sells to the realm at A (up to C units) and buys from the realm at B (up
to C units): if the realm-only price P* is above A the outside supplies the excess up
to C at A, and the price rises above A only for what C does not cover; the same below B.
"""
import sys

LO, HI = 20, 120      # the reference band standing in for the belief bounds


def qd(p):
    return max(0, (140 - p) // 2 + 1)


def clear(s):
    """Smallest price in [LO, HI] at which demand does not exceed supply s."""
    for p in range(LO, HI + 1):
        if qd(p) <= s:
            return p
    return HI


def tick(p_prev, outside):
    q = max(0, min(100, p_prev - 19))
    p_star = clear(q)
    imp = exp = 0
    p = p_star
    if outside:
        a, b, cap = outside
        if p_star > a:
            need = qd(a) - q
            imp = min(cap, need)
            p = a if imp == need else clear(q + cap)
        elif p_star < b:
            room = q - qd(b)
            exp = min(cap, room)
            p = b if exp == room else clear(q - cap)
    sold = qd(p)                                   # realm units bought at p
    unsold = max(0, q + imp - exp - sold)          # made, not bought (spoils)
    return p, q, imp, exp, unsold


def run(label, outside, ticks=400, p0=70):
    p, ps, qs, ims, exs, uns, bals, bal = p0, [], [], [], [], [], [], 0
    for _ in range(ticks):
        p, q, imp, exp, unsold = tick(p, outside)
        bal += (exp - imp) * p
        ps.append(p); qs.append(q); ims.append(imp); exs.append(exp); uns.append(unsold); bals.append(bal)
    t = slice(100, ticks)
    n = ticks - 100
    print(label)
    print(f"   price over ticks 100..{ticks - 1}: min {min(ps[t])}, max {max(ps[t])}, mean {sum(ps[t]) / n:.1f}")
    print(f"   units made: min {min(qs[t])}, max {max(qs[t])}; made and unsold per tick: mean {sum(uns[t]) / n:.1f}")
    print(f"   imports per tick: mean {sum(ims[t]) / n:.1f}; exports per tick: mean {sum(exs[t]) / n:.1f}")
    print(f"   cumulative money balance of trade: final {bals[-1]}, min {min(bals)}, max {max(bals)}")
    print(f"   first 12 prices: {ps[:12]}")


if __name__ == "__main__":
    print("realm-only equilibrium: Qd(P) = P - 19 at P = 60 (40 units)")
    run("realm alone", None)
    run("outside: ask 75, bid 45, capacity 40", (75, 45, 40))
    run("outside: ask 75, bid 45, capacity 10", (75, 45, 10))
    run("outside: ask 75, bid 45, capacity 3", (75, 45, 3))
    run("outside: ask 65, bid 55, capacity 40", (65, 55, 40))
    run("outside: ask 75, bid 45, capacity 40, start at equilibrium", (75, 45, 40), p0=60)
    run("realm alone, start at equilibrium", None, p0=60)
    sys.exit(0)
