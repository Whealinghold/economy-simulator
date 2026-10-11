"""Seeding rule of spec-world.md 7.1: SplitMix64 expansion of a seed into the
128-bit state of Go's math/rand/v2 PCG (DXSM), and a check against the values that
Go 1.25 printed in Docker (hard-coded below). Standard library only."""

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def finalise(z):
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def expand(seed):
    s = (seed + GAMMA) & M64
    hi = finalise(s)
    s = (s + GAMMA) & M64
    lo = finalise(s)
    return hi, lo


class PCG:
    """math/rand/v2 PCG: 128-bit LCG with the DXSM output function."""
    MUL = (2549297995355413924 << 64) | 4865540595714422341
    INC = (6364136223846793005 << 64) | 1442695040888963407

    def __init__(self, hi, lo):
        self.state = (hi << 64) | lo

    def uint64(self):
        self.state = (self.state * self.MUL + self.INC) & ((1 << 128) - 1)
        hi, lo = self.state >> 64, self.state & M64
        hi ^= hi >> 32
        hi = (hi * 0xDA942042E4DD58B5) & M64
        hi ^= hi >> 48
        hi = (hi * (lo | 1)) & M64
        return hi


# Printed by Go 1.25 (golang:1.25 in Docker): mixed states and first two words,
# and the first two words of rand.NewPCG(seed, 0).
GO_MIXED = {
    0: ("e220a8397b1dcdaf", "6e789e6aa1b965f4", "b18c97843f57bb91", "d7046f21f79eba31"),
    1: ("910a2dec89025cc1", "beeb8da1658eec67", "af9d32bf75779748", "b7672df032bf0473"),
    2: ("975835de1c9756ce", "bfc846100bfc1e42", "a28db69c539314fb", "fe57a921339b50b9"),
    42: ("bdd732262feb6e95", "28efe333b266f103", "61c88529c9612c1b", "2608d8a3075aa168"),
}
GO_NAIVE = {
    0: ("38ffff682123e08a", "acfc572dc29cb1fd"),
    1: ("9927a129abed2903", "16d0078c4a605356"),
    2: ("ae7a769ad189c26f", "d810a0376b22ae4e"),
    42: ("db881b72db87a99f", "f5f16d2d76b09fe4"),
}


def popcount(x):
    return bin(x).count("1")


def main():
    for seed, (hi, lo, w1, w2) in GO_MIXED.items():
        h, l = expand(seed)
        p = PCG(h, l)
        got = ("%016x" % h, "%016x" % l, "%016x" % p.uint64(), "%016x" % p.uint64())
        assert got == (hi, lo, w1, w2), (seed, got)
        q = PCG(seed, 0)
        gotn = ("%016x" % q.uint64(), "%016x" % q.uint64())
        assert gotn == GO_NAIVE[seed], (seed, gotn)
    print("mixed and naive seeding match Go 1.25 for seeds", sorted(GO_MIXED))

    # Distance between the first outputs of neighbouring seeds, both rules.
    n = 2000
    for name, make in (("mixed", lambda s: PCG(*expand(s))), ("naive", lambda s: PCG(s, 0))):
        total, lo_d, hi_d = 0, 64, 0
        for s in range(n):
            a, b = make(s), make(s + 1)
            d = 0
            for _ in range(4):
                d += popcount(a.uint64() ^ b.uint64())
            d /= 4
            total += d
            lo_d, hi_d = min(lo_d, d), max(hi_d, d)
        print("%-5s seeds 0..%d: mean differing bits between seed s and s+1 = %.2f (min %.1f, max %.1f of 64)"
              % (name, n, total / n, lo_d, hi_d))


main()
