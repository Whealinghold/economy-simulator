from a_labour import run
import statistics
print("== work floor DMIN (days), e=0.5, +10% perturbation, 30 seeds; D0=150")
for dmin in (1, 100, 120, 140):
    ends = sorted(run(0.5, seed=s, DMIN=dmin)[-1][0] for s in range(30))
    print(f"DMIN={dmin:4d} end price median {ends[15]:6d} min {ends[0]} max {ends[-1]}")
print("== work floor, e=0.5, -10% perturbation")
for dmin in (1, 100, 140):
    ends = sorted(run(0.5, seed=s, DMIN=dmin, perturb=0.9)[-1][0] for s in range(30))
    print(f"DMIN={dmin:4d} end price median {ends[15]:6d} min {ends[0]} max {ends[-1]}")
print("== e=1.5 buffer: price std over last 200 ticks, by seed (relative to no buffer)")
for kw in ({}, {"inv_buffer": True}):
    sds = []
    for s in range(30):
        ps = [x[0] for x in run(1.5, seed=s, steps=800, **kw)[-300:]]
        sds.append(statistics.pstdev(ps) / statistics.mean(ps))
    sds.sort()
    print(kw, f"rel std median {sds[15]:.3f} max {sds[-1]:.3f}")
print("== basin: e=0.5 start price vs end price (DMIN=1), shows the unstable interior point at 1000")
for p0 in (900, 980, 995, 1000, 1005, 1020, 1100):
    r = [run(0.5, seed=s, perturb=p0/1000)[-1][0] for s in range(10)]
    print(p0, sorted(r)[5])
print("== staple demand elasticity e at which a 2% bounded step recovers: half-life vs e (e>1), 1 seed median of 15")
for e in (1.05, 1.2, 1.5, 2.0, 3.0):
    hl = []
    for s in range(15):
        ser = run(e, seed=s, steps=600, perturb=1.2)
        end = ser[-1][0]
        t = next(i for i, x in enumerate(ser) if abs(x[0]-end) <= 0.1*(1200-end) + 0) if True else 0
        hl.append(t)
    hl.sort(); print(f"e={e}: ticks to close 90% of a +20% shock, median {hl[7]}")
