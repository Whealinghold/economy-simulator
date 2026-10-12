"""Arithmetic of spec-risk.md: the strike, the combined share, the loss of a line, the
share of a store, the rounding bias, the expected loss of a route, and the price-gap
shipper with a loss (stylised, floats only inside the model). Standard library only.

It is a model of the RULES of sections 3 to 6 and 8 with integers and placeholder
numbers. It does not show that any default is stable (note N-027)."""

import math

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def finalise(z):
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


class PCG:
    """math/rand/v2 PCG (DXSM) seeded as spec-world.md 7.1.2, checked in models/wld/seed.py."""
    MUL = (2549297995355413924 << 64) | 4865540595714422341
    INC = (6364136223846793005 << 64) | 1442695040888963407

    def __init__(self, seed):
        s = (seed + GAMMA) & M64
        hi = finalise(s)
        s = (s + GAMMA) & M64
        lo = finalise(s)
        self.state = (hi << 64) | lo

    def uint64(self):
        self.state = (self.state * self.MUL + self.INC) & ((1 << 128) - 1)
        hi, lo = self.state >> 64, self.state & M64
        hi ^= hi >> 32
        hi = (hi * 0xDA942042E4DD58B5) & M64
        hi ^= hi >> 48
        return (hi * (lo | 1)) & M64

    def draw(self, n):
        """The world's draw, spec-world.md 7.2.1: the high word of Uint64() * n."""
        return (self.uint64() * n) >> 64


def rhe(a, b):
    """a / b, a >= 0, b > 0, rounded half to even (spec-risk.md 3.4.1)."""
    q, r = divmod(a, b)
    if 2 * r > b or (2 * r == b and q % 2 == 1):
        q += 1
    return q


def prob(strike_bp, scale_bp=10_000):
    return min(10_000, rhe(strike_bp * scale_bp, 10_000))


def keep(struck_severities):
    """spec-risk.md 4.1.1 step 4a: the kept share after the struck rows in path order."""
    kept = 10_000
    for s in struck_severities:
        kept = rhe(kept * (10_000 - s), 10_000)
    return kept


def expected_loss_bp(rows):
    """spec-risk.md 8.1.1 over rows = [(P, SeverityBP)] in path order."""
    k = 10_000
    for p, s in rows:
        f = 10_000 - rhe(p * s, 10_000)
        k = rhe(k * f, 10_000)
    return 10_000 - k


def store_phi(stored, cap_k, sumcap, load):
    """spec-risk.md 6.1.2, one dimension, floors."""
    if sumcap == 0 or load == 0:
        return 0
    x = stored * cap_k // sumcap
    return x * 10_000 // load


def main():
    print("== 1. The world's words and the strike test (spec-risk.md 3.4.2) ==")
    for seed in (0, 1, 42):
        g = PCG(seed)
        w1, w2 = g.uint64(), g.uint64()
        d1, d2 = (w1 * 10_000) >> 64, (w2 * 10_000) >> 64
        print("seed %2d: words %016x %016x -> draw(10000) = %d, %d" % (seed, w1, w2, d1, d2))

    print()
    print("== 2. Worked example A: a hired shipment over two rows (spec-risk.md 18.1) ==")
    # rows: (north, mid, wagon) unit, StrikeBP 3 000 Severity 2 500; (mid, south, wagon) common,
    # StrikeBP 1 000 Severity 6 000. Seed 42: word 1 is the common row's (D13, P3.1a), word 2 the
    # unit row's (D14, at the record of the shipment).
    g = PCG(42)
    d13 = g.draw(10_000)
    d14 = g.draw(10_000)
    p_common, p_unit = prob(1_000), prob(3_000)
    struck = []
    print("D13 (common row, P=%d): draw %d -> %s" % (p_common, d13, "struck" if d13 < p_common else "spared"))
    print("D14 (unit row,   P=%d): draw %d -> %s" % (p_unit, d14, "struck" if d14 < p_unit else "spared"))
    sev = []
    if d14 < p_unit:
        sev.append(2_500)       # path order: the unit row (north, mid) comes first
    if d13 < p_common:
        sev.append(6_000)
    kept = keep(sev)
    loss_bp = 10_000 - kept
    print("LossBP = %d (kept %d)" % (loss_bp, kept))
    print("expected loss of the route (8.1.1) = %d" % expected_loss_bp([(p_unit, 2_500), (p_common, 6_000)]))
    # arrival: line a: 40 shipped, 3 spoiled on the way; line b: 5 shipped, none spoiled
    for name, shipped, spoiled in (("line a", 40, 3), ("line b", 5, 0), ("line c", 2, 0), ("line d", 6, 0)):
        held = shipped - spoiled
        lost = rhe(held * loss_bp, 10_000)
        print("%s: shipped %d spoiled %d held %d lost rhe(%d*%d,10000)=%d delivered %d"
              % (name, shipped, spoiled, held, held, loss_bp, lost, held - lost))
    # a contract of 30 units at price 20: 3 spoiled and 5 lost -> reached 22
    q, price, spoiled, lost = 30, 20, 3, 5
    reached = q - spoiled - lost
    print("contract: q=%d price=%d reached=%d paid=%d refund=%d" % (q, price, reached, price * reached, price * (q - reached)))

    print()
    print("== 3. Worked example B: a limited owner with two classes (spec-risk.md 18.2) ==")
    size = {"grain": (2, 1), "cloth": (1, 3)}          # (mass, bulk)
    hold = {"grain": 250, "cloth": 50}
    base = (100, 80)                                   # base room (mass, bulk), weight 1
    classes = {"bin": (200, 100, 2), "pit": (100, 200, 1)}   # (Cap mass, Cap bulk, stores owned)
    sumcap = [sum(c[0] * c[2] for c in classes.values()), sum(c[1] * c[2] for c in classes.values())]
    load = [sum(hold[c] * size[c][d] for c in hold) for d in (0, 1)]
    stored = [min(sumcap[d], max(0, load[d] - base[d])) for d in (0, 1)]
    print("load (mass, bulk) = %s, SumCap = %s, Stored = %s" % (load, sumcap, stored))
    phi = {}
    for k, (cm, cb, n) in classes.items():
        pm = store_phi(stored[0], cm, sumcap[0], load[0])
        pb = store_phi(stored[1], cb, sumcap[1], load[1])
        phi[k] = max(pm, pb)
        print("class %s: phi_mass %d phi_bulk %d phi %d" % (k, pm, pb, phi[k]))
    sev_store = {"bin": 4_000, "pit": 10_000}
    n_struck = {"bin": 1, "pit": 1}      # one bin struck by its unit row, the pit by its common row
    share = {k: min(10_000, n_struck[k] * rhe(phi[k] * sev_store[k], 10_000)) for k in classes}
    print("share (bp): %s" % share)
    for c, q0 in hold.items():
        taken = 0
        parts = []
        for k in ("bin", "pit"):
            l = min(q0 - taken, rhe(q0 * share[k], 10_000))
            taken += l
            parts.append("%s %d" % (k, l))
        print("%s: q0 %d lost %s total %d left %d" % (c, q0, ", ".join(parts), taken, q0 - taken))

    print()
    print("== 4. Worked example C: a trip to oneself (spec-risk.md 18.3) ==")
    vehicle = "cart"
    for c, q0 in (("grain", 12), ("cloth", 3), ("cart", 2)):
        if c == vehicle:
            print("%s: held %d, the mode's Vehicle: not exposed (5.4.1); a rule over every commodity would take %d"
                  % (c, q0, rhe(q0 * 2_500, 10_000)))
        else:
            print("%s: held %d lost rhe(%d*2500,10000) = %d" % (c, q0, q0, rhe(q0 * 2_500, 10_000)))

    print()
    print("== 5. The rounding bias of a line (spec-risk.md 5.3.2): lost units against q*share ==")
    shares = (500, 1_500, 2_500, 5_000, 7_500)
    print("q     " + "  ".join("%5d" % s for s in shares) + "   (lost / ideal q*s/10000)")
    for q in (1, 2, 3, 5, 10, 40, 100):
        cells = []
        for s in shares:
            cells.append("%d/%.2f" % (rhe(q * s, 10_000), q * s / 10_000))
        print("%-5d " % q + "  ".join("%9s" % c for c in cells))
    print("mean error over q = 1..20 (lost - ideal), by share:")
    for s in (500, 1_000, 2_500, 4_000, 5_000, 6_000, 7_500, 9_500):
        err = sum(rhe(q * s, 10_000) - q * s / 10_000 for q in range(1, 21)) / 20
        print("  share %5d: %+.3f units a line" % (s, err))

    print()
    print("== 6. The expected loss of a route against 500 000 simulated dispatches (8.1.1) ==")
    rows = [(prob(3_000), 2_500), (prob(1_000), 6_000)]
    g = PCG(7)
    total, total2 = 0, 0
    n = 500_000
    for _ in range(n):
        sev = [s for p, s in rows if g.draw(10_000) < p]
        v = 10_000 - keep(sev)
        total += v
        total2 += v * v
    mean = total / n
    se = ((total2 / n - mean * mean) / n) ** 0.5
    print("formula %d bp; mean simulated LossBP %.1f bp, standard error %.1f (n=%d)" % (expected_loss_bp(rows), mean, se, n))
    freq = sum(1 for _ in range(n) if g.draw(10_000) < 3_000) / n
    print("frequency of a strike at P=3000: %.4f (want 0.3000)" % freq)

    print()
    print("== 7. The price-gap shipper with a loss (spec-risk.md 16, RSK-A1) ==")
    # Part a. Totals: a one-sided model, floats inside the model. The origin price rises by alpha
    # per unit dispatched at once, the destination price falls by beta per unit that ARRIVES, d
    # ticks later, and arrivals are (1 - lam) of what was dispatched. Shipments are never negative.
    # Known: the shipper nets the loss out of the destination price. Unknown: it does not.
    def totals(g_, d, lam, known, alpha=0.5, beta=0.5, pd0=180.0, po0=100.0, f=20.0, n=3000):
        po, pd = po0, pd0
        q = []
        total, arrived = 0.0, 0.0
        for t in range(n):
            gap = ((1 - lam) * pd if known else pd) - po - f
            want = max(0.0, g_ * gap)
            q.append(want)
            total += want
            a = (1 - lam) * q[t - d] if t - d >= 0 else 0.0
            arrived += a
            po += alpha * want
            pd -= beta * a
        return total, arrived, pd, po

    print("pd0 180, po0 100, freight 20, gain 0.05, delay 3: shipped, arrived, wasted, final destination and origin price, real margin per unit")
    for lam_bp in (0, 2_500, 5_000):
        lam = lam_bp / 10_000
        for known in (True, False):
            tot, arr, pd, po = totals(0.05, 3, lam, known)
            print("  loss %4d %-7s shipped %6.2f arrived %6.2f wasted %5.2f  Pd %6.2f Po %6.2f  margin %+6.2f"
                  % (lam_bp, "known" if known else "unknown", tot, arr, tot - arr, pd, po, (1 - lam) * pd - po - 20.0))

    # Part b. Stability: the signed linear model of models/trn/pipeline.py with a loss. The gap
    # that the shipper reads is x (unknown) or y = (1-lam) Pd - Po - f (known).
    #   unknown: x[t+1] = x[t] - a q[t] - (1-a) (1-lam)   q[t-d],  q = g x
    #   known:   y[t+1] = y[t] - a q[t] - (1-a) (1-lam)^2 q[t-d],  q = g y
    def amplitude(g_, d, a, scale, n=4000):
        x = [1.0]
        q = []
        for t in range(n):
            want = g_ * x[t]
            q.append(want)
            arrived = q[t - d] if t - d >= 0 else 0.0
            x.append(x[t] - a * want - (1 - a) * scale * arrived)
            if abs(x[-1]) > 1e9:
                return float("inf")
        q4 = n // 4
        last = max(abs(v) for v in x[-q4:])
        if last < 1e-100:  # decayed to nothing: a ratio of two underflowed values is not a measure
            return 0.0
        before = max(abs(v) for v in x[-2 * q4:-q4])
        return last / before if before > 0 else 0.0

    def max_gain(d, a, scale, step=0.002):
        g_, best = step, 0.0
        while g_ < 4.0:
            if amplitude(g_, d, a, scale) < 0.999:
                best = g_
                g_ += step
            else:
                break
        return best

    print("largest gain that decays, delay 3, immediate share a = 0 (theory g0/(1-lam) unknown, g0/(1-lam)^2 known):")
    g0 = 2 * math.sin(math.pi / (2 * (2 * 3 + 1)))
    for lam_bp in (0, 2_500, 5_000):
        lam = lam_bp / 10_000
        gu = max_gain(3, 0.0, 1 - lam)
        gk = max_gain(3, 0.0, (1 - lam) ** 2)
        print("  loss %4d: unknown %.3f (theory %.3f); known %.3f (theory %.3f)"
              % (lam_bp, gu, g0 / (1 - lam), gk, g0 / (1 - lam) ** 2))

    # Part c. The same with half of the effect at dispatch (a = 0.5): the limit of L84 is then
    # at least about 2 for any delay, and the loss scales the arrival link only.
    print("largest gain that decays, delay 3, immediate share a = 0.5 (search stops at 4.0):")
    for lam_bp in (0, 2_500, 5_000):
        lam = lam_bp / 10_000
        gu = max_gain(3, 0.5, 1 - lam)
        gk = max_gain(3, 0.5, (1 - lam) ** 2)
        print("  loss %4d: unknown %.3f; known %.3f" % (lam_bp, gu, gk))


main()
