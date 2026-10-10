from b_bel import *
import statistics
SH = {400: (15000, 10000), 500: (10000, 10000)}
OFF = dict(BeliefSignificance=5000, BeliefMaxAge=-1)   # evidence off (stock only)
def shock(name, P=None, seeds=6, NS=20, NB=20, w=1, **kw):
    pk, rec, base, sdv = [], [], [], []
    for s in range(seeds):
        r = simulate(seed=s, ticks=900, Q=100, D=100, ed=0.5, es=0.5, P=P, shocks=SH, NS=NS, NB=NB, w=w, **kw)
        p = r['p']; b = statistics.mean(p[300:400]); base.append(b)
        pk.append(max(p[400:520]) / b)
        t = next((i for i in range(500, 900) if all(abs(x - b) <= 0.1*b for x in p[i:i+10])), None)
        rec.append(None if t is None else t - 500)
        sdv.append(statistics.pstdev(p[300:400]) / b)
    rk = [x for x in rec if x is not None]
    print(f"{name:40s} base {statistics.median(base):6.0f} peak/base {statistics.median(pk):.2f} recovered {len(rk)}/{seeds} median ticks {statistics.median(rk) if rk else '-'} relsd(pre) {statistics.median(sdv):.3f}")
print("== demand +50% for 100 ticks (ticks 400-500), ed=es=0.5, 20+20 agents")
shock("as written")
shock("stock only (evidence off)", OFF)
print("== sweep BeliefAdjustProb (as written)")
for g in (1000, 2500, 5000, 7500, 10000): shock(f"AdjustProb={g}", dict(BeliefAdjustProb=g))
print("== sweep BeliefStep (as written)")
for st in (50, 100, 200, 500, 1000): shock(f"BeliefStep={st}", dict(BeliefStep=st))
print("== cohorts: 20 households per side-group fixed; weight w, agents = 20/w x2 groups (as written)")
for w, n in ((1, 20), (2, 10), (4, 5), (10, 2)): shock(f"w={w} agents/side={n}", NS=n, NB=n, w=w)
