# Reference: bazaarBot

Status: reference, not a spec. It records what the bazaarBot source does, so
that the specs for the port can cite it. It defines no requirements and has no
area code. It does not override the rules in `doc/rules/`.

## 1. Provenance

- **Source:** <https://github.com/larsiusprime/bazaarBot>, read at commit
  `f93df8b` (merge of pull request 15, 10 July 2017). Language Haxe, built with
  OpenFL and Lime, with `hscript` for scripted logic. Version 0.0.1, described
  as "pre-alpha" in `haxelib.json`.
- **Licence:** MIT, copyright 2013 Lars A. Doucet. The port re-implements the
  algorithms and copies no code. If any substantial code is ever copied, the
  copyright and permission notice must be kept.
- **Origin of the algorithm:** the README says it is based on "Emergent
  Economies for Role Playing Games" by Jonathon Doran and Ian Parberry
  (LARC-2010-03, <http://larc.unt.edu/techreports/LARC-2010-03.pdf>). **That
  paper has not been read.** Where this reference says what the code does, it
  does not say what the paper intended. Comparing the two is an open task
  (section 12).
- **Method:** every statement below was taken from reading the source files. It
  was not run. There are no tests in the repository (no file with "test" or
  "spec" in its name).

## 2. Architecture

| File | Role |
|------|------|
| `bazaarbot/Market.hx` | One market: the round loop, clearing, history, bankruptcy signal, reporting |
| `bazaarbot/Economy.hx` | A list of markets, `simulate(rounds)`, and a bankruptcy hook the subclass overrides |
| `bazaarbot/agent/BasicAgent.hx` | Agent state: money, inventory, price beliefs, observed trade prices; quantity helpers |
| `bazaarbot/Agent.hx` | The Doran and Parberry agent: offer generation and the price-belief update, with the constants |
| `bazaarbot/agent/Inventory.hx` | Stock, ideal stock and unit sizes per good, and capacity |
| `bazaarbot/agent/Logic.hx` | Base class for what an agent does each round; `_produce` and `_consume` helpers |
| `bazaarbot/agent/LogicScript.hx`, `LogicHScript.hx` | Optional hscript-driven logic (not analysed further) |
| `bazaarbot/Offer.hx`, `utils/TradeBook.hx` | An offer, and the bid and ask lists per good |
| `bazaarbot/MarketData.hx` | Builds a market from a JSON settings file |
| `bazaarbot/utils/History.hx`, `HistoryLog.hx` | Per-good series of price, asks, bids, trades, and per-class profit |
| `bazaarbot/utils/Quick.hx` | Helpers: shuffle, sorts, `positionInRange`, `randomRange`, averages |
| `examples/doran_and_parberry/` | The example: five roles, `settings.json`, one `Logic*` class per role, and the replacement policy |

The library does not separate the engine from OpenFL: `Market.hx` imports
`openfl.Assets`, `flash.errors.Error` and `hscript`.

## 3. Data model

All quantities and prices are `Float`.

- **Good:** an `id` and a `size` (inventory space per unit).
- **Offer:** `good`, `units`, `unit_price`, `agent_id`.
- **Inventory:** per good, the amount held (`_stuff`), the ideal amount
  (`_ideal`) and the unit size (`_sizes`), plus `maxSize` for the whole
  inventory. `surplus(good)` is how much is above ideal, or 0. `shortage(good)`
  is how much is below ideal, or 0. `getEmptySpace()` is `maxSize` minus the sum
  of amount times size. `change(good, delta)` adds `delta` and floors the result
  at 0.
- **Agent:** `id`, `className`, `money`, `moneyLastRound`, an inventory, a
  `lookBack` (default 15), a price belief per good (a `[low, high]` pair), and
  an array of observed trade prices per good.
- **Market:** the goods, the agents (an array indexed by `id`), the trade book,
  the history, and a map of agent classes.
- **Settings (JSON):** `start_conditions.agents` (count per class), `goods`
  (`id`, `size`), and `agents` (per class: `money`, and an `inventory` with
  `start`, `ideal` and `max_size`, plus a `logic` name).

## 4. The round

`Market.simulate(rounds)` runs this once per round:

1. For each agent, in array order:
   1. `moneyLastRound = money`.
   2. `logic.perform(agent, market)`: the agent produces and consumes (section
      10). Production is instant.
   3. For each good: `generateOffers`, which puts at most one bid or ask for
      that good on the book (section 5).
2. For each good, in the order of the goods list: `resolveOffers` (section 6).
3. For each agent: if `money <= 0`, dispatch the bankruptcy signal. The example
   handler replaces the agent (section 9).
4. Increment the round counter, which is not otherwise used.

All offers of a round are placed before any is cleared, and all agents see the
previous round's history while placing them. Goods produced in step 1.2 can be
offered in the same round.

## 5. Order generation

For each good, `generateOffers` does one of two things:

- If `surplus >= 1`: place **one ask**, with a limit of 1 unit.
- Otherwise, if `shortage > 0` and the free space is at least one unit's size,
  compute a limit and place **one bid**:
  - if `shortage * unit_size <= space`, the limit is `shortage`;
  - otherwise the limit is `floor(space / shortage)`.

An agent never both bids and asks for the same good in a round.

**Price of an offer:** a uniformly random number in the agent's belief range for
that good (`Quick.randomRange`, from `Math.random`). It is independent of the
agent's cost, stock or budget.

**Quantity of a bid:** `min(limit, max(1, round(favorability * shortage)))`,
with `favorability = 1 - positionInRange(mean, low, high)`.
**Quantity of an ask:** `max(limit, max(1, round(favorability * surplus)))`,
with `favorability = positionInRange(mean, low, high)`.

Here `mean` is the market's average price over the last `lookBack` (15)
entries, and `[low, high]` is the minimum and maximum of the agent's
`observedTradingRange` for the good. `positionInRange` clamps its result to
`[0, 1]`. So an agent sells more when the market price is near the top of what
it has seen, and buys more when it is near the bottom.

There is **no budget check**: a bid is placed whatever the agent can pay.

## 6. Clearing

`resolveOffers(good)`:

1. Shuffle the bids and the asks (`Quick.shuffle`).
2. Sort bids by price, highest first, and asks by price, lowest first. The
   sort is the platform's `Array.sort`, whose stability the language does not
   guarantee.
3. Add up the total **units** on each side: `numBids`, `numAsks`.
4. While both lists are non-empty, take the top bid and the top ask:
   - `quantity = min(ask.units, bid.units)`;
   - `price = (ask.unit_price + bid.unit_price) / 2`;
   - move `quantity` units from seller to buyer and `quantity * price` money
     from buyer to seller;
   - call `updatePriceModel(success)` on both agents, using that `price`;
   - remove an exhausted offer from its list.
5. Every offer left in either list triggers `updatePriceModel(failure)` for its
   agent.
6. Record in the history: asks (total units), bids (total units), trades (units
   traded) and the price (`moneyTraded / unitsTraded`, or the previous price if
   nothing traded).
7. Compute each class's average profit and add it to the history (section 8).

**Notes:**

- The loop has **no check that the top bid is at least the top ask**. It
  matches until one list is empty, whatever the prices.
- The trade price is the unweighted average of the two prices, not the price of
  the earlier or the better offer.
- There is no check that the buyer can afford the trade or that the seller has
  the goods.
- A "failsafe" counter prints a trace after 1 000 loop iterations without an
  offer being removed. It does not break the loop.

## 7. The price-belief update

`Agent.updatePriceModel(market, act, good, success, unitPrice)` is called once
per **matched trade** and once per **unmatched offer**. So one agent can get
several updates for one good in a round, including a success followed by a
failure.

Constants: `SIGNIFICANT` 0.25, `SIG_IMBALANCE` 0.33, `LOW_INVENTORY` 0.1,
`HIGH_INVENTORY` 2.0, `MIN_PRICE` 0.01, `wobble` 0.05.

Let `pub` be the market's price of the previous round
(`getAverageHistoricalPrice(good, 1)`), `mean = (low + high) / 2`, and
`delta = mean - pub`.

**On success** (the trade price is appended to the agent's observed prices):

- If it was a buy and `delta > 0.25` (the agent was overpaying), shift both
  ends by `-delta / 2`.
- If it was a sell and `delta < -0.25` (the agent was underselling), shift both
  ends by `-delta / 2`.
- Then narrow the range: `low += wobble * mean`, `high -= wobble * mean`.

**On failure:**

- Shift both ends by `-delta / 2`, towards the market price.
- Then, with `stocks` the amount held and `ideal` its ideal amount:
  - if it was a buy and `stocks < 0.1 * ideal`: `wobble` doubles (the agent
    bids more liberally);
  - else if it was a sell and `stocks > 2.0 * ideal`: `wobble` doubles;
  - otherwise look at last round's total units offered:
    `s = (asks - bids) / (asks + bids)`. If `|s| > 0.33`, compute
    `newMean = pub * (1 - s)`, `delta = mean - newMean`, and shift both ends by
    `-delta / 2` again.
- Then widen the range: `low -= wobble * mean`, `high += wobble * mean`.

**Last:** if `low < 0.01`, set `low = 0.01`; else if `high < 0.01`, set
`high = 0.01`.

## 8. History and statistics

- **Seeding:** at market creation each good gets one entry of 1.0 in the price,
  asks, bids and trades series. So the first round's "market price" is 1.0.
- **`average(name, range)`:** the mean of the last `range` entries, or of all
  entries if there are fewer. With no entries it computes 0 / 0.
- **Initial beliefs:** an agent created at `init` takes the average price over
  its `lookBack` and sets its belief to `[0.5 * price, 1.5 * price]`. Its
  observed-price array starts with those two values.
- **Profit:** `money - moneyLastRound`. The class profit is computed and
  appended **once per good per round**, inside `resolveOffers`, not once per
  round. So the series has five entries per class per round in the example, and
  they are cumulative: later goods see more of the round's money movements.
  `getMostProfitableAgentClass(10)` averages the last 10 entries.
- **Class grouping:** `resolveOffers` sorts the agents by class name to group
  them, then sorts them back by id. `_agents[agent_id]` is used as an index
  throughout, so the array must be in id order outside those two sorts.
- **Queries used by the example:**
  - `getHottestGood(minimum = 1.5, range = 10)`: the good with the highest
    ratio of average bids to average asks, if above the minimum. If average asks
    are 0 and bids are above 0 it uses 0.5 as the asks.
  - `getCheapestGood(range, exclude)` and `getDearestGood`: by average price.

## 9. Bankruptcy and replacement

After clearing, every agent with `money <= 0` triggers `onBankruptcy`. The
example's handler `replaceAgent`:

1. Picks the most profitable class (above).
2. If `getHottestGood()` returns a good, replaces that choice with the class
   that makes the good. The mapping is a hard-coded `switch` on the strings
   "food", "wood", "ore", "metal" and "tools".
3. Creates a new agent of that class with the class's starting money (100) and
   starting inventory, gives it the old agent's id, and destroys the old agent
   along with its inventory.
4. Initialises the new agent's beliefs from the current average price.

A new agent always appears, so the population never changes and an economy
cannot die out.

## 10. The example chain

`settings.json`: 10 agents each of farmer, miner, woodcutter, refiner and
blacksmith (50 in all). Every agent starts with money 100 and `max_size` 20. The
goods are food (size 0.5, given as the string "0.5"), wood, ore, metal and tools
(size 1.0 each).

| Role | Starting stock | Ideal stock |
|------|----------------|-------------|
| farmer | food 0, tools 1, wood 0 | food 0, tools 1, wood 3 |
| miner | food 1, tools 1, ore 0 | food 3, tools 1, ore 0 |
| woodcutter | food 1, tools 1, wood 0 | food 3, tools 1, wood 0 |
| refiner | food 1, tools 1, metal 0, ore 0 | food 3, tools 1, metal 0, ore 5 |
| blacksmith | food 1, tools 0, metal 0, ore 0 | food 3, tools 1, metal 5, ore 0 |

What each role does each round (`jobs/Logic*.hx`). "Tools break" means 1 tool is
consumed with probability 0.1:

| Role | Needs | With tools | Without tools |
|------|-------|-----------|---------------|
| farmer | wood >= 1 | +4 food, -1 wood, tools break | +2 food, -1 wood |
| miner | food >= 1 | +4 ore, -1 food, tools break | +2 ore, -1 food |
| woodcutter | food >= 1 | +2 wood, -1 food, tools break | +1 wood, -1 food |
| refiner | food >= 1 and ore >= 1 | all ore to metal, -1 food, tools break | up to 2 ore to metal, -1 food |
| blacksmith | food >= 1 and metal >= 1 | all metal to tools, **no food consumed** | (same, tools are not needed) |

If a role's needs are not met it is **fined 2 money** (the money is destroyed).
A role other than the farmer that is also out of food and has a full inventory
then calls `makeRoomFor`: it discards 2 units of the cheapest good (by the
10-round average price), excluding food.

Only goods whose key appears in an agent's starting stock can ever be bid on
(section 11, item 5). The farmer does not consume food.

## 11. Defects and quirks

Each item gives the source and the effect. The "Port" column says what the
planned spec must decide. Items marked **confirmed** were checked in the code.

| # | Where | Behaviour | Port |
|---|-------|-----------|------|
| 1 | `Market.resolveOffers` | No bid-at-least-ask check: trades happen between a bid below the ask, at the average price. **Confirmed** | Decided: require crossing (`spec-clearing.md` sections 4.3 and 10) |
| 2 | `Market.resolveOffers`, `transferMoney` | No budget or stock check; money can go negative inside a round. **Confirmed** | Decided: every order is backed by a reservation at admission (`spec-clearing.md` section 3). Order sizing stays with `spec-agents.md` |
| 3 | `Quick.shuffle` | The loop skips the final swap (`if (ii > 1)` should be `>= 1`), so for two offers nothing is shuffled. **Confirmed** | Not ported. A seeded per-order tie-break key (`spec-clearing.md` section 4.2) |
| 4 | `Market.resolveOffers` | Sorting is unstable, so equal prices tie in an unspecified order. **Confirmed** | Decided: a total order ending in the input position (`spec-clearing.md` section 4.1) |
| 5 | `Inventory.shortage`, `surplus` | `shortage` returns 0 for a good not in the agent's starting stock, so it never bids for it. `surplus` reads a missing ideal as null. **Confirmed** | Define a missing ideal as 0 and an explicit tradable set |
| 6 | `Inventory.change` | The result is floored at 0, so consuming more than is held succeeds and the shortfall is silently ignored. **Confirmed** | Reject over-consumption |
| 7 | `BasicAgent.observeTradingRange` | `lookBack` is not applied: the range is the all-time minimum and maximum of the agent's own trades, and the array grows without limit. **Confirmed** | Decide a window. Check the paper |
| 8 | `Agent.generateOffers` | When space is short, the limit is `floor(space / shortage)`. It is dimensionally wrong: it should be `space / unit_size`. **Confirmed** | Use the correct formula |
| 9 | `Agent.updatePriceModel` | `SIGNIFICANT` (0.25) is an absolute price difference, so it depends on the price scale | Use basis points of the mean |
| 10 | `Agent.updatePriceModel` | The final clamp uses `else if`, so if both ends are below 0.01 only `low` is raised. The range can also invert (`low > high`) after repeated narrowing. **Confirmed** | Define the range invariant |
| 11 | `Agent.updatePriceModel` | Called per match, so an agent with a large order updates several times in a round, and can get a success then a failure for the same good | Decide per match or per round |
| 12 | `Market.resolveOffers` | Class profit is appended once per good, not once per round, and is cumulative. **Confirmed** | Record one value per round |
| 13 | `Market.simulate` | Replacement creates 100 money and the starting stock from nothing, and destroys the old agent's stock. The idle fine destroys 2 money. Money is not conserved | `spec-estates.md` and `spec-spawner.md` |
| 14 | `getHottestGood`, `getAgentClassThatMakesMost` | Hard-coded commodity names. Profit history is empty at first, so its average is NaN and the best class can be the empty string. **Confirmed** | Data, not code |
| 15 | `TradeBook.ask` | Checks the bids map before adding to asks. Works only because both maps are registered together. **Confirmed** | Do not port |
| 16 | `Inventory.fromData` | Calls `setIdeal` inside the loop, with growing arrays | Do not port |
| 17 | `LogicBlacksmith` | Requires food but does not consume it, unlike the other roles. **Confirmed** | Make intentional in the chain data |
| 18 | `settings.json` | Good sizes are strings ("0.5"), relying on dynamic typing | Typed config |
| 19 | `LogicHScript.hx` | Calls `getCheapestCommodity`, which does not exist (the method is `getCheapestGood`). **Confirmed**: it is stale | Not relevant |
| 20 | All of it | `Math.random()` is unseeded, so runs cannot be reproduced | Seeded world source |
| 21 | All of it | Production is instant; one market only; no travel or time | The other specs |

## 12. Corrections and open tasks

**Corrections to earlier statements in this project:**

- An earlier summary called `Agent.createAsk` an "inverted clamp". Re-reading
  the code, `max(ideal, limit)` matches its comment, and it cannot oversell
  because `generateOffers` asks only when `surplus >= 1` and passes a limit
  of 1. It is not a defect. Item 8 above (the bid limit) is.
- An earlier summary said a partial fill counted only as a success. In fact an
  agent can receive both a success update (for the matched part) and a failure
  update (for the unmatched remainder) in the same round (section 7).

**Open tasks:**

1. Read the Doran and Parberry paper and compare it with section 7 and
   section 6, in particular item 1 (the missing crossing check) and item 7 (the
   lookback window).
2. Run the example to see the dynamics the port should reproduce or improve:
   price series, bankruptcies and class populations. The source was not run.
3. Decide which quirks to keep as a "faithful mode" for comparison with the
   original and which to fix.

## 13. Where each piece goes in the port

| bazaarBot piece | economy-sim home | Planned spec |
|-----------------|------------------|--------------|
| Round loop, order of phases | `World.Step` | `spec-world.md` |
| Double auction (`resolveOffers`) | `Clearing` (default `AverageClearing`) over `internal/clearing` | `spec-clearing.md` |
| Price beliefs (`updatePriceModel`) | `internal/beliefs` | `spec-beliefs.md` |
| Order generation, favorability, ideal stock | `Strategy` (default in the root) | `spec-agents.md` |
| Inventory, capacity, sizes | `internal/ledger` | `spec-ledger.md` |
| `Logic*` classes | recipes and roles as data | `spec-production.md` |
| Bankruptcy and replacement | `Spawner` | `spec-spawner.md` |
| `History`, `getMarketReport` | events and `Snapshot` | `spec-observability.md` |
| `settings.json` | `Config` and `config/` | `spec-config.md` |
