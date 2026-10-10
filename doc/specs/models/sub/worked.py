"""The arithmetic of the worked examples of spec-subsistence.md section 16 (integer rules only).

Example A: an agrarian setting (invented).  Example B: an orbital habitat (invented).
Nothing here is a result about the library; it checks the integer rules of the spec.
"""
from fractions import Fraction
import math


def cover_ticks(stock, upkeep_total):
    """Cover = floor(Stock / U) for one covered commodity (spec 5.2)."""
    return stock // upkeep_total


def afford_ticks(money, cost, hunger):
    """Afford = 0 if the household was unfed last tick, else floor(Money / Cost) (spec 5.2)."""
    return 0 if hunger > 0 else money // cost


def cov_bp(q, e_bp, duration, u):
    """Cov(r, c) = q * E / (Duration * u_c), basis points of one household's upkeep of c per labour tick (spec 4.1)."""
    return q * e_bp // (duration * u)


def ramp(w, cover, exit_ticks):
    """k = ceil(w * (Exit - Cover) / Exit), cover capped at Exit (spec 5.5)."""
    c = min(cover, exit_ticks)
    return -(-w * (exit_ticks - c) // exit_ticks)


def produced(k, q, e_bp, residue):
    raw = k * q * e_bp + residue
    return raw // 10_000, raw % 10_000


print("== Example A: an agrarian setting")
# one-activity-per-good chain: farm 8 grain a tick, cut 3 fuel a tick, upkeep grain 3 and fuel 1
# goods rows: 8 x_f >= 3 lam, 3 x_c >= 1 lam; labour: x_f + x_c <= 1  =>  lam = 1 / (3/8 + 1/3)
lam = Fraction(1) / (Fraction(3, 8) + Fraction(1, 3))
print("hand-solved surplus factor lambda* =", lam, "=", float(lam), " SurplusBP =", int(lam * 10_000))
q, e_bp, d, u = 5, 7_500, 1, 3
print("Cov(glean, grain) =", cov_bp(q, e_bp, d, u), "bp   (the market bound needs it below SurplusBP", int(lam * 10_000), ")")
print("units per execution at E 7500:", Fraction(q * e_bp, 10_000))

print("-- one cutter household (weight 1), grain price 100, upkeep grain 3, money 150, stock 7, residue start 4000")
price, money, stock, hunger, residue = 100, 150, 7, 0, 4_000
enter, exit_, retry = 2, 3, 10
mode, jobs = False, 0
U = 3
cost = U * price
for t in range(1, 10):
    cov = cover_ticks(stock, U)
    aff = afford_ticks(money, cost, hunger)
    sec = cov + aff
    note = ""
    if not mode and sec < enter:
        mode, jobs = True, 0
        note = "ENTER"
    elif mode and cov >= exit_:
        mode, jobs = False, 0
        note = "LEAVE"
    made = 0
    if mode:
        made, residue = produced(1, q, e_bp, residue)
        stock += made
        jobs += 1
    k_own = made * price if mode else 0
    # consumption at the end of the tick (the market gives nothing: the cutter's fuel does not sell)
    eaten = min(stock, U)
    stock -= eaten
    hunger = 0 if eaten == U else hunger + 1
    print(f"t{t}: cover {cov} afford {aff} secure {sec} {note:5s} mode={mode!s:5} made {made} K_own {k_own:4d} stock after eating {stock} hunger {hunger}")

print("-- a cohort of 40 with cover 1, exit 3: ramp k =", ramp(40, 1, 3), "; with cover 0:", ramp(40, 0, 3), "; with cover 2:", ramp(40, 2, 3))

print("\n== Example B: an orbital habitat")
lam_b = Fraction(18, 2 * 4)
print("synthesise: 18 ration per 2 ticks, upkeep 4  => lambda* =", lam_b, " SurplusBP =", int(lam_b * 10_000))
qb, eb, db, ub = 15, 8_000, 3, 4
print("Cov(tend_tray, ration) =", cov_bp(qb, eb, db, ub), "bp;  units per execution:", Fraction(qb * eb, 10_000))
w = 40
U = w * ub
stock, money, price = 200, 300, 12
cost = U * price
cov = cover_ticks(stock, U)
aff = afford_ticks(money, cost, 0)
print(f"cohort of {w}: U = {U}, stock {stock} -> cover {cov}; cost per tick {cost}, money {money} -> afford {aff}; secure {cov + aff}")
k = ramp(w, cov, exit_)
print("enters the mode (secure < 2); ramp k =", k, "; draws of tray_volume (2 each) =", 2 * k)
for stock_tray in (54, 40):
    print(f"   commodity stock of tray_volume {stock_tray}: grant to the cohort alone = {min(k, stock_tray // 2)} executions")
made, res = produced(k, qb, eb, 0)
print("   output at completion after 3 ticks: ", made, "ration (k * q * E / 10000 =", k * qb * eb // 10_000, ")")
unfed_before_output = max(0, db - 1 - cov)
print("   unfed ticks before the first output if cover were", cov, ":", unfed_before_output, "; worst case (cover 0):", db - 1, "; Duration - 1 =", db - 1, "< ChangePrivation 3")
print("   K_own at the price of the completion tick (12):", made * price)
