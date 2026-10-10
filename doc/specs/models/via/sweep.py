import io,contextlib
full=open("proto3.py").read().split("# ---------------- cost probe")[0]
buf=io.StringIO()
with contextlib.redirect_stdout(buf): exec(full)
UP_A={"loaf":2,"fuel":1}
import math
def rhe(q):
    n,d=q.numerator,q.denominator;f=n//d;r=q-f
    return f+1 if r>F(1,2) or (r==F(1,2) and f%2==1) else f
def run(name,recs,commod,dur,yp,regs,H,up,floor=50):
    cols,out=solve(recs,commod,dur,regs,H,up,yp,F(1),False)
    for grp,cc,names,(val,x,y,piv) in out:
        lam=x[-1]
        p={nm[1]:yy for nm,yy in zip(names,y) if nm[0]=="bal"};w=[yy for nm,yy in zip(names,y) if nm[0]=="lab"][0]
        B=sum(up[c]*p[c] for c in up);s=F(1000)/B
        P={c:max(floor,rhe(p[c]*s)) for c in commod}
        W=rhe(w*s)
        print(name,grp,"lambda*=%.3f"%float(lam),"viable" if lam>=1 else "NOT","wage=%d"%W)
        best={}
        for i,k in enumerate(cc):
            g,r,E,J=cols[k]
            if x[i]<=0: continue
            nv=net(r,E,dur,commod)
            inp=sum(q*P[c] for c,q in r.inp.items())+sum(F(wr,dur[t])*P[t] for t,wr in r.tools.items())
            cost=inp+r.d*W
            for c,q in r.out.items():
                if c in r.discard or q==0: continue
                uc=math.ceil(cost/(q*E))   # share 10000 for the main output
                if r.name in("grow_grain",) and c=="straw": continue
                if c in("bran",): continue
                best[c]=(uc,r.name)
        for c,(uc,rn) in sorted(best.items()):
            print("   %-10s seed %5d  full unit cost %5d (%s)  ratio %.3f"%(c,P[c],uc,rn,P[c]/uc))
season_of=_old
recs,commod,dur,yp=bread(True)
run("bread",recs,commod,dur,yp,["north"],{"north":1000},UP_A if 'UP_A' in globals() else {"loaf":2,"fuel":1})
season_of=lambda t:"lit" if _old(t) in("spring","summer") else "dark"
recs,commod,dur,yp=space()

run("space",recs,commod,dur,yp,["crater_rim"],{"crater_rim":1000},{"biomass":2,"water":1})
