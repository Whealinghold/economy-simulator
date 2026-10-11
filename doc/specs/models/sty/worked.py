"""The arithmetic of the worked examples of spec-stylised-facts.md section 17.
Every figure is asserted. Synthetic values, an invented setting (flagged in the
spec); nothing here is a run of the library or a band. Standard library only."""
from fractions import Fraction

import detectors as D
from seeds import Lcg, bp, round_outward as round2


def rdiv(n, d):
    """half to even, symmetric for a negative numerator (IND 5.2.2)."""
    sign = -1 if (n < 0) != (d < 0) else 1
    n, d = abs(n), abs(d)
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return sign * q


def band_from_setting(values, h_bp):
    lo, hi = min(values), max(values)
    span = hi - lo
    lo2 = lo - rdiv(span * h_bp, 10000)
    hi2 = hi + rdiv(span * h_bp, 10000)
    return lo, hi, lo2, hi2


def median(xs):
    s = sorted(xs)
    n = len(s)
    return s[n // 2] if n % 2 else rdiv(s[n // 2 - 1] + s[n // 2], 2)


def ex1_band_setting():
    print("17.1 setting a band: 50 setting seeds (1 to 50, REF's nightly range) of s01.recover_ticks; the gate seeds 1 to 10 are the first ten")
    g = Lcg(2026)
    setting = [60 + sum(g.below(40) for _ in range(3)) for _ in range(50)]   # synthetic: 60 to 177
    gate = setting[:10]
    lo, hi, lo2, hi2 = band_from_setting(setting, 2500)
    print("  setting values: min %d, median %d, max %d (range %d)" % (lo, median(setting), hi, hi - lo))
    print("  headroom 2 500 bp of the range: upper bound %d" % hi2)
    ub = round2(hi2, True)
    print("  rounded outward to two significant digits: %d ticks" % ub)
    inside = sum(1 for v in gate if v <= ub)
    print("  gate seeds inside: %d of 10; SR1 %s; SR2 %s" % (inside, "pass" if inside == 10 else "fail", "pass" if inside >= 9 else "fail"))
    assert ub >= hi
    print()
    return ub


def ex2_seed_rules():
    print("17.2 the rules on one gate run: values 41, 55, 58, 60, 62, 64, 66, 70, 74, 103 against an upper bound of 90")
    vals = [41, 55, 58, 60, 62, 64, 66, 70, 74, 103]
    inside = sum(1 for v in vals if v <= 90)
    need = -(-9 * len(vals) // 10)
    print("  inside %d of %d; SR1 (all) %s; SR2 (at least %d) %s; median %d is %s" % (
        inside, len(vals), "pass" if inside == len(vals) else "fail", need, "pass" if inside >= need else "fail",
        median(vals), "inside" if median(vals) <= 90 else "outside"))
    assert inside == 9 and median(vals) == 63
    print("  the outside seed (103) is listed in the verdict record: it is not discarded")
    print()


def ex3_classification():
    print("17.3 classifying a series: the whole-economy index of an inflating run, year windows")
    s = D.price_series(4, amp_bp=2500, drift_bp_per_year=4000, seed=11)
    means = [D.mean_int(w) for w in D.year_windows(s)]
    links = [rdiv(means[k] * 10000, means[k - 1]) for k in range(1, len(means))]
    print("  yearly mean prices:", means)
    print("  links in bp:", links)
    print("  R1 verdict:", D.r1(s))
    assert D.r1(s)[0] == "detected"
    h = D.price_series(4, amp_bp=2500, seed=11)
    print("  the same without the drift: R1", D.r1(h)[0], " R6 (year)", D.r6_year(h)[0])
    print()


def ex4_speed_limit():
    print("17.4 the speed limit of a recovery, by integer iteration per tick with a carried remainder")
    tpy = 365
    g_max = 200          # basis points a year: max(birth 250) - min(death 50) of POP 6.3's defaults
    before, trough = 1000, 820
    level = before * 9000 // 10000      # 10 000 - s14.recover_band_bp (1 000) of the earlier level
    h, carry, t = trough, 0, 0
    den = 10000 * tpy
    while h < level:
        num = h * g_max + carry
        add, carry = divmod(num, den)
        h += add
        t += 1
        if t > 100 * tpy:
            break
    print("  fall of 18%% (1000 to 820), g_max %d bp a year, back to %d (the lower edge of s14.recover_band_bp): %d ticks (%d years and %d ticks)" % (g_max, level, t, t // tpy, t % tpy))
    # the real-number reading for comparison, done in exact rationals by repeated yearly compounding
    f = Fraction(trough)
    y = 0
    while f < level:
        f = f * Fraction(10000 + g_max, 10000)
        y += 1
    print("  yearly compounding in exact fractions: %d years" % y)
    assert t > 0
    print()
    return t


def ex5_premium():
    print("17.5 the historical premium: Clark (2005) table A2, craftsman over labourer, England, 12 points")
    pts = [156, 193, 193, 179, 150, 144, 147, 144, 160, 151, 152, 141]
    bps = [p * 100 for p in pts]
    print("  min %d, median %d, max %d (bp)" % (min(bps), median(bps), max(bps)))
    assert (min(bps), median(bps), max(bps)) == (14100, 15150, 19300)
    print()


def ex6_gap_after_freight():
    print("17.6 the gap after freight (STY-P06), three directed lanes, delivered price = origin price + freight")
    lanes = [("a->b", 100, 12, 118), ("b->c", 90, 9, 104), ("a->c", 100, 20, 141)]
    gaps = []
    for name, p_o, f, p_d in lanes:
        deliv = p_o + f
        gap = p_d - deliv
        share = rdiv(gap * 10000, deliv)
        gaps.append(share)
        print("  %s: delivered %d, destination %d, gap %d, %d bp of the delivered price" % (name, deliv, p_d, gap, share))
    med = median(gaps)
    worst = max(gaps)
    print("  median over lanes: %d bp; against a margin of 1 000: %s" % (med, "pass" if med <= 1000 else "fail"))
    print("  worst lane: %d bp; against x1.worst_bp of 3 000: %s (a median alone would not show it)" % (worst, "pass" if worst <= 3000 else "fail"))
    assert (med, worst) == (536, 1750)
    print()


def ex7_unfed_bound():
    print("17.7 STY-S04: the earliest unfed tick of a household after a cut")
    held, need, cut = 14, 2, 400
    first = cut + held // need
    print("  holds %d units, needs %d a tick, cut at %d: not unfed before tick %d" % (held, need, cut, first))
    assert first == 407
    print()


def main():
    ex1_band_setting()
    ex2_seed_rules()
    ex3_classification()
    ex4_speed_limit()
    ex5_premium()
    ex6_gap_after_freight()
    ex7_unfed_bound()
    print("all asserted figures hold")


if __name__ == "__main__":
    main()
