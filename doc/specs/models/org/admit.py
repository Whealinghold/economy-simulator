"""models/org/admit.py: checks of spec-institutions.md section 4 (the membership round).

The rule (4.2, 4.3), for one body with a scope of several regions: first the lapse pass
over every household (a member loses the membership if its role is not on the list, its
region is not in the scope, or its region is not the one it was admitted in); then, for
each region of the scope in ascending order, the eligible non-members are admitted in
order of (RoleSince ascending, key of (tick, agent) ascending, agent ID), each whole and
only if its weight fits the free places of that region; one that does not fit is skipped
and the round goes on.

Checked over random populations:
  A1  from a start with no member, the members of a region never exceed the places
      after a round, over several rounds with removals, role changes and moves between
      the regions of the scope (and out of it) in between; every member is eligible and
      in the region it was admitted in at the end of a round;
  A2  the result does not depend on the order of the list the round is given;
  A3  a second round with no change admits nobody and lapses nobody (idempotent);
  A4  the key is the SplitMix64 finaliser of tick * 0x9E3779B97F4A7C15 + agent
      (spec-resources.md 5.3.3), so no draw is taken;
  A5  a completion held until the headroom is at least 1: over random runs of removals,
      arrivals that bypass the headroom (an external agent, a migrant) and entry steps,
      a held trainee completes only with headroom, in queue order, before an entrant of
      the same entry step (step (1) before step (d)), and the households of the role
      never exceed the places because of a completion.

Then the arithmetic of the worked example A of section 13.1. Standard library only.
"""
import copy
import random

M = (1 << 64) - 1


def key(t, a):
    x = (t * 0x9E3779B97F4A7C15 + a) & M
    z = (x + 0x9E3779B97F4A7C15) & M
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M
    return z ^ (z >> 31)


def rnd(agents, tick, places, roles, scope, b=0):
    """agents: dicts {id, w, role, region, since, member, mregion}.
    member is a body index or -1; mregion the region of admission or -1.
    Returns (lapsed_role, lapsed_region, admitted, skipped, waiting)."""
    lapsed_role = lapsed_region = admitted = skipped = 0
    for a in agents:                       # the lapse pass (4.2.1)
        if a["member"] != b:
            continue
        if a["role"] not in roles:
            a["member"], a["mregion"] = -1, -1
            lapsed_role += 1
        elif a["region"] not in scope or a["region"] != a["mregion"]:
            a["member"], a["mregion"] = -1, -1
            lapsed_region += 1
    for r in sorted(scope):                # admission per region (4.3)
        used = sum(a["w"] for a in agents if a["member"] == b and a["mregion"] == r)
        elig = [a for a in agents
                if a["role"] in roles and a["region"] == r and a["member"] != b]
        elig.sort(key=lambda a: (a["since"], key(tick, a["id"]), a["id"]))
        for a in elig:
            if places == 0 or used + a["w"] <= places:
                a["member"], a["mregion"] = b, r
                used += a["w"]
                admitted += 1
            else:
                skipped += 1
    waiting = sum(1 for a in agents
                  if a["role"] in roles and a["region"] in scope and a["member"] != b)
    return lapsed_role, lapsed_region, admitted, skipped, waiting


def members_w(agents, r, b=0):
    return sum(a["w"] for a in agents if a["member"] == b and a["mregion"] == r)


def new_agent(rng, i, nregions, nroles, member=-1):
    return dict(id=i, w=rng.choice([1, 1, 1, 2, 3]), role=rng.randint(0, nroles),
                region=rng.randint(0, nregions), since=rng.randint(0, 20),
                member=member, mregion=-1)


def check_a1_a3(n, rng):
    # A2 and A3 from arbitrary start states (a member of a region may be anywhere)
    for case in range(n):
        N = rng.randint(1, 40)
        places = rng.randint(0, 30)
        roles, scope = {0, 1}, {0, 1}
        agents = []
        for i in range(N):
            a = new_agent(rng, i, 3, 2)
            if rng.random() < 0.5:
                a["member"], a["mregion"] = 0, rng.randint(0, 3)
            agents.append(a)
        t = rng.randint(0, 500)
        b = copy.deepcopy(agents)
        rnd(agents, t, places, roles, scope)
        c = copy.deepcopy(b)
        rng.shuffle(c)
        rnd(c, t, places, roles, scope)
        key_a = sorted((a["id"], a["member"], a["mregion"]) for a in agents)
        key_c = sorted((a["id"], a["member"], a["mregion"]) for a in c)
        assert key_a == key_c, f"A2 case {case}"
        r2 = rnd(agents, t, places, roles, scope)
        assert r2[0] == 0 and r2[1] == 0 and r2[2] == 0, f"A3 case {case}"
    # A1: from a start with nobody a member, rounds with churn in between
    moved_inside = 0
    for case in range(n):
        N = rng.randint(1, 40)
        places = rng.randint(1, 30)
        roles, scope = {0, 1}, {0, 1, 2}
        agents = [new_agent(rng, i, 3, 2) for i in range(N)]   # region 3 is out of scope
        next_id = N
        for t in range(8):
            rnd(agents, t, places, roles, scope)
            for r in scope:
                assert members_w(agents, r) <= places, f"A1 case {case} region {r}"
            for a in agents:
                if a["member"] == 0:
                    assert a["role"] in roles and a["region"] == a["mregion"] \
                        and a["region"] in scope, f"A1 stale member case {case}"
            # churn between rounds: no weight change (SET R3 does not remove, and a weight
            # change is the one thing that can put used above places)
            for a in list(agents):
                x = rng.random()
                if x < 0.05:
                    agents.remove(a)
                elif x < 0.10:
                    a["role"] = rng.randint(0, 2)
                elif x < 0.25:
                    old = a["region"]
                    a["region"] = rng.randint(0, 3)
                    if a["member"] == 0 and old != a["region"] and a["region"] in scope:
                        moved_inside += 1
            if rng.random() < 0.5:
                agents.append(new_agent(rng, next_id, 3, 2))
                next_id += 1
    return moved_inside


def check_a5(n, rng):
    """The held completion (4.5.2): a model of the entry step of one region and role."""
    completions = held_ticks = 0
    for case in range(n):
        places = rng.randint(1, 12)
        H = rng.randint(0, places)                    # a legal start
        held = []                                     # (application tick, id), all served TT
        nid = 0
        for t in range(40):
            # removals and arrivals that bypass the headroom (an external agent, a migrant)
            for _ in range(rng.randint(0, 2)):
                if H > 0 and rng.random() < 0.5:
                    H -= 1
            if rng.random() < 0.15:
                H += 1                                # may exceed the places: legal (SET R3)
            # trainees whose ticks served reached TrainingTicks join the held queue
            for _ in range(rng.randint(0, 2)):
                held.append((t - rng.randint(0, 5), nid))
                nid += 1
            # step (1): completions, in queue order, each only with headroom of 1
            held.sort()
            before = list(held)
            done = []
            for tr in list(held):
                if max(0, places - H) >= 1:
                    H += 1
                    held.remove(tr)
                    done.append(tr)
                    assert H <= places, f"A5 case {case}: a completion exceeded the places"
            assert done == before[:len(done)], f"A5 case {case}: not in queue order"
            if held:
                assert max(0, places - H) == 0, f"A5 case {case}: held with headroom"
                held_ticks += 1
            completions += len(done)
            # step (d): entrants take what headroom is left, after the completions
            room = max(0, places - H)
            H += min(room, rng.randint(0, 3))
    return completions, held_ticks


def run(n, seed):
    rng = random.Random(seed)
    moved = check_a1_a3(n, rng)
    assert key(0, 0) == 0xE220A8397B1DCDAF, "A4: first SplitMix64 output of seed 0"
    comp, held_ticks = check_a5(n, rng)
    print(f"{n} random populations over three regions, {moved} moves of a member inside the scope: A1 to A3 held")
    print(f"{n} random runs of the held completion, {comp} completions, {held_ticks} ticks with a held trainee: A5 held")
    print("A4 held (the key is the SplitMix64 finaliser)")


def example_a():
    """Example A of 13.1: a craft body, 12 places in one region, a household per agent."""
    roles, scope = {0}, {0}
    agents = [dict(id=100 + i, w=1, role=0, region=0, since=0, member=-1, mregion=-1)
              for i in range(12)]
    agents += [dict(id=112, w=1, role=0, region=0, since=5, member=-1, mregion=-1),
               dict(id=113, w=1, role=0, region=0, since=9, member=-1, mregion=-1)]
    out = rnd(agents, 0, 12, roles, scope)
    mem = sorted(a["id"] for a in agents if a["member"] == 0)
    wait = sorted(a["id"] for a in agents if a["member"] != 0)
    print("example A, tick 0: lapsed %d admitted %d skipped %d waiting %d" % (out[0] + out[1], out[2], out[3], out[4]))
    print("  members:", mem)
    print("  waiting:", wait)
    agents = [a for a in agents if a["id"] != 104]
    out = rnd(agents, 4, 12, roles, scope)
    wait = sorted(a["id"] for a in agents if a["member"] != 0)
    print("tick 4, one member removed: lapsed %d admitted %d skipped %d waiting %d" % (out[0] + out[1], out[2], out[3], out[4]))
    print("  admitted is the senior waiting household; still waiting:", wait)
    households = sum(a["w"] for a in agents)
    print("  households of the role %d, places 12, headroom %d" % (households, max(0, 12 - households)))
    # a move inside a scope of two regions: the member lapses and is a candidate at once
    agents = [dict(id=1, w=1, role=0, region=0, since=0, member=-1, mregion=-1),
              dict(id=2, w=1, role=0, region=1, since=0, member=-1, mregion=-1)]
    rnd(agents, 0, 1, {0}, {0, 1})
    agents[0]["region"] = 1                           # household 1 moves into region 1, which is full
    out = rnd(agents, 1, 1, {0}, {0, 1})
    print("move inside the scope: lapsed region %d, admitted %d, skipped %d; member of region 1: %s"
          % (out[1], out[2], out[3], [a["id"] for a in agents if a["member"] == 0 and a["mregion"] == 1]))


if __name__ == "__main__":
    run(3000, 4242)
    example_a()
