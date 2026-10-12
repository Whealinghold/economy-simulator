"""models/org/clamp.py: checks of spec-institutions.md section 6 (a legal price is a clamp).

The default clearing of spec-clearing.md 4 and 5 (sort, cross, midpoint rounded half to
even) is run over random books with and without a legal range, to check:

  P1  clamping the bids that exceed a cap puts every trade price at or below the cap
      (clamping the asks below a floor puts it at or above the floor), whatever the asks;
  P2  clamping trades at least as many units as rejecting the over-cap bids, and exactly
      the units an independent count gives (the k-th highest clamped bid against the
      k-th lowest ask);
  P3  the trade price is still between the two limits of every trade, so the checks of
      spec-clearing.md 8.2 hold for the clamped limits;
  P4  a cap and a floor together hold both bounds;
  P5  the range in minor units is min(MaxPrice, max(1, ref * bp / 10 000)): it is in
      [1, MaxPrice], a floor never exceeds its cap, and an admitted limit times a quantity
      fits int64 (spec-institutions.md 6.2.1).

Then the arithmetic of the worked example B of section 12.2. Standard library only.
Run in Docker, never on the host (see README).
"""
import random


def half_even_mid(a, b):
    """midpoint of integers a <= b, half to even (spec-clearing.md 5.2)."""
    d = b - a
    q = a + d // 2
    if d % 2 == 1:
        # exact midpoint ends in one half: the even neighbour
        return q if q % 2 == 0 else q + 1
    return q


def clear(bids, asks, rng):
    """bids, asks: lists of (limit, qty). Returns trades (price, qty). Ties: random key."""
    B = [(-p, rng.random(), i, p, q) for i, (p, q) in enumerate(bids)]
    A = [(p, rng.random(), i, p, q) for i, (p, q) in enumerate(asks)]
    B.sort()
    A.sort()
    bq = [b[4] for b in B]
    aq = [a[4] for a in A]
    trades = []
    i = j = 0
    while i < len(B) and j < len(A):
        bp, ap = B[i][3], A[j][3]
        if bp < ap:
            break
        q = min(bq[i], aq[j])
        trades.append((half_even_mid(ap, bp), q, ap, bp))
        bq[i] -= q
        aq[j] -= q
        if bq[i] == 0:
            i += 1
        if aq[j] == 0:
            j += 1
    return trades


def units_independent(bids, asks):
    """the number of units that can trade: expand to unit lists and compare the k-th."""
    bu = sorted((p for p, q in bids for _ in range(q)), reverse=True)
    au = sorted(p for p, q in asks for _ in range(q))
    k = 0
    while k < len(bu) and k < len(au) and bu[k] >= au[k]:
        k += 1
    return k


def legal(bids, asks, cap, floor):
    cb = [(min(p, cap) if cap else p, q) for p, q in bids]
    ca = [(max(p, floor) if floor else p, q) for p, q in asks]
    return cb, ca


def run(n_books, seed):
    rng = random.Random(seed)
    worst_over = 0
    stats = dict(books=0, trades=0, clamp_units=0, reject_units=0, free_units=0)
    for _ in range(n_books):
        ref = rng.randint(50, 2000)
        nb, na = rng.randint(1, 12), rng.randint(1, 12)
        bids = [(rng.randint(1, 3 * ref), rng.randint(1, 5)) for _ in range(nb)]
        asks = [(rng.randint(1, 3 * ref), rng.randint(1, 5)) for _ in range(na)]
        cap = rng.randint(max(1, ref // 2), 2 * ref) if rng.random() < 0.8 else 0
        floor = rng.randint(1, 2 * ref) if rng.random() < 0.3 else 0
        if cap and floor and floor > cap:
            floor = cap
        cb, ca = legal(bids, asks, cap, floor)
        tr = clear(cb, ca, random.Random(rng.random()))
        # P1 and P4
        for price, q, ap, bp in tr:
            if cap and price > cap:
                worst_over += 1
            if floor and price < floor:
                worst_over += 1
            assert ap <= price <= bp, "P3: price outside the limits"
        vol = sum(t[1] for t in tr)
        # P2: exact units against an independent count
        assert vol == units_independent(cb, ca), "P2: volume differs from the k-th rule"
        # rejecting the over-cap bids and the below-floor asks instead of clamping
        rb = [(p, q) for p, q in bids if not cap or p <= cap]
        ra = [(p, q) for p, q in asks if not floor or p >= floor]
        rvol = sum(t[1] for t in clear(rb, ra, random.Random(1)))
        assert vol >= rvol, "P2: clamping traded fewer units than rejecting"
        fvol = sum(t[1] for t in clear(bids, asks, random.Random(1)))
        stats["books"] += 1
        stats["trades"] += len(tr)
        stats["clamp_units"] += vol
        stats["reject_units"] += rvol
        stats["free_units"] += fvol
    assert worst_over == 0, "P1: a trade price broke the legal range"
    return stats


MAX_PRICE, MAX_QTY, MAX_BP = 10**9, 10**9, 10**6


def legal_range(ref, cap_bp, floor_bp):
    cap = min(MAX_PRICE, max(1, ref * cap_bp // 10000)) if cap_bp > 0 else None
    floor = min(MAX_PRICE, max(1, ref * floor_bp // 10000)) if floor_bp > 0 else None
    return cap, floor


def check_p5(n, seed):
    rng = random.Random(seed)
    clipped = 0
    for _ in range(n):
        ref = rng.choice([rng.randint(1, MAX_PRICE), MAX_PRICE, 1, rng.randint(1, 1000)])
        cb = rng.choice([rng.randint(1, MAX_BP), MAX_BP, 1, rng.randint(1, 20000)])
        fb = rng.randint(1, cb)
        cap, floor = legal_range(ref, cb, fb)
        assert 1 <= floor <= cap <= MAX_PRICE, "P5: range out of bound or inverted"
        assert cap * MAX_QTY < 2**63, "P5: limit times quantity overflows int64"
        assert ref * cb < 2**63, "P5: the product ref * cap_bp overflows int64"
        if ref * cb // 10000 > MAX_PRICE:
            clipped += 1
    return clipped


def example_b(cap_bp=12000):
    """Example B of 12.2: a cap of cap_bp basis points of a reference price of 1 000."""
    ref = 1000
    cap = max(1, ref * cap_bp // 10000)
    bids = [(1500, 3), (1400, 2), (1250, 4), (1200, 3), (1100, 5), (900, 2)]
    asks = [(800, 4), (1000, 4), (1150, 3), (1300, 5), (1500, 4)]
    cb, ca = legal(bids, asks, cap, 0)
    rng = random.Random(7)
    tr = clear(cb, ca, rng)
    tr_free = clear(bids, asks, random.Random(7))
    demand_at_cap = sum(q for p, q in bids if p >= cap)
    supply_at_cap = sum(q for p, q in asks if p <= cap)
    print("example B: reference", ref, "cap", cap)
    print("  clamped bids:", [(p, q) for (p, q), (p0, _) in zip(cb, bids) if p != p0])
    print("  trades under the cap (price, units):", [(t[0], t[1]) for t in tr])
    print("  units under the cap:", sum(t[1] for t in tr),
          "volume-weighted price:", sum(t[0] * t[1] for t in tr) // max(1, sum(t[1] for t in tr)))
    print("  trades without the cap:", [(t[0], t[1]) for t in tr_free])
    print("  units without the cap:", sum(t[1] for t in tr_free))
    print("  units wanted at or above the cap:", demand_at_cap,
          "units offered at or below the cap:", supply_at_cap)
    unmatched_b = sum(q for p, q in cb) - sum(t[1] for t in tr)
    unmatched_a = sum(q for p, q in ca) - sum(t[1] for t in tr)
    print("  unmatched bid units:", unmatched_b, "unmatched ask units:", unmatched_a)


if __name__ == "__main__":
    s = run(20000, 12345)
    print("20 000 random books, seed 12345: all of P1 to P4 held")
    print("  trades", s["trades"],
          "units with clamp", s["clamp_units"],
          "units with reject", s["reject_units"],
          "units with no law", s["free_units"])
    clipped = check_p5(50000, 777)
    print("P5 held over 50 000 random (ref, cap_bp, floor_bp); %d of them had a range above MaxPrice and were bounded" % clipped)
    example_b(12000)
    example_b(9000)
