"""MIG worked examples: the integer arithmetic of spec-migration.md 15 (examples A and B).
Standard library only. Every name and number is a placeholder of an invented setting."""
M32 = 2 ** 32
def weyl(i): return (i * 2654435769) % M32
def cdiv(a, b): return -(-a // b)

def margin(agent, gain, width):
    """g_a = MoveGain + floor(u * MoveWidth), u = weyl(id) / 2^32 (spec 6.4)."""
    return gain + (weyl(agent) * width) // M32

def estimate(out_q, yield_bp, out_price, inputs, duration, upkeep, prices):
    """net value per execution of one recipe at the given prices, per working tick, as a ratio of the upkeep (spec 6.2).
    inputs: list of (quantity, commodity); upkeep: list of (quantity, commodity); prices: commodity -> price."""
    value = out_q * yield_bp * out_price // 10_000
    cost = sum(q * prices[c] for q, c in inputs)
    net = value - cost
    per_tick = net // duration
    u = sum(q * prices[c] for q, c in upkeep)
    return value, cost, net, per_tick, u, per_tick * 10_000 // u

print("# Example A: a river valley, a cultivator of `upper` (agent 1042)")
GAIN, WIDTH, H = 1000, 5000, 180
print("weyl(1042) =", weyl(1042), " u =", weyl(1042) / M32, " margin g_a =", margin(1042, GAIN, WIDTH), "bp")
D = {("upper", "middle"): 40, ("upper", "lower"): 70}          # the journey, REG's Distance(from, to)
DN = {("lower", "upper"): 120, ("middle", "upper"): 70}         # the news, Distance(source, reader)
speed, info = 20, 40
print("journey k:", {k: max(1, cdiv(v, speed)) for k, v in D.items()})
print("news lag to upper:", {k[0]: 1 + cdiv(v, info) for k, v in DN.items()})
# home: the local report (window means), the household's own observed wage
local = {"staple": 298, "fuel": 405, "seed": 80}
u_home = 2 * local["staple"] + 1 * local["fuel"]
Va_wage = 640
print("home upkeep cost", u_home, " observed wage", Va_wage, " V_a =", Va_wage * 10_000 // u_home)
Va = Va_wage * 10_000 // u_home
dests = {
    "middle": dict(yield_bp=10_000, prices={"staple": 321, "seed": 80}, upkeep=[(2, "staple")]),
    "lower": dict(yield_bp=12_000, prices={"staple": 366, "seed": 82}, upkeep=[(2, "staple")]),
}
ranking = []
for d, c in dests.items():
    k = max(1, cdiv(D[("upper", d)], speed))
    value, cost, net, per_tick, u, vhat = estimate(12, c["yield_bp"], c["prices"]["staple"], [(2, "seed")], 4, c["upkeep"], c["prices"])
    veff = vhat * (H - k) // H
    ga = margin(1042, GAIN, WIDTH)
    need = Va * (10_000 + ga) // 10_000
    print(f"{d}: k {k}  value {value}  cost {cost}  net {net}  per tick {per_tick}  upkeep {u}  V_hat {vhat}  V_eff {veff}  needed {need}  passes {veff * 10_000 >= Va * (10_000 + ga)}  viable {vhat >= 8000}")
    ranking.append((-veff, d, k))
ranking.sort()
print("ranking:", [(d, -v, k) for v, d, k in ranking])
print("provisioning (upper's upkeep is 2 staple and 1 fuel a tick, the account stays in upper):")
for _, d, k in ranking:
    print(f"  to {d}: k {k} needs {2 * k} staple and {k} fuel; the household holds 6 staple and 3 fuel -> {'ok' if 6 >= 2 * k and 3 >= k else 'unprovisioned'}")
print("caps: upper households 600, out cap ceil(600*50/10000) =", cdiv(600 * 50, 10_000), "; middle households 1000, on-road cap ceil(1000*100/10000) =", cdiv(1000 * 100, 10_000))
print("a departure at the end of tick t arrives in the first phase of tick t + 1 + k:", {d: f"t+{1 + k}" for _, d, k in ranking})
print("review partition example: 40 reviewed, 12 pushed = NoGain 3 + Unprovisioned 2 + OutCap 4 + Started 3 =", 3 + 2 + 4 + 3)
print()
print("# Example B: an orbital colony, the asymmetry of the well")
D = {("belt", "orbital"): 3000, ("belt", "surface"): 5000, ("orbital", "belt"): 3000, ("orbital", "surface"): 2000, ("surface", "belt"): 12400, ("surface", "orbital"): 9400}
speed, H = 3000, 6
print("journey k:", {f"{a}>{b}": max(1, cdiv(v, speed)) for (a, b), v in D.items()})
print("horizon discount (H - k)/H in bp:", {f"{a}>{b}": (H - max(1, cdiv(v, speed))) * 10_000 // H for (a, b), v in D.items()})
GAIN, WIDTH = 1500, 3000
for who, aid, home, dest, Va, vhat in (("technician of orbital", 2203, "orbital", "surface", 9000, 15000),
                                       ("technician of surface", 2204, "surface", "orbital", 9000, 21000),
                                       ("technician of surface", 2204, "surface", "belt", 9000, 30000)):
    k = max(1, cdiv(D[(home, dest)], speed))
    veff = vhat * (H - k) // H
    ga = margin(aid, GAIN, WIDTH)
    print(f"{who} ({aid}) {home}>{dest}: k {k}  V_hat {vhat}  V_eff {veff}  g_a {ga}  needed {Va * (10_000 + ga) // 10_000}  passes {veff * 10_000 >= Va * (10_000 + ga)}")
