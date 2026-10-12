"""TEC model 4: the arithmetic of the worked examples of spec-technology.md 16 and the bounds of 9.5.

Integer rules from law.py (imported); floating point only in printouts. Placeholders (N-015, N-027)."""
import sys
sys.path.insert(0, "/m/tec")
from law import lg12, gain_log, hit, keyed, finalise, BP, TPY, weight, diffuse_step   # noqa: E402

print("== Example A: three regions, one field `tillage` (an example), Base 100 000, DoublingBP 200, CapBP 3 000, CatchUpBP 2 000, Reach 100")
base, d, cap, catch, reach = 100_000, 200, 3_000, 2_000, 100
X = [0, 0, 0]
A = [0, 0, 0]
carry = [0, 0, 0]
exec_per_tick = [900, 400, 150]                      # plain, hills, coast: households that complete a job each tick (an example)
dist = [[None, 20, 60], [20, None, 50], [60, 50, None]]
names = ["plain", "hills", "coast"]
print(f"   executions a tick: {exec_per_tick}; distances plain-hills 20, plain-coast 60, hills-coast 50; w(20) 8 000, w(50) 5 000, w(60) 4 000")
for t in range(0, 5 * TPY):
    for i in range(3):
        X[i] += exec_per_tick[i]
    G = [min(cap, gain_log(X[i], base, d) + A[i]) for i in range(3)]
    add = diffuse_step(G, dist, catch, reach, carry)
    for i in range(3):
        A[i] = min(cap, A[i] + add[i])
    if t + 1 in (1, TPY, 2 * TPY, 5 * TPY):
        G2 = [min(cap, gain_log(X[i], base, d) + A[i]) for i in range(3)]
        print(f"   after tick {t + 1:5d}: X {X}, doing g {[gain_log(X[i], base, d) for i in range(3)]}, A {A}, gain G {G2}")
print("   (cell `hills` practises 400 a tick and `coast` 150: their own doing is lower; the catch-up from `plain` adds to A)")

print("\n== One tick of P8.3c on the same cells at tick 364 (the end of year 1), by hand")
X = [900 * 364, 400 * 364, 150 * 364]
print(f"   X before the step {X}; executions this tick [900, 400, 150]")
Xn = [x + e for x, e in zip(X, [900, 400, 150])]
g0 = [gain_log(x, base, d) for x in X]
g1 = [gain_log(x, base, d) for x in Xn]
print(f"   lg12: {[lg12(x + base) for x in Xn]}, Base lg12 {lg12(base)}; doing g before {g0}, after {g1}; per cell Doing {[b - a for a, b in zip(g0, g1)]}")

print("\n== Bounds (9.5): every product of the module fits, with the placeholders")
MaxGainBP, MaxDoublingBP, MaxCatchUpBP, MaxTicksPerYear, MaxYieldBP, MaxWeight, MaxQuantity = 100_000, 10_000, 10_000, 100_000, 100_000, 10**6, 10**9
print(f"   lg12 difference at most 62 * 4096 = {62 * 4096}; times MaxDoublingBP {MaxDoublingBP}: {62 * 4096 * MaxDoublingBP} < 2^63")
print(f"   catch-up numerator: gap {MaxGainBP} * CatchUpBP {MaxCatchUpBP} + carry < 10 000 * {MaxTicksPerYear}: "
      f"{MaxGainBP * MaxCatchUpBP + BP * MaxTicksPerYear} < 2^63")
print(f"   discovery: K (2^64) * (10 000 * {MaxTicksPerYear}) is a 128-bit product whose high word is < {BP * MaxTicksPerYear}")
MaxTechBP = BP + MaxGainBP
print(f"   the yield chain of PRO 6.3 with tech: {MaxYieldBP} (site) * {MaxYieldBP} (shock) * {MaxTechBP} (tech) * {MaxYieldBP} (trainee or hired: MaxYieldBP) "
      f"* abundance 100 000 = {MaxYieldBP * MaxYieldBP * MaxTechBP * MaxYieldBP * 100_000:.3e}: the validator's check of PRO 10.3 uses the checked 128-bit helper")
print(f"   one execution's output at the bounds: {MaxWeight} * {MaxQuantity} * {MaxTechBP} / 10 000 would be {MaxWeight * MaxQuantity * MaxTechBP // BP:.3e}: "
      f"rejected by PRO 10.3 (the quantity bound), which is the point of the check")
print(f"   a contact weight is computed where it is used: (Reach - d) * 10 000 with Reach at most MaxDistance 10^9: {10**9 * BP:.3e} < 2^63 (no table)")
