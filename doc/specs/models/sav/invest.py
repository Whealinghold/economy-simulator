"""Model of the project test of spec-saving-and-investment.md 5 (a model of the rules, not of the library).

N households may each own one site (a mill). The price of its output falls with the number
of working sites. A household that has the funds applies the test of 5.4 with a belief of the
price (a lagged average of the published price), buys the inputs, starts a build of D ticks,
and the site then works and decays. Integers, placeholders (N-015, N-027); standard library only.

It shows what the rule's own parts do (the funds, the hurdle, the spread, the commit band).
It is a model of an invented market for the output, not of the library's clearing.
"""

N = 60
Q = 10            # units of output per tick of a working site
C_RUN = 100       # running cost per tick of a working site (inputs of its recipe)
V0 = 20           # net value per tick of the best alternative (the wage)
A, B = 833, 40    # demand: units = A - B * price
DUR = 5           # build duration
C_BUILD = 3000    # inputs of the build recipe
DURABILITY, DECAY = 600, 2
LIFE = DURABILITY // DECAY
WEYL = 2654435769


def weyl32(i):
    return (i * WEYL) % (1 << 32)


def price(k):
    p = (A - Q * k) * 100 // B       # price in hundredths
    return max(300, min(2500, p))


def run(hurdle_bp=15000, spread_bp=0, band_bp=0, payback=180, uniform=True, rich=False, ticks=1500, seed_rate=(8, 16)):
    money = []
    for i in range(N):
        if rich:
            money.append(C_BUILD + 100)
        elif uniform:
            money.append(weyl32(i * 7 + 3) * C_BUILD // (1 << 32))
        else:
            money.append(0)
    rate = [seed_rate[0] + weyl32(i * 13 + 5) * (seed_rate[1] - seed_rate[0] + 1) // (1 << 32) for i in range(N)]
    hurdle = [hurdle_bp + weyl32(i) * spread_bp // (1 << 32) for i in range(N)]
    state = ["idle"] * N          # idle, holding, building, owner
    left = [0] * N                # ticks left of the build / condition of the site
    cond = [0] * N
    belief = price(0)
    k_series = []
    builds = ruins = flaps = 0
    own_ticks = [0] * N
    own_gain = [0] * N
    for t in range(ticks):
        k = sum(1 for s in state if s == "owner")
        p = price(k)
        belief = belief + (p - belief) * 3000 // 10000
        for i in range(N):
            money[i] += rate[i]
            s = state[i]
            if s == "owner":
                v = p * Q // 100 - C_RUN
                money[i] += v
                own_ticks[i] += 1
                own_gain[i] += v - V0
                cond[i] -= DECAY
                if cond[i] <= 0:
                    state[i] = "idle"
                    ruins += 1
            elif s == "building":
                left[i] -= 1
                if left[i] <= 0:
                    state[i] = "owner"
                    cond[i] = DURABILITY
            elif s == "holding":
                g = belief * Q // 100 - C_RUN - V0
                ben = g * min(LIFE, payback)
                cost = C_BUILD + V0 * DUR
                h = hurdle[i] * (10000 - band_bp) // 10000
                if g > 0 and ben * 10000 >= h * cost:
                    state[i] = "building"
                    left[i] = DUR
                    builds += 1
                else:
                    money[i] += C_BUILD * 9 // 10   # inputs sold back at a loss
                    state[i] = "idle"
                    flaps += 1
            else:
                g = belief * Q // 100 - C_RUN - V0
                ben = g * min(LIFE, payback)
                cost = C_BUILD + V0 * DUR
                if g > 0 and ben * 10000 >= hurdle[i] * cost and money[i] >= C_BUILD:
                    money[i] -= C_BUILD
                    state[i] = "holding"
        k_series.append(k)
    tail = k_series[300:]
    return max(k_series), min(tail), max(tail), sum(tail) // len(tail), builds, ruins, flaps


if __name__ == "__main__":
    print("N = 60 households, one site each at most; life 300 ticks; hurdle, spread, band in basis points")
    print("case                                         | max K | min/max K after tick 300 | mean K | builds | ruins | flaps")
    cases = [
        ("everyone funded at t=0, hurdle 15 000", dict(rich=True)),
        ("uniform wealth, hurdle 15 000", dict()),
        ("uniform wealth, hurdle 10 000", dict(hurdle_bp=10000)),
        ("uniform wealth, hurdle 25 000", dict(hurdle_bp=25000)),
        ("uniform wealth, spread 5 000", dict(spread_bp=5000)),
        ("uniform wealth, spread 5 000, band 2 000", dict(spread_bp=5000, band_bp=2000)),
        ("funded at t=0, spread 5 000, band 2 000", dict(rich=True, spread_bp=5000, band_bp=2000)),
        ("funded at t=0, payback 60", dict(rich=True, payback=60)),
        ("funded at t=0, spread 20 000", dict(rich=True, spread_bp=20000)),
        ("funded at t=0, spread 40 000", dict(rich=True, spread_bp=40000)),
        ("funded at t=0, spread 80 000", dict(rich=True, spread_bp=80000)),
        ("uniform wealth, spread 40 000, band 2 000", dict(spread_bp=40000, band_bp=2000)),
        ("funded at t=0, spread 40 000, band 2 000", dict(rich=True, spread_bp=40000, band_bp=2000)),
    ]
    for name, kw in cases:
        mx, lo, hi, mean, b, r, f = run(**kw)
        print(f"{name:44s} | {mx:5d} | {lo:10d} {hi:13d} | {mean:6d} | {b:6d} | {r:5d} | {f:5d}")
    # who owns: ownership ticks by the household's rate of saving (the self-financing exclusion, 6.1)
    ticks = 1500
    # re-run once, uniform wealth, spread 40 000, band 2 000, keeping the ownership ticks per household
    def owners_by_rate():
        money = [weyl32(i * 7 + 3) * C_BUILD // (1 << 32) for i in range(N)]
        rate = [8 + weyl32(i * 13 + 5) * 9 // (1 << 32) for i in range(N)]
        hurdle = [15000 + weyl32(i) * 40000 // (1 << 32) for i in range(N)]
        state = ["idle"] * N
        left = [0] * N
        cond = [0] * N
        belief = price(0)
        own = [0] * N
        for t in range(ticks):
            k = sum(1 for s in state if s == "owner")
            p = price(k)
            belief = belief + (p - belief) * 3000 // 10000
            for i in range(N):
                money[i] += rate[i]
                s = state[i]
                if s == "owner":
                    money[i] += p * Q // 100 - C_RUN
                    own[i] += 1
                    cond[i] -= DECAY
                    if cond[i] <= 0:
                        state[i] = "idle"
                elif s == "building":
                    left[i] -= 1
                    if left[i] <= 0:
                        state[i] = "owner"
                        cond[i] = DURABILITY
                elif s == "holding":
                    g = belief * Q // 100 - C_RUN - V0
                    if g > 0 and g * min(LIFE, 180) * 10000 >= hurdle[i] * 8 // 10 * (C_BUILD + V0 * DUR):
                        state[i] = "building"
                        left[i] = DUR
                    else:
                        money[i] += C_BUILD * 9 // 10
                        state[i] = "idle"
                else:
                    g = belief * Q // 100 - C_RUN - V0
                    if g > 0 and g * min(LIFE, 180) * 10000 >= hurdle[i] * (C_BUILD + V0 * DUR) and money[i] >= C_BUILD:
                        money[i] -= C_BUILD
                        state[i] = "holding"
        order = sorted(range(N), key=lambda i: (rate[i], i))
        low = sum(own[i] for i in order[: N // 3])
        high = sum(own[i] for i in order[-(N // 3):])
        return low, high, sum(own)
    low, high, tot = owners_by_rate()
    print()
    print(f"ownership ticks by saving rate (uniform wealth, spread 40 000, band 2 000): slowest third {low}, fastest third {high}, all {tot}")
