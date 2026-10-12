"""Arithmetic of the worked examples of spec-settlements.md section 12. Integer arithmetic
only; the settings are invented examples and the numbers are placeholders (N-015, N-027).
Run with Docker (see README.md)."""
from band import effective, founded, target, Region

print("== Example A: a realm of five regions (an early-modern-like example setting)")
enter = [0, 300, 800, 2000]
names = ["hamlet", "village", "town", "city"]
band = 1000
print("classes", dict(zip(names, enter)), "band", band, "bp, dwell 30")
rows = {  # region: (ThresholdBP, floor class, households at construction)
    "centre": (10000, 0, 1400),
    "east": (10000, 0, 900),
    "north": (10000, 1, 150),
    "south": (10000, 0, 1100),
    "west": (6000, 0, 240),
}
for r, (thr, floor, h) in rows.items():
    E, L = effective(enter, thr, band)
    c = founded(E, floor, h)
    print(f"{r:7s} ThresholdBP {thr:5d} floor {names[floor]:7s} E={E[1:]} L={L[1:]} "
          f"households {h:5d} -> class {names[c]} (computed {names[founded(E, 0, h)]})")

print("-- the trace of centre: +3 households a tick from 1400, a plague at tick 205")
E, L = effective(enter, 10000, band)
for label, h_after in (("recovers to 1810 by tick 230", 1810), ("stays at 1790 at tick 230", 1790)):
    reg = Region(E, L, 0, 30, 1400)
    out = []
    for t in range(0, 241):
        h = 1400 + 3 * t if t <= 200 else (1790 if t < 230 else h_after)
        if t > 200 and t < 205:
            h = 2000 + 3 * (t - 200)
        was = reg.c
        held0 = reg.held
        reg.review(t, h)
        if reg.c != was:
            out.append(f"t={t} H={h} {names[was]}->{names[reg.c]} since={reg.since}")
        elif reg.held != held0 and reg.held - held0 == 1 and t in (205, 229):
            out.append(f"t={t} H={h} held (target {names[target(E, L, reg.c, 0, h)]}, "
                       f"ticks left {30 - (t - reg.since)})")
    print(f"  {label}: " + "; ".join(out) + f"; held ticks {reg.held}")

print("== Example B: an orbital colony (unit: a household is a crew of the colony)")
enter = [0, 20, 150]
names = ["outpost", "station", "hub"]
E, L = effective(enter, 10000, 1000)
print("classes", dict(zip(names, enter)), "E", E[1:], "L", L[1:], "(band 1000 bp)")
for h in (5, 19, 20, 149, 150, 136, 135):
    print(f"  households {h}: founded class {names[founded(E, 0, h)]}")
print("  a review at class hub: 134 ->", names[target(E, L, 2, 0, 134)],
      "; 135 ->", names[target(E, L, 2, 0, 135)], "(L = 135)")

print("== Example C: one class (a setting with no sizes)")
E, L = effective([0], 10000, 1000)
print("E", E, "L", L, "class for 0, 1, 10^9 households:",
      [founded(E, 0, h) for h in (0, 1, 10**9)], "; a review never changes it:",
      target(E, L, 0, 0, 0), target(E, L, 0, 0, 10**9))
