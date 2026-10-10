"""RES model 5: the arithmetic of the worked examples of spec-resources.md 16 and the sizing of the five-region proposal.

Integer rules of spec-resources.md 4 and 5; fractions only for the analysis of the viability row. Placeholders throughout;
the examples are settings of the author's invention (N-017) and claim nothing about any real land, forest or mine."""
from fractions import Fraction
from coupled import Res, interp, BP, ceil_div
from allocate import allocate

def season_of(day, bounds):
    for name, lo, hi in bounds:
        if lo <= day < hi: return name

# ---- Example A: the Earth calendar, 365 ticks a year, one tick a day, seasons from the Earth default (days from 1 Jan).
# spec-calendar.md 7.4: season_start 59 (1 March); spring 92, summer 92, autumn 91, winter 90 days.
SEASONS = [("spring", 0, 92), ("summer", 92, 184), ("autumn", 184, 275), ("winter", 275, 365)]   # days from the season cycle's start
def sigma(season, prof):
    return prof.get(season, 0) if prof else BP

def arable_year():
    """arable: refill 100% a year, regeneration only in autumn and winter at twice the rate; capacity 2 000 plots in north,
    380 growers each start one job in the first 20 ticks of spring (19 draws a tick), starting stock 1 700."""
    prof = {"autumn": 20_000, "winter": 20_000}
    r = Res("refill", 2_000, 10_000, 1_700, [])
    rows = []
    for year in range(4):
        trace = {}
        for day in range(365):
            seas = season_of(day, SEASONS)
            if day == 0: trace["spring start"] = r.s
            if seas == "spring" and day < 20:
                take = min(19, r.s); r.s -= take
            r.regen(m=BP, sig=prof.get(seas, 0))
            if day == 19: trace["after the draws"] = r.s
            if day == 91: trace["end of spring"] = r.s
            if day == 183: trace["end of summer"] = r.s
            if day == 274: trace["end of autumn"] = r.s
            if day == 364: trace["end of winter"] = r.s
        rows.append(trace)
    return rows

def woodland_runs():
    """woodland: logistic 30% a year, capacity 50 000, refuge 20. cut_wood draws 12 per execution; the abundance curve is
    (0, 0) (2 000, 6 000) (5 000, 9 000) (10 000, 10 000). A steady 9 000 units a year is drawn (750 a month)."""
    curve = [(0, 0), (2_000, 6_000), (5_000, 9_000), (10_000, 10_000)]
    out = []
    for label, per_year in (("steady 3 000 a year", 3_000), ("steady 3 600 a year (0.96 of F_max = 3 750)", 3_600), ("steady 4 500 a year (above F_max)", 4_500)):
        r = Res("logistic", 50_000, 3_000, 50_000, curve, refuge=20)
        trace = []; carry = 0
        for day in range(40 * 365):
            carry += per_year
            take = carry // 365; carry %= 365
            take = min(take, r.s); r.s -= take
            r.regen()
            if day % (5 * 365) == 0: trace.append((day // 365, r.s, r.abundance()))
        out.append((label, trace))
    return out

# ---- Example B: 640 ticks a year, no seasons used here.
def ice_runway():
    """ice_field: finite deposit. crater_rim capacity 3 000 000, cut_ice draws 30 per execution at yield 6 000 (18 ice an execution).
    1 000 households need 1 water a tick; melt turns 1 ice into 6 water: 1 000/6 ice a tick."""
    tpy = 640
    ice_per_tick = Fraction(1000, 6)
    execs = ice_per_tick / Fraction(18)                 # executions a tick at the full abundance
    draw_tick = execs * 30
    return tpy, float(ice_per_tick), float(execs), float(draw_tick), float(draw_tick * tpy), float(Fraction(3_000_000) / (draw_tick * tpy))

def ice_with_curve():
    """same, with an abundance curve (0, 2 000) (2 000, 8 000) (10 000, 10 000): the yield per execution falls when the deposit is low."""
    curve = [(0, 2_000), (2_000, 8_000), (10_000, 10_000)]
    C = 3_000_000; S = C; tick = 0; need_ice = Fraction(1000, 6); mile = {}
    while S > 0 and tick < 100 * 640:
        a = interp(curve, S * BP // C)                  # bp
        out_exec = Fraction(30) * Fraction(6_000, BP) * Fraction(a, BP)   # ice per execution
        execs = need_ice / out_exec
        take = int(execs * 30) + 1
        S -= min(S, take); tick += 1
        for p in (80, 50, 20, 5):
            if S * 100 <= p * C and p not in mile: mile[p] = tick / 640
    return tick / 640, mile

def via_row():
    """The viability row of the ice_field (spec-resources.md 11): the sustainable flow of a finite deposit is Initial / Horizon."""
    C = 3_000_000
    res = []
    for horizon in (50, 15, 10):
        phi_year = Fraction(C, horizon); phi_tick = phi_year / 640
        res.append((horizon, float(phi_year), float(phi_tick)))
    return res

def allocation_example():
    reqs = [("A", 4, 1), ("B", 3, 1), ("C", 3, 1), ("D", 2, 1), ("E", 1, 1)]
    # allocate() takes integer agent ids; map letters to 0..4
    ids = {n: i for i, (n, _, _) in enumerate(reqs)}
    g = allocate(10, [(ids[n], w, d) for n, w, d in reqs], 7, "hash")
    U = sum(w * d for _, w, d in reqs)
    rows = []
    for n, w, d in reqs:
        p, phi = divmod(w * 10, U)
        rows.append((n, w, p, phi, g[ids[n]]))
    return U, rows

# ---- the five-region proposal (spec-resources.md 18)
FIVE = {  # households, mean effective staple yield at the starts (bp) from spec-reference-chain.md 4.4.4 (hand-derived means)
    "north": (1200, 10_000), "east": (1000, 9_500), "south": (1000, 10_000), "west": (800, 7_000), "centre": (1000, 8_000)}

def five_sizing():
    rows = []
    for reg, (hh, ebar) in FIVE.items():
        need = Fraction(225, 100) * hh                           # staple a tick at lambda = 1: 2 upkeep + 0.25 through prepare
        execs_tick = need / (Fraction(1760) * Fraction(ebar, BP))   # net of the 40 seed an execution returns through save_seed
        year = execs_tick * 365
        # arable: refill 10 000 bp a year (100%), plan share 5 000 -> sustainable flow per year = C * 0.5 * 1.0
        slack_cap = year / Fraction(1, 2) * Fraction(22, 10)     # sustainable flow = 2.2 x the lambda = 1 draw
        bind_cap = year / Fraction(1, 2) * Fraction(11, 10)      # sustainable flow = 1.1 x
        rows.append((reg, hh, ebar, float(execs_tick), float(year), int(slack_cap) + 1, int(bind_cap) + 1))
    return rows

if __name__ == "__main__":
    print("== A1. arable (refill 100% a year, regenerating only in autumn and winter at x2), capacity 2 000, 19 draws a tick for the first 20 ticks of spring")
    print("   stock at: spring start | after the draws | end of spring | end of summer | end of autumn | end of winter")
    for i, t in enumerate(arable_year()):
        print(f"   year {i}: " + " | ".join(f"{t[k]:>5}" for k in ("spring start", "after the draws", "end of spring", "end of summer", "end of autumn", "end of winter")))
    print("\n== A2. woodland (logistic 30% a year, capacity 50 000, refuge 20), maximum sustainable draw F_max = 0.3 * 50 000 / 4 = 3 750 a year")
    for label, tr in woodland_runs():
        print(f"   {label}: year:stock:abundance " + "  ".join(f"{y}:{s}:{a}" for y, s, a in tr))
    print("\n== B1. the ice field of the habitat (640 ticks a year)")
    tpy, ice_t, ex_t, dr_t, dr_y, runway = ice_runway()
    print(f"   ice needed {ice_t:.1f} a tick; cut_ice executions {ex_t:.3f} a tick at 18 ice each; draws {dr_t:.1f} a tick = {dr_y:,.0f} a year; runway of 3 000 000 units at a flat yield: {runway:.1f} years")
    yrs, mile = ice_with_curve()
    print(f"   with the abundance curve (0, 2 000) (2 000, 8 000) (10 000, 10 000): exhausted after {yrs:.1f} years; years to 80/50/20/5% left: " + ", ".join(f"{p}%: {mile[p]:.1f}" for p in (80, 50, 20, 5) if p in mile))
    print("   the viability row (finite deposit: Initial / HorizonYears a year, per tick, against the 278 a tick the households need):")
    for h, py, pt in via_row():
        print(f"      horizon {h:>2} years: {py:>10,.0f} a year = {pt:7.1f} a tick" + ("   -> short" if pt < dr_t else "   -> enough"))
    print("\n== C. the ration of spec-resources.md 5.3: 10 plots, requests A 4, B 3, C 3, D 2, E 1 (13 wanted)")
    U, rows = allocation_example()
    print(f"   U = {U}; per agent: wanted, floor share, remainder numerator (of {U}), granted")
    for r in rows: print("   ", r)
    print("\n== D. the five-region proposal (spec-resources.md 18): arable draws a plot per grow_staple execution")
    print("   region households mean-E(bp) executions/tick executions/year capacity(slack 2.2x) capacity(binding 1.1x)")
    for r in five_sizing():
        print(f"   {r[0]:<7} {r[1]:>6} {r[2]:>8} {r[3]:>12.3f} {r[4]:>11.1f} {r[5]:>14} {r[6]:>17}")

# ---- additions: the first two ticks of the arable example, and the vats of example B
def arable_first_ticks():
    """The numerators and the carry of the first two autumn ticks of A1 by hand (spec-resources.md 16.1)."""
    r = Res("refill", 2_000, 10_000, 1_320, [])
    out = []
    for _ in range(2):
        n = r.rate * (r.cap - r.s) * BP * 20_000
        s0, c0 = r.s, r.c
        r.regen(m=BP, sig=20_000)
        out.append((s0, n, c0, r.s - s0, r.c))
    return out

def vats():
    """Example B: algae vats. 640 ticks a year; seasons lit 240 ticks (yield 10 000) and dark 400 ticks (2 500) in crater_rim;
    1 000 households need 2 biomass a tick; grow_algae gives 120 biomass at yield 10 000; one vat-use drawn per execution."""
    tpy = 640
    ebar = Fraction(240 * 10_000 + 400 * 2_500, tpy)                    # mean site factor, bp
    biomass_exec = Fraction(120) * ebar / BP
    execs_tick = Fraction(2_000) / biomass_exec
    execs_year = execs_tick * tpy
    cap, rate_bp = 30_000, 20_000
    f_hat = Fraction(rate_bp, BP) * cap                                  # maximum flow, at an empty stock, mbar = 1
    phi_plan = Fraction(rate_bp, BP) * (cap - cap // 2)                  # at the plan share 5 000
    h = execs_tick / 1000
    return float(ebar), float(biomass_exec), float(execs_tick), float(execs_year), float(f_hat), float(phi_plan), float(f_hat / (h * tpy)), float(phi_plan / (h * tpy))

if __name__ == "__main__":
    print("\n== A0. the first two autumn ticks of the arable stock, by the integer rule (stock, numerator, carry before, units added, carry after); D = 3.65e14")
    for row in arable_first_ticks(): print("   ", row)
    e, b, et, ey, fh, pp, nb, npl = vats()
    print("\n== B2. the vats of the habitat (640 ticks a year)")
    print(f"   mean site factor {e:.1f} bp; biomass per execution {b:.2f}; executions {et:.2f} a tick = {ey:,.0f} a year")
    print(f"   capacity 30 000 vat-uses, refill 20 000 bp a year: maximum flow {fh:,.0f} a year; at the plan share {pp:,.0f} a year")
    print(f"   households at which the draw equals them: N_bind {nb:,.0f} (empty stock) and {npl:,.0f} (plan share); headroom at 1 000 households {fh/ey:.2f} and {pp/ey:.2f}")
