import b_bel
from b_bel import *
import collections
cnt = collections.defaultdict(lambda: [0,0,0])  # n, sum ev up, sum ev down
orig = b_bel.update
def traced(a, fl, ce, o_limit, o_qty, o_fill, mp, age, bestbid, bestask, target, u, P):
    c0 = a.centre; w0=a.width
    res = orig(a, fl, ce, o_limit, o_qty, o_fill, mp, age, bestbid, bestask, target, u, P)
    return res
# simpler: recompute ev inside by copying the T logic
def kind(a, o_qty, o_fill):
    return ('bid' if a.bid else 'ask', 'full' if o_fill==o_qty else ('none' if o_fill==0 else 'part'))
def run(P, label):
    cnt.clear()
    import types
    src = open('b_bel.py').read()
    # instrument: wrap ev
    src = src.replace("    v = 0\n    if target > 0:", "    TR.append((a.bid, 'full' if unfilled==0 else ('none' if o_fill==0 else 'part'), ev))\n    v = 0\n    if target > 0:")
    ns = {'TR': []}
    exec(compile(src, 'b_bel_i', 'exec'), ns)
    TR = ns['TR']
    for s in range(4):
        ns['simulate'](seed=s, ticks=800, Q=100, D=100, ed=0.5, es=0.5, P=P)
    d = collections.defaultdict(lambda: [0,0,0])
    for bid, k, ev in TR:
        e = d[('bid' if bid else 'ask', k)]
        e[0]+=1; e[1]+= (ev>0); e[2]+= (ev<0)
    print(label)
    for k in sorted(d): print('  ', k, 'n', d[k][0], 'up', d[k][1], 'down', d[k][2])
run({}, 'as written')
run({'EV':'limit'}, 'EV=limit')
