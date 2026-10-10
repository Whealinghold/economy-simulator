"""Model 1 of spec-subsistence.md section 17: a pinned staple market with and without the
own-use mode.  A stylised model of the RULES of the spec, not of the library.

Producers: N_S self-employed households under the target-income rule of
spec-welfare-and-wages.md 5 with the work floor MinWorkShare (spec-stability.md 6.1):
days worked D = clamp(ceil(K / (A p)), DMIN, DMAX) of a year of T ticks, A units a working
tick, so supply per tick is N_S * A * D / T.  They are not consumers of the model.

Consumers: N_C households that each need U units of the staple per tick, work their role (a wage
w_i, uniform over [W_LO, W_HI]) every tick they are not in own-use mode, buy up to U * (1 + COVER)
units before eating with a hard budget (spec-agents.md 7.3, 7.4), and eat.  In own-use mode the
household works the own-use recipe instead (F units a tick, no wage, a source of Duration 1).  The
mode follows spec-subsistence.md 5: Secure = Cover + Afford (Afford is 0 when the household was
unfed last tick); enter below ENTER; leave when Cover reaches EXIT; after RETRY consecutive
own-use jobs the next free decision is a role job (the market test).  buyexit=True is the rejected
variant of 5.3 (also leave when the last bid was filled and the money buys EXIT ticks).

The market clears by proportional rationing at the posted price; the price moves as in
spec-stability.md 6.1: with probability one half by STEP basis points of itself in the
direction of excess demand, clamped to [PMIN, PMAX].  Integer arithmetic throughout.
There is no income feedback: a consumer's wage does not depend on the price, and the producers
are not consumers.  The model shows the structure of the rule, not the library.
"""
import random

T = 365
N_S, A, K, P0 = 1000, 10, 1_500_000, 1000
N_C, U = 1370, 3
W_LO, W_HI = 3600, 7200
STEP, PMIN, PMAX, GATE = 200, 100, 10_000, 5000
COVER_TARGET = 3          # ticks of cover a household tries to hold (AGT CoverTicks)


def supply(p, dmin):
    d = -(-K // (A * p))
    d = max(dmin, min(T, d))
    return N_S * A * d // T


def ration(bids, S):
    """Proportional rationing with the leftover units to the largest remainders, ties to the
    lower index (spec-resources.md 5.3 without the hash key: the model needs no fairness over runs)."""
    Q = sum(bids)
    got = [b * S // Q for b in bids]
    left = S - sum(got)
    order = sorted(range(len(bids)), key=lambda i: (-(bids[i] * S % Q), i))
    for i in order:
        if left <= 0:
            break
        if got[i] < bids[i]:
            got[i] += 1
            left -= 1
    return got


def run(own, enter=2, exit_=3, f=2, dmin=1, seed=1, ticks=600, p_start=1100, wlo=W_LO, whi=W_HI, retry=0, outage=None, p_fixed=False, buyexit=False):
    rng = random.Random(seed)
    w = [rng.randint(wlo, whi) for _ in range(N_C)]
    m = [U * P0 * 5 for _ in range(N_C)]      # five ticks of upkeep at the reference price
    s = [U * COVER_TARGET for _ in range(N_C)]   # cover 3 at the start
    flag = [False] * N_C
    jobs_ = [0] * N_C         # consecutive own-use jobs since the last role job (spec 5.4)
    hunger = [0] * N_C
    bought = [False] * N_C     # the market delivered to the household last tick (own order outcome)
    p = p_start
    series = []
    for t in range(ticks):
        # 1. the mode (spec 5.3): own state and the price of the own market
        entered = left = 0
        if own:
            for i in range(N_C):
                cover = s[i] // U
                afford = 0 if hunger[i] > 0 else m[i] // (p * U)
                secure = cover + afford
                if not flag[i] and secure < enter:
                    flag[i] = True; jobs_[i] = 0; entered += 1
                elif flag[i] and (cover >= exit_ or (buyexit and bought[i] and afford >= exit_)):
                    flag[i] = False; jobs_[i] = 0; left += 1
        # 2. work: the own-use recipe (a source of duration 1) or the role
        for i in range(N_C):
            if flag[i] and (not retry or jobs_[i] < retry):
                s[i] += f
                jobs_[i] += 1
            else:
                m[i] += w[i]
                jobs_[i] = 0
        # 3. bids: up to the cover target, hard budget
        bids = []
        for i in range(N_C):
            want = max(0, U * (1 + COVER_TARGET) - s[i])
            q = min(want, m[i] // p)
            bids.append(q)
        Q = sum(bids)
        S = supply(p, dmin)
        if outage and outage[0] <= t < outage[1]:
            S = 0                     # no seller at all: the market has failed at any price
        # 4. clearing at the posted price by proportional rationing
        if Q <= S:
            got = bids
        else:
            got = ration(bids, S)
        for i in range(N_C):
            s[i] += got[i]
            m[i] -= got[i] * p
            bought[i] = got[i] > 0
        # 5. consumption
        unfed = 0
        for i in range(N_C):
            if s[i] >= U:
                s[i] -= U; hunger[i] = 0
            else:
                s[i] = 0; hunger[i] += 1; unfed += 1
        series.append((p, Q, S, sum(1 for x in flag if x), unfed, entered + left))
        # 6. price rule
        ex = Q - S
        sig = 1 if ex > 0 else (-1 if ex < 0 else 0)
        if sig and rng.randrange(10000) < GATE:
            if not p_fixed:
                p = max(PMIN, min(PMAX, p + sig * max(1, p * STEP // 10000)))
    return series


def summarise(label, series):
    flips = sum(x[5] for x in series)
    tail = series[-150:]
    ps = [x[0] for x in tail]
    print(f"{label:46s} p_end={series[-1][0]:6d} tail min/max {min(ps):6d}/{max(ps):6d} "
          f"peak p {max(x[0] for x in series):6d} peak unfed {max(x[4] for x in series):5d} "
          f"unfed household-ticks {sum(x[4] for x in series):8d} peak on-own-use {max(x[3] for x in series):5d} mode changes {flips:6d}")


if __name__ == "__main__":
    print("N_S=%d A=%d K=%d P0=%d N_C=%d U=%d wages [%d,%d] supply at P0 %d, demand at P0 %d"
          % (N_S, A, K, P0, N_C, U, W_LO, W_HI, supply(P0, 1), N_C * U))
    cases = (
        ("floor off", dict(own=False)),
        ("own-use f=2, enter 2 exit 3, no retry", dict(own=True, f=2)),
        ("own-use f=2, enter 2 exit 3, retry 10", dict(own=True, f=2, retry=10)),
        ("own-use f=3, enter 2 exit 3, no retry", dict(own=True, f=3)),
        ("own-use f=3, enter 2 exit 3, retry 10", dict(own=True, f=3, retry=10)),
        ("own-use f=4, enter 2 exit 3, retry 10", dict(own=True, f=4, retry=10)),
        ("own-use f=3, enter 3 exit 4 (exit above cover), retry 10", dict(own=True, f=3, enter=3, exit_=4, retry=10)),
    )
    for dmin_label, dmin in (("DMIN 1 (no work floor)", 1), ("DMIN 146 (MinWorkShare 4000)", 146)):
        print(f"== {dmin_label}")
        for label, kw in cases:
            runs = sorted((run(dmin=dmin, seed=seed, **kw) for seed in range(5)), key=lambda sr: sr[-1][0])
            summarise(label + " (median of 5)", runs[2])
            print(f"     final prices over 5 seeds: {[sr[-1][0] for sr in runs]}")
    print("== an outage: no seller at all for ticks 100 to 199, the price held at the reference (the money is good, the market is empty)")
    for label, kw in (("floor off", dict(own=False)),
                      ("own-use f=3, enter 2 exit 3, retry 10", dict(own=True, f=3, retry=10)),
                      ("own-use f=2, enter 2 exit 3, retry 10", dict(own=True, f=2, retry=10)),
                      ("own-use f=4, enter 2 exit 3, retry 10", dict(own=True, f=4, retry=10))):
        series = run(dmin=1, p_start=P0, outage=(100, 200), p_fixed=True, **kw)
        summarise(label, series)
    print("== exit also by purchase: leave the mode when the last bid was filled and the money buys the exit cover")
    for dmin_label, dmin in (("DMIN 1", 1), ("DMIN 146", 146)):
        for label, kw in (("own-use f=3, exit by cover", dict(own=True, f=3, retry=10)),
                          ("own-use f=3, exit by cover or purchase", dict(own=True, f=3, retry=10, buyexit=True)),
                          ("own-use f=2, exit by cover or purchase", dict(own=True, f=2, retry=10, buyexit=True)),
                          ("own-use f=4, exit by cover or purchase", dict(own=True, f=4, retry=10, buyexit=True))):
            runs = sorted((run(dmin=dmin, seed=seed, **kw) for seed in range(5)), key=lambda sr: sr[-1][0])
            summarise(f"{dmin_label} {label}", runs[2])
