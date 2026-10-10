from b_bel import *
def go(name, P=None, seeds=8):
    st = [stats(simulate(seed=s, ticks=800, Q=100, D=100, ed=0.5, es=0.5, P=P), 400) for s in range(seeds)]
    f = lambda k: sorted(x[k] for x in st); n = seeds
    print(f"{name:34s} mean {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] unmet {f('short')[n//2]:6.1f} relsd {f('relsd')[n//2]:.3f}")
go("old rule (as in 6.2)")
go("revised, EvShare 5000 (default)", dict(REV=True, EvShare=5000))
go("revised, EvShare 10000", dict(REV=True, EvShare=10000))
go("revised, EvShare 2500", dict(REV=True, EvShare=2500))
go("evidence off", dict(BeliefSignificance=5000, BeliefMaxAge=-1))
