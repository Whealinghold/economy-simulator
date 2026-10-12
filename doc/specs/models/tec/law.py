"""TEC model 1: the integer rules of spec-technology.md 3 to 6 (the cell, 3.3; the sections 4.2 to 4.5 are its parts; the log law, the trend, the keyed discovery, the
diffusion step) and the identities they must keep.

A stylised model of the RULES, not of the library. Everything that would affect a result is integer (Python ints stand for the
checked 128-bit helper of spec-currency.md 7); floating point appears ONLY in the printouts and in the predictions. Every
number is a placeholder (N-015, N-027); nothing here shows that any default is stable."""
import random

BP = 10_000
TPY = 365
M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15


# ---- the finaliser of spec-world.md 7.1.2 (SplitMix64), the keyed hash of 5.2 -------------------------------------------
def finalise(z):
    z &= M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def keyed(key, t, cell, ncells):
    """K(key, t, cell): the wrap of the modular product is the hash (architecture.md 3.5)."""
    return finalise((key + ((t * ncells + cell + 1) * GAMMA)) & M64)


def hit(key, t, cell, ncells, discovery_bp):
    """True with probability discovery_bp / (10 000 * TPY): the high word of K * (10 000 * TPY) is in [0, 10 000 * TPY)."""
    n = BP * TPY
    u = (keyed(key, t, cell, ncells) * n) >> 64
    return u < discovery_bp


# ---- the log law of 4.3: a piecewise-linear log2 in 12-bit fixed point -------------------------------------------------
def lg12(v):
    """log2(v) * 4096 for v >= 1: the integer part exact, the fraction linear between powers of two (shifts only)."""
    k = v.bit_length() - 1
    top = (v >> (k - 12)) if k >= 12 else (v << (12 - k))
    return k * 4096 + (top - 4096)


def gain_log(x, base, doubling_bp):
    """g(X) = DoublingBP * (lg12(X + Base) - lg12(Base)) / 4096, truncating; 0 at X = 0."""
    return doubling_bp * (lg12(x + base) - lg12(base)) // 4096


def true_gain(x, base, doubling_bp):
    import math
    return doubling_bp * math.log2((x + base) / base)


def part1():
    print("== 1. the log law (4.2): exact at a doubling, bounded error between, monotone, no overflow")
    base, d = 100_000, 200
    print(f"   Base {base}, DoublingBP {d}: gain at X = Base * (2^k - 1) must be exactly k * DoublingBP")
    ok = True
    for k in range(0, 21):
        x = base * ((1 << k) - 1)
        g = gain_log(x, base, d)
        ok &= g == k * d
    print(f"   exact at 21 doublings: {ok}")
    worst = 0.0
    rng = random.Random(7)
    for _ in range(200_000):
        x = rng.randrange(0, base * 1_000_000)
        e = true_gain(x, base, d) - gain_log(x, base, d)
        worst = max(worst, abs(e))
    print(f"   largest |true - integer| over 200 000 random X: {worst:.2f} bp of gain "
          f"= {worst / d:.3f} of one doubling (the interpolation bound is 0.086)")
    mono = all(gain_log(x, base, d) <= gain_log(x + 1, base, d) for x in range(0, 100_000, 17))
    print(f"   monotone in X on a grid: {mono}")
    big = gain_log((1 << 50), 1, 10_000)
    print(f"   at X = 2^50, Base 1, DoublingBP 10 000 (the bounds): gain {big}, the product {10_000 * (lg12((1 << 50) + 1) - lg12(1))} "
          f"is far below 2^63 = {1 << 63}")
    print("   the years to each doubling for 1 000 households, one execution each, every tick (placeholder):")
    x, t, row = 0, 0, []
    for k in range(1, 9):
        need = base * ((1 << k) - 1)
        row.append(f"{k}: {need / 1000 / TPY:.1f}y")
    print("   " + "  ".join(row))


# ---- the trend of 5.1: a carried remainder over TicksPerYear -------------------------------------------------------------
def part2():
    print("\n== 2. the trend (4.3): carried remainder, exact over any whole number of years")
    for tr in (1, 137, 200, 9_999):
        carry, tot = 0, 0
        for t in range(3 * TPY):
            carry += tr
            add, carry = divmod(carry, TPY)
            tot += add
        print(f"   TrendBP {tr:5d}: after 3 years {tot:6d} (exact {3 * tr:6d}), carry {carry}")


# ---- the discovery of 4.5: keyed hash, one word ------------------------------------------------------------------------
def part3():
    print("\n== 3. discovery (4.4): the keyed hash hits at the stated rate and takes no word per tick")
    cells, years = 60, 200
    print(f"   {cells} active cells over {years} years ({cells * years * TPY} cell-ticks)")
    print(f"   {'DiscoveryBP':>11} {'expected':>9} {'observed':>9}")
    for dbp in (50, 500, 5_000, 10_000):
        for key in (0x1234_5678_9ABC_DEF0,):
            n = sum(hit(key, t, c, cells, dbp) for t in range(years * TPY) for c in range(cells))
            exp = cells * years * dbp / BP
            print(f"   {dbp:11d} {exp:9.1f} {n:9d}")
    k1, k2 = 0x1111, 0x2222
    a = [hit(k1, t, 0, 60, 5_000) for t in range(20_000)]
    b = [hit(k2, t, 0, 60, 5_000) for t in range(20_000)]
    same = sum(1 for x, y in zip(a, b) if x and y)
    print(f"   two keys, one cell, DiscoveryBP 5 000: hits {sum(a)} and {sum(b)}, hit on both {same} (independent: about "
          f"{sum(a) * sum(b) / 20_000:.1f})")
    c0 = [t for t in range(20_000) if hit(k1, t, 0, 60, 5_000)]
    c1 = [t for t in range(20_000) if hit(k1, t, 1, 60, 5_000)]
    print(f"   one key, cells 0 and 1: hit on both {len(set(c0) & set(c1))} of {len(c0)} and {len(c1)}")


# ---- the diffusion of 4.6 ------------------------------------------------------------------------------------------------
def weight(d, reach):
    return 0 if d is None or d >= reach else (reach - d) * BP // reach


def diffuse_step(G, dist, catch_bp, reach, carry):
    """One tick over one field: G the gains of the cells (pre-step snapshot), dist[s][r] the REG distance or None.
    Returns the added advance per cell and the new carry. Every target is read from the snapshot G."""
    n = len(G)
    add = [0] * n
    D = BP * TPY
    for r in range(n):
        front = 0
        for s in range(n):
            if s == r:
                continue
            w = weight(dist[s][r], reach)
            if w:
                front = max(front, G[s] * w // BP)
        gap = front - G[r]
        if gap > 0:
            carry[r] += gap * catch_bp
            a, carry[r] = divmod(carry[r], D)
            add[r] = a
    return add


def part4():
    print("\n== 4. diffusion (4.5): a lagging cell approaches w * frontier from below, never passes it; no order effect")
    reach = 100
    # three regions: 0 leads; 1 is at distance 20 from 0; 2 at distance 60 from 0 and 50 from 1; directed, symmetric here
    dist = [[None, 20, 60], [20, None, 50], [60, 50, None]]
    G = [3_000, 0, 0]
    carry = [0, 0, 0]
    A = [0, 0, 0]
    print(f"   Reach {reach}; frontier cell 0 at gain {G[0]}; distances 0-1 20, 0-2 60, 1-2 50; CatchUpBP 2 000 (20% of the gap a year)")
    print(f"   w(20) = {weight(20, reach)}, w(60) = {weight(60, reach)}, w(50) = {weight(50, reach)}; targets: "
          f"cell 1 -> {3000 * weight(20, reach) // BP}, cell 2 -> max({3000 * weight(60, reach) // BP}, cell 1 * w(50))")
    marks = {1: 0, 5: 0, 10: 0, 20: 0}
    for t in range(1, 25 * TPY + 1):
        add = diffuse_step(G, dist, 2_000, reach, carry)
        for i in range(3):
            A[i] += add[i]
            G[i] = G[i] if i == 0 else A[i]
        if t % TPY == 0 and t // TPY in marks:
            print(f"   year {t // TPY:2d}: gains {G}")
    print(f"   limits: cell 1 {3000 * weight(20, reach) // BP}; cell 2 the larger of the direct {3000 * weight(60, reach) // BP} and the relay "
          f"through cell 1 ({3000 * weight(20, reach) // BP} * w(50) = {3000 * weight(20, reach) // BP * weight(50, reach) // BP}); "
          f"a relay is a frontier like any, one tick behind, and can pass the direct limit by the product of the two losses (spec 4.5.5: bounded by the product of the weights of the legs, not by w of the total distance)")
    # order independence: permute the cells
    import itertools
    base = None
    ok = True
    for perm in itertools.permutations(range(3)):
        Gp = [[3_000, 400, 0][i] for i in perm]
        dp = [[dist[perm[a]][perm[b]] for b in range(3)] for a in range(3)]
        out = diffuse_step(Gp, dp, 2_000, reach, [0, 0, 0])
        inv = [0] * 3
        for pos, i in enumerate(perm):
            inv[i] = out[pos]
        if base is None:
            base = inv
        ok &= inv == base
    print(f"   the added advance does not depend on the order of the cells (6 permutations): {ok}")
    # a closed edge: no path, no contact
    dist2 = [[None, None, 60], [None, None, 50], [60, 50, None]]
    c2 = [0, 0, 0]
    out = diffuse_step([3_000, 0, 0], dist2, 2_000, reach, c2)
    print(f"   edge 0-1 closed (no path 0 to 1, 1 to 0): the step adds {out} and the carries are {c2}: cell 1 has no contact and its carry stays 0, "
          f"cell 2 (60 from cell 0) carries 1 200 * 2 000 = 2 400 000 of 3 650 000 toward its first unit")


def part5():
    print("\n== 5. the identity of the event (4.6.3): After = Before + Doing + Trend + Discovery + CatchUp - Clamped, Clamped >= 0")
    rng = random.Random(3)
    cap = 3_000
    worst_neg = 0
    ok = True
    for _ in range(100_000):
        X = rng.randrange(0, 5_000_000)
        A = rng.randrange(0, cap + 1)
        before = min(cap, gain_log(X, 100_000, 200) + A)
        dx = rng.randrange(0, 3_000)
        da = rng.randrange(0, 400)
        doing = gain_log(X + dx, 100_000, 200) - gain_log(X, 100_000, 200)
        A2 = min(cap, A + da)
        after = min(cap, gain_log(X + dx, 100_000, 200) + A2)
        clamped = before + doing + da - after
        ok &= clamped >= 0 and after >= before
        worst_neg = min(worst_neg, clamped)
    print(f"   100 000 random cells: Clamped >= 0 and After >= Before in every case: {ok} (smallest Clamped {worst_neg})")


if __name__ == "__main__":
    part1()
    part2()
    part3()
    part4()
    part5()
