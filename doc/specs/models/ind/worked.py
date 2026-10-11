"""The worked examples of spec-indicators.md section 19: one invented world
(two regions `alpha` and `beta`, four commodities), one window of two ticks,
every preset computed by the integer rules of sections 5 to 9 and asserted.

Standard library only; run in Docker, never on the host. The setting is an
example (note N-017): the words are data and nothing branches on them.
"""
from arith import rdiv, isqrt_round, gini_bp, quantile, BP

TPY = 365                      # TicksPerYear of the default calendar
OPEN_TICK, CLOSE_TICK = 10, 12  # the window [10, 12): steps 10 and 11
N_TICKS = CLOSE_TICK - OPEN_TICK

REG = ["alpha", "beta"]
COM = ["grain", "loaf", "wood", "cart"]   # cart: the haul unit of a hired mode
CLS = {"grain": ["food"], "loaf": ["food"], "wood": ["fuel"], "cart": ["carriage"]}
G, L, W, C = 0, 1, 2, 3

# ---- window flows by region (units): the BoundaryFlow totals of the window ----
flows = {
    "alpha": {
        "recipe_output": {G: 100, L: 30, W: 20, C: 6},
        "recipe_input": {G: 60},
        "consumption": {G: 10, L: 20},
        "want_consumption": {W: 5},
        "spoilage": {G: 3},
        "disposal": {W: 2},
        "haul_use": {C: 4},
        "initial": {G: 5},          # an external agent added at tick 11
    },
    "beta": {
        "recipe_output": {G: 50, W: 30},
        "consumption": {G: 35},
        "want_consumption": {W: 10},
        "outside_offer": {W: 20},
        "outside_unsold": {W: 5},
        "outside_purchases": {L: 6},
    },
}
# interregional: ShipmentArrived lines of the window (origin, dest, commodity)
arrived = [("alpha", "beta", G, dict(shipped=10, delivered=9, diverted=0, spoiled=1))]

# ---- price per (region, commodity): the window unit values ----
price = {
    "alpha": {G: 10, L: 25, W: 6, C: 5},
    "beta": {G: 13, L: 28, W: 8, C: 6},
}
# alpha grain as it is formed from the market reports of the two ticks
alpha_grain_reports = [(10, 30), (11, 20)]      # (Price, Volume)
sumPV = sum(p * v for p, v in alpha_grain_reports)
sumV = sum(v for _, v in alpha_grain_reports)
assert rdiv(sumPV, sumV) == 10                  # 520 / 50 = 10.4 -> 10

# ---- the two snapshots: held by agents + estate, pending, in flight ----
# opening stocks (units) per (region, commodity): held (agents + estate), pending
open_held = {"alpha": {G: 200, L: 40, W: 25, C: 3}, "beta": {G: 60, L: 10, W: 40, C: 0}}
open_pend = {"alpha": {L: 12}, "beta": {}}
open_transit = {"alpha": {}, "beta": {}}        # lines of shipments, by origin
close_transit = {"alpha": {G: 6}, "beta": {}}   # a shipment still in flight
close_pend = {"alpha": {L: 8}, "beta": {}}


def get(d, r, c):
    return d.get(r, {}).get(c, 0)


# interregional arrivals: units that leave an origin's system and join a
# destination's (Delivered + Diverted); Spoiled is a spoilage flow of the origin
inter_in = {r: {c: 0 for c in range(4)} for r in REG}
inter_out = {r: {c: 0 for c in range(4)} for r in REG}
for o, d, c, ln in arrived:
    inter_in[d][c] += ln["delivered"] + ln["diverted"]
    inter_out[o][c] += ln["delivered"] + ln["diverted"]
    # the spoiled units are part of alpha's spoilage total of the window
    assert flows[o]["spoilage"][c] >= ln["spoiled"]

# the closing held stock follows from the ledger (conservation, spec 7.3)
close_held = {}
for r in REG:
    close_held[r] = {}
    for c in range(4):
        f = lambda k: get(flows[r], k, c) if k in flows[r] else 0
        out = f("recipe_output")
        inp = f("recipe_input")
        use = (f("consumption") + f("want_consumption") + f("spoilage") + f("disposal")
               + f("haul_use") + f("outside_purchases"))
        ext_in = f("outside_offer") - f("outside_unsold") + f("initial")
        d_pend = get(close_pend, r, c) - get(open_pend, r, c)
        d_transit = get(close_transit, r, c) - get(open_transit, r, c)
        # system stock (held + own transit lines + pending) changes by:
        #   out - in + (accrual: pending is part of output, so it cancels) ...
        # held change = out - in - use + ext_in + inter_in - inter_out - d_transit - d_pend_effect
        # pending: units fixed at starts left the held stock as inputs and
        # return as output at completion; the created output is already in `out`.
        d_held = out - inp - use + ext_in + inter_in[r][c] - inter_out[r][c] - d_transit
        close_held[r][c] = get(open_held, r, c) + d_held

# ---- 7.1 / 7.2: production, expenditure, and the identity ----
def value(r, qty):
    return sum(price[r][c] * q for c, q in qty.items())


acct = {}
for r in REG:
    out = flows[r].get("recipe_output", {})
    inp = flows[r].get("recipe_input", {})
    d_pend = {c: get(close_pend, r, c) - get(open_pend, r, c) for c in range(4)}
    d_held = {c: close_held[r][c] - get(open_held, r, c) for c in range(4)}
    d_trans = {c: get(close_transit, r, c) - get(open_transit, r, c) for c in range(4)}
    P_flow = value(r, out) - value(r, inp)
    P_acc = P_flow + value(r, d_pend)
    Cc = value(r, flows[r].get("consumption", {})) + value(r, flows[r].get("want_consumption", {}))
    dS = value(r, d_held) + value(r, d_trans) + value(r, d_pend)
    loss = (value(r, flows[r].get("spoilage", {})) + value(r, flows[r].get("disposal", {})))
    carriage = value(r, flows[r].get("haul_use", {}))
    ext = value(r, flows[r].get("outside_purchases", {})) - (
        value(r, flows[r].get("outside_offer", {})) - value(r, flows[r].get("outside_unsold", {})))
    endow = value(r, flows[r].get("initial", {}))
    inter = value(r, inter_out[r]) - value(r, inter_in[r])      # net interregional exports
    rhs = Cc + dS + loss + carriage + ext + inter - endow
    acct[r] = dict(P_flow=P_flow, P_acc=P_acc, C=Cc, dS=dS, loss=loss, carriage=carriage,
                   ext=ext, inter=inter, endow=endow, rhs=rhs)
    assert P_acc == rhs, (r, P_acc, rhs)

realm_P = sum(acct[r]["P_acc"] for r in REG)
realm_inter = sum(acct[r]["inter"] for r in REG)
# the same identity at one price vector: the interregional term vanishes
one = {c: price["alpha"][c] for c in range(4)}
q_inter = {c: sum(inter_out[r][c] - inter_in[r][c] for r in REG) for c in range(4)}
assert all(v == 0 for v in q_inter.values())

# ---- 8.4: delivered price of the lane alpha -> beta for grain ----
# the hired shipment redeemed 4 haul units, bought at alpha's price of cart
haul_units, hauled_units = 4, 9      # units delivered + diverted of the 10 shipped (one spoiled in transit)
freight_per_unit = rdiv(haul_units * price["alpha"][C], hauled_units)   # 20 / 9 = 2.2 -> 2
delivered = price["alpha"][G] + freight_per_unit                           # 12
gap_local = price["beta"][G] - price["alpha"][G]                           # 3
gap_after_freight = price["beta"][G] - delivered                           # 1

# ---- 5.3: price index of the class `food`, base window prices ----
base_p = {G: 8, L: 20}
now_p = {G: 10, L: 24}
q0 = {G: 12, L: 18}   # the base window's household consumption (units)
qt = {G: 10, L: 20}   # this window's
lasp = rdiv(BP * sum(now_p[c] * q0[c] for c in q0), sum(base_p[c] * q0[c] for c in q0))
paas = rdiv(BP * sum(now_p[c] * qt[c] for c in qt), sum(base_p[c] * qt[c] for c in qt))
fish = isqrt_round(lasp * paas)

# ---- 5.4 and 5.5: inequality and the distribution of money ----
rows = [(1, 0), (2, 20), (4, 40), (1, 60)]
g_bp, gN, gD = gini_bp(rows)
q10, q50, q90 = quantile(rows, 1000), quantile(rows, 5000), quantile(rows, 9000)
share_below_40 = rdiv(BP * sum(w for w, v in rows if v < 40), sum(w for w, _ in rows))

# ---- 9: money by sector, per household, velocity ----
Mh, Mp, Mc = 5000, 800, 200
H = 100
M = Mh + Mp + Mc
money_per_household = rdiv(Mh, H)
turnover = 3000
velocity_bp = rdiv(BP * turnover, M)                       # per window
velocity_year_bp = rdiv(BP * turnover * TPY, M * N_TICKS)   # from exact sums, one rounding


# ---- trade balance of node `n` at gate beta ----
budget, unspent, receipts = 200, 32, 120
export_value = budget - unspent
import_value = receipts
net = export_value - import_value
imports_w = flows["beta"]["outside_offer"][W] - flows["beta"]["outside_unsold"][W]
exports_l = flows["beta"]["outside_purchases"][L]

# ---- real wage: wage per working household-tick to the upkeep cost ----
wage = 24
upkeep = {G: 2, L: 1}
upkeep_cost = sum(q * price["alpha"][c] for c, q in upkeep.items())      # 2*10 + 25 = 45
real_wage_bp = rdiv(BP * wage, upkeep_cost)

# ---- price spread across the two regions for grain ----
spread_bp = rdiv(2 * BP * (price["beta"][G] - price["alpha"][G]), price["beta"][G] + price["alpha"][G])   # one rounding: 60000 / 23

# ---- time: the annualisation of a window total ----
flow_window = 1150
annual = rdiv(flow_window * TPY, N_TICKS)

# ---- 7.2.4: value added by recipe on the completion basis, and the WIP row ----
recipes = {  # recipe: (region, executions, outputs {commodity: units}, inputs per execution)
    "grow": ("alpha", 50, {G: 100}, {}),
    "bake": ("alpha", 30, {L: 30}, {G: 2}),
    "cut": ("alpha", 10, {W: 20}, {}),
    "make_cart": ("alpha", 6, {C: 6}, {}),
}
by_recipe = {}
for k, (r, ex, outs, ins) in recipes.items():
    by_recipe[k] = value(r, outs) - ex * value(r, ins)
sum_recipes = sum(by_recipe.values())
wip_row = sum_recipes - acct["alpha"]["P_acc"]          # completed less started
assert sum_recipes == acct["alpha"]["P_flow"] == 1300
assert wip_row == -value("alpha", {L: get(close_pend, "alpha", L) - get(open_pend, "alpha", L)})
by_class = {"food": by_recipe["grow"] + by_recipe["bake"], "fuel": by_recipe["cut"],
            "carriage": by_recipe["make_cart"]}

# ---- 9 rows 4, 15, 17 to 20: the alpha / beta market figures ----
ref_p = {"alpha": {G: 8, L: 24}, "beta": {G: 12, L: 26}}
upkeep_hh = {G: 2, L: 1}
B = {r: sum(q * ref_p[r][c] for c, q in upkeep_hh.items()) for r in REG}          # 40, 50
level = {r: rdiv(BP * sum(q * price[r][c] for c, q in upkeep_hh.items()), B[r]) for r in REG}
cost_now = {r: sum(q * price[r][c] for c, q in upkeep_hh.items()) for r in REG}
level_ratio = rdiv(BP * cost_now["beta"] * B["alpha"], B["beta"] * cost_now["alpha"])   # one rounding
pinned = rdiv(BP * (6 + 1), 50)                                       # (AtFloor + AtCeiling) / Outcomes
dispersion = rdiv(BP * (12 - 9), 10)                                  # (High - Low) / Mean centre
price_range = rdiv(BP * 11, 10)                                       # High / Low of 10, 11
offset = rdiv(BP * (10 - 9), 9)                                       # window price 10 against VIA's 9
cover = rdiv(BP * close_held["alpha"][G] * N_TICKS, 10)               # held * n / consumed (Cons 10)
estate_grain = 37
unsold_share = rdiv(BP * estate_grain, close_held["alpha"][G])

# ---- 18.7: households account, running counts and the unfed share ----
grower_open = 40
hh = {"grower": grower_open, "baker": 10}
log = []
hh["grower"] += 7 - 4; log.append(("WeightChanged", 3))
hh["grower"] -= 1; log.append(("AgentBankrupted", -1))
tick10_h = hh["grower"]                                               # households at tick 10 = 42
hh["grower"] -= 1; hh["baker"] += 1; log.append(("RoleChanged", 1))
beta_grower = 2
hh["grower"] -= 2; log.append(("AgentRelocated", -2))
tick11_h = hh["grower"]                                               # 39
assert (tick10_h, tick11_h) == (42, 39)
unfed2 = [4, 6]
start10 = grower_open                                                       # running count at TickStarted of tick 10
start11 = tick10_h                                                        # at TickStarted of tick 11
unfed_hh_ticks_bp = rdiv(BP * sum(unfed2), start10 + start11)            # 10 / 82
account_residual = (hh["grower"] + hh["baker"] + beta_grower) - (grower_open + 10)
assert account_residual == 3 - 1                                      # births 3 less the death of one household
births_done, deaths_done = 3, 0

# ---- 18.8: the ratio distribution, with a negative ratio ----
ratio_rows = [(1, -5000), (2, 8000), (4, 10000), (1, 25000)]
Wr = sum(w for w, _ in ratio_rows)
ratio_mean = rdiv(sum(w * v for w, v in ratio_rows), Wr)
rq10, rq50, rq90 = quantile(ratio_rows, 1000), quantile(ratio_rows, 5000), quantile(ratio_rows, 9000)
ratio_below_1 = rdiv(BP * sum(w for w, v in ratio_rows if v < 10000), Wr)
# inequality of money: Top(1000), Bottom(4000), cohort share
S_m = sum(w * v for w, v in rows)
top_k = -(-8 * 1000 // BP)                  # ceil(8 * 1000 / 10000) = 1
assert top_k == 1
top_share = rdiv(BP * 60, S_m)              # the single top household holds 60
bottom_k = -(-8 * 4000 // BP)               # ceil(3.2) = 4
assert bottom_k == 4
bottom_value = 1 * 0 + 2 * 20 + 1 * 40     # (1, 0), (2, 20) and 1 of the 4 of (4, 40)
bottom_share = rdiv(BP * bottom_value, S_m)
cohort_share = rdiv(BP * sum(w for w, _ in rows if w > 1), sum(w for w, _ in rows))
openness = rdiv(BP * (value("beta", {L: 6}) + value("beta", {W: 15})), realm_P)

# ---- 18.9: mean of ratios against the ratio of sums; a stationary one-tick world ----
agents = [(1, 20, 10), (1, 20, 40)]          # (weight, income, cost)
mean_of_ratios = rdiv(sum(w * rdiv(BP * i, c) for w, i, c in agents), sum(w for w, _, _ in agents))
agg_ratio = rdiv(BP * sum(w * i for w, i, _ in agents), sum(w * c for w, _, c in agents))
# A (grower) makes 4 grain and sells 4 to B at 10; B makes 2 loaf from them, sells 2 to A at 25;
# A eats the 2 loaf. Stationary: stocks do not change.
pr = {"grain": 10, "loaf": 25}
income_A = 4 * pr["grain"]                                  # sales, no inputs
income_B = 2 * pr["loaf"] - 4 * pr["grain"]                 # sales less inputs bought
gdp_P = 4 * pr["grain"] + 2 * pr["loaf"] - 4 * pr["grain"]  # out - in
gdp_E = 2 * pr["loaf"]                                      # consumption; no stock change
assert income_A + income_B == gdp_P == gdp_E == 50

# ---- 18.10: calendar year and season windows with StartDay 100; `cold` wraps the year end ----
DAYS = 365
START_DAY = 100
WARM_FROM, WARM_TO = 59, 259                 # season 0 `warm`: days 59..258; season 1 `cold`: days 259..364 and 0..58 (wraps)


def moment(t):
    d = START_DAY + t
    return d // DAYS, d % DAYS, 0 if WARM_FROM <= d % DAYS < WARM_TO else 1


def first_tick(pred, lo=0, hi=10_000):
    while lo < hi:
        mid = (lo + hi) // 2
        if pred(mid):
            hi = mid
        else:
            lo = mid + 1
    return lo


y1 = first_tick(lambda t: moment(t)[0] >= 1)
y2 = first_tick(lambda t: moment(t)[0] >= 2)
year0, year1 = (0, y1), (y1, y2)
season_windows = []          # maximal runs of one season, across the year boundary, clipped to [0, y2)
t = 0
horizon = y2
while t < horizon:
    s = moment(t)[2]
    e = t
    while e < horizon and moment(e)[2] == s:
        e += 1
    season_windows.append((s, t, e))
    t = e
assert year0 == (0, 265) and year1 == (265, 630)
assert season_windows == [(0, 0, 159), (1, 159, 324), (0, 324, 524), (1, 524, 630)], season_windows

if __name__ == "__main__":
    print("closing held stock:", close_held)
    for r in REG:
        print(r, acct[r])
    print("realm P (accrual):", realm_P, " realm interregional term:", realm_inter)
    print("freight per unit:", freight_per_unit, "delivered:", delivered,
          "gap local:", gap_local, "gap after freight:", gap_after_freight)
    print("food index: Laspeyres", lasp, "Paasche", paas, "product", lasp * paas, "Fisher", fish)
    print("gini bp", g_bp, f"({gN}/{gD})", "P10/P50/P90", q10, q50, q90, "share below 40:", share_below_40)
    print("M", M, "M_h/H", money_per_household, "V window bp", velocity_bp, "V year bp", velocity_year_bp)
    print("export value", export_value, "import value", import_value, "net", net,
          "import wood units", imports_w, "export loaf units", exports_l)
    print("upkeep cost", upkeep_cost, "real wage bp", real_wage_bp)
    print("spread bp", spread_bp)
    print("annualised", annual)
    print("by recipe", by_recipe, "sum", sum_recipes, "WIP row", wip_row, "by class", by_class)
    print("basket B", B, "level", level, "beta against alpha", level_ratio)
    print("pinned", pinned, "dispersion", dispersion, "range", price_range, "offset", offset,
          "cover (bp of a tick)", cover, "unsold share", unsold_share)
    print("households per tick", tick10_h, tick11_h, "unfed household-ticks bp", unfed_hh_ticks_bp,
          "account residual", account_residual)
    print("ratio mean", ratio_mean, "P10/P50/P90", rq10, rq50, rq90, "share below 1", ratio_below_1)
    print("top", top_share, "bottom", bottom_share, "cohort share", cohort_share, "openness", openness)
    print("mean of ratios", mean_of_ratios, "aggregate", agg_ratio, "stationary GDP", gdp_P)
    print("year windows", year0, year1, "season windows", season_windows)
