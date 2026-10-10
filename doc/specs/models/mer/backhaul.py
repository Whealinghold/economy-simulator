"""The backhaul double count of spec-merchants.md 10 (README Still open 5, spec-transport.md 20.2).

The arithmetic of `models/trn/worked.py` (17.4) in two readings of the carrier's recipe. Integer
arithmetic only; every number is a placeholder of an example setting.

  round trip:  Duration 4 (2 ticks each way over a distance of 40 at speed 30), inputs 12 fodder,
               wear 20 on a vehicle of durability 400 and price 6 000, labour 4 household-ticks at 1 000,
               output 4 haul units: they serve the loaded leg only.
  one way:     the same carriers, with Duration 2, 6 fodder, wear 10, labour 2, output 4: the pack
               states the recipe for one direction (the pack lever of spec-transport.md 8.4).
"""


def ceil_div(a, b):
    return -((-a) // b)


def unit_cost(duration, fodder, wear, labour_ticks, output):
    cost = fodder * 40 + ceil_div(wear * 6000, 400) + labour_ticks * 1000
    return cost, ceil_div(cost * 10000, output * 10000)


for label, args in (("round trip, one loaded leg", (4, 12, 20, 4, 4)),
                    ("one way, the pack states one direction", (2, 6, 10, 2, 4))):
    cost, uc = unit_cost(*args)
    print(f"{label:<44} cost of an execution {cost:5d}, unit cost {uc:5d}, freight of a shipment of 6 haul units {6 * uc:6d}")

print()
print("Both legs loaded, round-trip recipe: the carrier's execution serves 8 haul units (4 each way)")
cost, _ = unit_cost(4, 12, 20, 4, 4)
uc2 = ceil_div(cost, 8)
print(f"  unit cost {uc2}, freight of a shipment of 6 haul units {6 * uc2}")
print()
print("The smallest gap a unit that a merchant ships on (fee share 5 000, a full carrying unit of 400 staple at a price of 300):")
for label, fr in (("round trip, one loaded leg", 6 * 1195), ("round trip, both legs loaded", 6 * uc2), ("one way recipe", 6 * 598)):
    gap = ceil_div(fr * 10000, 5000 * 400)
    print(f"  {label:<30} freight {fr:5d}, smallest gap {gap:3d} a unit, {gap * 10000 // 300 / 100:.1f} % of the price")
