"""models/crd/loans.py: seasonal credit for a long job, with a shock, in integers.

A model of the RULES of spec-credit.md (the default borrower of 10.2, the default
lender of 10.3, the round of 6, settlement of 8), with the economy around them
replaced by an assumed income: a borrower earns a lump `Y` at the end of each job.
It shows structure and sensitivity, not that any default is stable (N-027).

Money is conserved: a `market` account pays every income and receives every
upkeep and input purchase, so the sum over all accounts is constant. All numbers
are placeholders.

Run (Docker only, never a local interpreter):
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/crd":/w:ro -w /w python:3.12-slim python loans.py
"""
from alloc import allocate, interest, key, split

TPY = 365
U = 100          # upkeep cost of a borrower per tick
U_L = 80         # upkeep cost of a lender per tick
D = 60           # job duration, ticks
SLACK = 0        # SlackTicks: ticks added to the horizon for the time to sell (10.2)
S_IN = 200       # inputs bought at the start of a job
M0 = 1500        # a borrower's money at the start
Y_BASE = 7000    # the lump at the end of a job
NB, NL = 16, 4
LEND_CASH = 50_000
RATE, MAX_TERM, KEEP, SHARE = 600, 90, 3, 5000
MAX_RATE, MAX_DEBT_TICKS = 1500, 90
MIN_PRINCIPAL = 100
T_END = 480


def run(shield=1, write_off=0, credit=True, shock=None, plus=1, tier2_mode="full", m0=M0, jobs0=True, min_p=MIN_PRINCIPAL, slack=SLACK, buy_p3=False, late=True):
    """shock: (from_tick, to_tick, income) applied to jobs that complete in [from, to).
    plus: the structural tick of the horizon of 10.2 (the cash is in the balance one tick after
    the completion tick, WLD 6.1.8); 0 reproduces the horizon of 10.2 without its structural tick, which is a tick short.
    jobs0: False starts every borrower with no job (the not-started branch of 10.2 at tick 0).
    tier2_mode: "full" the cost of the next job's inputs whatever money allows (10.2);
    "afford" what tier 2 can spend after tier 1 (a request sized by it under-asks).
    buy_p3: the library's delay of the inputs: they are bought at P3.2 of a tick from the money held then (a loan
    formed at P3.3b is spendable from the next tick, 6.4.2) and the job starts at P2.7 of the next; False pays them at
    the start of the job from the money held at P2 (kind to the borrower, the older cases).
    late: the rule of 10.2 that adds one tick to the horizon of a not-started job whose money after tier 1 is below
    `Tier2` (the inputs are then bought a tick later, so the job starts and completes a tick later)."""
    market = 10**9
    money = {("b", i): m0 for i in range(NB)}
    money.update({("l", j): LEND_CASH for j in range(NL)})
    total0 = market + sum(money.values())
    job = {}          # borrower -> remaining ticks of the job in progress
    ready = {}        # borrower -> inputs bought at P3 of the previous tick (buy_p3)
    start0 = {i: 0 for i in range(NB)}
    for i in range(NB if jobs0 else 0):  # jobs already in progress, staggered: remaining 60, 56, 52, ...
        job[("b", i)] = D - (i * 4) % D
    claims = {}       # id -> dict(cred, debt, po, io, due, over)
    nxt = 1
    st = dict(unfed_b=0, unfed_l=0, jobs=0, formed=0, lent=0, int_paid=0, defaults=0,
              lost=0, refused=0, unfilled=0, repaid=0, jobs_late=0)
    for t in range(T_END):
        # P1.7a settlement
        debtors = sorted({c["debt"] for c in claims.values()
                          if c["due"] == t or c["over"] >= 0})
        for dname in debtors:
            mine = sorted(cid for cid, c in claims.items() if c["debt"] == dname
                          and (c["due"] == t or c["over"] >= 0))
            owed = [claims[c]["po"] + claims[c]["io"] for c in mine]
            pot = max(0, money[dname] - shield * U)
            pay = min(pot, sum(owed))
            for cid, sh in zip(mine, allocate(pay, owed, t, mine)):
                c = claims[cid]
                ip, pp = split(sh, c["io"])
                c["io"] -= ip
                c["po"] -= pp
                money[dname] -= sh
                money[c["cred"]] += sh
                st["int_paid"] += ip
                st["repaid"] += pp
            for cid in mine:
                c = claims[cid]
                if c["po"] == 0 and c["io"] == 0:
                    del claims[cid]
                    continue
                if c["due"] == t and c["over"] < 0:
                    c["over"] = t
                    st["defaults"] += 1
                if write_off and c["over"] >= 0 and t >= c["due"] + write_off:
                    st["lost"] += c["po"] + c["io"]
                    del claims[cid]
        # P2 starts
        for i in range(NB):
            b = ("b", i)
            if buy_p3:
                if b not in job and ready.pop(b, False):
                    job[b] = D
            elif b not in job and t >= start0[i] and money[b] >= S_IN:
                job[b] = D
                money[b] -= S_IN
                market += S_IN
        # P3 requests and offers
        requests, offers = [], []
        if credit:
            for i in range(NB):
                b = ("b", i)
                if b in job:
                    # job[b] counts this tick's step, so the library's Remaining is job[b] - 1
                    h, tier2 = job[b] - 1, 0
                elif t >= start0[i]:
                    # not started: it starts next tick, so the first completion is D ticks away
                    h = D
                    tier2 = S_IN if tier2_mode == "full" else min(S_IN, max(0, money[b] - U))
                    if buy_p3 and late and money[b] - U < S_IN:
                        h += 1   # 10.2: the inputs cannot be bought at once, so the job starts a tick later
                else:
                    continue
                h += plus + slack
                debt = sum(c["po"] for c in claims.values() if c["debt"] == b)
                room = max(0, MAX_DEBT_TICKS * U - debt)
                need = U * h + tier2 - money[b]
                ask = min(need, room)
                if ask >= min_p:
                    requests.append([b, ask, h, MAX_RATE])
            for j in range(NL):
                l = ("l", j)
                lendable = max(0, money[l] - KEEP * U_L)
                off = SHARE * lendable // 10_000
                if off >= min_p:
                    offers.append([l, off, MAX_TERM, RATE])
        if buy_p3:  # P3.2: inputs of a not-started job, after tier 1, from the money held (the loan is not yet in)
            for i in range(NB):
                b = ("b", i)
                if b not in job and t >= start0[i] and money[b] - U >= S_IN:
                    money[b] -= S_IN
                    market += S_IN
                    ready[b] = True
        # the round: lanes are regions; one region here
        overdue_debtors = {c["debt"] for c in claims.values() if c["over"] >= 0}
        adm = []
        for r in requests:
            if r[0] in overdue_debtors:
                st["refused"] += 1
            else:
                adm.append(r)
        adm.sort(key=lambda r: (-r[3], key(t, r[0][1]), r[0][1]))
        offers.sort(key=lambda o: (o[3], key(t, o[0][1] + 1000), o[0][1]))
        for r in adm:
            p = r[1]
            for o in offers:
                if p <= 0:
                    break
                if o[3] > r[3]:
                    break
                if o[2] < r[2] or o[1] <= 0:
                    continue
                x = min(p, o[1])
                if x < min_p:
                    if x == p:
                        break
                    continue
                i_amt = interest(x, o[3], r[2], TPY)
                money[o[0]] -= x
                money[r[0]] += x
                claims[nxt] = dict(cred=o[0], debt=r[0], po=x, io=i_amt, due=t + r[2], over=-1)
                nxt += 1
                st["formed"] += 1
                st["lent"] += x
                p -= x
                o[1] -= x
            st["unfilled"] += max(0, p)
        # P6 consumption
        for k in list(money):
            u = U if k[0] == "b" else U_L
            pay = min(money[k], u)
            money[k] -= pay
            market += pay
            if pay < u:
                st["unfed_b" if k[0] == "b" else "unfed_l"] += 1
        # completion: the lump arrives at the end of the tick
        for b in list(job):
            job[b] -= 1
            if job[b] == 0:
                del job[b]
                y = Y_BASE
                if shock and shock[0] <= t < shock[1]:
                    y = shock[2]
                market -= y
                money[b] += y
                st["jobs"] += 1
        assert market + sum(money.values()) == total0, "money conserved"
    owed_open = sum(c["po"] + c["io"] for c in claims.values())
    over_open = sum(c["po"] + c["io"] for c in claims.values() if c["over"] >= 0)
    st.update(open=len(claims), owed_open=owed_open, arrears=over_open,
              lender_cash=sum(v for k, v in money.items() if k[0] == "l"),
              borrower_cash=sum(v for k, v in money.items() if k[0] == "b"))
    return st


CASES = [
    ("A no credit", dict(credit=False)),
    ("B credit, shield 1", dict()),
    ("B0 as B, horizon without the +1", dict(plus=0)),
    ("C one bad cycle, shield 1", dict(shock=(180, 240, 1500))),
    ("D one bad cycle, shield 0", dict(shield=0, shock=(180, 240, 1500))),
    ("D2 one bad cycle, shield 3", dict(shield=3, shock=(180, 240, 1500))),
    ("E bad from tick 180 on, no write-off", dict(shock=(180, 10**9, 1500))),
    ("F bad from tick 180 on, write-off 120", dict(shock=(180, 10**9, 1500), write_off=120)),
    ("G0 money 150 (below the inputs), no credit", dict(credit=False, m0=150, jobs0=False)),
    ("G1 money 150, tier 2 as affordable", dict(m0=150, jobs0=False, tier2_mode="afford")),
    ("G2 money 150, tier 2 in full", dict(m0=150, jobs0=False)),
    ("G3 money 150, MinPrincipal 200, tier 2 as affordable", dict(m0=150, jobs0=False, tier2_mode="afford", min_p=200)),
    ("G4 money 150, MinPrincipal 200, tier 2 in full", dict(m0=150, jobs0=False, min_p=200)),
    ("H0 money 150, library delay, horizon J + 1", dict(m0=150, jobs0=False, buy_p3=True, late=False)),
    ("H1 the same, +1 when the inputs are not affordable (10.2)", dict(m0=150, jobs0=False, buy_p3=True)),
]

if __name__ == "__main__":
    cols = ["unfed_b", "jobs", "formed", "lent", "int_paid", "defaults", "lost", "refused", "unfilled",
            "open", "arrears", "lender_cash", "unfed_l"]
    print(f"{NB} borrowers (job {D} ticks, upkeep {U}, inputs {S_IN}, start money {M0}, lump {Y_BASE}),"
          f" {NL} lenders (cash {LEND_CASH}, rate {RATE} bp, keep {KEEP}, share {SHARE} bp), {T_END} ticks")
    print("%-50s" % "case" + "".join("%11s" % c for c in cols))
    for name, kw in CASES:
        r = run(**kw)
        print("%-50s" % name + "".join("%11d" % r[c] for c in cols))
    print()
    print("one bad cycle (income 1500 from tick 180 to 240): the shield, at SlackTicks 0 and 5")
    print("%-8s" % "shield" + "".join("%10s" % c for c in ("unfed_b", "refused", "defaults", "unfed_l")) + "   |   same at SlackTicks 5")
    for s in (0, 1, 2, 3, 5, 8):
        r0 = run(shield=s, shock=(180, 240, 1500))
        r5 = run(shield=s, shock=(180, 240, 1500), slack=5)
        print("%-8d" % s + "".join("%10d" % r0[c] for c in ("unfed_b", "refused", "defaults", "unfed_l")) + "   |   " + "".join("%10d" % r5[c] for c in ("unfed_b", "refused", "defaults", "unfed_l")))
    print("money conserved at every tick of every case (asserted)")
