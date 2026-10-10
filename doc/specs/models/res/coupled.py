"""RES model 3: the Malthusian loop of spec-population.md 5 closed by an endowment of spec-resources.md (N-098).

A stylised model of the RULES of the two specs, not of the library. Integer rules: the regeneration of spec-resources.md 4
(carried remainder; Python ints stand for the checked 128-bit helper), the abundance curve and the proportional ration of 5,
and the population rules of spec-population.md (piecewise-linear schedules by truncating integer interpolation, the exact
carried remainder of 3.8, deaths before births). The ECONOMY is replaced by one assumed function: the welfare ratio a
household reads is r = RMAX * f * a, with f the share of its draw that it was granted and a the abundance factor; the
signal is a boxcar mean over a window of T ticks after an optional first-order lag. Floating point appears ONLY in the
printouts and in the predictions. Every number is a placeholder (N-015, N-027); nothing here shows that any default is
stable."""
from collections import deque

TPY = 365
BP = 10_000
DPOP = BP * TPY                                    # POP carry denominator (spec-population.md 3.8)
BIRTH = [(8_000, 250), (16_000, 250)]
DEATH = [(8_000, 450), (16_000, 50)]
RSTAR = 12_000                                     # where the defaults cross
RMAX = 20_000                                      # ratio of a household that gets its full draw at full abundance
H = 1                                              # units a household draws a tick (executions a tick times the draw)
NSTAR = 100_000
LIN = [(0, 0), (10_000, 10_000)]                   # abundance proportional to the stock share

def interp(knots, x):
    if x <= knots[0][0]: return knots[0][1]
    if x >= knots[-1][0]: return knots[-1][1]
    for (x0, y0), (x1, y1) in zip(knots, knots[1:]):
        if x0 <= x <= x1:
            num = (y1 - y0) * (x - x0); q = abs(num) // (x1 - x0)
            return y0 + (q if num >= 0 else -q)

def ceil_div(a, b): return -((-a) // b)

class Res:
    """One endowment. shape in {'constant','refill','logistic'}; rate: units/year (constant) or bp/year; refuge in units."""
    def __init__(self, shape, cap, rate, init, curve, refuge=0, tpy=TPY):
        self.shape, self.cap, self.rate, self.refuge, self.curve, self.tpy = shape, cap, rate, refuge, curve, tpy
        self.s, self.c = init, 0
    def abundance(self):
        sig = self.s * BP // self.cap if self.cap else 0
        return interp(self.curve, sig) if self.curve else BP
    def regen(self, m=BP, sig=BP):
        d = 10**12 * self.tpy
        if self.shape == "constant": n = self.rate * m * sig * BP
        elif self.shape == "refill": n = self.rate * (self.cap - self.s) * m * sig if self.s < self.cap else 0
        else:
            sp = max(self.s, self.refuge)
            w = 0 if sp <= 0 or sp >= self.cap else ceil_div(sp * (self.cap - sp), self.cap)
            n = self.rate * w * m * sig
        add, c2 = divmod(self.c + n, d)
        if self.s + add >= self.cap: self.s, self.c = self.cap, 0
        else: self.s, self.c = self.s + add, c2

def kappa_pop():
    """k of spec-population.md 5.4 with eta = 1: r* g'(r*) / 10 000 per year, g' from the crossing knots."""
    gp = (BIRTH[1][1] - BIRTH[0][1]) / 8_000 + (DEATH[0][1] - DEATH[1][1]) / 8_000
    return RSTAR * gp / BP

def simulate(res, n0, T=30, tau_e=0, years=300, shock=None, flat=False, h=H, rmax=RMAX):
    """Run N and the stock together. Returns the yearly series of (year, N, S, ratio signal)."""
    N = n0; cb = cd = 0
    a0 = BP if flat else res.abundance()
    e = rmax * a0 // BP; win = deque([e] * T, maxlen=T); wsum = e * T
    out = []
    for t in range(years * TPY):
        if shock: shock(t, res)
        a = BP if flat else res.abundance()
        U = N * h
        take = min(U, res.s)
        f = BP if U == 0 else take * BP // U
        res.s -= take
        r_now = rmax * f // BP * a // BP
        e += (r_now - e) // tau_e if tau_e > 0 else (r_now - e)
        wsum += e - win[0]; win.append(e); r = wsum // T
        b = interp(BIRTH, r); d = min(BP, interp(DEATH, r))
        cd += N * d; cb += N * b
        deaths = min(cd // DPOP, N - 1); cd %= DPOP
        births = cb // DPOP; cb %= DPOP
        N = N - deaths + births
        res.regen()
        if t % TPY == 0: out.append((t // TPY, N, res.s, r))
    return out

def classify(series, Nstar):
    """crossings of N*, tail amplitude over N*, smallest and largest N/N*."""
    xs = [n for _, n, _, _ in series]
    sign = [1 if n > Nstar else -1 for n in xs]
    cross = sum(1 for a, b in zip(sign, sign[1:]) if a != b)
    tail = xs[-len(xs) // 4:]
    return cross, (max(tail) - min(tail)) / Nstar, min(xs) / Nstar, max(xs) / Nstar

def verdict(cross, amp, lo, hi, above):
    if lo < 0.02: return "collapse"
    if amp > 0.05: return "sustained cycle"
    side = (1 - lo) if above else (hi - 1)
    if side < 0.005: return "monotone"
    if amp < 0.01: return f"damped oscillation ({'undershoot' if above else 'overshoot'} {100*side:.1f}% of N*)"
    return "small residual"

def build(shape, rate_year, sig_star=0.6, curve=LIN, refuge=0):
    """Capacity chosen so that the fixed point is N* = 100 000 whatever the rate: F(S*) = N* h (a year of draws)."""
    draw_year = NSTAR * H * TPY
    if shape == "refill":
        cap = round(draw_year / (rate_year * (1 - sig_star)))
        return Res("refill", cap, round(rate_year * BP), int(sig_star * cap), curve)
    if shape == "logistic":
        cap = round(draw_year / (rate_year * sig_star * (1 - sig_star)))
        return Res("logistic", cap, round(rate_year * BP), int(sig_star * cap), curve, refuge=refuge * cap // BP)
    cap = 10**9
    return Res("constant", cap, draw_year, int(sig_star * cap), curve)

if __name__ == "__main__":
    kp = kappa_pop()
    print(f"POP gain with eta = 1: kappa = r* g'(r*) / 10 000 = {kp:.3f} per year; N* = {NSTAR:,}; RMAX = {RMAX}; r* = {RSTAR}; sigma* = r*/RMAX = {RSTAR/RMAX:.2f}")
    print("\n== 1. refilling stock (shape refill), abundance proportional to the stock share, window T = 1 (no delay), start 1.6 N*")
    thr = 4 * kp * (1 - 0.6) / 0.6
    print("   Linearisation: two eigenvalues of  l^2 + Rf l + Gamma = 0,  Gamma = kappa * F(S*)/S* = kappa * Rf * (1-sigma*)/sigma*.")
    print(f"   Monotone iff Rf^2 >= 4 Gamma, i.e. Rf >= 4 kappa (1-sigma*)/sigma* = {thr:.3f} a year; below it a damped oscillation")
    print(f"   {'Rf/yr':>6} {'predicted':>11} {'crossings':>9} {'min N/N*':>9}  simulated")
    for rf in (0.02, 0.05, 0.10, 0.16, 0.30, 0.50, 1.0, 3.0):
        res = build("refill", rf)
        ser = simulate(res, int(1.6 * NSTAR), T=1, years=600)
        cr, amp, lo, hi = classify(ser, NSTAR)
        pred = "monotone" if rf >= thr else "oscillates"
        print(f"   {rf:6.2f} {pred:>11} {cr:9d} {lo:9.3f}  {verdict(cr, amp, lo, hi, True)}")
    print("\n== 2. the same with the welfare window T = 365 (a delay of about half a year, spec-population.md 5.4) and a lag of 365")
    print(f"   {'Rf/yr':>6} {'crossings':>9} {'min N/N*':>9}  simulated")
    for rf in (0.16, 0.5, 1.0, 3.0):
        res = build("refill", rf)
        ser = simulate(res, int(1.6 * NSTAR), T=365, tau_e=365, years=600)
        cr, amp, lo, hi = classify(ser, NSTAR)
        print(f"   {rf:6.2f} {cr:9d} {lo:9.3f}  {verdict(cr, amp, lo, hi, True)}")
    print("\n== 3. logistic stock, r_s = 1 a year, sigma* = 0.6 (above the maximum-yield point 0.5), T = 30, different starts above N*")
    print("   The draw at N* is 96% of the maximum sustainable draw F_max = r_s C / 4; a start whose draw exceeds F_max cannot recover.")
    print(f"   {'start/N*':>8} {'draw/F_max':>10} {'crossings':>9} {'min N/N*':>9}  simulated")
    for start in (1.02, 1.04, 1.05, 1.1, 1.6):
        res = build("logistic", 1.0)
        ser = simulate(res, int(start * NSTAR), T=30, years=600)
        cr, amp, lo, hi = classify(ser, NSTAR)
        dm = start * 0.24 / 0.25
        print(f"   {start:8.2f} {dm:10.3f} {cr:9d} {lo:9.3f}  {verdict(cr, amp, lo, hi, True)}")
    print("   and a fixed point below the maximum-yield point (sigma* = 0.4, RMAX 30 000): unstable even for a small push")
    for start in (0.98, 1.02):
        res = build("logistic", 1.0, sig_star=0.4)
        ser = simulate(res, int(start * NSTAR), T=30, years=600, rmax=30_000)
        cr, amp, lo, hi = classify(ser, NSTAR)
        print(f"   start {start:.2f} N*: crossings {cr}, min N/N* {lo:.3f}, max {hi:.3f}: {verdict(cr, amp, lo, hi, start > 1)}")
    print("\n== 4. constant inflow (F' = 0): neutral in the linearisation; starts 1.1 N* and 0.9 N* and 1.6 N*, T = 30")
    for start in (0.9, 1.1, 1.6):
        res = build("constant", 0)
        ser = simulate(res, int(start * NSTAR), T=30, years=600)
        cr, amp, lo, hi = classify(ser, NSTAR)
        print(f"   start {start} N*: crossings {cr}, tail amp {amp:.3f}, N/N* between {lo:.2f} and {hi:.2f}: {verdict(cr, amp, lo, hi, start > 1)}")
    print("\n== 5. the flat curve (a pure quota, eta = 1 above the binding point), refill, T = 30")
    print("   N_bind = Rf * C / (365 h): the households at which the draw equals the refill at an empty stock;")
    print("   for N > N_bind the granted share is f = N_bind / N and r = RMAX * f, so N* = N_bind * RMAX / r*")
    for rf in (0.3, 3.0):
        draw_year = NSTAR * H * TPY
        cap = round(draw_year * (RSTAR / RMAX) / rf)               # N_bind = 0.6 N*(model), so the predicted N* is N_bind * RMAX / r* = N*(model)
        nbind = rf * cap / (TPY * H)
        nstar_pred = nbind * RMAX / RSTAR
        for start in (1.6, 0.6):
            res = Res("refill", cap, round(rf * BP), cap, [])
            ser = simulate(res, int(start * NSTAR), T=30, years=800, flat=True)
            cr, amp, lo, hi = classify(ser, NSTAR)
            print(f"   Rf {rf:4.1f}, start {start} N*(model): N_bind {nbind:,.0f}, predicted N* {nstar_pred:,.0f}; final N {ser[-1][1]:,} ({ser[-1][1]/nstar_pred:.3f} of predicted); crossings {cr}; tail amp {amp:.4f}")
