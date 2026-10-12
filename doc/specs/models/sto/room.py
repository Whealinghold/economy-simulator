"""Room arithmetic of spec-storage.md: a model of the RULES, not of the library.

Checks, with integers only and placeholder numbers (notes N-015, N-027):
  1. the closed-form start size (spec-storage.md 5.3.2) against the exact largest k:
     it never starts more than fits, and how far below the exact value it is;
  2. the target scaling of 6.2 (a target never needs more room than there is);
  3. a random sequence of bids, settlements, job starts and completions, arrivals,
     consumption and spoilage on one limited account: load + pledge + reserved never
     exceeds room, a completion and an arrival never fail, a bid admitted always settles.
Never run with a host interpreter: docker only (see README.md).
"""
import random

D = 10_000


def ceil_div(a, b):
    return -((-a) // b)


# ---- 1. the start size -------------------------------------------------------

def pend(k, q, E, r):
    """pending units of one output for k executions: carried remainder, floor."""
    return (k * q * E + r) // D


def fits_exact(k, free, ins, outs, E, res):
    """exact test, one dimension: pending output of k executions against the free room
    plus the inputs the start frees. ins: list of (qty, size); outs: list of (qty, size)."""
    freed = k * sum(q * s for q, s in ins)
    out = sum(pend(k, q, E, r) * s for (q, s), r in zip(outs, res))
    return out <= free + freed


def k_closed(k0, free, ins, outs, E, res):
    """largest k <= k0 under the linear bound with the executor's actual residues, or 0."""
    I = sum(q * s for q, s in ins)
    A = sum(q * E * s for q, s in outs)
    c = A - D * I
    # an output whose q * E is a multiple of D has no remainder to carry: produced is exactly k * q * E / D
    R = D * free - sum(s * (0 if (q * E) % D == 0 else r) for (q, s), r in zip(outs, res))
    hi, lo = k0, 1
    if c > 0:
        if R < 0:
            return 0
        hi = min(hi, R // c)
    elif R < 0:
        if c == 0:
            return 0
        lo = max(lo, ceil_div(-R, -c))
    return hi if hi >= lo else 0


def k_exact(k0, free, ins, outs, E, res):
    best = 0
    for k in range(1, k0 + 1):
        if fits_exact(k, free, ins, outs, E, res):
            best = k
    return best


def part1(rng, n=20000):
    worse = same = 0
    gap_max = 0
    for _ in range(n):
        ins = [(rng.randint(0, 5), rng.randint(0, 4)) for _ in range(rng.randint(0, 3))]
        outs = [(rng.randint(1, 8), rng.randint(0, 4)) for _ in range(rng.randint(1, 3))]
        E = rng.choice([5_000, 8_000, 10_000, 12_500])
        res = [rng.randrange(D) for _ in outs]
        k0 = rng.randint(1, 12)
        free = rng.randint(0, 80)
        kc = k_closed(k0, free, ins, outs, E, res)
        ke = k_exact(k0, free, ins, outs, E, res)
        # soundness: kc executions fit exactly, with the executor's own residues
        if kc:
            assert fits_exact(kc, free, ins, outs, E, res), (k0, free, ins, outs, E, res)
        assert kc <= k0
        if kc == ke:
            same += 1
        else:
            worse += 1
            gap_max = max(gap_max, ke - kc)
    return n, same, worse, gap_max


# ---- 2. the target scaling ---------------------------------------------------

def scale_targets(room, load, pledge, reserve, targets, sizes):
    """room, load, pledge: (mass, bulk); reserve and targets: dict commodity -> qty;
    sizes: dict commodity -> (mass, bulk). Returns the scaled targets and the factor in bp."""
    free0 = tuple(room[d] - pledge[d] for d in range(2))
    rload = tuple(sum(reserve.get(c, 0) * sizes[c][d] for c in sizes) for d in range(2))
    avail = tuple(free0[d] - rload[d] for d in range(2))
    assert min(avail) >= 0, "room below upkeep: a validation error"
    tl = tuple(sum(targets.get(c, 0) * sizes[c][d] for c in sizes) for d in range(2))
    f = D
    for d in range(2):
        if tl[d] > avail[d]:
            f = min(f, avail[d] * D // tl[d])
    return {c: t * f // D for c, t in targets.items()}, f


# ---- 3. a random sequence on one limited account -----------------------------

def part3(rng, ticks=4000):
    sizes = {0: (2, 1), 1: (1, 3), 2: (5, 5)}
    room = (60, 60)
    stock = {0: 0, 1: 0, 2: 0}
    pledge = [0, 0]
    job = None            # (remaining, pending dict)
    inbound = []          # (arrive, qty commodity 2)
    refused = {"bid": 0, "start": 0}
    admitted = settled = 0
    maxfill = 0

    def load(d):
        return sum(stock[c] * sizes[c][d] for c in stock)

    def check():
        for d in range(2):
            assert load(d) + pledge[d] <= room[d], (load(d), pledge[d], room[d])

    for t in range(ticks):
        # arrivals (marked credit from the pledge: never fails)
        for a in [x for x in inbound if x[0] == t]:
            inbound.remove(a)
            for d in range(2):
                pledge[d] -= a[1] * sizes[2][d]
            stock[2] += a[1]
        # job completion
        if job and job[0] == 1:
            for c, q in job[1].items():
                for d in range(2):
                    pledge[d] -= q * sizes[c][d]
                stock[c] += q
            job = None
        elif job:
            job = (job[0] - 1, job[1])
        check()
        # job start: 1 execution of 1 unit of commodity 2 -> 2 units of commodity 1
        if job is None and rng.random() < 0.5:
            ins = [(1, sizes[2])] if stock[2] >= 1 else None
            if ins is not None:
                outq = rng.randint(1, 9)
                free = [room[d] - load(d) - pledge[d] for d in range(2)]
                need = [outq * sizes[1][d] - sizes[2][d] for d in range(2)]   # inputs freed
                if all(need[d] <= free[d] for d in range(2)):
                    stock[2] -= 1
                    for d in range(2):
                        pledge[d] += outq * sizes[1][d]
                    job = (rng.randint(2, 4), {1: outq})
                else:
                    refused["start"] += 1
        check()
        # a dispatch to self-hired consignee: pledge at dispatch
        if rng.random() < 0.15:
            q = rng.randint(1, 4)
            free = [room[d] - load(d) - pledge[d] for d in range(2)]
            if all(q * sizes[2][d] <= free[d] for d in range(2)):
                for d in range(2):
                    pledge[d] += q * sizes[2][d]
                inbound.append((t + rng.randint(1, 3), q))
        # bids: reserve room; settle part
        reserved = [0, 0]
        orders = []
        for c in (0, 1, 2):
            if rng.random() < 0.6:
                q = rng.randint(1, 10)
                free = [room[d] - load(d) - pledge[d] - reserved[d] for d in range(2)]
                if all(q * sizes[c][d] <= free[d] for d in range(2)):
                    for d in range(2):
                        reserved[d] += q * sizes[c][d]
                    orders.append((c, q))
                    admitted += 1
                else:
                    refused["bid"] += 1
        for c, q in orders:
            f = rng.randint(0, q)                   # units that trade
            stock[c] += f                           # a settlement from the reservation: never fails
            settled += 1
            for d in range(2):
                reserved[d] -= q * sizes[c][d]      # release what did not trade
        assert reserved == [0, 0]
        check()
        # consumption and spoilage free room
        for c in stock:
            eat = min(stock[c], rng.randint(0, 2))
            stock[c] -= eat
            if rng.random() < 0.1:
                stock[c] -= stock[c] // 3
        check()
        maxfill = max(maxfill, max(load(d) + pledge[d] for d in range(2)))
    return admitted, settled, refused, maxfill


def worked():
    out = []
    # Example A (a farm setting, flagged in the spec): base room {mass 400, bulk 200} per household.
    # grain mass 2 bulk 1; a growing recipe makes 120 grain per execution (E 10 000), no inputs.
    room = (400, 200)
    for held in (0, 100, 160, 190):
        load = (held * 2, held * 1)
        free = tuple(room[d] - load[d] for d in range(2))
        kd = [k_closed(1, free[d], [], [(120, (2, 1)[d])], 10_000, [4_000]) for d in range(2)]
        out.append(("A", held, free, kd, min(kd)))
    # Example B (a space colony setting, flagged): a steward role with base room {40, 60} per household
    # and one store of class `depot` {mass 6 000, bulk 9 000}; a ration unit has mass 1 and bulk 2.
    room = (40 + 6_000, 60 + 9_000)
    size = (1, 2)
    load = [0, 0]
    pledge = [0, 0]
    out.append(("B", "units that fit", min(room[0] // size[0], room[1] // size[1]), "room", room))

    def free():
        return tuple(room[d] - load[d] - pledge[d] for d in range(2))

    for step, qty in (("bid", 4_500), ("bid", 40)):
        f = free()
        ok = all(qty * size[d] <= f[d] for d in range(2))
        if ok:
            for d in range(2):
                load[d] += qty * size[d]
        out.append(("B", step, qty, "admitted" if ok else "rejected no_room", "free before", f, "free after", free()))
    for d in range(2):                      # a cargo of 20 rations is dispatched to the steward
        pledge[d] += 20 * size[d]
    out.append(("B", "cargo of 20 pledged", "free", free()))
    for step, qty in (("bid", 11), ("bid", 10)):
        f = free()
        ok = all(qty * size[d] <= f[d] for d in range(2))
        if ok:
            for d in range(2):
                load[d] += qty * size[d]
        out.append(("B", step, qty, "admitted" if ok else "rejected no_room", "free before", f, "free after", free()))
    return out


if __name__ == "__main__":
    rng = random.Random(1)
    n, same, worse, gap = part1(rng)
    print(f"1. start size: {n} cases, closed form equals the exact largest k in {same}, "
          f"below it in {worse} (largest gap {gap} executions); it never started more than fits")
    tg, f = scale_targets((100, 100), (0, 0), (10, 10), {0: 6}, {0: 40, 1: 30}, {0: (2, 1), 1: (1, 3)})
    print(f"2. targets {{0: 40, 1: 30}} scaled by {f} bp to {tg} in room (100, 100) with pledge (10, 10) and reserve {{0: 6}}")
    a, s, r, m = part3(rng)
    print(f"3. 4000 ticks: {a} bids admitted, {s} settled whole, {r['bid']} bids and {r['start']} starts refused "
          f"for room, the highest committed load {m} of room 60")
    for row in worked():
        print("A", row)
