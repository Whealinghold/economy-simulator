"""The arithmetic of the worked examples of spec-saving-and-investment.md 16 (rules of 3 to 6).

Integers only, floor division unless stated. Placeholders (N-015, N-027). Standard library only.
It runs the rules as the spec states them so that the numbers in the text are checked.
"""

WEYL = 2654435769


def weyl32(i):
    return (i * WEYL) % (1 << 32)


def pv(net, duration):
    """value per tick, scaled by 10 000 (5.2.1)."""
    return net * 10000 // duration


def job_value(pv_b, pv_alt, w, cap):
    """J(c, K): the best job value per tick (scaled) of an agent of weight w with K executions of the site recipe."""
    return max(pv_b * min(w, cap), pv_alt * w)


def rate(decay, wear, k, duration):
    return decay + -(-wear * k // duration)


def build_test(w, cap_now, cls, pv_b, pv_alt, inputs_cost, duration_b, dur_ben, hurdle_bp, payback):
    k = min(w, cls["capacity"])
    r = rate(cls["decay"], cls["wear"], k, dur_ben)
    life = cls["durability"] // r if r else payback
    t_p = min(life, payback)
    dj = job_value(pv_b, pv_alt, w, cap_now + cls["capacity"]) - job_value(pv_b, pv_alt, w, cap_now)
    benefit = dj * t_p // 10000
    jnow = job_value(pv_b, pv_alt, w, cap_now)
    opp = jnow * duration_b // 10000
    cost = inputs_cost + opp
    return dict(rate=r, life=life, t_p=t_p, dj=dj, benefit=benefit, opp=opp, cost=cost,
                passes=benefit * 10000 >= hurdle_bp * cost, ratio_bp=benefit * 10000 // cost if cost else 0)


def drawdown(nets):
    s = peak = dd = 0
    for x in nets:
        s += x
        peak = max(peak, s)
        dd = max(dd, peak - s)
    return dd


if __name__ == "__main__":
    print("Example A: a mill (a pre-industrial example; unit: one grind)")
    U = 30
    cls = dict(durability=600, decay=1, wear=1, capacity=6)
    grind_net, grind_dur = 20, 1       # 3 flour at 20, 4 grain at 10: 60 - 40
    carry_net, carry_dur = 8, 1
    inputs = 30 * 8 + 10 * 12          # 30 timber at 8, 10 stone at 12
    t = build_test(w=1, cap_now=0, cls=cls, pv_b=pv(grind_net, grind_dur), pv_alt=pv(carry_net, carry_dur),
                   inputs_cost=inputs, duration_b=5, dur_ben=grind_dur, hurdle_bp=15000, payback=180)
    print("  pv grind", pv(grind_net, grind_dur), "pv carry", pv(carry_net, carry_dur), "inputs", inputs)
    for k_, v_ in t.items():
        print("  ", k_, v_)
    safety = 2
    outlay = inputs + U * (5 + 1 + safety)
    reserve = 3 * U
    print("  outlay", outlay, "reserve", reserve, "funds needed", outlay + reserve)
    hs = [(i, 15000 + weyl32(i) * 2000 // (1 << 32)) for i in (3, 7, 11)]
    print("  hurdles with a spread of 2 000 for ids 3, 7, 11:", hs)
    # the household saves 14 a tick above upkeep; money 100 at tick 0
    m = 100
    tick = 0
    while m < outlay + reserve:
        m += 14
        tick += 1
    print("  money 100 and a surplus of 14 a tick reaches", m, "at tick", tick)
    print("  a commit band of 2 000: hurdle", 15000 * (10000 - 2000) // 10000)

    print()
    print("Example B: the buffer of a producer with a 20-tick job")
    nets = []
    for cycle in range(2):
        for k in range(20):
            inc = -300 if k == 0 else (2500 if k == 19 else 0)
            nets.append(inc - 100)
    w = nets[-40:]
    dd = drawdown(w)
    print("  drawdown over the last 40 records", dd)
    for share, capt in [(10000, 30), (5000, 30), (10000, 10)]:
        print("  share", share, "cap ticks", capt, "-> precaution", min(capt * 100, share * dd // 10000), "(flat reserve 300)")

    print()
    print("Example C: a cold cell (a space-colony example; unit: one cell), a store class with a Worth row")
    for worth in (15, 3):
        benefit = worth * 100
        cost = 200 + 5 * 4
        print("  worth", worth, "benefit", benefit, "cost", cost, "ratio bp", benefit * 10000 // cost,
              "passes at 15 000:", benefit * 10000 >= 15000 * cost)

    print()
    print("Example A2: the repair of the mill at condition 120 of 600")
    cond, dur_, n_f = 120, 600, 1
    cap_f = -(-6 * cond // dur_)          # ceil(Capacity * Condition / Durability) = 2
    missing = dur_ - cond
    units = 100
    m_ex = -(-missing // units)           # least m with m * units >= missing = 5
    restored = min(missing, m_ex * units)
    r_ = rate(1, 1, 1, 1)
    inputs_rep = m_ex * 4 * 8             # 4 timber at 8 per execution
    pv_b_, pv_alt_ = pv(20, 1), pv(8, 1)
    dj1 = job_value(pv_b_, pv_alt_, 1, cap_f) - job_value(pv_b_, pv_alt_, 1, max(0, cap_f - 6))
    t_p = min(restored // r_, 180 * n_f)
    benefit = dj1 * t_p // 10000
    jnow = job_value(pv_b_, pv_alt_, 1, cap_f)
    opp = jnow * 1 // 10000               # the repair recipe lasts 1 tick
    cost = inputs_rep + opp
    print("   cap(f)", cap_f, "missing", missing, "m", m_ex, "restored", restored, "rate", r_, "t_p", t_p)
    print("   delta J per facility", dj1, "benefit", benefit, "inputs", inputs_rep, "opportunity", opp, "cost", cost,
          "ratio bp", benefit * 10000 // cost, "passes at 15 000:", benefit * 10000 >= 15000 * cost)
    # the build of the same class at the state of A2 (cap_now 2): no gain
    t2 = build_test(w=1, cap_now=cap_f, cls=dict(durability=600, decay=1, wear=1, capacity=6), pv_b=pv_b_, pv_alt=pv_alt_,
                    inputs_cost=360, duration_b=5, dur_ben=1, hurdle_bp=15000, payback=180)
    print("   a build at cap_now", cap_f, ": dJ", t2["dj"], "benefit", t2["benefit"], "passes:", t2["passes"])

    print()
    print("Example A3: reach, the goal in progress and the funding tick (5.5.6, 5.6.3, 5.6.5), example A's numbers")
    U, safety, floor_ = 30, 2, 90
    outlay_a = 360 + U * (5 + 1 + safety)
    goal = outlay_a - 0
    t_p = 180
    for money_, ring_n, ring_c in [(100, 30, 420), (100, 30, 60), (400, 30, 0), (690, 0, 0)]:
        need = max(0, goal + floor_ - money_)
        reach = need * ring_n <= max(0, ring_c) * t_p
        print("   money", money_, "ring n", ring_n, "C_n", ring_c, "need", need, "reachable:", reach)
    for remaining in (4, 3, 1):
        print("   build in progress, remaining", remaining, "-> goal", U * (remaining + 1 + safety))
    # the funding tick: the tier 2 spend on the project's own inputs is already out of avail
    money_ = 700
    tier2 = 360
    avail = money_ - tier2
    goal_full = outlay_a
    goal_net = max(0, goal_full - tier2)
    print("   funding tick: money", money_, "tier 2 on the inputs", tier2, "avail", avail,
          "e with the full goal", max(0, avail - (floor_ + goal_full)),
          "e with the goal net of tier 2", max(0, avail - (floor_ + goal_net)))

    print()
    print("Example C2: the Worth row saturates at Count (5.3.2), a store class, Count 2")
    for held in (0, 1, 2):
        count = 2
        benefit = 15 * 100 if held < count else 0
        print("   facilities held", held, "benefit", benefit, "passes at 15 000:", benefit * 10000 >= 15000 * 220 and benefit > 0)

    print()
    print("Example D: the tool of a build recipe that the household does not hold (5.4, 5.6.1)")
    inputs_d, tool_price, held_tool = 360, 50, 0
    toolbuy = max(0, 1 - held_tool) * tool_price
    print("   inputs", inputs_d, "tool to buy", toolbuy, "outlay", inputs_d + toolbuy + U * (5 + 1 + safety))
