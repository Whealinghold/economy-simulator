from b_bel import *
def go(name, P=None, seeds=8, **kw):
    st = []
    for s in range(seeds):
        r = simulate(seed=s, ticks=800, Q=100, D=100, ed=0.5, es=0.5, P=P, **kw)
        st.append(stats(r, 400))
    f = lambda k: sorted(x[k] for x in st)
    n = len(st)
    print(f"{name:44s} mean med {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd {f('relsd')[n//2]:.3f} short {f('short')[n//2]:.1f}")
go("as written (centre)")
go("EV=limit", dict(EV='limit'))
go("EV=limit PART=none", dict(EV='limit', PART='none'))
go("EV=limit PART=asfilled", dict(EV='limit', PART='asfilled'))
go("EV=centre PART=none", dict(PART='none'))
go("EV=centre PART=asfilled", dict(PART='asfilled'))
