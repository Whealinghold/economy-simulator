"""Integer arithmetic of the worked example of spec-transport.md section 17.

Everything here is a labelled placeholder (an example setting), not a result.
Run in Docker: docker run --rm -v "$PWD":/w -w /w python:3.12-slim python worked.py
"""
from itertools import product

REGIONS = ["north", "mid", "south"]
EDGES = {  # directed edges: (from, to) -> distance (spec-regions.md owns the graph)
    ("north", "mid"): 40, ("mid", "north"): 40,
    ("mid", "south"): 60, ("south", "mid"): 60,
    ("north", "south"): 90, ("south", "north"): 110,
}
INF = 10 ** 18


def shortest():
    d = {(a, b): (0 if a == b else EDGES.get((a, b), INF)) for a in REGIONS for b in REGIONS}
    for k, i, j in product(REGIONS, REGIONS, REGIONS):
        if d[(i, k)] + d[(k, j)] < d[(i, j)]:
            d[(i, j)] = d[(i, k)] + d[(k, j)]
    return d


def ceil_div(a, b):
    return -(-a // b)


def ticks(distance, speed, factor_bp=10_000):
    return max(1, ceil_div(distance * 10_000, speed * factor_bp))


def units_needed(load_mass, load_bulk, unit_mass, unit_bulk):
    n = 1
    if unit_mass > 0:
        n = max(n, ceil_div(load_mass, unit_mass))
    if unit_bulk > 0:
        n = max(n, ceil_div(load_bulk, unit_bulk))
    return n


GOODS = {  # mass, bulk, shelf life, price
    "staple": (3, 2, 0, 300), "fuel": (2, 6, 0, 100), "fresh": (1, 1, 3, 200),
    "fodder": (2, 3, 0, 40),
}


def load(hold):
    m = sum(q * GOODS[c][0] for c, q in hold.items())
    b = sum(q * GOODS[c][1] for c, q in hold.items())
    return m, b


d = shortest()
print("shortest distances")
for a in REGIONS:
    print(" ", a, [d[(a, b)] for b in REGIONS])
assert d[("south", "north")] == 100 and d[("north", "south")] == 90

print("travel ticks (foot speed 20, wagon and coach speed 30)")
for a, b in product(REGIONS, REGIONS):
    if a != b:
        print(f"  {a}->{b} dist {d[(a, b)]:>3}: foot {ticks(d[(a, b)], 20)}, wagon {ticks(d[(a, b)], 30)}")

print("example 1: a foot traveller, weight 1, unit (40, 30)")
hold = {"staple": 10, "fuel": 1, "fresh": 4}
m, b = load(hold)
print("  load", (m, b), "fits (40, 30):", m <= 40 and b <= 30, "units", units_needed(m, b, 40, 30))
hold2 = dict(hold, fuel=3)
m2, b2 = load(hold2)
print("  with fuel 3:", (m2, b2), "fits:", m2 <= 40 and b2 <= 30)
t = ticks(d[("north", "south")], 20)
print("  north->south ticks", t, "dispatched tick 10, arrives tick", 10 + t)
born, shelf = 10, 3
spoils_end_of = born + shelf - 1
print("  fresh born 10, shelf life 3 spoils at the end of tick", spoils_end_of, "; arrival", 10 + t, "-> delivered:", 10 + t <= spoils_end_of)
for tt in (1, 2, 3):
    print("  trip of", tt, "ticks from tick 10: arrival", 10 + tt, "delivered:", 10 + tt <= spoils_end_of)

print("example 2: a wagon, weight 2, crew 2, one vehicle (cap 1200, 900), feed 3 fodder, wear 5 per tick")
tw = ticks(d[("north", "south")], 30)
units = 1
print("  north->south ticks", tw, "feed", 3 * units * tw, "wear per vehicle", 5 * tw, "durability left", 400 - 5 * tw)

print("example 3: hire by coach, unit (1200, 900), span 10")
cargo = {"staple": 100}
m, b = load(cargo)
n = units_needed(m, b, 1200, 900)
s = ceil_div(d[("mid", "south")], 10)
tc = ticks(d[("mid", "south")], 30)
print("  load", (m, b), "units", n, "haul units", n * s, "ticks", tc)
# the producing recipe: one household, 4 ticks, 12 fodder, tool wear 20 of 400 on a 6000 item, wage 1000
inputs = 12 * 40
wear = ceil_div(20 * 6000, 400)
labour = 4 * 1000
cost = inputs + wear + labour
out = 4
unit_cost = ceil_div(cost * 10_000, out * 10_000)
print("  recipe cost", inputs, wear, labour, "=", cost, "unit cost", unit_cost)
freight = n * s * unit_cost
value = 100 * 300
print("  freight at unit cost", freight, "of cargo value", value, "=", freight * 10_000 // value, "bp")
full = {"staple": 400}
mf, bf = load(full)
print("  a full unit:", (mf, bf), "units", units_needed(mf, bf, 1200, 900), "freight share bp", freight * 10_000 // (400 * 300))
print("  haul units per execution: 1 unit * 40 distance / span 10 =", 40 // 10, "; round trip ticks", 2 * ticks(40, 30))
