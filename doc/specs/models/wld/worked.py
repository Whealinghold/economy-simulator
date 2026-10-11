"""The arithmetic of the worked examples of spec-world.md 19: the Weyl bucket of
a few agent IDs, the review ticks they give, and the stream positions of the tick
of example B. Standard library only."""


def weyl(i):
    return (i * 2654435769) % (1 << 32)


M64 = (1 << 64) - 1


def finalise(z):
    z &= M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def bucket(i, p):
    # the rule bucket of spec-world.md 9.1.1: high word of Weyl2(id) * p, Weyl2(id) = id * 0x6A09E667 mod 2^32
    return (((i * 0x6A09E667) % (1 << 32)) * p) >> 32


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
    # Example B: words of one tick
    intents = [2, 2, 3]          # agents 1, 2, 3
    orders_by_market = {"grain": (1, 2), "loaf": (2, 1)}   # (bids, asks)
    pos = 0
    print("example B stream positions in the tick (relative):")
    for a, k in zip((1, 2, 3), intents):
        print("  agent %d Trade: words %s" % (a, list(range(pos, pos + k))))
        pos += k
    for m, (b, s) in orders_by_market.items():
        print("  market %s keys (bids %d, asks %d): words %s" % (m, b, s, list(range(pos, pos + b + s))))
        pos += b + s
    n_orders = sum(b + s for b, s in orders_by_market.values())
    print("  outcomes: words %s" % list(range(pos, pos + n_orders)))
    pos += n_orders
    print("  words in the tick:", pos)
    print("with agent 3 informed and 4 candidates at E > 0: words in the tick:", pos + 4,
          "and agent 3's Trade starts at relative word", 4 + 4)


main()
