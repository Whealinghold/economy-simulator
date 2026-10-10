from b_bel import *
def m(name, ticks=800, seeds=8, burn=400, **kw):
    rs = [stats(simulate(seed=s, ticks=ticks, **kw), burn) for s in range(seeds)]
    f = lambda k: sorted(r[k] for r in rs)
    print(f"{name:40s} mean med {f('mean')[seeds//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd med {f('relsd')[seeds//2]:.3f} notrade {f('notrade')[seeds//2]:.2f} short/tick {f('short')[seeds//2]:.2f}")
print("== with a production stop when seller stock >= 150% of target (an inventory cap), balanced flows")
for ps_ in (None, 30000, 15000, 10000):
    m(f"prod_stop={ps_}", prod_stop=ps_)
print("== same, cost 700")
m("prod_stop=15000, cost 700", prod_stop=15000, seller_cost=700)
