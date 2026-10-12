"""models/crd/alloc.py: the integer rules of spec-credit.md 7, 8 and 9.

A model of the RULES of the spec, not of the library. Rules checked:

  interest   halfEven(x * rate_bp * term / (10_000 * ticks_per_year))          (7.1)
  allocate   pro rata by owed, largest remainder, ties by key then claim ID   (8.3)
  waterfall  the same rule with no shield, then write off the rest            (9.1)
  ledger     a mini ledger of accounts and claims under Lend, Settle,
             WriteOff and Reassign: the total of money and the pair identity
             of 3.6 hold exactly after every operation                        (3.2, 3.6)

Run (Docker only, never a local interpreter):
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/crd":/w:ro -w /w python:3.12-slim python alloc.py
"""
import random

M64 = (1 << 64) - 1


def half_even_div(n, d):
    """n / d rounded half to even, n >= 0, d > 0 (integers only)."""
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q


def interest(x, rate_bp, term, tpy=365):
    return half_even_div(x * rate_bp * term, 10_000 * tpy)


def key(t, ident):
    """The stateless key K(t, id) of spec-world.md 10.1.1 (SplitMix64 finaliser)."""
    z = (t * 0x9E3779B97F4A7C15 + ident) & M64
    z = (z + 0x9E3779B97F4A7C15) & M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


def allocate(pay, owed, t, ids):
    """Shares of `pay` among claims with `owed[i]`; sum(owed) >= pay >= 0."""
    total = sum(owed)
    assert 0 <= pay <= total
    if total == 0:
        return [0] * len(owed)
    shares = [pay * o // total for o in owed]
    frac = [(pay * o) % total for o in owed]
    left = pay - sum(shares)
    order = sorted(range(len(owed)), key=lambda i: (-frac[i], key(t, ids[i]), ids[i]))
    for i in order[:left]:
        shares[i] += 1
    return shares


def split(share, interest_owed):
    ip = min(share, interest_owed)
    return ip, share - ip


def check_allocation_properties(trials=20000, seed=7):
    rng = random.Random(seed)
    for n in range(trials):
        k = rng.randint(1, 6)
        owed = [rng.randint(1, 5000) for _ in range(k)]
        total = sum(owed)
        pay = rng.randint(0, total)
        ids = rng.sample(range(1, 1000), k)
        t = rng.randint(0, 1000)
        sh = allocate(pay, owed, t, ids)
        assert sum(sh) == pay, (pay, owed, sh)
        assert all(0 <= s <= o for s, o in zip(sh, owed)), (pay, owed, sh)
        # left < number of positive-fraction claims; a zero-fraction claim gets no extra
        for s, o in zip(sh, owed):
            exact, rem = divmod(pay * o, total)
            assert s == exact or (rem > 0 and s == exact + 1)
        # order invariance: permuting the claims moves no claim's share
        perm = list(range(k))
        rng.shuffle(perm)
        sh2 = allocate(pay, [owed[i] for i in perm], t, [ids[i] for i in perm])
        assert [sh2[perm.index(i)] for i in range(k)] == sh
    return trials


class Ledger:
    """accounts: dict id -> balance (non-negative); claims: dict id -> [cred, debt, P, I, Po, Io]."""

    def __init__(self):
        self.acc = {}
        self.claims = {}
        self.next = 1
        self.lent = self.repaid = self.int_paid = self.int_fixed = self.lost_p = self.lost_i = 0

    def total(self):
        return sum(self.acc.values())

    def lend(self, cred, debt, x, i):
        assert cred != debt and self.acc[cred] >= x >= 1
        self.acc[cred] -= x
        self.acc[debt] += x
        cid = self.next
        self.next += 1
        self.claims[cid] = [cred, debt, x, i, x, i]
        self.lent += x
        self.int_fixed += i
        return cid

    def settle(self, cid, ip, pp):
        c = self.claims[cid]
        assert 0 <= ip <= c[5] and 0 <= pp <= c[4] and self.acc[c[1]] >= ip + pp
        self.acc[c[1]] -= ip + pp
        self.acc[c[0]] += ip + pp
        c[5] -= ip
        c[4] -= pp
        self.int_paid += ip
        self.repaid += pp
        if c[4] == 0 and c[5] == 0:
            del self.claims[cid]

    def write_off(self, cid):
        c = self.claims.pop(cid)
        self.lost_p += c[4]
        self.lost_i += c[5]
        return c[4], c[5]

    def reassign(self, cid, new):
        self.claims[cid][0] = new

    def identity(self):
        po = sum(c[4] for c in self.claims.values())
        io = sum(c[5] for c in self.claims.values())
        assert po == self.lent - self.repaid - self.lost_p, "principal pair"
        assert io + self.int_paid + self.lost_i == self.int_fixed, "interest pair"
        for c in self.claims.values():
            assert 0 <= c[4] <= c[2] and 0 <= c[5] <= c[3] and (c[4] or c[5]) and c[0] != c[1]


def random_ledger_run(steps=6000, seed=11):
    rng = random.Random(seed)
    L = Ledger()
    for a in range(1, 13):
        L.acc[a] = rng.randint(0, 20_000)
    L.acc[100] = 0  # an estate account
    m0 = L.total()
    ops = {"lend": 0, "settle": 0, "write_off": 0, "reassign": 0}
    t = 0
    for _ in range(steps):
        t += 1
        r = rng.random()
        if r < 0.35:
            c, d = rng.sample(range(1, 13), 2)
            if L.acc[c] >= 1:
                x = rng.randint(1, L.acc[c])
                L.lend(c, d, x, interest(x, rng.randint(0, 2000), rng.randint(1, 120)))
                ops["lend"] += 1
        elif r < 0.75 and L.claims:
            cid = rng.choice(sorted(L.claims))
            c = L.claims[cid]
            cap = min(L.acc[c[1]], c[4] + c[5])
            if cap:
                pay = rng.randint(1, cap)
                ip, pp = split(pay, c[5])
                L.settle(cid, ip, pp)
                ops["settle"] += 1
        elif r < 0.85 and L.claims:
            L.write_off(rng.choice(sorted(L.claims)))
            ops["write_off"] += 1
        elif L.claims:
            cid = rng.choice(sorted(L.claims))
            if L.claims[cid][1] != 100:
                L.reassign(cid, 100)
                ops["reassign"] += 1
        assert L.total() == m0, "money conserved"
        L.identity()
    return ops


def waterfall(balance, claims, t):
    """claims: list of (id, principal_owed, interest_owed). Returns per-claim paid
    (interest, principal), lost (interest, principal) and the rest for the estate."""
    owed = [p + i for _, p, i in claims]
    pay = min(balance, sum(owed))
    sh = allocate(pay, owed, t, [c[0] for c in claims])
    out = []
    for (cid, p, i), s in zip(claims, sh):
        ip, pp = split(s, i)
        out.append((cid, ip, pp, i - ip, p - pp))
    return out, balance - pay


def worked():
    print("== interest (7.1, example A of spec section 20) ==")
    print("  5 700 at 600 bp for 65 ticks of 365:", interest(5700, 600, 65), "(exact 60.904)")
    print("  small: 3650 at 100 bp for 1 tick:", interest(3650, 100, 1), "(exact 0.1 -> 0)")
    print("  half: 1825 at 100 bp, 1 tick:", half_even_div(1825 * 100 * 1, 10_000 * 365), "(exact 0.05 -> 0)")
    print("  half: 18250 * 100 * 1 / 3650000 =", half_even_div(18250 * 100, 3_650_000), "(exact 0.5 -> 0, the even)")
    print("  half: 54750 * 100 / 3650000 =", half_even_div(54750 * 100, 3_650_000), "(exact 1.5 -> 2, the even)")
    print()
    print("== pro rata with the largest remainder (8.3, example B) ==")
    for pay in (1000, 1001):
        sh = allocate(pay, [600, 900], 10, [4, 9])
        print(f"  pay {pay} over owed [600, 900]:", sh, "sum", sum(sh))
    print("  three equal claims of 100, pay 100, ties by K(t, id):")
    for t in (1, 2, 3):
        print(f"    t={t}:", allocate(100, [100, 100, 100], t, [5, 6, 7]))
    print()
    print("== interest first inside a claim (8.3) ==")
    print("  share 70 of a claim owing interest 61 and principal 5 700 ->", split(70, 61))
    print()
    print("== example A of spec section 20.1: the loan, the pot at P1.7a (8.2), the allocation ==")
    x = 5700
    i = interest(x, 600, 65, 365)
    print("  request: need = 100 * 65 + 200 - 1000 =", 100 * 65 + 200 - 1000, "; interest on", x, "=", i, "; owed", x + i)
    for money in (7500, 3000, 80):
        pot = max(0, money - 100)
        pay = min(pot, x + i)
        ip, pp = split(pay, i)
        print(f"  money {money}: pot {pot}, pay {pay} (interest {ip}, principal {pp}), owed left {x + i - pay}")
    print()
    print("== example C of spec section 20.3 (the second setting): need and interest ==")
    print("  need = 50 * 33 + 120 - 300 =", 50 * 33 + 120 - 300, "; interest", interest(1470, 900, 33, 400), "(exact 10.915)")
    late = 1 if 80 - 50 < 120 else 0   # 10.2.1.2: the inputs cannot be bought at once
    h = 30 + 1 + late + 2
    print("  money 80: Late =", late, ", H =", h, ", need in full =", 50 * h + 120 - 80, "; sized by what tier 2 affords =", 50 * h + min(120, max(0, 80 - 50)) - 80)
    print()
    print("== the waterfall of a removal (9.1, example B second half of 20.2) ==")
    out, rest = waterfall(800, [(21, 450, 50), (22, 650, 50)], 33)
    for cid, ip, pp, li, lp in out:
        print(f"  claim {cid}: paid interest {ip}, principal {pp}; lost interest {li}, principal {lp}")
    print("  left for the estate:", rest)
    out, rest = waterfall(1500, [(21, 450, 50), (22, 650, 50)], 33)
    print("  with a balance of 1 500:", [(c, ip, pp, li, lp) for c, ip, pp, li, lp in out], "estate", rest)


if __name__ == "__main__":
    worked()
    print()
    print("allocation properties over random cases:", check_allocation_properties(), "cases, all hold")
    print("mini ledger, random operations:", random_ledger_run())
    print("money conserved and the pair identity held after every operation")
