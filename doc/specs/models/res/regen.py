"""RES model 1: the integer regeneration rules of spec-resources.md 4, and what the carried remainder buys.

A model of the RULES of the spec, not of the library. All arithmetic is integer (Python ints are unbounded, which stands
for the checked 128-bit helper of spec-currency.md 7; the asserts check the bounds of spec-resources.md 4.6).
Floating point is NOT used. Every number is a placeholder (N-015, N-027)."""
import random
from fractions import Fraction

BP = 10_000

def logistic_w(s, cap, refuge):
    """The logistic factor, rounded UP to a whole unit (4.3): w = ceil(S' * (C - S') / C) with S' = max(S, Refuge).
    0 when S' is 0 or at the capacity."""
    sp = max(s, refuge)
    if sp <= 0 or sp >= cap:
        return 0
    return -((-sp * (cap - sp)) // cap)          # ceil division

def numerator(shape, s, cap, rate, m, sig, refuge=0):
    """Numerator of the regeneration of one tick over D = 10^12 * TPY (4.2). rate is units a year for 'constant', basis
    points a year otherwise; m is the region/shock scale in bp; sig the season factor in bp."""
    if shape == "constant":
        return rate * m * sig * BP
    if shape == "refill":
        return rate * (cap - s) * m * sig if s < cap else 0
    if shape == "logistic":
        return rate * logistic_w(s, cap, refuge) * m * sig
    raise ValueError(shape)

def step(shape, s, c, cap, rate, m, sig, tpy, refuge=0):
    """One tick of regeneration with the carried remainder. Returns (new stock, new carry, units added)."""
    d = 10**12 * tpy
    n = numerator(shape, s, cap, rate, m, sig, refuge)
    tot = c + n
    add, c2 = divmod(tot, d)
    if s + add >= cap:                          # a full stock banks nothing (4.4)
        return cap, 0, cap - s
    return s + add, c2, add

def identity_check(trials=400, ticks=1_500, seed=11):
    """Identity: stock_end - stock_start + draws = floor((c0 + sum n)/D) - (carry_end... ) i.e. units added = floor((c0+sum n)/D).
    Draws change the path of S and so the numerators, but the units added are always the floor of the running total."""
    rng = random.Random(seed); bad = 0
    for _ in range(trials):
        shape = rng.choice(["constant", "refill", "logistic"]); tpy = rng.choice([1, 12, 365, 640, 100_000])
        cap = rng.randint(10**4, 10**12)
        rate = rng.randint(1, 10**6) if shape == "constant" else rng.randint(1, 100_000)
        m = rng.choice([2_500, 10_000, 31_000]); sig = rng.choice([5_000, 10_000, 40_000])
        d = 10**12 * tpy
        s = rng.randint(1, cap // 2); c = rng.randint(0, d - 1); run_c = c; units = 0; capped = False
        for _ in range(ticks):
            n = numerator(shape, s, cap, rate, m, sig, 0)
            s2, c2, a = step(shape, s, c, cap, rate, m, sig, tpy, 0)
            if s2 >= cap: capped = True; break
            run_c += n; units += a; c = c2
            s = s2 - min(s2 - 1, rng.randint(0, a + 2))
        if not capped and units != run_c // d:
            bad += 1
        if not capped and c != run_c % d:
            bad += 1
    return bad

def extinction_without_carry():
    """A logistic stock of a few units with a small rate: truncation without a carry regrows nothing (an absorbing
    state at small stocks); with the carry and the round-up factor it regrows (4.3)."""
    cap, rate, tpy, m, sig = 1_000_000, 500, 365, 10_000, 10_000       # 5% a year
    rows = []
    for s0 in (1, 2, 5, 20, 100):
        # no carry, truncation: add = floor(rate * w * m * sig / D) each tick
        s = s0; d = 10**12 * tpy
        for _ in range(20 * 365):
            s = min(cap, s + (numerator("logistic", s, cap, rate, m, sig) // d))
        no_carry = s
        s = s0; c = 0
        for _ in range(20 * 365):
            s, c, _a = step("logistic", s, c, cap, rate, m, sig, tpy)
        rows.append((s0, no_carry, s))
    return rows

if __name__ == "__main__":
    print("== 1. cumulative units added equal floor((carry0 + sum of numerators) / D), for 400 random runs")
    print("   violations:", identity_check())
    print("\n== 2. logistic, 5% a year, capacity 1 000 000, 20 years, with draws of nothing")
    print("   start stock | final stock without a carry | final stock with the carry (round-up factor)")
    for s0, a, b in extinction_without_carry():
        print(f"   {s0:>11} | {a:>27,} | {b:>12,}")
    print("\n== 3. bounds of 4.6: the largest numerator and denominator")
    big = numerator("refill", 0, 10**15, 100_000, 100_000, 100_000)
    print(f"   largest numerator (RateBP 1e5, capacity 1e15, scales 1e5): {big:.3e} < 2^127 = {2**127:.3e}: {big < 2**127}")
    print(f"   largest denominator 1e12 * 100 000 = {10**12*100_000:.3e} < 2^63 = {2**63:.3e}: {10**12*100_000 < 2**63}")
