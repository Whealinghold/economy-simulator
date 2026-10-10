import re,io,contextlib
src=open('proto3.py').read().split("UP_A=")[0]
exec(src)
UP_A={"loaf":2,"fuel":1}
recs,commod,dur,yp=bread(True)
cols,out=solve(recs,commod,dur,["north"],{"north":1000},UP_A,yp,F(1),False)
grp,cc,names,(val,x,y,piv)=out[0]
p={nm[1]:yy for nm,yy in zip(names,y) if nm[0]=="bal"};w=[yy for nm,yy in zip(names,y) if nm[0]=="lab"][0]
B=sum(UP_A[c]*p[c] for c in UP_A);scale=F(1000)/B
def rhe(q):  # round half to even
    n,d=q.numerator,q.denominator;f=n//d;r=q-f
    return f+1 if r>F(1,2) or (r==F(1,2) and f%2==1) else f
floor=50
P={c:max(floor,rhe(p[c]*scale)) for c in commod}
print("exact*scale:",{c:float(p[c]*scale) for c in commod});print("int seed:",P)
print("basket",sum(UP_A[c]*P[c] for c in UP_A))
W=w*scale
for i,k in enumerate(cc):
    g,r,E,J=cols[k];nv=net(r,E,dur,commod)
    va=sum(P[c]*nv[c] for c in commod)
    print("%-15s x=%-9.4f E=%.3f J=%-3d d=%-3d VA/exec=%9.1f wage*d=%9.1f dev=%+.2f%%"%(r.name,float(x[i]),float(E),J,r.d,float(va),float(r.d*W),100*float((va-r.d*W)/(r.d*W)) if x[i]>0 else 0))
print("wage per tick exact",float(W))
