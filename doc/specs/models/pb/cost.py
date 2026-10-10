# N-055: unit cost with labour counted at the wage, for the viable bread chain (spec-viability 15)
from fractions import Fraction as F
src=open('proto3.py').read().split("UP_A=")[0]
exec(src)
UP_A={"loaf":2,"fuel":1}
recs,commod,dur,yp=bread(True)
cols,out=solve(recs,commod,dur,["north"],{"north":1000},UP_A,yp,F(1),False)
grp,cc,names,(val,x,y,piv)=out[0]
p={nm[1]:yy for nm,yy in zip(names,y) if nm[0]=="bal"};w=[yy for nm,yy in zip(names,y) if nm[0]=="lab"][0]
B=sum(UP_A[c]*p[c] for c in UP_A);scale=F(1000)/B
def rhe(q):
    n,d=q.numerator,q.denominator;f=n//d;r=q-f
    return f+1 if r>F(1,2) or (r==F(1,2) and f%2==1) else f
floor=50
P={c:max(floor,rhe(p[c]*scale)) for c in commod}
W=w*scale
print("wage per working tick %.1f"%float(W))
print("%-10s %8s %9s %9s %9s %7s %7s"%("good","seed","in+wear","labour","fullcost","seed/old","seed/new"))
rows={}
for i,k in enumerate(cc):
    g,r,E,J=cols[k]
    if x[i]==0: continue
    nonq=[c for c in r.out if c not in r.discard]
    # cost shares: all to the main output (share 10000) -- byproducts get 0 (as in spec 12.1)
    main={"grow_grain":"grain","mill_flour":"flour","knead_dough":"dough","bake_loaf":"loaf","cut_wood":"fuel","mine_iron":"iron","make_hoe":"hoe","make_millstone":"millstone","make_oven":"oven","save_seed":"seed"}[r.name]
    inc=sum(q*P[c] for c,q in r.inp.items())
    wear=sum(F(wu*P[t],dur[t]) for t,wu in r.tools.items())
    lab=r.d*W
    q=r.out[main]*E
    old=(inc+wear)/q; new=(inc+wear+lab)/q
    rows[main]=(old,new)
    print("%-10s %8d %9.1f %9.1f %9.1f %7.2f %7.2f   (exec %s, d=%d, E=%.2f, x=%.4f)"%(main,P[main],float(old),float(lab/q),float(new),float(P[main]/old) if old else 0,float(P[main]/new),r.name,r.d,float(E),float(x[i])))
