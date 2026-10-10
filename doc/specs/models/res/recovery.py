"""RES model 4: permanent and recoverable depletion (spec-resources.md 6), and the basin of a logistic stock.

Uses the rules and the stylised economy of coupled.py (same placeholders, same caveats: a model of the RULES, not of the
library, and no number here claims that a default is stable). A shock sets the stock to a share of the capacity at a stated
year, as the adjustment slot of 6.5 would."""
from coupled import *

def shocked(res, year, share_bp):
    done = []
    def f(t, r):
        if t == year * TPY and not done:
            r.s = r.cap * share_bp // BP; r.c = 0; done.append(1)
    return f

def row(label, res, n0, years, shock=None, flat=False):
    ser = simulate(res, n0, T=30, years=years, shock=shock, flat=flat)
    pick = {y: (n, s) for y, n, s, _ in ser}
    cells = []
    for y in (90, 100, 110, 150, 250, 400, years - 1):
        n, s = pick[y]
        cells.append(f"{n/NSTAR:6.2f}/{s/res.cap:5.2f}")
    print(f"   {label:<34} " + " ".join(cells))
    return ser

if __name__ == "__main__":
    print("== 1. a blight at year 100 sets the stock to 5% of the capacity; start at the fixed point; each cell is N/N* / S/C")
    print(f"   {'':<34} " + " ".join(f"{'y'+str(y):>12}" for y in (90, 100, 110, 150, 250, 400, 599)))
    row("refill 0.5/yr", build("refill", 0.5), NSTAR, 600, shocked(None, 100, 500))
    row("logistic 1/yr, no refuge", build("logistic", 1.0), NSTAR, 600, shocked(None, 100, 500))
    row("logistic 1/yr, refuge 1% of C", build("logistic", 1.0, refuge=100), NSTAR, 600, shocked(None, 100, 500))
    row("constant inflow", build("constant", 0), NSTAR, 600, shocked(None, 100, 500))
    print("   (the shock function ignores its resource argument and acts on the run's own; the rows are separate runs)")
    print("\n== 2. a non-renewable deposit (constant shape, rate 0): capacity 1e9 units, start N = 0.5 N*; abundance proportional to the share left")
    res = Res("constant", 10**9, 0, 10**9, LIN)
    ser = simulate(res, NSTAR // 2, T=30, years=400)
    print("   year:N/N*:S/C  " + "  ".join(f"{y}:{n/NSTAR:.2f}:{s/res.cap:.2f}" for y, n, s, _ in ser if y in (0, 10, 20, 30, 40, 50, 60, 80, 100, 150, 250, 399)))
    print("\n== 3. the basin of the logistic stock of coupled.py 3: the largest start above N* that recovers, by bisection (T = 30, r_s = 1)")
    lo, hi = 1.0, 3.0
    for _ in range(14):
        mid = (lo + hi) / 2
        res = build("logistic", 1.0)
        ser = simulate(res, int(mid * NSTAR), T=30, years=300)
        if min(n for _, n, _, _ in ser) / NSTAR > 0.02: lo = mid
        else: hi = mid
    print(f"   recovers up to a start of {lo:.3f} N*, collapses from {hi:.3f} N* (the draw at N* is 0.96 F_max, so F_max is reached at 1.042 N*)")
    print("   with the welfare window T = 365 and a lag of 365 (a slower population):")
    lo, hi = 1.0, 3.0
    for _ in range(14):
        mid = (lo + hi) / 2
        res = build("logistic", 1.0)
        ser = simulate(res, int(mid * NSTAR), T=365, tau_e=365, years=300)
        if min(n for _, n, _, _ in ser) / NSTAR > 0.02: lo = mid
        else: hi = mid
    print(f"   recovers up to a start of {lo:.3f} N*, collapses from {hi:.3f} N*")
