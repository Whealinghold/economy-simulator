"""The integer arithmetic of the worked examples of spec-information.md section 12.

Every number is a labelled placeholder. The rules are integer; nothing here uses floating
point. Shortest distances are by Floyd-Warshall on the directed edges (the definition of
spec-regions.md 4.3.1: the least total length of a walk along open edges).

  Lag(r -> h) = 1 + ceil(Distance(r, h) / InfoSpeed)          spec 5.2
  vintage     = t - Lag                                        spec 5.4
  rumour      = halfEven(record * (10 000 + e) / 10 000), e in [-E, E], clamped to [1, MaxPrice]
"""
INF = None


def shortest(names, edges):
    n = len(names)
    ix = {s: i for i, s in enumerate(names)}
    d = [[INF] * n for _ in range(n)]
    for i in range(n):
        d[i][i] = 0
    for (a, b), w in edges.items():
        d[ix[a]][ix[b]] = w
    for k in range(n):
        for i in range(n):
            for j in range(n):
                if d[i][k] is not INF and d[k][j] is not INF:
                    if d[i][j] is INF or d[i][k] + d[k][j] < d[i][j]:
                        d[i][j] = d[i][k] + d[k][j]
    return d


def ceil_div(a, b):
    return (a + b - 1) // b


def half_even(num, den):
    q, r = divmod(num, den)
    if 2 * r > den or (2 * r == den and q % 2 == 1):
        q += 1
    return q


def distort(price, e, max_price=10**9):
    return min(max(half_even(price * (10_000 + e), 10_000), 1), max_price)


def lag_table(names, d, speed, title):
    print(title)
    print("source \\ reader: " + "  ".join(f"{h:>8}" for h in names))
    for i, r in enumerate(names):
        cells = []
        for j, h in enumerate(names):
            if i == j:
                cells.append(f"{'-':>8}")
            elif d[i][j] is INF:
                cells.append(f"{'none':>8}")
            else:
                cells.append(f"{1 + ceil_div(d[i][j], speed):>8}")
        print(f"{r:>16} " + "  ".join(cells))
    print()


# ---- Example A: a river valley (unit: one league), InfoSpeed 40 -------------------------
names_a = ["lower", "middle", "upper"]
edges_a = {("upper", "middle"): 40, ("middle", "upper"): 70, ("middle", "lower"): 30,
           ("lower", "middle"): 55, ("upper", "lower"): 90, ("lower", "upper"): 120}
d_a = shortest(names_a, edges_a)
print("Example A distances (from -> to):")
for i, r in enumerate(names_a):
    print(f"{r:>8}", d_a[i])
print()
lag_table(names_a, d_a, 40, "Example A lags, InfoSpeed 40 (rows: the source r, columns: the reader h)")

# the record of `staple` at `lower`, ticks 90..99: (price, price age, window mean) invented
rec = {90: (358, 0, 359), 91: (360, 0, 359), 92: (361, 0, 360), 93: (365, 0, 361),
       94: (365, 1, 362), 95: (372, 0, 363), 96: (370, 0, 365), 97: (368, 0, 366),
       98: (371, 0, 368), 99: (372, 0, 369)}
t = 100
for reader, j in (("middle", 1), ("upper", 2)):
    lag = 1 + ceil_div(d_a[0][j], 40)
    v = t - lag
    p, age, mean = rec[v]
    print(f"reader {reader} at tick {t} about staple at lower: Lag {lag}, vintage tick {v}, "
          f"record price {p}, price age {age}, window mean {mean}")
    for e in (-500, 0, 500):
        print(f"    e = {e:>5}: price {distort(p, e)}, mean {distort(mean, e)}")
print("an illustrative draw e = -230 for reader upper:", distort(370, -230), distort(365, -230))
print()

# a closure: middle > lower closed
edges_c = {k: v for k, v in edges_a.items() if k != ("middle", "lower")}
d_c = shortest(names_a, edges_c)
print("Example A with middle > lower closed: distances")
for i, r in enumerate(names_a):
    print(f"{r:>8}", d_c[i])
lag_table(names_a, d_c, 40, "Example A lags after the closure")
lag_before = 1 + ceil_div(d_a[2][0], 40)
lag_after = 1 + ceil_div(d_c[2][0], 40)
print(f"reader lower, source upper: Lag {lag_before} -> {lag_after}; at tick 100 the vintage moves "
      f"from {100 - lag_before} to {100 - lag_after}")
lag_before = 1 + ceil_div(d_a[1][0], 40)
lag_after = 1 + ceil_div(d_c[1][0], 40)
print(f"reader lower, source middle: Lag {lag_before} -> {lag_after}; vintage {100 - lag_before} -> {100 - lag_after}")
print()

# ---- Example B: an orbital colony (unit: delta-v in m/s), InfoSpeed 4000 --------------------
names_b = ["belt", "orbital", "surface"]
edges_b = {("surface", "orbital"): 9400, ("orbital", "surface"): 2000,
           ("orbital", "belt"): 3000, ("belt", "orbital"): 3000}
d_b = shortest(names_b, edges_b)
print("Example B distances (from -> to):")
for i, r in enumerate(names_b):
    print(f"{r:>8}", d_b[i])
print()
lag_table(names_b, d_b, 4000, "Example B lags, InfoSpeed 4000")
lag_table(names_b, d_b, 10**9, "Example B lags, InfoSpeed 10^9 (a channel that is practically instant): the floor of 2")
edges_b2 = {k: v for k, v in edges_b.items() if k != ("orbital", "surface")}
d_b2 = shortest(names_b, edges_b2)
lag_table(names_b, d_b2, 4000, "Example B without the edge orbital > surface: news cannot reach `surface`")

# ---- the first consumer's sizes -------------------------------------------------------------
keys, depth = 85, 60
entry_bytes = 8 + 8 + 4 + 4  # price, mean, price age, padding (an upper figure)
print(f"first consumer: {keys} source markets x depth {depth} = {keys * depth} entries, "
      f"about {keys * depth * entry_bytes // 1024} KiB")
informed, tradable, regions = 40, 8, 5
pairs = informed * tradable * (regions - 1)
print(f"{informed} informed agents x {tradable} tradable x {regions - 1} other regions = {pairs} candidate "
      f"rumours and as many draws a tick when the error is above 0")

# ---- the bound of the error ---------------------------------------------------------------------
print()
print("bounds of the error: price, E, low, high")
for price, e in ((370, 500), (1, 5000), (3, 5000), (1_000_000_000, 5000), (7, 500)):
    print(price, e, distort(price, -e), distort(price, e))
