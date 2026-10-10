"""Prototype of the graph queries of spec-regions.md (not of the library).

Checks, in pure integer Python:
 1. The definition of 4.3: dist = shortest path length over open edges;
    NextHop(s, t) = the smallest region index v among the out-neighbours of s with
    w(s, v) + dist(v, t) == dist(s, t); the path built by repeated NextHop is the
    lexicographically smallest of all shortest paths (compared as sequences of
    region indexes). Verified against brute force on random directed graphs with
    asymmetric lengths, ties and unreachable pairs.
 2. The tables of the three worked examples of section 12.
 3. The component rule of 4.5 (strongly connected components, labelled by the rank
    of the smallest member) against a reachability definition.
Run it in Docker (never with a local interpreter): see README.md.
"""
import heapq
import itertools
import random
import sys

INF = -1


def dijkstra_all(n, edges):
    """edges: dict (a, b) -> length >= 1. Returns dist[n][n] with INF for unreachable."""
    out = [[] for _ in range(n)]
    for (a, b), w in edges.items():
        out[a].append((b, w))
    dist = [[INF] * n for _ in range(n)]
    for s in range(n):
        d = {s: 0}
        h = [(0, s)]
        while h:
            du, u = heapq.heappop(h)
            if du > d.get(u, 1 << 62):
                continue
            for v, w in out[u]:
                nd = du + w
                if nd < d.get(v, 1 << 62):
                    d[v] = nd
                    heapq.heappush(h, (nd, v))
        for t, v in d.items():
            dist[s][t] = v
    return dist


def next_hop(n, edges, dist):
    nh = [[INF] * n for _ in range(n)]
    for s in range(n):
        for t in range(n):
            if s == t or dist[s][t] == INF:
                continue
            for v in range(n):  # ascending region index
                w = edges.get((s, v))
                if w is None or dist[v][t] == INF:
                    continue
                if w + dist[v][t] == dist[s][t]:
                    nh[s][t] = v
                    break
    return nh


def path(nh, s, t):
    p = [s]
    while s != t:
        s = nh[s][t]
        p.append(s)
    return p


def brute(n, edges, s, t):
    """Lexicographically smallest shortest simple path, by enumeration."""
    best = None
    out = {}
    for (a, b), w in edges.items():
        out.setdefault(a, []).append((b, w))

    def walk(u, seen, p, length):
        nonlocal best
        if u == t:
            key = (length, p[:])
            if best is None or key < best:
                best = key
            return
        for v, w in sorted(out.get(u, [])):
            if v not in seen:
                seen.add(v)
                p.append(v)
                walk(v, seen, p, length + w)
                p.pop()
                seen.discard(v)

    walk(s, {s}, [s], 0)
    return best


def scc(n, dist):
    comp = [-1] * n
    k = 0
    for r in range(n):
        if comp[r] != -1:
            continue
        for q in range(n):
            if comp[q] == -1 and dist[r][q] != INF and dist[q][r] != INF:
                comp[q] = k
        k += 1
    return comp


def random_check(trials, seed):
    rng = random.Random(seed)
    ties = unreach = 0
    for _ in range(trials):
        n = rng.randint(2, 7)
        edges = {}
        for a in range(n):
            for b in range(n):
                if a != b and rng.random() < 0.45:
                    edges[(a, b)] = rng.randint(1, 6)  # small range: many ties
        dist = dijkstra_all(n, edges)
        nh = next_hop(n, edges, dist)
        for s in range(n):
            for t in range(n):
                b = brute(n, edges, s, t) if s != t else (0, [s])
                if s == t:
                    assert dist[s][t] == 0
                    continue
                if b is None:
                    assert dist[s][t] == INF and nh[s][t] == INF
                    unreach += 1
                    continue
                assert dist[s][t] == b[0], (edges, s, t, dist[s][t], b)
                assert path(nh, s, t) == b[1], (edges, s, t, path(nh, s, t), b)
                # count pairs where a tie existed
                alts = 0
                for (a, v), w in edges.items():
                    if a == s and dist[v][t] != INF and w + dist[v][t] == dist[s][t]:
                        alts += 1
                if alts > 1:
                    ties += 1
        comp = scc(n, dist)
        # every pair in one component is mutually reachable and labels are by smallest member
        for r in range(n):
            for q in range(n):
                same = dist[r][q] != INF and dist[q][r] != INF
                assert (comp[r] == comp[q]) == same
        firsts = [min(i for i in range(n) if comp[i] == c) for c in sorted(set(comp))]
        assert firsts == sorted(firsts)
    return ties, unreach


def show(name, ids, edges):
    idx = {x: i for i, x in enumerate(sorted(ids))}
    names = sorted(ids)
    e = {(idx[a], idx[b]): w for (a, b), w in edges.items()}
    n = len(names)
    dist = dijkstra_all(n, e)
    nh = next_hop(n, e, dist)
    print("==", name, "== region order:", names)
    print("distance (row = from, column = to; - = unreachable)")
    print("%14s" % "", *["%12s" % c for c in names])
    for s in range(n):
        print("%14s" % names[s], *["%12s" % ("-" if dist[s][t] == INF else dist[s][t]) for t in range(n)])
    for s in range(n):
        for t in range(n):
            if s != t and dist[s][t] != INF:
                p = " > ".join(names[i] for i in path(nh, s, t))
                direct = e.get((s, t))
                note = ""
                if direct is not None and direct > dist[s][t]:
                    note = "   [edge %d longer than shortest %d]" % (direct, dist[s][t])
                print("  %s -> %s: %s (%d)%s" % (names[s], names[t], p, dist[s][t], note))
    comp = scc(n, dist)
    print("components:", {names[i]: comp[i] for i in range(n)})
    return names, dist, nh


def main():
    ties, unreach = random_check(3000, 20261010)
    print("random check passed: 3000 graphs, %d tied pairs, %d unreachable pairs" % (ties, unreach))
    # A: river valley (asymmetric: the upstream direction is longer)
    show("A river valley", ["upper", "middle", "lower"], {
        ("upper", "middle"): 40, ("middle", "upper"): 70,
        ("middle", "lower"): 30, ("lower", "middle"): 55,
        ("upper", "lower"): 90, ("lower", "upper"): 120,
    })
    # B: space colony (one-way: no edge from belt to surface; asymmetric gravity well)
    show("B orbital colony", ["orbital", "surface", "belt"], {
        ("surface", "orbital"): 9400, ("orbital", "surface"): 2000,
        ("orbital", "belt"): 3000, ("belt", "orbital"): 3000,
    })
    # C: a town and two hamlets (symmetric, with a tie)
    show("C town and hamlets", ["town", "hamlet_east", "hamlet_west"], {
        ("town", "hamlet_east"): 80, ("hamlet_east", "town"): 80,
        ("town", "hamlet_west"): 80, ("hamlet_west", "town"): 80,
        ("hamlet_east", "hamlet_west"): 160, ("hamlet_west", "hamlet_east"): 160,
    })
    # C': the same, edges listed in the opposite order: the result is the same table
    a = dijkstra_all(3, {(2, 0): 80, (0, 2): 80, (2, 1): 80, (1, 2): 80, (0, 1): 160, (1, 0): 160})
    b = dijkstra_all(3, {(0, 1): 160, (1, 0): 160, (0, 2): 80, (2, 0): 80, (1, 2): 80, (2, 1): 80})
    assert a == b
    # C'': rename `town` to `a_town`: it sorts first, so the tie of hamlet_east > hamlet_west
    # now resolves through the town (smallest index at the first step)
    show("C renamed: a_town", ["a_town", "hamlet_east", "hamlet_west"], {
        ("a_town", "hamlet_east"): 80, ("hamlet_east", "a_town"): 80,
        ("a_town", "hamlet_west"): 80, ("hamlet_west", "a_town"): 80,
        ("hamlet_east", "hamlet_west"): 160, ("hamlet_west", "hamlet_east"): 160,
    })
    # E: a proposal for the five-region reference world (section 13): four spokes from
    # `centre` and two pairs of neighbours
    sp = {}
    for r in ["east", "north", "south", "west"]:
        sp[("centre", r)] = 100
        sp[(r, "centre")] = 100
    for a_, b_ in [("north", "east"), ("south", "west")]:
        sp[(a_, b_)] = 130
        sp[(b_, a_)] = 130
    show("E five (proposal)", ["centre", "east", "north", "south", "west"], sp)
    # D: a closure (section 8): close middle>lower in A and recompute
    ids = ["upper", "middle", "lower"]
    idx = {x: i for i, x in enumerate(sorted(ids))}
    e = {(idx[a], idx[b]): w for (a, b), w in {
        ("upper", "middle"): 40, ("middle", "upper"): 70,
        ("lower", "middle"): 55, ("upper", "lower"): 90, ("lower", "upper"): 120}.items()}
    d2 = dijkstra_all(3, e)
    print("== A with middle>lower closed ==")
    names = sorted(ids)
    for s in range(3):
        print(" ", names[s], [("-" if x == INF else x) for x in d2[s]])
    # E: a factor on an edge (section 8): ceil(d * f / 10000)
    for d, f in [(30, 15000), (30, 10000), (7, 10001), (1, 1)]:
        print("effective(%d, %d bp) = %d" % (d, f, max(1, -(-d * f // 10000))))
    # B': orbital>surface absent: surface becomes a source (section 4.5)
    eb = {(1, 0): 3000, (0, 1): 3000, (2, 1): 9400}  # belt=0 orbital=1 surface=2
    db = dijkstra_all(3, eb)
    cb = scc(3, db)
    print("== B without orbital>surface ==", [[("-" if x == INF else x) for x in r] for r in db], "components", cb)
    assert cb == [0, 0, 1]
    # bound check of 4.2: the longest simple path at the bounds
    max_regions, max_dist = 1000, 10 ** 9
    print("longest shortest path <= %d; int64 max %d" % ((max_regions - 1) * max_dist, (1 << 63) - 1))
    return 0


if __name__ == "__main__":
    sys.exit(main())
