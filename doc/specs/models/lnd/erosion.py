"""Model of the erosion of private titles into the pool (spec-land.md 6.4, register LND-A4):
a holder that is removed hands its titles to the estate, nothing hands them back, and every birth
is landless. A model of the rule and not of the library. Run: python erosion.py
"""
import math
import random


def pool_share(holders=500, annual_death_bp=150, tpy=365, years=(10, 25, 46, 70, 100), seeds=40):
    """Mean titles still held by their original households and moved to the pool, by year.
    Each holder dies at a per-tick hazard p = annual_death_bp / (10 000 * tpy): the geometric
    time of death is drawn once per holder."""
    p = annual_death_bp / (10000 * tpy)
    out = {y: 0 for y in years}
    for s in range(seeds):
        rng = random.Random(1000 + s)
        deaths = [int(math.log(1 - rng.random()) / math.log(1 - p)) for _ in range(holders)]
        for y in years:
            out[y] += sum(1 for d in deaths if d >= y * tpy)
    return {y: out[y] / seeds for y in years}


if __name__ == '__main__':
    print('500 titles, a death rate of 1.5% a year, each death replaced by a landless birth, no re-grant; mean of 40 seeds')
    held = pool_share()
    for y, h in held.items():
        closed = 500 * math.exp(-0.015 * y)
        print('year %3d: held by households %6.1f, in the pool %6.1f (closed form held %6.1f)' % (y, h, 500 - h, closed))
    print('half of the titles are in the pool after %.1f years' % (math.log(2) / 0.015))
