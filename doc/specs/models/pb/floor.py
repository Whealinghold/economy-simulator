from b_bel import *
import statistics
print("N-055 knock-on: seller unit cost C (labour counted, so equilibrium price = 1000 = C when the wage is exact), buyers capped at 1000 (budget stand-in), shelf life 3, 6 seeds, ticks 100-300")
print("%-8s %-6s | %s" % ("C", "f", "ticks with a trade / share spoiled / mean price"))
for C in (1000, 1100, 1250):
    for f in (10000, 9000, 8000, 7000, 6000, 5000):
        tr, sp, pr = [], [], []
        for s in range(6):
            r = simulate(seed=s, ticks=300, Q=100, D=100, seller_cost=C, wtp=1000, shelf=3, P=dict(CostFloor=f, EV='guard', EVDIV=2))
            tr.append(sum(1 for v in r['vol'][100:] if v > 0)/200); sp.append(r['spoiled']/r['produced']); pr.append(statistics.mean(r['p'][100:]))
        print("%-8d %-6d | %.2f  %.2f  %5.0f   floor=%d" % (C, f, statistics.median(tr), statistics.median(sp), statistics.median(pr), C*f//10000))
