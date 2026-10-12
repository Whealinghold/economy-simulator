"""The grid and neighbour rule of spec-calibration.md 5.2, in integers.

Half-to-even division, the `geo`, `lin` and `set:` levels, the cut of a damper's off value
(and of every level beyond it), the invalid-level skip, the edge record (`bound`, `off`,
`rule`), the `z` case (a base of 0 has no grid) and a checked multiply that reports
overflow. The widening of 5.3 is not modelled. Run in Docker only:
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/clb":/w \
    -w /w python:3.12-slim python grid.py
"""
import random

INT64_MAX = (1 << 63) - 1


class GridOverflow(Exception):
    """The checked multiply of the Go grid reports overflow: an error, never a wrapped level."""


def rdiv(a, b):
    """a / b rounded half to even (b > 0), integers only."""
    q, r = divmod(a, b)
    twice = 2 * r
    if twice > b or (twice == b and q % 2 == 1):
        q += 1
    return q


def checked_mul(a, b):
    c = a * b
    if c > INT64_MAX:
        raise GridOverflow(f"{a} * {b}")
    return c


def geo_levels(b, lo, hi):
    """b * 2^k for k >= 0 and rdiv(b, 2^-k) for k < 0, kept in [max(lo, 1), hi]."""
    assert b >= 1 and lo <= b <= hi
    floor = max(lo, 1)
    out = {b}
    v = b
    while True:  # upward: stop at hi; a product above int64 is an overflow error
        if v > hi:
            break
        try:
            v = checked_mul(v, 2)
        except GridOverflow:
            break      # v > hi already holds for any hi of int64, so the level is outside
        if v > hi:
            break
        out.add(v)
    k = 1
    while True:  # downward
        v = rdiv(b, 2 ** k)
        if v < floor:
            break
        out.add(v)
        if v == floor:
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


def cut(levels, b, off):
    """5.2 step 2: remove the off value and every level beyond it (away from b), except b."""
    if off is None:
        return list(levels)
    if off < b:
        return [v for v in levels if v == b or v > off]
    if off > b:
        return [v for v in levels if v == b or v < off]
    return list(levels)       # b is at its off value: nothing is cut, b stays a level


def neighbours(levels, b, valid=lambda v: True, lo=None, hi=None, off=None, floor=None):
    """(lower, upper, edges, kept): `levels` is the grid of step 1; the cut of step 2 and
    the invalid skip of step 3 are applied here. edges is a list of
    ('below'|'above', 'bound'|'off'|'rule') for a side with no kept valid level:
    `off` when the nearest level of the uncut grid on that side was the off value,
    `bound` when the next level would be outside [lo, hi] or below the grid's own floor
    (a geo grid has no level below 1: `max(lo, 1)` is its lower bound), `rule` when it was invalid."""
    assert b in levels
    kept = cut(levels, b, off)
    below = [v for v in kept if v < b]
    above = [v for v in kept if v > b]
    edges = []
    lower = next((v for v in reversed(below) if valid(v)), None)
    upper = next((v for v in above if valid(v)), None)

    def kind(side):
        cand = [v for v in levels if (v < b if side == "below" else v > b)]
        nearest = (max(cand) if side == "below" else min(cand)) if cand else None
        if nearest is None:
            lo_eff = max((x for x in (lo, floor) if x is not None), default=None)
            at_bound = (lo_eff is not None and b <= lo_eff) if side == "below" else (hi is not None and b >= hi)
            return "bound" if at_bound else "rule"
        if off is not None and nearest == off:
            return "off"
        if nearest not in kept:       # beyond the cut with no kept level in between
            return "off"
        return "rule"

    if lower is None:
        edges.append(("below", kind("below")))
    if upper is None:
        edges.append(("above", kind("above")))
    return lower, upper, edges, kept


def build(kind, b, lo, hi, off=None, valid=lambda v: True, z=False):
    """A whole grid. Returns None for a `z` row whose base is 0 (no grid: unsearched at_off)."""
    if z and b == 0:
        return None
    if kind == "geo":
        lv = geo_levels(b, lo, hi)
    else:
        lv = lin_levels(b, int(kind[3:]), lo, hi)
    return neighbours(lv, b, valid, lo, hi, off, floor=1 if kind == "geo" else None)


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
        lo_n, up_n, _, _ = neighbours(g, b)
        i = g.index(b)
        assert lo_n == (g[i - 1] if i > 0 else None)
        assert up_n == (g[i + 1] if i + 1 < len(g) else None)
        # the cut: with an off value at lo or hi, no kept level other than b equals it, and b stays
        for off in (lo, hi):
            kept = cut(lin_levels(b, 7, lo, hi), b, off)
            assert b in kept and all(v == b or v != off for v in kept)
    # half to even: 3/2 = 1.5 -> 2, 5/2 = 2.5 -> 2, 7/2 = 3.5 -> 4
    assert (rdiv(3, 2), rdiv(5, 2), rdiv(7, 2), rdiv(1, 2), rdiv(50, 16)) == (2, 2, 4, 0, 3)
    # an overflowing doubling is an error of the checked multiply, never a wrapped level
    try:
        checked_mul(INT64_MAX, 2)
        raise AssertionError("no overflow reported")
    except GridOverflow:
        pass
    assert geo_levels(1 << 62, 1, INT64_MAX)[-1] == 1 << 62      # 2^63 would overflow: not a level
    # a z row at 0 has no grid; a lin base at its off value keeps the base, below is a bound
    assert build("geo", 0, 0, 1000, z=True) is None
    lo_n, up_n, edges, kept = build("lin500", 0, 0, 9999, off=0)
    assert (lo_n, up_n, edges) == (None, 500, [("below", "bound")]) and kept[0] == 0
    # RestBand: 0 is cut, 250 is the lower neighbour, below it is an `off` edge
    lo_n, up_n, edges, kept = build("lin250", 500, 0, 10000, off=0)
    assert (lo_n, up_n) == (250, 750) and 0 not in kept
    lo_n, up_n, edges, kept = build("lin250", 250, 0, 10000, off=0)
    assert (lo_n, up_n, edges) == (None, 500, [("below", "off")])


if __name__ == "__main__":
    selftest()
    print("selftest ok: 20000 random (base, bound) cases, levels sorted, unique, in bounds, base present;"
          " the cut, the z case, the overflow error")
    print()
    cases = [
        ("World.CostFloor", "geo", 7000, 1, 1_000_000, None, None),
        ("World.BeliefStep", "geo", 200, 1, 2500, None, "I01 at a cheapest price of 96"),
        ("World.EntryShare", "geo", 50, 0, 10000, None, None),
        ("World.RestBand", "lin250", 500, 0, 10000, 0, "off=0 is cut: the edge below 250 is `off`"),
        ("Role.TargetSpread", "lin500", 0, 0, 9999, 0, "base at its off value: kept; below is a bound"),
        ("World.CoverTicks (base 1)", "geo", 1, 0, 1000, None, "geo floor of 1 above a bound of 0: a bound edge (FR-CLB-010: max(lo, 1))"),
        ("World.CoverTicks", "geo", 3, 1, 1000, None, None),
        ("World.ReviewTicks", "geo", 10, 1, 1000, 1, "off=1 is the lower bound: cut"),
        ("World.ProduceHaltRatio", "geo", 30000, 10000, 1_000_000, None, None),
        ("World.WelfareWindow", "geo", 30, 1, 1000, None, None),
        ("Role.TargetRatio", "lin1000", 15000, 10000, 1_000_000, None, None),
        ("Role.MinWorkShare", "lin1000", 4000, 0, 6849, 0, "I05 at ExpectedWorkTicks 250, TicksPerYear 365; off=0 cut"),
        ("Subsistence.ExitTicks", "lin1", 3, 1, 1000, None, "I07 with EnterTicks 2"),
    ]
    print(f"{'knob':26s} {'base':>8s}  kept levels near the base (invalid ones not yet skipped)  neighbours / edges")
    for name, kind, b, lo, hi, off, note in cases:
        if name == "World.BeliefStep":
            valid = lambda v: -(-10000 // v) <= 96  # ceil(10000 / v) <= cheapest price 96
        elif name == "Subsistence.ExitTicks":
            valid = lambda v: v > 2
        else:
            valid = lambda v: True
        lower, upper, edges, kept = build(kind, b, lo, hi, off=off, valid=valid)
        i = kept.index(b)
        window = kept[max(0, i - 2): i + 3]
        print(f"{name:26s} {b:8d}  {str(window):44s} {lower} / {upper} {edges if edges else ''}" + (f"   [{note}]" if note else ""))
    print()
    print("a z row (MoneyFlow.Limit, geo) with the pack's value 0: no grid, unsearched at_off ->",
          build("geo", 0, 0, 10**15, z=True))
