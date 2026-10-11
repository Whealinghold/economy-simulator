"""Seed rules and band setting of spec-stylised-facts.md section 8 (a model of the
PROCEDURE, not of the library). Exact arithmetic with fractions for the binomial
tables; a small integer congruential generator for the synthetic band-setting
trials, so a run is the same on every platform. Standard library only.

Part A: the false-fail rate and the power of the three pass rules (all seeds,
nine in ten, median) for n gate seeds and n nightly seeds.
Part B: the chance that a new seed lies outside the envelope of m setting seeds
is 2 / (m + 1) for exchangeable values; exact by enumeration for small m and by
trials for m = 10 and 40.
Part C: the effect of the headroom rule of 8.4.3 on held-out seeds, on two
synthetic shapes (a bell-like and a one-sided skew).
"""
from fractions import Fraction
from itertools import permutations
from math import comb


def bp(x):
    """a Fraction as basis points, rounded half to even (the rule of IND 5.2.2)."""
    n = x * 10000
    q, r = divmod(n.numerator, n.denominator)
    twice = 2 * r
    if twice > n.denominator or (twice == n.denominator and q % 2 == 1):
        q += 1
    return q


def binom_le(n, k, q):
    return sum(comb(n, j) * q ** j * (1 - q) ** (n - j) for j in range(0, k + 1))


def ceil_div(a, b):
    return -(-a // b)


def part_a():
    print("A. pass rules: probability that a build passes, in basis points")
    print("   q = chance one seed is outside the band; rule SR1 all, SR2 at least ceil(9n/10) inside")
    print("   n   q      SR1(all)  SR2(9 in 10)")
    for n in (10, 50):
        need = ceil_div(9 * n, 10)
        allowed = n - need
        for qn, qd in ((1, 100), (2, 100), (5, 100), (20, 100), (30, 100), (50, 100)):
            q = Fraction(qn, qd)
            sr1 = (1 - q) ** n
            sr2 = binom_le(n, allowed, q)
            print("  %3d  %3d%%   %6d   %6d" % (n, qn, bp(sr1), bp(sr2)))
    print("   false fail = 10000 - pass at the small q; power = 10000 - pass at the large q")
    print()


def part_b():
    print("B. a new seed outside the envelope of m exchangeable seeds: 2/(m+1)")
    for m in (3, 5, 7):
        outside = 0
        total = 0
        for p in permutations(range(m + 1)):
            total += 1
            new = p[-1]
            rest = p[:-1]
            if new < min(rest) or new > max(rest):
                outside += 1
        assert Fraction(outside, total) == Fraction(2, m + 1)
        print("  m=%d exact: %d/%d = %d bp (2/(m+1) = %d bp)" % (m, outside, total, bp(Fraction(outside, total)), bp(Fraction(2, m + 1))))
    for m in (10, 40, 50):
        print("  m=%d formula: %d bp" % (m, bp(Fraction(2, m + 1))))
    print()


class Lcg:
    """63-bit LCG; the high bits are used (the low bits of an LCG are weak)."""

    def __init__(self, seed):
        self.s = seed & ((1 << 63) - 1)

    def next(self):
        self.s = (self.s * 6364136223846793005 + 1442695040888963407) & ((1 << 63) - 1)
        return self.s >> 20

    def below(self, n):
        return self.next() % n


def draw(g, shape):
    if shape == "bell":      # sum of four uniforms on 0..999: bell-like, integer
        return sum(g.below(1000) for _ in range(4))
    if shape == "skew":      # a square of a uniform: one-sided heavy tail, integer
        u = g.below(1000)
        return u * u // 250
    raise ValueError(shape)


def round_outward(x, up):
    """two significant digits, outward: up for an upper bound, down for a lower; a
    value of at most two digits is kept; a negative value rounds away from zero
    for a lower bound and toward zero for an upper bound (8.4.3)."""
    ax = abs(x)
    if ax < 100:
        return x
    step = 1
    while ax >= 100 * step:
        step *= 10
    q = x // step
    if up and q * step < x:
        q += 1
    return q * step


def part_c():
    print("C. headroom and rounding of the band-setting rule on 10 NEW seeds, i.e. what a hold-out gives (trials of 2000)")
    print("   band = [min - h*range, max + h*range] of m setting seeds, rounded outward to two significant digits")
    print("   (against REF's seeds the gate seeds are among the setting seeds, so the gate shows a change of the code)")
    print("   rows: P(all 10 gate seeds inside), P(at least 9 of 10 inside), in basis points")
    trials = 2000
    for shape in ("bell", "skew"):
        for m in (10, 40, 50):
            print("  shape %s, m=%d setting seeds" % (shape, m))
            for h in (0, 1000, 2500, 5000):
                g = Lcg(12345 + m * 7 + h)
                all_in = 0
                nine_in = 0
                for _ in range(trials):
                    vals = [draw(g, shape) for _ in range(m)]
                    lo, hi = min(vals), max(vals)
                    span = hi - lo
                    lo2 = round_outward(lo - span * h // 10000, False)
                    hi2 = round_outward(hi + span * h // 10000, True)
                    gate = [draw(g, shape) for _ in range(10)]
                    inside = sum(1 for v in gate if lo2 <= v <= hi2)
                    if inside == 10:
                        all_in += 1
                    if inside >= 9:
                        nine_in += 1
                print("    h=%5d bp   all=%5d   nine=%5d" % (h, all_in * 10000 // trials, nine_in * 10000 // trials))
    print()


if __name__ == "__main__":
    part_a()
    part_b()
    part_c()
    assert round_outward(201, True) == 210 and round_outward(184, True) == 190
    assert round_outward(-201, False) == -210 and round_outward(-201, True) == -200
    assert round_outward(99, True) == 99 and round_outward(2001, True) == 2100
    print("rounding asserted: 201 -> 210, 184 -> 190, -201 -> -210 (lower) and -200 (upper), 2001 -> 2100")
