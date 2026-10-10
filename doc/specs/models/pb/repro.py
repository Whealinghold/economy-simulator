from b_bel import *
def go(name, P=None, seeds=8, **kw):
    st = []
    for s in range(seeds):
        r = simulate(seed=s, ticks=800, Q=100, D=100, ed=0.5, es=0.5, P=P, **kw)
        st.append(stats(r, 400))
    f = lambda k: sorted(x[k] for x in st)
    n = len(st)
    print(f"{name:44s} mean med {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd {f('relsd')[n//2]:.3f} short {f('short')[n//2]:.1f}")
go("as written")
go("DIR evidence", dict(DIR=True))
go("stock component off", dict(BeliefStockLow=1, BeliefStockHigh=10**6))
go("stock off + DIR", dict(BeliefStockLow=1, BeliefStockHigh=10**6, DIR=True))
go("evidence off (stock only)", dict(BeliefSignificance=5000, BeliefMaxAge=-1))
go("fixed width", dict(BeliefWidthMin=2000, BeliefWidthMax=2000, BeliefWidthInit=2000))
go("sig 0 (always correct filled)", dict(BeliefSignificance=0))
