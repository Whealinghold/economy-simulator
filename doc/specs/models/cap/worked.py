"""Models behind spec-infrastructure.md sections 3 to 6, 16 and 18.

A model of the RULES of the spec, not of the library (there is no library): integer
arithmetic, placeholder numbers (notes N-015, N-027); nothing here shows that a default
is stable. Standard library only. Run in Docker, never with a local interpreter.
"""
import random

MAX_TF = 100_000      # MaxTransitFactorBP (spec-transport.md 11), a placeholder
MAX_TOTAL_BOOST = 200_000
MAX_ENDOWMENT = 10**15  # MaxEndowment (spec-resources.md), a placeholder


def eff(boost, cond, dur):                  # 3.3.2, a gain: floor
    return boost * cond // dur


def capf(capacity, cond, dur):              # 3.3.2, a capacity: ceil, 0 when ruined
    return 0 if cond <= 0 else -(-capacity * cond // dur)


def f_eff(f_slot, total_boost):             # 5.2.2
    if f_slot == 0:
        return 0
    b = min(total_boost, MAX_TOTAL_BOOST)
    return min(MAX_TF, f_slot * (10_000 + b) // 10_000)


def travel(d, v, f):                        # spec-transport.md 4.1
    return max(1, -(-d * 10_000 // (v * f)))


def cap_eff(capacity, scale, total_boost):  # 5.3.2, one truncation
    b = min(total_boost, MAX_TOTAL_BOOST)
    return min(MAX_ENDOWMENT, capacity * scale * (10_000 + b) // 10**8)


def assign(facs, served, cond, n, wear_per_use):
    """5.1.3: n executions to facilities most worn first by the TABLE condition (least
    condition, then lower ID), each at most what is left of its table cap(f) less served(f),
    the executions earlier starts of the tick put on it; wear goes to the running condition.
    Returns the executions placed."""
    order = sorted(facs, key=lambda f: (f["cond"], f["id"]))
    left = n
    for f in order:
        room = capf(f["cap"], f["cond"], f["dur"]) - served[f["id"]]
        take = min(left, max(0, room))
        left -= take
        served[f["id"]] += take
        cond[f["id"]] = max(0, cond[f["id"]] - take * wear_per_use)
    return n - left


def layer(edge_dist, shortest):
    """5.2.5: the facility layer of a pair applies only where the pair's own direct edge is
    its shortest path (the edge exists and its distance equals the pair's shortest distance)."""
    return edge_dist is not None and edge_dist == shortest


def part1_route():
    print("== 1. route: the travel time of a 60-unit edge at speed 20, Durability 2000, BoostBP 6000 (18.2)")
    print("   F_slot  state                                          B      F_eff   k")
    rows = [
        (10_000, [], "no road"),
        (10_000, [2000], "one road at 2000 (AddFacilities)"),
        (10_000, [1999], "one road built (1999 in the table)"),
        (10_000, [1000], "one road at half (1000)"),
        (10_000, [2000, 2000], "two roads at 2000"),
        (5_000, [2000], "one road at 2000, F_slot 5000 (damaged)"),
        (0, [2000], "one road at 2000, F_slot 0 (closed)"),
    ]
    for fs, conds, label in rows:
        b = sum(eff(6000, c, 2000) for c in conds)
        fe = f_eff(fs, b)
        k = "blocked" if fe == 0 else travel(60, 20, fe)
        print(f"   {fs:>6}  {label:<44} {b:>6} {fe:>7}   {k}")
    # the last condition at which k is still 2 (a built road: 1999 at its first table)
    c = 1999
    while travel(60, 20, f_eff(10_000, eff(6000, c, 2000))) == 2:
        c -= 1
    print(f"   k is 2 down to condition {c + 1} and 3 at {c}: {1999 - (c + 1)} ticks of decay after the first table")


def part1b_pairs():
    print("== 1b. which pair a layer applies to (5.2.5): road of 2000 on each direct edge, speed 20")
    print("   pair                      edge   shortest  layer  B      k")
    cases = [
        ("a to b (direct, shortest)", 60, 60),
        ("a to c (no direct edge)", None, 120),
        ("a to d (direct 110, path 100)", 110, 100),
    ]
    for label, edge, short in cases:
        ap = layer(edge, short)
        b = eff(6000, 2000, 2000) if ap else 0
        fe = f_eff(10_000, b)
        k = travel(short, 20, fe)
        e = "-" if edge is None else edge
        print(f"   {label:<30} {e!s:>4}   {short:>7}  {str(ap):<5} {b:>5}  {k}")
        if label.startswith("a to b"):
            assert k == 2
        else:
            assert b == 0 and k == travel(short, 20, 10_000)


def part3c_two_mills():
    print("== 3c. two mills of cap(f) 3 (cap_c 6): an own start of 3 and a hire of 3 (5.1.3)")
    facs = [dict(id=1, cond=900, cap=3, dur=1000), dict(id=2, cond=950, cap=3, dur=1000)]
    served = {1: 0, 2: 0}
    cond = {1: 900, 2: 950}
    a = assign(facs, served, cond, 3, 2)
    b = assign(facs, served, cond, 3, 2)
    print(f"   own 3 then hire 3 -> served {served}, conditions {cond}")
    assert (a, b) == (3, 3) and served == {1: 3, 2: 3}
    assert cond == {1: 894, 2: 944}


def part3b_principal():
    print("== 3b. capacity per principal and tick (5.1.2): own start and two hires against cap_c = 6")
    cap_c = 6
    used = 0
    taken = []
    for want in (4, 5, 3):                     # the own request, then two hires, in start order
        k = min(want, cap_c - used)
        used += k
        taken.append(k)
    print(f"   requests 4, 5, 3 -> executions {taken}, total {used}")
    assert taken == [4, 2, 0] and used == cap_c


def part2_ground():
    print("== 2. ground: capacity 1200, scale 10000, BoostBP 3000, Durability 500, Decay 1 (18.3)")
    lost = 0
    prev = cap_eff(1200, 10_000, eff(3000, 500, 500))
    print("   tick  condition  B     effective capacity  stock lost so far (stock kept at capacity)")
    for t in range(0, 501):
        cond = 500 - t
        b = eff(3000, cond, 500)
        c = cap_eff(1200, 10_000, b) if cond > 0 else 1200
        if c < prev:
            lost += prev - c
        prev = c
        if t % 100 == 0 or t == 499 or t == 500:
            print(f"   {t:>4}  {cond:>9}  {b:>5} {c:>18}  {lost}")
    assert prev == 1200 and lost == 1560 - 1200


def part3_assign():
    print("== 3. most worn first, two starts of one principal in a tick: 5000 random cases (invariants)")
    rnd = random.Random(12)
    worst = 0
    for _ in range(5000):
        nf = rnd.randint(1, 6)
        dur = rnd.choice([10, 100, 1000])
        facs = [dict(id=i + 1, cond=rnd.randint(1, dur), cap=rnd.randint(1, 8), dur=dur) for i in range(nf)]
        total_cap = sum(capf(f["cap"], f["cond"], dur) for f in facs)
        n = rnd.randint(0, total_cap)
        n1 = rnd.randint(0, n)                  # an own start, then a hire, in start order
        wpu = rnd.randint(0, dur)
        s2 = {f["id"]: 0 for f in facs}
        c2 = {f["id"]: f["cond"] for f in facs}
        p1 = assign(facs, s2, c2, n1, wpu)
        p2 = assign(facs, s2, c2, n - n1, wpu)
        s1 = {f["id"]: 0 for f in facs}
        c1 = {f["id"]: f["cond"] for f in facs}
        p = assign(facs, s1, c1, n, wpu)        # the same executions as one start
        assert p1 + p2 == n and p == n          # k <= cap_c, so every execution is placed
        assert s2 == s1 and c2 == c1            # the grouping of the starts changes nothing
        for f in facs:
            assert s2[f["id"]] <= capf(f["cap"], f["cond"], dur)   # no facility above its cap(f)
            assert 0 <= c2[f["id"]] <= f["cond"]
        assert sum(f["cond"] - c2[f["id"]] for f in facs) <= n * wpu
        worst = max(worst, len(facs))
    print(f"   passed; largest list {worst} facilities")


def mill(threshold, ticks=1000, weight=10, repair_units=100, repair_dur=3, repair_k_cap=10,
         dur=1000, decay=1, wpu=2, capacity=6, repair=True):
    cond, out, t, repairing = dur, 0, 0, 0
    grind_ticks = repair_ticks = 0
    ruined_at = None
    while t < ticks:
        if cond <= 0:
            ruined_at = t
            break
        if repairing:
            repairing -= 1
            repair_ticks += 1
            if repairing == 0:
                cond = min(dur, cond + repair_units * pending_k)
        elif repair and cond < threshold:
            missing = dur - cond
            pending_k = min(repair_k_cap, weight, -(-missing // repair_units))
            repairing = repair_dur
            repair_ticks += 1
            repairing -= 1
            if repairing == 0:
                cond = min(dur, cond + repair_units * pending_k)
        else:
            c = capf(capacity, cond, dur)
            k = min(weight, c)
            out += k
            grind_ticks += 1
            cond = max(0, cond - k * wpu)
        cond = max(0, cond - decay)
        t += 1
    return out, grind_ticks, repair_ticks, ruined_at, cond


def part4_mill():
    print("== 4. a mill: weight 10, Capacity 6, WearPerUse 2, Decay 1, Durability 1000 (18.1)")
    print("   policy                       executions   grind ticks  repair ticks  ruined at   condition at end (1 000 ticks)")
    out, g, r, ru, c = mill(0, repair=False)
    print(f"   never repaired               {out:>10}   {g:>11}  {r:>12}  {str(ru):>9}   {c}")
    for th in (100, 300, 600, 900):
        out, g, r, ru, c = mill(th)
        print(f"   repair below {th:<4}            {out:>10}   {g:>11}  {r:>12}  {str(ru):>9}   {c}")


def part5_cobweb():
    print("== 5. CAP-A1, a model of an INVENTED rule (a builder that builds on the price gap):")
    print("   price p = 1000 + 25 * (Q - K) with Q = 100, builders add g/1000 * (p - 1000) / 100 facilities, arriving after")
    print("   L ticks, each facility loses 1% a tick. Amplitude of K over ticks 300 to 600:")
    print("   g (per mille)   L=2    L=5    L=10")
    for g in (100, 250, 500, 1000, 2000):
        cells = []
        for lag in (2, 5, 10):
            K = 100 * 100                      # capacity in hundredths of a facility
            queue = [0] * lag
            lo, hi = 10**9, -10**9
            for t in range(600):
                p = max(0, 1000 + 25 * (100 * 100 - K) // 100)
                add = max(0, g * (p - 1000) // 1000)
                queue.append(add)
                K += queue.pop(0)
                K -= K // 100
                if t >= 300:
                    lo, hi = min(lo, K), max(hi, K)
            cells.append(f"{(hi - lo) / 100:>6.1f}")
        print(f"   {g:>13}   " + "  ".join(cells))


if __name__ == "__main__":
    part1_route()
    part1b_pairs()
    part2_ground()
    part3_assign()
    part3c_two_mills()
    part3b_principal()
    part4_mill()
    part5_cobweb()
