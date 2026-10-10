"""The integer arithmetic of the worked examples of spec-merchants.md 19 (every number is a placeholder).

Example A: a river valley, a hired mode, a broker that sells to a consignee under a delivery
contract. Example B: a mining colony, a self-carried mode, a trader that buys at a gate.
Integer arithmetic only; the rules are those of sections 5 and 6 of the spec.
"""

G64 = 0x9E3779B97F4A7C15
MASK = (1 << 64) - 1


def weyl_bp(agent_id):
    x = ((agent_id + 1) * G64) & MASK
    return (x * 10000) >> 64


def ceil_div(a, b):
    return -((-a) // b)


def trunc_div(a, b):
    """Division truncating toward zero (architecture.md 3.5)."""
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def fee_share(base, spread, agent_id):
    u = weyl_bp(agent_id)
    phi = base + trunc_div(spread * (2 * u - 10000), 10000)
    return u, max(1, min(10000, phi))


def half_even_mid(a, b):
    s = a + b
    q, r = divmod(s, 2)
    if r == 1 and q % 2 == 1:
        q += 1
    return q


def per_unit(unit_mass, unit_bulk, mass, bulk):
    caps = []
    if unit_mass > 0 and mass > 0:
        caps.append(unit_mass // mass)
    if unit_bulk > 0 and bulk > 0:
        caps.append(unit_bulk // bulk)
    return min(caps)


def units_needed(mass, bulk, unit_mass, unit_bulk):
    n = 1
    if unit_mass > 0:
        n = max(n, ceil_div(mass, unit_mass))
    if unit_bulk > 0:
        n = max(n, ceil_div(bulk, unit_bulk))
    return n


def line(label, value):
    print(f"{label:<62} {value}")


print("== Example A: a river valley (hired mode, contract) ==")
# graph: upper -> lower 60, lower -> upper 70; speed 30; info_speed 40
d_up_low, d_low_up = 60, 70
speed, haul_span, info_speed = 30, 10, 40
k = ceil_div(d_up_low, speed)
s = ceil_div(d_up_low, haul_span)
lag = 1 + ceil_div(d_low_up, info_speed)
line("travel time upper to lower k = ceil(60 / 30)", k)
line("haul units per carrying unit s = ceil(60 / 10)", s)
line("lag of the news from lower to upper = 1 + ceil(70 / 40)", lag)

mass, bulk = 2, 6
unit_mass, unit_bulk = 1200, 900
cap = per_unit(unit_mass, unit_bulk, mass, bulk)
line("fuel per carrying unit = min(1200 / 2, 900 / 6)", cap)
q = 1 * cap
n = units_needed(q * mass, q * bulk, unit_mass, unit_bulk)
line("carrying units for a lot of 150 fuel", n)

price_rumour, mean_rumour, source_age = 310, 305, 0
rhat = min(price_rumour, mean_rumour)
cut = 500
rnet = rhat * (10000 - cut) // 10000
line("rumour (price 310, mean 305, source age 0): Rhat = min", rhat)
line("hired: Rnet = Rhat * (10 000 - 500) / 10 000, floor", rnet)
pacq = 200
p_haul = 700
cm = n * s * p_haul
line("local fuel price Pacq", pacq)
line("freight Cm = n * s * 700", cm)
g = q * (rnet - pacq)
line("gross margin G = 150 * (289 - 200)", g)

for agent in (31, 32, 33, 34):
    u, phi = fee_share(5000, 2500, agent)
    test = cm * 10000 <= phi * g
    per_haul = g // (n * s)
    ceiling = per_haul * phi // 10000
    breakeven = rnet - ceil_div(cm * 10000, phi * q)
    asklimit = pacq + ceil_div(cm * 10000, phi * q)
    print(f"  agent {agent}: u = {u:5d} bp, share phi = {phi:5d} bp, test Cm*10000 <= phi*G: {test}, "
          f"per haul unit {per_haul}, haul bid ceiling {ceiling}, break-even buy {breakeven}, offer limit {asklimit}")

u, phi = fee_share(5000, 2500, 31)
asklimit = pacq + ceil_div(cm * 10000, phi * q)
take_limit = 300 * (10000 - 1000) // 10000
price = half_even_mid(asklimit, take_limit)
line("agent 31: offer limit L", asklimit)
line("consignee: take limit b = 300 * (10 000 - 1 000) / 10 000", take_limit)
line("contract price = midpoint half to even of (L + b) / 2", price)
escrow = price * q
line("escrow = price * 150", escrow)
avg_cost = 205
line("deliverer outcome: escrow - 150 * 205 - freight", escrow - q * avg_cost - cm)

print()
print("A perishable on the same lane (ShelfLife 3): fresh lot bought at t, shipped at t + 1 (age 1)")
shelf = 3
line("delivered iff k <= ShelfLife - 1 - age = 3 - 1 - 1", f"{k} <= {shelf - 1 - 1}: {k <= shelf - 1 - 1}")
shelf = 4
line("with ShelfLife 4", f"{k} <= {shelf - 1 - 1}: {k <= shelf - 1 - 1}")

print()
print(f"Settlement of the contract of agent 31 (price {price} a unit, 150 units, escrow {escrow})")
for label, reached in (("all reached", 150), ("128 reached, 22 spoiled", 128), ("none reached", 0)):
    pay = price * reached
    refund = price * (q - reached)
    line(f"  {label}: escrow_out to the deliverer / escrow_refund to the payer", f"{pay} / {refund}  (sum {pay + refund})")

print()
print("The cost comparison on the same lane: a self-carried 'porter' (speed 20, unit (60, 40), crew 1) against the barge")
k_p = ceil_div(d_up_low, 20)
cap_p = per_unit(60, 40, mass, bulk)
line("porter: travel time ceil(60 / 20), fuel per unit min(60 / 2, 40 / 6)", f"{k_p}, {cap_p}")
q_p = cap_p
g_p = q_p * (rhat - pacq)
upkeep_cost = 1 * 200
time_p = k_p * upkeep_cost * 10000 // 10000
line("porter: lot, gross margin G = 6 * (305 - 200), time cost 3 * 200", f"{q_p}, {g_p}, {time_p}")
u, phi = fee_share(5000, 2500, 31)
line("porter: cost <= phi * G / 10 000 ?", f"{time_p} <= {phi * g_p // 10000}: {time_p * 10000 <= phi * g_p}")

print()
print("== Example B: a mining colony (self-carried mode, a gate) ==")
d_hub_pit, speed_b = 250, 100
kb = ceil_div(d_hub_pit, speed_b)
line("travel time hub to pit ceil(250 / 100)", kb)
lag_b = 1 + ceil_div(250, 100)
line("lag of the news from pit to hub = 1 + ceil(250 / 100)", lag_b)
price_r, mean_r = 80, 78
rhat_b = min(price_r, mean_r)
pacq_b = 56
unit_mass_b, unit_bulk_b = 2000, 1500
cap_b = per_unit(unit_mass_b, unit_bulk_b, 1, 1)
line("rations per carrying unit", cap_b)
avail, other_inputs = 6000, 3 * 35
limit_max = 69
q_aff = (avail - other_inputs) // 58
line("lot affordable: floor((6 000 - 3 * 35) / 58)", q_aff)
qb = 100
wear_units = 5 * kb
wear_cost = ceil_div(wear_units * 4000, 400)
feed_cost = 1 * 1 * kb * 30
upkeep_cost_b = 1 * 56
time_cost = kb * upkeep_cost_b * 10000 // 10000
cs = wear_cost + feed_cost + time_cost
line("wear: ceil(5 * 3 * 4 000 / 400)", wear_cost)
line("feed: 1 * 1 * 3 cells at 30", feed_cost)
line("time: 3 * (1 * 56) * 10 000 / 10 000", time_cost)
line("trip cost Cs", cs)
gb = qb * (rhat_b - pacq_b)
line("gross margin G = 100 * (78 - 56)", gb)
u, phi = fee_share(5000, 2500, 17)
line("agent 17: u, phi", f"{u}, {phi}")
line("test Cs * 10 000 <= phi * G", f"{cs * 10000} <= {phi * gb}: {cs * 10000 <= phi * gb}")
be = rhat_b - ceil_div(cs * 10000, phi * qb)
line("break-even buy price", be)
line("share of the gross margin the trip costs, bp", cs * 10000 // gb)
print()
print("Weyl u(a) of the first 12 agent IDs from 1 (bp):", [weyl_bp(i) for i in range(1, 13)])
