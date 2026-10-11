"""The grid and neighbour rule of spec-calibration.md 5.2, in integers.

Half-to-even division, the `geo` and `lin` grids (`set:` is a listed order and has no
rule to model; the widening of 5.3 is not modelled), the invalid-level skip, the
edge record (`bound`, `rule`, `off`) and a checked multiply. Run in Docker only:
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/clb":/w \
    -w /w python:3.12-slim python grid.py
"""
import random


def rdiv(a, b):
    """a / b rounded half to even (b > 0), integers only."""
    q, r = divmod(a, b)
    twice = 2 * r
    if twice > b or (twice == b and q % 2 == 1):
        q += 1
    return q


def geo_levels(b, lo, hi):
    """b * 2^k for k >= 0 and rdiv(b, 2^-k) for k < 0, kept in [max(lo, 1), hi]."""
    assert b >= 1 and lo <= b <= hi
    floor = max(lo, 1)
    out = {b}
    v = b
    while True:  # upward
        v *= 2
        if v > hi or v >= 1 << 63:   # the checked multiply of Go reports overflow; here it stops
            break
        out.add(v)
    k = 1
    while True:  # downward
        v = rdiv(b, 2 ** k)
        if v < floor:
            break
        out.add(v)
        if v == floor:  # further halvings are below the floor or repeat it
            break
        k += 1
    return sorted(x for x in out if floor <= x <= hi)


def lin_levels(b, step, lo, hi):
    assert step >= 1 and lo <= b <= hi
    out = [b]
    v = b - step
    while v >= lo:
        out.append(v)
        v -= step
    v = b + step
    while v <= hi:
        out.append(v)
        v += step
    return sorted(out)


def neighbours(levels, b, valid=lambda v: True, lo=None, hi=None, off=None):
    """(lower, upper, edges): the nearest valid level on each side of b.
    edges is a list of ('below'|'above', 'bound'|'rule'|'off') for a side with none:
    `bound` when b is at the legal bound lo or hi, `off` when the only level beyond is the
    damper's off value (never a level, 4.3.1), `rule` otherwise (an invalid level or the
    geo floor of 1 above a bound of 0)."""
    assert b in levels
    below = [v for v in levels if v < b and v != off]
    above = [v for v in levels if v > b and v != off]
    edges = []
    lower = next((v for v in reversed(below) if valid(v)), None)
    if lower is None:
        cut_off = off is not None and off < b and off in levels
        edges.append(("below", "off" if cut_off else ("bound" if lo is not None and b <= lo else "rule")))
    upper = next((v for v in above if valid(v)), None)
    if upper is None:
        cut_off = off is not None and off > b and off in levels
        edges.append(("above", "off" if cut_off else ("bound" if hi is not None and b >= hi else "rule")))
    return lower, upper, edges


def selftest(rounds=20000, seed=7):
    rng = random.Random(seed)
    for _ in range(rounds):
        b = rng.randint(1, 3000)
        lo = rng.randint(0, b)
        hi = b + rng.randint(0, 100000)
        for levels in (geo_levels(b, lo, hi), lin_levels(b, rng.randint(1, 500), lo, hi)):
            assert b in levels
            assert levels == sorted(set(levels))
            assert all(lo <= v <= hi for v in levels)
        g = geo_levels(b, lo, hi)
        lo_n, up_n, _ = neighbours(g, b)
        i = g.index(b)
        assert lo_n == (g[i - 1] if i > 0 else None)
        assert up_n == (g[i + 1] if i + 1 < len(g) else None)
    # half to even: 3/2 = 1.5 -> 2, 5/2 = 2.5 -> 2, 7/2 = 3.5 -> 4
    assert (rdiv(3, 2), rdiv(5, 2), rdiv(7, 2), rdiv(1, 2), rdiv(50, 16)) == (2, 2, 4, 0, 3)


if __name__ == "__main__":
    selftest()
    print("selftest ok: 20000 random (base, bound) cases, levels sorted, unique, in bounds, base present")
    print()
    cases = [
        ("World.CostFloor", "geo", 7000, 1, 1_000_000, None),
        ("World.BeliefStep", "geo", 200, 1, 2500, "I01 at a cheapest price of 96"),
        ("World.EntryShare", "geo", 50, 0, 10000, None),
        ("World.RestBand", "lin250", 500, 0, 10000, "off=0 is cut: edge below is `off`"),
        ("World.CoverTicks (base 1)", "geo", 1, 0, 1000, "geo floor of 1 above a bound of 0: a rule edge"),
        ("World.CoverTicks", "geo", 3, 1, 1000, None),
        ("World.ReviewTicks", "geo", 10, 1, 1000, None),
        ("World.ProduceHaltRatio", "geo", 30000, 10000, 1_000_000, None),
        ("World.WelfareWindow", "geo", 30, 1, 1000, None),
        ("Role.TargetRatio", "lin1000", 15000, 10000, 1_000_000, None),
        ("Role.MinWorkShare", "lin1000", 4000, 0, 6849, "I05 at ExpectedWorkTicks 250, TicksPerYear 365"),
        ("Subsistence.ExitTicks", "lin1", 3, 1, 1000, "I07 with EnterTicks 2"),
    ]
    print(f"{'knob':26s} {'base':>8s}  levels near the base, skipped ones included       neighbours / edges")
    for name, kind, b, lo, hi, note in cases:
        lv = geo_levels(b, lo, hi) if kind == "geo" else lin_levels(b, int(kind[3:]), lo, hi)
        if name == "World.BeliefStep":
            floor_ok = lambda v: -(-10000 // v) <= 96  # ceil(10000 / v) <= cheapest price 96
            lower, upper, edges = neighbours(lv, b, floor_ok, lo, hi)
        elif name == "Role.MinWorkShare":
            lower, upper, edges = neighbours(lv, b)
        elif name == "Subsistence.ExitTicks":
            lower, upper, edges = neighbours(lv, b, lambda v: v > 2)
        elif name == "World.RestBand":
            lower, upper, edges = neighbours(lv, b, lo=lo, hi=hi, off=0)
        else:
            lower, upper, edges = neighbours(lv, b, lo=lo, hi=hi)
        i = lv.index(b)
        window = lv[max(0, i - 2): i + 3]
        print(f"{name:26s} {b:8d}  {str(window):44s} {lower} / {upper} {edges if edges else ''}" + (f"   [{note}]" if note else ""))
