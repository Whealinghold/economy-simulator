"""Merchants on one lane: lot lock, spread of the fee share, error, cap (spec-merchants.md 7, 18, MER-A1).

A stylised model of the rules of the merchant, not of the library: floating point inside the
model only, `random.Random` with a fixed seed (the library draws from the world's source).
It extends ../inf/shippers.py (INF-A2) and ../trn/pipeline.py (TRN-A1) with the three
rules that are the merchant's own: a lot lock (no new lot on a lane while an own lot of the
lane is in flight), a one-sided spread of the bars (a bar is never below the freight) and a
cap of one lot per merchant per tick.

Units: the freight (the cost the merchant pays for a lot, per unit of gap) is f = 1.

    G[t]      the true price gap between the dear and the cheap region
    seen_j[t] = G[t - lag] + n_j[t]       the rumoured gap (lag ticks old) with an independent
                                          error n_j[t], uniform in [-E, E]
    ship_j    = 1 if seen_j > bar_j * f and the merchant has no lot in flight
    bar_j     = 1 + W * u_j               u_j in [0, 1): the fixed spread (W = 0: identical)
    a lot moves the gap by s = Q / N in all: a*s at dispatch (the purchase at the origin
    moves its price at once) and (1 - a)*s at arrival, d ticks later
    G[t+1]    = G[t] + h * (Gstar - G[t]) - s * (a * shipped[t] + (1 - a) * arrived[t])

h is the pull of the gap back to Gstar (production and consumption regenerate it). The
steady flow that holds the gap at 1 f is h * (Gstar - 1) f a tick. With a lot lock a
merchant that ships at t is free again at t + d (the arrival is processed in phase 1,
before the merchant's next Trade), so one merchant ships at most once in d ticks.

Measured: the peak-to-peak size of G over the last half of the run, in units of f, the mean
gap, and the mean number of lots a tick.
"""
import random

WEYL32 = 2654435769  # Knuth's multiplier, 2^32 / golden ratio: the one POP 3.8.3 uses for the start of a death carry
MASK32 = (1 << 32) - 1


def weyl_bp(agent_id):
    """The fixed spread of the spec (5.7): u(a) in basis points, 0 to 9999.

    (a * 2 654 435 769) mod 2^32, then floor(x * 10000 / 2^32): the Weyl offset of spec-population.md 3.8.3,
    the one place in the module where arithmetic is modular by design (a hash, not a quantity)."""
    x = (agent_id * WEYL32) & MASK32
    return (x * 10000) >> 32


def bars(n, w, kind, first_id=1000):
    if kind == "identical" or w == 0:
        return [1.0] * n
    if kind == "even":  # equally spaced, the ideal spread
        return [1.0 + w * j / n for j in range(n)]
    if kind == "weyl":  # the spec's rule over consecutive agent IDs
        return [1.0 + w * weyl_bp(first_id + j) / 10000 for j in range(n)]
    raise ValueError(kind)


def simulate(n=50, q_total=0.5, lag=2, d=2, a=0.0, w=0.0, kind="identical", err=0.0, lock=False,
             h=0.1, gstar=2.0, ticks=6000, seed=7, first_id=1000):
    rng = random.Random(seed)
    s = q_total / n
    bar = bars(n, w, kind, first_id)
    g = [gstar]
    sent = []
    busy = [0] * n  # a merchant with a lot in flight is free again at this tick
    total_lots = 0
    for t in range(ticks):
        count = 0
        for j in range(n):
            if lock and busy[j] > t:
                continue
            idx = t - lag
            seen = g[idx] if idx >= 0 else gstar
            if err > 0:
                seen += rng.uniform(-err, err)
            if seen > bar[j]:
                count += 1
                busy[j] = t + d
        sent.append(count)
        arrived = sent[t - d] if t - d >= 0 else 0
        g.append(g[t] + h * (gstar - g[t]) - s * (a * count + (1 - a) * arrived))
        if t >= ticks // 2:
            total_lots += count
    half = g[ticks // 2:]
    return max(half) - min(half), sum(half) / len(half), total_lots / (ticks - ticks // 2)


def row(label, **kw):
    size, mean, lots = simulate(**kw)
    print(f"{label:<52} size {size:6.3f} f   mean gap {mean:6.3f} f   lots a tick {lots:6.2f} of {kw.get('n', 50)}")


if __name__ == "__main__":
    print("N = 50 merchants, Q = 0.5 f, rumour lag 2, transit 2, h = 0.1, Gstar = 2 f, share at dispatch a = 0")
    print("(the steady flow that holds the gap at 1 f is 0.1 f a tick, 10 lots of 50; a bar is never below 1 f)")
    base = dict()
    row("identical bars (the herd of INF-A2)", **base)
    row("lot lock only", lock=True)
    for w in (0.25, 0.5, 1.0, 2.0):
        row(f"spread W = {w} (even), no lock", w=w, kind="even")
    for w in (0.25, 0.5, 1.0, 2.0):
        row(f"spread W = {w} (even) and lot lock", w=w, kind="even", lock=True)
    for w in (0.25, 0.5, 1.0, 2.0):
        row(f"spread W = {w} (Weyl of the agent ID) and lock", w=w, kind="weyl", lock=True)
    print()
    print("independent error E (uniform, +-E f), identical bars, with and without the lock")
    for e in (0.1, 0.4, 0.8):
        row(f"error E = {e} f, no lock", err=e)
        row(f"error E = {e} f, lock", err=e, lock=True)
    print()
    print("share of the effect at dispatch a = 0.5 (the purchase moves the origin price at once)")
    row("identical bars, a = 0.5", a=0.5)
    row("lock, a = 0.5", a=0.5, lock=True)
    row("spread W = 1 and lock, a = 0.5", a=0.5, w=1.0, kind="even", lock=True)
    print()
    print("longer total delay: rumour lag 6, transit 4")
    row("identical bars", lag=6, d=4)
    row("lock", lag=6, d=4, lock=True)
    row("spread W = 1 (even), no lock", lag=6, d=4, w=1.0, kind="even")
    row("spread W = 1 (even) and lock", lag=6, d=4, w=1.0, kind="even", lock=True)
    row("spread W = 2 (even) and lock", lag=6, d=4, w=2.0, kind="even", lock=True)
    print()
    print("a larger herd against the same steady flow: Q = 1.0 f (twice the lot size per merchant)")
    row("identical bars", q_total=1.0)
    row("lock", q_total=1.0, lock=True)
    row("spread W = 1 (even), lock", q_total=1.0, w=1.0, kind="even", lock=True)
    row("spread W = 2 (even), lock", q_total=1.0, w=2.0, kind="even", lock=True)
    print()
    print("a cap on the lot (D39: LotUnits): Q = 0.25 f and 0.1 f")
    for q in (0.25, 0.1):
        row(f"identical bars, Q = {q} f", q_total=q)
        row(f"spread W = 1 (even) and lock, Q = {q} f", q_total=q, w=1.0, kind="even", lock=True)
    print()
    print("number of merchants at a fixed Q (spread W = 1 even, lock, error 0.2 f)")
    for n in (5, 10, 20, 50, 200):
        row(f"N = {n}", n=n, w=1.0, kind="even", lock=True, err=0.2)
    print()
    print("eight seeds (1 to 8): lowest and highest size, spread W = 1 (even) and lock, error 0.2 f; identical with error 0.4 f")
    sizes = [simulate(w=1.0, kind="even", lock=True, err=0.2, seed=sd)[0] for sd in range(1, 9)]
    print(f"spread + lock + error 0.2 f   {min(sizes):5.2f} f to {max(sizes):5.2f} f")
    sizes = [simulate(err=0.4, seed=sd)[0] for sd in range(1, 9)]
    print(f"identical, error 0.4 f        {min(sizes):5.2f} f to {max(sizes):5.2f} f")
    print()
    print("the smallest spread W (steps of 0.05 f, even bars) that settles the gap (size under 0.1 f), Q = 0.5 f, and the slope condition Q / W < 2 sin(pi / (2 (2 n + 1))), n = lag + transit")
    import math
    for lag, d in ((2, 2), (4, 2), (2, 4), (6, 4), (8, 8)):
        limit = 2 * math.sin(math.pi / (2 * (2 * (lag + d) + 1)))
        found = {}
        for lk in (False, True):
            found[lk] = None
            for step in range(1, 61):
                w = round(step * 0.05, 2)
                if simulate(w=w, kind="even", lock=lk, lag=lag, d=d)[0] < 0.1:
                    found[lk] = w
                    break
        need = 0.5 / limit
        print(f"lag {lag}, transit {d}: limit on the slope {limit:5.3f}, so W above {need:5.2f} f; model: no lock {found[False]}, lock {found[True]}")
    print()
    print("the Weyl spread of the agent ID: how even are N merchants with consecutive IDs, and with IDs of every 7th agent")
    for n in (5, 10, 20, 50):
        for step, first in ((1, 1000), (7, 1003)):
            u = sorted(weyl_bp(first + step * j) for j in range(n))
            gaps = [u[0]] + [u[i + 1] - u[i] for i in range(n - 1)] + [10000 - u[-1]]
            print(f"N = {n:<3} IDs {first}+{step}j: largest gap {max(gaps):5d} bp, smallest {min(gaps):5d} bp, ideal {10000 // n} bp")
