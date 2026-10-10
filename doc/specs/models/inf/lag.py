"""Price-gap shipping when the shipper reads a rumour (spec-information.md 11, INF-A1, INF-A2).

A stylised, linearised model of the one loop that rumours add to TRN-A1: it extends
doc/specs/models/trn/pipeline.py with an observation lag. It is a model of the rule,
not of the library; it uses floats inside the model only (the rules are integer).

State: the price of the dear region B and of the cheap region A, as deviations from
their balanced values. The gap is x = pB - pA. A shipper lives in A, so it reads the
price of A now (local, free) and the price of B as a rumour of lag `lag` (spec 5):

    obs[t] = pB[t - lag] - pA[t]
    q[t]   = g * obs[t]                     units dispatched at tick t
    pA[t+1] = pA[t] + a * q[t]              the goods leave A at dispatch (share a)
    pB[t+1] = pB[t] - (1 - a) * q[t - d]    and reach B d ticks later (share 1 - a)

Start: pB[0] = 1, pA[0] = 0 (a unit shock to B). A gain g "decays" when the largest
|x| over the last quarter of the run is below 0.999 of the largest over the quarter
before it, as in pipeline.py.

Spread (spec 6): the shippers of a region read B at lags lag, lag+1, ..., lag+S (one
vintage per agent by agent ID mod (S+1)), each with an equal share of the gain:

    obs_j[t] = pB[t - lag - j] - pA[t],   q[t] = g * mean_j obs_j[t]

Printed: the largest decaying gain by total delay n = lag + d (a = 0, where theory gives
2 sin(pi / (2 (2 n + 1)))), then with a share a at dispatch, then with a spread.
"""
import math


def run(g, lag, d, a, spread=0, n=3000):
    pa = [0.0]
    pb = [1.0]
    q = []
    x = [1.0]
    for t in range(n):
        obs = 0.0
        for j in range(spread + 1):
            idx = t - lag - j
            pb_seen = pb[idx] if idx >= 0 else 0.0
            obs += pb_seen - pa[t]
        obs /= spread + 1
        q.append(g * obs)
        arrived = q[t - d] if t - d >= 0 else 0.0
        pa.append(pa[t] + a * q[t])
        pb.append(pb[t] - (1 - a) * arrived)
        x.append(pb[-1] - pa[-1])
        if abs(x[-1]) > 1e9:
            return float("inf")
    q4 = n // 4
    last = max(abs(v) for v in x[-q4:])
    before = max(abs(v) for v in x[-2 * q4:-q4])
    if last < 1e-9:  # decayed to the rounding floor of pB - pA
        return 0.0
    return last / before if before > 1e-300 else 0.0


def max_stable_gain(lag, d, a, spread=0, step=0.002, top=4.0):
    """Largest g such that every gain on the grid up to g decays (a scan from 0)."""
    g = step
    last = 0.0
    while g < top:
        r = run(g, lag, d, a, spread)
        if r < 0.999:
            last = g
            g += step
        else:
            break
    return last


def theory(n):
    return 2 * math.sin(math.pi / (2 * (2 * n + 1)))


if __name__ == "__main__":
    d = 2
    print("a = 0, transit delay d = 2: the largest decaying gain by observation lag")
    print("lag  n=lag+d  model   theory 2 sin(pi/(2(2n+1)))")
    for lag in (0, 1, 2, 3, 4, 6, 8):
        print(lag, lag + d, round(max_stable_gain(lag, d, 0.0), 3), round(theory(lag + d), 3))
    print()
    print("a = 0.5 (half the effect at dispatch), d = 2, by observation lag")
    for lag in (0, 1, 2, 4, 8):
        print(lag, round(max_stable_gain(lag, d, 0.5), 3))
    print()
    print("spread: the same minimum lag, vintages lag..lag+S, a = 0, d = 2")
    print("lag  S  mean lag  largest decaying gain   point lag at the mean  point lag at the minimum")
    for lag in (1, 2, 4):
        for s in (0, 1, 2, 4):
            mean = lag + s / 2
            at_min = max_stable_gain(lag, d, 0.0)
            if s == 0:
                at_mean = at_min
            elif s % 2 == 0:
                at_mean = max_stable_gain(lag + s // 2, d, 0.0)
            else:
                at_mean = float("nan")
            print(lag, s, mean, round(max_stable_gain(lag, d, 0.0, s), 3),
                  round(at_mean, 3) if at_mean == at_mean else "-", round(at_min, 3))
