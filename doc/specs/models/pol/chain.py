"""models/pol/chain.py: the definitions of spec-polity.md 4.1 against a second
formulation, on random forests.

Not the library: a check that the definitions (chain, covers, common, crossing,
between) are what the spec says and have the stated properties. Standard library
only. Run with Docker (see README.md). Deterministic: every forest comes from a
seeded random.Random.
"""
import random

MAX_DEPTH = 8


def forest(rng, nj, nr):
    """parents[j] in {-1} + earlier ids, member[r] in {-1} + ids; depth <= MAX_DEPTH - 1 (MAX_DEPTH levels)."""
    parent, depth = [], []
    for j in range(nj):
        cands = [p for p in range(j) if depth[p] + 1 < MAX_DEPTH]
        if cands and rng.random() < 0.75:
            p = rng.choice(cands)
            parent.append(p)
            depth.append(depth[p] + 1)
        else:
            parent.append(-1)
            depth.append(0)
    member = [rng.choice([-1] + list(range(nj))) if rng.random() < 0.85 else -1 for _ in range(nr)]
    # relabel so that "sorted by ID" is a random permutation of the construction order
    perm = list(range(nj))
    rng.shuffle(perm)
    inv = {old: new for new, old in enumerate(perm)}
    parent2 = [None] * nj
    for old in range(nj):
        parent2[inv[old]] = -1 if parent[old] == -1 else inv[parent[old]]
    member2 = [-1 if m == -1 else inv[m] for m in member]
    return parent2, member2


# ---- formulation A: the definitions of 4.1.1, by chains and sets ----
def chain_a(parent, member, r):
    out, j = [], member[r]
    while j != -1:
        out.append(j)
        j = parent[j]
    return out


def common_a(parent, member, a, b):
    sb = set(chain_a(parent, member, b))
    for j in chain_a(parent, member, a):
        if j in sb:
            return j
    return -1


def crossing_a(parent, member, a, b):
    ca, cb = chain_a(parent, member, a), chain_a(parent, member, b)
    sa, sb = set(ca), set(cb)
    exit_ = [j for j in ca if j not in sb]
    enter = [j for j in reversed(cb) if j not in sa]
    return exit_, enter


# ---- formulation B: depth and a climb to the lowest common ancestor ----
def depth_of(parent, j):
    d = 0
    while parent[j] != -1:
        j = parent[j]
        d += 1
    return d


def crossing_b(parent, member, a, b):
    x, y = member[a], member[b]
    up_x, up_y = [], []
    dx = depth_of(parent, x) if x != -1 else -1
    dy = depth_of(parent, y) if y != -1 else -1
    # climb the deeper one; a region with no jurisdiction is above every root (depth -1)
    while dx > dy:
        up_x.append(x)
        x = parent[x]
        dx -= 1
    while dy > dx:
        up_y.append(y)
        y = parent[y]
        dy -= 1
    while x != y:  # both -1 only when they are in different trees or none
        if x == -1 and y == -1:
            break
        up_x.append(x)
        up_y.append(y)
        x, y = parent[x], parent[y]
    return up_x, list(reversed(up_y))


def regions_by_descent(parent, member, nj, j):
    kids = [[] for _ in range(nj)]
    for c, p in enumerate(parent):
        if p != -1:
            kids[p].append(c)
    seen, stack = set(), [j]
    while stack:
        k = stack.pop()
        seen.add(k)
        stack.extend(kids[k])
    return [r for r, m in enumerate(member) if m in seen]


def check(seed):
    rng = random.Random(seed)
    nj, nr = rng.randint(1, 30), rng.randint(1, 40)
    parent, member = forest(rng, nj, nr)
    n_pairs = n_border = 0
    for a in range(nr):
        for b in range(nr):
            ea, na = crossing_a(parent, member, a, b)
            eb, nb = crossing_b(parent, member, a, b)
            assert (ea, na) == (eb, nb), (seed, a, b, ea, na, eb, nb)
            n_pairs += 1
            if ea or na:
                n_border += 1
            # symmetry: leaving b's side of a->b is what entering is, reversed
            rb, rn = crossing_a(parent, member, b, a)
            assert rb == list(reversed(na)) and rn == list(reversed(ea)), (seed, a, b)
            # exit and enter are disjoint; a region with itself crosses nothing
            assert not set(ea) & set(na)
            if a == b:
                assert not ea and not na
            # common is a suffix of both chains
            c = common_a(parent, member, a, b)
            ca, cb = chain_a(parent, member, a), chain_a(parent, member, b)
            if c == -1:
                assert not set(ca) & set(cb)
                assert ea == ca and len(na) == len(cb)
            else:
                assert ca[ca.index(c):] == cb[cb.index(c):]
                assert ea == ca[:ca.index(c)]
    # covers and regions
    for j in range(nj):
        by_chain = [r for r in range(nr) if j in chain_a(parent, member, r)]
        assert by_chain == regions_by_descent(parent, member, nj, j), (seed, j)
    # between against the definition on random edges
    edges = sorted({(rng.randrange(nr), rng.randrange(nr)) for _ in range(rng.randint(0, 3 * nr))})
    edges = [(f, t) for f, t in edges if f != t]
    for a in range(nj):
        for b in range(nj):
            via_cross = [i for i, (f, t) in enumerate(edges)
                         if a in crossing_a(parent, member, f, t)[0] and b in crossing_a(parent, member, f, t)[1]]
            cov = lambda j, r: j in chain_a(parent, member, r)
            brute = [i for i, (f, t) in enumerate(edges)
                     if cov(a, f) and cov(b, t) and not cov(a, t) and not cov(b, f)]
            assert via_cross == brute, (seed, a, b)
            if a == b:
                assert not brute
    return n_pairs, n_border, nj, nr


def too_deep(parent):
    """spec 3.3.1: a jurisdiction with MAX_DEPTH or more ancestors is rejected."""
    for j in range(len(parent)):
        n, p = 0, parent[j]
        while p >= 0:
            n, p = n + 1, parent[p]
        if n >= MAX_DEPTH:
            return True
    return False


def depth_bound():
    # a chain of exactly MAX_DEPTH jurisdictions: the deepest region has MAX_DEPTH in its chain
    parent = [-1] + list(range(MAX_DEPTH - 1))
    member = [MAX_DEPTH - 1, 0, -1]
    assert chain_a(parent, member, 0) == list(range(MAX_DEPTH - 1, -1, -1))
    ex, en = crossing_a(parent, member, 0, 1)
    assert ex == list(range(MAX_DEPTH - 1, 0, -1)) and en == []
    assert not too_deep(parent)
    assert too_deep([-1] + list(range(MAX_DEPTH)))  # MAX_DEPTH + 1 levels
    return len(chain_a(parent, member, 0))


if __name__ == "__main__":
    tp = tb = 0
    for s in range(2000):
        p, b, _, _ = check(s)
        tp += p
        tb += b
    print("forests checked: 2000 (seeds 0..1999)")
    print("ordered region pairs compared: %d, of which a border: %d" % (tp, tb))
    print("deepest chain at the bound: %d" % depth_bound())
    print("all assertions held")
