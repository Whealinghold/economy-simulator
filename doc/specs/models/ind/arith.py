"""Integer arithmetic rules of spec-indicators.md section 5, checked against
exact rationals. Standard library only; run in Docker, never on the host.

Every function here is a model of a rule of the spec, not of the library.
"""
from fractions import Fraction
import itertools
import random

BP = 10_000
MAX_INDEX_BP = 2**31 - 1


def rdiv(n, d):
    """n / d rounded half to even; d > 0; n of either sign (spec 5.2)."""
    assert d > 0
    q, r = divmod(abs(n), d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q if n >= 0 else -q


def isqrt_round(n):
    """Nearest integer to sqrt(n); a square root of an integer is never a
    half, so there is no tie (spec 5.3.4)."""
    r = 1
    # integer square root by Newton, then the round-to-nearest test
    if n == 0:
        return 0
    x = n
    y = (x + 1) // 2
    while y < x:
        x = y
        y = (x + n // x) // 2
    r = x
    return r + 1 if n - r * r > r else r


def gini_bp(rows):
    """Weighted population Gini in basis points (spec 5.4). rows: [(w, v)],
    w >= 1, v >= 0. Returns (bp, N, D) with G = N / D."""
    rows = sorted(rows, key=lambda r: r[1])
    W = sum(w for w, _ in rows)
    S = sum(w * v for w, v in rows)
    if W == 0 or S == 0:
        return None
    T = 0
    acc = 0
    for w, v in rows:
        acc += w * (2 * T + w * v)
        T += w * v
    N = W * S - acc
    D = W * S
    return rdiv(BP * N, D), N, D


def gini_pairwise(rows):
    W = sum(w for w, _ in rows)
    S = sum(w * v for w, v in rows)
    tot = sum(wi * wj * abs(vi - vj) for (wi, vi) in rows for (wj, vj) in rows)
    return Fraction(tot, 2 * W * S)


def quantile(rows, q_bp):
    """Weighted nearest rank (spec 5.5.2)."""
    rows = sorted(rows, key=lambda r: r[1])
    W = sum(w for w, _ in rows)
    k = max(1, -(-q_bp * W // BP))
    cum = 0
    for w, v in rows:
        cum += w
        if cum >= k:
            return v
    raise AssertionError


def check_gini(trials=2000, seed=1):
    rng = random.Random(seed)
    worst = 0
    for _ in range(trials):
        n = rng.randint(1, 9)
        rows = [(rng.randint(1, 6), rng.randint(0, 50)) for _ in range(n)]
        r = gini_bp(rows)
        if r is None:
            continue
        bp, N, D = r
        assert Fraction(N, D) == gini_pairwise(rows), rows
        assert 0 <= bp <= BP
        # the exact value is within half a basis point of the integer
        assert abs(Fraction(BP * N, D) - bp) <= Fraction(1, 2)
        worst = max(worst, abs(Fraction(BP * N, D) - bp))
    return worst


def check_gini_max():
    """One household holds everything among W: G = (W - 1) / W."""
    out = []
    for W in (1, 2, 3, 10, 1000):
        rows = [(W - 1, 0), (1, 5)] if W > 1 else [(1, 5)]
        r = gini_bp(rows)
        out.append((W, r[0], rdiv(BP * (W - 1), W)))
        assert r[0] == rdiv(BP * (W - 1), W)
    return out


def check_isqrt(trials=20000, seed=2):
    rng = random.Random(seed)
    for _ in range(trials):
        n = rng.randint(0, 2**62)
        r = isqrt_round(n)
        # nearest integer: |r - sqrt(n)| <= 1/2  <=>  (r - 1/2)^2 <= n <= (r + 1/2)^2
        assert 4 * r * r - 4 * r + 1 <= 4 * n <= 4 * r * r + 4 * r + 1, n
    # the ties of the test never happen: r^2 + r + 1/4 is not an integer
    return True


def check_chain(links=400, seed=3):
    """A chain of Laspeyres links rounded half to even at each link against the
    exact rational product: the drift is bounded by the sum of the link errors
    (spec 5.3.5)."""
    rng = random.Random(seed)
    c = BP
    exact = Fraction(BP)
    worst = Fraction(0)
    for i in range(links):
        link = rng.randint(9_800, 10_300)  # a link index in bp
        c = rdiv(c * link, BP)
        exact = exact * Fraction(link, BP)
        worst = max(worst, abs(exact - c))
    return float(worst), links


def check_bounds():
    """The magnitudes of 5.6: MaxMoney 10^18, MaxWeight 10^6, W up to
    MaxHouseholds * groups; W * S * 10^4 against 2^127."""
    max_money = 10**18
    s = 3 * max_money  # the supply stays below 3 * MaxMoney (EXT 13.2)
    for w in (10**6, 10**9, 10**12, 10**15):
        need = w * s * BP
        print(f"W={w:>16} S=3e18  10^4*W*S = {need:.3e}  below 2^127 (1.7e38): {need < 2**127}")
    print("2^127 =", 2**127)


if __name__ == "__main__":
    print("gini: worst rounding error over random rows (bp):", float(check_gini()))
    print("gini max, (W-1)/W:", check_gini_max())
    print("isqrt_round nearest-integer property:", check_isqrt())
    print("chain drift (max abs error in bp over links):", check_chain())
    check_bounds()
    # half to even
    assert [rdiv(n, 2) for n in (1, 3, 5, -1, -3, -5)] == [0, 2, 2, 0, -2, -2]
    assert rdiv(25, 10) == 2 and rdiv(35, 10) == 4 and rdiv(-25, 10) == -2
    print("half-even checks ok")


# ---- revision after review: rows are {Weight, Total} (spec 4.5.1, 5.4.1) ----
from functools import cmp_to_key


def _cmp(a, b):
    # order by Total/Weight without dividing: Ta * wb against Tb * wa
    l, r = a[1] * b[0], b[1] * a[0]
    return -1 if l < r else (1 if l > r else 0)


def gini_totals(rows):
    """rows: [(w, T)], T the cohort's exact total, T >= 0. No division before
    the last one; sort by T/w by cross multiplication."""
    rows = sorted(rows, key=cmp_to_key(_cmp))
    W = sum(w for w, _ in rows)
    S = sum(t for _, t in rows)
    if W == 0 or S == 0:
        return None
    C = 0
    acc = 0
    for w, t in rows:
        acc += w * (2 * C + t)
        C += t
    N = W * S - acc
    D = W * S
    return rdiv(BP * N, D), N, D


def check_gini_totals(trials=3000, seed=4):
    rng = random.Random(seed)
    for _ in range(trials):
        n = rng.randint(1, 8)
        rows = [(rng.randint(1, 6), rng.randint(0, 120)) for _ in range(n)]  # totals need not divide
        r = gini_totals(rows)
        if r is None:
            continue
        W = sum(w for w, _ in rows)
        S = sum(t for _, t in rows)
        # pairwise definition with v = T / w as an exact fraction, a row counting w times
        tot = sum(wi * wj * abs(Fraction(ti, wi) - Fraction(tj, wj))
                  for (wi, ti) in rows for (wj, tj) in rows)
        assert Fraction(r[1], r[2]) == tot / (2 * W * S), rows
    return True


def quantile_totals(rows, q_bp):
    """Nearest rank by T/w; the value returned is rdiv(T, w) of the row."""
    rows = sorted(rows, key=cmp_to_key(_cmp))
    W = sum(w for w, _ in rows)
    k = max(1, -(-q_bp * W // BP))
    cum = 0
    for w, t in rows:
        cum += w
        if cum >= k:
            return rdiv(t, w)


def share_below_totals(rows, x):
    """households whose T/w < x, i.e. T < x * w, exact."""
    W = sum(w for w, _ in rows)
    return rdiv(BP * sum(w for w, t in rows if t < x * w), W)


def vectors():
    out = {}
    # deflator: nominal 1 300, real 1 000 -> 13 000 bp; N against R * deflator / 10 000
    N, R = 1300, 1000
    d = rdiv(BP * N, R)
    out["deflator_bp"] = d
    out["real_times_deflator"] = rdiv(R * d, BP)
    assert abs(N - rdiv(R * d, BP)) <= -(-R // 20000)       # the bound of FR-IND-013
    # a non-exact case: N = 1 001, R = 7
    N, R = 1001, 7
    d = rdiv(BP * N, R)
    assert abs(N * 1 - rdiv(R * d, BP)) <= -(-R // 20000) + 0
    # clamp
    big = rdiv(BP * 10**15, 3)
    out["clamped"] = min(big, MAX_INDEX_BP)
    # quantiles and shares over totals (a cohort total that does not divide)
    rows = [(1, 0), (2, 41), (4, 160), (1, 60)]
    out["P50"] = quantile_totals(rows, 5000)          # row of 2 holds 41: 20.5 each
    out["P25"] = quantile_totals(rows, 2500)          # rank 2: the cohort of 2, rdiv(41, 2) = 20 (half to even)
    out["share_below_20.5"] = share_below_totals(rows, 21)
    out["gini_nondivisible"] = gini_totals(rows)[0]
    # the property: a total that divides gives the old per-household form
    assert gini_totals([(1, 0), (2, 40), (4, 160), (1, 60)])[0] == 2788
    return out


if __name__ == "__main__":
    print("gini over totals vs pairwise (non-divisible totals):", check_gini_totals())
    print("vectors:", vectors())
