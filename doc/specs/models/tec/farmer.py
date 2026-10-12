"""TEC model 3: what a productivity gain does to the income of a staple producer (spec-technology.md 13.5, TEC-A4).

An ANALYSIS of the target-income rule of spec-welfare-and-wages.md 5 against a demand curve Q = A p^-e (the form of
spec-stability.md 6.1), not a run of the library. The households are n identical producers; each works D days a year
(D at most DMAX, at least DMIN, the work floor MinWorkShare) at y T units a day (T the technology factor) and wants a yearly
income K. Floating point is used here because this is a closed-form analysis, never because a rule needs it. Every number
is a placeholder (N-015, N-027)."""
n, DMAX, DMIN = 1_000, 300.0, 100.0
D0, P0, Y = 150.0, 100.0, 1.0                       # the start: 150 days a year at T = 1, price 100, one unit a day
K = P0 * Y * D0                                       # the yearly target income, 15 000
Q0 = n * D0 * Y


def outcome(e, T):
    """Interior equilibrium (the target is met, D = K / (p y T)) if D is in [DMIN, DMAX]; the floor DMIN binds otherwise."""
    A = Q0 * P0 ** e
    if e != 1.0:
        p_int = (n * K / A) ** (1.0 / (1.0 - e))     # independent of T
    else:
        p_int = P0
    D_int = K / (p_int * Y * T)
    if D_int >= DMIN:
        D, p = min(D_int, DMAX), p_int
        kind = "interior"
    else:
        D = DMIN
        Q = n * D * Y * T
        p = (A / Q) ** (1.0 / e)
        kind = "floor"
    Q = n * D * Y * T
    income = p * D * Y * T
    return kind, D, p, Q, income / K


print("models/tec/farmer.py: n = 1 000 producers, K = 15 000 a year, D0 = 150 days at T = 1, floor DMIN = 100 days, p0 = 100")
print("The interior equilibrium of the rule is stable only for e > 1 (spec-stability.md 6.1); for e < 1 the market is pinned and the")
print("floor sets the outcome (spec-stability.md 6.1: the upper pin is at the price where all households work the floor). Both shown.\n")
print(f"{'e':>4} {'T':>5} {'case':>9} {'days':>6} {'price':>8} {'output':>9} {'income / K':>10}")
for e in (0.5, 1.0, 1.5):
    for T in (1.0, 1.1, 1.25, 1.5, 2.0):
        kind, D, p, Q, rel = outcome(e, T)
        print(f"{e:4.1f} {T:5.2f} {kind:>9} {D:6.1f} {p:8.2f} {Q:9.0f} {rel:10.3f}")
    print()
print("Forced to the floor for e = 0.5 (the pinned market): income / K = (T * DMIN / D0)^(1 - 1 / e) * (DMIN / D0)... evaluated:")
e = 0.5
A = Q0 * P0 ** e
for T in (1.0, 1.1, 1.25, 1.5, 2.0):
    Q = n * DMIN * Y * T
    p = (A / Q) ** (1.0 / e)
    print(f"  T {T:4.2f}: floor-pinned price {p:7.2f}, income / K {p * DMIN * Y * T / K:6.3f}, ratio to T = 1: "
          f"{(p * DMIN * Y * T) / ((A / (n * DMIN * Y)) ** (1.0 / e) * DMIN * Y):6.3f} = T^(1 - 1/e) = {T ** (1 - 1 / e):6.3f}")
