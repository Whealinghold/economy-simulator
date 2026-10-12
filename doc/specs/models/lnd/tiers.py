"""Model of the tenure ration (spec-land.md 4): entitlement, own use, commons, let.

Integer arithmetic only, as the rules are. This is a model of the RULES of the spec, not of the
library. Run: python tiers.py [rule|worked|all]
"""
import random
import sys

M64 = (1 << 64) - 1
GOLD = 0x9E3779B97F4A7C15


def key(t, a):
    """The stateless tie-break key of spec-resources.md 5.3.3 (SplitMix64 finaliser)."""
    x = (t * GOLD + a) & M64
    z = (x + GOLD) & M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def ration(avail, reqs, t):
    """RES 5.3.2. reqs: list of (id, w, d). Returns {id: executions}."""
    u = sum(w * d for _, w, d in reqs)
    if u <= avail:
        return {i: w for i, w, d in reqs}
    g, fr, used = {}, {}, 0
    for i, w, d in reqs:
        g[i] = (w * avail) // u
        fr[i] = (w * avail) % u
        used += g[i] * d
    left = avail - used
    order = sorted([r for r in reqs if g[r[0]] < r[1]], key=lambda r: (-fr[r[0]], key(t, r[0]), r[0]))
    for i, w, d in order:
        if d <= left:
            g[i] += 1
            left -= d
    return g


def hamilton(total, parts, t):
    """Largest-remainder shares of `total` units by integer weights. parts: list of (id, weight).
    id -1 is the commons and loses every tie."""
    parts = [p for p in parts if p[1] > 0]
    if not parts or total == 0:
        return {i: 0 for i, _ in parts}
    w_all = sum(w for _, w in parts)
    base = {i: (total * w) // w_all for i, w in parts}
    rem = {i: (total * w) % w_all for i, w in parts}
    left = total - sum(base.values())
    order = sorted(parts, key=lambda p: (-rem[p[0]], (key(t, p[0]) if p[0] >= 0 else 1 << 64), p[0]))
    for i, _ in order[:left]:
        base[i] += 1
    return base


def tenure_round(a, shares, reqs, closed, rent, t, exempt=()):
    """One endowment, one tick. shares: {holder: bp}, sum <= 10000. reqs: list of dicts with
    id, principal, w, d. Returns grants by tier, the rent transfers and the counters."""
    s0 = 10000 - sum(shares.values())
    assert s0 >= 0
    ent = hamilton(a, sorted(shares.items()) + [(-1, s0)], t)
    own, unused = {}, {}
    for h in sorted(shares):
        rs = [(r['id'], r['w'], r['d']) for r in reqs if r['principal'] == h]
        g = ration(ent.get(h, 0), rs, t) if rs else {}
        unused[h] = ent.get(h, 0) - sum(g[i] * d for i, _, d in rs)
        own.update(g)
    d_of = {r['id']: r['d'] for r in reqs}
    res1 = {r['id']: r['w'] - own.get(r['id'], 0) for r in reqs}
    rs = [(i, res1[i], d_of[i]) for i in sorted(res1) if res1[i] > 0]
    com = ration(ent.get(-1, 0), rs, t) if rs else {}
    res2 = {i: res1[i] - com.get(i, 0) for i in res1}
    supply = 0 if closed else sum(unused.values())
    rs = [(i, res2[i], d_of[i]) for i in sorted(res2) if res2[i] > 0]
    let = ration(supply, rs, t) if rs and supply > 0 else {}
    g3 = sum(let.get(i, 0) * d_of[i] for i in let)
    sup = hamilton(g3, [(h, u) for h, u in sorted(unused.items()) if u > 0], t) if g3 else {}
    # attribution: tenants ascending id against lessors ascending id
    lessors = [[h, sup[h]] for h in sorted(sup)]
    pays = []  # (tenant, lessor, units)
    k = 0
    for i in sorted(let):
        need = let[i] * d_of[i]
        while need > 0:
            take = min(need, lessors[k][1])
            if take:
                pays.append((i, lessors[k][0], take))
                lessors[k][1] -= take
                need -= take
            if lessors[k][1] == 0:
                k += 1
    money = [(i, h, 0 if i in exempt else u * rent) for i, h, u in pays]
    unserved = sum((res2[i] - let.get(i, 0)) * d_of[i] for i in res2)
    grants = {r['id']: own.get(r['id'], 0) + com.get(r['id'], 0) + let.get(r['id'], 0) for r in reqs}
    return dict(ent=ent, own=own, com=com, let=let, unused=unused, grants=grants, pays=money,
                units_let=g3, unserved=unserved, idle=supply - g3)


def rule():
    rng = random.Random(7)
    for trial in range(20000):
        a = rng.randint(0, 300)
        nh = rng.randint(0, 6)
        shares = {}
        left = 10000
        for h in range(nh):
            s = rng.randint(1, max(1, left // 2)) if left > 1 else 0
            if s:
                shares[h] = s
                left -= s
        reqs = []
        for j in range(rng.randint(0, 12)):
            p = rng.choice(list(shares) + [100 + j]) if shares else 100 + j
            reqs.append(dict(id=100 + j, principal=p, w=rng.randint(1, 5), d=rng.randint(1, 3)))
        closed = rng.random() < 0.3
        rent = rng.randint(0, 50)
        t = rng.randint(0, 99)
        r = tenure_round(a, shares, reqs, closed, rent, t)
        drawn = sum(g * q['d'] for q in reqs for g in [r['grants'][q['id']]])
        assert drawn <= a, ('stock', trial)
        for q in reqs:
            assert 0 <= r['grants'][q['id']] <= q['w'], ('grant', trial)
        assert sum(r['ent'].values()) == a
        for h, u in r['unused'].items():
            assert u >= 0
        paid = sum(m for _, _, m in r['pays'])
        assert paid == rent * r['units_let'], ('rent', trial)
        if closed:
            assert r['units_let'] == 0
        # the result does not depend on how the requests are listed
        rr = tenure_round(a, shares, list(reversed(reqs)), closed, rent, t)
        assert rr['grants'] == r['grants'] and rr['pays'] == r['pays'], ('order', trial)
    print('rule: 20000 scenarios, conservation, bounds and rent identity hold')


def worked():
    shares = {h: 500 for h in range(10)}  # ten holders, 5 % each; commons 50 %
    reqs = []
    for h in range(10):
        reqs.append(dict(id=h, principal=h, w=3, d=1))
    for j in range(40):
        reqs.append(dict(id=100 + j, principal=1000 + j, w=2, d=1))
    r = tenure_round(100, shares, reqs, False, 40, 5)
    print('A=100, ten holders of 500 bp, commons 5 000 bp, rent 40 per unit, tick 5')
    print('entitlements (holders, commons):', [r['ent'][h] for h in range(10)], r['ent'][-1])
    print('own executions of the holders:', [r['own'][h] for h in range(10)], 'unused', [r['unused'][h] for h in range(10)])
    c = r['com']
    print('commons grants: 2 executions for', sorted(i for i in c if c[i] == 2).__len__(), 'landless, 1 for',
          sorted(i for i in c if c[i] == 1).__len__())
    print('let units', r['units_let'], 'unserved units', r['unserved'], 'idle let units', r['idle'])
    paid = {}
    for i, h, m in r['pays']:
        paid[h] = paid.get(h, 0) + m
    print('rent received by lessor:', [paid.get(h, 0) for h in range(10)], 'total', sum(paid.values()))
    print('units drawn:', sum(g * q['d'] for q in reqs for g in [r['grants'][q['id']]]))
    # stock 80: leftover entitlement
    shares = {h: 100 for h in range(50)}
    reqs = [dict(id=h, principal=h, w=1, d=1) for h in range(50)] + [dict(id=100 + j, principal=1000 + j, w=1, d=1) for j in range(30)]
    r = tenure_round(80, shares, reqs, False, 40, 5)
    e = [r['ent'][h] for h in range(50)]
    print('A=80, fifty holders of 100 bp: entitlements of 1 unit for', sum(e), 'of 50 holders; commons', r['ent'][-1],
          '; units drawn', sum(r['grants'].values()), '; let units', r['units_let'])


if __name__ == '__main__':
    which = sys.argv[1] if len(sys.argv) > 1 else 'all'
    if which in ('rule', 'all'):
        rule()
    if which in ('worked', 'all'):
        worked()
