"""spec-fiscal.md 9.7 and 19 (FIS-A4): a standing bid and ask of a treasury around the
reference price, on a toy market, integer arithmetic. Linear demand D(p) = A - B*p,
supply S_t = S0 + shock_t, the market clears at the price where demand equals supply
less what the treasury buys plus what it releases. Not the library."""
A, B, S0 = 10000, 100, 5000          # clearing price without a shock: (A - S0) / B = 50
REF = 50
BUY_LIM, SELL_LIM = REF * 8000 // 10000, REF * 12000 // 10000     # 40 and 60
Q = 500                              # units a tick
CASH0, STOCK0, HOLD = 150000, 3000, 6000


def lcg(seed):
    x = seed
    while True:
        x = (x * 6364136223846793005 + 1442695040888963407) & ((1 << 64) - 1)
        yield (x >> 33) % 4001 - 2000     # -2000 .. 2000 units, price -20 .. +20


def price(supply, net_take):
    # demand A - B p equals supply - net_take  ->  p = (A - supply + net_take) / B
    return (A - supply + net_take) // B


def simulate(shocks, treasury=True):
    cash, stock = CASH0, STOCK0
    prices, stocks, exhausted_cash, exhausted_stock = [], [], None, None
    for t, sh in enumerate(shocks):
        supply = S0 + sh
        p0 = price(supply, 0)
        take = 0
        if treasury:
            if p0 < BUY_LIM:      # a standing bid at 40: buys while the price is below it
                want = min(Q, B * BUY_LIM - (A - supply), HOLD - stock, cash // BUY_LIM)
                take = max(0, want)
            elif p0 > SELL_LIM:   # a standing ask at 60: sells while the price is above it
                want = min(Q, (A - supply) - B * SELL_LIM, stock)
                take = -max(0, want)
        p = price(supply, take)
        cash -= take * p
        stock += take
        if cash < BUY_LIM and exhausted_cash is None:
            exhausted_cash = t
        if stock == 0 and exhausted_stock is None:
            exhausted_stock = t
        prices.append(p)
        stocks.append(stock)
    return prices, stocks, cash, exhausted_cash, exhausted_stock


def summary(name, prices, lo=0, hi=None):
    ps = prices[lo:hi]
    out = sum(1 for p in ps if p < BUY_LIM or p > SELL_LIM)
    mad = sum(abs(p - REF) for p in ps) / len(ps)
    print(f"{name}: price {min(ps)} to {max(ps)}, mean |p - 50| {mad:.1f}, ticks outside 40..60: {out} of {len(ps)}")


g = lcg(7)
shocks = [next(g) for _ in range(400)]
no_t = simulate(shocks, treasury=False)[0]
wt, stocks, cash, ec, es = simulate(shocks)
print("iid shocks of -2000 to +2000 units (price -20 to +20), 400 ticks, band 40 to 60, Q 500, hold 6000")
summary("no treasury   ", no_t)
summary("with treasury ", wt)
print(f"stock between {min(stocks)} and {max(stocks)}; cash at the end {cash}; cash short from tick {ec}; stock zero from tick {es}")

# a persistent shortage: supply 1500 units below normal for 120 ticks (price 65 untreated)
sh = [0] * 40 + [-1500] * 120 + [0] * 40
no_t = simulate(sh, treasury=False)[0]
wt, st, cash, ec, es = simulate(sh)
summary("shortage, no treasury  ", no_t, 40, 160)
summary("shortage, with treasury", wt, 40, 160)
print(f"stock at ticks 40, 42, 44, 46, 48, 52: {[st[t] for t in (40, 42, 44, 46, 48, 52)]}; stock zero from tick {es}; prices at 41, 44, 47, 52, 100: {[wt[t] for t in (41, 44, 47, 52, 100)]}")
print("ticks of defence = stock / (units released a tick) =", STOCK0 // (Q))

# a persistent glut: supply 1500 units above normal for 120 ticks (price 35 untreated)
sh = [0] * 40 + [1500] * 120 + [0] * 40
no_t = simulate(sh, treasury=False)[0]
wt, st, cash, ec, es = simulate(sh)
summary("glut, no treasury  ", no_t, 40, 160)
summary("glut, with treasury", wt, 40, 160)
print(f"stock at ticks 40, 42, 44, 46, 52, 100: {[st[t] for t in (40, 42, 44, 46, 52, 100)]}; cash at the end {cash}; prices at 41, 44, 47, 100: {[wt[t] for t in (41, 44, 47, 100)]}")
print("the hold cap (6000) ends the purchases at tick", 40 + (HOLD - STOCK0) // 500)
