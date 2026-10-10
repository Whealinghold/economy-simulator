import statistics, random
from a_labour import run
import a_labour
# buffer loop sweep: modify tgt via monkeypatch by re-implementing run with parameters
def runb(e, tgt_frac, STEP, G, lo=5000, hi=15000, seed=1, steps=1500, N=1000, A=10, K=1500000, P0=1000, PMIN=100, PMAX=10000, DMAX=365):
    rng = random.Random(seed)
    S0 = N*A*(-(-K//(A*P0)))
    p = 1100; stock = 0; out = []
    for t in range(steps):
        d = max(1, min(DMAX, -(-K//(A*p))))
        S = N*A*d
        Qd = S0*(P0/p)**e
        stock += S - Qd
        tgt = tgt_frac*S0
        # stock ratio of the SELLERS' holdings: positive stock only; negative stock = backlog -> ratio 0
        ratio = max(0, stock)/tgt*10000 if tgt else 10000
        sig = 1 if ratio < lo else (-1 if ratio > hi else 0)
        if rng.randrange(10000) < G and sig:
            p = max(PMIN, min(PMAX, p + sig*max(1, p*STEP//10000)))
        out.append(p)
    return out
print("e=1.5 with a stock-ratio-only rule (an integrator): rel std of price, last 500 ticks, median of 10 seeds")
print("tgt_frac STEP G -> relstd")
for tgt in (0.1, 0.5, 1.0, 3.0):
    for STEP in (50, 200):
        for G in (2000, 5000, 10000):
            sds = []
            for s in range(10):
                ps = runb(1.5, tgt, STEP, G, seed=s)[-500:]
                sds.append(statistics.pstdev(ps)/statistics.mean(ps))
            sds.sort()
            print(f"{tgt:4.1f} {STEP:4d} {G:5d} {sds[5]:.3f}")
