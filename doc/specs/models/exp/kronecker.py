"""Spec section 4.5: the REJECTED space-filling design of spec-experiments.md (evidence).

Computes, in exact integers, the multipliers of the generalised golden ratio
(Kronecker) sequence in 2^64 fixed point, for d = 1 .. 16 dimensions:

    phi_d  = the positive root of x^(d+1) = x + 1
    alpha_j = floor(2^64 * frac(phi_d^-(j+1)))      j = 0 .. d-1
    u_{i,j} = (c + (i+1) * alpha_j) mod 2^64         i = 0, 1, 2, ...   c = 2^63

and checks the properties the spec claims: d = 1 gives the SplitMix64 gamma
0x9E3779B97F4A7C15; every one-dimensional projection is equidistributed (counts
per bin differ by at most 2 from N / B); and the design fills a square better
than a stateless hash draw of the same size (fewer empty cells, larger minimum
distance). A model of the rule, not of the library (note N-027).
"""
S_BITS = 320
S = 1 << S_BITS
MASK = (1 << 64) - 1


def phi_scaled(d):
    """phi_d * S as an integer: bisection on f(x) = x^(d+1) - x - 1 over [S, 2S]."""
    lo, hi = S, 2 * S
    while hi - lo > 1:
        mid = (lo + hi) // 2
        # f(mid) >= 0  <=>  mid^(d+1) >= (mid + S) * S^d
        if mid ** (d + 1) >= (mid + S) * S ** d:
            hi = mid
        else:
            lo = mid
    return lo


def alphas(d):
    x = phi_scaled(d)
    inv = S * S // x            # 1/phi, scaled by S
    out, p = [], inv
    for _ in range(d):
        frac = p % S
        out.append(frac >> (S_BITS - 64))
        p = p * inv // S
    return out


def point(i, alpha, c=1 << 63):
    return [(c + (i + 1) * a) & MASK for a in alpha]


def mulhi(u, n):
    return (u * n) >> 64


def splitmix_fin(z):
    z &= MASK
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & MASK
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & MASK
    return z ^ (z >> 31)


def hash_point(i, d, salt=0x243F6A8885A308D3):
    """A stateless random design: u_{i,j} = finalise(salt + (i * d + j + 1) * gamma)."""
    g = 0x9E3779B97F4A7C15
    return [splitmix_fin(salt + (i * d + j + 1) * g) for j in range(d)]


def axis_deviation(alpha_j, n, bins):
    counts = [0] * bins
    for i in range(n):
        counts[mulhi(point(i, [alpha_j])[0], bins)] += 1
    lo, hi = min(counts), max(counts)
    return lo, hi


def empty_cells(points, side):
    seen = set()
    for p in points:
        seen.add((mulhi(p[0], side), mulhi(p[1], side)))
    return side * side - len(seen)


def min_dist2(points):
    """Smallest squared toroidal distance between two points, in units of 2^-64 squared / 2^128."""
    best = None
    h = 1 << 63
    for a in range(len(points)):
        for b in range(a + 1, len(points)):
            s = 0
            for x, y in zip(points[a], points[b]):
                dx = abs(x - y)
                dx = min(dx, (1 << 64) - dx)
                s += (dx >> 32) ** 2
            best = s if best is None else min(best, s)
    return best


def main():
    print("== multipliers alpha_j = floor(2^64 * frac(phi_d^-(j+1)))")
    table = {}
    for d in (1, 2, 4, 8):
        a = alphas(d)
        table[d] = a
        print("d=%2d" % d)
        for j, v in enumerate(a):
            print("   j=%2d  0x%016x" % (j, v))
    assert table[1][0] == 0x9E3779B97F4A7C15, "d = 1 must give the SplitMix64 gamma"
    print("check: d=1, j=0 equals the SplitMix64 gamma 0x9E3779B97F4A7C15")

    print()
    print("== one-dimensional equidistribution: counts per bin for the first N points (d = 4, axis j)")
    print("   N    bins  axis  min  max  (a uniform grid would give N/bins)")
    for n, bins in ((16, 16), (50, 10), (100, 10), (256, 16), (1000, 20)):
        worst = (10 ** 9, 0)
        for j in range(4):
            lo, hi = axis_deviation(table[4][j], n, bins)
            worst = (min(worst[0], lo), max(worst[1], hi))
        print("  %4d  %4d  all   %3d  %3d   (N/bins = %.2f)" % (n, bins, worst[0], worst[1], n / bins))

    print()
    print("== a square of 8 x 8 cells, N = 64 points: empty cells and the smallest distance")
    n, side = 64, 8
    a2 = table[2]
    kron = [point(i, a2) for i in range(n)]
    hsh = [hash_point(i, 2) for i in range(n)]
    print("  Kronecker d=2   empty cells %2d   min distance^2 %d" % (empty_cells(kron, side), min_dist2(kron)))
    print("  hash draw  d=2  empty cells %2d   min distance^2 %d" % (empty_cells(hsh, side), min_dist2(hsh)))
    for salt in (1, 2, 3, 4, 5):
        h = [hash_point(i, 2, salt) for i in range(n)]
        print("  hash draw salt %d empty cells %2d   min distance^2 %d" % (salt, empty_cells(h, side), min_dist2(h)))

    print()
    print("== stability under growth: the first 8 points of d = 2 do not change when N grows")
    first8 = [point(i, a2) for i in range(8)]
    again = [point(i, a2) for i in range(100)][:8]
    assert first8 == again
    print("  equal: point i depends on i and d only (stateless)")

    print()
    print("== the first four points of d = 2, as a share of the unit square in basis points")
    for i in range(4):
        p = point(i, a2)
        print("  i=%d  (%5d, %5d) bp" % (i, mulhi(p[0], 10000), mulhi(p[1], 10000)))


if __name__ == "__main__":
    main()
