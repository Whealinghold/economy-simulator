"""Room arithmetic of spec-storage.md: a model of the RULES, not of the library.

Checks, with integers only and placeholder numbers (notes N-015, N-027):
  1. the closed-form start size (spec-storage.md 5.3.2) against the exact largest k, in ONE
     dimension: it never starts more than fits, and how far below the exact value it is;
  2. the target scaling of 6.2, including the clamp when the available room is negative;
  3. a random sequence on one limited account with an upkeep good: bids with reserved room, job
     starts and dispatches that withhold the upkeep room (5.3.2, 5.4.1), completions and
     arrivals from the pledge, the discard that makes room to eat (6.3.2): load + pledge +
     reserved never above the room, pledge + upkeep room never above the room, and the
     tier 1 bid always fits after the discard;
  4. a reduction of the weight (5.5.3): the scaled load always fits the new room, and a
     pledge that outlives the reduction is cut at the credit (5.3.4, 5.4.4);
  5. the one rule of 5.5.2 and 6.3.2 for what goes first (relative to the room, re-evaluated
     before each commodity), as random cases and the worked case of 18.3.
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
    """largest k <= k0 under the linear bound with the executor's actual residues, or 0.
    `free` is Free_d less the upkeep room U_d (the caller subtracts it, 5.3.2)."""
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
    big = []                      # cases whose gap is 3 or more
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
            if ke - kc >= 3:
                c = sum(q * E * s for q, s in outs) - D * sum(q * s for q, s in ins)
                big.append((ke - kc, c, free))
    return n, same, worse, gap_max, big


# ---- 2. the target scaling ---------------------------------------------------

def scale_targets(room, pledge, reserve, targets, sizes):
    """room, pledge: (mass, bulk); reserve and targets: dict commodity -> qty;
    sizes: dict commodity -> (mass, bulk). Returns the scaled targets and the factor in bp.
    Avail below 0 is clamped to 0 (6.2.2): f is then 0."""
    rload = tuple(sum(reserve.get(c, 0) * sizes[c][d] for c in sizes) for d in range(2))
    avail = tuple(max(0, room[d] - pledge[d] - rload[d]) for d in range(2))
    tl = tuple(sum(targets.get(c, 0) * sizes[c][d] for c in sizes) for d in range(2))
    f = D
    for d in range(2):
        if tl[d] > avail[d]:
            f = min(f, avail[d] * D // tl[d])
    return {c: t * f // D for c, t in targets.items()}, f


# ---- the one rule of what goes first (5.5.2, 6.3.2) ---------------------------

def binding(short, room):
    """the dimension with the largest shortfall relative to the room, by cross-multiplication;
    ties to mass. Only a dimension with a positive shortfall is considered."""
    best = None
    for d in range(2):
        if short[d] <= 0:
            continue
        if best is None or short[d] * max(room[best], 1) > short[best] * max(room[d], 1):
            best = d
    return best


def role_change_excess(room, sizes, lots):
    """5.5.2: move whole lots, oldest first, one commodity at a time, until the load fits.
    lots: dict commodity -> list of unit counts, oldest first. Returns dict commodity -> units."""
    moved = {}
    lots = {c: list(v) for c, v in lots.items()}
    while True:
        load = [sum(sum(v) * sizes[c][d] for c, v in lots.items()) for d in range(2)]
        short = [load[d] - room[d] for d in range(2)]
        d = binding(short, room)
        if d is None:
            return moved
        cand = [(sum(v) * sizes[c][d], c) for c, v in lots.items() if sum(v) > 0 and sizes[c][d] > 0]
        assert cand, "a positive shortfall has a load behind it"
        top = max(l for l, _ in cand)
        c = min(c for l, c in cand if l == top)
        need = short[d]
        while need > 0 and lots[c]:
            u = lots[c].pop(0)
            moved[c] = moved.get(c, 0) + u
            need -= u * sizes[c][d]


def discard_for(room, load, pledge, sizes, held, keep, need_units):
    """6.3.2: discard whole units of the surplus above `keep`, relative to the room, re-evaluated
    before each commodity, until `need_units` (commodity -> units, the tier 1 bids) fit the free room."""
    held = dict(held)
    out = {}
    while True:
        ld = [sum(held[c] * sizes[c][d] for c in held) for d in range(2)]
        nd = [sum(need_units.get(c, 0) * sizes[c][d] for c in sizes) for d in range(2)]
        short = [nd[d] - (room[d] - ld[d] - pledge[d]) for d in range(2)]
        d = binding(short, room)
        if d is None:
            return held, out
        sur = {c: held[c] - keep.get(c, 0) for c in held if held[c] > keep.get(c, 0) and sizes[c][d] > 0}
        assert sur, "6.3.3: the surplus always covers the shortfall"
        top = max(u * sizes[c][d] for c, u in sur.items())
        c = min(c for c, u in sur.items() if u * sizes[c][d] == top)
        n = min(sur[c], ceil_div(short[d], sizes[c][d]))
        held[c] -= n
        out[c] = out.get(c, 0) + n


# ---- 3. a random sequence on one limited account ------------------------------

def part3(rng, ticks=4000):
    sizes = {0: (2, 1), 1: (1, 3), 2: (5, 5)}
    room = (60, 60)
    U = 2                          # units of commodity 0 eaten at the end of each tick
    Ud = tuple(U * sizes[0][d] for d in range(2))          # the upkeep room (5.3.2)
    stock = {0: U, 1: 0, 2: 0}
    pledge = [0, 0]
    job = None                     # (remaining, pending dict)
    inbound = []                   # (arrive, qty of commodity 2)
    refused = {"bid": 0, "start": 0, "dispatch": 0}
    admitted = settled = discarded = tier1 = 0
    maxfill = 0

    def load(d):
        return sum(stock[c] * sizes[c][d] for c in stock)

    def check(committed=False):
        for d in range(2):
            assert load(d) + pledge[d] <= room[d], (load(d), pledge[d], room[d])
            if committed:
                assert pledge[d] + Ud[d] <= room[d], (pledge[d], Ud[d], room[d])

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
        # job start: 1 execution of 1 unit of commodity 2 -> outq units of commodity 1
        if job is None and rng.random() < 0.5 and stock[2] >= 1:
            outq = rng.randint(1, 9)
            free = [room[d] - load(d) - pledge[d] for d in range(2)]
            need = [outq * sizes[1][d] - sizes[2][d] for d in range(2)]   # inputs freed
            if all(need[d] <= free[d] - Ud[d] for d in range(2)):          # the upkeep room is withheld
                stock[2] -= 1
                for d in range(2):
                    pledge[d] += outq * sizes[1][d]
                job = (rng.randint(2, 4), {1: outq})
            else:
                refused["start"] += 1
        check(True)
        # a dispatch to oneself as consignee: pledge at dispatch, upkeep room withheld
        if rng.random() < 0.15:
            q = rng.randint(1, 4)
            free = [room[d] - load(d) - pledge[d] for d in range(2)]
            if all(q * sizes[2][d] <= free[d] - Ud[d] for d in range(2)):
                for d in range(2):
                    pledge[d] += q * sizes[2][d]
                inbound.append((t + rng.randint(1, 3), q))
            else:
                refused["dispatch"] += 1
        check(True)
        # tier 1: the units of commodity 0 short of the upkeep; the discard makes room for them
        need = max(0, U - stock[0])
        if need:
            held, out = discard_for(room, None, pledge, sizes, stock, {0: U}, {0: need})
            for c, n in out.items():
                stock[c] -= n
                discarded += n
            free = [room[d] - load(d) - pledge[d] for d in range(2)]
            assert all(need * sizes[0][d] <= free[d] for d in range(2)), "tier 1 must fit (6.3.3)"
            stock[0] += need            # the tier 1 bid is admitted and settles whole
            tier1 += 1
        # the other bids: reserve room; settle part
        reserved = [0, 0]
        orders = []
        for c in (0, 1, 2):
            if rng.random() < 0.9:
                q = rng.randint(1, 14)       # heavy buying, so the room is often full of surplus
                free = [room[d] - load(d) - pledge[d] - reserved[d] for d in range(2)]
                if all(q * sizes[c][d] <= free[d] for d in range(2)):
                    for d in range(2):
                        reserved[d] += q * sizes[c][d]
                    orders.append((c, q))
                    admitted += 1
                else:
                    refused["bid"] += 1
        for c, q in orders:
            f = rng.randint(q // 2, q)              # units that trade
            stock[c] += f                           # a settlement from the reservation: never fails
            settled += 1
            for d in range(2):
                reserved[d] -= q * sizes[c][d]      # release what did not trade
        assert reserved == [0, 0]
        check()
        # consumption (the upkeep) and spoilage free room
        assert stock[0] >= U, "the household is never unfed for room"
        stock[0] -= U
        for c in stock:
            if c and rng.random() < 0.3:
                stock[c] -= min(stock[c], rng.randint(0, 2))
            if rng.random() < 0.05:
                stock[c] -= stock[c] // 3
        check()
        maxfill = max(maxfill, max(load(d) + pledge[d] for d in range(2)))
    return admitted, settled, refused, maxfill, discarded, tier1


# ---- 4. a reduction of the weight ---------------------------------------------

def part4(rng, n=20000):
    """5.5.3: the scaled load fits the new room whenever load + pledge fit the old one."""
    sizes = {0: (2, 1), 1: (1, 3), 2: (3, 2)}
    for _ in range(n):
        w = rng.randint(2, 30)
        w2 = rng.randint(1, w - 1)
        base = (rng.randint(1, 40), rng.randint(1, 40))
        store = (rng.randint(0, 500), rng.randint(0, 500))
        room = tuple(w * base[d] + store[d] for d in range(2))
        held = {c: rng.randint(0, 60) for c in sizes}
        pledge = tuple(rng.randint(0, 200) for _ in range(2))
        ld = tuple(sum(held[c] * sizes[c][d] for c in sizes) for d in range(2))
        if any(ld[d] + pledge[d] > room[d] for d in range(2)):
            continue                                 # the invariant does not hold before: not a case
        held2 = {c: q * w2 // w for c, q in held.items()}
        ld2 = tuple(sum(held2[c] * sizes[c][d] for c in sizes) for d in range(2))
        room2 = tuple(w2 * base[d] + store[d] for d in range(2))
        assert all(ld2[d] <= room2[d] for d in range(2)), (w, w2, base, store, held, pledge)
    # the case of 18.4: a job's pledge and an inbound cargo outlive the reduction
    sizes = {"A": (2, 1), "B": (2, 2), "C": (1, 1)}
    w, w2, base, store = 10, 5, (100, 100), (200, 200)
    room = tuple(w * base[d] + store[d] for d in range(2))
    held = {"A": 100}
    pend_b, cargo_c = 450, 40                       # one execution pending 450 B, 40 C inbound
    ld = tuple(held["A"] * sizes["A"][d] for d in range(2))
    pl = tuple(pend_b * sizes["B"][d] + cargo_c * sizes["C"][d] for d in range(2))
    assert all(ld[d] + pl[d] <= room[d] for d in range(2))
    room2 = tuple(w2 * base[d] + store[d] for d in range(2))
    held2 = {"A": held["A"] * w2 // w}
    ld2 = tuple(held2["A"] * sizes["A"][d] for d in range(2))
    rows = [("before", room, ld, pl), ("after", room2, ld2, pl)]
    # P1.3: the cargo is credited first, as far as the room goes; P2.1: then the output
    free = [room2[d] - ld2[d] for d in range(2)]
    c_in = min(cargo_c, *(free[d] // sizes["C"][d] for d in range(2)))
    for d in range(2):
        free[d] -= c_in * sizes["C"][d]
    b_in = min(pend_b, *(free[d] // sizes["B"][d] for d in range(2)))
    return rows, c_in, cargo_c - c_in, b_in, pend_b - b_in


# ---- 5. what goes first -------------------------------------------------------

def part5(rng, n=5000):
    sizes = {0: (1, 3), 1: (3, 1), 2: (2, 2), 3: (1, 1)}
    for _ in range(n):
        room = (rng.randint(0, 120), rng.randint(0, 120))
        lots = {c: [rng.randint(1, 12) for _ in range(rng.randint(0, 4))] for c in sizes}
        moved = role_change_excess(room, sizes, lots)
        left = {c: sum(lots[c]) - moved.get(c, 0) for c in sizes}
        assert all(sum(left[c] * sizes[c][d] for c in sizes) <= room[d] for d in range(2))
    # 6.3.2 and 6.3.3: whenever pledge + the load of `keep` fits the room, the discard of the surplus
    # makes the tier 1 units fit
    sizes = {0: (2, 1), 1: (1, 3), 2: (5, 5)}
    cases = discards = 0
    for _ in range(n):
        room = (rng.randint(10, 100), rng.randint(10, 100))
        keep = {0: rng.randint(0, 3), 1: rng.randint(0, 3), 2: rng.randint(0, 2)}
        held = {c: keep[c] + rng.randint(0, 12) for c in sizes}
        pledge = (rng.randint(0, 30), rng.randint(0, 30))
        need = {c: rng.randint(0, 2) for c in sizes}
        ldk = tuple(sum((keep[c] + need[c]) * sizes[c][d] for c in sizes) for d in range(2))
        if any(pledge[d] + ldk[d] > room[d] for d in range(2)):
            continue
        ld = tuple(sum(held[c] * sizes[c][d] for c in sizes) for d in range(2))
        if any(ld[d] + pledge[d] > room[d] for d in range(2)):
            continue
        cases += 1
        left, out = discard_for(room, None, pledge, sizes, held, keep, need)
        discards += sum(out.values())
        for d in range(2):
            assert sum(left[c] * sizes[c][d] for c in sizes) + sum(need[c] * sizes[c][d] for c in sizes) \
                + pledge[d] <= room[d]
    # a deterministic case: room (60, 60); 10 of commodity 1 and 6 of commodity 2 held, tier 1 wants 2 of commodity 0
    held, out = discard_for((60, 60), None, (0, 0), sizes, {0: 0, 1: 10, 2: 6}, {}, {0: 2})
    dcase = out
    # the worked case of 18.3: A (1, 3) in three lots of 10, B (3, 1) in four lots of 10, room (100, 100)
    sizes = {"A": (1, 3), "B": (3, 1)}
    moved = role_change_excess((100, 100), sizes, {"A": [10, 10, 10], "B": [10, 10, 10, 10]})
    return moved, cases, discards, dcase


def worked():
    out = []
    # Example A (a farm setting, flagged in the spec): base room {mass 400, bulk 200} per household.
    # grain mass 2 bulk 1; upkeep 3 grain a tick, so the upkeep room is (6, 3); a growing recipe
    # makes 120 grain per execution (E 10 000), no inputs.
    room = (400, 200)
    Ud = (6, 3)
    for held in (0, 100, 160, 190):
        load = (held * 2, held * 1)
        free = tuple(room[d] - load[d] for d in range(2))
        avail = tuple(free[d] - Ud[d] for d in range(2))
        kd = [k_closed(1, avail[d], [], [(120, (2, 1)[d])], 10_000, [4_000]) for d in range(2)]
        out.append(("A", held, "free", free, "free less U", avail, "R", tuple(D * a for a in avail), "k", kd, min(kd)))
    # Example B (a space colony setting, flagged): a steward role with base room {40, 60} per household
    # and one store of class `depot` {mass 6 000, bulk 9 000}; a ration unit has mass 1 and bulk 2;
    # the upkeep is 2 rations a tick, so the upkeep room is (2, 4).
    room = (40 + 6_000, 60 + 9_000)
    size = (1, 2)
    Ud = (2, 4)
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
    f = free()
    ok = all(20 * size[d] <= f[d] - Ud[d] for d in range(2))        # the dispatch withholds the upkeep room
    assert ok
    for d in range(2):
        pledge[d] += 20 * size[d]
    out.append(("B", "cargo of 20 pledged", "free less U before", tuple(f[d] - Ud[d] for d in range(2)), "free after", free()))
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
    n, same, worse, gap, big = part1(rng)
    print(f"1. start size (one dimension): {n} cases, closed form equals the exact largest k in {same}, "
          f"below it in {worse} (largest gap {gap} executions); it never started more than fits")
    print(f"   cases with a gap of 3 or more: {len(big)}; (gap, c, free): {sorted(big, reverse=True)}")
    tg, f = scale_targets((100, 100), (10, 10), {0: 6}, {0: 40, 1: 30}, {0: (2, 1), 1: (1, 3)})
    print(f"2. targets {{0: 40, 1: 30}} scaled by {f} bp to {tg} in room (100, 100) with pledge (10, 10) and reserve {{0: 6}}")
    tg, f = scale_targets((100, 100), (95, 10), {0: 6}, {0: 40, 1: 30}, {0: (2, 1), 1: (1, 3)})
    print(f"   with pledge (95, 10) the available mass room is negative: clamped, f = {f}, targets {tg}")
    a, s, r, m, dsc, t1 = part3(rng)
    print(f"3. 4000 ticks, upkeep 2 units a tick: {a} bids admitted, {s} settled whole, {r['bid']} bids, "
          f"{r['start']} starts and {r['dispatch']} dispatches refused for room; {t1} tier 1 bids needed, "
          f"all fitted after {dsc} units discarded; never unfed for room; highest committed load {m} of room 60")
    rows, c_in, c_div, b_in, b_disp = part4(rng)
    print("4. weight 10 -> 5, 20 000 random reductions: the scaled load always fits the new room")
    for name, rm, ld, pl in rows:
        print(f"   {name}: room {rm}, load {ld}, pledge {pl}")
    print(f"   at P1.3 the cargo credits {c_in} of 40 (diverts {c_div}); at P2.1 the output credits {b_in} of 450 "
          f"and disposes {b_disp}")
    moved, cases, discards, dcase = part5(rng)
    print(f"5. 5000 random role changes: the excess always ends within the room; worked case of 18.3, units moved to the estate: {moved}")
    print(f"6. {cases} random discards: the tier 1 units always fit after the discard ({discards} units discarded in all); the case of 6.3.2, room (60, 60) full of surplus: discarded {dcase}")
    for row in worked():
        print("A", row)
