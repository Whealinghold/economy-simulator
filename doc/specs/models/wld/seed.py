"""The keyed streams of spec-world.md 7 (ADR 0008): the single source of the vectors.

Everything numeric that spec-world.md 7 and spec-checkpoint.md 6.8 quote is produced
here, and worked.py, stream.py and ../chk/worked.py import it, so there is one
implementation of the finaliser, of the key and of the PCG-DXSM, and one set of
vectors. A small Go program (stdlib math/rand/v2 only, not archived: a main package
under doc/ would be built by go vet) printed the same lines and the two outputs are
equal, see the README.

Rules modelled:
  finalise(z)                      the SplitMix64 finaliser (7.1.2)
  stream_state(seed, site, tick, id[, sub])   the key of a stream (7.1.3)
  K(t, id)                         the stateless tie key of RES 5.3.3 (7.1.7, 10.1.1)
  reseed(seed, salt)               seed' = finalise(seed xor finalise(salt)) (CHK 6.8.2)
  draw(n, word)                    the world's draw, the high word of word * n (7.2.1)

Standard library only. Run in Docker, never on the host:
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD":/w -w /w python:3.12-slim python seed.py
"""

import hashlib

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15

# The site constants of the slice (7.1.4). Literal, fixed values: a row's constant does
# not depend on its position in a table, so a later row appends and moves nothing.
# Each equals the first 8 bytes, big-endian, of SHA-256("economy-sim/site/" + name);
# the derivation is a record of where the literal came from, the literal is the rule.
SITES = {
    "siteConstruction": 0xC103123756D13F20,  # D1
    "siteWork": 0xDE594838D3779A37,          # D3
    "siteTrade": 0xFC14CD10E8A8EF43,         # D5
    "siteClearing": 0xD364E6AE73E1313F,      # D7
    "siteOutcome": 0xD9CC0A517EB0D9F5,       # D8
    "siteSpawner": 0xBE0B1D169200E9FF,       # D9
    "siteCreation": 0x9B8FFA303CB545DC,      # D10
}


def derive_site(name):
    return int.from_bytes(hashlib.sha256(("economy-sim/site/" + name).encode()).digest()[:8], "big")


for _n, _v in SITES.items():
    assert derive_site(_n) == _v, _n


def u64(x):
    """The two's-complement conversion of a signed 64-bit value to uint64 (7.1.3)."""
    assert -(1 << 63) <= x < (1 << 64)
    return x & M64


def finalise(z):
    z &= M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def chain(seed, site, tick, ident, sub=None):
    """s after the fold over (site, tick, id[, sub]); a bare seed is the empty fold."""
    s = u64(seed)
    words = [site, u64(tick), u64(ident)]
    if sub is not None:
        words.append(u64(sub))
    for w in words:
        s = finalise((s + w + GAMMA) & M64)
    return s


def state_of(s):
    hi = finalise((s + GAMMA) & M64)
    lo = finalise((s + 2 * GAMMA) & M64)
    return hi, lo


def stream_state(seed, site, tick, ident, sub=None):
    return state_of(chain(seed, site, tick, ident, sub))


def bare_state(seed):
    """The state of a bare seed: the empty fold. It is the seeding rule that ADR 0008
    replaced, and no site draws from it; its four rows pin the finaliser and the PCG."""
    return state_of(u64(seed))


def K(t, ident):
    """The stateless tie key of RES 5.3.3: x = t * gamma + id, z = x + gamma, mixed."""
    return finalise((u64(t) * GAMMA + u64(ident) + GAMMA) & M64)


def reseed(seed, salt):
    return finalise(u64(seed) ^ finalise(u64(salt)))


def draw(n, word):
    assert n >= 1
    return (word * n) >> 64


class PCG:
    """math/rand/v2 PCG: a 128-bit LCG with the DXSM output function."""
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
        return (hi * (lo | 1)) & M64

    def marshal(self):
        return b"pcg:" + (self.state >> 64).to_bytes(8, "big") + (self.state & M64).to_bytes(8, "big")


def words(hi, lo, n):
    p = PCG(hi, lo)
    return [p.uint64() for _ in range(n)]


# Printed by Go 1.25 (golang:1.25 in Docker) before ADR 0008: the state a bare seed gave
# under the old rule, which is the empty fold of the new one, and the first two words;
# and the first two words of rand.NewPCG(seed, 0). Kept so that the four rows of the
# finaliser test of the first code are the rows that were checked against Go.
GO_BARE = {
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

# The rows of table 7.1.5 (a): name, seed, site, tick, id, sub.
KEY_ROWS = [
    ("k1", 0, "siteWork", 0, 0, None),
    ("k2", 1, "siteTrade", 5, 17, None),
    ("k3", 42, "siteClearing", 100, 2, 3),
    ("k4", 42, "siteOutcome", 100, 17, 3),
    ("k5", M64, "siteConstruction", 0, 4, None),
    ("k6", 7, "siteCreation", -1, 12, None),
    ("k7", 7, "siteSpawner", 3, -1, None),
    ("k8", 7, "siteSpawner", -1, -1, None),
    ("k9", 1 << 63, "siteTrade", 1 << 40, (1 << 40) + 1, None),
]

# The rows of table 7.1.5 (b): K(t, id).
K_ROWS = [(0, 0), (1, 0), (0, 1), (5, 17), (100, 3), (1 << 40, 123456789), (-1, -1), (-1, 0)]

# The rows of table 6.8 of the checkpoint spec: reseed(seed, salt).
RESEED_SEEDS = [0, 1, 42, M64]
RESEED_SALTS = [0, 1, 2, 42]


def popcount(x):
    return bin(x).count("1")


def vectors():
    """Every quoted vector as one line of text; the Go program prints the same lines."""
    out = []
    for seed in sorted(GO_BARE):
        hi, lo = bare_state(seed)
        w = words(hi, lo, 2)
        out.append("bare seed=%d hi=%016x lo=%016x w0=%016x w1=%016x" % (seed, hi, lo, w[0], w[1]))
    for name, seed, site, tick, ident, sub in KEY_ROWS:
        s = chain(seed, SITES[site], tick, ident, sub)
        hi, lo = state_of(s)
        w = words(hi, lo, 2)
        out.append(
            "key %s seed=%d site=%s tick=%d id=%d sub=%s s=%016x hi=%016x lo=%016x w0=%016x w1=%016x draw1(w0)=%d draw100(w1)=%d"
            % (name, seed, site, tick, ident, "-" if sub is None else sub, s, hi, lo, w[0], w[1],
               draw(1, w[0]), draw(100, w[1])))
    for t, ident in K_ROWS:
        out.append("K t=%d id=%d key=%016x" % (t, ident, K(t, ident)))
    for seed in RESEED_SEEDS:
        for salt in RESEED_SALTS:
            s2 = reseed(seed, salt)
            hi, lo = stream_state(s2, SITES["siteWork"], 0, 0)
            out.append("reseed seed=%d salt=%d seed2=%016x work(0,0).w0=%016x"
                       % (seed, salt, s2, words(hi, lo, 1)[0]))
    for n, site in sorted(SITES.items()):
        out.append("site %s %016x" % (n, site))
    return out


def main():
    for seed, (hi, lo, w1, w2) in GO_BARE.items():
        h, l = bare_state(seed)
        p = PCG(h, l)
        got = ("%016x" % h, "%016x" % l, "%016x" % p.uint64(), "%016x" % p.uint64())
        assert got == (hi, lo, w1, w2), (seed, got)
        q = PCG(seed, 0)
        gotn = ("%016x" % q.uint64(), "%016x" % q.uint64())
        assert gotn == GO_NAIVE[seed], (seed, gotn)
    print("bare-seed rule and plain NewPCG(seed, 0) match Go 1.25 for seeds", sorted(GO_BARE))

    # The properties the rules rest on.
    assert finalise(0) == 0  # the one fixed point that matters: seed 0 with salt 0 reseeds to 0
    assert chain(5, 0, 0, 0) == chain(5, 0, 0, 0)
    # A stream of one key is a function of the key alone: re-deriving gives the same words.
    a = words(*stream_state(3, SITES["siteTrade"], 9, 4), 5)
    b = words(*stream_state(3, SITES["siteTrade"], 9, 4), 5)
    assert a == b
    # The first two words of two keys differing only in id share nothing visible.
    assert a != words(*stream_state(3, SITES["siteTrade"], 9, 5), 5)
    # The bound-1 draw is 0 and still consumes the word: the next draw sees w1, not w0.
    w = words(*stream_state(3, SITES["siteTrade"], 9, 4), 2)
    assert draw(1, w[0]) == 0 and draw(1, w[1]) == 0
    # Distinct salts give distinct seeds (finalise is a bijection of uint64).
    assert len({reseed(7, s) for s in range(1000)}) == 1000
    # K takes the extra increment of RES 5.3.3: the keys of (t, id) and (t, id + 1) are
    # not each other's.
    assert K(1, 0) != finalise((1 * GAMMA) & M64)
    print("properties hold")

    print("--- vectors")
    for line in vectors():
        print(line)

    print("--- distance between first outputs of neighbours (4 words each, 2000 pairs)")
    n = 2000
    makers = (
        ("bare mix", lambda s: PCG(*bare_state(s))),
        ("naive", lambda s: PCG(s, 0)),
        ("keyed, id s vs s + 1", lambda s: PCG(*stream_state(11, SITES["siteTrade"], 40, s))),
        ("keyed, tick s vs s + 1", lambda s: PCG(*stream_state(11, SITES["siteTrade"], s, 40))),
        ("keyed, seed s vs s + 1", lambda s: PCG(*stream_state(s, SITES["siteTrade"], 40, 7))),
    )
    for name, make in makers:
        total, lo_d, hi_d = 0, 64, 0
        for s in range(n):
            a, b = make(s), make(s + 1)
            d = sum(popcount(a.uint64() ^ b.uint64()) for _ in range(4)) / 4
            total += d
            lo_d, hi_d = min(lo_d, d), max(hi_d, d)
        print("%-24s 0..%d: mean differing bits = %.2f (min %.1f, max %.1f of 64)"
              % (name, n, total / n, lo_d, hi_d))


if __name__ == "__main__":
    main()
