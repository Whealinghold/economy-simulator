"""Spec section 4: the adopted space-filling design of spec-experiments.md.

A Latin hypercube whose permutations are counter-based: for N points and axis j,
the cell of point i is P(i), a permutation of [0, N) made by a 4-round balanced
Feistel network over the smallest even-width domain that holds N, with a keyed
SplitMix64 finaliser as the round function and cycle walking back into [0, N).
No stream, no state: point i of axis j is a pure function of (salt, j, N, i).

The value of an integer factor with range [lo, hi] is the cell midpoint:

    v = lo + ((2 * P + 1) * (hi - lo + 1)) // (2 * N)

Checks: every P is a permutation (N = 1 .. 300 and some larger), every axis is
exactly stratified, no point depends on another point, and the 2-D fill and the
column correlations against a stateless hash draw. A model of the rule, not of
the library (note N-027).
"""
MASK = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def fin(z):
    z &= MASK
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & MASK
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & MASK
    return z ^ (z >> 31)


def half_bits(n):
    """k with 2^(2k) >= n, at least 1."""
    k = 1
    while (1 << (2 * k)) < n:
        k += 1
    return k


def feistel(x, k, key):
    mask = (1 << k) - 1
    left, right = x >> k, x & mask
    for rnd in range(4):
        f = fin(key + (rnd + 1) * GAMMA + right) & mask
        left, right = right, left ^ f
    return (left << k) | right


def perm(i, n, key):
    """The cell of point i, a permutation of [0, n)."""
    k = half_bits(n)
    x = feistel(i, k, key)
    while x >= n:
        x = feistel(x, k, key)
    return x


def axis_key(salt, j, n):
    return fin(fin(salt + (j + 1) * GAMMA) + n)


def value(cell, n, lo, hi):
    return lo + ((2 * cell + 1) * (hi - lo + 1)) // (2 * n)


# 0x243F6A8885A308D3 is the DesignSalt of the Latin-hypercube design (spec-experiments.md 7), a design constant; it is not the reseed rule (spec-checkpoint.md 6.8.2).
def design(n, d, salt=0x243F6A8885A308D3):
    return [[perm(i, n, axis_key(salt, j, n)) for j in range(d)] for i in range(n)]


def hash_design(n, d, salt=0x243F6A8885A308D3):
    return [[fin(salt + (i * d + j + 1) * GAMMA) % n for j in range(d)] for i in range(n)]


def empty_cells(pts, n, side):
    seen = {(p[0] * side // n, p[1] * side // n) for p in pts}
    return side * side - len(seen)


def pearson(a, b):
    n = len(a)
    ma, mb = sum(a) / n, sum(b) / n
    cov = sum((x - ma) * (y - mb) for x, y in zip(a, b))
    va = sum((x - ma) ** 2 for x in a)
    vb = sum((y - mb) ** 2 for y in b)
    return cov / (va * vb) ** 0.5


def max_abs_corr(pts, d):
    cols = [[p[j] for p in pts] for j in range(d)]
    return max(abs(pearson(cols[a], cols[b])) for a in range(d) for b in range(a + 1, d))


def main():
    print("== the permutation property: every cell exactly once, for every N")
    for n in list(range(1, 301)) + [511, 512, 513, 1000, 4096, 5000]:
        for j in (0, 1, 7):
            key = axis_key(0x243F6A8885A308D3, j, n)
            cells = sorted(perm(i, n, key) for i in range(n))
            assert cells == list(range(n)), (n, j)
    print("  ok: N = 1..300, 511, 512, 513, 1000, 4096, 5000; axes 0, 1, 7")

    print()
    print("== statelessness: point i does not depend on the other points")
    big = design(200, 5)
    for i in (0, 7, 199):
        solo = [perm(i, 200, axis_key(0x243F6A8885A308D3, j, 200)) for j in range(5)]
        assert solo == big[i]
    print("  ok: a point computed alone equals the point in the full design")

    print()
    print("== a value of an integer factor with range [10, 19] (ten levels) for N = 10 and N = 25")
    for n in (10, 25):
        vals = sorted(value(c, n, 10, 19) for c in range(n))
        counts = {v: vals.count(v) for v in sorted(set(vals))}
        print("  N=%2d  points per level: %s" % (n, counts))

    print()
    print("== 8 x 8 cells, N = 64 points in 2-D: empty cells (mean of 20 salts) against a stateless hash draw")
    n = 64
    lhs_e = [empty_cells(design(n, 2, salt), n, 8) for salt in range(1, 21)]
    hsh_e = [empty_cells(hash_design(n, 2, salt), n, 8) for salt in range(1, 21)]
    print("  Latin hypercube  min %d  mean %.1f  max %d" % (min(lhs_e), sum(lhs_e) / 20, max(lhs_e)))
    print("  hash draw        min %d  mean %.1f  max %d" % (min(hsh_e), sum(hsh_e) / 20, max(hsh_e)))
    print("  (a hash draw leaves cells empty in expectation 64 * (63/64)^64 = %.1f)" % (64 * (63 / 64) ** 64))

    print()
    print("== largest absolute correlation between two columns, d = 8, over 20 salts")
    print("   N   Latin hypercube (min mean max)   hash draw (min mean max)")
    for n in (16, 32, 64, 128, 256):
        a = [max_abs_corr(design(n, 8, s), 8) for s in range(1, 21)]
        b = [max_abs_corr(hash_design(n, 8, s), 8) for s in range(1, 21)]
        print("  %3d   %.2f %.2f %.2f                  %.2f %.2f %.2f" % (
            n, min(a), sum(a) / 20, max(a), min(b), sum(b) / 20, max(b)))

    print()
    print("== the exact stratification of every axis: sorted cells of axis 3, N = 12")
    d12 = design(12, 4)
    print("  axis 3 cells by point:", [p[3] for p in d12])
    assert sorted(p[3] for p in d12) == list(range(12))
    print("  points 0..3 of N = 12 (cell of each axis):")
    for i in range(4):
        print("   i=%d  %s" % (i, d12[i]))


if __name__ == "__main__":
    main()
