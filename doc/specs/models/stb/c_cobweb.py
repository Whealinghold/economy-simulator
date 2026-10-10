"""Model (c): (1) production-delay cobweb, (2) occupation-choice cobweb.  Stylised, floats allowed: this is analysis of
the loop structure, not of the integer rule."""
import math, statistics, random
def cobweb(es, ed, d, mode, lam=1.0, G=0.5, step=0.02, slew=None, steps=600, kick=1.1, seed=0, pmin=0.1, pmax=10.0):
    """demand p = Q^(-1/ed) (normalised, equilibrium at p=1,Q=1). starts_t = (pe_t)^es, arrive at t+d.
    mode: 'naive' pe=p ; 'nerlove' pe += lam*(p-pe) ; 'bel' pe moves by at most step*pe with prob G toward p.
    slew: max relative change of starts per tick (entry limit)."""
    rng = random.Random(seed)
    pe = kick
    starts = [1.0] * (d + 1)
    prev_start = 1.0
    ps = []
    # initial price perturbation: price is p0=kick at t=0 before anything
    p = kick
    pend = [1.0] * d   # output arriving in the next d ticks
    for t in range(steps):
        if mode == 'naive': pe = p
        elif mode == 'nerlove': pe += lam * (p - pe)
        else:
            if rng.random() < G:
                dlt = max(-step * pe, min(step * pe, p - pe)); pe += dlt
        pe = max(pmin, min(pmax, pe))
        s = pe ** es
        if slew is not None:
            s = max(prev_start * (1 - slew), min(prev_start * (1 + slew), s))
        prev_start = s
        pend.append(s)
        q = pend.pop(0)                 # arrives now (started d ticks ago)
        p = max(pmin, min(pmax, q ** (-1.0 / ed)))
        ps.append(p)
    return ps
def amp(ps):
    tail = ps[-200:]
    return max(tail) / min(tail)
def occupation(e, T, r, h, steps=3000, x0=0.55, pmin=0.0):
    """share x of N in staple role A; B pays fixed 1. A's income per head = p = (x/x*)^(-1/e) with x*=0.5 (so equal at x=0.5).
    Decisions compare a T-tick moving average of past incomes (the welfare window). Fraction r re-decides per tick; hysteresis h."""
    xs = [x0] * 1
    hist = []
    pa_hist = []
    for t in range(steps):
        x = xs[-1]
        pa = (x / 0.5) ** (-1.0 / e)
        pa_hist.append(pa)
        w = pa_hist[-T:]
        avg = sum(w) / len(w)
        if avg > 1 + h: x = x + r * (1 - x)
        elif avg < 1 - h: x = x - r * x
        xs.append(x)
    return xs
if __name__ == "__main__":
    print("== (1) production-delay cobweb, demand elasticity ed=0.5; amplitude = max/min price over last 200 of 600 ticks after a +10% kick")
    print("naive expectations: es<ed stable, es>ed unstable (bounded only by price bounds 0.1..10)")
    for d in (1, 5, 20):
        row = []
        for es in (0.25, 0.5, 0.75, 1.0, 2.0):
            row.append(f"es={es}:{amp(cobweb(es, 0.5, d, 'naive')):7.2f}")
        print(f"delay d={d:2d} naive    ", "  ".join(row))
    print("Nerlove adaptive expectations (lam), es=2.0 (unstable under naive)")
    for d in (1, 5, 20):
        row = []
        for lam in (0.5, 0.2, 0.05, 0.02):
            row.append(f"lam={lam}:{amp(cobweb(2.0, 0.5, d, 'nerlove', lam=lam)):7.2f}")
        print(f"delay d={d:2d} nerlove  ", "  ".join(row))
    print("BEL-like bounded asynchronous expectations (step 2%, gate 1/2), es=2.0; median over 10 seeds")
    for d in (1, 5, 20):
        row = []
        for step in (0.05, 0.02, 0.005):
            a = sorted(amp(cobweb(2.0, 0.5, d, 'bel', step=step, seed=s)) for s in range(10))[5]
            row.append(f"step={step}:{a:7.2f}")
        print(f"delay d={d:2d} bel      ", "  ".join(row))
    print("entry limit (slew of starts per tick) on top of naive expectations, es=2.0")
    for d in (1, 5, 20):
        row = []
        for sl in (None, 0.2, 0.05, 0.01, 0.002):
            row.append(f"slew={sl}:{amp(cobweb(2.0, 0.5, d, 'naive', slew=sl)):7.2f}")
        print(f"delay d={d:2d} naive+slew", "  ".join(row))
    print("combined: bel step 2% + slew 1%, es=2.0")
    for d in (1, 5, 20):
        a = sorted(amp(cobweb(2.0, 0.5, d, 'bel', step=0.02, slew=0.01, seed=s)) for s in range(10))[5]
        print(f"d={d}: {a:.2f}")
    print()
    print("== (2) occupation-choice cobweb, mean-field. amplitude of share x (max-min over last 1000 of 3000 ticks)")
    print("  e  T   r    h   -> amp(x)")
    for e in (0.5, 2.0):
        for T in (1, 30):
            for r in (0.2, 0.05, 0.01):
                for h in (0.0, 0.1):
                    xs = occupation(e, T, r, h)
                    tail = xs[-1000:]
                    print(f" {e:3.1f} {T:3d} {r:5.2f} {h:4.2f}  {max(tail)-min(tail):.3f}  (x in [{min(tail):.3f},{max(tail):.3f}])")
