# Overview: the approach

A short summary of what this library is, how it is shaped and why. It is the
document to read first after the rules files that `AGENTS.md` requires. Detail lives in the specs (`specs/README.md`), the
plan in [roadmap.md](roadmap.md), and the reasons for the main decisions in
[adr/](adr/README.md). Where this file and one of those disagree, they win and
this file is corrected.

## 1. What it is

`economy-sim` is a Go library that simulates a market economy at the level of a
realm: several connected regions, each with its own market, not one market. It
is for games and simulations that want a plausible, living economy: prices that
respond to events, regions that specialise, shortages and famines that matter,
and a player who can disturb it. It is not for forecasting a real economy.

It is inspired by bazaarBot (Doran and Parberry, "Emergent Economies for Role
Playing Games"): agents with price beliefs trade in a double auction, and
supply chains emerge from recipes. We keep that idea, fix its defects and
extend it with regions, time delays, trade routes, carriers, perishability,
population and migration. We do not aim to reproduce it exactly (`roadmap.md`,
Decisions).

## 2. The picture

The realm: regions joined by trade and movement.

```
                 +---------------------------------------------+
                 | OUTSIDE WORLD  (a boundary account)         |
                 | buys, sells, pays, takes at set terms       |
                 +--------^--------------------+---------------+
                          | in (faucets)       | out (sinks)
+-------------------------+--------------------v--------------------------------+
| REALM  (polity: borders, laws, taxes, tolls, treasury)                        |
|                                                                               |
|  +--------------+     routes: distance, travel time, risk     +-------------+ |
|  |  REGION A    |<========= merchants and carriers ==========>|  REGION B   | |
|  |  settlements |   goods in transit (they spoil, get lost)   |  settlements| |
|  |  market      |<- - - -  rumours: late, approximate  - - - >|  market     | |
|  |  people      |                                             |  people     | |
|  +------+-------+                                             +------+------+ |
|         |                  migration (people move)                   |        |
|         +<-----------------------------------------------------------+        |
+-------------------------------------------------------------------------------+
```

Regions touch only through merchants, rumours and migration. There is no global
market. Money and goods cross the edge of the economy at the outside world and
nowhere else.

Inside one region, goods run down the page and money runs up.

```
 +-----------------------------+      +-----------------------------+
 | NATURE                      |      | CAPITAL                     |
 | sites, yield, depletion,    |      | tools, buildings, roads     |
 | seasons                     |      | (they wear out)             |
 +--------------+--------------+      +--------------+--------------+
                | raw materials                      | tools, buildings
                v                                    v
 +--------------------------------------------------------------------+
 | PRODUCTION   recipes, time, skills, hired labour, home output      |
 +-------+-------------------------------------------^----------------+
         | goods                                       | inputs, labour
         v                                             |
 +--------------------------------------------------------------------+
 | MARKETS   one per commodity per region                             |
 |   orders -> clearing -> trades -> price -> agents' price beliefs   |
 +-------+-------------------------------------------^----------------+
         | goods                                       | money
         v                                             |
 +--------------------------------------------------------------------+
 | HOUSEHOLDS (agents)                                                |
 |   earn -> pay upkeep, buy inputs and wants -> save                 |
 |   short of upkeep: home output, relief, then starve                |
 |   born, die, migrate                                               |
 +---------------------------------+----------------------------------+
                                   | savings -> investment -> CAPITAL
                                   '--> (and credit, if enabled)
```

How the pieces are layered:

```
  CONSUMER'S CODE:  Strategy   Clearing   Spawner   Policy   Observer
        (any of these can be replaced; defaults ship with the library)
                                   |
 +---------------------------------v----------------------------------------+
 | MODULES (optional, off unless configured)                                |
 |  credit  tax  hired labour  technology  risk  institutions  land  ...    |
 +---------------------------------+----------------------------------------+
                                   |
 +---------------------------------v----------------------------------------+
 | CORE: markets, agents, production, population, regions, trade            |
 +---------------------------------+----------------------------------------+
                                   |
 +---------------------------------v----------------------------------------+
 | LEDGER: accounts, balance sheets, reason-coded transfers, conservation   |
 | boundary accounts are the only places money and goods may enter or leave |
 +---------------------------------+----------------------------------------+
                                   | read only
 +---------------------------------v----------------------------------------+
 | MEASURES: price indices, production, destruction, inequality, welfare    |
 | (watched, compared with expected properties, never fed back to agents)   |
 +--------------------------------------------------------------------------+
```

## 3. Principles

1. **Supply chains are data.** Commodities, recipes and roles come from
   configuration. No code branches on a specific one (`AGENTS.md`).
2. **Setting-agnostic.** The library is for fantasy, science fiction or
   anything else. Names, currencies and content live in configuration or
   examples, never in the core. Money is an `int64` minor unit; display names
   are an optional formatter (`specs/spec-currency.md` section 4).
3. **Deterministic.** The same seed, configuration and calls give the same
   result on every platform. No wall-clock time, global randomness or floating
   point in anything that affects results.
4. **Conserved.** Money and goods change only through reason-coded transfers.
   Boundary accounts are the only places either may enter or leave, so every
   faucet and sink is named and countable.
5. **Macro is measured, never set.** Aggregates (price levels, inequality,
   output) are computed and compared with expected properties, never fed back
   into agents. Where a macro item cannot emerge, a structural piece provides
   it, as a module (`adr/0001-macro-economy-as-replaceable-layers.md`).
6. **Replaceable.** A consumer can tweak parameters, adjust which modules run,
   or replace a whole layer through an extension point. Defaults are stable
   constants, claimed only once the acceptance checks exist.
7. **Stable by design.** Stability is a requirement, not a hope. Every feedback
   loop is named, damped and tested (`roadmap.md`, "Stability").
8. **A library, not an application.** No server, transport or storage in the
   core. Observability is optional and a no-op by default.
9. **Persistable.** The library never stores anything. All state is plain data
   owned by the world, so a consumer can save a complete checkpoint and resume a
   run exactly (`adr/0002-persistence-by-checkpoint.md`).

## 4. How a consumer uses it

The consumer builds one `World`, keeps it for the life of the simulation and
drives it forward. The library is a stateful object that is stepped, not a
function called per request. The sketch is illustrative: the names in it come
from the rules (`rules/architecture.md` 6.1, 6.2, 12.3), but the tick and
query specs (WLD, INT) are not written.

```go
cfg := economysim.Config{ /* commodities, recipes, roles, regions */ }
w, err := economysim.NewWorld(cfg,
	economysim.WithSeed(42),
	economysim.WithLogger(logger),   // optional; no-op by default
	economysim.WithObserver(obs),    // optional; may be given more than once
)

for gameIsRunning {
	w.Step()                         // advance one tick
}
err = w.Run(ctx, 365)                // or many ticks; checks ctx between ticks

snap := w.Snapshot()                 // a deep copy of plain data
```

- **The world owns the state.** Agents, inventories, the ledger, price beliefs,
  goods in transit and the random source are all inside the `World`. The
  consumer holds the world and nothing else. It does not create agents,
  markets or orders: configuration defines commodities, recipes, roles and
  regions, the world spawns the agents, and they trade with each other.
- **Configuration is plain data.** `config/` can load it from a file, or the
  consumer can build the structs in Go.
- **Time moves only when the consumer says so.** `Step` and `Run` are
  synchronous, and the library starts no goroutines.
- **One caller at a time.** A `World` is not safe for concurrent use. A game
  loop serialises access naturally; a server puts a lock or an owning
  goroutine around it.
- **Reads are pull or push.** `Snapshot` (and the planned queries) return
  deep copies that can cross goroutines. An `Observer` receives events
  (`TradeExecuted`, `MarketCleared`, `AgentBankrupted`, ...) synchronously
  during `Step`, for metrics, tracing or a UI feed. Logs go to the injected
  `*slog.Logger`. Both default to doing nothing.
- **Influence is through extension points and shocks.** The consumer can
  replace `Strategy`, `Clearing`, `Spawner` or `Observer`, and (planned,
  INT) inject shocks such as a failed harvest or a closed road, or act as a
  player.
- **Persistence belongs to the consumer.** The library produces a checkpoint
  of the complete state as plain data with one canonical encoding. The consumer
  stores it and restores a world from it, and a resumed run is bit-identical to
  an uninterrupted one (`adr/0002-persistence-by-checkpoint.md`, proposed).

Two points are open and affect this shape: whether the consumer reads the
world only through `Snapshot` or also through targeted queries, and whether a
player is an ordinary agent or sits outside the agent model and calls in. Both
are decided in INT and bear on AGT and LED, so settle them before AGT.

## 5. What we can and cannot model

| We can model | With caution (unproven until it runs) | We do not model |
|--------------|---------------------------------------|-----------------|
| Price formation through bids and asks; production chains; perishable goods; trade between regions with delay, capacity and storage; starvation, migration and regional collapse; conserved money and goods; emergent wages | Macro behaviour emerging from the micro rules; the stability of the whole; skill premiums from tools and traineeships; saving, investment and a labour market | Forecasting a real economy; individual psychology, culture, politics or war as endogenous causes (they enter as shocks or policy); real innovation; complex finance such as stock markets |

## 6. Where we are

- **Status:** design only. There is no code. The documents are drafts, and none
  of the behaviour has been run or validated.
- **Written:** the rules (`rules/`), twenty-six economy specs (currency, goods,
  ledger, production, clearing, beliefs, agents, spawner, demand, scale, config, calendar, stability, viability, reference chain, welfare and wages, skill barriers, estates, carriage, regions, money supply, population, transport, resources, information, external sector), the evidence file, the
  bazaarBot reference, the roadmap, a draft vision and scope
  ([vision.md](vision.md)), ADR 0001 (accepted), ADR 0002 (proposed), ADR 0003 (accepted) and ADR 0004 (accepted by the supervisor, human review pending; the outside trades through pass-through counterparties). Every economy spec has a Requirements section.
- **Planned:** 25 more specs, listed in the roadmap with layers and
  dependencies, and a dependency picture of the first tier.
- **Known gaps:** an accepted vision and version-1 scope (a draft proposal
  exists); demand beyond subsistence; services, land and labour as goods; the player's place in the
  economy; explainability for designers; a cost budget; authoring tools.
  Several sources behind the numbers are unverified summaries
  (`specs/evidence.md`).
- **Not yet done:** running bazaarBot to see what stable behaviour looks like. The Doran and Parberry paper was read on 2026-10-10 (`specs/evidence.md`).

## 7. How the documents fit together

| Need | Read |
|------|------|
| What to read for a task | [documentation_index.md](documentation_index.md) |
| What version 1 is for and what it leaves out | [vision.md](vision.md) |
| The plan and the order, and which spec needs which | [roadmap.md](roadmap.md) |
| What to take into account before writing a spec | [notes.md](notes.md) |
| Why a decision was taken | [adr/README.md](adr/README.md) |
| The rules every change follows | [rules/](rules/architecture.md) |
| How an area of the economy works | [specs/README.md](specs/README.md) |
| What bazaarBot does | [specs/bazaarbot-reference.md](specs/bazaarbot-reference.md) |
| Where a number came from | [specs/evidence.md](specs/evidence.md) |

## 8. Next steps

1. Version 1 scope is accepted ([vision.md](vision.md), [ADR 0003](adr/0003-version-1-scope.md)). Still to confirm with the first consumer: the number of recipes and the tick rate.
2. Read the Doran and Parberry paper and run the bazaarBot example.
3. Build a small vertical slice (one region, a short chain, clearing, beliefs
   and stability checks) before writing the kingdom-level specs, and let it
   show which specs are really needed.

## Updating this file

Keep it short. Update it when the principles, the picture, the status or the
next steps change, and in the same change as any ADR that alters them.
