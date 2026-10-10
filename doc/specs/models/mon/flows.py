"""Integer model of the money flows of spec-money-supply.md (MON).

A MODEL of the rules, not of the library. Python integers are unbounded, so the
128-bit intermediates of the spec are not exercised; everything else (the carry,
the rate arithmetic, the order of operations, conservation) follows the spec.
All numbers are placeholders. Standard library only; run it in Docker.
"""

SCALE = 10_000  # basis points


def grant_step(carries, weights, amount, scale, den):
    """One tick of a rate grant (spec 6.1). Returns the payments, updates carries."""
    pay = []
    for i, w in enumerate(weights):
        num = w * amount * scale + carries[i]
        q, carries[i] = divmod(num, den * SCALE)
        pay.append(q)
    return pay


def levy_rate_step(bal, weights, rate_bp, exempt, carries, den):
    """One tick of a rate levy on the balance above the exemption (spec 6.3).
    Returns the amounts collected; a shortfall cannot occur for a rate below 100%
    but the cap by the available balance is applied as in the spec."""
    out = []
    for i, w in enumerate(weights):
        base = max(0, bal[i] - w * exempt)
        num = base * rate_bp + carries[i]
        q, carries[i] = divmod(num, SCALE * den)
        q = min(q, base)
        out.append(q)
    return out


def run_steady_state(ticks=6000, ticks_per_year=365, rate_bp_year=5_000,
                     stipend=300, n=50, exempt=0):
    """Hoarders: recipients never spend. The balance converges to
    exempt + stipend / rho, rho = rate_bp_year / (10_000 * ticks_per_year)."""
    weights = [1] * n
    bal = [0] * n
    gc = [i for i in range(n)]            # initial carry = account id (spec 6.2)
    lc = [0] * n
    boundary_grant = 0
    boundary_levy = 0
    trace = []
    for t in range(ticks):
        pay = grant_step(gc, weights, stipend, SCALE, 1)
        for i, p in enumerate(pay):
            bal[i] += p
            boundary_grant -= p
        got = levy_rate_step(bal, weights, rate_bp_year, exempt, lc, ticks_per_year)
        for i, g in enumerate(got):
            bal[i] -= g
            boundary_levy += g
        # conservation: accounts + both boundaries == 0
        assert sum(bal) + boundary_grant + boundary_levy == 0
        if t in (0, 99, 364, 729, 1459, 2919, 5999):
            trace.append((t + 1, sum(bal) // n))
    rho_num, rho_den = rate_bp_year, SCALE * ticks_per_year
    m_star = exempt + stipend * rho_den // rho_num
    return trace, m_star


def run_carry_example(amount=7, ticks_per_year=365):
    """A grant of 7 per year, spread by the carry: first payment tick and the
    total after a year, for two initial carries (spec 6.1, worked example)."""
    res = []
    for acct in (0, 100):
        c = [acct * SCALE]                # initial carry = (id mod den) in units of 1/(den*SCALE)
        # carry unit is 1/(den*SCALE); id offset in whole 1/den: id * SCALE
        total, first = 0, None
        for t in range(1, ticks_per_year + 1):
            p = grant_step(c, [1], amount, SCALE, ticks_per_year)[0]
            if p and first is None:
                first = t
            total += p
        res.append((acct, first, total))
    return res


def run_weight_check():
    """A cohort of weight w gets exactly w times a household's total over a year,
    up to the carry (< 1 unit): spec 6.1."""
    out = []
    for w in (1, 3, 40):
        c = [0]
        tot = 0
        for _ in range(365):
            tot += grant_step(c, [w], 7, SCALE, 365)[0]
        out.append((w, tot, w * 7))
    return out


def run_pair(n_r=50, n_p=450, ticks=400):
    """A stipend to class R paid for by a fixed levy on class P (the balanced
    pair of spec 9.2). R spends its stipend on goods sold by P (an internal
    trade, here a plain transfer split evenly, the remainder to the first
    accounts in ID order). The levy is 33 a tick per P household: 450 * 33 =
    14 850 against 15 000 granted. Reports the boundary balance of each reason:
    the pair is not exactly balanced, so money drifts by 150 a tick, and the
    boundary balances are the measure of it (spec 9.2, 8.3)."""
    wr, wp = [1] * n_r, [1] * n_p
    br, bp = [2_700] * n_r, [2_700] * n_p
    gc = [i for i in range(n_r)]
    b_initial = -(sum(br) + sum(bp))      # the endowment, created by `initial`
    b_grant = b_levy = 0
    for t in range(ticks):
        pay = grant_step(gc, wr, 300, SCALE, 1)
        for i, p in enumerate(pay):
            br[i] += p
            b_grant -= p
        spend = 0
        for i in range(n_r):              # R spends the stipend it just got
            take = min(300, br[i])
            br[i] -= take
            spend += take
        share, rem = divmod(spend, n_p)
        for i in range(n_p):
            bp[i] += share + (1 if i < rem else 0)
        for i in range(n_p):              # the fixed levy, capped by the balance
            take = min(33, bp[i])
            bp[i] -= take
            b_levy += take
        assert sum(br) + sum(bp) + b_initial + b_grant + b_levy == 0
    m = sum(br) + sum(bp)
    return b_grant, b_levy, b_grant + b_levy, m


def run_trace():
    """The hand-worked trace of spec 17.1, with the scale S in the numerator as the
    spec writes it: a household of weight 2 holding 1 000, a rate grant of 300 a
    tick and a rate levy of 5 000 basis points a year with an exemption of 500,
    365 ticks a year, S = 10 000. Returns (tick, grant, levy, balance) rows."""
    w, bal, S, Pg, Pl = 2, 1_000, 10_000, 1, 365    # the grant is per tick, the levy per year
    gcarry, lcarry = 0, 0
    rows = []
    for tick in range(4):
        num = w * 300 * S + gcarry
        g, gcarry = divmod(num, Pg * 10**4)
        bal += g
        base = max(0, bal - w * 500)
        num = base * 5_000 * S + lcarry
        d, lcarry = divmod(num, Pl * 10**8)
        d = min(d, base)
        bal -= d
        rows.append((tick, g, d, bal, lcarry))
    return rows


def run_topup():
    """Spec 10.2: a counterparty at Level 5 000 is topped up, and a rate levy of
    10 000 basis points a tick with Exempt 5 000 sweeps the excess."""
    out = []
    for bal in (3_800, 6_300):
        topup = max(0, 5_000 - bal)
        base = max(0, bal - 5_000)
        sweep = base * 10_000 * 10_000 // (1 * 10**8)
        out.append((bal, topup, sweep))
    return out


if __name__ == "__main__":
    tr, mstar = run_steady_state()
    print("hoarder balance per household at tick t (rate 5000 bp/year, stipend 300/tick, 365 ticks/year):")
    for t, m in tr:
        print(f"  t={t:5d}  balance={m}")
    print("  predicted bound exempt + stipend/rho =", mstar)
    tr, mstar = run_steady_state(exempt=1_000)
    print("with exempt 1000: final", tr[-1], "predicted", mstar)
    print("carry example (account, first payment tick, total in a year):", run_carry_example())
    print("weight check (weight, paid, w*7):", run_weight_check())
    print("pair (grant boundary, levy boundary, net, money in the economy):", run_pair())
    print("trace (tick, grant, levy, balance, levy carry):", run_trace())
    print("topup and sweep (balance, topup, sweep):", run_topup())
