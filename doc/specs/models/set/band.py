"""Models behind spec-settlements.md sections 4, 13 and 14: the review rule in integers, the
band and the dwell against census noise, and the staircase and absorbing-class toy.

A stylised model of the RULES of the spec, not of the library, and not a claim that any
default is stable (notes N-015, N-027). The economy is replaced by one assumed function of
the household count. Standard library only. Run with Docker (see README.md).

    python band.py rule | noise | loop | all
"""
import sys

# ---- the rule of 4.2 and 4.3, in integers ---------------------------------------------

def effective(enter, thr_bp, band_bp):
    """Effective enter thresholds E (ceil of Enter * ThresholdBP / 10000) and leave
    thresholds L (floor of E * (10000 - BandBP) / 10000); class 0 has E = L = 0."""
    E = [0] + [-(-e * thr_bp // 10000) for e in enter[1:]]
    L = [0] + [e * (10000 - band_bp) // 10000 for e in E[1:]]
    return E, L

def founded(E, floor, h):
    """The class at construction: the highest k with h >= E[k], never below the floor."""
    k = 0
    while k + 1 < len(E) and h >= E[k + 1]:
        k += 1
    return max(k, floor)

def target(E, L, c, floor, h):
    """The class a review gives, ignoring the dwell (4.3)."""
    up = c
    while up + 1 < len(E) and h >= E[up + 1]:
        up += 1
    if up > c:
        return up
    dn = c
    while dn > floor and h < L[dn]:
        dn -= 1
    return dn

class Region:
    def __init__(self, E, L, floor, dwell, h0):
        self.E, self.L, self.floor, self.dwell = E, L, floor, dwell
        self.c = founded(E, floor, h0)
        self.since = 0
        self.changes = 0
        self.held = 0

    def review(self, t, h):
        tg = target(self.E, self.L, self.c, self.floor, h)
        if tg == self.c:
            return 0
        if t - self.since < self.dwell:
            self.held += 1
            return 0
        self.c, self.since = tg, t
        self.changes += 1
        return 1

# ---- a deterministic generator (SplitMix64 words; no library draw is involved) --------

M64 = (1 << 64) - 1
class Gen:
    def __init__(self, seed):
        self.s = seed & M64
    def word(self):
        self.s = (self.s + 0x9E3779B97F4A7C15) & M64
        z = self.s
        z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
        z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
        return z ^ (z >> 31)
    def below(self, n):
        return (self.word() >> 32) * n >> 32

def rule():
    print("== the rule: invariants over every household count 0..3000, classes 0, 500, 2500")
    enter = [0, 500, 2500]
    for band in (0, 1000):
        E, L = effective(enter, 10000, band)
        bad = 0
        for c in range(3):
            for h in range(0, 3001):
                t = target(E, L, c, 0, h)
                # inside its band: L[t] <= h < E[t+1] (E beyond the top is infinity)
                lo_ok = h >= L[t]
                hi_ok = t + 1 >= len(E) or h < E[t + 1]
                if not (lo_ok and hi_ok):
                    bad += 1
        print(f"band {band:5d} bp  E={E} L={L}  states outside the band after a review: {bad}")
        # the same invariant with a floor f (4.3.2): H < E[t+1] always, and L[t] <= H unless t == f
        badf = nf = 0
        for f in range(3):
            for c in range(f, 3):
                for h in range(0, 3001):
                    t = target(E, L, c, f, h)
                    nf += 1
                    hi_ok = t + 1 >= len(E) or h < E[t + 1]
                    lo_ok = h >= L[t] or t == f
                    if not (t >= f and hi_ok and lo_ok):
                        badf += 1
        print(f"band {band:5d} bp  floors 0, 1, 2 from every class >= floor ({nf} reviews): "
              f"states outside the band, the floor excepted: {badf}")
        # the floor is silent: the count below L[f] at the floor (class_at_floor), counted once
        at_floor = sum(1 for h in range(0, 3001) if target(E, L, 1, 1, h) == 1 and h < L[1])
        print(f"band {band:5d} bp  floor 1: counts at the floor below its leave threshold: {at_floor}")
    E, L = effective(enter, 10000, 1000)
    print("a region at class 1 (E=500, L=450): households 449 -> class", target(E, L, 1, 0, 449),
          "; 450 ->", target(E, L, 1, 0, 450), "; 2500 ->", target(E, L, 1, 0, 2500),
          "; class 2 at 2249 ->", target(E, L, 2, 0, 2249), "; at 2250 ->", target(E, L, 2, 0, 2250))
    print("a jump over a class: class 0 at 3000 ->", target(E, L, 0, 0, 3000),
          "; class 2 at 0 with floor 1 ->", target(E, L, 2, 1, 0))
    for e1, thr in ((500, 5000), (5, 5000), (1, 10000), (6, 5000)):
        E, L = effective([0, e1], thr, 1000)
        print(f"Enter {e1} at ThresholdBP {thr}: E={E[1]} L={L[1]} (band {E[1]-L[1]} households)")
    # 4.2.2: the band is one household whenever E * BandBP <= 10000; L >= floor(E / 2) at MaxBandBP
    for e1, bp in ((3, 5000), (5, 1000), (10, 1000), (11, 1000), (7, 5000)):
        E, L = effective([0, e1], 10000, bp)
        print(f"Enter {e1} at BandBP {bp}: E={E[1]} L={L[1]} (band {E[1]-L[1]} households; "
              f"E*BandBP <= 10000: {E[1] * bp <= 10000}; floor(E/2) = {E[1] // 2})")

def noise():
    print("== class changes in 3650 ticks, mean (max) over 40 seeds; households an integer")
    print("   mean-reverting walk, reversion 1/20 a tick, base ON the threshold E=500")
    print("   step s: uniform in [-s, s]; the walk's standard deviation is printed")
    T, SEEDS = 3650, 40
    for s in (2, 5, 10, 20):
        var = (s * (s + 1) / 3) / (1 - (1 - 1 / 20) ** 2)
        print(f"-- s={s}, sd of households about {var ** 0.5:.1f} ({100 * var ** 0.5 / 500:.1f}% of 500)")
        print("   band bp \\ dwell:      0            30")
        for band in (0, 500, 1000, 2000):
            row = []
            for dwell in (0, 30):
                tot = mx = 0
                for seed in range(1, SEEDS + 1):
                    g = Gen(seed * 1000 + s)
                    E, L = effective([0, 500], 10000, band)
                    h = 500
                    r = Region(E, L, 0, dwell, h)
                    for t in range(T):
                        h += (500 - h) // 20 if h >= 500 else -((h - 500) // 20)
                        h += g.below(2 * s + 1) - s
                        r.review(t, h)
                    tot += r.changes
                    mx = max(mx, r.changes)
                row.append(f"{tot / SEEDS:7.1f} ({mx:3d})")
            print(f"   {band:5d}          " + "     ".join(row))

def loop():
    print("== staircase and absorbing class: an assumed net growth rate per class, basis points a tick")
    print("   classes: 0 (E 0), 1 (E 100), 2 (E 400), 3 (E 1600); rates by class (-4, +12, +20, +4) bp a tick")
    rates = (-4, 12, 20, 4)
    enter = [0, 100, 400, 1600]
    def run(band, dwell, floor, shock_t, shock_to, T=3650, h0=120):
        E, L = effective(enter, 10000, band)
        r = Region(E, L, floor, dwell, h0)
        h = h0 * 10000        # households in 1/10000, so rates in bp apply exactly
        trace = []
        for t in range(T):
            if t == shock_t:
                h = shock_to * 10000
            hh = h // 10000
            before = r.c
            r.review(t, hh)
            if r.c != before:
                trace.append((t, before, r.c, hh))
            h += h * rates[r.c] // 10000
        return r, h // 10000, trace
    for label, band, dwell, floor, st, sto in (
        ("growth only, band 1000 dwell 30        ", 1000, 30, 0, -1, 0),
        ("growth only, band 0 dwell 0            ", 0, 0, 0, -1, 0),
        ("shock to 60 at t=1500, band 1000 dwell 30", 1000, 30, 0, 1500, 60),
        ("shock to 60 at t=1500, floor class 1     ", 1000, 30, 1, 1500, 60),
        ("shock to 380 at t=1500, band 1000 dwell 30", 1000, 30, 0, 1500, 380),
        ("shock to 380 at t=1500, band 0 dwell 0    ", 0, 0, 0, 1500, 380),
    ):
        r, h, tr = run(band, dwell, floor, st, sto)
        print(f"{label}: final class {r.c}, households {h}, changes {r.changes}, held ticks {r.held}")
        print("     changes (tick, from, to, households):", tr)

if __name__ == "__main__":
    what = sys.argv[1] if len(sys.argv) > 1 else "all"
    if what in ("rule", "all"):
        rule()
    if what in ("noise", "all"):
        noise()
    if what in ("loop", "all"):
        loop()
