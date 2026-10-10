"""Model (a): target-income labour loop of spec-welfare-and-wages 5, integer only.
N identical self-employed households (weight 1). Each tick a household works
D days of its period; income per working day = A*p (A units/day, price p).
Target income per period K (money). Rule: D = clamp(ceil(K/(A*p)), 1, DMAX).
Supply per period S = N*A*D. Demand Q = QD0 * (P0/p)^e (computed with integer
fixed point via float only for the DEMAND CURVE of the model - this is a model of the rule,
not code; the spec rule itself is integer).
Price rule: spec BEL-style: each tick, with prob G, move p by +/- STEP bp toward
excess demand sign, clamp to [PMIN,PMAX]."""
import random, math, sys
def run(e, N=1000, A=10, K=1500000, P0=1000, seed=1, steps=400, p_init=None, G=0.5, STEP=200,
        PMIN=100, PMAX=10000, DMAX=365, DMIN=1, perturb=1.10, inv_buffer=False, spread=0):
    rng = random.Random(seed)
    # equilibrium: D0 = K/(A*P0) days. S0 = N*A*D0 ; choose QD0 = S0 so P0 is the equilibrium
    D0 = K / (A * P0)
    S0 = N * A * (-(-K // (A * P0)))
    Ks = [K * (10000 + rng.randint(-spread, spread)) // 10000 for _ in range(N)] if spread else [K] * N
    p = int((p_init or P0) * perturb)
    series = []
    stock = 0
    for t in range(steps):
        D = 0
        S = 0
        for k in Ks:
            d = -(-k // (A * p))          # ceil(K/(A p))
            d = max(DMIN, min(DMAX, d))
            S += A * d
        Qd = S0 * (P0 / p) ** e
        series.append((p, S, Qd))
        ex = Qd - S
        if inv_buffer:
            stock += S - Qd
            # stock ratio rule: target = 10% of S0
            tgt = 0.1 * S0
            ratio = stock / tgt if tgt else 1
            sig = 1 if ratio < 0.5 else (-1 if ratio > 1.5 else 0)
        else:
            sig = 1 if ex > 0 else (-1 if ex < 0 else 0)
        if rng.randrange(10000) < G * 10000 and sig:
            delta = max(1, p * STEP // 10000)
            p = max(PMIN, min(PMAX, p + sig * delta))
    return series
def summarise(label, series):
    ps = [s[0] for s in series]
    tail = ps[-100:]
    print(f"{label:48s} p_end={ps[-1]:6d} tail_min={min(tail):6d} tail_max={max(tail):6d} max={max(ps):6d} min={min(ps):6d}")
if __name__ == "__main__":
    print("== continuous-time check: sign of d(excess demand)/dp at equilibrium = (1-e) Q/p")
    for e in (0.3, 0.5, 0.9, 1.0, 1.1, 1.5, 2.0):
        P0 = 1000; h = 1
        S = lambda p: 1/p
        Qd = lambda p: p ** -e
        # normalised: equal at p=1
        d = ((1.001 ** -e - 1.001 ** -1) - (0.999 ** -e - 0.999 ** -1)) / 0.002
        print(f"e={e:4.1f} numerical slope {d:+.4f} analytic {(1-e):+.4f}  -> {'stable' if d<0 else 'unstable' if d>0 else 'neutral'}")
    print("== discrete simulation (N=1000, bounded steps 2%, gate 1/2), +10% price perturbation, 30 seeds")
    for e in (0.5, 1.5):
        for label, kw in (("no buffer", {}), ("inventory buffer", {"inv_buffer": True}), ("spread 3000", {"spread": 3000}),
                          ("perturb -10%", {"perturb": 0.90})):
            ends = []
            for seed in range(30):
                s = run(e, seed=seed, **kw)
                ends.append(s[-1][0])
            ends.sort()
            print(f"e={e} {label:18s} end-price median {ends[15]:6d} min {ends[0]:6d} max {ends[-1]:6d}  frac at PMAX {sum(1 for x in ends if x>=10000)/30:.2f} at PMIN {sum(1 for x in ends if x<=100)/30:.2f}")
