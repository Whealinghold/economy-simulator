from b_bel import *
import statistics
def many(name, seeds=10, **kw):
    out = []
    for s in range(seeds):
        r = simulate(seed=s, **kw)
        out.append(r)
    return out
def rep(name, rs, burn):
    st = [stats(r, burn) for r in rs]
    f = lambda k: sorted(x[k] for x in st)
    n = len(st)
    print(f"{name:44s} mean med {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd med {f('relsd')[n//2]:.3f} notrade {f('notrade')[n//2]:.2f} short {f('short')[n//2]:.2f}")
base = dict(ticks=800, Q=100, D=100)
print("== elastic flows, 20+20 agents w=1, ref 1000; ed = demand elasticity, es = supply elasticity; stats over ticks 400-800")
for ed, es in ((0, 0), (0.3, 0.3), (0.5, 0.5), (1.0, 1.0), (0.5, 1.5)):
    rep(f"ed={ed} es={es}", many("x", **base, ed=ed, es=es), 400)
