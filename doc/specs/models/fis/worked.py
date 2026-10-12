"""The arithmetic of the worked examples of spec-fiscal.md 18. Integer only. Uses the
tree of spec-polity.md 12.1 (a kingdom of five regions). Not the library."""
from cut import key, cut

REG = ["capital", "farm", "harbour", "hills", "mill"]
# chains innermost first (spec-polity.md 12.1)
CHAIN = {"capital": ["crown"], "farm": ["inland", "crown"], "harbour": ["coast", "crown"],
         "hills": ["uplands", "inland", "crown"], "mill": ["inland", "crown"]}
SEAT = {"crown": "capital", "inland": "mill"}


def crossing(a, b):
    ca, cb = CHAIN[a], CHAIN[b]
    return [j for j in ca if j not in cb], [j for j in reversed(cb) if j not in ca]


print("== A. tick 0 of a small fiscal pack on the kingdom")
HH = {"capital": 4, "farm": 10, "harbour": 6, "hills": 5, "mill": 5}
bal, region, aid = {}, {}, {}
nxt = 7                                   # estates 0-4, treasuries 5 (crown) and 6 (inland)
for r in REG:
    for k in range(HH[r]):
        aid[(r, k)] = nxt
        region[nxt] = r
        bal[nxt] = 3000
        nxt += 1
bal[aid[("harbour", 0)]] = 40
bal[aid[("harbour", 1)]] = 25
tr = {"crown": 50000, "inland": 5000}
print("agent IDs:", {r: (aid[(r, 0)], aid[(r, HH[r] - 1)]) for r in REG})

# 1. poll: crown, every worker household, 60 a tick, no exemption: capped by the balance
paid, short, byreg = {}, {}, {}
for a in sorted(bal):
    due = 60
    pay = min(due, bal[a])
    bal[a] -= pay
    tr["crown"] += pay
    byreg[region[a]] = byreg.get(region[a], 0) + pay
    if pay < due:
        short[a] = due - pay
print("poll applied by region:", byreg, "short:", short, "crown treasury:", tr["crown"])

# 2. tithe: inland covers farm, hills, mill; 100 bp a tick of the balance above 2000, carry 0
tb = {}
for a in sorted(bal):
    if region[a] in ("farm", "hills", "mill"):
        base = max(0, bal[a] - 2000)
        num = base * 100 * 10000
        due, carry = divmod(num, 10**8)
        bal[a] -= due
        tr["inland"] += due
        tb[region[a]] = tb.get(region[a], 0) + due
        paid[a] = carry
print("tithe by region:", tb, "inland treasury:", tr["inland"], "carry of one payer:", set(paid.values()))

# 3. remittance: crown taxes inland's treasury 2500 bp of the balance above 1000
base = max(0, tr["inland"] - 1000)
rem = base * 2500 // 10000
tr["inland"] -= rem
tr["crown"] += rem
print("remit due:", rem, "inland treasury:", tr["inland"], "crown:", tr["crown"])

# 4. payout: inland, 500 a household to the workers of farm, hills and mill; cut pro rata
rec = [(a, 500) for a in sorted(bal) if region[a] in ("farm", "hills", "mill")]
out = cut(0, rec, tr["inland"])
extra = [a for a, _ in rec if out[a] == 206 + 1]
print("payout due", sum(d for _, d in rec), "available", tr["inland"], "floor", 500 * tr["inland"] // 10000,
      "leftover", tr["inland"] - sum(500 * tr["inland"] // 10000 for _ in rec))
print("applied:", sorted(set(out.values())), "recipients with the extra unit:", len(extra), "of", len(rec),
      "the five without it:", sorted(a for a, _ in rec if out[a] == 206))
print("keys of those five:", {a: key(0, a) for a in sorted(a for a, _ in rec if out[a] == 206)})
tr["inland"] -= sum(out.values())
print("inland treasury after the payout:", tr["inland"])

print("\n== B. a toll: crossings of the shipments of the examples")
for a, b in (("hills", "harbour"), ("harbour", "mill"), ("farm", "mill"), ("capital", "mill")):
    ex, en = crossing(a, b)
    print(f"{a} > {b}: exit {ex} enter {en}")
# hills > harbour is the path hills > mill > harbour (edges 5 and 8 of spec-polity.md 12.1)
n, tot = 3, 0
for a, b in (("hills", "mill"), ("mill", "harbour")):
    ex, en = crossing(a, b)
    due = 0
    if "inland" in ex:
        due += 4 * n * 10000 // 10000            # toll `ferry`: inland, on exit, 4 a carrying unit
    print(f"edge {a} > {b}: toll {due}")
    tot += due
print("hills > harbour, n =", n, "carrying units: total toll", tot, "paid by the shipper to the treasury of inland (region mill)")
print("harbour > mill, toll `gate_in` (inland, on enter, 2 a unit), n = 3:", 2 * 3 * 10000 // 10000 if "inland" in crossing("harbour", "mill")[1] else 0)
print("a scale of 5000 and 3 units of toll 3:", 3 * 3 * 5000 // 10000, "(the product first, one truncation)")

print("\n== C. a duty with a carry: 3 % of the value of a seller's sales (300 bp)")
D = 10**8
trades = [("A", 10 * 53), ("A", 2 * 410), ("B", 4 * 52)]
base = {}
for p, v in trades:
    base[p] = base.get(p, 0) + v
carry = {"A": 0, "B": 0}
for p in ("A", "B"):
    num = base[p] * 300 * 10000 + carry[p]
    due, carry[p] = divmod(num, D)
    print(f"tick 0, seller {p}: base {base[p]}, due {due}, carry {carry[p]}")
num = 100 * 300 * 10000 + carry["A"]
due, c = divmod(num, D)
print(f"tick 1, seller A: base 100, due {due}, carry {c} (exactly 3 + 0.5 = 3.5 units owed over the two ticks: {40 + due} paid of 40.5 + 3 = 43.5)")

print("\n== D. a tariff and an export duty at a gate (the league of spec-polity.md 12.2)")
imp = 5 * 200
print("import: counterparty sells 5 cloth at 200 =", imp, "; duty 1500 bp from the domestic buyer:", imp * 1500 // 10000)
exp = 20 * 55
print("export: domestic seller sells 20 grain at 55 =", exp, "; duty 1000 bp from the seller:", exp * 1000 // 10000)
print("the two payers are in `delta`, the treasury of `west` is in `brook`: a transfer of", imp * 1500 // 10000 + exp * 1000 // 10000, "from region delta to region brook")
