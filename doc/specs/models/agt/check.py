# Arithmetic check of the worked examples in spec-agents.md (integer rules of the spec; a model of the spec, not code).
def spend_cap(e, H, share_bp):
    # concave: share * e*H/(e+H)
    return (share_bp * ((e*H)//(e+H))) // 10000

# Example A
money=2500; upkeep_cost=3*300+1*100; reserve=3*upkeep_cost
print("upkeep_cost",upkeep_cost,"reserve",reserve)
t1=1*105; t2=2*195
avail=money-t1-t2
print("after t1,t2",avail,"excess",max(0,avail-reserve))
# favour share
low,high,price=800,1000,900
pos=(price-low)*10000//(high-low)
share_sell=2000+(10000-2000)*pos//10000
print("pos",pos,"sell share",share_sell, "qty",max(1,1*share_sell//10000))
# Example B
money=8000
e=money-105-390-reserve
H=10*upkeep_cost
cap=spend_cap(e,H,5000)
print("e",e,"H",H,"e*H//(e+H)",(e*H)//(e+H),"cap",cap)
share_buy=2000+(10000-2000)*(10000-pos)//10000
print("buy share",share_buy)
fuel_buf=3; staple_buf=8
fq=fuel_buf*share_buy//10000; sq=staple_buf*share_buy//10000
print("fuel qty",fq,"staple qty",sq)
cap1=cap-fq*105; print("after fuel",cap1, "staple afford",cap1//310)
sq=min(sq,cap1//310); print("staple qty",sq, "left",cap1-sq*310)
# Example C
w=40; staple_need=120; fuel_need=40; hs=100; hf=40
fed=min(min(hs,staple_need)//3, min(hf,fuel_need)//1)
print("fed",fed,"unfed",w-fed)
# Example D
T=15000; band=500; Thi=T*(10000+band)//10000; ratio=15300
s=(Thi-ratio)*10000//(Thi-T)
k=-(-w*s//10000)
print("Thi",Thi,"s",s,"k",k)
