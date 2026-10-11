"""Fairness of a capped, tied admission (spec-world.md 10): who is admitted when
more candidates tie than the cap allows. Compares 'lower ID first' with the keyed
tie-break K(t, id) of 10.1.1 and with a rotation of the starting index. Standard
library only, no random numbers (the key is a stateless hash)."""

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def key(t, agent):
    z = (t * GAMMA + agent) & M64
    z = (z + GAMMA) & M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def run(rule, n=100, cap=5, ticks=1000):
    wins = [0] * n
    for t in range(ticks):
        ids = list(range(n))
        if rule == "id":
            order = ids
        elif rule == "key":
            order = sorted(ids, key=lambda a: (key(t, a), a))
        else:  # rotation
            r = t % n
            order = ids[r:] + ids[:r]
        for a in order[:cap]:
            wins[a] += 1
    return wins


def summary(name, wins):
    n = len(wins)
    decile = [sum(wins[i * n // 10:(i + 1) * n // 10]) for i in range(10)]
    print("%-9s min %4d max %4d  by decile of ID: %s" % (name, min(wins), max(wins), decile))


if __name__ == "__main__":
    print("100 tied candidates, cap 5 a tick, 1000 ticks: expected 50 admissions each")
    for rule in ("id", "key", "rotation"):
        summary(rule, run(rule))
    # Starts: first position in a region of 4 agents over 8 ticks (rotation of 6.1.4)
    print("first agent in the rotation of a region of 4, ticks 0..7:", [t % 4 for t in range(8)])
    # A cap that binds only some ticks: 7 candidates tie, cap 3, a candidate set that changes
    wins = {}
    for t in range(1000):
        cands = [a for a in range(20) if (a + t) % 3 != 0]   # about two thirds present
        order = sorted(cands, key=lambda a: (key(t, a), a))
        for a in order[:3]:
            wins[a] = wins.get(a, 0) + 1
    vals = [wins[a] for a in range(20)]
    print("changing candidate set, keyed ties: min %d max %d (mean %.1f)" % (min(vals), max(vals), sum(vals) / 20))
