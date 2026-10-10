# PROTOTYPE of the equations of spec-viability.md. Not the implementation. Exact rational arithmetic.
from fractions import Fraction as F
import time, sys

YEAR=365
SEAS=[("spring",92),("summer",92),("autumn",91),("winter",90)]
def season_of(t):  # Earth default: seasons start on day 59, one tick = one day
    d=(t-59)%YEAR
    for i,(n,l) in enumerate(SEAS):
        if d<l: return n
        d-=l

class Rec:
    def __init__(s,name,inputs,outputs,d=1,tools=(),sited=False,discard=()):
        s.name=name;s.inp=dict(inputs);s.out=dict(outputs);s.d=d;s.tools=dict(tools);s.sited=sited;s.discard=set(discard)

def start_profile(r,yp,region):
    """greedy starts for one household over a year; returns (jobs per year J, mean effective factor E as Fraction)."""
    prof=yp.get((region,r.name))
    def fac(t):
        if prof is None: return F(0) if r.sited else F(1)
        if 'all' in prof: return F(prof['all'],10000)
        return F(prof.get(season_of(t),0),10000)
    t=0;n=0;tot=F(0)
    while t<YEAR:
        f=fac(t)
        if f>0: n+=1;tot+=f;t+=r.d
        else: t+=1
    return n,(tot/n if n else F(0))

def simplex_max(A,b,c):
    """max c.z, A z <= b, z>=0, b>=0. Bland's rule. returns (value, z, y(duals), pivots)."""
    m=len(A);n=len(c)
    T=[[F(v) for v in A[i]]+[F(1) if j==i else F(0) for j in range(m)]+[F(b[i])] for i in range(m)]
    z=[-F(v) for v in c]+[F(0)]*m+[F(0)]
    basis=[n+i for i in range(m)];piv=0
    while True:
        col=next((j for j in range(n+m) if z[j]<0),None)
        if col is None: break
        best=None
        for i in range(m):
            if T[i][col]>0:
                ratio=T[i][-1]/T[i][col]
                if best is None or ratio<best[0] or (ratio==best[0] and basis[i]<basis[best[1]]): best=(ratio,i)
        if best is None: return None
        r=best[1];pv=T[r][col];T[r]=[v/pv for v in T[r]]
        for i in range(m):
            if i!=r and T[i][col]!=0:
                f=T[i][col];T[i]=[a-f*bb for a,bb in zip(T[i],T[r])]
        f=z[col];z=[a-f*bb for a,bb in zip(z,T[r])];basis[r]=col;piv+=1
    x=[F(0)]*(n+m)
    for i,bj in enumerate(basis): x[bj]=T[i][-1]
    return z[-1],x[:n],z[n:n+m],piv

def build(recs,commod,tools_dur,regions,H,upkeep,yp,phi,trade):
    """regions: list; H[region]=households; upkeep: dict commodity->qty per household per tick.
    trade=True: one pooled balance for the realm; False: autarky (call per region)."""
    cols=[]  # (region,rec,E,J)
    for g in regions:
        for r in recs:
            J,E=start_profile(r,yp,g)
            if J>0: cols.append((g,r,E,J))
    return cols

def net(r,E,tools_dur,commod):
    v={c:F(0) for c in commod}
    for c,q in r.out.items():
        if c not in r.discard: v[c]+=E*q
    for c,q in r.inp.items(): v[c]-=q
    for t,w in r.tools.items(): v[t]-=F(w,tools_dur[t])
    return v

def solve(recs,commod,tools_dur,regions,H,upkeep,yp,phi,trade,use_cap=True,label=""):
    cols=build(recs,commod,tools_dur,regions,H,upkeep,yp,phi,trade)
    nx=len(cols);C=list(commod)
    groups=[regions] if trade else [[g] for g in regions]
    out=[]
    for grp in groups:
        cc=[k for k,(g,r,E,J) in enumerate(cols) if g in grp]
        D={c:sum(H[g]*upkeep.get(c,0) for g in grp) for c in C}
        rows=[];b=[];names=[]
        for c in C:  # -N x + lam D <= 0
            rows.append([(-net(cols[k][1],cols[k][2],tools_dur,commod)[c]) for k in cc]+[D[c]]);b.append(0);names.append(("bal",c))
        for g in grp:
            rows.append([F(cols[k][1].d) if cols[k][0]==g else F(0) for k in cc]+[F(0)]);b.append(phi*H[g]);names.append(("lab",g))
        if use_cap:
            for i,k in enumerate(cc):
                g,r,E,J=cols[k]
                if J*r.d<YEAR*1 and True:
                    rows.append([F(1) if j==i else F(0) for j in range(len(cc))]+[F(0)]);b.append(F(H[g]*J,YEAR));names.append(("cap",i))
        cobj=[F(0)]*len(cc)+[F(1)]
        res=simplex_max(rows,b,cobj)
        out.append((grp,cc,names,res))
    return cols,out

def report(label,recs,commod,tools_dur,regions,H,upkeep,yp,phi,trade):
    cols,out=solve(recs,commod,tools_dur,regions,H,upkeep,yp,phi,trade)
    print("==",label,"trade" if trade else "autarky")
    for grp,cc,names,res in out:
        if res is None: print(" group",grp,": LP unbounded?!");continue
        val,x,y,piv=res
        lam=x[-1]
        print(" group",grp,"lambda*=%s (%.3f) pivots=%d"%(lam,float(lam),piv), "VIABLE" if lam>=1 else "NOT VIABLE (upkeep not coverable)")
        p={};w={}
        for nm,yy in zip(names,y):
            if nm[0]=="bal": p[nm[1]]=yy
            if nm[0]=="lab": w[nm[1]]=yy
        # labour share needed for lambda=1 : phi/lambda (valid when no cap binds)
        print("   labour share to subsist (if caps slack) phi/lam = %.3f"%float(phi/lam) if lam>0 else "")
        act=[(cols[k][0],cols[k][1].name,float(x[i])) for i,k in enumerate(cc) if x[i]>0]
        print("   active:",[(g,n,round(q,4)) for g,n,q in act])
        print("   prices (basket=1):",{c:round(float(v),4) for c,v in p.items() if v>0})
        print("   wage/tick:",{g:round(float(v),4) for g,v in w.items()})
        # check complementary slackness: va = d*w for active, <= for inactive
        bad=0
        for i,k in enumerate(cc):
            g,r,E,J=cols[k];nv=net(r,E,tools_dur,commod);va=sum(p[c]*nv[c] for c in commod)
            wg=w[g] if g in w else w[grp[0]]
            if x[i]>0 and va>r.d*wg: bad+=1
            if va>r.d*wg+0: bad+=1 if x[i]==0 else 0
        print("   dual feasibility violations:",bad)
    return cols,out

# ------------------------------------------------ Example A: bread chain (completed with placeholders)
def bread(fixed):
    g=lambda a:a
    grain_out=1500 if fixed else 6
    seed_in=60 if fixed else 1
    recs=[Rec("grow_grain",{"seed":seed_in},{"grain":grain_out,"straw":100 if fixed else 2},d=120,tools={"hoe":1},sited=True),
          Rec("save_seed",{"grain":1},{"seed":1}),
          Rec("mill_flour",{"grain":3},{"flour":12,"bran":6},tools={"millstone":1}),
          Rec("knead_dough",{"flour":1},{"dough":12}),
          Rec("bake_loaf",{"dough":2,"fuel":1},{"loaf":18},d=2,tools={"oven":1}),
          Rec("cut_wood",{},{"fuel":12}),
          Rec("mine_iron",{},{"iron":18}),
          Rec("make_hoe",{"iron":1},{"hoe":1}),Rec("make_millstone",{"iron":2},{"millstone":1}),Rec("make_oven",{"iron":2},{"oven":1})]
    commod=["seed","grain","straw","flour","bran","dough","loaf","fuel","iron","hoe","millstone","oven"]
    dur={"hoe":2000,"millstone":2000,"oven":2000}
    yp={("north","grow_grain"):{"spring":10000},("south","grow_grain"):{"spring":12000,"autumn":7000},
        ("far_south","grow_grain"):{"autumn":10000},
        ("north","mine_iron"):{"all":10000},("south","mine_iron"):{"all":5000},("far_south","mine_iron"):{"all":15000}}
    # mine_iron is sited; make it unsited elsewhere: give profiles for all regions above
    return recs,commod,dur,yp

UP_A={"loaf":2,"fuel":1}
for fixed in (False,True):
    recs,commod,dur,yp=bread(fixed)
    H={"north":1000}
    report("bread chain, "+("repaired quantities" if fixed else "AS WRITTEN in spec-production 12.1"),recs,commod,dur,["north"],H,UP_A,yp,F(1),False)

# three regions: yields differ; wood unsited
print("\n#### three-region check (repaired chain)")
recs,commod,dur,yp=bread(True)
for r in recs:
    if r.name=="cut_wood": r.sited=False
yp[("north","cut_wood")]={"all":10000};yp[("south","cut_wood")]={"all":6000};yp[("far_south","cut_wood")]={"all":12000}
regs=["north","south","far_south"];H={g:1000 for g in regs}
report("3 regions",recs,commod,dur,regs,H,UP_A,yp,F(1),False)
report("3 regions",recs,commod,dur,regs,H,UP_A,yp,F(1),True)

# ------------------------------------------------ Example B: space colony
def space():
    recs=[Rec("cut_ice",{},{"ice":30},tools={"drill":2},sited=True),
          Rec("melt",{"ice":1},{"water":6}),
          Rec("grow_algae",{"water":4},{"biomass":120},d=12,sited=True),
          Rec("dig_regolith",{},{"regolith":30}),
          Rec("refine_wafer",{"regolith":3},{"wafer":1}),
          Rec("refine_solder",{"regolith":2},{"solder":1}),
          Rec("fab_board",{"wafer":2,"solder":1},{"board":1,"slag":1},d=3,tools={"fabricator":1},discard=("slag",)),
          Rec("make_drill",{"board":1},{"drill":1}),Rec("make_fabricator",{"board":3,"solder":1},{"fabricator":1})]
    commod=["ice","water","biomass","regolith","wafer","solder","board","slag","drill","fabricator"]
    dur={"drill":500,"fabricator":1000}
    yp={("crater_rim","cut_ice"):{"all":6000},("crater_rim","grow_algae"):{"lit":10000,"dark":2500},
        ("deep_crater","grow_algae"):{"all":0},("deep_crater","cut_ice"):{"all":10000}}
    return recs,commod,dur,yp
recs,commod,dur,yp=space()
# season names for this planet: reuse 'lit' as spring/summer, 'dark' as autumn/winter (prototype shortcut)
_old=season_of
def season_of(t): return "lit" if _old(t) in("spring","summer") else "dark"
UP_B={"biomass":2,"water":1}
print()
report("space colony crater_rim",recs,commod,dur,["crater_rim"],{"crater_rim":1000},UP_B,yp,F(1),False)
report("space colony two regions",recs,commod,dur,["crater_rim","deep_crater"],{"crater_rim":600,"deep_crater":400},UP_B,yp,F(1),True)
report("space colony deep_crater alone (cannot grow algae)",recs,commod,dur,["deep_crater"],{"deep_crater":400},UP_B,yp,F(1),False)

# ---------------- cost probe: synthetic realm, 5 regions, 30 commodities, ~45 recipes (pivot count, not Go speed)
import random
random.seed(1)
regs=["r%d"%i for i in range(5)]
com=["c%d"%i for i in range(30)]
rr=[];yp2={}
for k in range(30):
    ins={}
    if k>=3:
        for j in random.sample(range(k),2): ins[com[j]]=random.randint(1,3)
    rr.append(Rec("m%d"%k,ins,{com[k]:random.randint(8,20)},d=random.choice([1,1,2,3])))
for k in range(15):
    rr.append(Rec("alt%d"%k,{com[random.randrange(k+3,30)] if False else com[random.randrange(0,max(k+2,3))]:1},{com[k+3]:random.randint(8,20)},d=random.choice([1,2])))
for g in regs:
    for r in rr: yp2[(g,r.name)]={"all":random.choice([5000,8000,10000,12000])}
H={g:1000 for g in regs}
up={com[27]:1,com[28]:1,com[29]:1}
t=time.time();c,o=solve(rr,com,{},regs,H,up,yp2,F(1),True)
print("synthetic 5x30x45 pooled:",o[0][3][3],"pivots", "%.1fs python Fractions"%(time.time()-t), "lambda=",float(o[0][3][1][-1]))
