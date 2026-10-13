"""WLD-A5: what one extra draw moves (spec-world.md 7.6). A toy world of 200 agents over
50 ticks, each agent drawing one word a tick; one agent takes one extra word at tick 1.
Counts how many later draws differ under three rules:

  one positional stream (architecture.md 7.3 before ADR 0008, rejected),
  a stream per tick (the alternative of 7.6.2 before ADR 0008, rejected),
  a stream per (site, tick, agent) (ADR 0008, adopted: 7.1.3).

The keyed stream is the one of seed.py. Standard library only.

Run in Docker, never on the host:
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD":/w -w /w python:3.12-slim python stream.py
"""

from seed import GAMMA, M64, SITES, PCG, finalise, stream_state


class SplitMix:
    def __init__(self, seed):
        self.s = seed & M64

    def next(self):
        self.s = (self.s + GAMMA) & M64
        return finalise(self.s)


def run(extra_at_tick, mode, agents=200, ticks=50, seed=7, extra_agent=0):
    """The word of each agent at each tick; `extra_agent` takes one more word, first, at
    `extra_at_tick`, which is the whole difference between the two runs."""
    out = []
    single = SplitMix(seed)
    for t in range(ticks):
        row = []
        if mode == "positional":
            for a in range(agents):
                if t == extra_at_tick and a == extra_agent:
                    single.next()
                row.append(single.next())
        elif mode == "per tick":
            g = SplitMix(finalise(seed ^ ((t * GAMMA) & M64)))
            for a in range(agents):
                if t == extra_at_tick and a == extra_agent:
                    g.next()
                row.append(g.next())
        else:
            for a in range(agents):
                p = PCG(*stream_state(seed, SITES["siteTrade"], t, a))
                if t == extra_at_tick and a == extra_agent:
                    p.uint64()
                row.append(p.uint64())
        out.append(row)
    return out


def differing(a, b):
    d = sum(1 for ta, tb in zip(a, b) for x, y in zip(ta, tb) if x != y)
    return d, sum(len(t) for t in a)


def main():
    for label, mode in (("one positional stream (rejected)", "positional"),
                        ("a stream per tick (rejected)", "per tick"),
                        ("a stream per (site, tick, agent)", "keyed")):
        d, total = differing(run(None, mode), run(1, mode))
        print("%-34s draws that differ after the change: %d of %d (%.2f%%)" % (label, d, total, 100.0 * d / total))


if __name__ == "__main__":
    main()
