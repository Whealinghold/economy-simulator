"""Spec section 9: the integer reductions of spec-experiments.md.

Defines, with Python integers standing for the library's 128-bit sums, the
reductions over the replicas of one point and the comparison of two points,
and checks them against exact rational arithmetic (fractions) on random data:

  rdiv            n / d rounded half to even (IND 5.2.2)
  isqrt_nearest   the integer nearest to sqrt(n) (IND 5.3.4), no tie possible
  nearest_rank    the weighted nearest-rank quantile with weight 1 (IND 5.5.2)
  std_sample      isqrt_nearest(rdiv(n * SS - S^2, n * (n - 1)))
  sign_tail_ppm   the exact two-sided sign-test tail in parts per million, n <= 62
  effect_range    the range of the bin means of a factor, and its share of the total
  elasticity_bp   rdiv(10 000 * dy * x0, y0 * dx)

A model of the rules, not of the library (note N-027).
"""
import random
from fractions import Fraction
from math import comb, isqrt


def rdiv(n, d):
    """n / d rounded half to even; d > 0."""
    assert d > 0
    q, r = divmod(n, d)
    twice = 2 * r
    if twice > d or (twice == d and q % 2 == 1):
        q += 1
    return q


def isqrt_nearest(n):
    r0 = isqrt(n)
    return r0 + 1 if n - r0 * r0 > r0 else r0


def nearest_rank(sorted_vals, q_bp):
    n = len(sorted_vals)
    k = max(1, -(-q_bp * n // 10000))
    return sorted_vals[k - 1]


def mean(vals):
    return rdiv(sum(vals), len(vals))


def std_sample(vals):
    n = len(vals)
    s = sum(vals)
    ss = sum(v * v for v in vals)
    var_num = n * ss - s * s
    return isqrt_nearest(rdiv(var_num, n * (n - 1)))


def sign_tail_ppm(wins, losses):
    n = wins + losses
    assert n <= 62
    if n == 0:
        return None
    m = max(wins, losses)
    tail = sum(comb(n, j) for j in range(m, n + 1))
    return min(1_000_000, rdiv(1_000_000 * 2 * tail, 1 << n))


def effect_range(bin_means):
    return max(bin_means) - min(bin_means)


def shares(ranges):
    tot = sum(ranges)
    return [rdiv(10000 * r, tot) if tot else None for r in ranges]


def elasticity_bp(y0, y1, x0, x1, n0=1, n1=1):
    """From the exact sums y0, y1 of n0 and n1 runs (spec 8.5.3)."""
    if y0 <= 0 or x0 == 0 or x1 == x0:
        return None
    return rdiv(10000 * (y1 * n0 - y0 * n1) * x0, y0 * n1 * (x1 - x0))


def check():
    rng = random.Random(7)
    # rdiv against Fraction rounding (half to even)
    for _ in range(20000):
        n = rng.randint(-10**9, 10**9)
        d = rng.randint(1, 10**4)
        f = Fraction(n, d)
        want = round(f)           # Python rounds half to even on a Fraction
        assert rdiv(n, d) == want, (n, d)
    # isqrt_nearest is the nearest integer to the real square root
    for _ in range(20000):
        n = rng.randint(0, 10**12)
        r = isqrt_nearest(n)
        assert abs(Fraction(r) ** 2 - n) <= abs(Fraction(r + 1) ** 2 - n)
        assert r == 0 or abs(Fraction(r) ** 2 - n) <= abs(Fraction(r - 1) ** 2 - n)
    # quantile definition: the first sorted value whose rank reaches ceil(q n / 10 000)
    for n in (1, 2, 3, 7, 10, 33):
        vals = sorted(rng.randint(0, 100) for _ in range(n))
        for q in (0, 1000, 2500, 5000, 7500, 9000, 10000):
            k = max(1, -(-q * n // 10000))
            assert nearest_rank(vals, q) == vals[k - 1]
        assert nearest_rank(vals, 0) == vals[0] and nearest_rank(vals, 10000) == vals[-1]
    # sign test against the exact binomial with fractions
    for n in range(1, 63):
        for w in range(0, n + 1):
            l = n - w
            m = max(w, l)
            exact = Fraction(2 * sum(comb(n, j) for j in range(m, n + 1)), 2 ** n)
            want = min(1_000_000, round(exact * 1_000_000))
            assert sign_tail_ppm(w, l) == want, (n, w)
    # sum of squares fits in 128 bits for the largest bound: values up to 2^40, 10^6 replicas
    assert (10**6) * (2**41) ** 2 < 2**126   # SS of 10^6 readings of magnitude 2^40 (a difference is 2^41)
    assert (10**6) * (10**6) * (2**40) ** 2 < 2**126   # Used * SS - S^2 at the largest Used
    assert 10000 * 2**41 * 2**63 < 2**127   # an elasticity numerator for single-run groups with x an int64; larger groups use a checked product (spec 8.5.3)
    print("all checks passed")


def examples():
    print()
    print("== replicas of one point: ten readings of a price range, in basis points of the base")
    vals = [10400, 10900, 10100, 11800, 10300, 10700, 10500, 12100, 10200, 10600]
    s = sorted(vals)
    print("  readings sorted :", s)
    print("  n=%d  min=%d  max=%d  mean=%d  std=%d" % (len(s), s[0], s[-1], mean(s), std_sample(s)))
    for q in (1000, 2500, 5000, 7500, 9000):
        print("  Q(%5d bp) = %d   (rank k = %d)" % (q, nearest_rank(s, q), max(1, -(-q * 10 // 10000))))
    print("  pass share of 'price range at most 11 000 bp': %d of 10 -> %d bp" % (
        sum(1 for v in vals if v <= 11000), rdiv(10000 * sum(1 for v in vals if v <= 11000), 10)))

    print()
    print("== a failed run in the same point: n planned 10, one failed, nine used")
    used = [v for v in vals if v != 10600]
    print("  the run that read 10 600 failed: used %d, failed 1" % len(used))
    print("  mean of nine=%d   pass share against planned 10: %d bp (the failed run does not pass)" % (
        mean(used), rdiv(10000 * sum(1 for v in used if v <= 11000), 10)))
    used2 = [v for v in vals if v != 12100]
    print("  had the run that read 12 100 failed instead: pass share %d bp, mean of nine %d" % (
        rdiv(10000 * sum(1 for v in used2 if v <= 11000), 10), mean(used2)))

    print()
    print("== the comparison of two points, eight paired replicas")
    a = [100, 104, 98, 110, 101, 99, 107, 103]
    b = [103, 109, 97, 118, 107, 99, 113, 104]
    d = [y - x for x, y in zip(a, b)]
    wins = sum(1 for x in d if x > 0)
    losses = sum(1 for x in d if x < 0)
    ties = sum(1 for x in d if x == 0)
    print("  differences     :", d)
    print("  mean difference : %d   median : %d" % (mean(d), nearest_rank(sorted(d), 5000)))
    print("  wins %d  losses %d  ties %d  two-sided sign tail %d ppm" % (wins, losses, ties, sign_tail_ppm(wins, losses)))
    print("  relative, from the exact sums of the same pairs: rdiv(10 000 * %d, %d) = %d bp" % (sum(d), sum(a), rdiv(10000 * sum(d), sum(a))))
    print("  (from the rounded means it would be rdiv(10 000 * %d, %d) = %d bp: a second rounding, not used)" % (mean(d), mean(a), rdiv(10000 * mean(d), mean(a))))
    print("  DiffOfMeans from exact sums: rdiv(%d * %d - %d * %d, %d * %d) = %d" % (sum(b), len(a), sum(a), len(b), len(a), len(b), rdiv(sum(b) * len(a) - sum(a) * len(b), len(a) * len(b))))

    print()
    print("== sensitivity on a 3 x 3 grid: bin means of two factors")
    y = {(0, 0): 100, (0, 1): 104, (0, 2): 108, (1, 0): 110, (1, 1): 114, (1, 2): 118, (2, 0): 140, (2, 1): 144, (2, 2): 148}
    fa = [mean([y[(i, j)] for j in range(3)]) for i in range(3)]
    fb = [mean([y[(i, j)] for i in range(3)]) for j in range(3)]
    ra, rb = effect_range(fa), effect_range(fb)
    print("  factor A bin means %s  range %d" % (fa, ra))
    print("  factor B bin means %s  range %d" % (fb, rb))
    print("  shares of the total range: A %d bp, B %d bp" % tuple(shares([ra, rb])))

    print()
    print("== one-at-a-time elasticity: a knob from 7 000 to 8 000 moves the mean of a measure from 12 000 to 11 100")
    print("  elasticity %d bp: a 1 percent rise of the knob moves the measure by %.3f percent" % (elasticity_bp(12000, 11100, 7000, 8000), elasticity_bp(12000, 11100, 7000, 8000) / 10000))


if __name__ == "__main__":
    check()
    examples()
