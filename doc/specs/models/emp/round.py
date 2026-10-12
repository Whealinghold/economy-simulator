"""The labour round of spec-employment.md sections 5 and 6, with the ledger checks.

A prototype of the spec's rules, not of the library. Standard library only. Integer
arithmetic throughout; the ties of the round are broken by the stateless key K(t, id)
of spec-world.md 10.1.1 (SplitMix64 finaliser of t * 0x9E3779B97F4A7C15 + id, smaller
first, then the ID): no word of the world's source is taken (spec-employment.md 5.3.1).
A random.Random is used only to build the scenarios and to choose interruptions.

It implements: the order by value then K(t, id) (5.3), first-fit matching with the wage
`R + phi * (Bid - R)` half to even (5.4), formation with its refusals, the closing of a
lane or a vacancy and the **fit** of a vacancy (5.5), the escrow, the release of one
tick's wage at P2.1 or P2.7 and the completion with the refund (6.2, 6.3), and three
interruptions (6.4): the worker removed, the employer removed (escrow to the estate),
and, in `scenario`, a random mix. After every step it checks that the money of all
accounts (workers, employers, escrows, estate) is constant, that no balance is negative
and that an escrow holds at least what 6.2 says.
"""
import random

def half_even_div(n, d):
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q

M64 = (1 << 64) - 1

def finalise(x):
    """The SplitMix64 finaliser (modular uint64 arithmetic: the wrap is the hash)."""
    x &= M64
    x = ((x ^ (x >> 30)) * 0xBF58476D1CE4E5B9) & M64
    x = ((x ^ (x >> 27)) * 0x94D049BB133111EB) & M64
    return x ^ (x >> 31)

def K(t, agent_id):
    """The stateless tie key of spec-world.md 10.1.1; takes no draw and keeps no state."""
    return finalise(t * 0x9E3779B97F4A7C15 + agent_id)

def agent_id(name):
    """Employers E<i> have ID i, workers W<i> have ID 1000 + i."""
    return int(name[1:]) + (1000 if name[0] == "W" else 0)

class World:
    def __init__(self, seed):
        self.tick = seed                 # the tick of the scenario: any value, K depends on it
        self.rng = random.Random(seed)
        self.money = {}          # account -> balance (agents, "estate", escrows)
        self.hires = {}          # id -> dict
        self.next_hire = 1
        self.total = 0

    def open(self, name, amount):
        self.money[name] = amount
        self.total += amount

    def move(self, a, b, amt):
        assert amt >= 1
        assert self.money[a] >= amt, (a, self.money[a], amt)
        self.money[a] -= amt
        self.money[b] += amt

    def check(self):
        assert sum(self.money.values()) == self.total
        assert all(v >= 0 for v in self.money.values())
        for h in self.hires.values():
            need = h["wage"] * h["k"] * h["left"]
            assert self.money[h["escrow"]] >= need, (h, self.money[h["escrow"]])

def round_(w, offers, vacancies, stock_capacity, max_open):
    """offers: list of dicts (agent, households, reserve); vacancies: list of dicts
    (agent, positions, bid, share_bp, duration). stock_capacity[agent] = executions the
    employer's stock and tools allow beyond its own request. One region (one lane).
    Returns the hires formed and the counts (households unmatched; positions unfilled by
    the reason that closed the vacancy)."""
    for o in offers:                                   # 5.3.1: the stateless key, no draw
        o["key"] = (K(w.tick, agent_id(o["agent"])), agent_id(o["agent"]))
    for v in vacancies:
        v["key"] = (K(w.tick, agent_id(v["agent"])), agent_id(v["agent"]))
        v["left"] = v["positions"]
        v["fit"] = v["positions"]                      # the largest block it can still take
        v["closed"] = None
    offers = sorted(offers, key=lambda o: (o["reserve"], o["key"]))
    vacs = sorted(vacancies, key=lambda v: (-v["bid"], v["key"]))
    formed = []
    counts = dict(no_vacancy=0, employer_funds=0, no_input=0, too_many=0, below_unit=0)
    lane_closed = False
    for o in offers:
        m = o["households"]
        placed = False
        for v in vacs:
            if lane_closed:
                break
            if v["closed"] or v["bid"] < o["reserve"] or v["left"] < m or m > v["fit"]:
                continue
            wage = o["reserve"] + half_even_div(v["share_bp"] * (v["bid"] - o["reserve"]), 10000)
            if wage < 1:
                counts["below_unit"] += 1              # the pair only
                continue
            if len(w.hires) >= max_open:               # check 1: the lane is closed
                counts["too_many"] += v["left"]
                lane_closed = True
                break
            reason = None
            if stock_capacity.get(v["agent"], 0) < m:  # checks 3 and 4 (the block)
                reason = "no_input"
            elif w.money[v["agent"]] < wage * v["duration"] * m:   # check 6 (the block)
                reason = "employer_funds"
            if reason:
                v["fit"] = min(v["fit"], m - 1)
                if v["fit"] == 0 and not v["closed"]:
                    v["closed"] = reason
                    counts[reason] += v["left"]        # the positions counted once
                continue
            bill = wage * v["duration"] * m
            esc = "escrow%d" % w.next_hire
            w.open(esc, 0)
            w.move(v["agent"], esc, bill)
            h = dict(id=w.next_hire, employer=v["agent"], worker=o["agent"], k=m, wage=wage,
                     left=v["duration"], escrow=esc, reserve=o["reserve"], bid=v["bid"])
            assert o["reserve"] <= wage <= v["bid"]
            w.hires[w.next_hire] = h
            w.next_hire += 1
            stock_capacity[v["agent"]] -= m
            v["left"] -= m
            formed.append(h)
            placed = True
            break
        if not placed:
            counts["no_vacancy"] += m
    return formed, counts

def release(w, h):
    """one working tick: P2.1 for a continued or completing job, P2.7 for the start tick"""
    w.move(h["escrow"], h["worker"], h["wage"] * h["k"])
    h["left"] -= 1
    if h["left"] == 0:
        rem = w.money[h["escrow"]]
        if rem:
            w.move(h["escrow"], h["employer"], rem)
        del w.hires[h["id"]]
    w.check()

def end_by_worker(w, h):
    rem = w.money[h["escrow"]]
    if rem:
        w.move(h["escrow"], h["employer"], rem)
    del w.hires[h["id"]]
    w.check()

def end_by_employer_removed(w, h):
    rem = w.money[h["escrow"]]
    if rem:
        w.move(h["escrow"], "estate", rem)
    del w.hires[h["id"]]
    w.check()

def scenario(seed):
    rng = random.Random(seed * 7919 + 1)
    w = World(seed)
    w.open("estate", 0)
    n_emp = rng.randint(1, 4)
    n_wrk = rng.randint(1, 12)
    emps = ["E%d" % i for i in range(n_emp)]
    wrks = ["W%d" % i for i in range(n_wrk)]
    for e in emps:
        w.open(e, rng.randint(0, 5000))
    for x in wrks:
        w.open(x, rng.randint(0, 300))
    offers = [dict(agent=x, households=rng.choice([1, 1, 1, 2, 3]), reserve=rng.choice([0, 0, 40, 100]))
              for x in wrks]
    vacancies = [dict(agent=e, positions=rng.randint(1, 5), bid=rng.randint(1, 900),
                      share_bp=rng.choice([3000, 5000, 7500, 10000]),
                      duration=rng.choice([1, 1, 2, 5])) for e in emps]
    cap = {e: rng.randint(0, 6) for e in emps}
    formed, counts = round_(w, offers, vacancies, cap, max_open=rng.choice([3, 1000]))
    w.check()
    for h in list(formed):                              # run the jobs with random interruptions
        while h["id"] in w.hires:
            r = rng.random()
            if r < 0.08:
                end_by_worker(w, h)
            elif r < 0.12:
                end_by_employer_removed(w, h)
            else:
                release(w, h)
    assert all(w.money[k] == 0 for k in w.money if k.startswith("escrow"))
    w.check()
    return len(formed), counts

if __name__ == "__main__":
    formed_total = 0
    agg = dict(no_vacancy=0, employer_funds=0, no_input=0, too_many=0, below_unit=0)
    for s in range(1, 501):
        n, c = scenario(s)
        formed_total += n
        for k in agg:
            agg[k] += c[k]
    print("500 scenarios: hires formed", formed_total, "| unmatched households / positions closed by reason:", agg)
    print("every step: total money constant, balances non-negative, escrow >= wage * k * ticks left, wage within [reserve, bid], escrows empty at the end")
