"""TEC model 2: a productivity gain in the Malthusian loop of spec-population.md 5 closed by an endowment of spec-resources.md 8.

A stylised model of the RULES of the specs, not of the library. It imports the integer rules of models/res/coupled.py (the
regeneration with a carried remainder, the abundance curve, the proportional ration, the piecewise-linear schedules, the exact
carried remainder of births and deaths, deaths before births) and changes ONE thing: the ratio a household reads is
r = RMAX * f * a * T, with T = (10 000 + G) / 10 000 the technology factor of spec-technology.md. The economy is the same single
assumed function as in coupled.py; floating point appears ONLY in the printouts and in the predictions. Every number is a
placeholder (N-015, N-027); nothing here shows that any default is stable."""
import sys
from collections import deque

sys.path.insert(0, "/m/res")
from coupled import Res, build, interp, BIRTH, DEATH, RSTAR, RMAX, NSTAR, LIN, TPY, BP, DPOP, H, classify, verdict  # noqa: E402

GPRIME = (BIRTH[1][1] - BIRTH[0][1]) / 8_000 + (DEATH[0][1] - DEATH[1][1]) / 8_000     # 0.05 bp a year per bp of ratio
KAPPA = RSTAR * GPRIME / BP                                                              # 0.06 a year (eta = 1)


def sim_quota(nbind, n0, gain_of_t, years, T_w=30, tau_e=0):
    """Regime A of RES 8.2: a pure quota. The granted share is min(1, nbind / N); no stock. Returns yearly (year, N, r, G)."""
    N = n0
    cb = cd = 0
    e = RMAX * min(BP, nbind * BP // max(N, 1)) // BP * (BP + gain_of_t(0)) // BP
    win = deque([e] * T_w, maxlen=T_w)
    wsum = e * T_w
    out = []
    for t in range(years * TPY):
        G = gain_of_t(t)
        f = min(BP, nbind * BP // max(N, 1))
        r_now = RMAX * f // BP * (BP + G) // BP
        e += (r_now - e) // tau_e if tau_e > 0 else (r_now - e)
        wsum += e - win[0]
        win.append(e)
        r = wsum // T_w
        b = interp(BIRTH, r)
        d = min(BP, interp(DEATH, r))
        cd += N * d
        cb += N * b
        deaths = min(cd // DPOP, N - 1)
        cd %= DPOP
        births = cb // DPOP
        cb %= DPOP
        N = N - deaths + births
        if t % TPY == 0:
            out.append((t // TPY, N, r, G))
    return out


def sim_stock(res, n0, gain_of_t, years, T_w=30, flat=False, h=H):
    """The coupled loop of coupled.py with the technology factor in the ratio. Returns yearly (year, N, stock, r)."""
    N = n0
    cb = cd = 0
    a0 = BP if flat else res.abundance()
    e = RMAX * (BP + gain_of_t(0)) // BP * a0 // BP
    win = deque([e] * T_w, maxlen=T_w)
    wsum = e * T_w
    out = []
    for t in range(years * TPY):
        G = gain_of_t(t)
        a = BP if flat else res.abundance()
        U = N * h
        take = min(U, res.s)
        f = BP if U == 0 else take * BP // U
        res.s -= take
        r_now = RMAX * f // BP * a // BP * (BP + G) // BP
        e = r_now
        wsum += e - win[0]
        win.append(e)
        r = wsum // T_w
        b = interp(BIRTH, r)
        d = min(BP, interp(DEATH, r))
        cd += N * d
        cb += N * b
        deaths = min(cd // DPOP, N - 1)
        cd %= DPOP
        births = cb // DPOP
        cb %= DPOP
        N = N - deaths + births
        res.regen()
        if t % TPY == 0:
            out.append((t // TPY, N, res.s, r))
    return out


def part_a():
    print("== A. a pure quota (RES 8.2 regime A, eta = 1): N* = N_bind * RMAX * T / r*, the population tracks a moving N*")
    nbind = 60_000                                    # N* = 60 000 * 20 000 / 12 000 = 100 000 at T = 1
    print(f"   r* = {RSTAR}, RMAX = {RMAX}, g' = {GPRIME:.3f}, kappa = r* g' / 10 000 = {KAPPA:.3f} a year (time constant "
          f"{1 / KAPPA:.1f} years)")
    print("   A1. a trend of TrendBP of gain a year, from the fixed point N*(T = 1) = 100 000, no cap in the window")
    print("       predicted premium (the ratio above r* once the population tracks): TrendBP * 10 000 / (10 000 + G) / g'  bp")
    print(f"       {'TrendBP':>7} {'year':>5} {'G':>6} {'N/N*':>7} {'ratio':>7} {'premium':>8} {'predicted':>9}")
    for tr in (25, 50, 100):
        gain = lambda t, tr=tr: tr * t // TPY
        ser = sim_quota(nbind, NSTAR, gain, 61)
        for yr in (20, 40, 60):
            _, N, r, G = ser[yr]
            nstar = nbind * RMAX * (BP + G) // (BP * RSTAR)
            pred = tr * BP / (BP + G) / GPRIME
            print(f"       {tr:7d} {yr:5d} {G:6d} {N / nstar:7.3f} {r:7d} {r - RSTAR:8d} {pred:9.0f}")
    print("   A2. a one-off step of +1 000 bp (a discovery), from N*(T = 1): the ratio jumps, the population follows, e-fold time 1 / kappa")
    ser = sim_quota(nbind, NSTAR, lambda t: 1_000, 121)
    nstar2 = nbind * RMAX * 11_000 // (BP * RSTAR)
    print(f"       N*(T = 1.1) = {nstar2}; ratio at year 0: {ser[0][2]}, 5: {ser[5][2]}, 10: {ser[10][2]}, 20: {ser[20][2]}, 40: {ser[40][2]}, 80: {ser[80][2]}")
    dev0 = nstar2 - NSTAR
    e_fold = next((y for y, N, r, G in ser if nstar2 - N <= dev0 / 2.718281828), None)
    print(f"       the gap N*' - N falls to 1/e of its start at year {e_fold} (the linearisation: {1 / KAPPA:.1f}); "
          f"N at year 120 is {ser[120][1]} of {nstar2}")
    print("   A3. the trend reaches its cap (CapBP 1 000 at TrendBP 50: year 20): the premium is paid back, with no overshoot")
    cap = 1_000
    ser = sim_quota(nbind, NSTAR, lambda t: min(cap, 50 * t // TPY), 121)
    print("       year:  " + "  ".join(f"{y:5d}" for y in (10, 20, 30, 40, 60, 80, 120)))
    print("       ratio: " + "  ".join(f"{ser[y][2]:5d}" for y in (10, 20, 30, 40, 60, 80, 120)))
    mx = max(ser, key=lambda s: s[1])
    nstar_cap = nbind * RMAX * (BP + cap) // (BP * RSTAR)
    print(f"       N/N*' at year 120: {ser[120][1] / nstar_cap:.3f}; the largest N/N*' over the run: "
          f"{max(s[1] for s in ser) / nstar_cap:.3f} (no overshoot of the new fixed point: monotone)")
    print("   A4. eta: with a curve through the origin the elasticity of the ratio to N is below 1 and the premium is larger (1 / eta)")
    print("       see part C (a stock): the same step, a refilling stock with a linear abundance curve")


def part_b():
    print("\n== B. a refilling stock with a linear abundance curve (RES 8.2 regime B): the stock share falls with T")
    print("   sigma*(T) = r* / (RMAX T) = 0.6 / T; N*(T) = R C (1 - sigma*) / (h TPY); a(sigma*) = 10 000 r* / (r_full)")
    print(f"   {'T':>5} {'sigma* pred':>11} {'N* pred':>9} {'N at 400y':>10} {'ratio':>6} {'sigma sim':>9} {'min N/N*0':>9}")
    res0 = build("refill", 0.5)
    cap = res0.cap
    for T in (1.0, 1.1, 1.2, 1.5):
        res = build("refill", 0.5)
        G = round((T - 1) * BP)
        ser = sim_stock(res, NSTAR, lambda t: G, 400)
        sig_pred = RSTAR / (RMAX * T)
        nstar = res.rate / BP * cap * (1 - sig_pred) / (H * TPY)
        _, N, S, r = ser[-1]
        lo = min(n for _, n, _, _ in ser) / NSTAR
        print(f"   {T:5.2f} {sig_pred:11.3f} {nstar:9.0f} {N:10d} {r:6d} {S / cap:9.3f} {lo:9.3f}")
    print("   (the rate 0.5 a year is above the monotone threshold 0.16, so the approach is monotone for every T tried; the "
          "threshold 4 kappa (1 - sigma) / sigma rises as sigma* falls:)")
    for T in (1.0, 1.2, 1.5, 2.0):
        s = RSTAR / (RMAX * T)
        print(f"   T {T:4.2f}: sigma* {s:.3f}, monotone iff R >= {4 * KAPPA * (1 - s) / s:.3f} a year")


def part_c():
    print("\n== C. a logistic stock: a gain moves sigma* toward the maximum-yield point 0.5; below it the fixed point is a saddle")
    print("   sigma*(T) = 0.6 / T, so T_crit = 1.2: a gain of +20% at the start is the edge (the plan share 6 000, a linear curve):")
    print("   the validator's inequality  a(5 000) * (10 000 + CapBP) >= a(sigma_plan) * 10 000  gives CapBP >= 2 000 here")
    print(f"   {'T':>5} {'sigma*':>7} {'N*/N*0':>7} {'min N/N*0':>9} {'max N/N*0':>9} {'stock/cap 300y':>14}  outcome")
    for T in (1.00, 1.05, 1.10, 1.15, 1.18, 1.20, 1.25, 1.40):
        res = build("logistic", 1.0)
        capc = res.cap
        G = round((T - 1) * BP)
        ser = sim_stock(res, NSTAR, lambda t: G, 300)
        s = RSTAR / (RMAX * T)
        nstar = NSTAR * (s * (1 - s)) / 0.24
        xs = [n for _, n, _, _ in ser]
        lo, hi = min(xs) / NSTAR, max(xs) / NSTAR
        cr, amp, lo2, hi2 = classify(ser, nstar)
        outcome = "collapse" if min(xs) < 0.02 * NSTAR else verdict(cr, amp, min(xs) / nstar, max(xs) / nstar, True)
        print(f"   {T:5.2f} {s:7.3f} {nstar / NSTAR:7.3f} {lo:9.3f} {hi:9.3f} {ser[-1][2] / capc:14.3f}  {outcome}")
    print("   a ramp that crosses it: TrendBP of gain a year from T = 1, no cap, from the fixed point; the stock is exhausted when it falls below 2% of its capacity:")
    print(f"   {'TrendBP':>7} {'crossing':>9} {'N/N*0 at it':>12} {'exhausted':>11} {'years after':>12} {'ratio at 10y before':>20}")
    for tr in (100, 50, 25):
        res = build("logistic", 1.0)
        ser = sim_stock(res, NSTAR, lambda t, tr=tr: tr * t // TPY, 400)
        cross = 2_000 // tr
        ex = next((y for y, n, s, r in ser if s < 0.02 * res.cap), None)
        atc = ser[cross][1] / NSTAR
        before = ser[ex - 10][3] if ex else 0
        print(f"   {tr:7d} {cross:7d} y {atc:12.3f} {str(ex) + ' y':>11} {str(ex - cross) + ' y':>12} {before:20d}")


if __name__ == "__main__":
    print(f"models/tec/malthus.py: schedules of models/res/coupled.py; r* = {RSTAR}; g' = {GPRIME:.3f}; kappa = {KAPPA:.3f} a year\n")
    part_a()
    part_b()
    part_c()
