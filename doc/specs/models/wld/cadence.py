"""Load of a rotation cadence, (tick + bucket(id, P)) mod P = 0 (spec-world.md 9).
Reviews per tick for n agents, with consecutive IDs, with IDs spaced by a stride
(as when roles are created in a regular interleave), with the raw ID, with the
Weyl value and with the rule bucket of 9.1.1 (a second Weyl multiplier); the correlation of the bucket with
the migration margin offset (F4 of the review); and the drift of one agent's
review tick against the year. Standard library only."""

M64 = (1 << 64) - 1
PURPOSE = 0x5245564945572D31          # "REVIEW-1": the purpose constant of 9.1.1


def finalise(z):
    z &= M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def weyl(i):
    return (i * 2654435769) % (1 << 32)


def weyl2(i):
    """the second Kronecker sequence (multiplier 0x6A09E667): the rule of 9.1.1 (the hash was the rejected alternative)"""
    return (i * 0x6A09E667) % (1 << 32)


def mixed32(i):
    return finalise(i ^ PURPOSE) >> 32


def bucket(period, spread=weyl2):
    """offset in [0, period): the high word of spread(i) * period."""
    return lambda i: (spread(i) * period) >> 32


def loads(ids, period, ticks, offset):
    return [sum(1 for i in ids if (t + offset(i)) % period == 0) for t in range(ticks)]


def report(label, per):
    print("  %-38s min %3d max %3d mean %.1f" % (label, min(per), max(per), sum(per) / len(per)))


def main():
    n, ticks = 1000, 720
    for period in (10, 30):
        print("period %d, %d agents, %d ticks" % (period, n, ticks))
        for stride in (1, 2, 3, 5, 10, 30, 31):
            ids = [100 + k * stride for k in range(n)]
            report("raw ID, stride %d" % stride, loads(ids, period, ticks, lambda i: i))
        for stride in (1, 3, 10, 30):
            ids = [100 + k * stride for k in range(n)]
            report("Weyl value mod P, stride %d" % stride, loads(ids, period, ticks, weyl))
        for stride in (1, 2, 3, 5, 10, 30, 31):
            ids = [100 + k * stride for k in range(n)]
            report("Weyl bucket, stride %d" % stride, loads(ids, period, ticks, bucket(period, weyl)))
        for stride in (1, 2, 3, 5, 10, 30, 31):
            ids = [100 + k * stride for k in range(n)]
            report("hash bucket, stride %d" % stride, loads(ids, period, ticks, bucket(period, mixed32)))
        for stride in (1, 2, 3, 5, 10, 30, 31):
            ids = [100 + k * stride for k in range(n)]
            report("rule bucket (Weyl-2), stride %d" % stride, loads(ids, period, ticks, bucket(period)))
    # R5 of the re-verification: starting IDs, group sizes, and the rank correlation of Weyl2 with Weyl32
    print("load over starting IDs 0, 100 and 12345, strides 1 to 31, rule bucket")
    for period, nn in ((30, 1000), (10, 1000), (30, 5000), (10, 5000), (30, 200)):
        lo, hi = 10**9, 0
        for start in (0, 100, 12345):
            for stride in range(1, 32):
                per = loads([start + k * stride for k in range(nn)], period, period, bucket(period))
                lo, hi = min(lo, min(per)), max(hi, max(per))
        print("  n %4d period %2d: %3d to %3d a tick, mean %.1f, ceil(1.2 n / P) = %d" % (nn, period, lo, hi, nn / period, -(-12 * nn // (10 * period))))
    xs = [weyl(i) for i in range(1000, 2000)]
    ys = [weyl2(i) for i in range(1000, 2000)]
    def ranks(v):
        order = sorted(range(len(v)), key=lambda i: v[i])
        r = [0] * len(v)
        for rank, i in enumerate(order):
            r[i] = rank
        return r
    rx, ry = ranks(xs), ranks(ys)
    m = len(xs)
    d2 = sum((a - b) ** 2 for a, b in zip(rx, ry))
    print("Spearman rank correlation of Weyl2 with Weyl32 over 1 000 consecutive IDs: %.4f (Weyl2 is a function of Weyl32; the rule is met operationally)" % (1 - 6.0 * d2 / (m * (m * m - 1))))
    print("a group smaller than its period: 3 agents, period 30, consecutive IDs")
    report("raw ID", loads([7, 8, 9], 30, 360, lambda i: i))
    report("rule bucket (Weyl-2)", loads([7, 8, 9], 30, 360, bucket(30)))

    # F4: the migration margin g_a = MoveGain + floor(Weyl32(id) * MoveWidth / 2^32) and the review tick.
    gain, width, period, need = 1000, 5000, 30, 3000
    ids = list(range(1000, 2000))
    meet = [i for i in ids if gain + (weyl(i) * width >> 32) <= need]
    print("margin and review tick: %d of %d households meet a margin of %d (MoveGain %d, MoveWidth %d, period %d)"
          % (len(meet), len(ids), need, gain, width, period))
    for label, off in (("raw ID", lambda i: i), ("Weyl bucket", bucket(period, weyl)), ("hash bucket", bucket(period, mixed32)), ("rule bucket (Weyl-2)", bucket(period))):
        first = [(-off(i)) % period for i in meet]     # tick of the period at which the household reviews
        per = [first.count(t) for t in range(period)]
        print("  %-18s reviews of those households per tick of the period: min %d max %d, empty ticks %d, ticks used %d"
              % (label, min(per), max(per), per.count(0), period - per.count(0)))
    # the same household's two cadences (30 and 10) with the mixed buckets
    ids = range(20000)
    same = sum(1 for i in ids if bucket(30)(i) % 10 == bucket(10)(i) % 10)
    print("agents whose 30-tick review tick is also a 10-tick review tick, rule buckets (Weyl-2): %.1f%% (raw IDs: 100%%)"
          % (100.0 * same / len(ids)))
    print("drift of one agent's first review in successive years, period 30, TicksPerYear 365")
    start = []
    for year in range(6):
        base = year * 365
        start.append(next(t for t in range(base, base + 365) if (t + 11) % 30 == 0) - base)
    print("  tick of the year of the first review:", start, "(shift of TicksPerYear mod P = %d a year)" % (365 % 30))
    print("reviews of one agent in a year of 365 ticks at period 30:",
          sorted({sum(1 for t in range(y * 365, (y + 1) * 365) if (t + 11) % 30 == 0) for y in range(12)}))


main()
