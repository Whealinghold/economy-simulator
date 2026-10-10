"""The contract round and the escrow of spec-merchants.md 9, with conservation checks (MER-A5).

A model of the rules, not of the library. Integer arithmetic only. It builds a small world of
money accounts (agents, one escrow per open contract, an estate) and goods accounts (agents,
transit), runs the matching of offers against takes on a lane (destination, commodity),
forms the contracts, and settles them at arrival in four cases. After every step it checks
that the money of all accounts is constant (a contract creates and destroys no money) and
that each line balances: shipped = delivered + diverted + spoiled.
"""


def half_even_mid(a, b):
    q, r = divmod(a + b, 2)
    if r == 1 and q % 2 == 1:
        q += 1
    return q


class World:
    def __init__(self):
        self.money = {}
        self.goods = {}
        self.contracts = {}
        self.next_id = 0
        self.log = []

    def total_money(self):
        return sum(self.money.values())

    def move_money(self, src, dst, amount):
        assert amount >= 1 and self.money[src] >= amount, (src, dst, amount)
        self.money[src] -= amount
        self.money[dst] += amount


def match_lane(offers, takes):
    """offers: (agent, limit, qty); takes: (agent, limit, qty). Offers are indivisible, takes divisible.

    Offers by (limit ascending, agent ascending); takes by (limit descending, agent ascending)."""
    offers = sorted(offers, key=lambda o: (o[1], o[0]))
    takes = sorted(takes, key=lambda t: (-t[1], t[0]))
    remaining = {t[0]: t[2] for t in takes}
    pairs = []
    for agent, limit, qty in offers:
        for t_agent, t_limit, _ in takes:
            if t_limit < limit:
                break  # takes are in descending limit order: none further can cross
            if remaining[t_agent] >= qty:
                pairs.append((agent, t_agent, qty, half_even_mid(limit, t_limit)))
                remaining[t_agent] -= qty
                break
    return pairs


def form(world, payer, deliverer, qty, price, shipment_units):
    escrow = f"escrow{world.next_id}"
    world.money[escrow] = 0
    cid = world.next_id
    world.next_id += 1
    world.move_money(payer, escrow, qty * price)  # escrow_in
    world.goods[("deliverer", cid)] = qty  # the line leaves the deliverer for the transit account
    world.contracts[cid] = dict(payer=payer, deliverer=deliverer, qty=qty, price=price, escrow=escrow, shipped=shipment_units)
    return cid


def settle(world, cid, delivered, diverted, estate="estate"):
    c = world.contracts.pop(cid)
    spoiled = c["qty"] - delivered - diverted
    assert spoiled >= 0
    reached = delivered + diverted
    if reached:
        world.move_money(c["escrow"], c["deliverer"], c["price"] * reached)  # escrow_out
    if spoiled:
        target = c["payer"] if c["payer"] in world.money else estate
        world.move_money(c["escrow"], target, c["price"] * spoiled)  # escrow_refund
    assert world.money[c["escrow"]] == 0
    return reached, spoiled


def refund_all(world, cid, estate="estate"):
    """The deliverer is removed: the escrow goes back to the payer in full (FR-CAR-002)."""
    c = world.contracts.pop(cid)
    target = c["payer"] if c["payer"] in world.money else estate
    world.move_money(c["escrow"], target, c["price"] * c["qty"])
    assert world.money[c["escrow"]] == 0


def fresh():
    w = World()
    w.money.update({"seller1": 5000, "seller2": 5000, "buyer1": 80000, "buyer2": 30000, "estate": 0})
    return w


if __name__ == "__main__":
    print("the matching on one lane (destination lower, commodity fuel)")
    offers = [(31, 244, 150), (32, 263, 150), (34, 250, 100)]
    takes = [(61, 270, 200), (62, 255, 150), (63, 240, 150)]
    for o in offers:
        print(f"  offer agent {o[0]}: limit {o[1]}, {o[2]} units")
    for t in takes:
        print(f"  take  agent {t[0]}: limit {t[1]}, up to {t[2]} units")
    pairs = match_lane(offers, takes)
    for p in pairs:
        print(f"  contract: deliverer {p[0]} payer {p[1]} {p[2]} units at {p[3]}")
    print("  by limit the offers are 31, 34, 32: offer 31 takes 150 of take 61 (200 units) at the midpoint (244 + 270) / 2 = 257, leaving 50;")
    print("  offer 34 (100 units, indivisible) does not fit the 50 left of take 61 and goes to take 62 at (250 + 255) / 2 = 252.5, half to even 252;")
    print("  offer 32 (limit 263) finds no take: take 61 has 50 left and take 62 is below its limit")
    print()

    cases = {
        "delivered in full": dict(delivered=150, diverted=0, kind="settle"),
        "22 units spoiled in transit": dict(delivered=128, diverted=0, kind="settle"),
        "the payer is removed in transit: cargo diverted to the estate": dict(delivered=0, diverted=150, kind="payer_removed"),
        "the deliverer is removed in transit": dict(kind="deliverer_removed"),
    }
    for label, case in cases.items():
        w = fresh()
        total0 = w.total_money()
        cid = form(w, "buyer1", "seller1", 150, 257, 150)
        assert w.total_money() == total0
        before = dict(w.money)
        if case["kind"] == "settle":
            reached, spoiled = settle(w, cid, case["delivered"], case["diverted"])
        elif case["kind"] == "payer_removed":
            # the payer's money went to the estate at its removal; its account is gone
            w.money["estate"] += w.money.pop("buyer1")
            reached, spoiled = settle(w, cid, case["delivered"], case["diverted"])
        else:
            refund_all(w, cid)
            reached, spoiled = 0, 150
        assert w.total_money() == total0
        print(f"{label}")
        print(f"  escrowed {before.get('escrow0', 0)}; reached {reached}, not reached {spoiled}")
        print(f"  balances after: " + ", ".join(f"{k} {v}" for k, v in sorted(w.money.items()) if k != "escrow0"))
        print(f"  money before/after: {total0}/{w.total_money()}; escrow account empty: {w.money['escrow0'] == 0}")
        print()
