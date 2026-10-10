"""MIG model 1: the migration loop of spec-migration.md 8 and 10, in a few regions.

A stylised model of the RULES of the spec, not of the library, and not a claim that any default is stable
(notes N-015, N-027). The migration rules are the spec's integer rules: a review rotation by agent ID, the settling
rule, the push gate, the destination estimate from a rumoured income, the horizon discount, the margin with a
per-agent width from the Weyl offset, the outflow cap, the cap on the households on the road to a destination, and the
timed road with the arrival at t + 1 + k. The economy is replaced by ONE assumed function: the income per working
tick of a region, in basis points of the upkeep, is V_r(n) = A_r * (n0 / n)^eta, where n is the households of the
region that are NOT on the road (a fixed factor with diminishing returns). Floating point is used ONLY for that
function and for the printouts. A rumour of a region is its V as it was `Lag` ticks ago with an independent bounded
error (spec-information.md 5). A household observes the mean V of the working ticks of its own window; the ticks on
the road are not working ticks (spec-welfare-and-wages.md 4.3). Every number is a placeholder."""
import math, random, sys
from collections import deque

M32 = 2 ** 32
def weyl(i):                      # POP 3.8.3: a fixed function of the ID, in [0, 2^32)
    return (i * 2654435769) % M32
def cdiv(a, b):
    return -(-a // b)

class P:
    """The parameters of one run. Names follow spec-migration.md 8."""
    def __init__(self, **kw):
        self.N = 1000; self.T = 30; self.ticks = 2400
        self.R = 30; self.below = 10**9; self.gain = 1000; self.spread = 5000
        self.cool = 60; self.H = 180; self.out_share = 50; self.in_share = 100
        self.move_speed = 40; self.info_speed = 40; self.E = 500; self.entry_ratio = 8000
        self.eta = 1.0; self.work = 6800          # work share in bp, for the push gate only
        self.A = [10000, 10000]; self.split = None
        self.D = {(0, 1): 80, (1, 0): 80}
        self.seed = 1; self.use_caps = True; self.settle = True; self.abs_spread = True
        self.__dict__.update(kw)

def run(p):
    rng = random.Random(p.seed)
    nreg = len(p.A)
    n0 = p.N / nreg
    split = p.split or [p.N // nreg] * nreg
    region = []
    for r in range(nreg): region += [r] * split[r]
    region += [nreg - 1] * (p.N - len(region))
    since = [-10**9] * p.N
    road = [None] * p.N                                  # (to, arrives)
    ring = [deque() for _ in range(p.N)]                 # (working, V) per tick, at most T
    sumV = [0] * p.N; nw = [0] * p.N
    Vhist = [[] for _ in range(nreg)]
    series = [[] for _ in range(nreg)]; flight = [[] for _ in range(nreg)]
    started = 0
    blocked = {"out": 0, "in": 0, "gain": 0, "viable": 0, "far": 0, "rumour": 0}
    for t in range(p.ticks):
        for i in range(p.N):                              # phase 1: arrivals
            if road[i] is not None and road[i][1] <= t:
                region[i] = road[i][0]; since[i] = t; road[i] = None
        nwork = [0] * nreg
        for i in range(p.N):
            if road[i] is None: nwork[region[i]] += 1
        V = [int(p.A[r] * (n0 / max(1, nwork[r])) ** p.eta) for r in range(nreg)]
        for r in range(nreg): Vhist[r].append(V[r])
        for i in range(p.N):                              # phase 6: the welfare record of the tick
            work = road[i] is None
            v = V[region[i]] if work else 0
            ring[i].append((work, v)); sumV[i] += v; nw[i] += 1 if work else 0
            if len(ring[i]) > p.T:
                w0, v0 = ring[i].popleft(); sumV[i] -= v0; nw[i] -= 1 if w0 else 0
        hh = [0] * nreg; inflight = [0] * nreg            # phase 7: the migration pass, after the entry step
        for i in range(p.N):
            hh[region[i]] += 1
            if road[i] is not None: inflight[road[i][0]] += 1
        for r in range(nreg): series[r].append(hh[r]); flight[r].append(inflight[r])
        cands = []
        for i in range(p.N):
            if road[i] is not None or (t + i) % p.R: continue
            need = max(p.cool, p.T) if p.settle else 0
            if t - since[i] < need: continue
            ratio = (sumV[i] * p.work // 10000) // max(1, len(ring[i]))
            if ratio >= p.below: continue
            Va = sumV[i] // nw[i] if nw[i] else 0
            r = region[i]; ranking = []
            for d in range(nreg):
                if d == r or (r, d) not in p.D: continue
                dist = p.D[(r, d)]
                lag = 1 + cdiv(dist, p.info_speed)
                if t - lag < 0: blocked["rumour"] += 1; continue
                e = rng.randint(-p.E, p.E) if p.E else 0
                vhat = Vhist[d][t - lag] * (10000 + e) // 10000
                k = max(1, cdiv(dist, p.move_speed))
                if k >= p.H: blocked["far"] += 1; continue
                veff = vhat * (p.H - k) // p.H
                if vhat < p.entry_ratio: blocked["viable"] += 1; continue
                if p.abs_spread: g_a = p.gain + (weyl(i) * p.spread) // M32
                else: g_a = p.gain * (10000 + ((2 * weyl(i) - M32) * p.spread) // M32) // 10000
                if veff * 10000 < Va * (10000 + g_a): blocked["gain"] += 1; continue
                ranking.append((-veff, d, k))
            if ranking:
                ranking.sort(); cands.append((ratio, i, ranking))
        cands.sort()
        outc = [0] * nreg
        for ratio, i, ranking in cands:
            r = region[i]
            if p.use_caps and outc[r] >= cdiv(hh[r] * p.out_share, 10000): blocked["out"] += 1; continue
            for _, d, k in ranking:
                if p.use_caps and inflight[d] >= cdiv(hh[d] * p.in_share, 10000): blocked["in"] += 1; continue
                road[i] = (d, t + 1 + k); inflight[d] += 1; outc[r] += 1; started += 1
                break
    return series, flight, started, blocked

def two_region(label, **kw):
    """Two equal regions started 800/200 (equilibrium 500/500): the overshoot is the lowest nA after first reaching 500;
    recrossings counts the reversals through 500 +- 10 after that."""
    p = P(**kw)
    s, fl, started, bl = run(p)
    nA = s[0]; half = p.N // 2
    first = next((t for t, x in enumerate(nA) if x <= half), None)
    if first is None:
        lo = min(nA); cross = 0
    else:
        tail = nA[first:]; lo = min(tail); cross = 0; state = 0
        for x in tail:
            sg = 1 if x > half + 10 else -1 if x < half - 10 else 0
            if sg and state and sg != state: cross += 1
            if sg: state = sg
    needed = p.N * 0.3
    print(f"{label:44s} t(500) {str(first):>5s}  lowest {lo:4d}  recrossings {cross:3d}  moves/needed {started / needed:6.2f}  end {nA[-1]:4d}")
    return first, lo, cross, started / needed

if __name__ == "__main__":
    args = sys.argv[1:] or ["all"]
    class _W(str):
        pass
    which = args
    if "all" in which or "ablation" in which:
        print("# 1. each damper alone against an undamped base (g 300, R 10, no width, no caps, no settling, no horizon), eta 1")
        base = dict(gain=300, R=10, spread=0, use_caps=False, settle=False, H=10**6, split=[800, 200])
        two_region("base: nothing", **base)
        two_region("review period R 30", **{**base, "R": 30})
        two_region("review period R 60", **{**base, "R": 60})
        two_region("width 5000 added to the margin", **{**base, "spread": 5000})
        two_region("relative spread +-9000 of the margin", **{**base, "spread": 9000, "abs_spread": False})
        two_region("margin 2000", **{**base, "gain": 2000})
        two_region("caps out 50 / on-road 100", **{**base, "use_caps": True})
        two_region("settling (no re-move within T)", **{**base, "settle": True})
        two_region("horizon H 60 (a trip of 3 ticks loses 5%)", **{**base, "H": 60})
        two_region("the spec's defaults (R 30, g 1000, w 5000, caps)", **{**base, "gain": 1000, "R": 30, "spread": 5000, "use_caps": True, "settle": True, "H": 180})
        print()
        print("# 1b. the same, with settling on (the rule of 5.2 is a rule, not a choice): each other damper alone")
        base = dict(gain=300, R=10, spread=0, use_caps=False, settle=True, H=10**6, split=[800, 200])
        two_region("settling only", **base)
        two_region("+ review period R 30", **{**base, "R": 30})
        two_region("+ review period R 60", **{**base, "R": 60})
        two_region("+ width 2000", **{**base, "spread": 2000})
        two_region("+ width 5000", **{**base, "spread": 5000})
        two_region("+ margin 1000", **{**base, "gain": 1000})
        two_region("+ margin 2000", **{**base, "gain": 2000})
        two_region("+ caps out 50 / on-road 100", **{**base, "use_caps": True})
        two_region("+ horizon H 60", **{**base, "H": 60})
        print()
    if "all" in which or "caps" in which:
        print("# 2. the caps with width 0 (the width would hide them): outflow share per tick, share of the destination on the road")
        base = dict(gain=1000, R=30, spread=0, split=[800, 200])
        for out in (10, 20, 50, 100, 10000):
            for inn in (50, 100, 300, 10000):
                two_region(f"out {out:5d} in {inn:5d}", **{**base, "out_share": out, "in_share": inn})
        print()
    if "all" in which or "pipeline" in which:
        print("# 3. the invisible pipeline: the road time k, with width 0 and margin 1000 (so the dispersion cannot hide it)")
        for k in (1, 3, 8, 20):
            dist = 40 * k
            for caps in (False, True):
                two_region(f"road {k:2d} ticks, caps {'on ' if caps else 'off'} (out 50 / on-road 100)",
                           gain=1000, R=30, spread=0, use_caps=caps, split=[800, 200], D={(0, 1): dist, (1, 0): dist}, info_speed=10**9, H=10**4, out_share=50, in_share=100)
        print()

def herd(label, **kw):
    """Three regions: 0 is poor and loaded, 1 and 2 are equally good. Reports the largest number of households on the
    road to one destination at once, the split of arrivals, and the peak-to-peak of (n1 - n2) after the first 200 ticks."""
    base = dict(N=1500, A=[9000, 12000, 12000], split=[1000, 250, 250], ticks=1500, gain=1000, R=30, spread=0, E=0,
                D={(0, 1): 80, (0, 2): 80, (1, 0): 80, (2, 0): 80, (1, 2): 80, (2, 1): 80})
    p = P(**{**base, **kw})
    s, fl, started, bl = run(p)
    diff = [a - b for a, b in zip(s[1][200:], s[2][200:])]
    print(f"{label:46s} max on road to one {max(max(fl[1]), max(fl[2])):4d}  end {s[0][-1]:4d}/{s[1][-1]:4d}/{s[2][-1]:4d}  (n1-n2) p-p {max(diff) - min(diff):4d}  started {started:5d}")

if __name__ == "__main__":
    if "all" in which or "herding" in which:
        print("# 4. herding: region 0 is poor, 1 and 2 are equally good; every household of 0 sees the same ranking")
        herd("identical, exact news, no caps", use_caps=False)
        herd("rumour error 500 (independent), no caps", use_caps=False, E=500)
        herd("rumour error 500 and width 5000, no caps", use_caps=False, E=500, spread=5000)
        herd("caps out 50 / on-road 100, exact news", E=0)
        herd("caps on-road 100 only (out uncapped), exact news", out_share=10000)
        herd("caps, error 500 and width 5000 (the defaults)", E=500, spread=5000)
        print()

if __name__ == "__main__":
    if "all" in which or "defaults" in which:
        print("# 5. the spec's default parameters (R 30, g 1000, width 5000, out 50, on-road 100, cooldown 60, H 180, error 500)")
        print("# over seeds, elasticities and road times; two equal regions started 800/200, equilibrium 500")
        for eta in (0.5, 1.0):
            for k in (1, 3, 8):
                res = []
                for seed in range(1, 6):
                    dist = 40 * k
                    p = P(eta=eta, seed=seed, split=[800, 200], D={(0, 1): dist, (1, 0): dist})
                    s, fl, started, bl = run(p)
                    nA = s[0]; first = next((t for t, x in enumerate(nA) if x <= 500), None)
                    lo = min(nA[first:]) if first is not None else min(nA)
                    res.append((first, lo, nA[-1], started))
                print(f"eta {eta} road {k:2d}: " + "  ".join(f"[{f if f is not None else '-':>4},{lo:4d},{e:4d},{st:4d}]" for f, lo, e, st in res))
        print("# columns per seed: [tick of first reaching 500, lowest after, end, started]")
        print()
        print("# an unequal pair: A = 10000 and 13000, eta 0.5, equilibrium split 0.59/0.41 of the households")
        for seed in (1, 2, 3):
            p = P(eta=0.5, seed=seed, A=[10000, 13000], split=[500, 500], ticks=3000)
            s, fl, started, bl = run(p)
            print(f"seed {seed}: nA at 0, 300, 600, 1200, 2400, 2999: " + ", ".join(str(s[0][t]) for t in (0, 300, 600, 1200, 2400, 2999)) + f"   started {started}  eq {1000 * (10000/13000) ** (1/0.5) / (1 + (10000/13000) ** (1/0.5)):.0f}")
        print()

if __name__ == "__main__":
    if "all" in which or "swing" in which:
        print("# 6. when does the flow reverse? outflow share x window against the margin (width 0, no cap on the road, eta 1, R 30, T 30)")
        print("# product = MoveOutShare * T (bp of the households per window); ratio = product / MoveGain")
        print(f"{'out':>6s} {'gain':>6s} {'product':>8s} {'ratio':>6s} {'lowest':>7s} {'recross':>8s}")
        for g in (300, 1000, 2000, 4000):
            for out in (20, 50, 100, 200, 300, 10000):
                p = P(gain=g, spread=0, split=[800, 200], out_share=out, in_share=10000)
                s, fl, started, bl = run(p)
                nA = s[0]; first = next((t for t, x in enumerate(nA) if x <= 500), None)
                if first is None: lo = min(nA); cross = 0
                else:
                    tail = nA[first:]; lo = min(tail); cross = 0; state = 0
                    for x in tail:
                        sg = 1 if x > 510 else -1 if x < 490 else 0
                        if sg and state and sg != state: cross += 1
                        if sg: state = sg
                prod = out * p.T
                print(f"{out:6d} {g:6d} {prod:8d} {prod / g:6.1f} {lo:7d} {cross:8d}")
        print()

def continuous(kt, z0=0.2, dt=0.002, horizon=800.0):
    """dz/dt = -kappa z(t - tau) with tau = 1: the delayed integrator (a midpoint step, so the integrator itself adds no
    instability). Returns (the lowest z / z0 reached: negative means an overshoot; the amplitude of the last quarter of the
    run over that of the quarter before it: below 1 the oscillation decays, above 1 it grows)."""
    tau = 1.0; kappa = kt / tau
    n = int(horizon / dt); lag = int(round(tau / dt))
    z = [z0] * (lag + 1)
    for _ in range(n):
        zl = z[-1 - lag]; zl1 = z[-lag]
        z.append(z[-1] - dt * kappa * 0.5 * (zl + zl1))
    q = len(z) // 4
    last = max(abs(x) for x in z[-q:]); before = max(abs(x) for x in z[-2 * q:-q])
    return min(z) / z0, (last / before if before > 1e-300 else 0.0)

if __name__ == "__main__":
    if "all" in which or "linear" in which:
        print("# 0. the linearised loop is a delayed integrator dz/dt = -kappa z(t - tau): stable iff kappa*tau < pi/2")
        print("kappa*tau   lowest z/z0   amplitude, last quarter over the one before   reading")
        for kt in (0.2, 0.30, 0.36, 0.40, 0.50, 1.0, 1.4, 1.5, 1.56, 1.58, 1.60, 2.0, 3.0):
            lo, a = continuous(kt)
            print(f"{kt:8.2f}   {lo:11.4g}   {a:16.4g}   " + ("monotone" if lo > -1e-9 else "decays with overshoot" if a < 0.999 else "grows"))
        print()
