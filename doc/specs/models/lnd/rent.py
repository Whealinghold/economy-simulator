"""Model of the rent step (spec-land.md 5.5, 17): a posted rent that moves by a bounded step with
the units unserved, against tenants that respond to the rent with a delay.

Integers only. A model of the RULE, not of the library. Run: python rent.py
"""

def run(step, delay, supply=20, base=40, floor_bp=5000, ceil_bp=20000, ticks=400, n=60, lo=10, hi=100):
    """n tenants with a value per unit spread evenly on [lo, hi]; each wants 1 unit when its value is
    at least the rent it saw `delay` ticks ago. The let supply is `supply` units a tick."""
    vals = [lo + (hi - lo) * j // (n - 1) for j in range(n)]
    floor = base * floor_bp // 10000
    ceil = base * ceil_bp // 10000
    st = max(1, base * step // 10000) if step else 0
    posted = base
    hist = [base] * (delay + 1)
    tail = []
    for t in range(ticks):
        seen = hist[-(delay + 1)]
        demand = sum(1 for v in vals if v >= seen)
        served = min(demand, supply)
        unserved = demand - served
        idle = supply - served
        if st:
            if unserved > 0:
                posted = min(ceil, posted + st)
            elif idle > 0:
                posted = max(floor, posted - st)
        hist.append(posted)
        if t >= ticks // 2:
            tail.append(posted)
    return min(tail), max(tail)


def clearing_rent(supply=20, n=60, lo=10, hi=100):
    vals = [lo + (hi - lo) * j // (n - 1) for j in range(n)]
    best = None
    for r in range(0, hi + 1):
        if sum(1 for v in vals if v >= r) <= supply:
            best = r
            break
    return best


if __name__ == '__main__':
    print('market-clearing rent for 20 units and 60 tenants valued 10..100:', clearing_rent())
    print('step (bp of base 40), delay (ticks) -> (min, max) of the posted rent over the last half of 400 ticks')
    for step in (0, 100, 250, 500, 1000, 2500):
        row = []
        for delay in (0, 1, 3, 8):
            lo, hi = run(step, delay)
            row.append('%d:(%d,%d)' % (delay, lo, hi))
        print('step %5d  ' % step, '  '.join(row))
    print('with the band narrowed to 8000..12000 bp of base 40, step 1000, delay 3:', run(1000, 3, floor_bp=8000, ceil_bp=12000))
    print('with the supply above the demand at the floor (60 units): the rent falls to the floor', run(500, 1, supply=60))
