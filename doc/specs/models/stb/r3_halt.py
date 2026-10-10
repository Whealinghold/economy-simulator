from b_bel import *
def m(name, ticks=800, seeds=8, burn=400, **kw):
    rs = [stats(simulate(seed=s, ticks=ticks, **kw), burn) for s in range(seeds)]
    f = lambda k: sorted(r[k] for r in rs)
    n = seeds
    print(f"{name:44s} mean med {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd {f('relsd')[n//2]:.3f} short/tick {f('short')[n//2]:.2f}")
print("inelastic flows, Q=D=10, 20+20 agents, seller target = OutputCoverTicks(1) ticks of output (AGT 7.3), halt at ProduceHaltRatio of that target")
for stop in (None, 30000, 20000, 15000, 10000):
    m(f"cover_s=1 prod_stop={stop}", cover_s=1, prod_stop=stop)
print("same, cover_s=2 (the earlier run)")
for stop in (None, 30000): m(f"cover_s=2 prod_stop={stop}", cover_s=2, prod_stop=stop)
