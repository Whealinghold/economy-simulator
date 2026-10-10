from b_bel import *
def m(name, ticks=600, seeds=8, **kw):
    rs = [stats(simulate(seed=s, ticks=ticks, **kw), 300) for s in range(seeds)]
    f = lambda k: sorted(r[k] for r in rs)
    print(f"{name:34s} mean price med {f('mean')[seeds//2]:7.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd med {f('relsd')[seeds//2]:.3f} notrade {f('notrade')[seeds//2]:.2f} short/tick {f('short')[seeds//2]:.2f}")
print("== balanced fixed flows (inelastic), 20+20 agents, weight 1, 600 ticks, stats over ticks 300-600")
m("spec rule as written (DIR off)")
m("direction-restricted evidence", P=dict(DIR=True))
m("spec rule, seller cost 700", seller_cost=700)
m("spec rule, cost 700, DIR", seller_cost=700, P=dict(DIR=True))
m("spec rule, cost 1000", seller_cost=1000)
