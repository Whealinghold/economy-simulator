from b_bel import *
import statistics
SH = {400: (15000, 10000), 500: (10000, 10000)}
def run(name, P=None, seeds=8):
    st = []; pk = []; rec = []
    for s in range(seeds):
        r = simulate(seed=s, ticks=900, Q=100, D=100, ed=0.5, es=0.5, P=P, shocks=SH)
        st.append(stats(r, 400)); p = r['p']; b = statistics.mean(p[300:400]); pk.append(max(p[400:520])/b)
        t = next((i for i in range(500, 900) if all(abs(x-b) <= 0.1*b for x in p[i:i+10])), None)
        rec.append(t-500 if t is not None else None)
    f = lambda k: sorted(x[k] for x in st); n = seeds
    rk = [x for x in rec if x is not None]
    print(f"{name:42s} mean {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] unmet {f('short')[n//2]:6.1f} relsd {f('relsd')[n//2]:.3f} peak {statistics.median(pk):.2f} return {statistics.median(rk) if rk else '-'} ({len(rk)}/{seeds})")
print("== elastic flows ed=es=0.5, 20+20 agents, shock +50% consumption ticks 400-500; stats ticks 400-900")
run("old rule (centre, step delta)")
run("revised, EvShare 5000", dict(REV=True, EvShare=5000))
run("revised, EvShare 10000", dict(REV=True, EvShare=10000))
run("revised, EvShare 2500", dict(REV=True, EvShare=2500))
run("evidence off", dict(BeliefSignificance=5000, BeliefMaxAge=-1))
print("== inelastic flows Q=D=10, revised rule share 5000")
for cs, stop in ((1, None), (1, 30000)):
    rs = [stats(simulate(seed=s, ticks=800, cover_s=cs, prod_stop=stop, P=dict(REV=True, EvShare=5000)), 400) for s in range(8)]
    f = lambda k: sorted(x[k] for x in rs)
    print(f"cover_s={cs} halt={stop}: mean {f('mean')[4]:.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd {f('relsd')[4]:.3f} unmet {f('short')[4]:.1f}")
print("== N-039 freeze, unit cost 1100, buyers capped 1000, shelf life 3, revised rule")
for cf in (10000, 9000, 7000, 5000):
    tr, sp, p = [], [], []
    for s in range(6):
        r = simulate(seed=s, ticks=300, Q=100, D=100, seller_cost=1100, wtp=1000, shelf=3, P=dict(CostFloor=cf, REV=True, EvShare=5000))
        tr.append(sum(1 for v in r['vol'][100:] if v > 0)/200); sp.append(r['spoiled']/r['produced']); p.append(statistics.mean(r['p'][100:]))
    print(f"CostFloor={cf:5d} floor {1100*cf//10000:5d} trade ticks {statistics.median(tr):.2f} spoiled {statistics.median(sp):.2f} price {statistics.median(p):.0f}")
print("== cohorts (revised rule)")
for w, n in ((1, 20), (4, 5), (10, 2)):
    sds = []
    for s in range(6):
        r = simulate(seed=s, ticks=900, Q=100, D=100, ed=0.5, es=0.5, P=dict(REV=True, EvShare=5000), NS=n, NB=n, w=w)
        sds.append(stats(r, 400)['relsd'])
    print(f"w={w} agents/side={n} relsd {statistics.median(sds):.3f}")
