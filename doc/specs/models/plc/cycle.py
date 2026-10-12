"""PLC-A1, A2, A4: the integer rule of spec-policy.md 8.3 on a plant with delay,
with and without its dampers.  Everything that the rule does is integer (basis
points, truncation toward zero, as the library does); the plant is a toy.

Rule:  e = reading - Target;  if |e| <= Deadband: hold
       delta = sign * e * Move / Span   (truncated toward zero)
       delta = clamp(delta, -MaxStep, +MaxStep);  v' = clamp(v + delta, Min, Max)
Plant: reading[k] = Base + Shock(k) + Slope * (v[k-d] - 10000) / 10000   (+ a small bounded
       disturbance from a fixed integer sequence); Shock is 0 before tick 20 and 600 after, so the
       equilibrium lever is 10000 + 600 * 10000 / 4000 = 11500
"""
def run(Move, Span, Deadband, MaxStep, Min, Max, d, slope, steps=600,
        Target=1000, Base=1000, noise=0, sign=+1, shock=600, shock_at=20):
    v = 10000
    hist = [10000] * (d + 1)
    changes = 0
    pinned = 0
    vs = []
    seed = 12345
    for k in range(steps):
        seed = (seed * 1103515245 + 12345) % (2**31)
        n = (seed % (2 * noise + 1)) - noise if noise else 0
        reading = Base + (shock if k >= shock_at else 0) + slope * (hist[0] - 10000) // 10000 + n
        e = reading - Target
        nv = v
        if abs(e) > Deadband:
            num = sign * e * Move
            delta = int(num / Span) if num >= 0 else -int(-num / Span)  # toward zero
            if MaxStep > 0:
                delta = max(-MaxStep, min(MaxStep, delta))
            nv = max(Min, min(Max, v + delta))
        if nv != v:
            changes += 1
        if nv in (Min, Max):
            pinned += 1
        v = nv
        hist.append(v)
        hist.pop(0)
        vs.append(v)
    tail = vs[steps // 2:]
    return changes, pinned, min(tail), max(tail)

hdr = f"{'case':44s} {'changes':>7s} {'pinned':>6s} {'tail min':>8s} {'tail max':>8s}"
print("Plant slope -4000 (reading falls 4000 per 10000 of lever), Target 1000, a +600 shock at tick 20: the equilibrium lever is 11500.")
print("Raising the lever lowers the reading, so the rule raises the lever when the reading is above")
print("target (sign +1, the balancing direction).  K = Move/Span * 0.4 is the loop gain per decision.")
print(hdr)
cases = []
# loop gain K = (Move/Span) * 0.4 per unit of reading per bp of lever; Span 1000 reading units
for name, kw in [
    ("d=0 K=0.4 no dampers",              dict(Move=1000, Span=1000, Deadband=0, MaxStep=0, d=0)),
    ("d=0 K=1.9 (below 2)",               dict(Move=4750, Span=1000, Deadband=0, MaxStep=0, d=0)),
    ("d=0 K=2.4 (above 2)",               dict(Move=6000, Span=1000, Deadband=0, MaxStep=0, d=0)),
    ("d=2 K=0.4 (bound 0.618)",           dict(Move=1000, Span=1000, Deadband=0, MaxStep=0, d=2)),
    ("d=2 K=0.8 (above 0.618), no dampers", dict(Move=2000, Span=1000, Deadband=0, MaxStep=0, d=2)),
    ("d=2 K=0.8 MaxStep 200",             dict(Move=2000, Span=1000, Deadband=0, MaxStep=200, d=2)),
    ("d=2 K=0.8 MaxStep 50",              dict(Move=2000, Span=1000, Deadband=0, MaxStep=50, d=2)),
    ("d=2 K=0.8 MaxStep 50 Deadband 40",  dict(Move=2000, Span=1000, Deadband=40, MaxStep=50, d=2)),
    ("d=2 K=0.8 bounds 8000..12000",      dict(Move=2000, Span=1000, Deadband=0, MaxStep=0, d=2, Min=8000, Max=12000)),
    ("d=2 K=0.4 Deadband 40 noise 30",    dict(Move=1000, Span=1000, Deadband=40, MaxStep=0, d=2, noise=30)),
    ("d=2 K=0.4 Deadband 0  noise 30",    dict(Move=1000, Span=1000, Deadband=0, MaxStep=0, d=2, noise=30)),
]:
    kw.setdefault("Min", 0)
    kw.setdefault("Max", 40000)
    ch, pin, lo, hi = run(slope=-4000, **kw)
    print(f"{name:44s} {ch:7d} {pin:6d} {lo:8d} {hi:8d}")

print()
print("PLC-A2, a stale reading: the reading never changes (a price carried forward at 1500,")
print("target 1000), the rule keeps stepping: windup to the bound, with and without MaxAge.")
def stale(steps, Deadband, MaxStep, Min, Max, guard):
    v = 10000
    n = 0
    for k in range(steps):
        reading, age = 1500, k + 1
        if guard is not None and age > guard:
            continue                              # undefined reading: hold
        e = reading - 1000
        if abs(e) > Deadband:
            delta = e * 500 // 1000
            if MaxStep:
                delta = max(-MaxStep, min(MaxStep, delta))
            nv = max(Min, min(Max, v + delta))
            if nv != v:
                n += 1
            v = nv
    return v, n
for name, kw in [("no guard, bounds 0..40000", dict(guard=None, Min=0, Max=40000)),
                 ("no guard, bounds 8000..12000", dict(guard=None, Min=8000, Max=12000)),
                 ("MaxAge 3 (reading undefined after 3 ticks)", dict(guard=3, Min=0, Max=40000))]:
    v, n = stale(100, 0, 100, **{k: x for k, x in kw.items()})
    print(f"{name:46s} final lever {v:6d}  changes {n}")
