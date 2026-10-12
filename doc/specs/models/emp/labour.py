"""Stylised model of the hired-labour loop (spec-employment.md section 17, EMP-A1, EMP-A2).

A prototype of the spec's rules, not of the library. Integer arithmetic for every
rule; the gate of the price rule uses a seeded random.Random (the library draws from
the world's source). Standard library only.

World: N workers (weight 1, no recipe, Hireable) eat `u` units of one staple a tick at
the market price p. E employers (weight 1) each hold H hired places and run a recipe
of Duration 1 that turns an input costing `c` into `a` units of the staple. Each tick:

  1. an employer's bid is the value of one execution net of its input, a*p_prev - c;
     it believes the price is the last published one;
  2. a worker offers its household when its ratio is below its band top, or when the
     floor MinWorkShare binds (the target-income rule of WEL 5, a weight-1 agent);
  3. the round matches offers (reservation 0) to vacancies in a random order, the wage
     is the midpoint (bid + 0) / 2 half to even;
  4. output = a * hired; the staple price moves by 2% with probability 1/2 toward the
     sign of excess demand (the BEL step and gate), clamped to [P0/10, 10*P0];
  5. the worker's record: income = wage if hired else 0, cost = u * p; ratio over a
     window of T ticks, padded with DefaultRatio 10 000.

Optional: the production halt D24 on the employers' stock (they post no vacancy while
stock >= halt_bp/10000 of one tick of demand).
"""
import random, sys

def half_even_div(n, d):
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q

def run(N, E, H, a, c, u, p0, ticks, seed, floor_bp=4000, band_bp=500, target_bp=15000,
        T=30, halt_bp=None, start_bp=11000, bid_bp=10000, lo_div=10, hi_mul=10):
    rng = random.Random(seed)
    p = p0 * start_bp // 10000
    lo, hi = p0 // lo_div, p0 * hi_mul
    rings = [[(u * p0, u * p0)] * T for _ in range(N)]   # padded at ratio 10 000
    work_flag = [[0] * T for _ in range(N)]
    stock = 0
    out = []
    p_prev = p
    for t in range(ticks):
        # 1. the employers' bid and vacancies
        bid = bid_bp * (a * p_prev - c) // 10000
        V = E * H if bid >= 2 else 0
        if halt_bp is not None and stock * 10000 >= halt_bp * N * u:
            V = 0
        # 2. offers
        offers = []
        for i in range(N):
            inc = sum(x for x, _ in rings[i]); cost = sum(y for _, y in rings[i])
            ratio = inc * 10000 // cost if cost else 10000
            worked = sum(work_flag[i])
            if worked * 10000 < floor_bp * T:
                offers.append(i)
            elif ratio < target_bp * (10000 + band_bp) // 10000:
                offers.append(i)
        rng.shuffle(offers)
        hired = offers[:V]
        wage = half_even_div(bid, 2) if bid >= 2 else 0
        hired_set = set(hired)
        # 4. supply, stock, price
        supply = a * len(hired)
        demand = N * u
        stock = max(0, stock + supply - demand)
        if rng.random() < 0.5:
            if demand > supply:
                p = min(hi, p + p * 2 // 100 + 1)
            elif demand < supply:
                p = max(lo, p - p * 2 // 100 - 1)
        # 5. records
        for i in range(N):
            rings[i].pop(0); work_flag[i].pop(0)
            rings[i].append((wage if i in hired_set else 0, u * p))
            work_flag[i].append(1 if i in hired_set else 0)
        p_prev = p
        out.append((p, len(hired), len(offers), wage, stock))
    return out

def summary(out):
    tail = out[-100:]
    ps = [x[0] for x in tail]
    emp = sum(x[1] for x in tail) / len(tail)
    off = sum(x[2] for x in tail) / len(tail)
    return min(ps), max(ps), emp, off, tail[-1][4]

if __name__ == "__main__":
    P0 = 1000
    print("N E H a c u | halt floor | seeds ending (min,max price of last 100 ticks; mean hired; mean offering; last stock)")
    cases = [
        # vacancy-bound, glut: a*V = 6*120 = 720 > demand 600
        dict(N=200, E=20, H=6, a=6, c=1200, u=3),
        # vacancy-bound with the stock halt
        dict(N=200, E=20, H=6, a=6, c=1200, u=3, halt_bp=30000),
        # vacancy-bound, shortage: a*V = 6*80 = 480 < demand 600
        dict(N=200, E=20, H=4, a=6, c=1200, u=3),
        # labour-bound: few workers, many places
        dict(N=100, E=40, H=6, a=6, c=1200, u=3),
        dict(N=100, E=40, H=6, a=6, c=1200, u=3, floor_bp=0),
        # cheap inputs (c small): wage and cost scale together
        dict(N=100, E=40, H=6, a=6, c=100, u=3),
        dict(N=100, E=40, H=6, a=6, c=100, u=3, floor_bp=0),
    ]
    for cs in cases:
        res = []
        for seed in range(1, 11):
            res.append(summary(run(p0=P0, ticks=400, seed=seed, **cs)))
        lo = min(r[0] for r in res); hi = max(r[1] for r in res)
        emp = sum(r[2] for r in res) / len(res)
        off = sum(r[3] for r in res) / len(res)
        print(cs, "->", "price", lo, hi, "hired %.1f offering %.1f" % (emp, off))
