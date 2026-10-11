"""The failure-regime detectors of spec-stylised-facts.md section 6, in integer
arithmetic, run on synthetic series (a model of the DEFINITIONS, not of the
library; no series here is a run). Standard library only; a small integer
congruential generator makes the noise the same on every platform.

What it checks:
  1. each detector fires on the series built to show its regime and not on a
     healthy seasonal series (the discrimination table);
  2. a detector returns `unknown` for a series that is too short or whose prices
     are stale (the third class of 6.1.2);
  3. the ordering rule of 6.5.2: every fact band's prior is at most the threshold
     of the regime it backs (the registry check, on the priors);
  4. what a window shorter than the season does to a range reading (4.1.3): the
     share of successive 120-tick windows of a stationary seasonal series that a
     fact or a regime on the range would flag, against year windows.
"""
TPY = 365

# priors of 6.2 (placeholders)
R = dict(
    r1_rise=3000, r1_windows=2, r1_m_factor=30000,
    r2_vol=8000, r2_share=5000, r2_unsold=5000, r2_top=7000, r2_windows=2,
    r3_fall=2000, r3_recover=1000, r3_horizon=730, r3_horizon_factor=50000, r3_role_ticks=365,
    r4_vol=5000, r4_windows=2,
    r5_pinned=5000, r5_pinned_ticks=60, r5_freeze=60,
    r6_factor=30000, r6_growth=2500, r6_windows=2, r6_swing=1000,
    r7_disp=50, r7_spike=20000,
    r8_offset=3000, r8_unmet=3000,
    r9_fall=3000, r9_windows=2,
    stale_max=1000, active_min=5000, min_windows=3,
)


class Lcg:
    def __init__(self, seed):
        self.s = seed & ((1 << 63) - 1)

    def below(self, n):
        self.s = (self.s * 6364136223846793005 + 1442695040888963407) & ((1 << 63) - 1)
        return (self.s >> 20) % n


def tri(t):
    u = (t % TPY) * 4000 // TPY
    if u < 1000:
        return u
    if u < 3000:
        return 2000 - u
    return u - 4000


def price_series(years, amp_bp, drift_bp_per_year=0, noise_bp=200, seed=1):
    """published price per tick: base 10 000, a triangular seasonal swing of
    +-amp_bp, a compound drift per year (integer, per tick), and bounded noise."""
    g = Lcg(seed)
    out = []
    level = 10000 * 10000   # scaled by 10^4 to keep integer precision
    # per-tick factor numerator/denominator from the yearly drift, applied once a year
    for t in range(years * TPY):
        if t > 0 and t % TPY == 0:
            level = level * (10000 + drift_bp_per_year) // 10000
        base = level // 10000
        p = base * (10000 + amp_bp * tri(t) // 1000) // 10000
        p = p * (10000 + g.below(2 * noise_bp + 1) - noise_bp) // 10000
        out.append(max(p, 1))
    return out


def year_windows(series):
    n = len(series) // TPY
    return [series[k * TPY:(k + 1) * TPY] for k in range(n)]


def mean_int(xs):
    s = sum(xs)
    q, r = divmod(s, len(xs))
    if 2 * r > len(xs) or (2 * r == len(xs) and q % 2 == 1):
        q += 1
    return q


def range_bp(xs):
    return max(xs) * 10000 // min(xs)


# ---- detectors: each returns ("none" | "detected" | "unknown", detail) ----

def r1(prices, money_ratio_bp=10000, stale_bp=0):
    if stale_bp > R['stale_max']:
        return "unknown", "stale"
    w = year_windows(prices)
    if len(w) < R['min_windows']:
        return "unknown", "windows"
    means = [mean_int(x) for x in w]
    links = [means[k] * 10000 // means[k - 1] for k in range(1, len(means))]
    run = 0
    for k, link in enumerate(links):
        run = run + 1 if link - 10000 >= R['r1_rise'] else 0
        if run >= R['r1_windows']:
            return "detected", "onset window %d" % (k + 1)
    if money_ratio_bp >= R['r1_m_factor']:
        return "detected", "money"
    return "none", ""


def r9(prices, volumes):
    w = year_windows(prices)
    v = year_windows(volumes)
    if len(w) < R['min_windows']:
        return "unknown", "windows"
    means = [mean_int(x) for x in w]
    vol = [sum(x) for x in v]
    run = 0
    for k in range(1, len(means)):
        link = means[k] * 10000 // means[k - 1]
        falling = vol[k] < vol[k - 1]
        run = run + 1 if (10000 - link >= R['r9_fall'] and falling) else 0
        if run >= R['r9_windows']:
            return "detected", "onset window %d" % k
    return "none", ""


def r6_year(prices, active_bp=10000):
    if active_bp < R['active_min']:
        return "unknown", "active"
    w = year_windows(prices)
    if len(w) < R['min_windows']:
        return "unknown", "windows"
    return _r6_over(w)


def r6_ticks(prices, length):
    w = [prices[i:i + length] for i in range(0, len(prices) - length + 1, length)]
    return _r6_over(w)


def _r6_over(w):
    ranges = [range_bp(x) for x in w]
    run = 0
    for k in range(1, len(ranges)):
        grew = ranges[k] * 10000 > ranges[k - 1] * (10000 + R['r6_growth'])
        run = run + 1 if (ranges[k] > R['r6_factor'] and grew) else 0
        if run >= R['r6_windows']:
            return "detected", "onset window %d" % k
    return "none", ""


def speed_limit_ticks(h_from, h_to, rate_bp_per_year):
    """t_min of STY-S02: ticks to regrow from h_from to h_to at the fastest rate
    (per tick, carried remainder over 10 000 * TPY), the one definition of 5.3."""
    h, carry, t = h_from, 0, 0
    den = 10000 * TPY
    while h < h_to:
        add, carry = divmod(h * rate_bp_per_year + carry, den)
        h += add
        t += 1
        if t > 1000 * TPY:
            return None
    return t


def r3(households, rate_bp_per_year=20000):
    """R3 with the horizon a multiple of the speed limit (6.2): not back at
    10 000 - recover_bp within max(horizon, factor * t_min) ticks of the minimum;
    a run too short for that horizon is unknown ('needs a longer run')."""
    h0 = households[0]
    low = min(households)
    if len(households) < 2:
        return "unknown", "windows"
    tmin = households.index(low)
    if low * 10000 > h0 * (10000 - R['r3_fall']):
        return "none", ""
    target = h0 * (10000 - R['r3_recover']) // 10000
    if any(x >= target for x in households[tmin:]):
        return "none", ""
    tm = speed_limit_ticks(low, target, rate_bp_per_year)   # the same t_min as STY-S02: to 10 000 - recover_band_bp of the earlier level
    horizon = max(R['r3_horizon'], tm * R['r3_horizon_factor'] // 10000) if tm else R['r3_horizon']
    if len(households) - 1 - tmin < horizon:
        return "unknown", "needs a longer run (horizon %d, available %d)" % (horizon, len(households) - 1 - tmin)
    return "detected", "onset tick %d" % tmin


def longest_run(flags):
    best = cur = 0
    for f in flags:
        cur = cur + 1 if f else 0
        best = max(best, cur)
    return best


def r5a(share_bp_per_tick):
    n = longest_run([s >= R['r5_pinned'] for s in share_bp_per_tick])
    return ("detected", "run %d" % n) if n >= R['r5_pinned_ticks'] else ("none", "")


def r5b(unmatched_both_no_volume):
    n = longest_run(unmatched_both_no_volume)
    return ("detected", "run %d" % n) if n >= R['r5_freeze'] else ("none", "")


def r7(dispersion_bp_lowest, active_outcomes=1):
    if active_outcomes == 0:
        return "unknown", "no outcomes"
    return ("detected", "low %d" % dispersion_bp_lowest) if dispersion_bp_lowest < R['r7_disp'] else ("none", "")


def r8(offset_bp, unmet_bp, have_reference=True):
    if not have_reference:
        return "unknown", "no steady-state price"
    hit = abs(offset_bp) >= R['r8_offset'] and unmet_bp >= R['r8_unmet']
    return ("detected", "offset %d" % offset_bp) if hit else ("none", "")


def r4(volumes, households_bp_of_start):
    v = [sum(x) for x in year_windows(volumes)]
    if len(v) < R['min_windows']:
        return "unknown", "windows"
    if households_bp_of_start < 10000 - R['r3_fall']:
        return "none", "collapse is R3"
    run = 0
    for k in range(1, len(v)):
        run = run + 1 if v[k] * 10000 < v[0] * R['r4_vol'] else 0
        if run >= R['r4_windows']:
            return "detected", "onset window %d" % k
    return "none", ""


def r2(volumes, pool_share_bp_by_window):
    v = [sum(x) for x in year_windows(volumes)]
    if len(v) < R['min_windows']:
        return "unknown", "windows"
    falling = v[-1] * 10000 < v[0] * R['r2_vol']
    run = 0
    hit = False
    for s in pool_share_bp_by_window:
        run = run + 1 if s >= R['r2_share'] else 0
        if run >= R['r2_windows']:
            hit = True
    return ("detected", "pool share") if (falling and hit) else ("none", "")


# ---- scenarios -------------------------------------------------------------

def volumes_flat(years, level=100):
    return [level] * (years * TPY)


def volumes_decaying(years, yearly_bp):
    out = []
    level = 100 * 10000
    for t in range(years * TPY):
        if t > 0 and t % TPY == 0:
            level = level * yearly_bp // 10000
        out.append(level // 10000)
    return out


def discrimination():
    healthy = price_series(4, amp_bp=2500)               # seasonal, stationary
    inflating = price_series(4, amp_bp=2500, drift_bp_per_year=4000)
    deflating = price_series(4, amp_bp=2500, drift_bp_per_year=-4000)
    cobweb = []
    # a series whose seasonal+cycle range doubles each year
    for y in range(4):
        amp = 1500 * (2 ** y)
        cobweb += price_series(1, amp_bp=min(amp, 9000), seed=7 + y)
    flat_v = volumes_flat(4)
    fall_v = volumes_decaying(4, 6000)
    rows = []
    rows.append(("healthy seasonal", "R1", r1(healthy)))
    rows.append(("healthy seasonal", "R6 year", r6_year(healthy)))
    rows.append(("healthy seasonal", "R9", r9(healthy, flat_v)))
    rows.append(("inflation +40% a year", "R1", r1(inflating)))
    rows.append(("deflation -40% a year, volume falling", "R9", r9(deflating, fall_v)))
    rows.append(("deflation -40% a year, volume flat", "R9", r9(deflating, flat_v)))
    rows.append(("cobweb, range doubling", "R6 year", r6_year(cobweb)))
    rows.append(("healthy, two years only", "R1", r1(healthy[:2 * TPY])))
    rows.append(("healthy, stale prices", "R1", r1(healthy, stale_bp=2500)))
    hh_collapse3 = [600 - (400 * t) // 365 if t < 365 else 200 for t in range(0, 3 * TPY)]
    hh_collapse6 = [600 - (400 * t) // 365 if t < 365 else 200 for t in range(0, 6 * TPY)]
    hh_recover = [600 - (400 * t) // 365 if t < 365 else min(600, 200 + (t - 365)) for t in range(0, 3 * TPY)]
    rows.append(("households 600 -> 200, no recovery, 3 years", "R3", r3(hh_collapse3)))
    rows.append(("households 600 -> 200, no recovery, 6 years", "R3", r3(hh_collapse6)))
    rows.append(("households 600 -> 200 -> 600", "R3", r3(hh_recover)))
    rows.append(("pinned 80 ticks", "R5a", r5a([8000] * 80 + [0] * 100)))
    rows.append(("pinned 40 ticks", "R5a", r5a([8000] * 40 + [0] * 100)))
    rows.append(("frozen 70 ticks", "R5b", r5b([True] * 70 + [False] * 30)))
    rows.append(("lockstep dispersion 10 bp", "R7", r7(10)))
    rows.append(("offset 4 000, unmet 4 000", "R8", r8(4000, 4000)))
    rows.append(("offset 4 000, no reference", "R8", r8(4000, 4000, have_reference=False)))
    rows.append(("volume to 36% a year, households fine", "R4", r4(fall_v, 10000)))
    rows.append(("volume to 36% a year, households 70%", "R4", r4(fall_v, 7000)))
    pool = [3000, 6000, 6500]
    rows.append(("pool share 60% twice, volume falling", "R2", r2(fall_v, pool)))
    rows.append(("pool share 60% twice, volume flat", "R2", r2(flat_v, pool)))
    print("1. discrimination: verdict per series")
    for name, det, (cls, why) in rows:
        print("   %-42s %-8s %-9s %s" % (name, det, cls, why))
    print()


def ordering_rule():
    # implication pair of 6.5.2: a pinned run of r5.pinned_ticks is at least s16.pinned_share_bp of a year
    assert R['r5_pinned_ticks'] * 10000 // TPY >= 1000, "pinned run implies the fact fails"
    print("   r5.pinned_ticks %d is %d bp of a year, at least s16.pinned_share_bp 1000" % (R['r5_pinned_ticks'], R['r5_pinned_ticks'] * 10000 // TPY))
    # band prior, threshold prior, kind ("upper": band <= threshold; "lower": band >= threshold)
    pairs = [
        ("s16.pinned_share_bp", 1000, "r5.pinned_bp", R['r5_pinned'], "upper"),
        ("x4.pinned_share_bp", 1000, "r5.pinned_bp", R['r5_pinned'], "upper"),
        ("s07.freeze_ticks", 30, "r5.freeze_ticks", R['r5_freeze'], "upper"),
        ("s03.pinned_ticks", 30, "r5.pinned_ticks", R['r5_pinned_ticks'], "upper"),
        ("s04.pinned_ticks", 30, "r5.pinned_ticks", R['r5_pinned_ticks'], "upper"),
        ("s05.range_factor", 20000, "r6.factor", R['r6_factor'], "upper"),
        ("x4.range_factor", 20000, "r6.factor", R['r6_factor'], "upper"),
        ("s05.range_growth_bp", 1000, "r6.growth_bp", R['r6_growth'], "upper"),
        ("s12.range_factor", 20000, "r6.factor", R['r6_factor'], "upper"),
        ("s01.unmet_bp", 1500, "r8.unmet_bp", R['r8_unmet'], "upper"),
        ("p07.drift_bp", 500, "r1.rise_bp", R['r1_rise'], "upper"),
        ("p07.drift_bp", 500, "r9.fall_bp", R['r9_fall'], "upper"),
        ("s14.recover_ticks (4 x t_min)", 40000, "r3.horizon_factor x t_min", R['r3_horizon_factor'] * 10000 // 10000 * 1, "upper"),
        ("s06.swing_bp", 300, "r6.swing_bp", R['r6_swing'], "upper"),
        ("s01.offset_bp", 1500, "r8.offset_bp", R['r8_offset'], "upper"),
        ("h02.offset_bp", 2000, "r8.offset_bp", R['r8_offset'], "upper"),
        ("s10.dispersion_min_bp", 100, "r7.disp_bp", R['r7_disp'], "lower"),
        ("m05.pool_share_bp", 3000, "r2.share_bp", R['r2_share'], "upper"),
        ("x4.households_bp", 9000, "10000 - r3.fall_bp", 10000 - R['r3_fall'], "lower"),
    ]
    print("3. ordering rule of 6.5.2 on the priors")
    ok = True
    for fb, fv, tb, tv, kind in pairs:
        good = fv <= tv if kind == "upper" else fv >= tv
        ok = ok and good
        print("   %-24s %6d  %-6s %-20s %6d  %s" % (fb, fv, "<=" if kind == "upper" else ">=", tb, tv, "ok" if good else "VIOLATED"))
    assert ok
    print()


def windows_vs_season():
    print("4. a stationary seasonal series read by year windows and by 120-tick windows")
    print("   amp = seasonal swing +-amp bp; pairs = successive windows; 'grow' = range above the previous by more than")
    print("   s05.range_growth_bp (1 000); 'fact' = grows and range above 20 000 (fails STY-P03's two conjuncts);")
    print("   'regime' = R6's rule alone (range above 30 000 and growth above 2 500), two in a row")
    print("   amp    year windows           120-tick windows")
    print("         range  grow fact  R6    pairs grow fact  regime-run")
    for amp in (2500, 4000, 5000, 6000):
        s = price_series(8, amp_bp=amp, noise_bp=200, seed=3)
        yw = year_windows(s)
        yr = [range_bp(x) for x in yw]
        yg = sum(1 for k in range(1, len(yr)) if yr[k] * 10000 > yr[k - 1] * 11000)
        yf = sum(1 for k in range(1, len(yr)) if yr[k] * 10000 > yr[k - 1] * 11000 and yr[k] > 20000)
        y6 = r6_year(s)[0]
        tw = [s[i:i + 120] for i in range(0, len(s) - 119, 120)]
        tr = [range_bp(x) for x in tw]
        pairs = len(tr) - 1
        tg = sum(1 for k in range(1, len(tr)) if tr[k] * 10000 > tr[k - 1] * 11000)
        tf = sum(1 for k in range(1, len(tr)) if tr[k] * 10000 > tr[k - 1] * 11000 and tr[k] > 20000)
        t6 = r6_ticks(s, 120)[0]
        print("  %5d  %5d %4d %4d  %-9s %4d %4d %4d  %s" % (amp, max(yr), yg, yf, y6, pairs, tg, tf, t6))
    print()


def short_series():
    print("2. a series shorter than the rule needs is unknown, not none:")
    print("   R1 on 2 years:", r1(price_series(2, 2500)))
    print("   R6 on 2 years:", r6_year(price_series(2, 2500)))
    print()


if __name__ == "__main__":
    discrimination()
    short_series()
    ordering_rule()
    windows_vs_season()
