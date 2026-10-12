"""Stylised money circuit of a world with hired labour (spec-employment.md section 11, EMP-A3).

A prototype of the spec's rules, not of the library. Standard library only. Every
number is a placeholder (N-015, N-027).

The question: workers get wages, employers keep the rest of each execution's value, so
who buys the output? The model fixes the price (it isolates the quantity adjustment)
and has one good, a staple made by a source recipe (no inputs, Duration 1, `a` units per
execution) that every household eats, `u` units a tick.

  * Wage: half-even `R + phi * (Bid - R)` with reservation R = 0 and Bid = a*p, `phi`
    the worker share of the margin in basis points (5 000 is the midpoint).
  * Workers spend what they hold on upkeep, up to `u` units a tick, and nothing else.
  * An employer buys its upkeep, then spends the **cap of AGT 7.4.4** on more of the
    staple (a stand-in for the buffers and the wants of DEM 4.5):
        e   = max(0, money - ReserveTicks * UpkeepCost)
        H   = SpendHalfTicks * UpkeepCost
        cap = SpendMaxShare * (e * H / (e + H)) / 10 000
  * The production halt of AGT 7.1.5: employers post no vacancy while the stock of the
    staple is at or above `halt_bp` of one tick's output target, taken as the units sold
    in the last tick.
  * Money is conserved: wages, purchases and revenue only move it. Revenue is shared
    equally between the employers (they are identical).

The closed form to check is section 11.2. With the halt the output follows sales, sales
are the workers' wages plus the employers' spending, and the employers' spending cannot
exceed their upkeep plus the asymptote of the cap, `SpendMaxShare * SpendHalfTicks`
ticks of upkeep, so the hired households settle on average at

    h* = min( E * H_places,  E * u_e * (1 + SpendMaxShare * SpendHalfTicks / 10 000)
                              / ((1 - phi) * a) )

with `phi` the worker share (5 000 is the midpoint). With SpendMaxShare 0 the employers
buy upkeep only: `h* = E * u_e / ((1 - phi) * a)`. The halt makes hiring come in bursts;
the check is on the mean over a long window.
"""
def half_even_div(n, d):
    q, r = divmod(n, d)
    if 2 * r > d or (2 * r == d and q % 2 == 1):
        q += 1
    return q

def run(N=200, E=10, H=10, a=6, u=3, ue=3, p=1000, phi_bp=5000, smax_bp=0, shalf=10,
        ticks=6000, halt_bp=30000, reserve_ticks=3, mw0=10, me0=40):
    wage = half_even_div(phi_bp * a * p, 10000)
    upkeep_cost = ue * p
    m_w = [mw0 * u * p] * N
    m_e = [me0 * ue * p] * E
    stock = 0
    sold_last = N * u + E * ue
    hist = []
    for t in range(ticks):
        V = 0 if stock * 10000 >= halt_bp * sold_last else E * H
        hired = 0
        for e in range(E):
            k = min(H if V else 0, m_e[e] // wage if wage else 0)
            hired += k
            m_e[e] -= k * wage
        start = (t * 7) % N
        for j in range(hired):
            m_w[(start + j) % N] += wage
        stock += a * hired
        demand = 0
        for i in range(N):
            q = min(u, m_w[i] // p)
            m_w[i] -= q * p
            demand += q
        for e in range(E):
            q = min(ue, m_e[e] // p)
            m_e[e] -= q * p
            ex = max(0, m_e[e] - reserve_ticks * upkeep_cost)
            hh = shalf * upkeep_cost
            cap = smax_bp * (ex * hh // (ex + hh)) // 10000 if ex + hh else 0
            extra = cap // p
            m_e[e] -= extra * p
            demand += q + extra
        sold = min(stock, demand)
        stock -= sold
        sold_last = max(1, sold)
        revenue = sold * p
        share, rem = divmod(revenue, E)
        for e in range(E):
            m_e[e] += share + (1 if (e - t) % E < rem else 0)
        hist.append((hired, sold, stock, sum(m_w), sum(m_e)))
    return hist

if __name__ == "__main__":
    N, E, H, a, u, ue = 200, 10, 10, 6, 3, 3
    print("phi_bp smax_bp | last 3000 ticks: mean hired, predicted h*, mean sold per tick, units per worker-tick | money in workers, in employers")
    for phi in (3000, 5000, 7500, 9000):
        for smax in (0, 2500, 5000, 9000):
            h = run(phi_bp=phi, smax_bp=smax)
            tail = h[-3000:]
            hired = sum(x[0] for x in tail) / len(tail)
            sold = sum(x[1] for x in tail) / len(tail)
            pred = min(E * ue * (1 + smax * 10 / 10000) / ((1 - phi / 10000) * a), E * H)
            print("%5d %5d | hired %6.1f  h* %6.1f  sold %6.1f  per worker %.2f | Mw %d Me %d" % (
                phi, smax, hired, pred, sold, (sold - E * ue) / N, tail[-1][3], tail[-1][4]))
