"""RES model 2: how a scarce endowment is shared in one tick (spec-resources.md 5.3), and what the alternatives cost.

Three rules are compared on the same demand:
  seq   - first come, first served in the rotation order of spec-production.md 5.5.1 (start at tick mod n);
  prop_rot - proportional share, leftovers by largest remainder, ties by the rotation order;
  prop_hash - the spec's rule: proportional share, leftovers by largest remainder, ties by a stateless hash of (tick, agent).
Integer arithmetic throughout. A model of the RULES, not of the library; no number is a claim about any pack."""
M64 = (1 << 64) - 1

def mix(tick, agent):
    """splitmix64 finaliser of x = tick * 0x9E3779B97F4A7C15 + agent (mod 2^64): a stateless total-order key; no random draw,
    no state (spec-resources.md 5.3.3)."""
    z = (tick * 0x9E3779B97F4A7C15 + agent + 0x9E3779B97F4A7C15) & M64
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)

def allocate(avail, reqs, tick, tie):
    """reqs: list of (agent, w, d). Returns {agent: executions granted}. tie in {'hash','rot'}."""
    U = sum(w * d for _, w, d in reqs)
    if U <= avail:
        return {a: w for a, w, d in reqs}
    grant = {}; fr = []; used = 0
    n = len(reqs)
    for pos, (a, w, d) in enumerate(reqs):
        p, phi = divmod(w * avail, U)
        grant[a] = p; used += p * d
        if p < w:
            key = mix(tick, a) if tie == "hash" else (pos - tick) % n
            fr.append((-phi, key, a, d))
    rem = avail - used
    fr.sort()
    for _, _, a, d in fr:
        if d <= rem:
            grant[a] += 1; rem -= d
    return grant

def sequential(avail, reqs, tick):
    n = len(reqs); out = {}
    for i in range(n):
        a, w, d = reqs[(tick + i) % n]
        k = min(w, avail // d); out[a] = k; avail -= k * d
    return out

def run(rule, n, wt, avail, ticks=365):
    agents = list(range(n)); reqs = [(a, wt, 1) for a in agents]
    tot = {a: 0 for a in agents}; run_len = {a: 0 for a in agents}; worst = {a: 0 for a in agents}
    for t in range(ticks):
        if rule == "seq": g = sequential(avail, reqs, t)
        elif rule == "prop_rot": g = allocate(avail, reqs, t, "rot")
        else: g = allocate(avail, reqs, t, "hash")
        for a in agents:
            tot[a] += g[a]
            if g[a] == 0: run_len[a] += 1; worst[a] = max(worst[a], run_len[a])
            else: run_len[a] = 0
    per_hh = [tot[a] / (wt * ticks) for a in agents]
    return min(per_hh), max(per_hh), max(worst.values()), sum(worst.values()) / n

if __name__ == "__main__":
    print("== 1000 agents of weight 1, one execution each wanted per tick, 365 ticks")
    print("   f = granted share of the demand; columns: min and max share granted over a year to one agent; the longest")
    print("   run of ticks in which some agent got nothing (max over agents) and the mean of agents' longest runs")
    print(f"   {'f':>4} {'rule':>10} {'min':>7} {'max':>7} {'longest':>8} {'mean longest':>13}")
    for f_num in (3, 5, 9):
        avail = 1000 * f_num // 10
        for rule in ("seq", "prop_rot", "prop_hash"):
            lo, hi, w, mw = run(rule, 1000, 1, avail)
            print(f"   {f_num/10:>4} {rule:>10} {lo:7.3f} {hi:7.3f} {w:8d} {mw:13.2f}")
    print("\n== weight invariance: 20 agents of weight 50 against 1000 agents of weight 1, the same demand, f = 0.5")
    for rule in ("prop_hash",):
        lo, hi, w, mw = run(rule, 20, 50, 500)
        print(f"   cohorts  {rule}: per-household share min {lo:.3f} max {hi:.3f}")
        lo, hi, w, mw = run(rule, 1000, 1, 500)
        print(f"   units    {rule}: per-household share min {lo:.3f} max {hi:.3f}")
    lo, hi, w, mw = run("seq", 20, 50, 500)
    print(f"   cohorts  seq: per-household share min {lo:.3f} max {hi:.3f}  (the first cohorts in the order take everything)")
    print("\n== mixed draws: 3 agents want 1 execution of draw 5, 4 want 1 of draw 2, 2 want 3 of draw 1; available 14")
    reqs = [(0, 1, 5), (1, 1, 5), (2, 1, 5), (3, 1, 2), (4, 1, 2), (5, 1, 2), (6, 1, 2), (7, 3, 1), (8, 3, 1)]
    g = allocate(14, reqs, 3, "hash")
    print("   demand U =", sum(w * d for _, w, d in reqs), " grants:", g, " units taken:", sum(g[a] * d for a, w, d in reqs))
