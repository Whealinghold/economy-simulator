"""Size of a checkpoint under two integer encodings (spec-checkpoint.md 10).

A model of the field inventory of spec-checkpoint.md 4, not of the library. It counts the
integers each section writes for a world and the bytes they take as minimal LEB128 varints
(zigzag for signed values) and as 8-byte fixed-width integers. The value magnitudes are
invented placeholders (N-015, N-027): the aim is the ratio between the two encodings and
which section dominates, not a prediction.

Run in Docker: docker run --rm -v "$PWD":/w -w /w python:3.12-slim python size.py
"""

import random

random.seed(20261011)


def uvar(v):
    n = 1
    while v >= 0x80:
        v >>= 7
        n += 1
    return n


def svar(v):
    return uvar((v << 1) ^ (v >> 63) if v >= 0 else ((-v) << 1) - 1)


class Count:
    def __init__(self):
        self.ints = 0
        self.var = 0
        self.raw = 0  # bytes that are not integers (hashes)

    def u(self, v):
        self.ints += 1
        self.var += uvar(v)

    def s(self, v):
        self.ints += 1
        self.var += svar(v)

    def b(self, n):
        self.raw += n

    @property
    def fixed(self):
        return self.ints * 8 + self.raw

    @property
    def varint(self):
        return self.var + self.raw


def agent_sections(n_agents, ta, window, tick, lots_per_commodity, recipes_per_role, modules):
    """Per-section counters for n_agents household agents (weight 1)."""
    secs = {}

    def sec(name):
        secs.setdefault(name, Count())
        return secs[name]

    for a in range(n_agents):
        # ledger: the agent account
        c = sec("ledger")
        c.u(1)  # kind
        c.u(a % 5)  # region
        c.u(0)  # closed
        c.u(a + 5)  # owner reference
        c.s(random.randint(0, 60000))  # money
        nlots = max(1, int(round(ta * lots_per_commodity)))
        c.u(nlots)
        for _ in range(nlots):
            c.u(random.randint(0, 29))  # commodity
            c.u(random.randint(1, 60))  # quantity
            perish = random.random() < 0.3
            c.s(tick - random.randint(0, 3) if perish else 0)  # born
            c.u(random.randint(1, 100) if random.random() < 0.1 else 0)  # durability
        # agents
        c = sec("agents")
        c.u(1)  # id delta
        c.u(1)  # account delta
        c.u(a % 6)  # role
        c.u(1)  # weight
        c.u(0)  # flags: external, mortal
        c.u(0)  # strategy name index
        c.u(random.randint(0, 2))  # unfed
        c.u(random.randint(0, 3))  # hunger
        c.u(random.randint(0, 20))  # privation
        c.s(tick - random.randint(0, tick))  # role since
        c.u(random.randint(0, 3650000) if "population" in modules else 0)  # death carry
        c.u(0)  # own use
        c.u(0)  # own use jobs
        c.u(0)  # journey flag
        c.u(0)  # directive flag
        # beliefs
        c = sec("beliefs")
        for _ in range(int(round(ta))):
            c.u(random.randint(50, 3000))  # centre
            c.u(random.randint(500, 2500))  # width
        # production
        c = sec("production")
        has_job = random.random() < 0.6
        c.u(1 if has_job else 0)
        if has_job:
            c.u(random.randint(0, 44))  # recipe
            c.u(random.randint(1, 60))  # remaining
            c.u(1)  # executions
            c.u(a + 5)  # principal
            c.u(2)  # pending items
            c.u(random.randint(1, 40))
            c.u(random.randint(1, 40))
        for _ in range(recipes_per_role * 2):  # residues
            c.u(random.randint(0, 9999))
        # welfare
        c = sec("welfare")
        c.u(window)  # records held
        c.u(random.randint(10000, 20000))  # target ratio
        for _ in range(window):
            c.s(random.randint(0, 3000) if random.random() > 0.3 else 0)  # income
            c.u(random.randint(800, 1200))  # cost
            c.u(0 if random.random() > 0.05 else random.randint(1, 500))  # relief
            c.u(0)  # grant
            c.u(0)  # own-use receipt
            c.u(0)  # own-use household-ticks
            c.u(1 if random.random() < 0.7 else 0)  # working flag
            c.u(1 if random.random() < 0.7 else 0)  # working household-ticks
        if "demand" in modules:
            c = sec("demand")
            for _ in range(3):
                c.u(random.randint(0, 9999))
    return secs


def fixed_sections(n_markets, n_sources, rumour_age, price_window, n_boundary, n_regions, modules, closed):
    secs = {}

    def sec(name):
        secs.setdefault(name, Count())
        return secs[name]

    c = sec("ledger")
    for _ in range(n_boundary):
        c.u(0)
        c.s(-random.randint(0, 10**6))
    for _ in range(closed):
        c.u(1)  # kind
        c.u(1)  # closed marker
    c = sec("markets")
    for _ in range(n_markets):
        c.u(random.randint(50, 3000))  # price
        c.u(random.randint(0, 5))  # age
        c.u(1)
        c.u(random.randint(50, 3000))  # best bid
        c.u(1)
        c.u(random.randint(50, 3000))  # best ask
        c.u(price_window)
        for _ in range(price_window):
            c.u(random.randint(50, 3000))
        c.u(random.randint(0, 5))  # bound run
    c = sec("estates")
    for _ in range(n_regions):
        c.u(20)
        for _ in range(20):
            c.u(random.randint(0, 29))
            c.s(0)
            c.u(0)
            c.u(random.randint(50, 3000))
    if "information" in modules:
        c = sec("information")
        for _ in range(n_sources):
            c.u(rumour_age)
            for _ in range(rumour_age):
                c.u(random.randint(50, 3000))
                c.u(random.randint(50, 3000))
                c.u(random.randint(0, 5))
    c = sec("world")
    c.u(10000)
    c.u(0)
    return secs


def report(title, n_agents, ta, window, tick, modules, closed=0, n_markets=85, n_regions=5):
    secs = agent_sections(n_agents, ta, window, tick, 1.5, 3, modules)
    fixed = fixed_sections(n_markets, n_markets, 60, 15, 250, n_regions, modules, closed)
    total_var = total_fix = 0
    print(f"\n{title}: {n_agents} agents, |T_a| {ta}, window {window}, modules {sorted(modules) or 'none'}, closed accounts {closed}")
    print(f"  {'section':12s} {'ints':>10s} {'varint B':>12s} {'fixed B':>12s}")
    allsecs = {}
    for d in (secs, fixed):
        for k, v in d.items():
            t = allsecs.setdefault(k, Count())
            t.ints += v.ints
            t.var += v.var
            t.raw += v.raw
    for k in sorted(allsecs):
        v = allsecs[k]
        print(f"  {k:12s} {v.ints:10d} {v.varint:12d} {v.fixed:12d}")
        total_var += v.varint
        total_fix += v.fixed
    hdr = 4 + 1 + 1 + 32 + 10 + 10 + 32  # magic, version, hash version, hash, tick, seed, trailer
    total_var += hdr
    total_fix += hdr
    ints = sum(v.ints for v in allsecs.values())
    print(f"  {'total':12s} {ints:10d} {total_var:12d} {total_fix:12d}   (fixed/varint = {total_fix / total_var:.1f})")
    return ints, total_var


if __name__ == "__main__":
    base = set()
    full = {"demand", "population", "information"}
    report("A. first consumer, default window", 5000, 7.7, 30, 3650, base)
    report("B. first consumer, long-job window", 5000, 7.7, 120, 3650, full)
    report("C. first consumer, modules, 10 000 deaths", 5000, 7.7, 30, 36500, full, closed=10000)
    report("D. full-detail realm", 200000, 8.0, 30, 3650, full)
