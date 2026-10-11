"""Spec section 8: when a paired comparison is valid, and how a harness sees it.

A toy world. One stream of words (a SplitMix64 counter, standing in for the
world's PCG: equal states <=> equal number of words drawn). Each tick draws one
word for a shock and, depending on the state, some extra words (so the count of
words per tick depends on the state, as in the library: orders admitted,
intents considered). Two parameter settings A and B run from the same seed.

Two worlds:
  calm  an extra word only while the price is above a threshold (counts differ
        rarely and by one word, so the offset between the two streams stays small)
  busy  (p - 100) // 4 extra words while the price is above 100 (counts differ by
        many words, so the offset between the streams random-walks away)

For each pair of settings we report:
  aligned   seeds whose two runs drew the same number of words at every sample
            tick, i.e. whose runs used the same word at every position
  first     the first sample tick at which the counts differ (min / median / max)
  paired    the sample standard deviation of B_s - A_s over seeds s
  sd indep  isqrt_nearest(vA + vB), vA and vB the rounded sample variances of A_s
            and B_s: the spread of the difference of two independent runs
  gain_bp   rdiv(10 000 * sd indep, sd(d)): spec 7.4.2 exactly

The mean of B_s - A_s is an unbiased estimate of the difference of the two
means whether or not the runs are aligned; alignment says why pairing helps,
and the gain says how much it did. A model of the rule, not of the library
(note N-027).
"""
from math import isqrt


def rdiv(n, d):
    q, r = divmod(n, d)
    t = 2 * r
    if t > d or (t == d and q % 2 == 1):
        q += 1
    return q


def isqrt_nearest(n):
    r0 = isqrt(n)
    return r0 + 1 if n - r0 * r0 > r0 else r0

MASK = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def fin(z):
    z &= MASK
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & MASK
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & MASK
    return z ^ (z >> 31)


def run(seed, theta, world, ticks=300, sample=25):
    """(measure, words drawn so far at each sample tick)."""
    state = fin(seed * GAMMA + 1)
    k = 0
    p = 100
    total = 0
    fingerprints = []
    for t in range(ticks):
        k += 1
        w = fin(state + k * GAMMA)
        shock = (w % 41) - 20
        p += shock - (theta * (p - 100)) // 100
        if world == "calm":
            extra = 1 if p > 112 else 0
        else:
            extra = max(0, (p - 100) // 4)
        for _ in range(extra):
            k += 1
            p -= fin(state + k * GAMMA) % 3 - 1
        total += p
        if (t + 1) % sample == 0:
            fingerprints.append(k)
    return total // ticks, fingerprints


def var(vals):
    """Sample variance rounded half to even (spec 8.3 and 7.4.2)."""
    n = len(vals)
    s = sum(vals)
    ss = sum(v * v for v in vals)
    return rdiv(n * ss - s * s, n * (n - 1))


def std(vals):
    return isqrt_nearest(var(vals))


def study(world, theta_a, theta_b, seeds=200):
    paired, unpaired, aligned_end, first_div = [], [], 0, []
    ra, rb = [], []
    for s in range(1, seeds + 1):
        ma, fa = run(s, theta_a, world)
        mb, fb = run(s, theta_b, world)
        mu, _ = run(s + 1000, theta_b, world)
        ra.append(ma)
        rb.append(mb)
        paired.append(mb - ma)
        unpaired.append(mu - ma)
        if fa == fb:
            aligned_end += 1
        else:
            first_div.append(next(i for i, (x, y) in enumerate(zip(fa, fb)) if x != y) + 1)
    return paired, unpaired, aligned_end, first_div, ra, rb


def main():
    print("ticks 300, a sample every 25 ticks (12 samples), 200 seeds; measure = mean price over the run")
    print("theta is the pull to 100 per tick in percent")
    for world in ("calm", "busy"):
        print()
        print("world %s" % world)
        print("  A   B   aligned     first (min/med/max)   sd(d)  sd indep  gain_bp (7.4.2)")
        for ta, tb in ((10, 10), (10, 11), (10, 12), (10, 20)):
            paired, unpaired, aligned, fd, ra, rb = study(world, ta, tb)
            fd.sort()
            fds = "%2d / %2d / %2d" % (fd[0], fd[len(fd) // 2], fd[-1]) if fd else "   none    "
            sd = std(paired)
            sd_ind = isqrt_nearest(var(ra) + var(rb))
            gain = str(rdiv(10000 * sd_ind, sd)) if sd else "inf"
            print("  %2d  %2d  %3d of 200  %s     %4d   %4d    %s" % (ta, tb, aligned, fds, sd, sd_ind, gain))


if __name__ == "__main__":
    main()
