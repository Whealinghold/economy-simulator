"""models/ins/cover.py: the integer rules of spec-insurance.md 5 to 9 and two stylised loops.

A model of the RULES of the spec, not of the library. Nothing here shows that a default is
stable (note N-027); every number is a placeholder. Parts:

  1  the premium of a route cover and its price per unit (5.2)
  2  the claim at an arrival: loss value, due, the payout cap (7.1, 7.2)
  3  the pay-out of one underwriter's policies: pro rata with the largest remainder, the
     shield, the shortfall as a claim (7.3, 7.4)
  4  a store cover, with the lost units of spec-risk.md 18.2 (5.3, 7.5)
  5  the value cap: the payout against a rising published price (6)
  6  the underwriter's capital under unit rows and under a common row (INS-A2, INS-A4)
  7  the shipper that nets the recovery of a cover (INS-A1), the linear model of
     models/rsk/loss.py part 7 with the arrival link unchanged and the read changed

Run (Docker only, never a local interpreter):
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/ins":/w:ro -w /w python:3.12-slim python cover.py
"""
import math
import random

M64 = (1 << 64) - 1


def rhe(a, b):
    """a / b rounded half to even, a >= 0, b > 0."""
    q, r = divmod(a, b)
    if 2 * r > b or (2 * r == b and q % 2 == 1):
        q += 1
    return q


def key(t, ident):
    """The stateless key K(t, id) of spec-world.md 10.1.1 (SplitMix64 finaliser)."""
    z = (t * 0x9E3779B97F4A7C15 + ident) & M64
    z = (z + 0x9E3779B97F4A7C15) & M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def allocate(pay, owed, t, ids):
    """Shares of `pay` among `owed` (spec-credit.md 8.3): pro rata, largest remainder, ties by key then ID."""
    total = sum(owed)
    assert 0 <= pay <= total
    if total == 0:
        return [0] * len(owed)
    shares = [pay * o // total for o in owed]
    frac = [(pay * o) % total for o in owed]
    left = pay - sum(shares)
    order = sorted(range(len(owed)), key=lambda i: (-frac[i], key(t, ids[i]), ids[i]))
    for i in order[:left]:
        shares[i] += 1
    return shares


def unit_value(pub, ref, cap_bp):
    """min(published price, ValueCapBP * reference price / 10 000), truncated (6.1)."""
    return min(pub, ref * cap_bp // 10_000)


def expected_loss_bp(rows):
    """spec-risk.md 8.1.1: rows are (strike_bp, severity_bp) in path order."""
    k = 10_000
    for p, s in rows:
        f = 10_000 - rhe(p * s, 10_000)
        k = rhe(k * f, 10_000)
    return 10_000 - k


def main():
    # ------------------------------------------------------------------ part 1
    print("== 1. The premium of a route cover (spec-insurance.md 5.2, example A of 17.1) ==")
    rows = [(3_000, 2_500), (1_000, 6_000)]            # the rows of spec-risk.md 18.1
    L = expected_loss_bp(rows)
    cover_bp, loading_bp, cap_bp = 8_000, 13_000, 12_000
    lines = [("grain", 40, 12, 10), ("cloth", 5, 90, 80), ("rope", 2, 30, 30), ("tar", 6, 25, 20)]
    # (name, shipped, published price at the origin at P3.3c, effective reference price)
    total = 0
    for name, q, pub, ref in lines:
        v = unit_value(pub, ref, cap_bp)
        total += q * v
        print("  %-5s shipped %2d published %3d reference %3d unit value %3d line %4d" % (name, q, pub, ref, v, q * v))
    sum_insured = total
    rate_bp = rhe(L * cover_bp * loading_bp, 10 ** 8)
    premium = rhe(sum_insured * L * cover_bp * loading_bp, 10 ** 12)
    max_payout = rhe(cover_bp * sum_insured, 10_000)
    print("  expected loss L = %d bp; sum insured %d; rate %d bp; premium %d; max payout %d" % (L, sum_insured, rate_bp, premium, max_payout))
    fair = rhe(sum_insured * L * cover_bp, 10 ** 8)
    print("  fair premium (loading 10 000) = %d; premium / fair = %.3f" % (fair, premium / fair))

    # ------------------------------------------------------------------ part 2
    print()
    print("== 2. The claim at the arrival (7.1, 7.2): lost units 9, 1, 0, 2 of spec-risk.md 18.1 ==")
    lost = {"grain": 9, "cloth": 1, "rope": 0, "tar": 2}
    pub_now = {"grain": 14, "cloth": 90, "rope": 30, "tar": 25}     # grain rose to 14 since the dispatch
    ref = {n: r for n, _, _, r in lines}
    loss_value = 0
    for n, u in lost.items():
        v = unit_value(pub_now[n], ref[n], cap_bp)
        loss_value += u * v
        print("  %-5s lost %d at published %3d: unit value %3d" % (n, u, pub_now[n], v))
    due_raw = rhe(cover_bp * loss_value, 10_000)
    due = min(max_payout, due_raw)
    print("  loss value %d; cover_bp * loss value = %d; due = min(max payout %d, that) = %d" % (loss_value, due_raw, max_payout, due))

    # ------------------------------------------------------------------ part 3
    print()
    print("== 3. One underwriter, three policies due at one position (7.3, 7.4) ==")
    t = 15
    ids = [101, 107, 112]
    owed = [due, 150, 90]
    for balance, shield in ((1_000, 100), (400, 100), (150, 100), (80, 100)):
        pot = max(0, balance - shield)
        pay = min(pot, sum(owed))
        shares = allocate(pay, owed, t, ids)
        short = [o - s for o, s in zip(owed, shares)]
        print("  balance %4d shield %3d pot %4d owed %s pay %4d shares %s shortfall (claims) %s" % (balance, shield, pot, owed, pay, shares, short))
        assert sum(shares) == pay and all(0 <= s <= o for s, o in zip(shares, owed))
    # the identity of the shortfall: paid + claimed = due, to the unit
    # a removal: the waterfall of spec-credit.md 9.1 pays the same claims pro rata with no shield
    print("  removal (no shield): balance 150, claims %s -> shares %s" % ([40, 60, 200], allocate(150, [40, 60, 200], t, [1, 2, 3])))

    # ------------------------------------------------------------------ part 4
    print()
    print("== 4. A store cover with the lost units of spec-risk.md 18.2 (5.3, 7.5) ==")
    held = {"grain": (250, 11, 10), "cloth": (50, 70, 80)}       # units, published, reference
    held_value = sum(q * unit_value(p, r, cap_bp) for q, p, r in held.values())
    share_bp, store_rate_bp = 5_000, 150
    s_ins = held_value * share_bp // 10_000
    prem = rhe(s_ins * store_rate_bp, 10_000)
    mp = rhe(cover_bp * s_ins, 10_000)
    lv = 133 * unit_value(11, 10, cap_bp) + 27 * unit_value(70, 80, cap_bp)
    d = min(mp, rhe(cover_bp * lv, 10_000))
    print("  held value %d; sum insured %d (share %d); premium %d; max payout %d" % (held_value, s_ins, share_bp, prem, mp))
    print("  loss value (133 grain, 27 cloth) %d; cover_bp * loss = %d; due = %d (the cap binds)" % (lv, rhe(cover_bp * lv, 10_000), d))

    # ------------------------------------------------------------------ part 5
    print()
    print("== 5. The value cap against a rising published price (6.1): 10 units lost, reference 10, cap 12 000 bp ==")
    for pub in (10, 12, 15, 25, 40):
        v = unit_value(pub, 10, 12_000)
        print("  published %3d unit value %3d loss value %4d payout at cover 8 000: %4d" % (pub, v, 10 * v, rhe(8_000 * 10 * v, 10_000)))

    # ------------------------------------------------------------------ part 6
    print()
    print("== 6. The underwriter's capital under unit rows and under a common row (INS-A2, INS-A4) ==")
    print("  one underwriter, money 1 000, upkeep 5 a tick, policies of max payout 100 each, strike probability 0.05,")
    print("  loading 13 000 on the expected payout; capacity = cap_bp * available money / 10 000 less open liability;")
    print("  pays above a shield of 3 ticks of upkeep; the shortfall is a claim paid first from later income, and an")
    print("  underwriter in arrears writes nothing; removed when its money is below one tick of upkeep. 300 seeds, 300 ticks; demand for cover is at most 20 policies a tick.")

    def run(seed, cap, common, ticks=300, p=0.05, mp=100, load=1.3, upkeep=5, shield=15, start=1_000, max_n=20):
        r = random.Random(seed)
        bal, debt = start, 0
        shortfall_total = 0
        for tick in range(ticks):
            # settlement of the claim first (P1.7a), then upkeep
            pot = max(0, bal - shield)
            pay = min(pot, debt)
            bal -= pay
            debt -= pay
            if bal < upkeep:
                return tick, shortfall_total + debt          # removed: the debt is lost to the insured
            bal -= upkeep
            # the round: nothing written in arrears
            n = 0
            if debt == 0:
                n = min(max_n, max(0, (cap * bal // 10_000) // mp))
            premium = int(round(n * mp * p * load))
            bal += premium
            # the strikes of the tick
            if common:
                hit = n if r.random() < p else 0
            else:
                hit = sum(1 for _ in range(n) if r.random() < p)
            due = hit * mp
            pot = max(0, bal - shield)
            pay = min(pot, due)
            bal -= pay
            debt += due - pay
            shortfall_total += due - pay
        return ticks, shortfall_total

    print("  cap_bp  rows    removed by 300   mean tick removed   mean shortfall per run")
    for cap in (5_000, 10_000, 20_000, 40_000):
        for common in (False, True):
            gone, tick_sum, short_sum = 0, 0, 0
            for s in range(300):
                tick, sh = run(s, cap, common)
                if tick < 300:
                    gone += 1
                    tick_sum += tick
                short_sum += sh
            print("  %6d  %-6s  %5d / 300   %17s   %10.1f" % (cap, "common" if common else "unit", gone,
                  ("%.1f" % (tick_sum / gone)) if gone else "-", short_sum / 300))

    # ------------------------------------------------------------------ part 7
    print()
    print("== 7. The shipper that nets the recovery of a cover (INS-A1): the loop of models/rsk/loss.py part 7 ==")
    # The arrival link is the PHYSICAL arrival (1 - lam): a cover moves money and no goods. What a cover
    # changes is what the shipper READS: it nets the recovery, so it reads lam_r = lam * (1 - cover).
    def totals(g_, d, lam, lam_read, alpha=0.5, beta=0.5, pd0=180.0, po0=100.0, f=20.0, n=3000):
        po, pd = po0, pd0
        q = []
        total, arrived = 0.0, 0.0
        for tt in range(n):
            gap = (1 - lam_read) * pd - po - f
            want = max(0.0, g_ * gap)
            q.append(want)
            total += want
            a = (1 - lam) * q[tt - d] if tt - d >= 0 else 0.0
            arrived += a
            po += alpha * want
            pd -= beta * a
        return total, arrived, pd, po

    lam, cover, load = 0.25, 0.8, 1.3
    print("  delay 3, gain 0.05, loss 2 500 bp, cover 8 000 bp, loading 13 000 on the expected payout")
    for label, lam_read in (("lossless read (loss ignored)", 0.0), ("reads the loss, no cover", lam), ("nets the recovery", lam * (1 - cover))):
        tot, arr, pd, po = totals(0.05, 3, lam, lam_read)
        premium_unit = lam * cover * load * po
        recovery_unit = cover * lam * po
        margin_cash = (1 - lam) * pd - po - 20.0 + recovery_unit - premium_unit
        margin_goods = (1 - lam) * pd - po - 20.0
        print("  %-30s shipped %6.2f arrived %6.2f  Pd %6.2f Po %6.2f  margin of goods %+7.2f  margin with cover and premium %+7.2f"
              % (label, tot, arr, pd, po, margin_goods, margin_cash))

    def amplitude(g_, d, a, scale, n=4000):
        x = [1.0]
        q = []
        for tt in range(n):
            want = g_ * x[tt]
            q.append(want)
            arrived = q[tt - d] if tt - d >= 0 else 0.0
            x.append(x[tt] - a * want - (1 - a) * scale * arrived)
            if abs(x[-1]) > 1e9:
                return float("inf")
        q4 = n // 4
        last = max(abs(v) for v in x[-q4:])
        if last < 1e-100:
            return 0.0
        before = max(abs(v) for v in x[-2 * q4:-q4])
        return last / before if before > 0 else 0.0

    def max_gain(d, a, scale, step=0.002):
        g_, best = step, 0.0
        while g_ < 4.0:
            if amplitude(g_, d, a, scale) < 0.999:
                best = g_
                g_ += step
            else:
                break
        return best

    g0 = 2 * math.sin(math.pi / (2 * (2 * 3 + 1)))
    print("  largest gain that decays, delay 3, immediate share a = 0 (theory g0 / scale):")
    for label, lam_read in (("lossless read", 0.0), ("reads the loss", lam), ("nets the recovery", lam * (1 - cover))):
        scale = (1 - lam) * (1 - lam_read)
        print("    %-18s scale %.4f: %.3f (theory %.3f)" % (label, scale, max_gain(3, 0.0, scale), g0 / scale))


main()
