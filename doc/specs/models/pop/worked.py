"""POP worked examples of spec-population.md 12: the integer arithmetic, checked here. Not code of the library."""
def interp(knots, x):
    if x <= knots[0][0]: return knots[0][1]
    if x >= knots[-1][0]: return knots[-1][1]
    for (x0, y0), (x1, y1) in zip(knots, knots[1:]):
        if x0 <= x <= x1:
            num = (y1 - y0) * (x - x0); q = abs(num) // (x1 - x0)
            return y0 + (q if num >= 0 else -q)
def offset(agent_id, D): return ((agent_id * 2_654_435_769) % 2**32) * D // 2**32

print("== A. an agrarian realm: TicksPerYear 365, D =", 10_000*365)
BA = [(8_000, 250), (16_000, 250)]; DA = [(8_000, 450), (16_000, 50)]
for r in (5_000, 8_000, 12_000, 14_000, 20_000): print(f"  ratio {r:>6}: birth {interp(BA, r)}, death {interp(DA, r)}")
D = 10_000 * 365
H, d = 36, interp(DA, 14_000); num = H * d; c = offset(17, D)
print(f"  agent 17: weight 40, unfed 4, H = 36, d = {d}, per-tick numerator {num}, offset {c}")
t = 0
while True:
    t += 1; c += num
    if t <= 3 or c >= D:
        print(f"    tick {t}: carry {c}", "-> death 1, carry", c % D) if c >= D else print(f"    tick {t}: carry {c}")
    if c >= D: break
nb = 1000 * 250; cb = 0
for t in range(1, 17):
    cb += nb
    if cb >= D:
        print(f"  role of 1000 households, b = 250: numerator {nb}; birth at tick {t} with carry {cb % D}"); break
print("  funding: pool 30000, EntryPoolShare 5000 -> budget", 30_000*5_000//10_000, "; stake = 10000 * BirthStake 5000 / 10000 =", 10_000*5_000//10_000, "-> funded up to", (30_000*5_000//10_000)//(10_000*5_000//10_000))
rs = 12_000; print("  gain test r*·Δrate·T =", rs*400*30, " vs 10000·Δratio·TPY =", 10_000*8_000*365)
print("\n== B. a colony: TicksPerYear 640, D =", 10_000*640)
BB = [(6_000, 300), (18_000, 700)]; DB = [(6_000, 900), (18_000, 100)]
for r in (4_000, 6_000, 9_000, 12_000, 15_000, 18_000): print(f"  ratio {r:>6}: birth {interp(BB, r)}, death {interp(DB, r)}")
D = 10_000 * 640
b9, d9 = interp(BB, 9_000), interp(DB, 9_000)
print(f"  400 households at ratio 9000: b {b9}, d {d9}; numerators {400*b9}, {400*d9}; per tick {400*b9/D:.4f} births, {400*d9/D:.4f} deaths")
d3 = min(10_000, d9 * 30_000 // 10_000); print(f"  DeathScale 30000 (a disease): d = {d3}; numerator {400*d3}; per tick {400*d3/D:.4f}")
print("  gain test r*·Δrate·T =", 12_000*800*30, " vs 10000·Δratio·TPY =", 10_000*12_000*640)
print("  cap: HouseholdCap 600, households 598, births due 3 -> allowed", min(3, 600-598))
