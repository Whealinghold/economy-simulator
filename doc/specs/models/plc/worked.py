"""The arithmetic of the worked examples of spec-policy.md 14 (an invented kingdom,
spec-polity.md 12.1).  Integer arithmetic; toward zero unless 'half even' is said."""
MASK = 2**32 - 1
def weyl2(i): return (i * 0x6A09E667) & MASK
def bucket(i, P): return (weyl2(i) * P) >> 32
def rdiv_even(n, d):
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q
def trunc_div(n, d):
    q = abs(n) // abs(d)
    return q if (n >= 0) == (d > 0) else -q

print("== actors, ascending ID: coast_council 0, crown_council 1, inland_council 2")
for idx, P in ((0, 10), (1, 10), (2, 30)):
    b = bucket(idx, P)
    ticks = [t for t in range(0, 130) if (t + b) % P == 0]
    print(f"actor {idx} Every {P}: Weyl2 {weyl2(idx)} bucket {b}; decision ticks 0..129: {ticks[:6]} ... {ticks[-2:]}")

print("\n== A: schedule of crown_council (steps at 0, 120, 240; Every 10, bucket 4)")
steps = [(0, 10000), (120, 15000), (240, 10000)]
def sched(t):
    cur = None
    for at, v in steps:
        if at <= t: cur = v
    return cur
b = bucket(1, 10)
dec = [t for t in range(0, 260) if (t + b) % 10 == 0]
last = 10000
for t in dec:
    v = sched(t)
    if v is not None and v != last and t <= 250:
        print(f"tick {t}: schedule value {v} (current {last}) -> applied")
        last = v
print("first decision at or after 120:", [t for t in dec if t >= 120][0])

print("\n== B: rule of inland_council (Every 30, bucket 24) on unfed_share, lever stipend")
print("decision ticks:", [t for t in range(0, 130) if (t + bucket(2, 30)) % 30 == 0])
Target, Deadband, Span, Move, MaxStep, Min, Max = 500, 100, 1000, 2000, 1000, 0, 20000
H = 400 + 150 + 450
def step(v, unfed):
    m = rdiv_even(10000 * unfed, H)
    e = m - Target
    if abs(e) <= Deadband:
        return v, m, e, "in_band", 0
    delta = trunc_div(e * Move, Span)          # Raise: sign +1
    cut = ""
    if abs(delta) > MaxStep:
        delta = MaxStep if delta > 0 else -MaxStep
        cut = "stepped"
    nv = max(Min, min(Max, v + delta))
    return nv, m, e, cut or "applied", delta
v = 10000
for unfed in (90, 70, 52, 160, 30):
    nv, m, e, why, delta = step(v, unfed)
    print(f"unfed {unfed:3d} of {H}: reading {m:5d} bp, error {e:+5d}, delta {delta:+5d}, {why:8s} {v} -> {nv}")
    v = nv

print("\n== gauges")
mr = [(12000, 300), (8000, 100), (15000, 200)]
num = sum(a * w for a, w in mr); den = sum(w for _, w in mr)
print("mean_ratio:", num, "/", den, "=", num / den, "-> half even", rdiv_even(num, den))
pr = [("farm", 24, 1, 400), ("hills", 30, 5, 150), ("mill", 20, 2, 450)]
MaxAge = 3
use = [(p, w) for _, p, age, w in pr if age <= MaxAge]
n = sum(p * w for p, w in use); d = sum(w for _, w in use)
print("price with MaxAge 3:", n, "/", d, "=", n / d, "-> half even", rdiv_even(n, d))
print("unfed_share 900 bp check:", rdiv_even(10000 * 90, H))

print("\n== C: a border lever of coast_council (Against inland) in the kingdom of spec-polity.md 12.1")
chain = {"capital": ["crown"], "farm": ["inland", "crown"], "harbour": ["coast", "crown"],
         "hills": ["uplands", "inland", "crown"], "mill": ["inland", "crown"]}
edges = [("capital","harbour"),("capital","mill"),("farm","mill"),("harbour","capital"),
         ("harbour","mill"),("hills","mill"),("mill","capital"),("mill","farm"),
         ("mill","harbour"),("mill","hills")]
def crossing(a, b):
    ex = [j for j in chain[a] if j not in chain[b]]
    en = [j for j in reversed(chain[b]) if j not in chain[a]]
    return ex, en
def between(a, b):
    return [i for i, (f, t) in enumerate(edges) if a in crossing(f, t)[0] and b in crossing(f, t)[1]]
print("Between(coast, inland):", between("coast", "inland"), " Between(inland, coast):", between("inland", "coast"))
print("lever keys of Against inland held by coast:", sorted(between("coast", "inland") + between("inland", "coast")))
print("Between(coast, crown):", between("coast", "crown"), "(nested: none)")

print("\n== E: the checks of a returned list (current stipend 11000, Min 0, Max 20000, MaxStep 1000)")
cur = 11000
entries = [("lever 0", 25000), ("lever 0 again", 9000), ("lever 7", 1)]
v = max(0, min(20000, entries[0][1])); print("entry 1: clamp to Max ->", v, end="; ")
d = max(-1000, min(1000, v - cur)); print("step limit ->", cur + d, "(applied, clamped, stepped)")
print("entry 2: duplicate (the lever already had an entry in this list)")
print("entry 3: not_held (no lever 7)")
