"""Model of the precaution buffer of spec-saving-and-investment.md 4 (a model of the rules, not of the library).

Integer arithmetic, floor division, placeholders (N-015, N-027). Standard library only.

Part A: one household, a two-state income, the buffer against unfed ticks.
Part B: a representative household in a circular flow with a leakage, a cut in the
        autonomous income for a stretch of ticks, and the buffer as an amplifier.
Part C: the drawdown of a producer with a long job (the working capital of its cycle).
"""

U = 100            # upkeep cost of one tick
RESERVE_TICKS = 3  # AGT ReserveTicks
SPEND_MAX = 5000   # AGT SpendMaxShare
SPEND_HALF = 10    # AGT SpendHalfTicks
T = 30             # welfare window


def cap(avail, target, u=U):
    """spec-agents.md 7.4.4 with the target of 3.3 in place of the reserve."""
    e = max(0, avail - target)
    h = SPEND_HALF * u
    if e + h == 0:
        return 0
    return SPEND_MAX * (e * h // (e + h)) // 10000


def drawdown(nets):
    """4.1: the deepest fall from a peak of the cumulative net, over the window."""
    s = peak = dd = 0
    for x in nets:
        s += x
        peak = max(peak, s)
        dd = max(dd, peak - s)
    return dd


def precaution(share_bp, cap_ticks, nets, u=U):
    if share_bp == 0:
        return 0
    return min(cap_ticks * u, share_bp * drawdown(nets) // 10000)


class Lcg:
    def __init__(self, seed):
        self.s = seed

    def below(self, bp):  # true with probability bp / 10 000
        self.s = (self.s * 6364136223846793005 + 1442695040888963407) % (1 << 64)
        return (self.s >> 33) % 10000 < bp


def part_a(share_bp, cap_ticks, ticks=20000, seed=7, y_good=160, y_bad=30):
    rng = Lcg(seed)
    good = True
    m = 600
    ring = []
    unfed = 0
    held = 0
    disc = 0
    for _ in range(ticks):
        y = y_good if good else y_bad
        pay = min(m, U)
        if pay < U:
            unfed += 1
        avail = m - pay
        target = max(RESERVE_TICKS * U, precaution(share_bp, cap_ticks, ring))
        d = min(cap(avail, target), avail)
        m = avail - d + y
        disc += d
        held += m
        ring.append(y - U)
        if len(ring) > T:
            ring.pop(0)
        good = (not rng.below(500)) if good else rng.below(1500)
    return unfed, held // ticks, disc // ticks


def part_b(share_bp, cap_ticks, leak_bp=2000, x_base=40, cut_bp=10000, cut_from=200, cut_len=30, ticks=700):
    # steady state of the circular flow: y = (1 - leak) * s + x, s = U + d
    m = 800
    s_prev = U + 100
    ring = []
    base_disc = None
    min_disc = None
    rec_tick = None
    peak_buf = 0
    min_m = m
    for t in range(ticks):
        x = x_base
        if cut_from <= t < cut_from + cut_len:
            x = x_base * (10000 - cut_bp) // 10000
        y = (10000 - leak_bp) * s_prev // 10000 + x
        pay = min(m, U)
        avail = m - pay
        prec = precaution(share_bp, cap_ticks, ring)
        target = max(RESERVE_TICKS * U, prec)
        d = min(cap(avail, target), avail)
        s = pay + d
        m = avail - d + y
        s_prev = s
        ring.append(y - U)
        if len(ring) > T:
            ring.pop(0)
        if t == cut_from - 1:
            base_disc = d
        if t >= cut_from:
            if min_disc is None or d < min_disc:
                min_disc = d
            peak_buf = max(peak_buf, prec)
            min_m = min(min_m, m)
            if rec_tick is None and t > cut_from + cut_len and d * 100 >= base_disc * 90:
                rec_tick = t - (cut_from + cut_len)
    return base_disc, min_disc, rec_tick, peak_buf, min_m


def part_c():
    # a 20-tick job: inputs 300 at the start (income -300), nothing for 19 ticks, sales 2 500 at the end
    nets = []
    for cycle in range(2):
        for k in range(20):
            inc = -300 if k == 0 else (2500 if k == 19 else 0)
            nets.append(inc - U)
    return nets[-40:], drawdown(nets[-40:])


if __name__ == "__main__":
    print("Part A: one household, income 160 (good) or 30 (bad), upkeep 100, 20 000 ticks, window 30")
    print("share_bp cap_ticks | unfed ticks | mean money | mean discretionary")
    for share, capt in [(0, 30), (5000, 30), (10000, 30), (20000, 30), (10000, 5)]:
        u, h, d = part_a(share, capt)
        print(f"{share:8d} {capt:9d} | {u:11d} | {h:10d} | {d:10d}")
    print()
    print("Part B: representative household, circular flow, leakage 2 000 bp, autonomous income 40 cut by 10 000 bp for 30 ticks at tick 200")
    print("share_bp cap_ticks | discretionary before | lowest | ticks to 90% after the cut ends | peak buffer | lowest money")
    for share, capt in [(0, 30), (5000, 30), (10000, 30), (20000, 30), (10000, 3)]:
        b, lo, rec, pk, mm = part_b(share, capt)
        print(f"{share:8d} {capt:9d} | {b:20d} | {lo:6d} | {str(rec):31s} | {pk:11d} | {mm:12d}")
    print()
    nets, dd = part_c()
    print("Part C: producer with a 20-tick job, window of 40 records: drawdown =", dd, "(inputs 300, upkeep 100, sales 2 500)")
    print()
    print("Part B2: as B with leakage 3 000 bp and a cut of 60 ticks (a deep and long one: the hole in the cash flow is 30 a tick)")
    print("share_bp cap_ticks | discretionary before | lowest | ticks to 90% after the cut ends | peak buffer | lowest money")
    for share, capt in [(0, 30), (5000, 30), (10000, 30), (20000, 30), (10000, 10)]:
        b, lo, rec, pk, mm = part_b(share, capt, leak_bp=3000, x_base=40, cut_bp=10000, cut_len=60, ticks=900)
        print(f"{share:8d} {capt:9d} | {b:20d} | {lo:6d} | {str(rec):31s} | {pk:11d} | {mm:12d}")
