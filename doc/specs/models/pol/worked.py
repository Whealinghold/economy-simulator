"""models/pol/worked.py: the arithmetic of the worked examples of spec-polity.md 12.

Example A (a kingdom, flagged): chains, crossings, Between, agent IDs, three combining
modes. Example B (two crowns, a free city and an outside node): crossings and IDs.
Standard library only. Docker only (see README.md).
"""
from itertools import product

# ---------- example A ----------
regions = sorted(["capital", "farm", "harbour", "hills", "mill"])
jur = {"coast": "crown", "crown": None, "inland": "crown", "uplands": "inland"}
jids = sorted(jur)
member = {"capital": "crown", "farm": "inland", "harbour": "coast", "hills": "uplands", "mill": "inland"}
seat = {"crown": "capital", "inland": "mill"}
edges = sorted([("capital", "harbour"), ("harbour", "capital"), ("capital", "mill"), ("mill", "capital"),
                ("mill", "farm"), ("farm", "mill"), ("mill", "hills"), ("hills", "mill"),
                ("harbour", "mill"), ("mill", "harbour")], key=lambda e: (regions.index(e[0]), regions.index(e[1])))


def chain(r):
    out, j = [], member.get(r)
    while j:
        out.append(j)
        j = jur[j]
    return out


def crossing(a, b):
    ca, cb = chain(a), chain(b)
    return [j for j in ca if j not in cb], [j for j in reversed(cb) if j not in ca]


def common(a, b):
    for j in chain(a):
        if j in chain(b):
            return j
    return None


def depth(j):
    d = 0
    while jur[j]:
        j = jur[j]
        d += 1
    return d


print("A: RegionIDs", {r: i for i, r in enumerate(regions)})
print("A: JurisdictionIDs", {j: i for i, j in enumerate(jids)})
for r in regions:
    print("A: chain(%s) = %s" % (r, chain(r)))
for a, b in [("hills", "harbour"), ("harbour", "hills"), ("farm", "mill"), ("farm", "capital"),
             ("hills", "farm"), ("capital", "mill")]:
    ex, en = crossing(a, b)
    c = common(a, b)
    print("A: crossing(%s, %s): exit=%s enter=%s common=%s border depth=%s" %
          (a, b, ex, en, c, depth(c) if c else -1))
print("A: edges by EdgeID", {i: e for i, e in enumerate(edges)})
for a, b in [("coast", "inland"), ("inland", "coast"), ("coast", "crown"), ("uplands", "coast")]:
    ids = [i for i, (f, t) in enumerate(edges)
           if a in crossing(f, t)[0] and b in crossing(f, t)[1]]
    print("A: between(%s, %s) = %s" % (a, b, ids))

n, m = len(regions), 0
gov = [j for j in jids if j in seat]
print("A: estate agents 0..%d, counterparties %d, treasuries %s, households from %d" %
      (n - 1, m, {j: n + m + k for k, j in enumerate(gov)}, n + m + len(gov)))

# the three combining modes along the chain of `hills` (6.1.2); numbers are placeholders
add_rows = {"crown": 200, "inland": 100, "uplands": 50}      # basis points, a levy
cap_rows = {"crown": 90, "inland": 80}                       # a cap, no row for uplands or coast
scale_rows = {"crown": 9000, "inland": 8000}                 # factors over 10 000
for r in regions:
    ch = chain(r)
    add = sum(add_rows.get(j, 0) for j in ch)
    inner = next((cap_rows[j] for j in ch if j in cap_rows), None)
    num, den = 1000, 1
    for j in reversed(ch):
        if j in scale_rows:
            num *= scale_rows[j]
            den *= 10000
    scaled = (2 * num + den) // (2 * den)  # rounded once, half up (the owner states the rule)
    print("A: %-8s add=%4d bp  innermost cap=%s  scaled(1000)=%d" % (r, add, inner, scaled))

# ---------- example B ----------
regionsB = sorted(["ashford", "brook", "city", "delta", "far"])
jurB = {"east": None, "west": None}
memberB = {"ashford": "east", "brook": "west", "delta": "west"}   # city and far: none


def chainB(r):
    j = memberB.get(r)
    return [j] if j else []


def crossB(a, b):
    ca, cb = chainB(a), chainB(b)
    return [j for j in ca if j not in cb], [j for j in reversed(cb) if j not in ca]


print("B: RegionIDs", {r: i for i, r in enumerate(regionsB)})
for a, b in [("delta", "far"), ("far", "delta"), ("city", "ashford"), ("ashford", "brook"), ("brook", "delta"),
             ("city", "far")]:
    ex, en = crossB(a, b)
    print("B: crossing(%s, %s): exit=%s enter=%s" % (a, b, ex, en))
nB, mB, gB = 5, 2, 2  # five regions, two counterparties (one node, one gate, both sides), two treasuries
print("B: estates 0..%d, counterparties %d..%d, treasuries east=%d west=%d, households from %d" %
      (nB - 1, nB, nB + mB - 1, nB + mB, nB + mB + 1, nB + mB + gB))
