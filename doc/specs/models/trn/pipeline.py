"""Price-gap shipping with a transit delay (spec-transport.md section 12, TRN-A1).

A stylised, linearised model of the one loop that transport adds between two
regions. It is a model of the rule, not of the library: it uses floats inside the
model only (the rules themselves are integer).

State: x[t], the price gap between a dear region B and a cheap region A, in units
of one unit of shipped quantity's effect on the gap (so the sensitivity is folded
into the gain g).

Rule: each tick a shipper dispatches q[t] = g * x[t] units (the gap times a gain).
Effect on the gap: the goods leave A at dispatch, which raises A's price at once
(share `a` of the sensitivity); they reach B `d` ticks later, which lowers B's price
then (share 1 - a).

    x[t+1] = x[t] - a * q[t] - (1 - a) * q[t - d]

Variants:
  plain    q[t] = g * x[t]
  aware    q[t] = g * (x[t] - (1 - a) * sum(q in flight))    (the shipper counts what is
           already on the way: the `Inbound` of spec-transport.md 9.4)
  capped   q[t] = min(cap, g * x[t])                         (a limit on what can be shipped)

The amplitude is max |x| over the last quarter of the run, from x[0] = 1. A gain
"decays" when that amplitude is below 0.999 of the one over the quarter before it.
"""
import math


def run(g, d, a, variant="plain", cap=None, n=4000, capped_or_plain_returns_ratio=False):
    x = [1.0]
    q = []
    for t in range(n):
        inflight = sum(q[max(0, t - d):t]) if d > 0 else 0.0
        if variant == "aware":
            want = g * (x[t] - (1 - a) * inflight)
        else:
            want = g * x[t]
        if variant == "capped":
            want = max(-cap, min(cap, want))
        q.append(want)
        arrived = q[t - d] if t - d >= 0 else 0.0
        x.append(x[t] - a * want - (1 - a) * arrived)
        if abs(x[-1]) > 1e9:
            return float("inf")
        if max(abs(v) for v in x[-(d + 2):]) < 1e-12:
            return 0.0  # decayed to nothing: stop before floating-point underflow
    q4 = n // 4
    last = max(abs(v) for v in x[-q4:])
    before = max(abs(v) for v in x[-2 * q4:-q4])
    if capped_or_plain_returns_ratio:
        return last / before if before > 0 else 0.0
    return last


def max_stable_gain(d, a, variant="plain", cap=None, step=0.002):
    """Largest g such that every gain on the grid up to g decays (a scan from 0, not a
    bisection: with an immediate share the stable gains need not form one interval)."""
    g = step
    last = 0.0
    while g < 4.0:
        if run(g, d, a, variant, cap, capped_or_plain_returns_ratio=True) < 0.999:
            last = g
            g += step
        else:
            break
    return last


if __name__ == "__main__":
    print("maximum gain g that decays, by delay d (ticks) and immediate share a")
    print("d  a=0  (theory 2sin(pi/(2(2d+1))))   a=0.5   a=0.8   aware a=0")
    for d in (0, 1, 2, 3, 4, 6, 8, 12, 20):
        theory = 2 * math.sin(math.pi / (2 * (2 * d + 1)))
        print(
            d,
            round(max_stable_gain(d, 0.0), 3),
            round(theory, 3),
            round(max_stable_gain(d, 0.5), 3),
            round(max_stable_gain(d, 0.8), 3),
            round(max_stable_gain(d, 0.0, "aware"), 3),
        )
    print()
    print("a=0, g above the limit, with a cap on the shipment per tick (amplitude of the gap):")
    for d in (2, 6):
        limit = max_stable_gain(d, 0.0)
        for mult in (1.5, 3.0):
            g = limit * mult
            row = [round(run(g, d, 0.0, "plain"), 3) if run(g, d, 0.0) != float("inf") else "inf"]
            for cap in (0.5, 0.2, 0.05):
                row.append(round(run(g, d, 0.0, "capped", cap), 3))
            print(f"d={d} g={g:.3f} ({mult}x limit): uncapped, cap 0.5, 0.2, 0.05 ->", row)
