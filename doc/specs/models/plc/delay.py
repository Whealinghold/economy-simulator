"""PLC-A1: an integral rule on a plant with a delay of d decisions.

Rule (linearised, no deadband, no step limit, no bounds):
    u[k+1] = u[k] - K * (y[k] - r)      y[k] = u[k-d]      (plant gain folded into K)
Characteristic polynomial  z^(d+1) - z^d + K = 0.  Classic result: stable iff
    K < 2 * sin(pi / (2 * (2d + 1))).
This script finds the largest stable K by simulation (bisection) and compares it with
the closed form, for d = 0..8, and also shows the cadence effect: for a plant delay of
L ticks and a rule that decides every P ticks, d = max(0, ceil(L / P) - 1).
Floating point is used here because this is a model of the rule, not the library.
"""
import math

def diverges(K, d, steps=60000):
    u = [0.0] * (d + 1)
    u[-1] = 1.0                      # one unit of initial disturbance
    peak = 1.0
    for _ in range(steps):
        # y[k] = u[k-d]; u[k+1] = u[k] - K*y[k]
        y = u[-1 - d]
        nxt = u[-1] - K * y
        u.append(nxt)
        u.pop(0)
        a = abs(nxt)
        if a > peak:
            peak = a
        if a > 1e6:
            return True
    return abs(u[-1]) > 1.0 and peak > 2.0 and abs(u[-1]) >= peak * 0.5

def bound_by_simulation(d):
    lo, hi = 0.0, 2.5
    for _ in range(40):
        mid = (lo + hi) / 2
        if diverges(mid, d):
            hi = mid
        else:
            lo = mid
    return lo

print("d  closed form 2 sin(pi/(2(2d+1)))   simulated bound")
for d in range(0, 9):
    cf = 2 * math.sin(math.pi / (2 * (2 * d + 1)))
    print(f"{d}  {cf:10.4f}                       {bound_by_simulation(d):10.4f}")

print()
print("Cadence: a plant delay of L ticks, a rule that decides every P ticks, d = max(0, ceil(L/P) - 1)")
print("L(ticks)  P   d   largest K per decision   largest K per tick (K/P)")
for L in (30, 60):
    for P in (1, 5, 10, 30, 60):
        # d = ceil(L/P) - 1 decisions of extra delay beyond the one inherent in sampling
        d = max(0, math.ceil(L / P) - 1)
        K = 2 * math.sin(math.pi / (2 * (2 * d + 1)))
        print(f"{L:5d}   {P:3d}  {d:2d}   {K:8.4f}                 {K / P:8.4f}")
