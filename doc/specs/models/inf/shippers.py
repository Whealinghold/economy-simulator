"""Threshold shippers reading rumours: noise and spread as dispersion (spec-information.md 11, INF-A2).

A stylised model of herding: N shippers in the cheap region A each ship one lot when the
rumoured gap exceeds the freight. It is a model of the rule, not of the library; floats
inside the model only, and `random.Random` with a fixed seed (the library's draws come
from the world's source, spec 7).

Units: the freight is f = 1. A lot moves the gap by s. Q = N * s is what every shipper
shipping at once would move per tick.

    obs_j[t] = G[t - lag - j_off] + n_j[t]          the rumour of the gap, with a vintage
                                                    offset j_off of the shipper (0..S) and
                                                    an error n_j[t] (uniform in [-E, E])
    ship_j   = 1 if obs_j[t] > f else 0
    G[t+1]   = G[t] + h * (Gstar - G[t]) - s * (lots that arrive at t)     a lot sent at t
                                                    arrives at t + d

h is the pull of the gap back towards Gstar (the markets regenerate their gap: production,
consumption); without it a one-sided gap would simply end below the freight. The steady
flow needed to hold the gap at f is h * (Gstar - f) per tick.

Measured: the peak-to-peak size of G over the last half of the run, in units of f, and the
mean number of lots per tick. A gap that stays near f and moves little has a small size.
Also a static spread of the shippers' own thresholds (`margin_w`: no draw per tick).
Configurations: identical shippers (no error, no spread), spread only, error only, one
error shared by all shippers of the tick (a common shock), error and spread together,
and the effect of N at a fixed Q.
"""
import random


def simulate(n_ship, q_total, lag, d, spread, err, shared=False, h=0.1, gstar=2.0, f=1.0,
             ticks=6000, seed=7, margin_w=0.0):
    rng = random.Random(seed)
    s = q_total / n_ship
    g = [gstar]
    sent = []
    lots = 0
    for t in range(ticks):
        common = rng.uniform(-err, err) if shared else 0.0
        count = 0
        for j in range(n_ship):
            off = j % (spread + 1)
            idx = t - lag - off
            seen = g[idx] if idx >= 0 else gstar
            noise = common if shared else (rng.uniform(-err, err) if err > 0 else 0.0)
            bar = f * (1 - margin_w + 2 * margin_w * j / max(1, n_ship - 1))
            if seen + noise > bar:
                count += 1
        sent.append(count)
        arrived = sent[t - d] if t - d >= 0 else 0
        g.append(g[t] + h * (gstar - g[t]) - s * arrived)
        if t >= ticks // 2:
            lots += count
    half = g[ticks // 2:]
    size = max(half) - min(half)
    return size, lots / (ticks - ticks // 2)


def row(label, **kw):
    size, lots = simulate(**kw)
    print(f"{label:<44} size of the gap {size:6.3f} f   lots per tick {lots:6.2f} of {kw['n_ship']}")


if __name__ == "__main__":
    base = dict(n_ship=50, q_total=0.5, lag=2, d=2, spread=0, err=0.0)
    print("N = 50 shippers, Q = 0.5 f, lag 2, transit 2, h = 0.1, Gstar = 2 f")
    print("(the steady flow that holds the gap at f is 0.1 f a tick, 10 lots of 50)")
    row("identical shippers", **base)
    for s in (1, 2, 4, 8):
        row(f"spread S = {s}", **{**base, "spread": s})
    for e in (0.05, 0.1, 0.2, 0.4, 0.8):
        row(f"independent error E = {e} f", **{**base, "err": e})
    for e in (0.2, 0.8):
        row(f"shared error E = {e} f (one draw per tick)", **{**base, "err": e, "shared": True})
    row("spread S = 2 and error E = 0.2 f", **{**base, "spread": 2, "err": 0.2})
    row("spread S = 4 and error E = 0.4 f", **{**base, "spread": 4, "err": 0.4})
    print()
    print("a static spread of the shippers' own thresholds (no error, no spread of vintage): the bar of shipper j is\n f * (1 - w + 2 w j / (N - 1)), fixed, no draw")
    for w in (0.1, 0.2, 0.4, 0.8):
        row(f"thresholds spread w = {w}", **{**base, "margin_w": w})
    print()
    print("the same Q = 0.5 f shared among more shippers (error E = 0.2 f, spread 0)")
    for n in (5, 20, 50, 200, 1000):
        row(f"N = {n}", **{**base, "n_ship": n, "err": 0.2})
    print()
    print("a smaller Q (the cap on what the shippers can ship, D39): identical shippers")
    for q in (0.5, 0.25, 0.1, 0.05):
        row(f"Q = {q} f", **{**base, "q_total": q})
    print()
    print("longer total delay (lag 6, transit 4), E = 0.4 f, spread 0 and 4")
    row("lag 6, transit 4, E = 0.4 f", **{**base, "lag": 6, "d": 4, "err": 0.4})
    row("lag 6, transit 4, E = 0.4 f, S = 4", **{**base, "lag": 6, "d": 4, "err": 0.4, "spread": 4})
    print()
    print("the condition Q / (2 E) < 2 sin(pi / (2 (2 n + 1))), n = lag + transit = 4 (limit 0.346):")
    print("size of the gap for the error E and the static spread w that give the same slope")
    for q in (0.25, 0.5):
        for width in (0.2, 0.4, 0.8):
            slope = q / (2 * width)
            e = simulate(**{**base, "q_total": q, "err": width})[0]
            w = simulate(**{**base, "q_total": q, "margin_w": width})[0]
            print(f"Q = {q:<4} E or w = {width:<3} slope {slope:5.3f} "
                  f"{'above' if slope > 0.346 else 'below'} the limit: error {e:5.3f} f, thresholds {w:5.3f} f")
    print()
    print("eight seeds (1 to 8) for the rows with an error: lowest and highest size of the gap")
    for label, kw in (("E = 0.2 f", dict(err=0.2)), ("E = 0.4 f", dict(err=0.4)), ("E = 0.8 f", dict(err=0.8)),
                      ("shared E = 0.2 f", dict(err=0.2, shared=True)), ("shared E = 0.8 f", dict(err=0.8, shared=True))):
        sizes = [simulate(**{**base, **kw, "seed": sd})[0] for sd in range(1, 9)]
        print(f"{label:<20} {min(sizes):5.2f} f to {max(sizes):5.2f} f")
