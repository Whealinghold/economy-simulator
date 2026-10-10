from b_bel import *
import statistics
def go(name, P=None, seeds=8, **kw):
    st = []
    for s in range(seeds):
        r = simulate(seed=s, ticks=800, Q=100, D=100, ed=0.5, es=0.5, P=P, **kw)
        st.append(stats(r, 400))
    f = lambda k: sorted(x[k] for x in st)
    n = len(st)
    print(f"{name:44s} mean med {f('mean')[n//2]:6.0f} [{f('mean')[0]:.0f},{f('mean')[-1]:.0f}] relsd {f('relsd')[n//2]:.3f} short {f('short')[n//2]:.1f}")
SH = {400: (15000, 10000), 500: (10000, 10000)}
def shock(name, P=None, seeds=6, **kw):
    pk, rec, base, sdv = [], [], [], []
    for s in range(seeds):
        r = simulate(seed=s, ticks=900, Q=100, D=100, ed=0.5, es=0.5, P=P, shocks=SH, **kw)
        p = r['p']; b = statistics.mean(p[300:400]); base.append(b)
        pk.append(max(p[400:520]) / b)
        t = next((i for i in range(500, 900) if all(abs(x - b) <= 0.1*b for x in p[i:i+10])), None)
        rec.append(None if t is None else t - 500)
        sdv.append(statistics.pstdev(p[300:400]) / b)
    rk = [x for x in rec if x is not None]
    print(f"   shock {name:30s} base {statistics.median(base):6.0f} peak/base {statistics.median(pk):.2f} recovered {len(rk)}/{seeds} median ticks {statistics.median(rk) if rk else '-'} relsd(pre) {statistics.median(sdv):.3f}")
OFF = dict(BeliefSignificance=5000, BeliefMaxAge=-1)
if __name__ == "__main__":
  for name, P in (("as written", {}), ("limit", dict(EV="limit")), ("consensus", dict(EV="consensus")), ("stock only", OFF)):
    go(name, P); shock(name, P)
