from b_bel import *
import statistics
print("== N-039: perishable glut / hard cost floor. 20+20 agents, ref 1000, seller unit cost 1100, buyer willingness-to-pay cap 1000, shelf life 3 ticks")
print("   (buyer cap stands in for a budget; trades need a bid >= an ask >= the seller floor)")
for cf in (10000, 9000, 8000, 5000):
    tr, sp, p = [], [], []
    for s in range(6):
        r = simulate(seed=s, ticks=300, Q=100, D=100, seller_cost=1100, wtp=1000, shelf=3, P=dict(CostFloor=cf))
        tr.append(sum(1 for v in r['vol'][100:] if v > 0)/200)
        sp.append(r['spoiled']/r['produced'])
        p.append(statistics.mean(r['p'][100:]))
    print(f"CostFloor={cf:5d} (floor {1100*cf//10000}) ticks with a trade {statistics.median(tr):.2f}  share of output spoiled {statistics.median(sp):.2f}  mean price {statistics.median(p):.0f}")
print("== same with WTP 1200 (buyers can pay cost): no freeze expected")
r = simulate(seed=0, ticks=300, Q=100, D=100, seller_cost=1100, wtp=1200, shelf=3)
print("ticks with trade", sum(1 for v in r['vol'][100:] if v > 0)/200, "spoiled", r['spoiled']/r['produced'])
