"""WLD-A5: a single stream is positional (spec-world.md 7.6). Two runs of a toy
world of 200 agents over 50 ticks, each agent drawing one word a tick from one
stream, differ by one extra draw at tick 1. Counts how many later draws differ,
for one stream and for a stream per tick (the alternative of 7.6.2, not adopted).
Standard library only."""

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def finalise(z):
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


class SplitMix:
    def __init__(self, seed):
        self.s = seed & M64

    def next(self):
        self.s = (self.s + GAMMA) & M64
        return finalise(self.s)


def run(extra_at_tick, per_tick_streams, agents=200, ticks=50, seed=7):
    out = []
    single = SplitMix(seed)
    for t in range(ticks):
        g = SplitMix(finalise(seed ^ (t * GAMMA))) if per_tick_streams else single
        if t == extra_at_tick:
            g.next()          # one agent takes one more word than in the other run
        out.append([g.next() for _ in range(agents)])
    return out


def differing(a, b):
    d = sum(1 for ta, tb in zip(a, b) for x, y in zip(ta, tb) if x != y)
    return d, sum(len(t) for t in a)


if __name__ == "__main__":
    for label, per_tick in (("one stream (architecture.md 7.3)", False), ("a stream per tick (7.6.2)", True)):
        base = run(None, per_tick)
        other = run(1, per_tick)
        d, total = differing(base, other)
        print("%-34s draws that differ after the change: %d of %d (%.1f%%)" % (label, d, total, 100.0 * d / total))
