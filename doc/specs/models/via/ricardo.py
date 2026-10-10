src=open('proto3.py').read().split("UP_A=")[0]
exec(src)
UP_A={"loaf":2,"fuel":1}
recs,commod,dur,yp=bread(True)
for r in recs:
    if r.name=="cut_wood": r.sited=False
yp[("north","cut_wood")]={"all":10000};yp[("south","cut_wood")]={"all":6000};yp[("far_south","cut_wood")]={"all":12000}
regs=["north","south","far_south"];H={g:1000 for g in regs}
def prices(trade):
    cols,out=solve(recs,commod,dur,regs,H,UP_A,yp,F(1),trade);res={}
    for grp,cc,names,(val,x,y,piv) in out:
        p={nm[1]:yy for nm,yy in zip(names,y) if nm[0]=="bal"}
        res[tuple(grp)]=(p,{(cols[k][0],cols[k][1].name):x[i] for i,k in enumerate(cc)},x[-1])
    return res
a=prices(False);t=prices(True)
pt=list(t.values())[0][0]
for (u,v) in [("grain","fuel"),("loaf","fuel"),("iron","grain"),("loaf","grain")]:
    ra={g[0]:float(p[u]/p[v]) for g,(p,_,_) in a.items()}
    print(u,"/",v,"autarky",{k:round(r,3) for k,r in ra.items()},"trade",round(float(pt[u]/pt[v]),3) if pt[v] else None)
xt=list(t.values())[0][1]
print("trade activity by region:")
for g in regs: print(" ",g,{n:round(float(q),3) for (gg,n),q in xt.items() if gg==g and q>0})
print("lambda autarky",{g[0]:round(float(v[2]),3) for g,v in a.items()},"pooled",round(float(list(t.values())[0][2]),3))
