"""The integer arithmetic of the two worked examples of spec-employment.md section 19.

A prototype of the spec's rules, not of the library. Standard library only. Every
number is a labelled placeholder (N-015, N-027); the settings (a river valley and a
mining colony) are examples, flagged as such.
"""
def half_even_div(n, d):
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q

def ceil_div(n, d):
    return -(-n // d)

def wage(reserve, bid, share_bp):
    """5.4: the wage per household-tick, half to even, between the reservation and the bid."""
    return reserve + half_even_div(share_bp * (bid - reserve), 10000)

print("== Example A: a river valley, a mill that hires hands (placeholders) ==")
# beliefs of the miller: flour centre 700, grain centre 150; recipe mill_grain: 3 grain -> 2 flour, Duration 1
value = 2 * 700 * 10000 // 10000               # q_o * E * Centre / 10 000, E = 10 000 (HireYieldBP 10 000)
cost = 3 * 150
net = value - cost
bid = net // 1                                  # net value per tick of labour (Duration 1)
print("value per execution", value, "input cost", cost, "net", net, "bid per tick", bid)
w_hand = wage(0, bid, 5000)
print("hand (no recipe, reservation 0): wage", w_hand, "employer's margin per execution", bid - w_hand)
# the miller: money 6 000, grain 12, own execution 1 (3 grain), MaxHires 4, upkeep cost 300 at its beliefs
money, grain, own_k, maxhires = 6000, 12, 1, 4
upkeep_cost = 300
by_stock = grain // 3 - own_k
by_cash = (money - upkeep_cost) // (bid * 1)
positions = min(maxhires, by_stock, by_cash)
print("positions: by stock", by_stock, "by cash", by_cash, "MaxHires", maxhires, "->", positions)
offers = 5
hired = min(positions, offers)
escrow = w_hand * 1 * hired
print("hands offering", offers, "hired", hired, "unmatched households", offers - hired,
      "escrow", escrow, "miller's cash after formation", money - escrow)
print("output created for the miller at P2.7 (d = 1):", hired * 2, "flour; grain taken", hired * 3,
      "; wage paid", hired * w_hand, "; refund", escrow - hired * w_hand)
# the hand's welfare: upkeep 2 grain at 150 = 300 a tick
print("a hired hand's ratio on a working tick:", w_hand * 10000 // 300, "bp; days to reach a target of 15 000 in a 365-tick year:",
      ceil_div(15000 * 300 * 365, 10000 * w_hand))
# the employer-side identity (section 11.2): hired settles at E*u_e/((1-phi)*a) with employers who eat upkeep only
phi = 5000
print("h* with E=1 miller, upkeep 2 units, a=2 flour: ", 1 * 2 / ((1 - phi / 10000) * 2), "hired households; places 4")

print()
print("== Example B: a mining colony, a rig operator that hires crew (placeholders) ==")
# recipe drill_ore: Duration 3, input charge x1, requires rig (WearPerUse 1, Durability 500), output ore x5
ore, charge, rig_price = 400, 120, 20000
hire_yield = 8000                                # unskilled crew: HireYieldBP 8 000
value = 5 * hire_yield * ore // 10000            # floor(q_o * E * Centre / 10 000)
wear = ceil_div(1 * rig_price, 500)
net = value - (charge + wear)
bid = net // 3
print("value per execution", value, "charge", charge, "wear", wear, "net", net, "bid per tick", bid)
# the crew household has its own recipe gather_scrap: net 60 per tick of its own labour (reservation)
reserve = 60
w_crew = wage(reserve, bid, 5000)
print("crew with own alternative", reserve, "-> wage per household-tick", w_crew,
      "; an unskilled hand with reservation 0 -> wage", wage(0, bid, 5000))
d = 3
print("escrow of one crew household for the job:", w_crew * d, "; employer margin per execution:", (bid - w_crew) * d)
# the rig operator's cash: 1 500; two rigs held, own execution 1 -> one place left
money = 1500
print("employer funds for one hire:", money >= w_crew * d, "(", money, "against", w_crew * d, ")")
# a pay schedule of the job: start tick s at P2.7, then P2.1 on s+1 and s+2
print("wage released at s, s+1, s+2: ", [w_crew] * 3, "; completion at s+2 refunds", w_crew * d - 3 * w_crew)
# a fall of the worker's weight 3 -> 2 after the second tick: executions scale, escrow remainder refunded at the end
k0, k1 = 3, 2
paid = w_crew * k0 * 2 + w_crew * k1 * 1
print("weight 3 -> 2 after two ticks: paid", paid, "of an escrow of", w_crew * d * k0, "refund", w_crew * d * k0 - paid)

print()
print("== The reservation of a household with an own recipe (4.3.2) ==")
net_own, dur_own = 180, 3
print("own recipe net", net_own, "per", dur_own, "ticks -> reservation per tick", net_own // dur_own)

print()
print("== The closed form of section 11.2: hired households settle at h* (see circuit.py) ==")
E, ue, a = 10, 3, 6
for phi in (3000, 5000, 7500, 9000):
    for smax in (0, 5000):
        h = E * ue * (10000 + smax * 10 // 1) / 10000 / ((1 - phi / 10000) * a)
        # SpendHalfTicks 10: the asymptote of the cap is SpendMaxShare * 10 ticks of upkeep
        print("phi %5d SpendMaxShare %5d  h* = %.1f of %d places" % (phi, smax, min(h, 100), 100))
