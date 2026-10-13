"""The arithmetic of the worked examples of spec-world.md 19: the Weyl bucket of a few
agent IDs and the review ticks they give, the creation draws of example A and the
streams of the tick of example B, each under the keyed streams of 7 (ADR 0008). The
finaliser, the key and the PCG are the ones of seed.py. Standard library only.

Run in Docker, never on the host:
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD":/w -w /w python:3.12-slim python worked.py
"""

from seed import SITES, draw, stream_state, words


def weyl(i):
    return (i * 2654435769) % (1 << 32)


def bucket(i, p):
    # the rule bucket of spec-world.md 9.1.1: high word of Weyl2(id) * p, Weyl2(id) = id * 0x6A09E667 mod 2^32
    return (((i * 0x6A09E667) % (1 << 32)) * p) >> 32


def stream(seed, site, tick, ident, sub=None):
    return stream_state(seed, SITES[site], tick, ident, sub)


def main():
    print("Weyl32 and bucket for ids 10..15")
    for i in range(10, 16):
        print("  id %d  weyl32 %d  bucket(P=3) %d  bucket(P=10) %d" % (i, weyl(i), bucket(i, 3), bucket(i, 10)))
    print("review ticks 0..8 at P=3 for ids 10..15 (an agent acts when (t + bucket) mod 3 == 0)")
    for i in range(10, 16):
        ticks = [t for t in range(9) if (t + bucket(i, 3)) % 3 == 0]
        print("  id %d: %s" % (i, ticks))
    print("reviews per tick, ids 10..15, ticks 0..8:",
          [sum(1 for i in range(10, 16) if (t + bucket(i, 3)) % 3 == 0) for t in range(9)])
    print("rotation start of a region of 4 agents (ids 10..13), ticks 0..7:",
          [10 + (t % 4) for t in range(8)])

    # Example A (spec-world.md 19.1): the creation draws, seed 0, agents 4 to 10, tick 0.
    print("example A, seed 0: the creation draws of agents 4..10, one stream each (siteConstruction, 0, id)")
    for a in range(4, 11):
        w = words(*stream(0, "siteConstruction", 0, a), 2)
        print("  agent %d  target word %016x  residue word %016x  residue = draw(10000) = %d"
              % (a, w[0], w[1], draw(10000, w[1])))

    # Example B (spec-world.md 19.2): the streams of one tick, seed 42, tick 7.
    seed, t = 42, 7
    print("example B, seed %d, tick %d: one stream per site call, words counted from 0 in each" % (seed, t))
    intents = {1: 2, 2: 2, 3: 3}                 # agents 1, 2, 3 consider 2, 2 and 3 intents (D5)
    for a, k in intents.items():
        w = words(*stream(seed, "siteTrade", t, a), k)
        print("  D5 agent %d Trade, key (siteTrade, %d, %d): words 0..%d  first word %016x" % (a, t, a, k - 1, w[0]))
    # region 0 (r0): grain is commodity 0 and loaf is commodity 1; the admitted orders are
    # grain: ask of agent 1, ask of agent 2, bid of agent 3; loaf: bid of 1, bid of 2, ask of 3.
    markets = {0: (1, 2), 1: (2, 1)}             # commodity -> (bids, asks)
    for c, (b, s) in markets.items():
        n = b + s
        w = words(*stream(seed, "siteClearing", t, 0, c), n)
        print("  D7 market (r0, commodity %d), key (siteClearing, %d, 0, %d): words 0..%d (%d bids then %d asks)"
              "  first word %016x" % (c, t, c, n - 1, b, s, w[0]))
    orders = [(1, 0), (2, 0), (3, 0), (1, 1), (2, 1), (3, 1)]   # (agent, commodity), admission order
    for a, c in orders:
        w = words(*stream(seed, "siteOutcome", t, a, c), 1)
        print("  D8 outcome of agent %d in commodity %d, key (siteOutcome, %d, %d, %d): word 0 = %016x"
              % (a, c, t, a, c, w[0]))
    n_words = sum(intents.values()) + sum(b + s for b, s in markets.values()) + len(orders)
    print("  words in the tick: %d, in %d streams" % (n_words, len(intents) + len(markets) + len(orders)))
    print("  with agent 3 informed and 4 candidates at E > 0: %d words, the four more in agent 3's own D4 stream;"
          " no word of any other stream moves" % (n_words + 4))
    assert words(*stream(seed, "siteTrade", t, 3), 3) == words(*stream(seed, "siteTrade", t, 3), 3)


if __name__ == "__main__":
    main()
