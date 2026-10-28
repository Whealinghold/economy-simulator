# Spec: The reference chain and the acceptance scenarios

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-REF-NNN` and
`NFR-REF-NNN`, which tests cite (testing.md section 4). They are in section 15.

Layer: fixture ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)). This
is a test fixture and a list of acceptance scenarios. It is not part of the library API,
adds no configuration field and no code to the core. The fixture is **data**: a set of
configuration documents in the file format of `spec-config.md` 6, kept in
`testdata/chains/` (architecture.md 2.6, testing.md 10.4). Needs every spec of tier 1:
`spec-goods.md`, `spec-ledger.md`, `spec-production.md`, `spec-clearing.md`,
`spec-beliefs.md`, `spec-agents.md`, `spec-scale.md`, `spec-config.md`,
`spec-calendar.md`, `spec-welfare-and-wages.md`, `spec-estates.md`,
`spec-skill-barriers.md`, `spec-viability.md` and `spec-stability.md`. It reads
`spec-spawner.md` and `spec-demand.md`, which were written at the same time by other
authors; section 7.7 and the assumptions in section 2 say what is taken from them.

**Every number in this spec is a placeholder.** Nothing has run in Go. The numbers
that come from a run (the surplus factors, the censuses, the seed tables, the
break-even gains) were computed by a Python prototype of the equations of
`spec-viability.md` 3 to 8 with exact fractions, run in Docker (section 14), over the
very documents printed in section 4: the prototype loads them with the layering of
`spec-config.md` 6.4 and reads the numbers it quotes from them. It is not the Go
design. The packs are **setting-neutral content invented for the fixture**; none of
their quantities is a default, a calibration or a claim of stability (N-015, N-027).
This spec does not close N-027: no default macro constant is stable until the bands
of `spec-stylised-facts.md` exist and a run has been compared with them.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

The reference chain is the standard fixture of the integration and end-to-end tests
(testing.md 2, 10.4): the configuration that the invariant, determinism, golden-file and
configuration-only tests run, and that every stability scenario of
`spec-stability.md` 8 is built from. This spec says what the fixture contains, why, how
its variants are layered, what each scenario runs on it, what it measures and what form
the pass or fail takes.

It answers four requests that other specs left for it:

- `spec-stability.md` 8 and note N-047: the chain must contain a self-employed staple
  role with a demand elasticity below 1, a perishable, a recipe cycle, a long recipe, a
  durable with a replacement cycle, a role with a job longer than the base welfare
  window, a region that can lose its income, a warm start, and a world whose only
  demand is upkeep (section 3);
- `spec-viability.md` 13.2: the acceptance scenarios A1 to A10 are assigned to packs
  (section 7.3), and note N-070: the regional yields of the three-region run are listed
  (section 4.4.3);
- `spec-production.md` 8.4 and `README.md`, "Still open" 1: one trainee-eligible recipe
  above the break-even of training and one below (section 4.5);
- note N-030: five regions with different climates, about 1 000 households each and 30
  commodities (section 4.4.4), with the recipe count (45) that the note lacked.

Out of scope: the **bands** (the numbers a measure must fall inside) are
`spec-stylised-facts.md`'s; the RPG kingdom data pack is content under `examples/`
(roadmap, "Deliverables that are not specs"); the experiment harness is
`spec-experiments.md`'s; nothing here changes a rule of any other spec.

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | Is the fixture code or data? | Data only: TOML documents loaded by `config.LoadFS`. The tests that use it are code, in the packages they test | Architecture.md 9.4 and testing.md 6.4 require a chain from a configuration file alone; a fixture in Go would hide a dependence on code |
| 2 | One chain or several? | One **base** document and **packs** layered on it (`spec-config.md` 6.4): a region document per variant, a warm-start document per variant, overlays that change one thing | A variant differs from another by a few tables; layering by key (later wins, no deletion) keeps each file small and the difference visible |
| 3 | Whose names? | Setting-neutral and functional (`staple`, `fuel`, `mineral`, `tool_a`, `grower`). No coin, metal, title, race or farming word. The second setting (a crater colony) is labelled setting-specific content and shares no ID with the first | `AGENTS.md` setting-agnostic rule; vision section 5 item 3 asks for a second, unrelated setting |
| 4 | Where do the numbers come from? | Invented, then checked: the quantities are those that make every variant pass the four viability layers with room, tuned in the prototype of section 14. The reference prices and the warm-start tables are the **solver's output** (rounded as `spec-viability.md` 6.3), not hand-written | A hand-written seed table would test nothing; a table that the solver can regenerate is a test of the solver (A9) |
| 5 | How big? | The base chain is bazaarBot-sized (9 commodities, 10 recipes, 6 roles, 600 households). The first consumer's world is the base plus seven product lines: 30 commodities, 45 recipes, 20 roles, 5 000 households in five regions | N-030 and `vision.md` 2; the base alone keeps most scenarios fast, the large world is for scale, cost and golden tests |
| 6 | Scale | Weight 1 everywhere (agents are households), except where a scenario sets a scale (S10) | `vision.md` 2 and N-030: the first consumer needs no weight above 1; it also avoids the cohort lockstep (N-014) in the standard fixture |
| 7 | Bands | Not set here. Each scenario states its measure and the form of its pass, and names the band as a key of a bands document that STY fills. Until a key has a value the scenario reports the measure and asserts only what is exact | The task and `spec-skill-barriers.md` FR-SKL-025: bands live in the reference chain's data and are set by a run that a human reviews; inventing numbers would be a claim of validation |
| 8 | What do SPN and DEM change? | Nothing in the base documents. The spawner is taken at its defaults. Demand is a **module** and the base has none (`spec-agents.md` 11.7: a world whose only demand is upkeep is needed by S19); the demand overlay of 4.4.6 adds it | Both specs are drafts written at the same time; the base must not depend on a decision they may change |
| 9 | Trade between regions | The multi-region variants are valid and viable as sets of regions that do not trade, and their pooled behaviour is checked statically by the solver (A3, A5). The dynamic trade scenarios (X1, X2) wait for `spec-merchants.md` (REG, TRN, INF and EXT are written), and are specified here only as far as they can be | `spec-regions.md`, `spec-transport.md`, `spec-information.md` and `spec-external-sector.md` now exist; the merchant and its shipper do not, and inventing one here would be a decision for MER |

Assumptions about specs that are not final (so a change there names what to revisit):

1. **SPN** (`spec-spawner.md`). Defaults for every field of its section 11. A role the
   default spawner may not fill is marked `Closed` (the `visitor` role). The spawner
   does not re-seed a role that has no agents (FR-SPN-042), so a region's missing role
   (a line not consumed there) is not a fault. Scenario S06 and S14 need its failure
   ladder (FR-SPN-013 to 015); its scenarios S26 and S27 are in 7.7.
2. **DEM** (`spec-demand.md`). Base packs have no `[modules.demand]` table; the module is
   switched on only by the overlay of 4.4.6 (FR-DEM-001). The names `Want`, `Taste`
   and the file layout of its 3.2 are taken as printed.
3. **INT** (`spec-integration.md`). Yield shocks are set between steps by
   `SetYieldShocks` (`spec-integration.md` 5.1), and an external agent is added by
   `AddAgent` with a sticky `Direct` directive (4.1, 4.3); INT 9.1 says a script is a
   list of (tick, call) and fixes no file format, so the shock script format of 7.1
   stays this spec's. Nothing else of INT is used.
4. **OBS** (`spec-observability.md`). The events of observability-and-logging.md 4.3
   as `spec-observability.md` 4.1 fixes their fields, and the snapshot fields of its 5.3
   that section 9 lists as needs.
5. **WLD** (`spec-world.md`). The phase table of `spec-world.md` 6 and its
   construction phase (5.6, with `spec-ledger.md` 5.2.4).

## 3. What the fixture must contain

The table lists each feature the notes and the register ask for, where it is in the
packs, and the scenario that needs it. A test reads the effective configuration (not
the text) and checks each row (FR-REF-002).

| ID | Feature | Where | Needed by |
|----|---------|-------|-----------|
| C01 | A **self-employed staple role** whose households sell what they grow and whose staple has a demand elasticity below 1 | role `grower`; the `staple` is upkeep (`spec-agents.md` 11.6: elasticity 0 to 1) | S04, S05, S16 |
| C02 | A **perishable** in the upkeep | `prepared`, `shelf_life = 4` | S07 |
| C03 | A **recipe cycle** that is productive, with a cost-push gain below 1 | `grow_staple` takes 40 `seed`, `save_seed` returns `seed` from `staple`; `g = CostFloor / 10 000 * 40 / 1800` | S08, A2 |
| C04 | A **long recipe** (a delayed supply response, and a season window) | `grow_staple`, 60 ticks, startable in spring only | S05, S11, S12 |
| C05 | A **durable with a replacement cycle** | `tool_a` (durability 10) and `tool_b` (8), made by `make_tool_a` and `make_tool_b` | S17, S06 |
| C06 | A role with a **job longer than the base welfare window** (30) | `grower`: effective window 120 (`window_extended`) | S11 |
| C07 | A **region that can lose its income**: an estate with a pool, relief and a failure ladder | variant `famine`, with the `estates` overlay | S14 |
| C08 | A **warm start** | the `warm-*.toml` documents, solver output | A1, S18, S22 |
| C09 | A world with **upkeep as the only demand** | every variant without the `demand` overlay | S19 |
| C10 | A **source** (no inputs), a **sited** recipe, a **discard** output | `cut_fuel`, `dig_mineral`, `gather_seed`; `refine` discards `waste` | A1, FR-PRO-030 |
| C11 | **Several inputs, outputs and tools**, a recipe that needs a tool and one that does not | `prepare` (tool), `prepare_basic` (none); `refine` (two inputs, two outputs) | A1 |
| C12 | **Climate as data**: yields that differ by region and season, 0 where a recipe cannot run | `two`, `three`, `five` | A3, A5, X1 to X3 |
| C13 | **Two trainee-eligible recipes, one above and one below the break-even** of training | `prepare_basic` (gain +136 at the seeds of `one`), `finish_a_basic` (gain -7 011 at the seeds of `five`, region `centre`) | S06, `README.md` "Still open" 1 |
| C14 | A role for **external agents**, closed to the spawner | `visitor` | S01, S13 |
| C15 | A **second, unrelated setting** | `second/second.toml` | A4, X4 |
| C16 | **Defective variants**, one defect each | `defects/` | A2 |
| C17 | The world size of the first consumer | `five` | X3, A10, S10 |

## 4. The packs

### 4.1 Files and layering

All files live in `testdata/chains/` (architecture.md 2.6). A **pack** is the list of
documents merged in order, later over earlier (`spec-config.md` 6.4): tables merge by
key, a scalar or a list in a later document replaces the earlier one, no document
deletes a key. A variant is `base`, its region document, and its warm-start document.

```
testdata/chains/
  base.toml                 commodities, recipes and roles of the base chain; the world's upkeep
  lines.toml                seven product lines (21 commodities, 35 recipes, 14 roles)
  one.toml  two.toml  three.toml  five.toml  famine.toml
                            region documents: regions, yields, extra upkeep, groups
  warm-one.toml  warm-two.toml  warm-three.toml  warm-five.toml  warm-famine.toml
                            warm-start documents: the solver's output, never hand-written
  demand.toml               the demand overlay (module on, a luxury chain, wants, tastes)
  one-demand.toml  warm-one-demand.toml
  overlay/                  estates, skill, a5, nofloor, notool, imbalance-cook, glut-prepared,
                            h200, long-grow, overhang, poor, perturbed, faucet, faucet-sunk,
                            sink, trade, resources, external, own-use, population, migration,
                            settlements, storage, employment, tenure, credit, risk,
                            infrastructure, cover, polity, polity-governed, fiscal,
                            fiscal-governed, institutions, institutions-inert, policy,
                            policy-directed, saving, technology (4.5)
  calibrated/<stage>-<n>.toml  overlays written by a calibration (`spec-calibration.md`),
                            layered after the pack they calibrate (4.5)
  defects/                  d1 to d6: one defect each
  second/second.toml  second/warm-second.toml
                            the second setting, which shares no ID with the first
  expected/<pack>.toml      the effective-configuration hash, the warnings, the solver report
  bands/<pack>.toml         the bands of the scenarios (values: STY); bands/one.toml declares
                            s10.tolerance_bp and s10.dispersion_min_bp, unset
  bands/regimes.toml        the regime thresholds, one file for every pack (STY 6.5.1)
  scripts/<scenario>.toml   shock scripts of 7.1
```

| Variant | Documents, in order | Regions | Households | Commodities | Recipes | Roles |
|---------|---------------------|---------|-----------:|------------:|--------:|------:|
| `one` | `base`, `one`, `warm-one` | 1 | 600 | 9 | 10 | 7 |
| `two` | `base`, `two`, `warm-two` | 2 | 1 000 | 9 | 10 | 7 |
| `three` | `base`, `three`, `warm-three` | 3 | 1 800 | 9 | 10 | 7 |
| `five` | `base`, `lines`, `five`, `warm-five` | 5 | 5 000 | 30 | 45 | 21 |
| `famine` | `base`, `famine`, `overlay/estates`, `warm-famine` | 1 | 600 | 9 | 10 | 7 |
| `one-demand` | `base`, `demand`, `one-demand`, `warm-one-demand` | 1 | 600 | 12 | 13 | 9 |
| `second` | `second/second`, `second/warm-second` | 1 | 1 000 | 10 | 10 | 5 |

The role count includes `visitor`, which has no group. A **cold** variant is the same
list without its warm-start document. The region document and the warm-start document of
a variant share no key, so their order does not matter (FR-REF-003).

The numbers in the documents are in minor units, whole units and basis points as
`spec-config.md` fixes. Comments in the files are part of the fixture: each file starts
with a comment that names this spec and the section that describes it (testing.md 5.6).

### 4.2 The base document

Nine commodities, ten recipes, six roles and a role for external agents. Upkeep per
household per tick: 2 `staple`, 1 `prepared`, 1 `fuel`. The calendar is Earth's (365
ticks a year, seasons spring, summer, autumn and winter, `spec-calendar.md` 7.4), so
there is no `[calendar]` table. The reference prices are the solver's prices for region
`north` of `one`, scaled so that the basket is 1 000 (`spec-currency.md` 5.2) and
rounded half to even; a price below the resolution floor 50 would be raised to it, and
`waste` is a discard-only commodity whose price is never used.

```toml
schema = 1

[world]
anchor_hold = 5000
upkeep = [ { commodity = "staple", quantity = 2 }, { commodity = "prepared", quantity = 1 }, { commodity = "fuel", quantity = 1 } ]

[commodity.seed]
reference_price = 222
[commodity.staple]
reference_price = 96
shelf_life = 700
classes = [ "food" ]
[commodity.prepared]
reference_price = 628
shelf_life = 4
classes = [ "food" ]
[commodity.fuel]
reference_price = 179
raw = true
classes = [ "fuel" ]
[commodity.mineral]
reference_price = 837
raw = true
classes = [ "material" ]
[commodity.refined]
reference_price = 3190
classes = [ "material" ]
[commodity.waste]
reference_price = 50
[commodity.tool_a]
reference_price = 13915
durability = 10
classes = [ "tool" ]
[commodity.tool_b]
reference_price = 17105
durability = 8
classes = [ "tool" ]

[recipe.grow_staple]
duration = 60
sited = true
inputs   = [ { commodity = "seed", quantity = 40 } ]
outputs  = [ { commodity = "staple", quantity = 1800, cost_share = 10000 } ]
requires = [ { commodity = "tool_a", wear_per_use = 10 } ]
[recipe.gather_seed]
sited = true
outputs = [ { commodity = "seed", quantity = 2, cost_share = 10000 } ]
[recipe.save_seed]
inputs  = [ { commodity = "staple", quantity = 20 } ]
outputs = [ { commodity = "seed", quantity = 20, cost_share = 10000 } ]
[recipe.prepare]
inputs   = [ { commodity = "staple", quantity = 2 }, { commodity = "fuel", quantity = 1 } ]
outputs  = [ { commodity = "prepared", quantity = 8, cost_share = 10000 } ]
requires = [ { commodity = "tool_b", wear_per_use = 1 } ]
[recipe.prepare_basic]
trainee_eligible = true
trainee_yield = 7000
inputs   = [ { commodity = "staple", quantity = 2 }, { commodity = "fuel", quantity = 1 } ]
outputs  = [ { commodity = "prepared", quantity = 4, cost_share = 10000 } ]
[recipe.cut_fuel]
sited = true
outputs  = [ { commodity = "fuel", quantity = 14, cost_share = 10000 } ]
[recipe.dig_mineral]
sited = true
outputs  = [ { commodity = "mineral", quantity = 3, cost_share = 10000 } ]
[recipe.refine]
duration = 2
inputs   = [ { commodity = "mineral", quantity = 5 }, { commodity = "fuel", quantity = 2 } ]
outputs  = [ { commodity = "refined", quantity = 3, cost_share = 10000 }, { commodity = "waste", quantity = 1, cost_share = 0, discard = true } ]
[recipe.make_tool_a]
duration = 3
inputs  = [ { commodity = "refined", quantity = 2 } ]
outputs = [ { commodity = "tool_a", quantity = 1, cost_share = 10000 } ]
[recipe.make_tool_b]
duration = 3
inputs  = [ { commodity = "refined", quantity = 3 } ]
outputs = [ { commodity = "tool_b", quantity = 1, cost_share = 10000 } ]

[role.grower]
recipes = [ "grow_staple", "save_seed", "gather_seed" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "seed", quantity = 40 }, { commodity = "staple", quantity = 10 }, { commodity = "tool_a", quantity = 1 } ]
[role.cook]
recipes = [ "prepare", "prepare_basic" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 }, { commodity = "tool_b", quantity = 1 } ]
[role.forester]
unskilled = true
recipes = [ "cut_fuel" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
[role.digger]
unskilled = true
recipes = [ "dig_mineral" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
[role.refiner]
recipes = [ "refine" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "mineral", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
[role.maker]
recipes = [ "make_tool_a", "make_tool_b" ]
start_stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "refined", quantity = 5 }, { commodity = "staple", quantity = 10 } ]
[role.visitor]
# an external agent (the player or a scripted buyer) is added to this role at run time; it has no recipes
closed = true       # the spawner places nobody in it (spec-spawner.md 6.2)
trades = [ "fuel", "prepared", "staple" ]
```

Reading it. The **bazaarBot shape** (`bazaarbot-reference.md` 10) is kept: a food chain
(`grower`, `cook`), a fuel gatherer (`forester`), a mineral gatherer (`digger`), a
refiner and a toolmaker (`maker`). What differs is what the register asks for:

- `grow_staple` is a 60-tick job that can start only in spring (the profile of the
  region document); with the Earth calendar a household can start two jobs a year (start
  capacity 2, `spec-viability.md` 3.1), the second at about day 120. It takes 40 `seed`
  and wears a whole `tool_a` (wear 10 of durability 10), so the tool is a flow of one
  per job. `gather_seed` is the **wild source** of the seed cycle: without a source
  outside the cycle the structural check (`spec-config.md` 7.5) and V1 reject the
  chain, and a cycle whose only source is itself is not a chain that can start (6.2).
- `prepared` lasts 4 ticks. It is upkeep, made all year, so its supply gap is 0 and the
  check V4 passes; a cook that cannot sell it in four ticks loses it (S07).
- The tools are the durables. `prepare` wears one `tool_b` item a use, so a `tool_b`
  lasts eight executions, about two weeks for a cook who works every other tick. That
  is faster than a real tool, chosen so that the toolmaker is a role with a visible share
  (about 9% of households) and the replacement demand is a flow worth testing (S17).
  `make_tool_a` and `make_tool_b` are 3-tick jobs.
- `refine` has two inputs and two outputs, one of which is a discard (`waste`): the
  recipe never puts it in an inventory (`spec-production.md` 7.2).
- `prepare_basic` is the trainee-eligible recipe (`spec-production.md` 8): no tool, half
  the output of `prepare`, trainee yield 7 000. For a skilled cook it is dominated (4
  units against 8 less the tool wear), so only a trainee runs it.
- `start_stock` gives each role the stock of the warm-start table without the lean
  reserve, so the role defaults alone pass `start_stock_below_target` of
  `spec-agents.md` 13; the warm-start table overrides them per group.
- `forester` and `digger` are marked `unskilled`: they are the baseline of the observed
  skill premium of the skilled roles (`spec-welfare-and-wages.md` 4.3; S06).
- `visitor` is the role of an external agent (the player, or a scripted buyer of S01). It
  has no recipes and no group; `closed = true` keeps the default spawner from placing
  anyone in it (FR-SPN-041).

The base alone is not a world: it has no region. The region documents add them.

### 4.3 The product lines

`lines.toml` adds seven **product lines** `a` to `g`. A line is three commodities
(`raw_k`, `semi_k`, `good_k`), five recipes and two roles, all of one shape, so the file
is mechanical and each line is a copy of the others with the letter changed. The shape,
for line `a`:

```toml
[commodity.raw_a]
reference_price = 127
raw = true
classes = [ "raw_line" ]
[commodity.semi_a]
reference_price = 509
classes = [ "semi_line" ]
[commodity.good_a]
reference_price = 107
classes = [ "good_line" ]

[recipe.gather_a]
sited = true
outputs  = [ { commodity = "raw_a", quantity = 20, cost_share = 10000 } ]
[recipe.gather_a_deep]
sited = true
outputs  = [ { commodity = "raw_a", quantity = 28, cost_share = 10000 } ]
requires = [ { commodity = "tool_a", wear_per_use = 1 } ]
[recipe.process_a]
duration = 2
inputs   = [ { commodity = "raw_a", quantity = 8 }, { commodity = "fuel", quantity = 1 } ]
outputs  = [ { commodity = "semi_a", quantity = 12, cost_share = 10000 } ]
[recipe.finish_a]
inputs   = [ { commodity = "semi_a", quantity = 12 } ]
outputs  = [ { commodity = "good_a", quantity = 100, cost_share = 10000 } ]
requires = [ { commodity = "tool_b", wear_per_use = 1 } ]
[recipe.finish_a_basic]
trainee_eligible = true
trainee_yield = 3000
inputs   = [ { commodity = "semi_a", quantity = 12 } ]
outputs  = [ { commodity = "good_a", quantity = 30, cost_share = 10000 } ]

[role.gatherer_a]
recipes = [ "gather_a", "gather_a_deep" ]
[role.crafter_a]
recipes = [ "process_a", "finish_a", "finish_a_basic" ]
```

Lines `b` to `g` are the same with the letter changed, and without `trainee_eligible` in
`finish_k_basic` (only `finish_a_basic` is eligible, with yield 3 000; the others are
ordinary dominated recipes). The reference prices of the lines are the mean over the
regions that use the line of the autarky seeds, rounded: raw 100 to 141, semi 446 to
512, good 98 to 109 (the file lists each). `gather_k_deep` produces 40% more raw for a
tool wear of 1 in 10; at the solver's prices it costs 10% more per unit than
`gather_k`, so it is dominated until the raw price rises by about a tenth, which gives
the choice of recipe by margin (STB-L48) something to decide.

The good of a line is not in the world's upkeep. A region consumes a line's good
through its **extra upkeep** (`spec-agents.md` 5.2): one unit a tick of each of three
goods (4.4.4). A line's good is therefore sold only in the regions that consume it.

### 4.4 The variants

#### 4.4.1 `one`: one region, a bazaarBot-style chain

One region, `north`, 600 households in six roles. The yields are the whole of the
region's climate: staple in spring only, fuel, mineral and wild seed all year.

```toml
[region.north]
reference_basket = 1000
[region.north.yield.cut_fuel]
all = 10000
[region.north.yield.dig_mineral]
all = 10000
[region.north.yield.gather_seed]
all = 10000
[region.north.yield.grow_staple]
seasons = { spring = 10000 }
[region.north.group.cook]
households = 136
[region.north.group.digger]
households = 30
[region.north.group.forester]
households = 90
[region.north.group.grower]
households = 253
[region.north.group.maker]
households = 55
[region.north.group.refiner]
households = 36
```

The census is the **balanced census** of the solver (`spec-viability.md` 8.1; FR-VIA-028):
the households of a role are those its activities need at `lambda = 1`, with the
start-capacity row counted (growers can start only two jobs a year, so they need three
times the labour their work alone suggests), and the 600 households are the largest-
remainder apportionment of those needs. The available work share divides every term alike
and cancels in the proportion, so it is not in the census formula (`spec-viability.md` 5.2;
N-306). Because the realm can make 2.512 baskets, the
census gives every role about 1.8 times the households a basket needs, which is the
glut of `spec-viability.md` 4.7 and the reason the production halt exists (S19).

The warm-start document of `one`, the output of the prototype of `SolveSteadyState`
(prices scaled so the basket is 1 000, rounded half to even; the wage is the labour
dual; money by `spec-viability.md` 8.3.1 with `MoneyBufferTicks` 10; stock by 8.4.1 with
`StockCoverTicks` 5; jobs by 8.5 with the stagger of 8.4.3; the lean reserve of
`staple` is the use per tick times the 118 ticks to the first harvest, spread over the
growers):

```toml
[warm_start.region.north]
wage = 2512
[warm_start.region.north.price]
fuel = 179
mineral = 837
prepared = 628
refined = 3190
seed = 222
staple = 96
tool_a = 13915
tool_b = 17105
[warm_start.region.north.group.cook]
money = 11360
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 }, { commodity = "tool_b", quantity = 1 } ]
[warm_start.region.north.group.digger]
money = 10989
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
[warm_start.region.north.group.forester]
money = 10989
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
[warm_start.region.north.group.grower]
money = 78810
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "seed", quantity = 40 }, { commodity = "staple", quantity = 664 }, { commodity = "tool_a", quantity = 1 } ]
[warm_start.region.north.group.maker]
money = 22557
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "refined", quantity = 5 }, { commodity = "staple", quantity = 10 } ]
jobs = [ { recipe = "make_tool_a", in_progress = 418 }, { recipe = "make_tool_b", in_progress = 5113 } ]
[warm_start.region.north.group.refiner]
money = 16531
stock = [ { commodity = "fuel", quantity = 5 }, { commodity = "mineral", quantity = 5 }, { commodity = "prepared", quantity = 4 }, { commodity = "staple", quantity = 10 } ]
jobs = [ { recipe = "refine", in_progress = 5492 } ]
```

What the table shows: growers have no job in progress, because tick 0 falls on day 0 and
spring begins on day 59, so the set of virtual start offsets of `grow_staple` is empty
(`spec-viability.md` 8.5.2) and the table writes no job for them; they hold the lean
reserve instead (664 `staple` each, enough for the whole realm for 118 ticks). A
perishable stock is cut to its shelf life (4 `prepared`). The one-tick recipes have no
job in progress at tick 0. The warm start creates, at construction, 24 638 321 minor
units of money and the goods of the table (`staple` 171 462, `seed` 10 120, `tool_a` 253,
`tool_b` 136, `refined` 275, `mineral` 180, `fuel` 3 000, `prepared` 2 400) plus the
inputs of the jobs in progress (A7).

#### 4.4.2 `two`: two regions, an arbitrage opportunity

`north` (600 households) is rich in minerals and poor in fuel; `south` (400) the
reverse and plants twice a year at lower yields. Yields in basis points (0 = the recipe
cannot run):

| Recipe | `north` | `south` |
|--------|--------:|--------:|
| `grow_staple` | spring 10 000 | spring 8 000, autumn 5 000 |
| `cut_fuel` | all 6 000 | all 12 000 |
| `dig_mineral` | all 12 000 | all 6 000 |
| `gather_seed` | all 10 000 | all 10 000 |

Census (balanced, per region): `north` cook 124, digger 23, forester 137, grower 232,
maker 51, refiner 33; `south` cook 98, digger 37, forester 54, grower 143, maker 41,
refiner 27. Both regions have `reference_basket = 1000`. The seeds of the warm start
differ by region, which is the opportunity: `staple` 85 and 134, `fuel` 265 and 132,
`mineral` 619 and 1 231 (north, south). Both regions are viable alone (autarky `lambda*`
2.229 and 2.216) and pooled the realm reaches 2.626. The staple to fuel price ratio is
0.322 in `north` and 1.015 in `south` in autarky (a factor of 3.2), and 1.001 pooled.
A merchant who buys fuel where it is cheap and sells it where it is dear has a margin
of that size before any transport cost; the gap that the dynamic scenario X1 must see
close. The document `two.toml` is the table above in the layout of `one.toml`; the
census of the file is the one printed here.

#### 4.4.3 `three`: three regions, comparative advantage (N-030, N-070)

Three regions of 600 households with the yields of `spec-production.md` 12.1 for the
crop and the iron-like mineral, and wood-like fuel yields that `spec-production.md`
12.1 did not list (note N-070: they are listed here):

| Recipe | `north` | `south` | `far_south` |
|--------|--------:|--------:|------------:|
| `grow_staple` | spring 10 000 | spring 12 000, autumn 7 000 | autumn 10 000 |
| `cut_fuel` | all 10 000 | all 6 000 | all 12 000 |
| `dig_mineral` | all 10 000 | all 5 000 | all 15 000 |
| `gather_seed` | all 10 000 | all 10 000 | all 10 000 |

`far_south` plants in autumn, a reverse hemisphere. Census: `north` as in `one`;
`south` cook 143, digger 63, forester 158, grower 140, maker 58, refiner 38;
`far_south` cook 142, digger 21, forester 78, grower 265, maker 57, refiner 37.
Autarky `lambda*` are 2.512, 2.060 and 2.666 (the regions differ, A3); pooled it is 2.666.
In the pooled solution all eight active activities are active in a strict subset of the
regions: `cut_fuel` and `dig_mineral` only in `far_south`, `refine` only in `north`, the
tools only in `south`, and `grow_staple` in `far_south` and `north`. The pooled relative
prices of staple to fuel, mineral to staple, refined to staple, prepared to staple and
`tool_b` to staple lie in the range of the three autarky ratios, each at an end of it:
the ratio of the region with the best yields (`far_south`), for example 0.641 for staple
to fuel against autarky ratios of 0.346, 0.537 and 0.641. The yield ratios are large
enough to show specialisation (note N-070 asked for this).

#### 4.4.4 `five`: the first consumer's world

Five regions with different climates and sizes (`vision.md` 2, N-030): 1 200, 1 000,
1 000, 800 and 1 000 households, 5 000 in all, at weight 1, so 5 000 agents (about 53 ms
a tick on the placeholder cost model of `spec-scale.md` 6.3; mean tradable set 7.7, close
to the reference size 8). The world is the base plus the seven lines: 30 commodities, 45
recipes, 20 roles with groups. A region consumes the goods of three lines through its
extra upkeep, and has a positive yield for each, so every region is viable alone:

| Region | Households | Extra upkeep (one unit of each) | `reference_basket` |
|--------|-----------:|----------------------------------|-------------------:|
| `north` | 1 200 | `good_a`, `good_c`, `good_d` | 1 318 |
| `east` | 1 000 | `good_b`, `good_e`, `good_f` | 1 298 |
| `south` | 1 000 | `good_c`, `good_f`, `good_g` | 1 304 |
| `west` | 800 | `good_a`, `good_c`, `good_e` | 1 309 |
| `centre` | 1 000 | `good_a`, `good_d`, `good_g` | 1 319 |

Yields (basis points, 0 where the recipe cannot run; the deep variant of a gatherer has
the same factor as its plain one):

| Recipe | `north` | `east` | `south` | `west` | `centre` |
|--------|--------:|-------:|--------:|-------:|---------:|
| `grow_staple` | spring 10 000 | spring 12 000, autumn 7 000 | autumn 10 000 | spring 8 000, summer 6 000 | spring 8 000, autumn 8 000 |
| `cut_fuel` | 12 000 | 8 000 | 6 000 | 10 000 | 14 000 |
| `dig_mineral` | 8 000 | 5 000 | 6 000 | 15 000 | 10 000 |
| `gather_a` | 12 000 | 6 000 | 0 | 10 000 | 8 000 |
| `gather_b` | 0 | 10 000 | 12 000 | 8 000 | 6 000 |
| `gather_c` | 8 000 | 0 | 10 000 | 12 000 | 6 000 |
| `gather_d` | 10 000 | 8 000 | 0 | 6 000 | 12 000 |
| `gather_e` | 0 | 12 000 | 8 000 | 10 000 | 6 000 |
| `gather_f` | 6 000 | 10 000 | 12 000 | 0 | 8 000 |
| `gather_g` | 10 000 | 0 | 6 000 | 8 000 | 12 000 |

Census, `north`: cook 222, crafter a/c/d 53 each, digger 75, forester 126, gatherer a/c/d
6, 9 and 7, grower 414, maker 110, refiner 72. `east`: cook 200, crafter b/e/f 48 each,
digger 108, forester 171, gatherer b/e/f 6, 5 and 6, grower 196, maker 99, refiner 65.
`south`: cook 164, crafter c/f/g 39 each, digger 74, forester 187, gatherer c/f/g 5, 4
and 9, grower 306, maker 81, refiner 53. `west`: cook 165, crafter a/c/e 40, 40 and 39,
digger 30, forester 112, gatherer a/c/e 5, 4 and 5, grower 222, maker 84, refiner 54.
`centre`: cook 218, crafter a/d/g 52 each, digger 60, forester 107, gatherer a/d/g 9, 6
and 6, grower 256, maker 110, refiner 72. There are 60 groups. The smallest have four
households (a gatherer of a line), below the floor `K` of 10, so each household is its
own agent and the group is thin by construction (`spec-scale.md` 5.2); that is a
property of the line structure, not of the scale.

Autarky `lambda*` are 1.904 (`centre`), 1.704 (`east`), 1.915 (`north`), 1.642 (`south`)
and 1.805 (`west`); pooled 2.039. The labour a household must give to subsist is 52% to
61% of its ticks, under the 68.5% that `ExpectedWorkTicks` 250 allows. Only the regions
that consume a line trade its goods, so each region has 17 tradable commodities and the
world has 85 active markets, not the 150 of the note (N-030): the clearing term of the
cost model is smaller than the note counted, and the agent term (5 000 agents) is as
counted.

#### 4.4.5 `famine`: a closed region that can starve

`famine` is `one` with poor land (staple yield 3 000 against 10 000 in spring), the
`estates` module on, and the census and seeds re-solved for it. Its purpose is the
starvation collapse of `vision.md` 5 item 1 and scenario S14: a region whose margin is
thin, with a pool, relief and the failure ladder, hit by a harvest failure.

```toml
[region.north]
reference_basket = 1000
pool_release = 100
relief_below = 10000
[region.north.yield.cut_fuel]
all = 10000
[region.north.yield.dig_mineral]
all = 10000
[region.north.yield.gather_seed]
all = 10000
[region.north.yield.grow_staple]
seasons = { spring = 3000 }
[region.north.group.cook]
households = 68
[region.north.group.digger]
households = 17
[region.north.group.forester]
households = 45
[region.north.group.grower]
households = 417
[region.north.group.maker]
households = 33
[region.north.group.refiner]
households = 20
```

Autarky `lambda*` 1.315 (labour share to subsist 76.0%; nominal 1.112), so the pack
carries the warnings `subsistence_exceeds_expected_work` (76% against the 68.5% that
`ExpectedWorkTicks` allows) and `target_above_surplus`. A start-capacity row binds for
`grow_staple` and the labour row is slack: at full labour the wage dual is 0 and every
good that does not use the capacity would price at the floor. The seed table is therefore
built, by the rule of `spec-viability.md` 5.3.4 (FR-VIA-058), from the **nominal**
programme, where labour binds: wage 1 624, `staple` 219, `fuel` 116, `mineral` 541,
`refined` 2 062, `tool_b` 11 060 (against 2 512, 96, 179, 837, 3 190 and 17 105 in `one`,
whose land is three times as good), and the report lists the region under `NominalSeed`.
At a yield of 4 000 the full-labour table has a wage (755) and would be used; at 2 200 the
region is not viable (`lambda*` 0.964). The pool starts empty (the table seeds none), so
relief begins only after the first removals.

#### 4.4.6 The demand overlay and `one-demand`

For the scenarios of `spec-demand.md` (S23 to S25; 7.7) the
fixture needs a luxury role, a want with two alternatives and a taste table for two
roles (note N-080). `demand.toml` adds three luxury commodities, three recipes, two
roles, two wants and the tastes of eight roles. The module is on because the `[modules.demand]`
table is present (`spec-demand.md` 3.1). `one-demand.toml` is `one.toml` with the census
re-solved for the wants. The demand overlay is the file `testdata/chains/demand.toml` (not a file under `overlay/`), and the module-alone golden of the module is the pack `one-demand`, which carries its own census, so it departs from testing.md 7.1.1 (the reference chain with an overlay): see that rule.

```toml
[commodity.treat_a]
reference_price = 664
classes = [ "luxury" ]
[commodity.treat_b]
reference_price = 1079
classes = [ "luxury" ]
[commodity.comfort]
reference_price = 1425
classes = [ "luxury" ]

[recipe.make_treat_a]
inputs  = [ { commodity = "prepared", quantity = 1 }, { commodity = "fuel", quantity = 1 } ]
outputs = [ { commodity = "treat_a", quantity = 5, cost_share = 10000 } ]
[recipe.make_treat_b]
duration = 2
inputs  = [ { commodity = "staple", quantity = 2 }, { commodity = "fuel", quantity = 1 } ]
outputs = [ { commodity = "treat_b", quantity = 5, cost_share = 10000 } ]
[recipe.make_comfort]
inputs  = [ { commodity = "refined", quantity = 1 } ]
outputs = [ { commodity = "comfort", quantity = 4, cost_share = 10000 } ]

[role.treat_maker]
recipes = [ "make_treat_a", "make_treat_b" ]
[role.comfort_maker]
recipes = [ "make_comfort" ]

[modules.demand.want.comfort]
items = [ { commodity = "comfort", weight = 10000 } ]
quantity = 1
period = 10
rank = 1
elasticity = 15000
[modules.demand.want.treat]
items = [ { commodity = "treat_a", weight = 10000 }, { commodity = "treat_b", weight = 10000 } ]
quantity = 1
period = 5
rank = 2
elasticity = 20000
substitution = 2

[modules.demand.taste.comfort_maker.comfort]
scale = 10000
[modules.demand.taste.cook.comfort]
scale = 10000
[modules.demand.taste.cook.treat]
scale = 5000
[modules.demand.taste.digger.comfort]
scale = 10000
[modules.demand.taste.forester.comfort]
scale = 10000
[modules.demand.taste.grower.comfort]
scale = 10000
[modules.demand.taste.grower.treat]
scale = 10000
[modules.demand.taste.maker.comfort]
scale = 10000
[modules.demand.taste.refiner.comfort]
scale = 10000
[modules.demand.taste.treat_maker.comfort]
scale = 10000
[modules.demand.taste.treat_maker.treat]
scale = 10000
```

The census of `one-demand`, solved with the wants as a fixed extra demand at the
reference (the solver does not yet include wants, `spec-viability.md` open question 3,
so this is the prototype's reading): cook 117, comfort maker 23, digger 39, forester 79,
grower 219, maker 48, refiner 46, treat maker 29. Its autarky `lambda*` with the wants
counted is 2.040; the full wants of a household cost about 317 a tick at the seeds
(`comfort` 142, `treat` 175), 32% of the basket, so the ceiling `1 + Wmax / U` of the mean
welfare ratio (`spec-demand.md` 7) is about 1.32 against the default target 1.5 and the
warning `target_above_demand` is expected (S24 runs the case on purpose).
The two alternatives of the `treat` want differ in price by 62% (664 and 1 079), which is
the price gap that S25 needs. The warm-start document of `one-demand` is
the one of `one` with its census and the three new prices.

#### 4.4.7 `second`: the second setting

A crater colony, 1 000 households in `crater_rim`: ten commodities, ten recipes, five
roles, a two-season calendar. It is the recipe set of `spec-production.md` 12.2 as
rescaled by `spec-viability.md` 15, plus one recipe that closes a tool cycle (6.2). It
shares no commodity, recipe or role ID with the reference chain; it is **setting-specific
content**, kept in the fixture only to prove that a second, unrelated setting needs
configuration alone (vision 5 item 3, architecture.md 9.4; scenarios A4 and X4).

```toml
schema = 1

[world]
upkeep = [ { commodity = "biomass", quantity = 2 }, { commodity = "water", quantity = 1 } ]

[calendar]
seasons = [ { id = "lit", days = 184 }, { id = "dark", days = 181 } ]
season_start = 59

[commodity.ice]
reference_price = 112
raw = true
[commodity.water]
reference_price = 346
[commodity.biomass]
reference_price = 327
shelf_life = 60
[commodity.regolith]
reference_price = 66
raw = true
[commodity.wafer]
reference_price = 2163
[commodity.solder]
reference_price = 2097
[commodity.board]
reference_price = 12364
[commodity.slag]
reference_price = 50
[commodity.drill]
reference_price = 14330
durability = 500
[commodity.fabricator]
reference_price = 41155
durability = 1000

[recipe.cut_ice]
sited = true
outputs  = [ { commodity = "ice", quantity = 30, cost_share = 10000 } ]
requires = [ { commodity = "drill", wear_per_use = 2 } ]
[recipe.melt]
inputs  = [ { commodity = "ice", quantity = 1 } ]
outputs = [ { commodity = "water", quantity = 6, cost_share = 10000 } ]
[recipe.grow_algae]
duration = 12
sited = true
inputs  = [ { commodity = "water", quantity = 4 } ]
outputs = [ { commodity = "biomass", quantity = 120, cost_share = 10000 } ]
[recipe.dig_regolith]
outputs = [ { commodity = "regolith", quantity = 30, cost_share = 10000 } ]
[recipe.refine_wafer]
inputs  = [ { commodity = "regolith", quantity = 3 } ]
outputs = [ { commodity = "wafer", quantity = 1, cost_share = 10000 } ]
[recipe.refine_solder]
inputs  = [ { commodity = "regolith", quantity = 2 } ]
outputs = [ { commodity = "solder", quantity = 1, cost_share = 10000 } ]
[recipe.fab_board]
duration = 3
inputs   = [ { commodity = "wafer", quantity = 2 }, { commodity = "solder", quantity = 1 } ]
outputs  = [ { commodity = "board", quantity = 1, cost_share = 10000 }, { commodity = "slag", quantity = 1, cost_share = 0, discard = true } ]
requires = [ { commodity = "fabricator", wear_per_use = 1 } ]
[recipe.make_drill]
inputs  = [ { commodity = "board", quantity = 1 } ]
outputs = [ { commodity = "drill", quantity = 1, cost_share = 10000 } ]
[recipe.make_fabricator]
inputs  = [ { commodity = "board", quantity = 3 }, { commodity = "solder", quantity = 1 } ]
outputs = [ { commodity = "fabricator", quantity = 1, cost_share = 10000 } ]
[recipe.salvage]
duration = 50
outputs = [ { commodity = "fabricator", quantity = 1, cost_share = 10000 } ]

[role.driller]
recipes = [ "cut_ice", "melt" ]
[role.algae_tender]
recipes = [ "grow_algae" ]
[role.regolith_miner]
recipes = [ "dig_regolith" ]
[role.smelter]
recipes = [ "refine_wafer", "refine_solder" ]
[role.fabber]
recipes = [ "fab_board", "make_drill", "make_fabricator", "salvage" ]

[region.crater_rim]
reference_basket = 1000
[region.crater_rim.yield.cut_ice]
all = 6000
[region.crater_rim.yield.grow_algae]
seasons = { lit = 10000, dark = 2500 }
[region.crater_rim.group.driller]
households = 377
[region.crater_rim.group.algae_tender]
households = 608
[region.crater_rim.group.regolith_miner]
households = 5
[region.crater_rim.group.smelter]
households = 5
[region.crater_rim.group.fabber]
households = 5
```

Autarky `lambda*` 1.966, the number `spec-viability.md` 15 gave for the same recipes, and
the prices match its list (biomass 327, board 12 364, drill 14 330, fabricator 41 155,
regolith 66, wafer 2 163, solder 2 097, water 346, ice 112): the prototype of this spec
reproduces that one. The tool sector is five households per role, the balanced census
being a fraction of a household (0.16 fabbers), and a floor of five keeps each role
alive. `cut_ice` omits the `Draws` of `spec-production.md` 12.2 because the resources
module (RES) did not exist when this fixture was built; the deposit is then limited by labour only. `spec-resources.md` now exists, and `overlay/resources.toml` (4.5) can give it the draws. The warm-start
document `second/warm-second.toml` is the solver output in the layout of 4.4.1.

### 4.5 Overlays and defect packs

An **overlay** is a document that changes one thing of a variant. It is layered after
the variant's documents. The list, with its use:

| Overlay | Content | Used by |
|---------|---------|---------|
| `overlay/estates.toml` | `[modules.estates]`; the region fields keep their defaults | S14, S06, S19 variants that need a pool |
| `overlay/skill.toml` | `training_ticks = 120`, `max_trainees = 3`, `teaching_load = 1000` on `cook` and `crafter_a`; weight 1 only | S06 |
| `overlay/a5.toml` | `south` of `two` cannot grow the staple: spring and autumn yields 0 | A5 |
| `overlay/nofloor.toml` | `min_work_share = 0` for `grower` | S04 |
| `overlay/notool.toml` | `requires = []` for `prepare` | S06 |
| `overlay/imbalance-cook.toml` | cooks 163, growers 226 (the census 20% off) | S06 |
| `overlay/glut-prepared.toml` | cooks 272 (twice the balanced number) | S07 |
| `overlay/h200.toml` | 200 households in every role | S10 |
| `overlay/long-grow.toml` | `grow_staple`: 85 ticks, 2 500 staple (`lambda*` 2.523) | S12 |
| `overlay/overhang.toml` | growers start with 1 992 `staple` (three times) | S16 |
| `overlay/poor.toml` | every group starts with 1 000 money | S20 |
| `overlay/perturbed.toml` | `prepare` makes 6, not 8 (`lambda*` 2.298): the warm table is stale | S22 |
| `overlay/faucet.toml` | `[modules.money_supply]` with one grant flow `stipend` on the `grower` households of `north`: 20 a tick (2% of the basket), a placeholder; defined by `spec-money-supply.md` 17.6 | S13 |
| `overlay/faucet-sunk.toml` | the same, plus a rate levy on the same role (`rate_bp` 3650 a year, `exempt` 3000); `spec-money-supply.md` 17.6 | S13, S33 |
| `overlay/sink.toml` | a rate levy on the households that have no paired grant (needs `[modules.money_supply]`, `spec-money-supply.md` 17.6); a sink with no faucet beside it | R9's provoked run (`spec-stylised-facts.md` 6.5.3); no scenario of the register |
| `overlay/trade.toml` (edge part) | the directed edges of `spec-regions.md` 12.4 for `two`, `three` and `five`, one file each, `overlay/trade-two.toml`, `overlay/trade-three.toml` and `overlay/trade-five.toml` (`[region.<from>.edge.<to>]`, distances placeholders; an edge table in a pack without its region would create a bare region, so the edge part cannot be one file); the modes and carrier recipes are TRN's (`spec-transport.md` 3, 17.4); with them a `[modules.information]` table (`info_speed`, `rumour_error_bp`, `rumour_max_age`) and `informed = true` on the merchant role (`spec-information.md` 12.1 shows the keys); the merchant itself is MER's: a role with `strategy = "merchant"`, `informed = true`, a `[role.<id>.merchant]` block and `trades`, a consignee role with `consigns = true`, and `[modules.merchants]`, as in `spec-merchants.md` 19.1 (the Go line is `economysim.NewWorld(cfg, strategy.Options()...)`); the pack's weights are 1 for the merchant role (`ErrMerchantWeight`) and the role has no recipe (`ErrMerchantShape`) | X1, X2, S28, S29, S46, S47, S67 to S72 |
| `overlay/population.toml` | `[modules.population]` with the birth and death schedules, `Renewal` 0 and wants on, at the placeholders of `spec-population.md` 18.3; the fixed factor is `overlay/resources.toml` (the binding set) or a fixed-factor double | S37 to S40, STY-W02, W03, W06, CLB C9 |
| `overlay/resources.toml` | the resources and endowments of `spec-resources.md` 18.1: `arable` (a `refill` stock for `grow_staple`), `deposit` (a finite stock for `dig_mineral`) and `stand` (a `logistic` stock with a refuge for `cut_fuel`), at the capacities of 18.2; a slack set and a binding set | S41 to S45 |
| `overlay/external.toml` | the outside world: one agentless node on a gate region of `one` or `three` with a few lines (`spec-external-sector.md` 18.2 is a worked configuration, `[modules.external.node.<id>]`); placeholders | S51 to S55 |
| `overlay/own-use.toml` | the subsistence floor: an own-use recipe on the staple role of `one` (a sited source with no inputs or tools, `spec-subsistence.md` 3.2, 16.1 is the shape) with a yield row in each region that is to offer it, and `[modules.subsistence]` at the placeholders of `spec-subsistence.md` 12 (`enter_ticks` 2, `exit_ticks` 3, `retry_jobs` 10); the coverage is set to 6 667, 10 000 and 13 333 basis points of the upkeep for S56; for S59 a `refill` endowment that the recipe draws on (`spec-resources.md` 18.1 is the shape); an empty row `[modules.subsistence.role.<id>]` for a role that is not to take the mode | S56 to S60 |
| `overlay/migration.toml` | the migration module on the trade overlay: `[modules.information]` and `[modules.migration]` with `move_speed` chosen against the edge lengths of `overlay/trade.toml` so that the shortest journey is 1 or 2 ticks and the longest under `CoverTicks`, no role `Fixed` (and `fixed = true` on the chain's roles for the second leg of S65), on `five` with the trade overlay (S61 uses two of its regions and S62 three); `spec-migration.md` 15.3 | S61 to S66 |
| `overlay/settlements.toml` | `[modules.settlements]` on `five`: the classes of `spec-settlements.md` 12.1 with a `merchant` role gated by the flag `fair` and a ceiling, and a role with a group in each region whose class carries or lacks the flag; the merchants module is on, so it is layered on `overlay/trade.toml`; `famine` is the base of S104; the base packs keep the module off | S102 to S104 (S105 and S106 are harness tests) |
| `overlay/storage.toml` | `[modules.storage]`: a base room for the households and the producers and one store class (`spec-storage.md`); the base packs keep the module off | S91 to S96 |
| `overlay/employment.toml` | `[modules.employment]` on `one`: an employer role with wants and a `Hireable` role with no recipe (`spec-employment.md` 22). The prices of a pack with employers sit above the solver's by the employers' margin and its warm start has no hire in flight (`spec-viability.md` 4.7, 8.1), so its bands are its own | S97 to S101 |
| `overlay/tenure.toml` | `[modules.tenure]`: titles on the `refill` endowment of `overlay/resources.toml` for `one` and `five`, and the closed case of `spec-land.md` 15.2; layered after `overlay/resources.toml` | S107 to S111 |
| `overlay/credit.toml` | `[modules.credit]` on `one`: a borrower role on the role with the 60-tick job (C04) and a lender role in one region, with its warm start (`spec-credit.md` 24) | S114 to S118 |
| `overlay/risk.toml` | `[modules.risk]`: route rows on the edges of `overlay/trade.toml` and one store row (`spec-risk.md` 16); layered after `overlay/trade.toml` | S126 to S131 |
| `overlay/infrastructure.toml` | `[modules.infrastructure]`: the facility classes of `spec-infrastructure.md` with initial rows held by a role, and a recipe with `Needs` (a site), a route class on an edge of `overlay/trade.toml` and a ground class on an endowment of `overlay/resources.toml` where a scenario uses them | S119 to S125 |
| `overlay/cover.toml` | `[modules.insurance]`: an insured role and an underwriter on the trade overlay with `overlay/risk.toml` (`spec-insurance.md` 17); it needs credit (`overlay/credit.toml`) or a faucet (`overlay/faucet.toml`) for the loan that pays a premium | S160 to S165 |
| `overlay/polity.toml` | `[modules.polity]` on `five`: a root jurisdiction over two child jurisdictions, one of three regions and one of two, ungoverned and with no money, so that every pack with it is exactly neutral (`spec-polity.md` 12.4.1) | S112 |
| `overlay/polity-governed.toml` | the same with a seat and `start_money` on the root (`spec-polity.md` 12.4.1); layered after `overlay/polity.toml` | S113; the base of the fiscal and policy overlays |
| `overlay/fiscal.toml` | `[modules.fiscal]` on `five`: a head tax, a duty and a payout, and on `three` a toll on a border with `overlay/trade.toml` (`spec-fiscal.md` 19); layered after `overlay/polity.toml` | S133 to S136 |
| `overlay/fiscal-governed.toml` | the same flows with the treasuries of the governed jurisdictions; layered after `overlay/polity-governed.toml` | S132 (every flow at scale 0), S133 to S136 |
| `overlay/institutions.toml` | `[modules.institutions]` on `five`: a body over the one skilled role of the hub region with places at 80% of its starting households, a `produce` right and a `train` right, and a control keyed by the jurisdiction of two child jurisdictions over one staple at levels given by S137's grid (`spec-institutions.md` 13.5); layered after `overlay/polity.toml` | S137 to S140 |
| `overlay/institutions-inert.toml` | the same bodies, rights and controls with `Places` 0 and both levels 0, so that S141 asserts neutrality | S141 |
| `overlay/policy.toml` | `[modules.policy]` on `five`: the actors and rules of `spec-policy.md` 14; layered after `overlay/polity-governed.toml` and `overlay/fiscal-governed.toml` | S155 to S157, S159 |
| `overlay/policy-directed.toml` | the actor that a consumer directs with `DirectPolicy` (`spec-policy.md` 15); layered after `overlay/policy.toml` | S158 |
| `overlay/saving.toml` | `[modules.saving]`: a `Saver` block on a producer role with a long job, on a role with a site recipe, a build recipe and a repair recipe, and on an employer role; S146 also layers `overlay/faucet.toml`; the build and repair legs need `overlay/infrastructure.toml` | S143 to S146 |
| `overlay/technology.toml` | `[modules.technology]`: a field on the growers of `one` and `three`, learning only, then each mechanism in turn (`spec-technology.md` 14.4) | S148 to S154 |

**Layering order of the later overlays.** The overlays above are layered after the variant, and among themselves in this order, so that each finds what it needs: `overlay/trade.toml`, `overlay/resources.toml`, `overlay/faucet.toml`, then `settlements`, `storage`, `employment`, `tenure`, `credit`, `risk`, `infrastructure`, `cover`, `polity`, `polity-governed`, `fiscal`, `fiscal-governed`, `institutions` (or `institutions-inert`), `policy`, `policy-directed`, `saving` and `technology`. It is the module order of `spec-config.md` 3.1, except that `overlay/polity-governed.toml` precedes `fiscal-governed` and `policy`, and `overlay/risk.toml` and `overlay/credit.toml` precede `cover`. A scenario layers only the overlays it names and those they need.

**The all-modules pack** of `spec-observability.md` 9.7 (item 2: the reference chain with every
module on, so that every event type occurs) is not a variant and is not a file: the
test builds it in memory as `five` with the overlays of this table that switch a module
on (estates, faucet, trade, resources, external, own-use, population, migration, and the fourteen later overlays of this table in the layering order above, so that every module of the library is on), and
the golden rendering of its events is `testdata/golden/all/events-all-modules.txt`. It names
its modules, so FR-REF-018 is not broken: no variant of 4.1 contains one unnamed.

**The five-region endowment sizing is hand-derived.** The capacities of `spec-resources.md` 18.2 (a slack set at 2.2 times and a binding set at 1.1 times the reference programme's draw) come from the census of `five`, the mean site factor of each region's `grow_staple` and an upkeep of 2.25 `staple` a household a tick; they are an order of magnitude and not a calibration. When `overlay/resources.toml` is built, REF replaces them with the capacities derived from the solver's exact flows of `five` (`Capacity = draw a year / (load * (RateBP / 10^4) * (1 - share))`), so that no autarky `lambda*` is lowered by a resource row in the slack set and the fixed factor acts in the binding set (RES 18.2; N-092). The sizes are placeholders until the solver has produced them.

**The three modules of wave 4 on the five-region pack (SUB, MIG, MER).** Each is an overlay and none changes a base document, so the viability of every variant is unchanged. *SUB:* an own-use recipe is `Sited` (`ErrOwnUseNotSited`), so in `five` the overlay gives a yield row in each region that is to offer the floor, and absence means the recipe cannot run there (`spec-subsistence.md` 3.2, 3.5); an own-use-only recipe is no activity of the reference programme (FR-SUB-037), so the census and the autarky `lambda*` of `five` do not change. The bound of `spec-subsistence.md` 4.2 compares the coverage of the recipe with the region's surplus factor (`SurplusBP`); the overlay is built with a coverage below it, and the per-region figure comes from the solver's report (`spec-viability.md` 10.3). *MIG:* the module needs the trade overlay (its edges give the distances) and `[modules.information]`; `five` has 5 000 households and no edges until `overlay/trade.toml`, and `spec-migration.md` 15.3 runs S61 to S66 on `five` with the two overlays (S61 uses two of its regions, S62 three). *MER:* a merchant role has no recipe and weight 1, so it joins a pack as an added role with a few agents and is in no census of the base documents; whether the solver accepts a role with no recipe in a pack is not stated by `spec-merchants.md` (open, 16 item 2). The numbers of all three overlays are placeholders that REF derives from the solver and the edge lengths when it builds the files.

`overlay/skill.toml`, in full, because it sets the traineeship of the two skilled roles:

```toml
[role.cook]
training_ticks = 120
max_trainees = 3
teaching_load = 1000
[role.crafter_a]
training_ticks = 120
max_trainees = 3
teaching_load = 1000
```

**The two eligible recipes** (C13). The trainer's gain per trainee-tick is `output_a - U_a
- TeachingLoad * W_trainer / 10 000` (`spec-skill-barriers.md` 4.3), with `output_a` the
value of the trainee's output (`TraineeYield` of a full execution) less its inputs
(`spec-production.md` 8.3). At the seeds of `one` (`prepared` 628, `staple` 96, `fuel` 179,
wage 2 512, a basket of 1 000 for the trainee's upkeep): `prepare_basic` makes 4 units
worth 2 512; at yield 7 000 the trainee's output is 1 758 less inputs 371, 1 387; the
gain is 1 387 - 1 000 - 251 = **+136**, so training pays, and its break-even yield is
6 457. `finish_a_basic` in region `centre` of `five` makes 30 `good_a` (value 3 330 at the
seeds) from 12 `semi_a` (inputs worth 6 432): even at yield 10 000 its output is below its
inputs (-3 102), the trainee's upkeep there is 1 326 and the trainer's wage 2 521, so
the gain at 3 000 is -5 433 - 1 326 - 252 = **-7 011** (-4 680 at yield 10 000), and
training never pays. The first is above the break-even by 10% of the trainee's upkeep,
the second far below it; the second is below it by construction, which is the case
`spec-production.md` 8.4 asks for. The first is close to break-even, so a price move of
a few percent turns it (STB-L37).

**The defect packs** are overlays on `one`, each with one defect and the rule that must
reject it (A2). The consequences listed come from the same defect:

| File | Defect | Rule that must reject it | Also reported |
|------|--------|--------------------------|---------------|
| `defects/d1_seed_cycle.toml` | `grow_staple` returns 30 staple for 40 seed, and the wild seed source is switched off | `cycle_not_productive` | `unproducible_in_realm` for seed, staple, prepared |
| `defects/d2_upkeep_unmade.toml` | the yield of `cut_fuel` is 0: no region makes fuel, and `fuel` is not marked `raw` in the overlay (it adds `[commodity.fuel]` with `raw = false`, because `base.toml` marks `fuel` raw and a raw good is sourced, `spec-viability.md` 4.2.1) | `unproducible_in_realm` | fuel, prepared, refined, staple, tool_a, tool_b; `role_cannot_work` for `forester` |
| `defects/d3_idle_role.toml` | a role with a recipe that can run nowhere, 10 households | `role_cannot_work` | nothing else |
| `defects/d4_long_job.toml` | `grow_staple` yields 90 (the first draft of `spec-production.md` 12.1 was of this kind) | `labour_short` (`lambda*` 0.219) | nothing else |
| `defects/d5_storage_gap.toml` | `staple` lasts 200 ticks against a supply gap of 304 | `storage_gap` | nothing else |
| `defects/d6_floor_full.toml` | `cost_floor = 10000` | the warning `cost_floor_at_full_cost`; no error | nothing else |

The text of `d1` to `d6` is in the files; each is six lines or fewer. `d1` is the one that
shows a trap (6.2): the wild source of seed makes the seed cycle productive however poor
the farm job is, so the defect needs the source off as well; with the source on, the
same farm job is caught by `labour_short` (`lambda*` 0.073).
## 5. Viability of each variant

Checked in the prototype of section 14 on the documents of section 4: V1 (sourced and
runnable), V2 (productive cycles, Hawkins-Simon), V3 (the reference programme) and V4
(storable), `spec-viability.md` 4. "Labour" is the share of its ticks that a household
must work to subsist at full labour (`1 / lambda*`); "nominal" is `lambda*` when the
available work share is the expected one, 250 / 365. All are placeholders (N-015).

| Pack | V1 | V2 (`s*` of the seed cycle) | V4 | Autarky `lambda*` (labour) | Nominal | Pooled `lambda*` (pivots) |
|------|----|------------------------------|----|-----------------------------|---------|---------------------------|
| `one` | pass | 18.7 | pass | 2.512 (39.8%) | 1.721 | 2.512 (10) |
| `two` | pass | 18.7 | pass | north 2.229 (44.9%), south 2.216 (45.1%) | 1.527, 1.518 | 2.626 (19) |
| `three` | pass | 18.7 | pass | north 2.512, south 2.060 (48.5%), far_south 2.666 (37.5%) | 1.721, 1.411, 1.826 | 2.666 (30) |
| `five` | pass | 18.7 | pass | north 1.915 (52.2%), east 1.704 (58.7%), south 1.642 (60.9%), west 1.805 (55.4%), centre 1.904 (52.5%) | 1.311, 1.167, 1.124, 1.237, 1.304 | 2.039 (202; 1.7 s in Python fractions) |
| `famine` | pass | 16.1 | pass | 1.315 (76.0%) | 1.112 | 1.315 (9) |
| `one-demand` | pass | 18.7 | pass | 2.040 with the wants as fixed demand | - | - |
| `second` | pass | board and fabricator: 0.5 | pass | 1.966 (50.9%) | - | 1.966 (10) |

The supply gaps of V4: `staple` has a gap of 304 ticks in the spring-only regions (123
and 304 where a region plants twice, 184 in the `west` pattern) against a shelf life of
700, so the check passes with room; `prepared`, `fuel` and the lines' goods are made
every tick (gap 0). `lambda*` for `one` is also the result of every other pack with the
same climate (`north` of `three`). The five-region pooled solution has 202 pivots; the
synthetic realm of `spec-viability.md` 15 had 306.

**Defects (A2).** Run on `one` with one overlay each:

| Pack | V1 | V2 | V4 | `lambda*` | Reading |
|------|----|----|----|-----------|---------|
| `d1_seed_cycle` | `unproducible_in_realm`: seed, staple, prepared | fails: `s*` 0 | pass | - | rejected by `cycle_not_productive` and its consequences |
| `d1` with the wild source on | pass | pass | pass | 0.073 | not a V2 case: the source makes the cycle productive; it is `labour_short` |
| `d2_upkeep_unmade` | `unproducible_in_realm`: fuel, prepared, refined, staple, tool_a, tool_b; `role_cannot_work`: forester | pass | pass | 0 | rejected |
| `d3_idle_role` | `role_cannot_work`: idle | pass | pass | 2.512 | rejected, nothing else |
| `d4_long_job` | pass | pass | pass | 0.219 | rejected by `labour_short` |
| `d5_storage_gap` | pass | pass | `storage_gap`: staple, gap 304, shelf 200 | 2.512 | rejected |
| `d6_floor_full` | pass | pass | pass | 2.512 | accepted; the warning `cost_floor_at_full_cost` |

**A5.** `two` with `overlay/a5.toml`: autarky `south` has `lambda*` 0 and V1 fails for
it (`prepared` and `staple` unproducible there), which is the warning
`region_not_self_sufficient`; `north` alone 2.229; pooled 2.590 and V1 passes. The
warning names the import: the staple.

**Warnings known by hand.** Not the full list: the first Go run fixes it (`expected/`).
`window_extended` for `grower` (effective window 120 against 30), informational, in
every variant. `target_above_surplus` in `five` (nominal 1.12 to 1.31 under the mean
target 1.5) and, per region, in `three` (`south`, 1.411) and `famine` (nominal 1.112);
none in `one` (1.721) and `two` (1.52). `subsistence_exceeds_expected_work` in `famine`
only (76.0%; the other labour shares are at most 61%, under 68.5%). `region_not_self_sufficient` in no
pack but A5. Whether `start_stock_below_target`, `tool_unaffordable` and the
`source_unbounded` family fire is decided by the Go validator; the packs were written to
avoid them and the golden file is the authority (FR-REF-005). `famine` does not load in Stage 1, and its programme at full labour has a tight capacity row, so the labour share is absent there; its warnings (`subsistence_exceeds_expected_work` at 76.0% and `target_above_surplus` at 1.112) are a Stage 2 matter.

`cold_start_lockstep` (`spec-world.md` 15.2, informational) is expected by every cold
pack (a variant without its warm-start document, 4.1) and also by a warm pack whose
warm-start document gives a role with a recipe of `Duration` above 1 no job row: the
`grower` role of `one`, `two`, `three`, `five`, `famine` and `one-demand` (4.4.1).
The list of `second` is the Go run's (FR-REF-005). `one` + `overlay/poor` (S20) also expects `reserve_exceeds_money` (FR-AGT-060): a group starts with 1 000 against a reserve of 3 000.

## 6. Findings about the specs this was built on

These came out of building the packs. In round 5 each was settled in the spec that owns
it; the text says where, and keeps the finding as the record.

6.1. **The worked examples of `spec-production.md` 12 have an unsourced cycle.** Example
A takes 60 `seed` to make 1 500 `grain`, and `save_seed` makes `seed` from `grain`. No
recipe makes `seed` or `grain` without the other, so the structural check of
`spec-config.md` 7.5 rejects the chain unless `seed` is marked `raw`, and V1 of
`spec-viability.md` 4.2 (which wants a *recipe* with no inputs or an endowment for a
raw commodity) may reject it even then. Example B has the same shape: `fab_board` needs
a `fabricator`, `make_fabricator` needs boards, and no recipe breaks the loop. Both
passed the viability prototype of `spec-viability.md` 15 because it did not run V1.
This fixture closes the loops with a wild source (`gather_seed`) and a slow salvage
recipe (`salvage`, 50 ticks). **Settled in round 5:** `spec-production.md` 12 gives both
examples the same kind of source and says they are checked by V1 (FR-PRO-046);
`spec-viability.md` 15 re-ran the prototype with V1 (both fail as first completed, both
pass with the source).

6.2. **A source that needs a tool whose chain needs the source is an unsourced cycle.**
`cut_fuel` and `dig_mineral` must be tool-free: a tool is made from `refined`, which
needs `mineral` and `fuel`, so a `dig_mineral` that wore a tool would be a cycle with no
way in. The same holds for `refine` and a tool made from `refined`. The rule is
`spec-config.md` 7.5 (a required tool is an input), and the error message should name
the cycle so an author can see it. **Settled in round 5:** the error names the cycle
(`spec-config.md` 7.5, FR-CFG-058) and `spec-production.md` 10.6 is the authoring note.

6.3. **V1 and the `Raw` marker disagree.** `spec-config.md` 7.5 counts a commodity marked
`Raw` as sourced; `spec-viability.md` 4.2 says a `Raw` commodity is sourced "with such an
activity or an endowment". For a raw commodity with no source recipe the two give
different answers. The packs avoid the question: every raw commodity has a source recipe,
and the seed (which is in a cycle) is not marked raw. **Settled in round 5:** the marker
makes a commodity sourced in both checks, and the warning `raw_without_source` says when
nothing real stands behind it (`spec-viability.md` 4.2.1, `spec-config.md` 7.5,
FR-VIA-059, FR-CFG-059).

6.4. **The wage dual can be 0 in a viable, ordinary pack.** In the one-region pack (staple yield
3 800 and below) the start-capacity row of `grow_staple` binds and the labour row
is slack, so the solver's wage is 0 and the seeds of goods that do not use the capacity
fall to the floor. `spec-viability.md` 5.3.4 treats this as a degenerate case, but it is
reached by lowering one yield. The table is then a poor start. **Settled in round 5:**
the seeds and wage of such a region come from the nominal programme, where labour binds
(`spec-viability.md` 5.3.4, FR-VIA-058); `famine` now uses a yield of 3 000 and shows it.

6.5. **The balanced census puts some roles below the scale floor.** In `five` the
smallest groups (the gatherer of a line) are 4 to 9 households. They are below `K`, so
each household is its own agent; a group of four is a thin market by construction.
`spec-scale.md` 5.2 allows it; the standard fixture accepts it and the scale-consistency
check S10 uses `one`, whose groups are 30 and over. The run-time warning
`group_below_floor` (`spec-config.md` 7.4.2, `spec-spawner.md` 12) concerns groups of
weight above 1 and does not fire in a weight-1 world.

6.6. **The closed upkeep-only world is a glut by construction.** The census gives every
role 1.8 times the households a basket needs (`one`: 600 households where 331 would do),
because the apportionment shares out all the households. Nothing buys the surplus
(`spec-viability.md` 4.7) until the demand overlay is on; S19 is the scenario that
shows its size. *Owner:* none new (N-054 stands).

## 7. The scenarios

### 7.1 Rules common to every scenario

7.1.1. **Packs and runs.** A scenario names a pack and a list of overlays; the test
builds the world from those documents (`config.LoadFS`) and a seed, and runs it with the
default strategy and the default spawner. The documents are the files of 4; a scenario
that varies one scalar generates a one-key document in memory (for example
`[world]` with `cost_floor = 9000`), which is also a pack, or is a `Change` of an
experiment (`spec-experiments.md` 3.1: a path and an integer value, `Seed0` and
`Replicas` for the seeds), which is the same one-key document built by the harness.

7.1.2. **Seeds and lengths.** Section 8.2 fixes them: a band scenario runs the seeds 1 to
10, an invariant test the seeds of its row, the golden test seed 1. A failure message
names the scenario ID, the pack, the seed and the measure (testing.md 6.2).

7.1.3. **Shocks and agents.** A shock is a line of `scripts/<scenario>.toml`, applied
between two steps through the integration calls of `spec-integration.md` (FR-STB-014:
`SetYieldShocks`, `AddAgent` and `Direct`), never by editing state (the Stage 1 exception is below):

```toml
[[shock]]            # sets the yield multiplier of spec-production.md 6.8
tick = 400           # applied before the step of this tick
region = "north"
recipe = "cut_fuel"
yield_bp = 5000      # 10000 clears it
[[buyer]]            # an external agent of the role visitor (spec-agents.md 10)
tick = 400
region = "north"
money = 2000000      # created by `initial` (spec-ledger.md 7.2.1)
buy = { commodity = "fuel", quantity_share_bp = 500, ticks = 60 }   # a sticky directive; the share is of the market's mean volume over the 100 ticks before
```

The format is this spec's: INT fixes the calls, not a file format (`spec-integration.md` 9.1).
A `[[shock]]` is one entry of `SetYieldShocks`; a `[[buyer]]` is an `AddAgent` with
`money` followed by a `Direct` with `Sticky`. The `ticks = 60` is the harness's: a
directive applies to the next step only unless it is `Sticky` (FR-INT-027), so the
harness empties the mailbox after 60 ticks with a `Direct` of a directive with
nothing in it (FR-INT-026).

**Stage 1 seam.** `integration.go` is Stage 2, so in Stage 1 a `[[shock]]` is applied
between steps by the world row's unexported yield-shock setter (the function
`SetYieldShocks` will call), reached through `export_test.go`; the format of
`scripts/<scenario>.toml` stays as above. A `[[buyer]]` is not buildable until `AddAgent`
and `Direct` exist (N-296). This is an accepted Stage 1 exception to FR-REF-021 and
FR-STB-014; Stage 2 replaces the seam with the public call.

7.1.4. **Measures** are computed from events, the snapshot, INT's `Agent(a)` queries
and IND's presets (`spec-indicators.md` 9) only (FR-STB-012; FR-REF-027) by a
recording observer in a `_test.go` file (a test that imports `indicator` is in
`indicator`'s external test package, 17). The names used below:
the *price series* is `MarketCleared.Price` per region, commodity and tick; the *volume
series* its volume; *pinned* is the condition of the `Warn` "price beliefs at bound"
(more than half of a market's outcomes at a bound, FR-BEL-031); *households* is the sum
of the `Weight` of `AgentSpawned` less `AgentBankrupted` and the changes of `WeightChanged`
(observability-and-logging.md 4.3); *unfed* is `UpkeepShortfall`; *relief* is
`ReliefPaid`; *spoiled* is the `BoundaryFlow` of the reason `spoilage`.

7.1.5. **Bands.** A band is a number that a measure must fall inside. It lives in
`bands/<pack>.toml`, one table per scenario, and is set by `spec-stylised-facts.md`
(FR-SKL-025):

```toml
[s14]
expects = ["r3"]       # a regime the scenario must show (STY 6.4); absent for most
[s01.recover_ticks]
unit = "ticks"         # no `value` line: unset (a value of 0 is a value, STY 6.5.1)
# value = 90           # present once a reviewed run has set it
# set_by = "commit=<id>;config=<hash>;seeds=1-50"
```

The file has one table per key, named `<scenario>.<key>`, and a scenario table that
holds its `expects`. A key with a `value` has a `set_by` of the form
`commit=<id>;config=<hash>;seeds=<a>-<b>`: the run that set it, reviewed by a human
(FR-STB-015 asks that each pack be run and recorded; the review is STY 8.4). A
scenario asserts a band only when its key has a value. A key with no value makes the
test **report** the measure (a `t.Logf` line the CI keeps) and assert the exact checks
and the O checks (a sign, an order or a derived bound, STY 3.2) only. This is how the
suite runs before STY has data, and it is the only use of a missing band; a band is
never set from fewer than the seeds of 8.2 and never from the run of the code under
test alone without review (FR-REF-024).

Two kinds of key have no scenario table of their own: the thresholds of the regimes
(`r1.rise_bp`, `r8.unmet_bp` and the others of STY 6.5) are keys of
`bands/regimes.toml`, one file for every pack (STY 6.5.1), and the keys of facts that
no scenario of this spec holds are in the tables of 7.10, named by fact ID.

7.1.6. **Level.** Every scenario is an integration test or an end-to-end test and begins
with the `testing.Short()` skip (testing.md 10.5, FR-STB-007). The level is stated in
each entry: *integration* (a few hundred ticks, one small pack) or *end-to-end* (a
year or more over many seeds, or the five-region pack).

7.1.7. **Traceability.** The requirement IDs of each entry are the `Requirement:` line of
its test (testing.md 4.2). A scenario that asserts a band cites the requirement of the
stability entry it exercises (FR-STB-009 to FR-STB-011 and 018 to 024 for the ones STB
states) and the registered loop (STB-Lnn) in its comment.

### 7.2 The register's scenarios, S01 to S22 (S15 is unused)

Each entry: the packs; the setup; the measure; the form of the pass or fail (a band key is
`<id>.<name>`); the requirements; the level.

**S01. A shock to one commodity, one region.** *Packs:* `one`. *Setup:* 700 ticks. Two
legs, run separately. (a) Supply: `cut_fuel` at 5 000 bp from tick 400 to 459 (in Stage 1 through the seam of
7.1.3). (b)
Demand: from tick 400 for 60 ticks a visitor buys `fuel` at 5% of its market's mean
volume over ticks 300 to 399 (leg (b) and the visitor wait for INT, N-296). *Measure:* the price series of `fuel` in `north`; `P0` the
mean over ticks 300 to 399; the *recovery time* the first tick after the shock at which
the mean price over ten ticks lies within `s01.band_bp` of `P0` and stays there to the
end; the *long-run offset* the gap in basis points between the mean over the last 100
ticks and the seed price of the warm start (the flow-balancing price of the solver);
and the *unmet need*, the share of household-ticks of the window that were unfed
(`Unfed`, `spec-indicators.md` 9 row 12, in basis points).
*Pass:* recovery time at most `s01.recover_ticks`; offset at most `s01.offset_bp`, and
unmet need at most `s01.unmet_bp` (a signed offset beyond its band with unmet need
above its bound is the regime R8 of `spec-stylised-facts.md` 6.2; a negative offset
alone fails the band and is not a regime); in
both legs no price leaves its belief bounds (FR-STB-008). *Requirements:* FR-STB-018,
FR-STB-008, FR-BEL-014, FR-BEL-036, FR-BEL-034, FR-CLR-019, FR-CLR-020, FR-AGT-053 to
055 (the visitor). *Loops:* STB-L01, L02, L04. *Level:* integration.

**S02. A stale price with a bid and an ask.** *Packs:* none of the chain. The case is a
world built in Go in the test with one commodity, two external agents and a gap `g`
between their limits and a stale published price; the chain would add nothing. *Measure:*
the number of adjustments `n` until the bid and the ask cross. *Pass (exact):* `n <=
ceil(g / Delta)`, `Delta` the unit of adjustment `max(1, Centre * BeliefStep / 10 000)`.
*Requirements:* FR-BEL-034, FR-BEL-010, FR-BEL-011. *Loop:* STB-L03. *Level:* integration.

**S03. A market pinned at a cost floor.** *Packs:* `one`. *Setup:* 600 ticks. `cut_fuel` at
20 000 bp from tick 100 to 399, so fuel is in glut and its sellers' belief falls to the
cost floor of their full unit cost. *Measure:* the log records of the `Warn` "price
beliefs at bound" for `fuel` in `north`, and the BeliefsUpdated counts at the floor.
*Pass (exact):* the `Warn` is emitted at the tenth consecutive pinned tick and each tenth
after, once per market, never per agent (FR-BEL-031); the market is reported pinned (F5)
for no longer than `s03.pinned_ticks`. *Requirements:* FR-BEL-031, FR-BEL-004, FR-BEL-030.
*Loops:* STB-L03, L09. *Level:* integration.

**S04. A staple role and a shortage.** *Packs:* `one`, and `one` + `overlay/nofloor`.
*Setup:* 1 400 ticks. `grow_staple` at 5 000 bp from tick 424 to 515 (a half harvest in
the second spring). *Measure:* the staple price series; the longest run of ticks the
staple price sits at a bound (pinned); the grower role's share of working ticks; and the
**measured demand elasticity** of the staple: the arc elasticity
`rdiv(10 000 * (q1 - q0) * (p1 + p0), (p1 - p0) * (q1 + q0))`, in basis points, between
the ten-tick mean price `p` and quantity bought `q` before (`q0`, `p0`) and during
(`q1`, `p1`) the shock, integer arithmetic only (`spec-stylised-facts.md` STY-W05). *Pass:* the price
stays inside its belief bounds at every tick (exact, FR-STB-008); the longest pinned run
is at most `s04.pinned_ticks` (F5); the elasticity is **recorded** with the pack
(FR-STB-009) and asserted to be in `[0, s04.elasticity_max_bp]` (it should be 0 to 1,
`spec-agents.md` 11.6). The helper reports the elasticity signed (negative for a
shortage) and the band check compares its absolute value. In Stage 1 the shock is applied
through the seam of 7.1.3. The run without the floor (`min_work_share` 0) shows the other
outcome of `spec-stability.md` 6.1 and is recorded beside it. *Requirements:* FR-STB-009,
FR-STB-008, FR-AGT-072, FR-WEL-027, FR-WEL-021 to 023. *Loops:* STB-L14, L15 to L18, L44,
L51. *Level:* end-to-end.

**S05. A delayed supply response.** *Packs:* `one`. *Setup:* 1 610 ticks (four seasonal years from tick 0; tier G, `spec-stylised-facts.md` 4.5). From tick 400
for 180 ticks a visitor buys `staple` at 10% of its market's mean volume. The response is
a 60-tick job that can start only in the second spring (ticks 424 to 515). *Measure:* the
price range (maximum over minimum of the daily price) of `staple` in the shocked season
of four successive years, one point per year (`SeasonWindows`, `spec-indicators.md` 10.2.2, FR-IND-052). *Pass:* the range of each window is not larger
than the one before by more than `s05.range_growth_bp`, or stays under `s05.range_factor`;
otherwise the pack is reported as F6 (cobweb blow-up). *Requirements:* FR-STB-010,
FR-BEL-014, FR-AGT-035, FR-AGT-036. *Loops:* STB-L21, L48, L51. *Level:* end-to-end.

**S06. An income gap between two roles, tool and traineeship on and off.** *Packs:* `one`
+ `overlay/imbalance-cook`, with the four combinations of `overlay/skill` (traineeship)
and `overlay/notool` (the cook's tool removed). *Setup:* 1 095 ticks; the cooks start 20%
above the balanced census (163 against 136). *Measure:* the cooks' share of the region's
households over time against the balanced share (2 267 bp); the largest swing (maximum
less minimum of the share over the last 730 ticks); the observed premium of `cook` over
the unskilled roles (`forester`, `digger`) (FR-WEL-019); the traineeship counters
(FR-SKL-021, FR-SKL-022). *Pass:* the share returns to within `s06.share_band_bp` of the
balanced share and the swing is at most `s06.swing_bp` (F6), in each combination; with
the tool the premium is above 10 000 and rises with the tool's price, with the tool off it
is within `s06.free_tool_tol_bp` of 10 000 (FR-SKL-023); the whole-series band is
reported only (FR-SKL-024). The trainer takes `prepare_basic` apprentices (gain +136) and
no `finish_a_basic` ones: the second recipe is not on the cook's list in `one`, so that
leg of C13 is exercised in `five` (X3). *Requirements:* FR-SKL-005, FR-SKL-010 to
FR-SKL-017, FR-SKL-023, FR-SKL-024, FR-SPN-018 to FR-SPN-021. *Loops:* STB-L36, L37.
*Level:* end-to-end.

**S07. A perishable glut with a hard cost floor (N-039).** *Packs:* `one` +
`overlay/glut-prepared` (736 households, twice the cooks), run with `cost_floor` 10 000,
9 000, 7 000 (the default) and 5 000. *Setup:* 400 ticks. *Measure:* the share of ticks
with a trade in `prepared` in `north`; the share of `prepared` made that spoiled (the
`spoilage` flow over the output of `RecipeExecuted`); the longest run of ticks with no
trade while both sides have unmatched orders (F5). *Pass:* at the default floor the
longest freeze is at most `s07.freeze_ticks` and the share spoiled at most
`s07.spoiled_bp`; the other floors are **recorded** as a curve beside it (the curve of
`spec-stability.md` 6.2 run 5: the market freezes whenever the floor is above what
buyers can pay). *Requirements:* FR-STB-011, FR-BEL-038, FR-BEL-004, FR-GDS-009,
FR-GDS-010. *Loops:* STB-L09, L25. *Level:* integration.

**S08. A cost shock to a raw input passes through the chain.** *Packs:* `one`. *Setup:*
700 ticks; `dig_mineral` at 5 000 bp from tick 400 to 519. *Measure:* the series of
`mineral`, `refined`, `tool_a`, `tool_b` against their means over ticks 300 to 399; the
time to settle within `s08.band_bp`; the cycle gain of every cycle of the pack.
*Pass:* every price settles within `s08.settle_ticks` of the end of the shock; **exact:**
`Warnings()` of `one` holds no `cycle_cost_gain` and no `cost_floor_at_full_cost`, the
cycle gain of the seed cycle is `CostFloor / 10 000 * 40 / 1 800` (0.0156 at 7 000) and
is below 1, and `d6_floor_full` gives `cost_floor_at_full_cost` and no error.
*Requirements:* FR-STB-017, FR-BEL-038, FR-CFG-044, FR-PRO-035 to FR-PRO-037. *Loops:*
STB-L10, L28. *Level:* integration.

**S09. A market with no sellers.** *Packs:* `one`. *Setup:* 700 ticks; `cut_fuel` at 0 bp
from tick 400 to 549. *Measure:* the price series of `fuel`; the `Warn` records.
*Pass (exact):* the price reaches the top of its belief bounds and never exceeds them
(FR-STB-008); the `Warn` "price beliefs at bound" is emitted once per ten ticks for that
market and not per agent. *Requirements:* FR-BEL-002, FR-BEL-004, FR-BEL-031. *Loops:*
STB-L05, L08. *Level:* integration.

**S10. The same chain at scales 1, 4 and 10.** *Packs:* `one` + `overlay/h200` (200
households in each of six roles, 1 200 in all), with `world.scale` 1, 4 and 10 and
`min_agents` 10 (1 200, 300 and 120 agents). *Setup:* the warm start of `one` (the warm layer `warm-one.toml`), which is per
household, so the totals of money and goods are the same at every scale (FR-SCL-007);
365 ticks from tick 0 with no settling. *Measure:* the time-averaged price of each
commodity at scales 4 and 10 against scale 1; the dispersion of belief centres (the
difference between `HighCentre` and `LowCentre` of `BeliefsUpdated`, as a share of the
mean) per market. *Pass:* the averaged price at each scale stays within `s10.tolerance_bp`
of its value at scale 1 (the placeholder is 10%, `spec-scale.md` 7.2); the dispersion
stays above `s10.dispersion_min_bp` (both keys are declared in `bands/one.toml` and unset until STY gives them values). **Exact:** the weights of each group add up to 200
(FR-SCL-003) and money and goods at tick 0 are equal at the three scales. *Requirements:*
FR-SCL-002, FR-SCL-003, FR-SCL-007, FR-SCL-010, FR-SCL-016, FR-SCL-017, FR-AGT-052. *Loops:* STB-L13,
L33, L34. *Level:* end-to-end.

**S11. A healthy producer with a job longer than the window.** *Packs:* `one` +
`overlay/estates` (so relief exists to be wrongly paid). *Setup:* 400 ticks. *Measure:*
for the growers, `Unfed`, relief paid, the effective window, and the welfare ratio.
*Pass (exact):* from tick 0 to 399 no grower is unfed, no relief is paid to a grower, the
effective window of `grower` in the snapshot is 120 and that of the other roles is 30,
`window_extended` was reported for `grower` only, and the ratio of a grower is never
reported absent. *Requirements:* FR-WEL-009, FR-WEL-001, FR-WEL-007, FR-AGT-014,
FR-AGT-015, FR-EST-017, FR-EST-018. *Loops:* STB-L19, L20. *Level:* integration.

**S12. A delay near the season length, and a coarse tick.** *Packs:* `one` +
`overlay/long-grow` (a job of 85 ticks against a planting window of 92). *Setup:* 1 460
ticks (four years). *Measure:* the yearly range (maximum over minimum) of the `staple`
price, in each of four years. *Pass:* the range in each year after the first stays within
`s12.range_factor` and grows by at most `s05.range_growth_bp` from year to year (S12 has no key of its own). The **coarse-tick leg** (a
calendar of five days a tick) is not built: it needs the pack restated with the
durations and per-tick quantities rescaled (`spec-calendar.md` 6.1, FR-CUR-004) and a
choice of `ExpectedWorkTicks`; it waits for CAL and STY (7.6). *Requirements:* FR-PRO-021,
FR-CAL-009, FR-CAL-019, FR-CAL-020. *Loops:* STB-L23, L24, L28. *Level:* end-to-end.

**S13. A faucet and no sink.** Defined by `spec-money-supply.md` (17.6, 16). *Packs:* `one` + `overlay/faucet.toml` (a grant flow with no sink), and `one` + `overlay/faucet-sunk.toml` (the same with a rate levy on the same role). *Pass:* the exact accounting holds (the boundary balance of the flow's declared reason equals the money granted, and the money supply moves by exactly that; FR-MON-043 and the identities of `spec-ledger.md` 9.1); the price drift is **reported** and has no band until STY exists. The requirements it serves are FR-EST-016 (money enters only by a named flow) and the `BoundaryFlow` of the flow's reason. *Loop:* STB-L30 (merged with L69).

**S14. A region that loses its income.** *Packs:* `famine` (the estates module is in the
pack). *Setup:* 1 800 ticks. The spring of the second year is hit by a harvest failure of
depth `m` in `{7 000, 5 000, 2 500, 0}` bp: `grow_staple` at `m` from tick 424 to 515.
Four runs, one script each: `scripts/s14-7000.toml`, `scripts/s14-5000.toml`,
`scripts/s14-2500.toml` and `scripts/s14-0.toml` (7.1.3); the bands are
`bands/famine.toml`, with `s14.recover_band_bp`, `s14.recover_ticks` and `s14.fall_bp`
declared and unset until STY gives them values (7.1.5); the outcomes are written to
`expected/famine.scenarios.txt`. *Measure:* the households (sum of weights) over time as a share of the start;
the unfed households and the longest run of ticks with any unfed (`UpkeepShortfall`);
relief paid and the pool balance; the removals by cause; the tick at which households
return to within `s14.recover_band_bp` of the pre-shock level, if they do. *Pass:*
for each depth, either the households recover within `s14.recover_ticks`, or the run is
classed as collapse (F3) and **the collapse is of the stated kind**: households fall by
at most `s14.fall_bp` and the ladder rungs act in order (FR-SPN-013 to 015). Exact:
no balance or inventory is negative and goods and money are conserved at every tick; the
pool is never overdrawn (FR-EST-022). The depth below which the region does not recover
is **reported** as the result of the scenario. A leg on `famine` with the population overlay (4.5) is the longer horizon of the recovery: tier L (`spec-stylised-facts.md` 4.5), and it waits for POP and the code. The scenario is tier G. *Requirements:* FR-STB-005 (coverage),
FR-EST-014 to FR-EST-020, FR-AGT-013 to FR-AGT-017, FR-SPN-012 to FR-SPN-016, FR-SPN-024
to FR-SPN-026. *Loops:* STB-L11, L31, L35, L38, L39, L50. *Level:* end-to-end.

**S16. Price-inelastic flows with an initial overhang.** *Packs:* `one` +
`overlay/overhang` (growers start with three times the staple). *Setup:* 1 095 ticks.
*Measure:* the share of ticks in the last year in which the `staple` price is at a bound.
*Pass:* the share is at most `s16.pinned_share_bp`; otherwise the run is reported as F5
and the pack rejected for this default set. *Requirements:* FR-AGT-036, FR-AGT-070,
FR-BEL-004. *Loops:* STB-L06, L44. *Level:* end-to-end.

**S17. A durable's replacement from a uniform starting age.** *Packs:* `one` warm, and
`one` cold (no warm-start document) as the control. *Setup:* 120 ticks. *Measure:* the
number of `OrderPlaced` bids for `tool_b` per tick; the ratio of its maximum to its mean
over ticks 0 to 119. *Pass:* in the warm run the ratio is at most `s17.spike_factor`; in
the cold run (every cook starts with one new tool) the ratio is **larger** than the warm
run's by `s17.control_gap_bp`, so the measure can see a spike. Exact: the remaining
durability of the `tool_b` items of the cooks at tick 0 is `8 - floor(i * 8 / n)` for
agent `i` of `n` (FR-VIA-038). *Requirements:* FR-VIA-037, FR-VIA-038, FR-GDS-014,
FR-SKL-002. *Loops:* STB-L26, L27. *Level:* integration.

**S18. The warm start of a long-job chain (VIA A8).** *Packs:* `one`. *Setup:* the first
3 ticks. *Measure:* `RecipeExecuted` completions per tick of `refine` (duration 2),
`make_tool_a` and `make_tool_b` (duration 3). *Pass (exact):* at the stagger of
`spec-viability.md` 8.4.3, `refine` completes 10 and 10 in ticks 0 and 1 (20 busy agents),
`make_tool_b` 10, 9 and 9 (28 busy), none above twice the mean (the placeholder of A8).
*Cold leg* (`spec-world.md` WLD-S3: a cold start with a recipe of `Duration` above 1 shows
the pulse the stagger removes): `one` without `warm-one`, the same 3 ticks and the same
measure, **reported** and not asserted, with the warning `cold_start_lockstep` expected
(5). WLD-S3 asks a recipe of `Duration` 10 over 40 ticks; the pack has none, so the leg
uses `refine` (2) and `make_tool_a`, `make_tool_b` (3) over the 3 ticks; a `Duration`-10
leg waits for a pack that REF does not define yet.
*Requirements:* FR-VIA-034, FR-VIA-035, FR-VIA-037, FR-PRO-013. *Loop:* STB-L22. *Level:*
integration.

**S19. The default `TargetRatio` in a closed upkeep-only economy.** *Packs:* `one` with
the halt on (default) and off (`produce_halt_ratio` 1 000 000, a document generated by
the test). *Setup:* 1 095 ticks. *Measure:* the aggregate ratio (`HouseholdIncome` of `spec-indicators.md` 9, when `OK`) and the
household-weighted mean welfare ratio per role beside it, over the last 365 ticks; the
share of money in the pool and escrow, `(M_p + M_c) / M`; the unsold stock per produced good
at the end, as a share of the output target of one tick. *Pass:* the aggregate ratio is
within `s19.ratio_band_bp` of 10 000 (the identity of `spec-viability.md` 4.7), the share
of money in the pool and escrow at most `m05.pool_share_bp`; `HouseholdIncome` is `OK`
false in version 1 (`spec-indicators.md` 9 row 21: the income fields are a future ask of
`spec-observability.md`), so until they exist the ratio band is **reported** with the mean
of ratios beside it and not asserted (`spec-stylised-facts.md` STY-M03, class R). In Stage 1 the
measures are `RoleRow.MeanRatioBP` (the mean ratio per role), `HeldByAgents` and
`HeldByEstate` of the market rows and `ProductionHalted`; the aggregate-ratio band is
reported only. With the halt on the unsold stock of
each produced good stays below `s19.stock_bound_bp` of the halt threshold; with the halt
off the stock of at least one good grows past that bound (the measure can see the glut).
*Requirements:* FR-STB-021, FR-AGT-070, FR-WEL-032, FR-WEL-031, FR-AGT-066. *Loops:*
STB-L46, L44. *Level:* end-to-end.

**S20. Starting money below the reserve.** *Packs:* `one` + `overlay/poor` (1 000 money per
household in every group). *Setup:* 400 ticks. *Measure:* the tick at which each group's
mean money first reaches `ReserveTicks` times its upkeep cost (3 000). *Pass:* a net earner
reaches it within `s20.ticks`, or the world reports the unreachable reserve by a `Warn`
(FR-STB-022; the `Warn` `reserve_exceeds_money`, FR-AGT-060, is also asserted at
construction as its own exact check); no household is removed by the ladder in this run
unless it is unfed for `RemovePrivation` ticks. The B part (a net earner reaches the
reserve within `s20.ticks`) needs a pack in which a group starts below the reserve and
earns, which a closed economy cannot give by a rise of the mean: it is tier N and waits
(7.6). *Requirements:* FR-STB-022, FR-AGT-040 to FR-AGT-042, FR-AGT-060.
*Loops:* STB-L45, L11. *Level:* integration.

**S21. Seed prices against the cost band.** *Packs:* every variant except `famine`
(whose capacity rent makes the seeds unequal to the full cost) and the defects. *Setup:*
no run; the check reads the warm-start table and the recipes. *Measure:* for each region and
each commodity with a known unit cost, the seed against `UnitCost * CostFloor` and
`UnitCost * CostCeiling`, the unit cost being the full cost at the solver's wage rounded
up (FR-VIA-027). *Pass (exact):* every seed lies inside the band, or the solver's report
lists it; for `one` the seed over the full unit cost is within 1% of 1 (FR-VIA-054); for
`famine` the table comes from the nominal programme (`spec-viability.md` 5.3.4), so the
report lists the commodities whose seed is not the full cost at the nominal wage.
*Requirements:* FR-STB-023, FR-VIA-027, FR-VIA-054, FR-PRO-035. *Loops:* STB-L52, L09.
*Level:* integration.

**S22. A perturbed chain from a warm and from a cold start.** *Packs:* `one` +
`overlay/perturbed` (`prepare` makes 6), with `warm-one` (stale: solved for 8) and without
it (cold). *Setup:* 1 095 ticks. *Measure:* the failure regime of `spec-stability.md` 7
(none, or one of F1 to F8) in the last year of each run, by the classifier STY writes, and
the price path of the first 100 ticks. *Pass:* both runs end in the same regime (FR-STB-024).
The two paths are recorded: the warm start must not make the chain look better than it is.
*Requirements:* FR-STB-024, FR-VIA-047, FR-VIA-040. *Loops:* STB-L53, L14, L21. *Level:*
end-to-end.

### 7.3 The viability scenarios, A1 to A10

**A1. One region, the reference chain.** *Packs:* `one`. *Pass:* `lambda* >= 1` (it is
2.512); every active activity is within `ResidualWarnBP` of the wage times its duration
(the largest residual in the report, FR-VIA-025); the warm table loads; **a warm run keeps
each price within `a1.price_band_bp` of its seed for the first 30 ticks** (the window of
the welfare ratio) **and a cold run does not do better** (the cold run is the same pack
without `warm-one`); the seed over the full unit cost is within 1% of 1. *Requirements:*
FR-VIA-003, FR-VIA-013, FR-VIA-025, FR-VIA-031, FR-VIA-047, FR-VIA-054. *Level:*
integration.

**A2. The defective variants.** *Packs:* `one` + each of `defects/d1` to `d5`; `d6` is the
warning case. *Pass (exact):* each is rejected with `errors.Is(err, ErrNotViable)` and a
`*ViabilityError` whose rule is the one of the table in 5; `d3` and `d4` and `d5`
report that rule only; `d6` loads and carries `cost_floor_at_full_cost`. *Requirements:*
FR-VIA-004 to FR-VIA-008, FR-VIA-011, FR-STB-017, FR-BEL-038. *Level:* unit (the
validator on the documents; no run).

**A3. Three regions with different yields.** *Packs:* `three`. *Pass (exact, from the
solver):* each region's autarky `lambda*` differs (2.512, 2.060, 2.666); the pooled
solution has activities active in a strict subset of the regions (all eight); the pooled
relative prices lie in the range of the autarky ratios or are listed
(`ratio_outside_autarky`, FR-VIA-045). *Requirements:* FR-VIA-043 to FR-VIA-045,
FR-VIA-013, FR-PRO-028. *Level:* unit/integration (the solver).

**A4. The second setting.** *Packs:* `second`. *Pass:* the same checks as A1 hold with no
Go specific to it (V1 to V4, solve, table loads, a run satisfies the invariants and the
stable-state predicate of X4). *Requirements:* FR-VIA-048, FR-PRO-046, FR-CFG-014,
FR-CFG-015. *Level:* integration (X4).

**A5. A region that cannot make a staple.** *Packs:* `two` + `overlay/a5`. *Pass (exact):*
autarky of `south` fails, pooled passes, and the warning `region_not_self_sufficient`
names the staple as the import. *Requirements:* FR-VIA-009, FR-VIA-043, FR-VIA-044.
*Level:* unit.

**A6. Alternative optima and a slack region.** No pack: it is a unit test of the solver on
a hand-made programme (a degenerate or tied one), which `spec-viability.md` 5.3 describes.
The chain's tools are interchangeable between regions in the pooled solution of `two` and
`three`, so the report is expected to flag `AlternativeOptima` there; this was not
computed by the prototype. A slack-labour pack could not be built from this chain, because
every unsited recipe can run anywhere (a region's labour always has a use). *Requirements:*
FR-VIA-019. *Level:* unit.

**A7. Warm-start conservation.** *Packs:* `one`, `five`, `second`. *Pass (exact):* the
audit passes at tick 0 and every tick after; construction money equals the sum over groups
of households times `Money` (24 638 321 in `one`, 202 868 377 in `five`); the goods equal
the stocks plus the inputs of the jobs in progress. *Requirements:* FR-VIA-032, FR-VIA-036,
FR-LED-035, FR-LED-045, FR-LED-046. *Level:* integration.

**A8.** Is S18. **A9. Determinism of the solver.** *Packs:* every variant. *Pass (exact):*
two solves on two architectures give the same table bytes and hash (FR-REF-007).
*Requirements:* FR-VIA-020, FR-VIA-022. *Level:* integration, on two architectures.
**A10. Cost.** *Packs:* `five`. *Pass:* `SolveSteadyState` on the five-region pack (30
commodities, 45 recipes, 202 pivots in the prototype) finishes in under `a10.seconds`
(placeholder, measured in the slice) and has a benchmark with `b.ReportAllocs()`.
*Requirements:* NFR-VIA-002, NFR-VIA-003. *Level:* benchmark.

### 7.4 The scenarios this spec adds, X1 to X8

**X1. Two regions, arbitrage.** *Packs:* `two`. *Static part (runnable now):* in autarky
the staple to fuel ratio differs by a factor of at least 2 between the regions (it is
3.2); pooled it lies between them or is flagged. *Dynamic part (needs the code; REG, TRN, INF and MER are written):*
with an edge, a carrier and a merchant role (an overlay `overlay/trade.toml`: its edge part is `spec-regions.md` 12.4, its modes, haul recipe and consignee follow `spec-transport.md` 3, 17.4, the merchant and the consignee are MER's (`spec-merchants.md` 19.1 shows the keys: `[modules.merchants]`, a role with `strategy = "merchant"` and a `[role.<id>.merchant]` block, a consignee role with `consigns`) and the rumours are INF's (`spec-information.md` 12.1)), the fuel price in the two regions converges: after `x1.warmup` ticks the
gap between the two is at most the transport cost of the edge plus `x1.margin_bp`, and the worst lane at most `x1.worst_bp`, in each of two
successive years, in the last year. Until then it is a skipped test with the issue ID of the missing specs
(FR-REF-038). *Requirements:* FR-VIA-043 to FR-VIA-045, FR-CAR-003 (hired carriage is an ordinary market of the haul unit, `spec-transport.md` 8), FR-TRN-005 (travel time) and FR-REG-* for the edge.
*Level:* integration (static), end-to-end (dynamic).

**X2. Three regions, comparative advantage.** *Packs:* `three`. *Static:* A3. *Dynamic
(needs trade and the spawner):* with trade on, the households of the region with the best
yield move toward the activities the pooled solution assigns to it; each region's
occupation shares move toward the pooled census within `x2.share_band_bp` over four
years. Skipped until the code exists (REG, TRN, INF and MER are written; it needs `overlay/trade.toml` with the merchant role of `spec-merchants.md` 19.1). *Level:* end-to-end.

**X3. The first consumer's world.** *Packs:* `five` (cold, and warm). *Setup:* 400 ticks,
seeds 1 and 2; and `five` + `overlay/skill` for the second eligible recipe. *Pass:* the
invariants of 7.5 hold; the per-tick cost is **reported** against the budget of
`spec-scale.md` 6.3 (53 ms for 5 000 agents, a placeholder) and asserted below
`x3.ms_per_tick`; the number of active markets is 85 and the mean tradable set 7.7
(computed from the configuration); `finish_a_basic` is refused to the trainers of
`crafter_a` for `gain_negative` and `prepare_basic` is accepted (FR-SKL-016, 12). *Requirements:*
FR-SCL-016, NFR-SCL-001, NFR-SCL-002, FR-SKL-016, FR-SKL-022. *Level:* end-to-end.

**X4. The configuration-only chain test (testing.md 6.4).** *Packs:* `second` (warm).
*Setup:* the pack is loaded from `second/` alone, with no Go specific to it, built and run
for 1 095 ticks. *Pass:* the **stable-state predicate** holds over the last 365 ticks:
(a) the invariants of 7.5; (b) every tradable commodity that has a producer trades in at
least `x4.trade_share_bp` of the ticks (the paper's criterion is 80%, `spec-beliefs.md`
19); (c) the mean volume per active market is at least 1 unit a tick (the paper's); (d) no
market is pinned for more than `x4.pinned_share_bp` of the ticks (F5); (e) households are
not below `x4.households_bp` of the start (F3); (f) the price range of each commodity in
the window is under `x4.range_factor` (F6). The criteria (b) and (c) are the only prior
definition of "stable" for this family of models and are scale-dependent, a lead and not a
calibration (`spec-beliefs.md` 19). *Requirements:* FR-REF-017, FR-REF-031, FR-CFG-004
(a file and a Go-built config agree: the test builds `second` both ways), FR-PRO-046.
*Level:* integration.

**X5. A checkpoint taken mid-run and resumed (vision 5 item 4, ADR 0002).** *Packs:*
`one` and `five`. *Setup:* run 200 ticks, save, restore into a new world, run 200 more;
compare with an uninterrupted run of 400. *Pass (exact):* the two runs give equal events
and an equal final rendering (8.5), and the two checkpoints are byte-equal at every
boundary of the continuation (`spec-checkpoint.md` 8.2.1). The requirements are
FR-CHK-056 and FR-CHK-057 and, for the state, FR-AGT-064, FR-PRO-043, FR-WEL-034,
FR-SKL-027, FR-SPN-043, FR-SPN-052. *Level:* end-to-end.

**X6. The golden file and the determinism on two architectures.** See 8.5 and 8.6.

**X7. The invariants over seeds.** See 7.5.

**X8. The observability tests on the reference chain.** *Packs:* `one` and `five`.
*Pass (exact):* (a) the cardinality test: every `commodity`, `role`, `recipe` and `side` of
every event that becomes a metric attribute is a value of the configuration of `five`
(observability-and-logging.md 8.1); (b) non-interference: `one` for 200 ticks with and
without a logger and an observer gives equal renderings; (c) the allocation test: a full
`Step` of `one` with the defaults allocates no more than a baseline constant held in the
test (8.2 there); (d) composition and emission order for a scripted step of `one`. *Level:*
integration.

### 7.5 The invariants over seeds

The six invariants of testing.md 6.3 are asserted after every tick:

1. goods are conserved except by recipes (and the reasons that register as flows:
   spoilage, wear, consumption, disposal);
2. money is conserved except by explicit faucets and sinks (there are none in the
   packs after construction);
3. no inventory or balance is negative;
4. no trade has a buyer or seller that does not exist;
5. the clearing price lies between the matched bid and ask;
6. identical seed and configuration give identical results.

| Pack | Ticks | Seeds |
|------|------:|-------|
| `one` | 400 | 1 to 16 |
| `three` | 400 | 1 to 4 |
| `famine` (with the shock of S14 at depth 0) | 800 | 1 to 4 |
| `five` | 400 | 1 and 2 |
| `second` | 400 | 1 to 4 |
| `one-demand` | 400 | 1 to 4 |

Money and quantities are integers, so the checks are exact equality (testing.md 6.3). The
test runs with the audit option on (`WithAudit`, `spec-ledger.md` 12.4) and checks the
events as well as the totals. A failure reports the seed.

The rows of `famine`, `five` and `one-demand` need the estates part, regions that
trade and the demand module, so the first slice runs `one`, `second` and `three` (`three`
only if the probe of the slice shows that the Stage 1 world runs it) and the other rows are
added with the module that makes each buildable (N-295). The table is not renumbered, and
the invariant and determinism tests of 8.2 are table-driven so that a row is added and no
test is rewritten.

### 7.6 What no scenario can run yet

| Item | Waits for | Next step |
|------|-----------|-----------|
| X1 dynamic, X2 dynamic: price convergence and specialisation between regions | the code; REG, TRN, INF and MER are written | the edge part of `overlay/trade.toml` is `spec-regions.md` 12.4 and the mode and haul recipe follow `spec-transport.md` 17.4; add the merchant and consignee roles of `spec-merchants.md` 19.1 and INF's `[modules.information]` for the rumours; add the two tests |
| S28: a closure of the middle link and its reopening (REG-A1) | the code; INT (the public call that sets an edge status); TRN and MER are written | the REG row (Stage 2 row 3) builds the graph leg on the edge part of the trade overlay, the closure set through the unexported status setter (the public call is S80's): distances, components and relocation refusals at each step; the price leg waits for TRN and MER: assert the cut region's households fall no faster than the closed-region variant and the price gap returns inside a band |
| S29: a town as one region against a town and two hamlets (REG-A2) | the code (TRN and MER are written) | the REG row (Stage 2 row 3) builds the graph leg on the edge part of the trade overlay (a town as one region against a town and two hamlets); the leg that asserts thin markets are reported waits for TRN and MER |
| S30, S31, S32: price-gap shipping with and without `Inbound`, a perishable at the travel time, a haul recipe at three elasticities (TRN-A1, A2, A5, A3) | the code; the shipper is MER's (written) and the mechanism `spec-transport.md` | need the merchant strategy (`strategy.Merchant`) and `overlay/trade.toml` |
| S33 to S36: a rate levy on a hoarding class, a levy below the upkeep line, growing households with and without a pool grant, the scale seam (MON) | the packs exist (`overlay/faucet.toml`, `overlay/faucet-sunk.toml`, `one-demand`); the scale seam needs INT's public call | add the tests; S33 to S35 can run once the code exists |
| S37 to S40: approach to the fixed point from both sides, a shock and recovery, a closed region with an empty pool, the closed ceiling (POP) | RES (the fixed factor of the Malthusian loop; now written, but nothing has run) or a test double, and `overlay/population` | the population overlay uses `overlay/resources.toml` (the binding set) or a fixed-factor double, wants on and `Renewal` 0 (`spec-population.md` 18.3, `spec-resources.md` 18.3) |
| S41 to S45: the resource-population loop, a logistic stock above and below the maximum draw, a finite deposit, the ration with cohorts, a blight and a drought (RES) | `overlay/resources.toml` (REF, with the solver's capacities); S41 needs the population overlay; S45 needs INT's shock slots | build the overlay; S42 to S44 can run once the code exists |
| S46, S47: shippers reading rumours; a phantom gap from a stale source (INF) | the code; the shipper is MER's (written); the first assertion of S47 runs with a recording strategy | need the merchant strategy and `overlay/trade.toml` with `[modules.information]` |
| S48, S49, S50: a closure and a reopening seen through the rumours, a checkpoint with the error on, a seasonal swing against a lag (INF) | `[modules.information]` and a recording strategy; S48 needs INT's edge-status call | add with `overlay/trade.toml`; S49 and S50 can run once the code exists |
| S51 to S55: a corridor and a cobweb, an import-only good and a closed edge, a walk, a cap and a window, the invariant run (EXT) | `overlay/external.toml` (REF); S52 needs INT's edge-status call | build the overlay; S51, S53 to S55 can run once the code exists |
| S56 to S60: the subsistence floor under a staple shortage and an outage, a lost income, own-use jobs against the ladder, a scarce commons, the invariant run (SUB) | `overlay/own-use.toml` (REF); S56 needs the pack of S04; S57 uses `famine`; S59 needs a `refill` endowment from `overlay/resources.toml`; S60 needs the code and a checkpoint | build the overlay (7.9); S58 and S60 can run once the code exists |
| S61 to S66: a migration cobweb after a yield shock, herding on one destination, relief on and off, a journey beyond `CoverTicks`, emptying a chain, the invariants over seeds (MIG) | `overlay/migration.toml` on `overlay/trade.toml` (REF); S61 to S65 wait for the code; S66 needs the code and `WithAudit` | build the overlays (7.9); S66 can run once the code exists |
| S67 to S72: the merchant lane with the lock and the spread, contracts and escrow, a thin source market, mode choice and relocation, a gate region, a checkpoint with contracts in flight (MER) | `overlay/trade.toml` with the merchant and consignee roles of `spec-merchants.md` 19.1 (REF); S71 needs `overlay/external.toml`; S72 needs the code and a checkpoint | add with `overlay/trade.toml` (7.9); S68 and S72 can run once the code exists |
| S73 to S76, S79 to S83: the cap, the cadence, the cold start, the stream, a failed harvest, a closed road with a consumer rule, the sequence of calls, a player and a restore, a forced move (WLD, INT) | the code; INT is written; S80 needs `overlay/trade.toml` and its merchant as S28 | assigned in 7.11; S79 to S83 and the unit models of S73, S74 and S76 can run once the code exists |
| A1, the run legs (a warm run within `a1.price_band_bp` for 30 ticks, a cold run not better); A7 and A8 (conservation, stagger) | a running world with the solver: the stage that owns it (the world row for A7 and A8; A1's legs need the warm start applied to a built world) | listed here and not tested in Stage 1; add them with that stage |
| S12, the coarse-tick leg | CAL experience and STY | restate the pack at five days a tick and add the leg |
| X5: checkpoint and resume | the code (`spec-checkpoint.md` exists) | add the test: byte-equal checkpoints at every boundary of the continuation (`spec-checkpoint.md` 8.2.1) |
| All bands | the keys of `spec-stylised-facts.md` 3.6 and 5.11, then a run that a human reviews (STY 8.4) | STY 8.4 sets a band from a reviewed run of these packs; a calibration (CLB) never sets one, and EXP is the harness that runs the seeds |
| S20, the B part (a net earner reaches the reserve) | a pack in which a group starts below the reserve and earns; the code | tier N (`spec-stylised-facts.md` 4.5): add the pack when the code exists |

**Band keys of the scenarios of this table** (`spec-stylised-facts.md` 5.11; the table has no
band column, and a key is unset until a reviewed run sets it, 7.1.5): S28 `s28.band_bp` (the
band, in basis points, within which the gap returns to its value before the closure) and
`s28.reopen_ticks` (the ticks it has to); S29 `s29.disp_bp` (the dispersion of the three
prices once trade is on); S40 `s40.hunger_bp` (the unfed share at the fixed point, the
closed ceiling of `spec-population.md` 5.3); S43 `s43.runway_bp` (the run-down time within
that share of `endowment_runway`); S44 `s44.grant_band_bp` (the grant per household
within that share of the mean of the ration) and `s44.denied_ticks` (the longest run of
ticks an agent may be denied).

### 7.7 Scenarios of SPN and DEM

`spec-spawner.md` proposes S26 and S27 and `spec-demand.md` S23 to S25 (IDs assigned in
round 5; the register entries are `spec-stability.md`'s). The packs they need are in 4.

- **S26, a role-income shock with the caps on and off** (`spec-spawner.md` 17).
  *Packs:* `one`, with `entry_share` at its default and at 0 and `exit_share` at 0 (three
  generated documents). *Setup:* 1 095 ticks; a yield shock on `cook`'s input (`prepare`
  has no yield profile; the shock acts on `dig_mineral` for `digger` income, or on
  `cut_fuel` for `forester`), 50% for 120 ticks from tick 400. *Measure:* the role's share
  of households and the role changes per tick (`RoleChanged`), the `EntryLimited` counts.
  *Pass:* with the caps on the share swings by at most `s26.share_band_bp` and the changes
  per tick do not exceed `ceil(Households * EntryShare / 10 000)` (exact); with the caps off
  the share swings more (the measure can see it, O). *Requirements:* FR-SPN-021, FR-SPN-022, FR-SPN-035. *Level:*
  end-to-end.
- **S27, a region where no role is viable** (`spec-spawner.md` 17). *Packs:* `famine`
  with the yield shock of 0 on every recipe of `north` from tick 100 to the end of the
  run (a script of `scripts/s27.toml`), so that no role has income. *Pass:* no entrant is made, the `Warn` "role extinct" or the
  counter `NoViableRole` shows, and the households fall. *Requirements:* FR-SPN-025,
  FR-SPN-042. *Level:* end-to-end.
- **S23, a self-employed luxury role under the target-income rule** (`spec-demand.md`
  13). *Packs:* `one-demand`, with `elasticity` of `treat` at 5 000, 10 000, 15 000 and 30 000
  and the budget headroom by `spend_max_share` (generated documents). *Measure:* the
  price of `treat_a` and whether it converges from a start 10% above and 10% below.
  *Pass:* the price converges to within `s23.band_bp` of the reference after the
  transient only when both conditions of `spec-demand.md` 6.3 hold (O), and the lower edge
  of the basin is reported. *Requirements:* FR-DEM-001 and the schedule
  requirements of `spec-demand.md` 18. *Level:* end-to-end.
- **S24, the closed economy with a ladder** (`spec-demand.md` 13). *Packs:*
  `one-demand` and `one` for comparison, the default `TargetRatio` and one above the
  ceiling. *Pass:* the aggregate ratio (`HouseholdIncome` when `OK`, the mean of ratios beside it;
  `HouseholdIncome` is `OK` false in version 1, so until OBS adds the income fields the
  band is reported and not asserted, as in S19) is near 10 000 without wants, above it
  with them and at most `1 + Wmax / U` (about 1.32 here), and the glut is bounded with a
  target above the ceiling. *Requirements:* FR-AGT-070 and `spec-demand.md` 18. *Level:* end-to-end.
- **S25, money below the reserves and a substitution flip** (`spec-demand.md` 13).
  *Packs:* `one-demand` with the total money at 0.5, 1 and 2 times the sum of the reserves
  (the warm-start `money` scaled) and `substitution` 4 on `treat`, whose alternatives
  differ in price by 62%. *Pass:* wants are not bought below the reserves (exact), and the shares of
  the two alternatives swing by at most `s25.swing_bp` over a year, without damping
  lost. *Level:* end-to-end.
### 7.8 What the requirements of the new specs now let a scenario cite

| Scenario or check | Cites now |
|-------------------|-----------|
| S14, S26, S27 (the failure ladder, entry, exit) | FR-SPN-004, FR-SPN-012 to FR-SPN-016, FR-SPN-021 to FR-SPN-026, FR-SPN-032 to FR-SPN-036, FR-SPN-041, FR-SPN-042, FR-SPN-045, FR-SPN-046 |
| S06 (role shares return, `Closed` visitor) | FR-SPN-018 to FR-SPN-021, FR-SPN-041, FR-SPN-049 |
| S23, S24, S25 (wants) | FR-DEM-008 to FR-DEM-011 (shares, schedule, choke), FR-DEM-016 and FR-DEM-017 (budget, ladder), FR-DEM-020, FR-DEM-029 to FR-DEM-031 (Engel, closed economy, ceiling), FR-DEM-035 and FR-DEM-036 (`WantShortfall`, the `Warn`), FR-DEM-039 (conservation) |
| X4 and the file tests of the demand overlay | FR-DEM-043 (the file keys `modules.demand.want.<id>`, `modules.demand.taste.<role>.<want>`, `modules.demand.region_taste.<region>.<want>`, which `demand.toml` uses), FR-CFG-060 to FR-CFG-062 |
| A1, A2, `Validate` on every pack | FR-VIA-055 to FR-VIA-059, FR-CFG-058, FR-CFG-059, FR-CFG-065, FR-CFG-066, FR-PRO-046 |
| `famine` warm table | FR-VIA-058 |
| `NewWorld` on `visitor` (no group), a closed role | FR-CFG-066, FR-SPN-041 |

### 7.9 Scenarios of SUB, MIG and MER (S56 to S72)

`spec-subsistence.md` 18.3, `spec-migration.md` 17.2 and `spec-merchants.md` 17 propose
SUB-S1 to SUB-S5, MIG-S1 to MIG-S6 and MER-S1 to MER-S6; the register gives them the final
IDs S56 to S72 (`spec-stability.md` 4.11, 8). None can run yet: each waits for the code, for the
overlay named here, and for the specs in 7.6. The overlays are in 4.5; the packs are:

| Scenario | Pack and overlays | Waits for |
|----------|-------------------|-----------|
| S56 (SUB-S1) | `one` + `overlay/own-use.toml`, the pack of S04 with a shortage injected and an outage of the staple market; coverage 6 667, 10 000 and 13 333, the retry on and off, the work floor on and off | the code; the overlay |
| S57 (SUB-S2) | `famine` + `overlay/own-use.toml`, with `overlay/estates.toml` (the pool, relief and the ladder) | the code; the overlay |
| S58 (SUB-S3) | `one` + `overlay/own-use.toml` with own-use recipes of duration 1, 3 and 8 against `ChangePrivation` 3 | the code; the overlay |
| S59 (SUB-S4) | `one` + `overlay/own-use.toml` + the `refill` endowment of `overlay/resources.toml` that the recipe draws on; a thousand agents of weight 1 and twenty cohorts of weight 50 | the code; the overlays |
| S60 (SUB-S5) | `one` + `overlay/own-use.toml`, a range of seeds, `WithAudit`, a checkpoint in the middle of a mode, a relocation of a household in the mode | the code; the checkpoint leg is `spec-checkpoint.md` 8.2.3 (FR-CHK-057) |
| S61 to S66 (MIG-S1 to MIG-S6) | `five` + `overlay/trade.toml` + `overlay/migration.toml`; S61 uses two of its regions and S62 three; S61 and S63 shock a yield of one region between ticks; S64 uses an edge whose journey exceeds `CoverTicks`; S65 sets `fixed` on the chain's roles for its second leg; S63 runs with `overlay/estates.toml` on and off; S66 runs over a range of seeds with `WithAudit` | the code; the overlays; INT for a closure during a journey (S66) |
| S67 (MER-S1) | `two` + `overlay/trade.toml` with a merchant role of 20 and then 50 agents, the lock and the spread on and off, `fee_share_spread_bp` 0, 1 000 and 2 500, rumour lags of 1, 2 and 4 (it is S46 and S30 with the merchant) | the code; the overlay |
| S68 (MER-S2) | `two` + `overlay/trade.toml` with a consignee role (`consigns`) and brokers, a spoiling commodity, a closed edge, and the removal of a deliverer and of a payer scripted in transit | the code; INT for the closure and the removals |
| S69 (MER-S3) | `two` + `overlay/trade.toml` with a thin source market (S47 with the merchant) | the code; the overlay |
| S70 (MER-S4) | `three` + `overlay/trade.toml` in a chain, a hired mode and a self-carried one, the price of haul units raised stepwise | the code; the overlay |
| S71 (MER-S5) | `one` or `three` + `overlay/trade.toml` + `overlay/external.toml` (S52 and S55 with the merchant) | the code; the overlays |
| S72 (MER-S6) | the reference world with merchants and consignees, 200 ticks, a checkpoint at 80 and a resume | the code; the checkpoint leg is `spec-checkpoint.md` 8.2.3 (FR-CHK-057) |

**Band keys of these scenarios** (`spec-stylised-facts.md` 5.11; the table has no band
column, and a key is unset until a reviewed run sets it, 7.1.5): S56 `s56.unfed_drop_bp`
(the unfed household-ticks fall by at least that share when the floor covers the upkeep);
S61 `s61.journeys_factor` (the journeys within that factor of the minimum, in basis
points); S62 `s62.split_band_bp` (the arrivals divide between equal destinations within
that band); S63 `s63.journeys_band_bp` (the journeys differ by less than it between
relief on and off); S65 `s65.role_share_bp` (no role of the chain falls below that share
of its start). The scenarios of S58, S59, S60, S64, S66, S68, S71 and S72 have only exact
checks.

A scenario here asserts its exact checks (conservation, the audit, bit-identical resume) and
its O checks whether or not a band exists (FR-REF-023); the others report until STY sets bands. The packs of S56 to
S72 add no feature to the fixture's census (section 3): SUB, MIG and MER are modules and
overlays on the base chain.


### 7.10 Facts that no scenario holds

Some facts of `spec-stylised-facts.md` have a band and no scenario of 7.2 to 7.9: they
run on a pack, over a window, and no entry above names them. Their keys are in
`bands/<pack>.toml` of the pack they run on, as tables named by the key
(`[d01.unfed_bp]`, 7.1.5), with the same rule as every band: unset until a reviewed run
sets it (STY 8.4), and the fact is **reported** until then. The tables below give, per
pack, the facts, their keys and the prior of STY (a placeholder, `spec-stylised-facts.md`
3.6); the measure and the pass are STY's. The thresholds of the regimes are not here:
they are `bands/regimes.toml` (7.1.5).

**`one`** (`second` carries the same three facts):

| Fact | Key | Prior | Window |
|------|-----|-------|--------|
| STY-P07, the price index drifts at most a stated rate in a closed pack with constant households (`one`, `second`) | `p07.drift_bp` | 500 a year | the yearly link after the transient |
| STY-D01, almost no hunger from a viable warm start | `d01.unfed_bp` | 100 | the year |
| STY-H02, a warm start holds: the offset, the households and the money per household | `h02.offset_bp`, `h02.money_bp` | 2 000, 2 000 | the years after the transient |

**`two`, `three` and `five`**: STY-D01 and STY-H02 with their three keys
(`d01.unfed_bp`, `h02.offset_bp`, `h02.money_bp`), run for 1 095 ticks (8.2); `five` is
run for seeds 1 and 2.

**`one-demand`**:

| Fact | Key | Prior | Window |
|------|-----|-------|--------|
| STY-D01 | `d01.unfed_bp` | 100 | the year |
| STY-W01, the mean welfare ratio of the unskilled roles with demand beyond the upkeep | `w01.mean_low_bp`, `w01.mean_high_bp` | 10 000, 30 000 | the year |
| STY-W02, a binding fixed factor: households started above and below meet (with `overlay/resources.toml` and `overlay/population.toml`, 4.5; the scenarios S37 and S41 of 7.6 run it) | `w02.n_band_bp`, `w02.ratio_band_bp` | 500, 500 of `r*` | tier L |
| STY-W06, natural increase bounded by the schedules and the money limit (S37, S39 of 7.6) | `w06.fixed_point_bp`, `w06.money_cap_bp` | 100 a year, 1 000 | tier L |

**`one` + `overlay/external.toml`** (the pack of S51):

| Fact | Key | Prior | Window |
|------|-----|-------|--------|
| STY-T03, a quote is a limit: the share of ticks near the quote | `t03.quote_band_bp`, `t03.pull_bp` | 1 000, 5 000 | the year |

### 7.11 Scenarios of WLD, CHK and INT (S73 to S83)

`spec-world.md` 17 proposes WLD-S1 to WLD-S4, `spec-checkpoint.md` 15 CHK-S1 and CHK-S2 and
`spec-integration.md` 16 INT-S1 to INT-S5; the register gives them S73 to S83 (`spec-stability.md`
4.12, 8). The packs and overlays are assigned here. The code is the thing every one waits for; the
overlays are in 4.5. An asserted check of one of these scenarios is a skipped test with the issue ID
of what it waits for, as in 7.9 (FR-REF-038).

| Scenario | Pack and overlays | Waits for |
|----------|-------------------|-----------|
| S73 (WLD-S1) | none: a unit model of the ordering device (a cap of 5 on 100 tied candidates over 1 000 ticks, the places O1 to O5 of `spec-world.md` 10.2 in turn) on a world built in Go in the test, as S02; no seeds | the code |
| S74 (WLD-S2) | none: a unit model of the review rule (1 000 and 5 000 agents, periods 10 and 30, every stride of 1 to 31, three starting IDs) built in Go in the test; no seeds | the code |
| S75 (WLD-S3) | `one` with `warm-one` and without, as S18 (7.2): S18's cold leg is this scenario's, over its 3 ticks; the `Duration` 10 leg over 40 ticks waits for a pack that has such a recipe | the code; a `Duration` 10 recipe in a pack (not defined) |
| S76 (WLD-S4) | none for the unit model of the stream; `one` for the two runs, seed 1, 100 ticks (a placeholder), the second with a test double of `Strategy` that makes one extra draw of the world's source in tick 0 | the code |
| S77, S78 (CHK-S1, CHK-S2) | none to assign: harness tests of `spec-checkpoint.md` 15; S77 takes `one` and `five` as its fixture | not built by REF |
| S79 (INT-S1) | `three`, the staple recipe `grow_staple` of `north` shocked by `SetYieldShocks` of 3 000 for `k` ticks (60 and 120, placeholders: one job and two) and then cleared; the shock is set before the spring start; no overlay (the region is measured alone) | the code |
| S80 (INT-S2) | `three` + `overlay/trade.toml` (the middle link closed by `SetEdgeStatuses`, then open), as S28 (7.6); and `overlay/faucet.toml`, whose grant flow `stipend` is the flow the consumer's rule scales from a published price (a rule written in Go in the test) | the code; the merchant of `overlay/trade.toml` (as S28) |
| S81 (INT-S3) | `one` with a role for external agents (4.2), 100 ticks (a placeholder), seed 1: the same calls twice; two orders of calls on different keys; a run with an `AddAgent` call, with `WithAudit`; the events in the golden rendering of 8.5 | the code |
| S82 (INT-S4) | `one` with the external agent of S81 as the player, a directive and a shock, a checkpoint at tick 40 and a resume to tick 100; the non-mortal and the mortal player on `famine` for the hunger leg | the code; the checkpoint leg is `spec-checkpoint.md` 8.2.3 |
| S83 (INT-S5) | two legs: the household on a journey on `five` + `overlay/trade.toml` + `overlay/migration.toml` (the pack of S61), and the household with an open contract on `two` + `overlay/trade.toml` with a consignee role (the pack of S68) | the code; the overlays |
| S84 to S90 (EXP-S1 to EXP-S4, CLB-S1 to CLB-S3) | none to assign: harness tests of `spec-experiments.md` 17 and `spec-calibration.md` 11; S88 takes `one` as its fixture | not built by REF |

**Band keys of these scenarios** (`spec-stylised-facts.md` 5.11; the table has no band column,
and a key is unset until a reviewed run sets it, 7.1.5): S79 `s79.recover_ticks` (the ticks the
staple price takes to return after the shock clears); S80 holds S28's two keys,
`s28.band_bp` and `s28.reopen_ticks`. S73, S74 and S76 have **no band**: they are reported, and
the bounds of S73 and S74 are the owner's derived bounds. S75's cold leg is reported and not
asserted, as in S18. S77, S78 and S81 to S83 have only exact checks, asserted from the day they
can run (FR-REF-023); S82 also has an O check (the non-mortal player stays at hunger).


### 7.12 Scenarios of the fourteen later modules (S91 to S165)

The owner specs propose their scenarios under provisional names and the register numbers them
(`spec-stability.md` 4.13 to 4.15, 8). Each runs with its module on over a range of seeds with
`WithAudit`; the pack and overlays are assigned here, the overlays are in 4.5 and the code is the thing
every one waits for (FR-REF-038). A scenario that is a **harness test** is of the class of S77 and S78: it
asserts exact checks only, is outside the band rule of `spec-stylised-facts.md` 5.11 (FR-STY-098) and, when it
needs no overlay, takes any pack as its fixture and is not built by REF.

| Scenario | Pack and overlays | Class |
|----------|-------------------|-------|
| S91 to S96 (STO-S1 to STO-S6) | `one` + `overlay/storage.toml`; S94 (two shippers and one consignee) on `two` + `overlay/trade.toml` + `overlay/storage.toml` | the invariant of `spec-storage.md` 4.4 every tick; the rest reported |
| S97 to S101 (EMP-S1 to EMP-S5) | `one` + `overlay/employment.toml`; S98 a labour-bound pack of it (more places than offers); S99 removes an employer and a worker in a job | S101 is a harness test: a module on takes no word, with or without offers |
| S102 to S104 (SET-S1 to SET-S3) | `five` + `overlay/trade.toml` + `overlay/settlements.toml` with the population module and a refilling endowment (S102, S103); `famine` with the overlay (S104) | exact checks and the class sequence; bands are STY's |
| S105, S106 (SET-S4, SET-S5) | none to assign: harness tests of `spec-settlements.md` 13 (inertness; a checkpoint during a held change) | not built by REF |
| S107 to S111 (LND-S1 to LND-S5) | `one` or `five` + `overlay/resources.toml` + `overlay/tenure.toml`; S109 with `overlay/own-use.toml` and `overlay/migration.toml`; S110 with `overlay/population.toml` | the invariants asserted; the rest reported |
| S112, S113 (POL-S1, POL-S2) | `five` + `overlay/polity.toml` against `five` (S112); `five` + `overlay/polity-governed.toml` against S112's run (S113), seeds 1 to 10, 200 ticks | harness tests: exact, outside the band rule |
| S114 to S118 (CRD-S1 to CRD-S5) | `one` + `overlay/credit.toml`, cold and warm start (S114, S115); S116 invariants over seeds; S117 a cohort borrower; S118 calls and a checkpoint with open claims | exact checks asserted; the rest reported |
| S119, S125 (CAP-S1, CAP-S7) | none to assign: harness tests of `spec-infrastructure.md` 16 (the module on with no class equals off; a checkpoint mid-build) | not built by REF |
| S120 to S124 (CAP-S2 to CAP-S6) | `one` + `overlay/infrastructure.toml`; S121 on `three` + `overlay/trade.toml` (a road between two regions); S123 with `overlay/population.toml` and `overlay/resources.toml` | the invariants asserted; the rest reported |
| S126 to S131 (RSK-S1 to RSK-S6) | `five` or `two` + `overlay/trade.toml` + `overlay/risk.toml`; S126 with every scale at 0 against the module off; S129 with `overlay/storage.toml` for the buffer store | exact identities every tick |
| S132 to S136 (FIS-S1 to FIS-S5) | `five` + `overlay/polity-governed.toml` + `overlay/fiscal-governed.toml` (S132 with every flow at scale 0, against the world without the module); S135 on `one`; S136 on `three` + `overlay/trade.toml` | S132 is a harness test; the others have exact accounting checks and report the rest |
| S137 to S141 (ORG-S1 to ORG-S5) | S137 `one` or `three` + `overlay/institutions.toml` (with the trade overlay for contracts); S138 `five` + `overlay/institutions.toml`; S139 the same with `overlay/migration.toml`; S140 `one` + `overlay/own-use.toml`; S141 `five` + `overlay/institutions-inert.toml` against `five` | S141 is a harness test; S137 reports its measures |
| S142, S147 (SAV-S1, SAV-S6) | none to assign: harness tests of `spec-saving-and-investment.md` 14 (the module on with no block equals off; a checkpoint while saving and mid-build) | not built by REF |
| S143 to S146 (SAV-S2 to SAV-S5) | `one` + `overlay/saving.toml` (the welfare module on); S144 and S145 with `overlay/infrastructure.toml`; S146 with `overlay/faucet.toml` | the identities asserted; the rest reported |
| S148 to S154 (TEC-S1 to TEC-S7) | `one` + `overlay/technology.toml`; S149 with `overlay/population.toml` and `overlay/resources.toml`; S150 on `three` + `overlay/trade.toml`; S151 with `overlay/resources.toml` and `overlay/population.toml` | S148 and S153 are harness tests on `one` over a range of seeds; the rest report |
| S155 to S159 (PLC-S1 to PLC-S5) | `five` + `overlay/polity-governed.toml` + `overlay/fiscal-governed.toml` + `overlay/policy.toml`; S158 with `overlay/policy-directed.toml`; S156 adds INT-S1's yield shock (S79) | S155, S158, S159 are harness tests; S156 passes on a band STY sets; S157 is a report |
| S160 to S165 (INS-S1 to INS-S6) | `two` + `overlay/trade.toml` + `overlay/risk.toml` + `overlay/cover.toml` (+ `overlay/credit.toml` for S161); S162 over several underwriters | exact identities every tick; S163 and S164 report |

The harness tests of these scenarios are S101, S105, S106, S112, S113, S119, S125, S132, S141, S142, S147, S148,
S153, S155, S158 and S159; the list is what FR-STY-098 reads (a count is never written). A scenario here
asserts its exact checks and its O checks whether or not a band exists (FR-REF-023); the others report until
STY sets bands. The packs add no feature to the fixture's census (section 3): each module is a module and an
overlay on the base chain.


## 8. How the tests use the fixture

### 8.1 Levels (testing.md 2)

| Use | Level | Where |
|-----|-------|-------|
| The loader and validators on the documents, A2, A3, A5, S21, the expected-results check, the regeneration of the warm tables | unit and integration (no run, or a few ticks) | `config/` and the root package |
| S01 to S03, S07 to S09, S11, S17, S18, S20, A1, A7, X4, X8 | integration (a few hundred ticks, one small pack) | root package, external test package |
| S04 to S06, S10, S12, S14, S16, S19, S22, X3, X5, golden, invariants over many seeds | end-to-end (a year or more, many seeds, or `five`) | root package, external test package |
| The scenarios and facts that read an IND preset (for example S19 and S24, and the detector facts of `spec-stylised-facts.md`) | as their level above | `indicator/scenario_*_test.go` (REF) and `indicator/stylised_*_test.go` (STY), package `indicator_test`, sharing `stylised_helpers_test.go` (the recording observer, owned by STY): FR-IND-001 lets no other package import `indicator` |
| `FuzzLoad` and the validator fuzz tests | fuzz | `config/` |

All pack documents are the **seed inputs** of the loader fuzz test (`f.Add`, testing.md
7.4), so the fixture also feeds that test. Tests in a package other than the root read the
documents by relative path (`../testdata/chains/...`, testing.md 10.4) through
`config.LoadFS(os.DirFS(...))`; none reads outside the module.

### 8.2 Seeds and lengths

| Test | Pack | Ticks | Seeds |
|------|------|------:|-------|
| Band scenarios (S01, S03 to S08, S10, S12, S14, S16, S17, S19 to S22, A1, X1 to X4) | as stated | as stated in 7.2 to 7.4 | 1 to 10 (`Seed0` 1, `Replicas` 10 of `spec-experiments.md` 3.1) |
| Nightly repeat of the band scenarios | as stated | as stated | 1 to 50 (`Replicas` 50) |
| Tier L (`spec-stylised-facts.md` 4.5: a run longer than 1 800 ticks) | as stated | as stated | 1 to 10 |
| Detector facts (`spec-stylised-facts.md` D01, G01, H02) | `two`, `three`, `one-demand` | 1 095 | 1 to 4 |
| Detector facts | `five` | 1 095 (about 58 s a seed) | 1 and 2 |
| Hold-out (a recommendation, **no band and no calibration choice is set from it**) | as stated | as stated | 51 to 60 |
| Invariants | 7.5 | 7.5 | 7.5 |
| Golden | `one`; `five` | 400 | 1 |
| Determinism (twice, same seed; once with another seed) | `one`, `five` | 400 | 1 and 2 |
| Checkpoint X5 | `one`, `five` | 200 + 200 | 1 |
| Configuration-only X4 | `second` | 1 095 | 1 to 4 |

400 ticks cover a full Earth year plus the first harvest of every group (the first
completions are at tick 118 and the stagger spreads them over the next 90), and the first
tool replacements. 1 095 ticks are three years: the first lets the transient out, the last
two are the window. These lengths and seeds are test constants, not configuration, and are
placeholders; the cost at the placeholder budget of `spec-scale.md` 6 is about 4 ms a tick
for `one` (600 agents of 6.5 us), so 16 seeds of 400 ticks take about 26 seconds, and
about 53 ms a tick for `five`, so 400 ticks take about 21 seconds. Neither is a
measurement (NFR-REF-002). A calibration's search blocks (`Seed0` 101 and up, `spec-calibration.md` 7.1) are its own, outside every set above, and the gate is never an input to its choices; the hold-out of 51 to 60 is run only if wanted (a SHOULD).

### 8.3 What a scenario test looks like

```go
// TestScenarioS04_StapleShortage verifies that a half harvest in the second spring
// does not pin the staple price for longer than the band allows, and records the
// measured demand elasticity of the staple.
//
// Setup: pack one (and one + nofloor), seeds 1 to 10, 1 400 ticks, shock script s04.
//
// Requirement: FR-STB-009, FR-STB-008, FR-REF-020
func TestScenarioS04_StapleShortage(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	// ...
}
```

The comment is the contract (testing.md 5.1, 5.6): the scenario, the pack, the seeds and
why. The failure message follows `got`/`want` and names the seed (testing.md 3.3, 6.2).

### 8.4 The expected results of a pack

`expected/<pack>.toml` holds what the load of a pack must give, so a change to a pack, a
default or the solver shows up as a failing test and not a silent drift:

```toml
config_hash = ""            # the hex SHA-256 of the effective configuration (FR-CFG-025)
surplus_bp = 0              # floor(lambda* * 10 000), per region and pooled
labour_share_bp = 0
pivots = 0
unpriced = []               # UnpricedCommodities and UnpricedWages
warnings = []               # [{ rule = "...", path = "..." }], in the order of FR-CFG-012
income_bp = []              # per group, IncomeBP of spec-viability.md 5.2 (FR-VIA-060); a group
                            # without one is listed with the reason (FR-REF-041)
```

The prototype's values for the first rows are in 5; the Go run replaces them and the file
is checked in with the pack (FR-REF-005, FR-REF-006). Changing it follows the rule for the
golden file (8.7).

### 8.5 The golden rendering (testing.md 7.1)

The golden file of a pack (`testdata/golden/off/<pack>.txt`, testing.md 7.1.1) is a text rendering with no floating point, no map order, no
pointer and no wall-clock time; integers in decimal, sections and lines in a fixed order,
UTF-8, LF. For a run of `T` ticks with a seed:

1. **Header:** the format version; the effective-configuration hash; the seed and `T`;
   the warnings (rule and path), sorted; the solver report (`surplus_bp`, labour share,
   pivots, unpriced).
2. **Samples,** at ticks 0, 1, 2, 5, 10, 20, 50, 100, 200, 300 and `T - 1`, for every
   region: for every tradable commodity the published price, its age, the volume, the
   unmatched bid and ask units and the best unmatched limits (`MarketCleared`); the total
   held by agents and by the estate; and for every role the agents, the households, the
   unfed households; and for every region the mean money per household rounded down
   (`MoneyHouseholds / Households` of the region row, since `RoleRow` has no money
   field, FR-REF-027). The pool balance and the pending output by commodity.
3. **Event counts:** for the whole run, the count of each event type and, for
   `OrderRejected`, `AgentBankrupted` and `RecipeAbandoned`, the count by reason or cause;
   per window of 100 ticks, the counts of `TradeExecuted`, `RecipeExecuted`,
   `OrderPlaced` and `UpkeepShortfall`.
4. **Flows:** the `BoundaryFlow` totals of the whole run by region, reason and commodity,
   in that order, and the construction bucket (observability: N-072).
5. **Final state:** the agents and households by region and role; the goods and money
   totals; the jobs in progress by recipe.
6. **Checkpoint digest:** the SHA-256 of the checkpoint at tick `T`: the 32 bytes of its
   trailer (`spec-checkpoint.md` 8.3), as the last line of the file, `checkpoint_digest `
   and the 64 lower-case hexadecimal digits. **Item 6 is added by the restore change
   (Stage 2), which moves `testdata/golden/off/one.txt` once for it**, with the
   explanation of testing.md 7.2 (item 6 is new and nothing else in the file changes);
   until then the file ends with item 5. From then on the line is rendered for every
   pack and every set (off, module-alone, all), and the digest moves with every change
   of a pack's economy (a rebalance), which gives the explanation for item 6 too.

**A module that is off leaves no trace** (decision D5 of the plan of Stage 0;
architecture.md 2.10.5). A count, an amount or a flag at zero is not written, a
section or a row with nothing in it is not written, an event type or a reason with
a count of 0 is not listed in item 3 or 4, and a warning is listed in the header
only when it fired. So a field or an event that a module appends changes the file
of a pack only when the module is on, the effective-configuration hash of the
header is unchanged by an append (`spec-config.md` 8.2), and the files of
`testdata/golden/off/` do not move when a module is added (testing.md 7.1.1).
The rendering of the events of `spec-observability.md` 9.5.2 follows the same
rule.

For `one` this is about 150 lines; for `five` about 1 200 (85 markets at 11 samples).
The file is regenerated only by the flag `-update` and a change needs the explanation of
testing.md 7.2: which behaviour changed and which requirement or fix it serves
(FR-REF-029). A change caused by a Go toolchain upgrade is explained like any other.

### 8.6 Determinism on two architectures

The golden-file and determinism tests run in CI on amd64 and arm64 (architecture.md 7.4)
with the same file: the rendering of `one` and of `five` must be byte-identical on both,
so the file is checked in once and compared on both. The solver's tables are compared the
same way (A9). The scenario tests with bands run on one architecture; an architecture
difference in them would show first in the golden file.

### 8.7 When a pack changes

A change to a document of `testdata/chains/` changes the effective hash and so the
expected file, the warm-start tables (regenerated by the solver, never edited by hand, and
checked back by FR-REF-007) and possibly the golden files. The PR says which of the
behaviours of the register it changes (architecture.md 9.8). A change to a quantity of the
base is a change to every pack; a change to a region document touches one.

A change that adds or alters a commodity, recipe, role, upkeep line, yield or region of
the base, of `lines.toml`, of a region document or of an overlay that a pack layers
re-runs the rebalance (the runbook): (1) re-solve the table and read the census, the
imbalance and `IncomeBP` of every group; (2) fix the data until the static income gate
of FR-REF-041 holds; (3) run `one`, or the pack the change touches, warm from tick 0
over the ticks and seeds of 8.2 and check the predicate of X4 (FR-REF-042 for `one`);
(4) regenerate the warm-start documents and the `expected/` files with `-update`; (5)
move the golden files once, in that change, with the explanation of testing.md 7.2; (6)
record the run with the pack (FR-REF-039). A pack edit that skips the gate is a finding.

## 9. Observability needs of the scenarios

The measures of 7 use only the events of observability-and-logging.md 4.3, the snapshot,
INT's `Agent(a)` queries and IND's presets (FR-STB-012, FR-REF-027). What each needs, of
OBS (`spec-observability.md`):

| Need | Used by | Source today |
|------|---------|--------------|
| `MarketCleared` with price, age, volume, unmatched units and best limits, once per market per tick | S01, S04, S05, S08, S09, S10, S12, golden | rules 4.3 (done) |
| `BeliefsUpdated` with beliefs at floor and ceiling and the low, high and mean centre | S03, S09, S10, X4, S16 | rules 4.3 (done) |
| `OrderPlaced`, `OrderRejected` with `reason` | S17, S20, X3, golden | rules 4.3 (done) |
| `RecipeExecuted` (start tick, executions, outputs, disposed) and `RecipeAbandoned` | S07, S18, S17, golden | rules 4.3 (done) |
| `AgentSpawned` and `AgentBankrupted` with `Weight` and `Cause`; `WeightChanged` and `RoleChanged` | S06, S14, S26, X2, households by role | rules 4.3 (done: `WeightChanged` and `RoleChanged` are in the table) |
| `UpkeepShortfall`, `ReliefPaid`, `EstateOpened` | S11, S14, S20 | rules 4.3 (done) |
| `BoundaryFlow` with the reasons `spoilage`, `consumption`, `disposal`; the money given at creation by `AgentSpawned.StartMoney` and the relief by `ReliefPaid` (neither is a `BoundaryFlow` reason, `spec-observability.md` 4.4.1) | S07, S14, S19, A7, golden | rules 4.3 (done) |
| `EntryLimited` and `NoViableRole` | S26, S27 | rules 4.3 (done: `EntryLimited`; `NoViableRole` is one of its counts) |
| `TraineeshipStarted`, `Completed`, `Ended` | S06, X3 | rules 4.3 (done) |
| The household-weighted mean welfare ratio per role and region in the Snapshot | S19, S24 | answered: `RoleRow.MeanRatioBP` (`spec-observability.md` 5.3.4). It is the mean of ratios; the aggregate ratio (`HouseholdIncome`, `spec-indicators.md` 9 row 21) is `OK` false in version 1 and its income fields are a future ask of OBS (`spec-indicators.md` 7.5.1) |
| The unsold stock of each commodity per region in the Snapshot (agents and estate separately) | S16, S19, golden | answered: `HeldByAgents` and `HeldByEstate` of the market rows (`spec-observability.md` 5.3.5) |
| A count of the production halt (an aggregate per region and role per tick) | S16, S19 | answered: `ProductionHalted` (`spec-observability.md` 4.4.6; N-067) |
| The pool balance per region, the pending output, the number of agents with a job by recipe | S04, S11, S14, golden | `spec-estates.md` 2.4, `spec-production.md` FR-PRO-042 (done) |
| Whether the construction bucket is a `BoundaryFlow` with `Tick` -1 | A7, golden | answered: yes, once, at the entry of the first step (`spec-observability.md` 4.4.2; N-072) |
| The realised income of a role per window: gross sales from `TradeExecuted` and the recipe-input cost from `RecipeExecuted`, with the role of each agent from the snapshot at tick 0, `AgentSpawned` and `RoleChanged` | the rebalance runbook (8.7), FR-REF-042 | a recording observer in a `_test.go` file (FR-REF-027); no event of its own |
| The measure of each scenario computable from the above, without unexported state | all | FR-STB-012 |

The three rows that were requests are answered by `spec-observability.md`; everything else exists in a rule or a spec. The one remaining ask is the income fields behind `HouseholdIncome` (`spec-indicators.md` 7.5.1), not adopted for version 1. The test
needs no event of its own and no back door (architecture.md 10.2): a recording observer in
a `_test.go` file collects events, and `Snapshot` is called between steps.

## 10. Parameters

This spec adds **no configuration field** and no row to the parameter-owner table. The
quantities in the documents of 4 are data of a fixture, each a placeholder; the run
lengths and seeds of 8.2 are test constants; the bands are the data of
`bands/<pack>.toml` and belong to STY. The names a test generates in memory (`cost_floor`,
`produce_halt_ratio`, `scale`, `entry_share`, `exit_share`, `elasticity`) are existing
fields of other specs, set to values outside their defaults inside bounds.

## 11. Validation

No rule is added to `spec-config.md`. The fixture is the first user of the rules, and
asks that two things be settled there (section 6): the message of the unsourced-cycle
rule, and which of V1 and 7.5 governs a raw commodity with no source. The expected results
of 8.4 are the record of what validation must say about each pack.

## 12. State contributed to a checkpoint

None, because the fixture is configuration and tests. The checkpoint tests (X5) use it; the checkpoint legs of X5, S60 and S72 are `spec-checkpoint.md` 8.2.3.

## 13. Entries for the stability register

None new: this spec adds no rule and so no loop (FR-STB-003). The register points here for
the scenarios; the loops they exercise are named in 7. For the register owner:

- the "REF" hook rows of `spec-stability.md` 4.8 and 10 point to section 7 here;
- S22 is the check of STB-L53 (a warm start is not a stable start), S06 of the census
  imbalance of `spec-viability.md` 19, S19 of STB-L46;
- the IDs are final: S23 to S25 are DEM's, S26 and S27 SPN's (7.7); S56 to S60 are SUB's, S61 to S66 MIG's and S67 to S72 MER's (7.9), S73 to S76 WLD's, S77 and S78 CHK's, S79 to S83 INT's (7.11) S84 to S90 EXP's and CLB's, and S91 to S165 those of the fourteen later modules (7.12); the harness tests among them (S77, S78, S84 to S90 and the list of 7.12) REF does not build as band scenarios; the damper IDs are
  `spec-stability.md`'s to assign.

## 14. Placeholders, the prototype and sources

**Placeholders.** Every number of the packs: the quantities of the recipes, the
durabilities, the durations, the yields, the censuses, the household counts, the extra
upkeep, the traineeship fields, the wants; every number computed from them (5); every
run length, seed list and tolerance of 7 and 8; the cost estimates of 8.2 (the
placeholder cost model of `spec-scale.md` 6); the stable-state criteria of X4, which are
a lead from a paper (below). None is a default, and none is claimed to be stable.

**The prototype.** `lp.py`, `build.py`, `finalize.py`, `emit_lib.py`, `final.py`,
`verify.py` and the reports in `/tmp/economy-boss/ref/`, run with `docker run --rm -v
/tmp/economy-boss/ref:/w -w /w python:3.12-slim python <file>`, outside the repository
(N-068). `lp.py` loads the TOML documents with `tomllib`, merges them as `spec-config.md`
6.4 (a table-versus-scalar clash is an error), builds the activities (start capacity `J`
and mean factor `E` by the greedy count of `spec-viability.md` 3.1, on the Earth calendar
or the calendar of the document), runs V1 (the producibility fixpoint on existing
activities), V2 (strongly connected components of the material graph, and the exact
programme of 4.3 for each cyclic one), V4 (the longest cyclic run without a completion)
and the reference programme with exact fractions and Bland's rule (taken from the
prototype of `spec-viability.md` 15), then scales the dual to a basket of 1 000, rounds
half to even with the floor 50, apportions the census by the largest remainder with the
start-capacity row counted, and writes the seed table by the rules of 8.2 to 8.5 there.
The lean reserve of `staple` is the use per tick at `lambda = 1` times the ticks to the
first completion (an approximation: the Go rule of 8.4.1 differs in `centre` of `five`, where the cumulative shortfall peaks at tick 301 and a grower holds 1 092 `staple` against the document's 1 087). Wants are counted in
the demand pack as a fixed extra demand at the reference. The prices of the second
setting reproduce those of `spec-viability.md` 15 (a check that the prototype is the same
one). The prototype is not the Go design and decides nothing.

**Sources.** No default is set from a source. Two outside sources were consulted on how
to treat acceptance bands, and neither sets a number:

| Source | Status | Taken |
|--------|--------|-------|
| Windrum, Fagiolo and Moneta (2007), "Empirical Validation of Agent-Based Models: Alternatives and Prospects", JASSS 10(2), 8. <https://www.jasss.org/10/2/8.html> | opened through a summarising fetch, not read in full | that reproducing a stylised fact does not by itself explain it, that several models can give the same stationary distribution, and that joint replication of a larger set of facts discriminates better: so a band set from a run is a description to be reviewed, the scenarios test several facts together, and a band is never set from a single seed (FR-REF-024) |
| Gode and Sunder (1993), zero-intelligence traders | **search snippet only**; not opened | only that market structure alone gives high allocative efficiency, which is why the scenarios test the market's outcome and not the agents' cleverness; no number |
| Doran and Parberry (2012), journal version | not reopened here; read in full by the author of `spec-beliefs.md` 19 | the stability criteria of X4 (b) and (c): none untraded for more than 20% of rounds, mean volume above one unit |
| `bazaarbot-reference.md` 10 | read in full | the shape of the base chain: a food chain, two gatherers, a refiner and a toolmaker; the size of ten agents a role for `K` |

## 15. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-REF-001 | Every configuration document of `testdata/chains/` MUST load with `config.LoadFS` (the files of `expected/`, `bands/` and `scripts/` are not configuration documents and are read by the tests; the overlays of `calibrated/` are) and every variant of 4.1 MUST build a world with no Go code specific to it. |
| FR-REF-002 | The effective configuration of the packs MUST contain every feature C01 to C17 of section 3, and a test MUST check each from the configuration and not from the text of a file. |
| FR-REF-003 | A variant MUST be the documents listed for it in 4.1 merged in that order, and its region document and its warm-start document MUST share no key, so that either order gives the same effective configuration and hash. |
| FR-REF-004 | No identifier, label or comment of the reference chain MUST use setting vocabulary, and the second setting MUST share no commodity, recipe or role ID with it. |
| FR-REF-005 | `Validate` of every variant, overlay combination and the second setting MUST return no error, and `Warnings()` MUST equal the list in `expected/<pack>.toml`, which includes `cold_start_lockstep` for every cold pack and for each warm pack that gives a role with a long recipe no job row (5). |
| FR-REF-006 | Every variant MUST pass V1 to V4, and the solver's report (the surplus and labour share in basis points, the pivots, the unpriced commodities) MUST equal `expected/<pack>.toml`. |
| FR-REF-007 | Each warm-start document MUST be byte-identical to the table that `SolveSteadyState` produces for the layered configuration without it, on two architectures. |
| FR-REF-008 | Every group of every variant MUST be at scale 1, so that every agent has weight 1, except in scenario S10. |
| FR-REF-009 | The variant `one` MUST have one region of 600 households in six roles at the balanced census, nine commodities and ten recipes, and `lambda*` of at least 2. |
| FR-REF-010 | The variant `two` MUST have two regions that are each viable alone and whose autarky ratio of staple to fuel price differs by a factor of at least 2. |
| FR-REF-011 | The variant `three` MUST have three regions that are each viable alone, whose yields differ for at least three recipes and whose autarky surplus factors differ, and its pooled solution MUST have at least one activity active in a strict subset of the regions. |
| FR-REF-012 | The variant `five` MUST have five regions, 5 000 households, 30 commodities and 45 recipes, each region MUST be viable alone, and the pooled surplus factor MUST exceed each autarky one. |
| FR-REF-013 | The variant `famine` MUST be viable at full labour with a surplus factor below 2, MUST have the estates module on, and MUST have a positive starting wage in its warm-start document. |
| FR-REF-014 | The packs MUST contain exactly two trainee-eligible recipes, one whose trainer's gain at the seeds is at least 0 and one whose gain is below 0, and a test MUST compute both from the table. |
| FR-REF-015 | The demand overlay MUST contain a self-employed luxury role, a want with two alternatives and a taste table for at least two roles, and with it the module MUST be on. |
| FR-REF-016 | Each defect pack of 4.5 MUST be rejected with the rule named for it, `d3`, `d4` and `d5` with that rule only, and `d6` MUST load with the warning `cost_floor_at_full_cost`. |
| FR-REF-017 | The second setting MUST build, validate and run from its documents alone, and a test MUST also build it as a Go `Config` and compare the hash (FR-CFG-004). |
| FR-REF-018 | No variant except those that name them MUST contain a demand module, a money faucet or an external sector. |
| FR-REF-019 | The seeds of every warm-start table MUST be at or above the resolution floor and inside the reference band, and the reference basket of every region MUST be within `BasketTolerance` of the cost of its baseline upkeep at the reference prices. |
| FR-REF-020 | Each scenario of 7.2 to 7.4 that can run MUST have one test that runs the packs and setup stated for it, whose `Requirement:` line lists the IDs of its entry and whose failure message names the scenario, the pack, the seed and the measure. |
| FR-REF-021 | A shock or an external agent in a scenario MUST be applied between steps through the integration calls of `spec-integration.md` (`SetYieldShocks`, `AddAgent`, `Direct`) from a script of `scripts/`, and MUST NOT be applied by editing state. In Stage 1 a shock is applied through the unexported yield-shock setter of 7.1.3, until `integration.go` exists. |
| FR-REF-022 | A band MUST be read from `bands/<pack>.toml` (a regime threshold from `bands/regimes.toml`), a scenario MUST assert a band only when its key has a value, and MUST otherwise report its measure and assert only its exact checks and its O checks (7.1.5). |
| FR-REF-023 | The exact checks (class X of `spec-stylised-facts.md` 5.11) and the O checks (class O: a sign, an order or a derived bound) MUST be asserted whether or not a band exists, while its key is unset: X in S02, S03, S08 (the cycle gain), S09, S10, S11, S13, S14, S17, S18, S21, A2, A3, A5 and A7, and O in S01, S06, S08, S14, S19, S20 and S22. The scenarios of 7.6 and 7.9 assert theirs from the day they can run (FR-REF-038). |
| FR-REF-024 | A band MUST be set only from a run over at least the seeds of 8.2 that a human has reviewed, MUST record the run that set it, and MUST NOT be set from one seed or from the code under test without review. |
| FR-REF-025 | The seeds and lengths of 8.2 MUST be used, and a test MUST NOT run a different number of seeds without the change being listed in the PR. |
| FR-REF-026 | Every scenario test MUST be skipped with `-short` and MUST state its level. |
| FR-REF-027 | The measure of every scenario MUST be computed from the events of observability-and-logging.md 4.3, the snapshot, INT's `Agent(a)` queries and IND's presets by a recording observer in a `_test.go` file, without unexported state. |
| FR-REF-028 | The six invariants of testing.md 6.3 MUST be asserted after every tick for the packs, ticks and seeds of 7.5, with exact equality and the audit on, and a failure MUST name the seed. |
| FR-REF-029 | The golden file of `one` and of `five` MUST have the content of 8.5, MUST be regenerated only with `-update`, and a PR that changes it MUST explain what behaviour changed and which requirement or fix it serves. |
| FR-REF-030 | Two runs of one pack and seed MUST give byte-identical renderings, a run with another seed MUST give a different one, and the renderings of `one` and `five` MUST be identical on amd64 and arm64. |
| FR-REF-031 | The configuration-only chain test MUST run `second` to the stable-state predicate of X4 and MUST pass with no code specific to the pack. |
| FR-REF-032 | A world restored from a checkpoint taken after 200 ticks of `one` or `five` and run for 200 more MUST give the same events and rendering as an uninterrupted run of 400, and the two checkpoints MUST be byte-equal at every boundary of the continuation (`spec-checkpoint.md` 8.2.1). |
| FR-REF-033 | The scale-consistency scenario S10 MUST run `one` with `overlay/h200` at scales 1, 4 and 10 and assert the exact invariants of 7.2 there. |
| FR-REF-034 | The cardinality of event fields that become metric attributes MUST be tested over `five`, the no-op allocation of a `Step` over `one`, and the non-interference of a logger and an observer over `one`. |
| FR-REF-035 | Every scenario S01 to S22 of `spec-stability.md` 8 except S15 MUST be covered by a test, or listed in 7.6 with the spec it waits for, so that FR-STB-005 can be checked. |
| FR-REF-036 | Every scenario A1 to A10 of `spec-viability.md` 13.2 MUST be covered by a test, or listed with the reason it needs no pack (A6). |
| FR-REF-037 | A scenario that waits for an unwritten spec MUST be a test skipped with the reason and an issue ID (testing.md 5.7), and MUST NOT be left out. |
| FR-REF-038 | The dynamic parts of X1 and X2, the scenarios S28 to S83 of 7.6, 7.9 and 7.11, S91 to S165 of 7.12, and the coarse-tick leg of S12 MUST be skipped tests with the issue ID of the spec they wait for until it exists. |
| FR-REF-039 | Each pack that ships MUST be run through the scenarios that its content allows and the result recorded with it (FR-STB-015), as its `expected/` file and the report of the run. |
| FR-REF-040 | The configuration documents of `testdata/chains/` (not the files of `expected/`, `bands/` and `scripts/`) MUST be the seed corpus of the loader fuzz test. |
| FR-REF-041 | Every group of every variant on the base except `second` (4.4.7, unbalanced by design) MUST have a static `IncomeBP` (FR-VIA-060) of at least 8 000, and the highest of a pack MUST be at most 1.5 times its lowest; a group without an `IncomeBP` (no labour share, as in `famine`; a recipe shared by several roles; a thin gatherer group named in `expected/<pack>.toml`) MUST be listed there with its reason, and a test over the solver MUST check every pack. |
| FR-REF-042 | The variant `one` MUST pass the predicate of X4 (7.4) from its warm start over 1 095 ticks on seeds 1 to 4, with the invariants of 7.5 asserted every tick; the word "stable" is not used of the pack (`spec-stylised-facts.md` 7.3.2). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-REF-001 | Loading and validating `five`, with V2 and V3 under the size gate, SHOULD take under one second on the reference machine (NFR-CFG-001). |
| NFR-REF-002 | The integration tests on `one` SHOULD take under one minute together, and the golden test of `five` under one minute (placeholders from the cost model of 8.2; measured in the slice). |

## 16. Open questions

1. **Bands.** The method is answered by `spec-stylised-facts.md` 8 and the keys by its 3.6
   and 5.11; no value exists until a first run that a human reviews. *Next step:* STY 8.4,
   after a first run of `one`, `three`, `five` and `famine` that a human reviews
   (FR-REF-024). Until then the band scenarios report.
2. **Trade between regions.** `two`, `three` and `five` run as sets of regions that do
   not trade. The edge table, the carrier and the rumour model are REG's, TRN's and INF's (written), and the merchant is MER's (REG, TRN, INF and MER are written: `spec-regions.md` 12.4 gives the edges, `spec-transport.md` 3 and 17.4 the modes and a haul recipe, `spec-merchants.md` 19.1 the merchant and consignee roles). *Next step:* add `overlay/trade.toml` (with the merchant role) and the
   dynamic parts of X1 and X2; the packs are built so the regions are viable alone, so
   the overlay adds trade and removes nothing.
3. **A faucet.** Resolved: `spec-money-supply.md` 17.6 defines the overlays and S13 is runnable (4.5, 7.2). EXT may add more flows.
4. **The coarse-tick leg of S12.** Needs the pack restated at five days a tick (durations
   divided by five, per-tick quantities multiplied by five, `ExpectedWorkTicks` below 73).
   *Next step:* after CAL's experience; author `overlay/coarse5.toml`.
5. **Warm start with capacity rent. Answered in round 5** (6.4): the nominal programme
   supplies the table. *Open:* whether the nominal table is a better start than the
   full-labour one in a run (S22 compares warm and cold, not two warm tables). *Next step:*
   a `famine` run from the nominal and from the reference table.
6. **Raw commodities and V1, the unsourced cycles. Answered in round 5** (6.1 to 6.3).
7. **The lean reserve and the stagger of a seasonal job.** The prototype's rule (use rate
   times the ticks to the first completion; jobs spread by the offsets of VIA 8.5.2) is
   one reading. *Next step:* the Go solver's table replaces it (FR-REF-007), and a run
   shows whether growers' stocks last until the first harvest (S18 and the first year of
   the invariants).
8. **Wants in the solver. Answered in round 5:** the programme carries the wants as a fixed
   final demand (`spec-viability.md` 3.2, FR-VIA-055); the census of `one-demand` is that
   reading. *Next step:* when the Go solver exists `warm-one-demand` is regenerated by it.
9. **Tiny groups.** The line gatherers are 4 to 9 households. *Next step:* S10-style
   consistency check on `five` after the slice has run; if the thin markets dominate the
   noise, raise the gatherer yields so the groups grow.
10. **The expected warnings.** Derived by hand for a few; the full list is the Go
    validator's. *Next step:* the first run fills `expected/`.
11. **The shock script format** is this spec's: `spec-integration.md` exists and fixes the
   calls (`SetYieldShocks`, `AddAgent`, `Direct`), not a file format (7.1.3). *Next step:*
   none; INT may offer one if a second consumer wants it.
12. **Pack size against test time.** `five` is the largest world the first consumer
    needs; whether it is too slow for every CI run is for the slice (NFR-REF-002).

## 17. Where this lives in the code

| Concern | Home |
|---------|------|
| The documents of 4 | `testdata/chains/` (architecture.md 2.6) |
| The expected results, bands and shock scripts | `testdata/chains/expected/`, `bands/`, `scripts/` |
| The golden files | `testdata/golden/` (`off/one.txt` and `off/five.txt` of 8.5; `off/events-<pack>.txt`, `spec-observability.md` 9.5.3, and the module-on sets `<module>/` and `all/`, testing.md 7.1.1; `experiment-one.txt`, `spec-experiments.md`; `stylised/<pack>.txt` and `stylised/<pack>.stability.txt`, `spec-stylised-facts.md` 7.3.3 and 8.5; the last two are tied to no module set and stay outside `off/`) |
| The records of a calibration | `testdata/calibration/<pack>/<stage>-<n>.txt` (`spec-calibration.md`), and its overlays in `testdata/chains/calibrated/` (4.1) |
| The scenario tests, invariants, golden, determinism, X4, X5, X8 that need no `indicator` | the root package's external test package (`economysim_test`), in files named for the scenario group; skipped with `-short` |
| The scenario tests that read an IND preset (REF) and the fact tests of STY | `indicator/scenario_*_test.go` and `indicator/stylised_*_test.go`, package `indicator_test`, sharing `stylised_helpers_test.go` (the recording observer, owned by STY); each re-runs its pack and seed and keeps its own small recorder, because FR-IND-001 lets no other package import `indicator` |
| The loader, validator and fuzz tests that read the documents | `config/`, reading `../testdata/chains/` |
| The regeneration test of the warm-start tables, A2, A3, A5, A10 | the root package, with the solver (`viability.go`) |
| The recording observer and the measure helpers | `_test.go` files (testing.md 3.6): helpers that set up, not helpers that compare |
