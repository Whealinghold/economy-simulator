"""Spec section 17: the arithmetic of the worked examples of spec-experiments.md.

Names (`cost_floor`, `belief_step`, `north`, `grow_staple`) are invented examples
used as data; every number is a placeholder (note N-027).
"""
from lhs import axis_key, perm, value

SALT = 0x243F6A8885A308D3


def grid(sizes):
    """Points of a full grid, the last factor fastest: a list of tuples of level indices."""
    out = [()]
    for n in sizes:
        out = [p + (i,) for p in out for i in range(n)]
    return out


def oat(sizes):
    """The baseline (all level 0), then for each factor each level above 0 with the others at 0."""
    base = tuple(0 for _ in sizes)
    pts = [base]
    for f, n in enumerate(sizes):
        for l in range(1, n):
            p = list(base)
            p[f] = l
            pts.append(tuple(p))
    return pts


def overlay(changes):
    """One document per change: the table of the path and one key (config 6.2)."""
    docs = []
    for path, v in changes:
        *table, key = path.split(".")
        docs.append("[%s]\n%s = %d\n" % (".".join(table), key, v))
    return docs


def main():
    print("== A. a grid: cost_floor in {7 000, 8 000, 9 000, 10 000} x belief_step in {100, 200, 400}")
    levels = ([7000, 8000, 9000, 10000], [100, 200, 400])
    pts = grid([len(l) for l in levels])
    print("  %d points" % len(pts))
    for idx, p in enumerate(pts):
        print("   point %2d  cost_floor=%5d  belief_step=%3d" % (idx, levels[0][p[0]], levels[1][p[1]]))

    print()
    print("== B. one at a time: the same factors, level 0 the baseline")
    pts = oat([4, 3])
    print("  %d points (1 + 3 + 2)" % len(pts))
    for idx, p in enumerate(pts):
        print("   point %d  cost_floor=%5d  belief_step=%3d" % (idx, levels[0][p[0]], levels[1][p[1]]))

    print()
    print("== C. a Latin hypercube, N = 8, two ranged factors: cost_floor in [6 000, 10 000], belief_step in [100, 400]")
    n = 8
    keys = [axis_key(SALT, j, n) for j in range(2)]
    ranges = [(6000, 10000), (100, 400)]
    cells_by_axis = [[perm(i, n, keys[j]) for i in range(n)] for j in range(2)]
    for i in range(n):
        cs = [cells_by_axis[j][i] for j in range(2)]
        vs = [value(cs[j], n, *ranges[j]) for j in range(2)]
        print("   point %d  cells (%d, %d)  cost_floor=%5d  belief_step=%3d" % (i, cs[0], cs[1], vs[0], vs[1]))
    for j in range(2):
        assert sorted(cells_by_axis[j]) == list(range(n))
    print("  each axis uses every cell once")

    print()
    print("== D. the overlay documents of point 5 of example A (cost_floor=8 000, belief_step=200 is point 4; point 5 is 8 000 and 400)")
    p = grid([4, 3])[5]
    for d in overlay([("world.cost_floor", levels[0][p[0]]), ("world.belief_step", levels[1][p[1]])]):
        print("   " + d.replace("\n", "\n   ").rstrip())

    print()
    print("== E. seeds: Seed0 = 1, 4 replicas, 3 points")
    print("   Same     :", [[1 + r for r in range(4)] for _ in range(3)])
    print("   PerPoint :", [[1 + p * 4 + r for r in range(4)] for p in range(3)])

    print()
    print("== F. a branch plan: one burn-in of 365 ticks, then three branches of 60 ticks, seed 1")
    print("   burn-in  ticks 0..364, seed 1, checkpoint c at tick 365")
    for b, desc in enumerate(["no call", "SetYieldShocks north/grow_staple 5000", "SetYieldShocks 5000, cleared at tick 425"]):
        print("   branch %d  restore c, WithReseed(salt %d), %s" % (b, 7, desc))
    print("   a paired branch keeps the same salt in every branch; a Monte Carlo of branches gives each its own")

    print()
    print("== G. the size of a design")
    print("   grid 4 x 3 x 2 x 5 levels = %d points; with 10 replicas %d runs" % (4 * 3 * 2 * 5, 4 * 3 * 2 * 5 * 10))
    print("   one at a time over those four factors = %d points" % (1 + 3 + 2 + 1 + 4))


if __name__ == "__main__":
    main()
