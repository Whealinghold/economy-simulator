"""The choose step (7.2), the Pareto set (7.5) and the alias test (10.2) of
spec-calibration.md, in integers. Run in Docker only (see grid.py).

A POINT is a dict: index, levels (knob -> grid index relative to the base, 0 = the
base), passes (criterion -> seeds inside of R). A CRITERION is (name, kind, need)
with kind X, O, D (detector) or B; X, O and D need all R seeds, B needs ceil(9R/10).
"""


def need(kind, R):
    return R if kind in ("X", "O", "D") else R - R // 10   # = ceil(9R/10), exact: not a half-to-even division (FR-CLB-013)


def feasible(points, criteria, R):
    out = []
    for p in points:
        if all(p["passes"][name] >= need(kind, R) for name, kind in criteria):
            out.append(p)
    return out


def distance(p):
    return sum(abs(v) for v in p["levels"].values())


def changed(p):
    return sum(1 for v in p["levels"].values() if v != 0)


def choose(points, criteria, R):
    """The feasible point closest to the base in grid steps; ties: fewer knobs changed,
    then the lower index. None when no point is feasible."""
    f = feasible(points, criteria, R)
    if not f:
        return None
    return min(f, key=lambda p: (distance(p), changed(p), p["index"]))


def pareto(points, criteria, R):
    """Among points that pass every X, O and D criterion, the non-dominated vectors of
    the B pass counts. Returns [] when no point passes the exact facts."""
    exact = [(n, k) for n, k in criteria if k != "B"]
    bs = [n for n, k in criteria if k == "B"]
    ok = [p for p in points if all(p["passes"][n] >= need(k, R) for n, k in exact)]
    out = []
    for p in ok:
        dominated = False
        for q in ok:
            if q is p:
                continue
            ge = all(q["passes"][n] >= p["passes"][n] for n in bs)
            gt = any(q["passes"][n] > p["passes"][n] for n in bs)
            if ge and gt:
                dominated = True
                break
        if not dominated:
            out.append(p)
    return out


E_MAX = 1 << 26        # |e| bound of the alias test (10.2); EXP clamps at 2^31 - 1
M_MAX = 16             # measures
I128 = 1 << 127


def rdiv(n, d):
    """integer division, half to even (IND 5.2.2)"""
    if d < 0:
        n, d = -n, -d
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q


def collinear(a, b, tol_bp):
    """cos^2 of two integer vectors at least 10 000 - tol_bp, in integers (no division
    before the comparison). Both vectors must be non-zero. A vector above the bounds
    is an overflow error, as a checked multiply of Go would report (architecture.md 3.5)."""
    if len(a) > M_MAX or any(abs(x) > E_MAX for x in a + b):
        raise OverflowError("alias test input out of bounds")
    dot = sum(x * y for x, y in zip(a, b))
    na = sum(x * x for x in a)
    nb = sum(y * y for y in b)
    if na == 0 or nb == 0:
        return False
    lhs, rhs = 10000 * dot * dot, (10000 - tol_bp) * na * nb
    assert lhs < I128 and rhs < I128    # fits signed 128 bits at the stated bounds
    return lhs >= rhs


def aliased(a, b, tol_bp):
    """The procedure's test (10.2): a reading that is None (OK false), above 2^26 in absolute
    value or Clamped (EXP clamps at 2^31 - 1) is EXCLUDED, never wrapped and never an error;
    the test runs over the measures where both are defined. More than 16 measures are tested
    by fact group by the caller."""
    pairs = [(x, y) for x, y in zip(a, b)
             if x is not None and y is not None and abs(x) <= E_MAX and abs(y) <= E_MAX]
    if not pairs:
        return False
    return collinear([x for x, _ in pairs], [y for _, y in pairs], tol_bp)


def to_level(levels, v):
    """A Spread value maps to the nearest grid level; a tie goes to the LOWER level (7.2.3: a rule
    of its own, not a division)."""
    return min(levels, key=lambda l: (abs(l - v), l))


def cos2_bp(a, b):
    dot = sum(x * y for x, y in zip(a, b))
    na = sum(x * x for x in a)
    nb = sum(y * y for y in b)
    return rdiv(10000 * dot * dot, na * nb)   # half to even


if __name__ == "__main__":
    R = 10
    crit = [("x_o", "X"), ("r5", "D"), ("p02", "B"), ("p04", "B")]
    pts = [
        dict(index=0, levels={"cost_floor": 0, "evidence_share": 0}, passes={"x_o": 10, "r5": 10, "p02": 7, "p04": 10}),
        dict(index=1, levels={"cost_floor": -1, "evidence_share": 0}, passes={"x_o": 10, "r5": 10, "p02": 9, "p04": 10}),
        dict(index=2, levels={"cost_floor": +1, "evidence_share": 0}, passes={"x_o": 10, "r5": 10, "p02": 4, "p04": 10}),
        dict(index=3, levels={"cost_floor": 0, "evidence_share": -1}, passes={"x_o": 10, "r5": 10, "p02": 8, "p04": 10}),
        dict(index=4, levels={"cost_floor": 0, "evidence_share": +1}, passes={"x_o": 10, "r5": 10, "p02": 6, "p04": 9}),
        dict(index=5, levels={"cost_floor": -1, "evidence_share": +1}, passes={"x_o": 10, "r5": 10, "p02": 10, "p04": 10}),
        dict(index=6, levels={"cost_floor": -1, "evidence_share": -1}, passes={"x_o": 10, "r5": 10, "p02": 10, "p04": 10}),
    ]
    f = feasible(pts, crit, R)
    print("feasible points:", [p["index"] for p in f], "distances:", [distance(p) for p in f])
    c = choose(pts, crit, R)
    print("chosen:", c["index"], c["levels"])
    assert [p["index"] for p in f] == [1, 5, 6] and c["index"] == 1
    # the default (point 0) is not feasible here: p02 passes 7 of 10, below the 9 that SR2 needs
    assert need("B", 10) == 9 and need("B", 40) == 36

    pts2 = [
        dict(index=0, levels={"k": 0}, passes={"x_o": 10, "r5": 10, "p02": 10, "p04": 5}),
        dict(index=1, levels={"k": 1}, passes={"x_o": 10, "r5": 10, "p02": 7, "p04": 9}),
        dict(index=2, levels={"k": 2}, passes={"x_o": 10, "r5": 10, "p02": 6, "p04": 6}),
        dict(index=3, levels={"k": -1}, passes={"x_o": 10, "r5": 10, "p02": 9, "p04": 4}),
    ]
    assert choose(pts2, crit, R) is None
    par = pareto(pts2, crit, R)
    print("no feasible point; the Pareto set:", [p["index"] for p in par])
    assert [p["index"] for p in par] == [0, 1]

    # the alias test: three measures (Pinned.Share, Turnover.Volume, PriceRange), elasticities in bp
    a = [300, -450, 0]
    b = [600, -900, 0]
    c3 = [300, 100, -200]
    print("alias A,B:", collinear(a, b, 500), cos2_bp(a, b), "  A,C:", collinear(a, c3, 500), cos2_bp(a, c3))
    assert collinear(a, b, 500) and not collinear(a, c3, 500)
    # one measure cannot tell two knobs apart: every pair of non-zero scalars is collinear
    assert collinear([120], [-35], 500)
    print("one measure: [120] and [-35] collinear:", collinear([120], [-35], 500), cos2_bp([120], [-35]))
    # the bounds of the 128-bit test: the largest legal vectors still fit, one above is an overflow error
    big = [E_MAX] * M_MAX
    assert collinear(big, big, 500)
    for bad in ([E_MAX + 1], [1] * (M_MAX + 1)):
        try:
            collinear(bad, bad, 500)
            raise SystemExit("overflow not reported")
        except OverflowError:
            pass
    print("bounds: |e| <= 2^26, m <= 16 fit 128 bits; one above is an overflow error")
    # a reading above 2^26, a clamped one (2^31 - 1) or an undefined one is excluded, not an error
    assert aliased([2**31 - 1, 5, None], [999, 10, 7], 500)          # only the second measure is read
    assert not aliased([2**31 - 1], [4], 500) and not aliased([None], [None], 500)
    # half to even in the rounded cos^2 and the exact seed threshold
    assert rdiv(5, 2) == 2 and rdiv(7, 2) == 4 and rdiv(-5, 2) == -2
    assert need("B", 25) == 23 and need("B", 10) == 9 and need("B", 50) == 45
    # a Spread value maps to the nearest level; a tie goes to the lower level (a rule of its own)
    assert to_level([100, 200, 400], 300) == 200 and to_level([100, 200, 400], 290) == 200
