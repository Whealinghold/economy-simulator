"""POP model 2: the stake loop of spec-population.md 4.5 and 5.6 (money only).

Isolates the funding of births. The welfare ratio is held fixed, so the schedules give constant rates (b and d, bp a
year, same for every household); only the money leg varies. Households hold money that is exchanged in random pairs
(each pair re-splits its combined money at random: conserved, exponential-like spread); a death moves the dying
household's money to the pool; a birth takes the stake from the pool, in whole households, within the budget
EntryPoolShare of the pool (SPN 7.3), and none if the pool cannot pay. Relief is left out (it would return pool money to
households). Deaths are taken poorest first (SPN 8.2 as written before this spec) or spread (a fixed stride through the
households, a stand-in for 'each household dies with its own probability'). The exchange process is an assumption,
not a model of the market. Every number is a placeholder."""
import random
TPY = 365; D = 10_000 * TPY

def run(stake, order, N0=1000, years=100, mean_money=12_000, pool0=0, seed=1, entry_pool_share=5_000, b=250, d=250):
    rng = random.Random(seed)
    m = [mean_money] * N0
    for _ in range(200 * N0):
        i, j = rng.randrange(len(m)), rng.randrange(len(m))
        if i != j:
            tot = m[i] + m[j]; a = rng.randrange(tot + 1); m[i], m[j] = a, tot - a
    pool = pool0; cb = cd = 0; stride = 0; funded_tot = due_tot = 0; out = []
    for t in range(years * TPY):
        N = len(m)
        for _ in range(max(1, N // 20)):
            i, j = rng.randrange(N), rng.randrange(N)
            if i != j:
                tot = m[i] + m[j]; a = rng.randrange(tot + 1); m[i], m[j] = a, tot - a
        cd += N * d; cb += N * b
        deaths = min(cd // D, N - 1); cd %= D
        births = cb // D; cb %= D
        for _ in range(deaths):
            if order == "poorest": k = min(range(len(m)), key=m.__getitem__)
            else: stride = (stride + 7919) % len(m); k = stride
            pool += m.pop(k)
        budget = pool * entry_pool_share // 10_000
        due_tot += births
        n = births if stake == 0 else min(births, budget // stake)
        funded_tot += n; pool -= stake * n; m.extend([stake] * n)
        if t % (10 * TPY) == 0: out.append((t // TPY, len(m), pool, sum(m) // len(m), funded_tot, due_tot))
    return out

if __name__ == "__main__":
    print("== A. births = deaths (250 bp), pool empty at the start; N0 1000; mean money 12 000")
    print("year:N(pool) every 20 years | births funded / due")
    for stake in (5_000, 10_000):
        for order in ("poorest", "spread"):
            o = run(stake, order)
            print(f"stake {stake:>6} deaths {order:>8} | " + "  ".join(f"{y}:{n}({p//1000}k)" for y, n, p, _, _, _ in o[::2]) + f" | {o[-1][4]}/{o[-1][5]}")
    print("\n== B. births above deaths (b 400, d 250), deaths spread: growth until the money runs out; M = N0 x 12 000 = 12 000 000")
    print("year:N (mean money per household) every 20 years; N_M = M / stake")
    for stake in (10_000, 5_000, 2_500):
        o = run(stake, "spread", years=200, b=400, d=250)
        print(f"stake {stake:>6} N_M {12_000_000 // stake:>6} | " + "  ".join(f"{y}:{n}({mm})" for y, n, p, mm, _, _ in o[::2]))
