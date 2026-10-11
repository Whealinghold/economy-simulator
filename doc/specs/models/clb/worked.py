"""Every number of the worked examples of spec-calibration.md 19, asserted.

Imports the other models of this folder. Invented settings, placeholder numbers
(N-027). Run in Docker only (see grid.py).
"""
from fractions import Fraction as F

import choose
import cost
import grid
import stop

rdiv = grid.rdiv

# 19.1 grids and neighbours ------------------------------------------------------
lv = grid.geo_levels(7000, 1, 1_000_000)
assert lv[lv.index(7000) - 2: lv.index(7000) + 3] == [1750, 3500, 7000, 14000, 28000]
assert grid.neighbours(lv, 7000)[:2] == (3500, 14000)

lv = grid.geo_levels(200, 1, 2500)
floor_ok = lambda v: -(-10000 // v) <= 96            # I01 at a cheapest price of 96
assert [(v, -(-10000 // v)) for v in (50, 100, 200, 400)] == [(50, 200), (100, 100), (200, 50), (400, 25)]
low, up, edges = grid.neighbours(lv, 200, floor_ok)
assert (low, up, edges) == (None, 400, [("below", "rule")])

lv = grid.geo_levels(50, 0, 10000)
assert grid.neighbours(lv, 50)[:2] == (25, 100) and rdiv(50, 4) == 12   # 12.5 to the even 12

lv = grid.lin_levels(4000, 1000, 0, 250 * 10000 // 365)                  # I05: 6 849
assert 250 * 10000 // 365 == 6849 and grid.neighbours(lv, 4000)[:2] == (3000, 5000)

lv = grid.lin_levels(3, 1, 1, 1000)
assert grid.neighbours(lv, 3, lambda v: v > 2) == (None, 4, [("below", "rule")])   # I07, EnterTicks 2

# 19.2 a screen -------------------------------------------------------------------
def elasticity(x0, xl, y0, yl):
    num, den = 10000 * (yl - y0) * x0, y0 * (xl - x0)
    if den < 0:  # rdiv takes a positive divisor
        num, den = -num, -den
    return rdiv(num, den)

assert elasticity(7000, 14000, 600, 1500) == 15000
assert elasticity(7000, 3500, 600, 300) == 10000
assert elasticity(5000, 6000, 470, 560) == 9574 and elasticity(5000, 4000, 470, 380) == 9574
assert elasticity(5000, 6000, 60, 55) == -4167 and elasticity(5000, 4000, 60, 70) == -8333
rng_cf, rng_es = 520 - 410, 560 - 380
assert (rng_cf, rng_es) == (110, 180)
assert (rdiv(10000 * rng_es, rng_cf + rng_es), rdiv(10000 * rng_cf, rng_cf + rng_es)) == (6207, 3793)
assert elasticity(30000, 60000, 800, 700) == -1250     # a finding: expected +, measured -

# 19.3 choose ---------------------------------------------------------------------
assert choose.need("B", 10) == 9 and choose.need("B", 50) == 45 and choose.need("B", 25) == 23
crit = [("x_o", "X"), ("r5", "D"), ("p02", "B"), ("p04", "B")]
pts = [dict(index=i, levels=lv_, passes=dict(x_o=10, r5=10, p02=p2, p04=p4)) for i, lv_, p2, p4 in [
    (0, {"cost_floor": 0, "evidence_share": 0}, 7, 10), (1, {"cost_floor": -1, "evidence_share": 0}, 9, 10),
    (2, {"cost_floor": 1, "evidence_share": 0}, 4, 10), (3, {"cost_floor": 0, "evidence_share": -1}, 8, 10),
    (4, {"cost_floor": 0, "evidence_share": 1}, 6, 9), (5, {"cost_floor": -1, "evidence_share": 1}, 10, 10),
    (6, {"cost_floor": -1, "evidence_share": -1}, 10, 10)]]
assert [p["index"] for p in choose.feasible(pts, crit, 10)] == [1, 5, 6]
assert [choose.distance(p) for p in pts] == [0, 1, 1, 1, 1, 2, 2]
assert choose.choose(pts, crit, 10)["index"] == 1

# 19.4 the seed arithmetic ----------------------------------------------------------
a = stop.p_at_least(10, 9, F(85, 100))
c = stop.p_at_least(50, 45, F(85, 100))
assert stop.bp(a) == 5443 and stop.bp(c) == 2194
assert stop.bp(1 - (1 - a) ** 16) == 10000 and stop.bp(1 - (1 - a) ** 64) == 10000
assert stop.bp(stop.p_at_least(10, 9, F(99, 100))) == 9957
assert stop.bp(stop.p_at_least(10, 9, F(95, 100))) == 9139 and stop.bp(stop.p_at_least(50, 45, F(95, 100))) == 9622
assert stop.bp(stop.p_at_least(10, 9, F(99, 100)) ** 63) == 7639   # 63 conjuncts (3 checks, 7 packs, 3 points) at p = 0.99
assert stop.bp(stop.p_at_least(10, 9, F(95, 100)) ** 63) == 34
assert (1 - a) ** 16 < F(1, 100000)                      # not certainty: 9 999.97 bp

# 19.6 an overlay -------------------------------------------------------------------
def overlay(path_value):
    out = []
    for path, value in path_value:
        segs = path.split(".")
        out.append("[" + ".".join(segs[:-1]) + "]\n" + segs[-1] + " = " + str(value) + "\n")
    return out

assert overlay([("world.cost_floor", 3500)]) == ["[world]\ncost_floor = 3500\n"]

# 19.7 the alias test -----------------------------------------------------------------
assert choose.collinear([300, -450, 0], [600, -900, 0], 500)
assert choose.cos2_bp([300, -450, 0], [300, 100, -200]) == 495

# 19.8 the cost count -----------------------------------------------------------------
s = {x["name"]: x for x in cost.STAGES}["C4a beliefs"]
assert (s["screen"], s["explore"], s["search_runs"], s["confirm_runs"]) == (33, 60, 930, 200)
assert cost.run_ms("one", 1095) // 1000 == 4 and round(cost.hours(s["ms"]), 1) == 1.5
tot_runs = sum(x["search_runs"] + x["confirm_runs"] for x in cost.STAGES)
assert tot_runs == 7012 and round(cost.hours(sum(x["ms"] for x in cost.STAGES))) == 39
packs7 = ("one", "second", "two", "three", "five", "famine", "one-demand")
assert (1 + 2 * 49) * len(packs7) * 20 == 13860
assert round(cost.hours((1 + 2 * 49) * 20 * sum(cost.run_ms(p, 1095) for p in packs7)), 0) == 57
print("all worked examples hold")
