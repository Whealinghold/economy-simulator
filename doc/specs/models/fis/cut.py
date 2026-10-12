"""Models the arithmetic of spec-fiscal.md 8.4 (the pro rata cut with the largest
remainder and the keyed tie) and 6.3 (the carry of a duty). Integer only, standard
library only. Not the library; checks that the rules have the properties claimed."""
import random

MASK = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


def key(t, agent):
    """The stateless key of spec-resources.md 5.3.3 (the SplitMix64 finaliser)."""
    x = (t * GAMMA + agent) & MASK
    z = (x + GAMMA) & MASK
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & MASK
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & MASK
    return z ^ (z >> 31)


def cut(t, dues, avail):
    """dues: list of (id, due). Returns {id: applied}. spec-fiscal.md 8.4."""
    total = sum(d for _, d in dues)
    if total <= avail:
        return {i: d for i, d in dues}
    q, rem = {}, {}
    for i, d in dues:
        num = d * avail
        q[i], rem[i] = num // total, num % total
    left = avail - sum(q.values())
    order = sorted((i for i, d in dues if d > 0), key=lambda i: (-rem[i], key(t, i), i))
    for i in order[:left]:
        q[i] += 1
    return q


def main():
    rng = random.Random(20261012)
    n_inst = 20000
    cut_inst = 0
    max_left = 0
    for _ in range(n_inst):
        n = rng.randint(1, 120)
        dues = [(i * 7 + 3, rng.choice([0, 1, 2, 5, 10, 37, 100, 1000, rng.randint(0, 10**6)]))
                for i in range(n)]
        total = sum(d for _, d in dues)
        avail = rng.randint(0, total + 5)
        t = rng.randint(0, 10**6)
        out = cut(t, dues, avail)
        # exact use of the treasury, never above a due, never negative
        assert sum(out.values()) == min(avail, total)
        assert all(0 <= out[i] <= d for i, d in dues)
        # independent of the order the recipients are listed in
        sh = dues[:]
        rng.shuffle(sh)
        assert cut(t, sh, avail) == out
        # a recipient that is due nothing gets nothing
        assert all(out[i] == 0 for i, d in dues if d == 0)
        if total > avail:
            cut_inst += 1
            floorsum = sum((d * avail) // total for _, d in dues)
            max_left = max(max_left, avail - floorsum)
            # at most one unit more than the floor of the proportional share
            assert all(out[i] - (d * avail) // total in (0, 1) for i, d in dues)
    print(f"cut: {n_inst} instances, {cut_inst} short; all identities held; largest leftover {max_left}")

    # the key breaks equal remainders differently from one tick to the next
    dues = [(i, 3) for i in range(10)]
    first = {}
    for t in range(1000):
        out = cut(t, dues, 15)           # 30 due, 15 available: 1.5 each, 5 get the extra unit
        for i, v in out.items():
            first[i] = first.get(i, 0) + (v - 1)
    print("extra units over 1000 ticks, 10 equal recipients, 5 extra a tick:",
          [first[i] for i in range(10)], "sum", sum(first.values()))
    lowest = {}
    for t in range(1000):
        out = cut(t, dues, 15)
        for i, v in out.items():
            lowest[i] = lowest.get(i, 0) + (1 if v == 2 else 0)
    # the rule "lower ID first" would give 1000 to ids 0 to 4 and 0 to the rest
    print("by the key; by lower ID first would be [1000]*5 + [0]*5")

    # the carry of a duty (6.3): exact over any run, no drift
    D = 10**8
    for _ in range(2000):
        rate = rng.randint(1, 10000)
        c = rng.randint(0, D - 1)
        c0 = c
        paid = 0
        acc = 0
        for _ in range(rng.randint(1, 60)):
            base = rng.choice([0, 1, 9, 100, 530, 1350, rng.randint(0, 10**7)])
            scale = rng.choice([10000, 10000, 5000, 0, 25000])
            num = base * rate * scale + c
            due, c = divmod(num, D)
            paid += due
            acc += base * rate * scale
        assert paid == (acc + c0) // D and c == (acc + c0) % D
    print("duty carry: 2000 runs, the total paid equals floor((sum + c0) / 10^8) exactly")


if __name__ == "__main__":
    main()
