"""spec-fiscal.md 19 (FIS-A2): the CASH channel of a duty, one sector, exact.
Households hold Mh and spend a share s of it each tick on goods sold by other
households; the sellers pay a duty at rate tau on their sales into a treasury; the
treasury pays back a share r of its balance each tick. Nothing else reacts. A model
of the rule, not of the library, and no claim that any default is stable."""
from fractions import Fraction as F

M = F(1_000_000)
s = F(1, 2)


def run(tau, r, ticks):
    mh, tr = M, F(0)
    rev = []
    for _ in range(ticks):
        duty = tau * s * mh
        pay = r * tr
        rev.append(duty)
        mh = mh - duty + pay
        tr = tr + duty - pay
    return rev, mh, tr


taus = [F(k, 20) for k in range(1, 21)]          # 5% to 100% of sales
print("revenue per tick at tick t, share of M (x1000), by duty rate; r = 0 (a hoard)")
print("tau   " + "  ".join(f"t={t:<4}" for t in (0, 5, 20, 100)))
for tau in taus:
    rev, _, _ = run(tau, F(0), 101)
    print(f"{float(tau):.2f}  " + "  ".join(f"{float(rev[t] / M * 1000):6.1f}" for t in (0, 5, 20, 100)))
for t in (0, 5, 20, 100):
    best = max(taus, key=lambda x: run(x, F(0), t + 1)[0][t])
    print(f"t={t}: the best tau on the grid is {float(best):.2f}; the formula 1/(s*(t+1)) = {float(1 / (s * (t + 1))):.3f}")

print()
print("steady state, share of M: revenue and households' money, tau = 0.10 and 0.50")
for r in (F(0), F(1, 20), F(1, 5), F(1)):
    row = []
    for tau in (F(1, 10), F(1, 2)):
        rev, mh, tr = run(tau, r, 2000)
        row.append(f"tau {float(tau):.2f}: revenue {float(rev[-1] / M):.4f} Mh {float(mh / M):.4f}")
    cl = []
    for tau in (F(1, 10), F(1, 2)):
        cl.append(f"{float(tau * s * M / (1 + tau * s / r) / M):.4f}" if r else "0")
    print(f"r={float(r):.2f}: " + "; ".join(row) + f"; closed form revenue {cl}")
