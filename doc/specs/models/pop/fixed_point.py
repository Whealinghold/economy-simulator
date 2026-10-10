"""POP model 1: the Malthusian fixed point of spec-population.md 5, and its gain/delay condition.

A stylised model of the RULES of the spec, not of the library. The population rules are the spec's integer rules
(schedules by integer interpolation, the exact carried remainder of 3.6, deaths before births, MinHouseholds floor).
The economy is replaced by one assumed function: the mean welfare ratio the households would read at population N is
R(N) = RA * (N0 / N)^eta (basis points), which stands for diminishing returns to labour on a fixed factor (a
Cobb-Douglas share 1 - eta). Floating point is used ONLY for that function and for the analysis printouts.
The ratio the households read is a first-order lag (tau_e ticks) of R(N) averaged over a window of T ticks.
Every number is a placeholder (N-015, N-027)."""
import math, sys
from collections import deque

TPY = 365                      # TicksPerYear
D = 10_000 * TPY               # denominator of the carry (3.6)

def interp(knots, x):
    """Piecewise linear, flat outside the first and last knot, increment truncated toward zero (3.4)."""
    if x <= knots[0][0]: return knots[0][1]
    if x >= knots[-1][0]: return knots[-1][1]
    for (x0, y0), (x1, y1) in zip(knots, knots[1:]):
        if x0 <= x <= x1:
            num = (y1 - y0) * (x - x0)
            q = abs(num) // (x1 - x0)
            return y0 + (q if num >= 0 else -q)

def ramp(m):
    """The gain sweep: the death ramp is narrowed to 1/m of its default width around the replacement ratio, rates
    unchanged (450 down to 50), so the slope at the crossing is m times the default and nothing is clipped."""
    w = 4_000 // m
    return [(RSTAR - w, 450), (RSTAR + w, 50)]

BIRTH = [(8_000, 250), (16_000, 250)]     # flat: Clark ch.4 draft p.92 (see spec 3.4); height is a placeholder
DEATH = [(8_000, 450), (16_000, 50)]
RSTAR = 12_000                  # where the defaults cross (250 and 250)

def run(N_start, m=1, T=30, tau_e=30, eta=0.5, years=200, RA=16_000, N0=100_000, floor=1,
        births_on=True, shock=None, dscale_fn=None, report_every=10):
    birth = BIRTH; death = ramp(m)
    Nstar = N0 * (RA / RSTAR) ** (1 / eta)
    N = N_start
    e = int(RA * (N0 / N) ** eta)
    win = deque([e] * T, maxlen=T)
    cb = cd = 0
    traj = []
    for t in range(years * TPY):
        A = RA if shock is None else shock(t, RA)
        R = int(A * (N0 / N) ** eta)
        e += int((R - e) / tau_e)
        win.append(e)
        r = sum(win) // len(win)
        ds = 10_000 if dscale_fn is None else dscale_fn(t)
        b = interp(birth, r)
        d = min(10_000, interp(death, r) * ds // 10_000)
        cb += N * b; cd += N * d
        deaths = cd // D; cd %= D
        deaths = max(0, min(deaths, N - floor))      # MinHouseholds: deaths held at the floor (counted in the spec)
        births = cb // D; cb %= D
        N = N - deaths + (births if births_on else 0)
        if t % (report_every * TPY) == 0: traj.append((t // TPY, N, r))
    return Nstar, N, traj

def classify(traj_full, Nstar):
    xs = [n for _, n in traj_full]
    sign = [1 if n > Nstar else -1 for n in xs]
    cross = sum(1 for a, b in zip(sign, sign[1:]) if a != b)
    tail = xs[-len(xs) // 4:]
    amp = (max(tail) - min(tail)) / Nstar
    return cross, amp

def run_full(**kw):
    """Same as run but returns the yearly series for classification."""
    N_start = kw.pop('N_start'); m = kw.pop('m', 1); T = kw.pop('T', 30); tau_e = kw.pop('tau_e', 30)
    eta = kw.pop('eta', 0.5); years = kw.pop('years', 300); RA = 16_000; N0 = 100_000
    birth = BIRTH; death = ramp(m)
    Nstar = N0 * (RA / RSTAR) ** (1 / eta)
    N = N_start; e = int(RA * (N0 / N) ** eta); win = deque([e] * T, maxlen=T); cb = cd = 0; series = []
    for t in range(years * TPY):
        R = int(RA * (N0 / N) ** eta)
        e += int((R - e) / tau_e); win.append(e); r = sum(win) // len(win)
        cb += N * interp(birth, r); cd += N * interp(death, r)
        deaths = min(cd // D, N - 1); cd %= D; births = cb // D; cb %= D
        N = N - deaths + births
        if t % TPY == 0: series.append((t // TPY, N))
    return Nstar, series

def k_per_year(m, eta):
    """Linearised gain k = r* g'(r*) eta / 10_000 per year, g = b - d in bp per year (slopes from the crossing knots)."""
    gprime = m * ((BIRTH[1][1] - BIRTH[0][1]) / 8_000 + (DEATH[0][1] - DEATH[1][1]) / 8_000)
    return RSTAR * gprime * eta / 10_000

if __name__ == "__main__":
    print("== 1. approach to the fixed point (defaults, eta 0.5, T 30, tau_e 30)")
    for start in (100_000, 300_000):
        Ns, N, traj = run(start)
        print(f"N* = {Ns:,.0f}; start {start:,}")
        print("  year:N :", "  ".join(f"{y}:{n:,}" for y, n, _ in traj[:12]))
        print(f"  final N {N:,} ({100*(N/Ns-1):+.2f}% of N*)")
    print("\n== 2. gain x delay (eta 0.5). k = linearised gain per year; tau = T/2 + tau_e in years; prediction: k*tau <= 1/4 monotone")
    print(f"{'m':>4} {'T':>5} {'tau_e':>6} {'k/yr':>7} {'k*tau':>7} {'crossings':>9} {'tail amp':>9}  verdict")
    for T, tau_e in ((30, 30), (365, 365), (1095, 365)):
        for m in (1, 4, 16, 64):
            Ns, series = run_full(N_start=300_000, m=m, T=T, tau_e=tau_e)
            k = k_per_year(m, 0.5); tau = (T / 2 + tau_e) / TPY
            cr, amp = classify(series, Ns)
            verdict = "monotone" if cr <= 1 and amp < 0.01 else ("damped oscillation" if amp < 0.01 else ("sustained/growing" if amp > 0.05 else "small residual"))
            print(f"{m:>4} {T:>5} {tau_e:>6} {k:7.3f} {k*tau:7.3f} {cr:>9} {amp:9.4f}  {verdict}")
    print("\n== 3. a yield shock and a plague (defaults)")
    def fall(t, RA):
        y = t / TPY
        return RA * 6 // 10 if 20 <= y < 30 else RA     # technology level at 60% for ten years
    Ns, N, traj = run(int(100_000 * (16_000/12_000) ** 2), shock=fall, years=120, report_every=5)
    print(f"yield 60% for years 20 to 30; N* = {Ns:,.0f}")
    print("  year:N:ratio ", "  ".join(f"{y}:{n:,}:{r}" for y, n, r in traj))
    def plague(t):
        y = t / TPY
        return 40_000 if 20 <= y < 21 else 10_000        # death schedule x4 for one year
    Ns, N, traj = run(int(100_000 * (16_000/12_000) ** 2), dscale_fn=plague, years=120, report_every=5)
    print("death schedule x4 for year 20")
    print("  year:N:ratio ", "  ".join(f"{y}:{n:,}:{r}" for y, n, r in traj))
