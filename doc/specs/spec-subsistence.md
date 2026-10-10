# Spec: Subsistence: household production for own use, the floor under market failure

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SUB-NNN` and
`NFR-SUB-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: module, off unless `Config.Modules.Subsistence` is present
([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md) decisions 1 to 4;
`spec-config.md` 3.2). Version 1 (`vision.md` section 4: "the subsistence floor needs a
minimal endowment, not ownership or rent"). Needs `spec-production.md` (3, the recipe; 4.2,
the refusals; 5, jobs; 6, the yield table), `spec-agents.md` (3.3 and 3.4, roles and the
tradable set; 5, upkeep and `Hunger`; 6.4, the view; 7.1, the default `Work`),
`spec-welfare-and-wages.md` (2.1, income; 4.3, the observed wage; 5, the target-income
rule), `spec-resources.md` (5, the seam and the ration; 10, open access),
`spec-spawner.md` (5, the failure ladder), `spec-estates.md` (2.6, relief),
`spec-ledger.md` (7 and 8, reasons and recipes), `spec-regions.md` (5 and 6, overrides and
relocation), `spec-viability.md` (3 and 4, the steady state) and `spec-config.md` (3, 4, 7).
Land and rent (LND) are later; this spec leaves a hook (section 9.5).

**Every number in this spec is a placeholder.** Nothing has been run in the library,
bazaarBot has not been run, and no acceptance band exists (notes N-015, N-027). The two
scripts of section 17 are small integer models of the rules of this spec, run in Docker
Python; they are not claims about the library and no default is claimed to be stable. The
example settings of section 16 are invented and flagged as examples; the core text names no
setting.

Terms in **bold** on first use are new domain terms. They must be added to the glossary in
`doc.go` before any code uses them (architecture.md 3.2). "Consumer" means the library
consumer (a game or server), never a household (N-001, N-017). The word *floor* already has
several uses in these specs (the work floor, the cost floor, the resolution floor); this
spec says **subsistence floor** for the whole mechanism of the roadmap row, and uses the
terms **own-use recipe** and **own-use mode** for its parts, so that none is mistaken for
another.

## 1. Purpose and scope

A market can fail for a household without failing for the economy: its goods do not sell,
or the staple is dear beyond its money, or nobody sells the staple at all. The hard budget
(`spec-clearing.md` 3.3) stops the negative balance of the original and the cost is the
budget loop (STB-L11): no cash, no bids, no inputs, no output, no income. The failure
ladder (`spec-spawner.md` 5) ends in removal, and relief (`spec-estates.md` 2.6) is paid
from a pool that a failing region empties (STB-L39). The register lists the missing piece as
D15, "a floor under market failure: household production outside the market, and relief",
and the roadmap row is *household production outside the market (growing or making basics
for oneself) as the floor under market failure, so a failed market does not mean
starvation*. This spec decides:

- what the activity **is**, as data: an **own-use recipe**, an ordinary recipe named on a
  list, with the constraints that make it a floor (section 3);
- why the market stays preferred: the productivity bound and the entry test (section 4);
- when a household takes the activity up and leaves it, by a rule on its own state with a
  hysteresis, and what the default `Work` does with it (section 5);
- what its output is in the ledger (section 6) and how it counts in the welfare ratio and
  the observed wage (section 7);
- where it sits against the failure ladder (section 8), and how it draws on an endowment
  (section 9), a season and a shock (section 10);
- what it does to markets, with a model (section 11);
- parameters, validation, observability, checkpoint state, worked examples in two unrelated
  settings, the models run, the entries for the stability register, the notes settled,
  requirements and open questions (sections 12 to 21).

Out of scope, with the spec that owns it: **migration**, whether, when and where a
household moves (`spec-migration.md`, MIG; section 5.8 states the seam); **relief** and the
pool (`spec-estates.md`, EST); the **failure ladder** (`spec-spawner.md`, SPN); tenure, rent
and who may draw on an endowment (`spec-land.md`, LND, later; section 9.5 is the hook); the
endowment itself (`spec-resources.md`, RES); the target-income rule and the welfare ratio
(`spec-welfare-and-wages.md`, WEL; this spec only says what they count); storage
(`spec-storage.md`, later); the tick phases (`spec-world.md`, WLD).

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later sections give
the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is household production for own use, as data? | An **own-use recipe** is an ordinary recipe (`spec-production.md` 3) named on a list: a module default list and a per-role row that replaces it. It must be a **source** (no inputs, no required tools) and **sited** (it runs only where the yield table gives it a positive factor). It is not a flag on a recipe and not a second recipe format | One recipe format (vision goal 5). A flag on the recipe could not differ by role. Source: a floor that needs a market input is not under the market. Sited: availability is stated per region, in the yield table that already carries climate (N-030, N-104) |
| 2 | What is it in the ledger? | The output is created by an ordinary recipe execution, reason `recipe_output`, into the household's own account, and eaten by the ordinary upkeep (`consumption`). No new reason, no in-kind channel, no new account | Conservation and the audit are unchanged (`spec-ledger.md` 8.1, 8.2); a separate channel would be a second faucet that every identity, every event and every test must learn (N-107) |
| 3 | Why is the market preferred? | Two guards. (a) A bound on productivity: the own-use recipe yields less upkeep per labour tick than the market's surplus factor, `Cov < SurplusBP` (4.2), checked as a warning. (b) An entry test: the mode starts only when the household's own secure upkeep is short (5.3), so a household that the market feeds never takes it | The bound is the labour share to subsist: the market chain makes `SurplusBP / 10 000` baskets per labour tick, so it needs a smaller share of a household's ticks than the floor does (4.2). The entry test is dynamic: even a generous own-use recipe is not used by a household that is fed |
| 4 | When does a household take it up and leave it? | The world computes a flag, **own-use mode**, from the household's own state only: stock, money, `Hunger`, and the price of its own market. Enter when its **secure ticks** (stock cover plus the ticks its money can buy) fall below `EnterTicks`; leave when its stock cover reaches `ExitTicks`; after `RetryJobs` own-use jobs the next free decision is a role job (5.3, 5.5) | Own state, not an aggregate (roadmap, "Macro is measured, never set"). Leaving by cover and not by money keeps a household from flapping in an empty market where its money buys nothing; the retry keeps it from living on the floor after the market has recovered (model, 11.2) |
| 5 | Who computes the flag, and who acts on it? | The world computes it, like `Hunger`, `Privation` and the work-floor count (`spec-agents.md` 5.4.1, 7.1.3), because a `Strategy` is a pure function and keeps no state. The default `Work` acts on it; a consumer's strategy reads `OwnUse` from the view and may ignore it | The hysteresis needs state; the decision stays the strategy's (`spec-agents.md` 6.2) |
| 6 | How does it meet the target-income rule (STB-L14)? | In the mode the work share is 10 000: survival comes before the target (`spec-agents.md` 7.1.3). The survival rule made a household work harder at its role; the mode gives it something that works when the role does not feed it | The survival rule is unit-elastic and "not a damper of L14" (`spec-agents.md` 7.1.3); the mode changes what the hungry household works at, not the elasticity of the role |
| 7 | Where does it sit in the failure ladder? | Rung 0, the household's own response. It changes none of the ladder's rules: `Privation` counts unfed ticks whatever fed the household, and a role change abandons an own-use job like any job (8) | The ladder is SPN's. The mode lowers the pressure that reaches it by feeding the household |
| 8 | How does it count in the welfare ratio and the observed wage? | The units created by an own-use job are an in-kind receipt in `income_t`, valued at creation at the prices used for `cost_t`; they and the job's ticks are **excluded** from the observed wage (7) | Without it a household that feeds itself reads a ratio of 0 and draws relief and a migration signal; with it counted in the wage, the occupation signal and the cost anchor would be polluted by labour that is not market labour |
| 9 | How does it draw on an endowment? | Through `Draws` like any recipe: open access, the ration of `spec-resources.md` 5.3, no priority. LND's hooks apply to its requests as to any (9) | N-108; RES 10 |
| 10 | What does it do to markets? | A household that cannot afford a staple feeds itself and its bids shrink with its stock, so the demand of the cash-bound falls faster than a fixed expenditure. In a stylised model it removes the shortage pin and 97% to 100% of the unfed household-ticks when it can feed a household; it does not anchor the interior and it can turn the pinned shortage into the pinned glut (11) | The first price-responsive flow for a staple that is not invented demand (N-043, N-052); the cost is stated |
| 11 | Can it interrupt a job? | No. It is chosen at a free decision. A household in a long job cannot take it until the job ends (5.7) | `spec-production.md` 5.2.2: a job is not interruptible by choice. The lean season of a long-job role is open question 1 |
| 12 | What happens at a relocation? | The mode and its counter are dropped; the first evaluation is in the new region (5.8) | The own-use recipe, its yield and its endowment are the old region's |
| 13 | Does it draw a random number? | No. Every rule is a function of the household's state with a total order | Determinism (architecture.md 7.3, 7.4) |
| 14 | New region-keyed tables? | None. A region differs through the yield table and the endowment rows, which are already region-keyed (3.5) | N-104: do not invent a per-region field with its own rules |
| 15 | New loops? | Five new entries (A2 to A6) and two merged (A1 into L11, A7 into L18, L40 and L46) (18): the budget loop is damped, a staple's demand becomes price-responsive, labour can be withdrawn to the floor, the commons can be drawn down by the failing, the floor can be too slow for the ladder, a household can be held on the floor | `spec-stability.md` 9: a spec that adds a loop and no entry is not done |

## 3. The own-use recipe as data

### 3.1 What it is

3.1.1. An **own-use recipe** is a recipe (`spec-production.md` 3.1) that a household may
run for itself when the market does not feed it. It has no field of its own: what makes it
an own-use recipe is that its ID is on the effective own-use list of the household's role
(3.3). Everything else is the recipe's: a `Duration`, outputs, a yield by region and season
(`spec-resources.md` 7), optional `Draws`. A job of an own-use recipe is an ordinary job,
one per agent, started, continued, completed and abandoned by `spec-production.md` 5.

3.1.2. The recipe format is not extended. The words for the activity in a setting appear only
in the examples of section 16; the core knows a recipe and a list.

3.1.3. No code branches on a recipe, commodity or role identifier (architecture.md 9.1).
The own-use list is data, the covered set (3.3.2) is derived from data, and the mode (5)
reads numbers.

### 3.2 The module and its rows

The module is `Config.Modules.Subsistence`, a pointer: nil is off, non-nil is on
(`spec-config.md` 3.2). It needs the recipes and the roles, which always exist, and no other
module; the resources module is needed only by an own-use recipe that has `Draws`
(`spec-production.md` 10.2).

```go
// SubsistenceConfig is the subsistence module. Plain data.
type SubsistenceConfig struct {
	Recipes    []string      // the default own-use recipes of every role, sorted by recipe ID
	Roles      []RoleOwnUse  // the role-keyed rows, sorted by role ID
	EnterTicks int64         // the mode starts when secure ticks fall below this (5.3)
	ExitTicks  int64         // the mode ends when stock cover reaches this (5.3)
	RetryJobs  int64         // own-use jobs in a row before a role job is tried; 0 never (5.5)
}

// RoleOwnUse replaces, for one role, the default list of own-use recipes.
type RoleOwnUse struct {
	Role    string
	Recipes []string // may be empty: the role has no own-use recipe
}
```

3.2.1. **The rows are the replace mode** of `spec-regions.md` 5.1.2 applied to a role-keyed
table: the effective list of a role is its row's list if it has a row, else `Recipes`. An
empty row is the way to say that a role has none (a role that must not subsist: a closed
role that only external agents fill, a household that lives on a grant). One row per role
(`ErrDuplicateID`), a row naming an unknown role is `ErrUnknownRole`, absence is the default
with no warning (`spec-regions.md` 5.3.3).

3.2.2. **File layout** (`spec-config.md` 6.2; the spelling of the module table is open until
N-102 is closed, and follows `[modules.information]`): the module is the table
`[modules.subsistence]` with the keys `recipes`, `enter_ticks`, `exit_ticks` and
`retry_jobs`, and a role row is `[modules.subsistence.role.<id>]` with the key `recipes`, so
that layers merge by key (`spec-config.md` 6.4).

3.2.3. **Canonical order:** every list by ID, rows by role.

### 3.3 The effective list of a role and the covered set

3.3.1. The **effective own-use list** of a role is the list of 3.2.1, sorted by recipe ID,
at most `MaxOwnUseRecipes` long. A recipe may be on the lists of many roles.

3.3.2. The **covered set** `F` of a role is the set of its upkeep commodities
(`spec-agents.md` 5.2, in any region) that are a non-discard output of at least one recipe
on its effective own-use list. `F` is derived at construction, in canonical commodity order.
A role whose `F` is empty is never evaluated for the mode (5.6) and draws the warning
`own_use_no_upkeep` (13.3): its own-use recipes cannot feed it.

3.3.3. A role with an empty `Recipes` list (a trader, a carrier, a household on a grant,
`spec-agents.md` 3.3.1) may have a non-empty own-use list and takes the mode like any
other.

### 3.4 What an own-use recipe must be

Each rule has a sentinel and a failing-case test (13.2).

3.4.1. **A source.** It has no `Inputs` and no `Requires` (`ErrOwnUseNotSource`). A floor
that needs a purchased input or a tool depends on the market it is the floor under: a
household that cannot buy an input cannot run the recipe. An input that the recipe returns
itself (a cycle of the recipe's own) is not allowed in version 1 either (open question 3).

3.4.2. **Sited.** It has `Sited` set (`ErrOwnUseNotSited`). An unsited source runs everywhere
at a factor of 10 000 unless a region says otherwise, which would put a free source of the
staple in every region the pack ever adds. A sited recipe runs only where the yield table
lists it (`spec-production.md` 6.2), so a pack states, region by region, where a household
can subsist. This is the single place where the regions differ (3.5).

3.4.3. **Not also a role recipe.** A recipe may not be on both the effective own-use list and
`Role.Recipes` of the same role (`ErrOwnUseOverlap`). The classification of a job, of the
income it earns (7) and of the ticks it takes is by the pair (role, recipe), so a recipe
that is both would make one job two things. A pack that wants a role to gather its own staple
as part of its trade lists the recipe in `Recipes`, and then it is not subsistence.

3.4.4. **Bounds.** At most `MaxOwnUseRecipes` (8) entries per list, each a known recipe
(`ErrUnknownRecipe`), none twice in a list (`ErrDuplicateID`); `EnterTicks`, `ExitTicks` and
`RetryJobs` within the bounds of 13.1.

### 3.5 Regions differ by the yield table and the endowment rows (N-104)

The module has no region-keyed table and no per-region parameter. Whether a region offers
subsistence is whether its yield row for an own-use recipe is positive
(`spec-resources.md` 7, `spec-production.md` 6.2) and, for a recipe with `Draws`, whether its
endowment can supply an execution (`spec-resources.md` 9.2). How rich the subsistence is
there is the yield factor and the endowment's capacity. A region with no floor (a walled
town, a barren rock) has no row, or a row of 0. Nothing else is needed; adding a table would
say the same thing twice (the reason `spec-resources.md` 7.1.2 gives for refusing a climate
entity). This applies note N-104 and the mechanism of `spec-regions.md` 5.1.

### 3.6 What the rest of the specs do with the list

| Where | What changes | Edit (section 19, handoff h) |
|-------|--------------|-------------------------------|
| `spec-production.md` 4.2, FR-PRO-008 | a request is on the role's list if it is on `Role.Recipes` or on the effective own-use list; the first refusal check is still `not_on_role` | yes |
| `spec-production.md` 9.3 | an own-use recipe **never sets a cost anchor**: the anchor is the smallest `unit_cost` over recipes of `Role.Recipes`; the cost of a household's own labour is not what a seller's price is anchored to | yes |
| `spec-production.md` 10.4 | `unused_recipe` counts an own-use list as a use | yes |
| `spec-agents.md` 3.4 | the tradable set of a role includes the non-discard outputs of its own-use recipes (a household may hold and sell what it makes) | yes |
| `spec-viability.md` 3.1, 4.2 | an own-use recipe that is on no role's `Recipes` is **not an activity** of the reference programme and does not make a commodity producible: the steady state assumes the markets work, and the floor is for when they do not | yes |
| `spec-config.md` 7.5 | the structural producibility pass may keep counting it; V1 refines (an upkeep good made only by an own-use recipe is `unproducible_in_realm`) | optional |

## 4. Why the market stays preferred

### 4.1 Productivity per labour tick

For an own-use recipe `r` in a region, with an effective factor `E_r` (`spec-production.md`
6.3, in basis points, including the abundance of a drawn endowment), a household's upkeep
`u_c` of a covered commodity `c` (`spec-agents.md` 5.2, per household) and a non-discard
output of `q_{r,c}` units per execution, the **coverage** of `c` by `r` is

```
Cov(r, c) = q_{r,c} * E_r / (Duration_r * u_c)      basis points of one household's upkeep of c per labour tick
```

(integer, truncated, 128-bit intermediates). `Cov` is 10 000 when one tick of work makes
exactly one tick of upkeep. The coverage of a recipe for a role is `max over c in F of
Cov(r, c)`, and it is the figure that `FloorBP` of the validation uses (13.3).

### 4.2 The bound

In the reference programme of `spec-viability.md` 3.2 the realm can make `lambda*` upkeep
baskets per household per labour tick, `SurplusBP = floor(lambda* * 10 000)`, so a household
needs to work only `LabourShareBP = ceil(10^8 / SurplusBP)` basis points of its ticks to
make its own upkeep through the chain (`spec-viability.md` 3.2: the labour share to
subsist). On an own-use recipe a household that works every tick makes `FloorBP / 10 000`
baskets, so it needs `ceil(10^8 / FloorBP)` of its ticks. An own-use recipe that needs
less labour than the market chain would be a better livelihood than the market everywhere it
runs. So

```
FloorBP(r, region) = max over c in F of Cov(r, c)   <   SurplusBP of the region
```

is the **bound**. It is checked at validation as the warning `own_use_beats_market` for every
region where the recipe has a positive factor, against the surplus factor of the autarky
programme of that region (`spec-viability.md` 4.4, 9.5) and, for a region that trades, the
pooled one; the smaller is used. It is a warning and not an error because the check is
size-gated (`viability_check_skipped`) and optimistic (`spec-viability.md` 4.6), and because
a setting may want a generous floor and say so by ignoring the warning. The solver is asked
to expose the per-region figure in its report (handoff h, `spec-viability.md` 10.3).

4.2.1. **Why this inequality and not another.** A household chooses between a labour tick
spent in the market chain and one spent on the floor. A tick in the chain makes
`SurplusBP / 10 000` baskets of upkeep reachable, by wage and purchase; a tick on the floor
makes `FloorBP / 10 000` baskets directly. The market is preferred where the first is larger,
which is the same as the household needing a smaller share of its ticks to subsist. The
inequality uses only numbers that exist: the recipe, the yield table and the solver's figure.

4.2.2. **The bound is a ceiling and not a target.** A floor far below the bound is weak
(4.5); one just below it makes the market only slightly better than home production, and
the household's choice then rests on the entry test (4.4) alone.

### 4.3 What the bound does not cover

It is a steady-state statement. In a run a price can leave the equilibrium: a staple at
twice its reference price makes the market buy half the baskets, and the floor can then be
the better tick (model, 11.2: it is, for the households whose wage cannot buy the staple).
That is the intended use. The bound cannot say that the floor never beats the market at a
price of a failing market; it says that, where the market works, it does not.

### 4.4 The second guard: the entry test

The bound could be met and the floor still be used by a household that the market feeds, if
the rule for taking it up looked only at productivity. It does not: the mode starts only
when the household's own **secure ticks** (5.2) fall below `EnterTicks`, which a household
with enough stock or enough money at the going price never does. The first guard is a
property of the pack; the second is a property of the rule.

### 4.5 The floor may feed less than a household needs

`FloorBP` below 10 000 is legal: the floor then feeds a fraction of the upkeep (the
informational `own_use_partial`). A household on such a floor is fed in part: `Fed` counts a
household only if every upkeep commodity met its full need (`spec-agents.md` 5.4), so it
stays **unfed**, its `Privation` climbs more slowly or not at all, and the ladder acts as it
would (section 8). The model shows that below 10 000 the mode can be worse than none
without the retry of 5.5 (11.2); the warning `own_use_trap` fires for that combination.

## 5. Taking up the activity and leaving it

### 5.1 Own state only

The mode reads, for a household, only: its **stock** of each covered commodity, its
**money**, its **`Hunger`**, its effective **upkeep**, the **price of the covered commodities
in its own market** (the market report, `spec-agents.md` 6.4.1, free to every agent), its
mode and its count of own-use jobs. It reads no regional average, no other region's price,
no count of other households and no measure. A test changes any other household's state, the
regional wage and every other region and finds the flag of this household unchanged
(FR-SUB-015).

### 5.2 Cover, cost, afford and secure

For an agent of weight `w`, with `U_c = w * u_c` its effective upkeep of `c`
(`spec-agents.md` 5.2) and `F` the covered set of its role (3.3.2), at the start of the
production phase, after the continuations and completions of the tick (5.4):

```
Cover   = min over c in F of  floor( Stock_c / U_c )               ticks of upkeep held in stock
Cost    = sum over c in F of  U_c * Price_c                        money a tick of the covered upkeep costs at the going price
Afford  = 0                                  if Hunger > 0
        = floor( Money / Cost )              otherwise             ticks of upkeep the money buys
Secure  = min( Cover + Afford, MaxOwnUseTicks )                    ticks of upkeep secured
```

`Stock_c` is the agent's whole balance (no reservation exists at that point,
`spec-agents.md` 6.4.1), `Money` its available cash, `Price_c` the published price of `c` in
its region (`spec-clearing.md` 5.4; the effective reference price if the market has never
traded). The products and the sum are 128-bit checked; `Cost` is at least 1 because every
upkeep quantity and every price is at least 1. Every term is a total for the agent, so a
cohort of `w` identical households and `w` agents of weight 1 have the same `Cover`,
`Afford` and `Secure` (FR-SUB-016, `spec-scale.md` 4.3).

5.2.1. **Why `Afford` is 0 after a hungry tick.** A household that went unfed last tick has
shown that, whatever its money would buy at the posted price, the market did not deliver:
nobody sold, or its bid was beaten. Counting the money would keep it out of the mode in the
one case in which the mode is for. This is the household's own failed order, read as the
state `Hunger` that the world already keeps (`spec-agents.md` 5.4.1), not as a new record.

5.2.2. **Why `Afford` counts the whole of `Money`.** Tier 1 of the budget spends on upkeep
before any reserve (`spec-agents.md` 7.4.3), so the whole balance is available to buy the
covered goods. The reserve rule limits buffers, not food.

### 5.3 The mode: enter, leave, and the hysteresis

`OwnUse` is a flag of the agent, false at creation. In each tick, for an eligible agent
(5.6):

```
if not OwnUse and Secure < EnterTicks:   OwnUse = true,  OwnUseJobs = 0      (enter)
if OwnUse     and Cover  >= ExitTicks:   OwnUse = false, OwnUseJobs = 0      (leave)
```

5.3.1. **The band.** `ExitTicks` is greater than `EnterTicks` (`ErrOwnUseBand`). The entry
test uses `Secure` (stock and money), the exit test uses `Cover` (stock alone), so the band
is not symmetric on purpose: a household enters when it cannot be sure of enough food and
leaves when it holds enough. A household that has entered with cover `c < EnterTicks` leaves
only after its stock has grown to `ExitTicks`, which takes at least one job of floor output
above its eating. That is the damper SUB-D1 of section 18.

5.3.2. **Why leave by stock and not by money (a variant that was run and refused).** An exit
"when the last bid was filled and the money buys `ExitTicks`" lets a household that the
market has just fed leave at once. In the model of 17.1 it raised the unfed household-ticks
from 16 299 to 63 650 and the mode changes from 21 168 to 107 106 for the same floor
(`models/sub/staple.out`, last block): the household leaves, the price rises, it enters
again. An exit by stock only is slower and steadier. A household with money in an empty market
is also the case the entry rule of 5.2.1 treats: it enters on the unfed tick and stays until
it holds stock.

5.3.3. **Reachability.** The stock that purchases can build is bounded by the household's
own targets: after eating, the default strategy holds `CoverTicks` ticks of upkeep
(`spec-agents.md` 7.3, `Keep` less `Reserve`). An `ExitTicks` above `CoverTicks` can be
reached only by the floor's own surplus, so a household whose floor does not out-produce
its eating can never leave by cover and leaves only by the retry. The warning
`own_use_exit_unreachable` fires for `ExitTicks > CoverTicks` (13.3); the model's last case of
11.2 shows the hazard (1 069 households stuck on the floor, at a price of 100).

### 5.4 When the flag is computed, and by whom

The world computes the flag in phase 2 of the tick (`spec-agents.md` 4), after the
continuations and completions of jobs and after the snapshot of the endowments
(`spec-resources.md` 5.2), for every eligible agent in agent-ID order, and before `Work` is
called for any agent. The flag step is a pure function of the state above (the package
`internal/ownuse`, 22); it draws nothing and does not depend on the order of agents. It is
computed for agents in the middle of a job too, so that the flag is current when the job
ends. The view passed to `Work` and `Trade` carries `OwnUse`, `OwnUseJobs`, `Cover` and
`Secure` (6.4.1 of `spec-agents.md`, handoff h) so that a consumer's strategy can use them
and does not recompute them.

### 5.5 What the default `Work` does in the mode

The default strategy (`spec-agents.md` 7.1) is changed in one place: before the recipe choice
of 7.1.4 it applies this rule to an agent that is free.

1. **Not in the mode:** nothing changes. The work share is the target-income rule's (7.1.2),
   the recipes are the role's. An own-use recipe is never chosen outside the mode.
2. **In the mode:** the work share is 10 000 (survival comes before the target,
   `spec-agents.md` 7.1.3).
3. **Which recipe.** Let `F_short` be the covered commodities whose cover is below
   `ExitTicks`. Among the own-use recipes with `MaxExecutions >= 1` the agent requests the one
   with the greatest `sum over c in F_short of Cov(r, c)` (4.1), compared as exact fractions,
   ties to the lower recipe ID. The value is a quantity of upkeep per labour tick and uses no
   price and no belief. A recipe with value 0 is not requested.
4. **How many.** `k = min( w, MaxExecutions, max(1, ceil( w * (ExitTicks - Cover) / ExitTicks )) )`.
   A weight-1 agent runs 1. A cohort puts in the mode's job the share of its households that
   is short of cover, so its labour moves in steps of `1 / w` and not all at once (the form
   of D28, `spec-agents.md` 9.3). The other households of the cohort rest during the job:
   an agent has one job (`spec-production.md` 5.1.2), so a cohort cannot work an own-use
   recipe and a role recipe at once. This is exact at weight 1, which is the first consumer's
   case (N-030), and an approximation above it (open question 7).
5. **The retry (D3).** Every started own-use job adds 1 to `OwnUseJobs`; a job of any other
   recipe resets it to 0. If `RetryJobs > 0`, `OwnUseJobs >= RetryJobs` and the agent has a
   role recipe that is eligible by 7.1.4 (it can run and is worth starting), the request is
   that role recipe instead of the own-use recipe. The role job is a test of the market: if
   the household earns, its `Afford` rises; if not, the mode continues. If no role recipe is
   eligible, the own-use job goes on.
6. **Nothing runnable.** If no own-use recipe has `MaxExecutions >= 1` (a zero yield, no
   site, no endowment, `spec-production.md` 4.2) the household is **blocked**: `Work` treats
   the tick as if it were not in the mode, and the flag stays. The household is counted in
   `Blocked` (14.1).
7. **The halt does not apply.** The production halt (`spec-agents.md` 7.1.5) does not apply
   to an own-use recipe: its stop is the exit cover. The executions of 4 are bounded by the
   shortfall, so a job overshoots by at most one execution's output per household.

5.5.1. A consumer's strategy sees the same flag. The `Directed` strategy's directive
overrides `Work` as for any recipe (`spec-agents.md` 10.4); the fallback is the default.

### 5.6 Who is evaluated

The flag is evaluated for an agent that: is not an estate or other pseudo-agent; is not a
trainee (its board comes from the trainer, `spec-skill-barriers.md` 4.2, as for the ladder,
`spec-spawner.md` 5.2.3); is not away (on a transport trip, `spec-transport.md` 5.5.2, or on a migration journey,
`spec-migration.md`): such an agent has no `Work` and no view; has an effective own-use list with a non-empty covered set; and has
a positive effective upkeep. For any other agent `OwnUse` is false and its counter 0. An
external agent is evaluated like any other (its fallback strategy is the default).

### 5.7 No interruption

A job runs to its end (`spec-production.md` 5.2.2). The mode is acted on at a free decision,
so a household in the middle of a long role job cannot take the floor until the job
completes, and a household in the middle of an own-use job cannot return to the market until
it completes. The consequence for a role with long jobs, the *lean season*, is stated in
10.3 and is open question 1. The cost of the alternative, an abandon with a reason, is that
the inputs of the job are lost (`spec-production.md` 5.4.1).

### 5.8 Role change, weight change, removal and relocation

| Event | `OwnUse` and `OwnUseJobs` | Notes |
|-------|---------------------------|-------|
| Role change (`spec-spawner.md` 4.2) | `OwnUseJobs` reset to 0; `OwnUse` kept, then re-evaluated at the next phase 2 against the new role's list and covered set | an own-use job in progress is abandoned with `role_changed` like any job |
| Weight change (`spec-agents.md` 3.7) | unchanged | cover, cost and afford are totals and follow the new weight at once |
| Removal | dropped | the job is abandoned with `agent_removed` |
| **Relocation** (`spec-regions.md` 6), and the departure of a timed move (`spec-migration.md`) | **dropped: `OwnUse` false, `OwnUseJobs` 0**, at the departure and again at the relocation | the own-use recipe, its yield and its endowment are the old region's. MIG abandons an own-use job first, as for any job (`spec-regions.md` 6.3, `region_changed` of `spec-production.md` 5.4). SUB has no reason and no event of its own for the drop. The first evaluation is in the new region at its next phase 2, from the household's stock, money and the new region's prices |
| A shipment (`spec-transport.md`), or a journey while it lasts | the agent is away and not evaluated; its flag is kept (a journey dropped it at departure) | evaluated at the first phase 2 after the arrival |

The seam with MIG (`spec-migration.md`, N-090, N-110): MIG's trigger reads the household's
welfare ratio (WEL), which includes the own-use receipts (7), so a household that feeds
itself on the floor has a ratio near what the floor covers and not 0, and the pressure to
move is lower for it than for an unfed household with no floor. SUB exposes `OwnUse` and
`OwnUseJobs` in the household's view; MIG MAY read them as the household's own state. SUB
reads nothing of MIG. A household that has moved starts with no floor state (above).

## 6. The ledger form of the output

### 6.1 An ordinary recipe execution

The output of an own-use job is created at completion in the executor's account by the
ledger reason `recipe_output`, a faucet from the boundary, exactly as for any recipe
(`spec-production.md` 5.3.1, `spec-ledger.md` 8.2). A source recipe has no start step
(`spec-ledger.md` 8.2.1: a transform with only outputs). It is eaten, if it is an upkeep
good, by the ordinary consumption of phase 6 (`consumption`, `spec-agents.md` 5.3). The
household's stock, its lots, their ages and spoilage are those of any goods. Nothing in the
ledger knows that the job was an own-use job.

### 6.2 Why not an in-kind channel

A separate channel (a reason `own_use`, a direct credit to consumption, a household account
class) was refused for four reasons. (1) It would be a second faucet of goods with its own
identity to check, in a design where every faucet and sink is a named reason of a closed
registry and a tick's accounting is `recipe_input`, `recipe_output`, `disposal`,
`consumption`, `spoilage`, `wear` (`spec-ledger.md` 7.2; N-107). (2) The goods would not be
stock until eaten, so the household could not hold, sell or carry them, and the `Cover` of
5.2 would need a second term. (3) Spoilage, the earmarks of a reservation and the estate path
at removal would each need a rule for the second kind of good. (4) The audit and the sectoral
identities of `spec-ledger.md` 9 would change. The cost of the ordinary form is that the
goods are fungible with the household's other stock: the event and the welfare record
identify an own-use job by its recipe (the pair of 3.4.3), not the lots, and a household may
sell what it made (open question 2).

### 6.3 What the household does with the goods

They are stock. The default `Trade` keeps `Keep_c` and offers any surplus above it
(`spec-agents.md` 7.3.3, 7.3.5): a household that has made more than it needs sells the rest
at the going price, as it would sell anything. This is not a competitor in the sense of 4,
because the executions are bounded by the shortfall (5.5 item 4) and the mode ends at the exit
cover, so a surplus is at most the overshoot of one job, and a sale of it is a market sale
like any other. The mode raises no price belief and sets no cost anchor (3.6).

### 6.4 Conservation and the audit

The identities of `spec-ledger.md` 9.1 hold with the module on and off. For the testing of
"goods are conserved except by recipes" (testing.md 6.3) the units fixed at the start of an
own-use job equal the units created by `recipe_output` plus the pending units plus the units
dropped from pending, as for any job (`spec-ledger.md` 8.2.2). The module adds no ledger
reason (FR-SUB-003) and no line to `BoundaryFlow`.

## 7. The welfare ratio and the observed wage

`spec-welfare-and-wages.md` 2.1 defines income as the profit of the tick, with in-kind
receipts counted at the value they have in the agent's region (6.2 there). An own-use job is
the case that the definition has no place for: the household produces the good it eats, so
it has no sale to count. This section decides how it counts. WEL owns the formulas; the edits
are in the handoff.

### 7.1 What is counted

The units that an own-use job creates at completion are an **in-kind receipt** of the tick of
completion: `K^own_t = sum over the non-discard outputs o of the own-use jobs completed in
tick t of (units created) * Price_o(t)`. The units are those actually created, after the
carried remainder of the yield (`spec-production.md` 6.6). The income of the tick becomes

```
income_t = R_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t
```

and the welfare ratio is otherwise unchanged. Nothing is counted when the goods are eaten:
the receipt is at creation, and the consumption is part of `cost_t`, as the consumption of
bought goods is.

### 7.2 At the prices of `cost_t`

`Price_o(t)` is the price at which `spec-welfare-and-wages.md` 2.1 values the cost of the
tick's upkeep: the published price of the commodity in the agent's region in tick `t`
(`spec-clearing.md` 5.4), the effective reference price if it has never traded. The reason is
the property that makes the ratio mean something: a household that makes exactly its upkeep
reads `10 000`, whatever the price, because numerator and denominator move together. A
valuation at the reference price against a cost at the market price would give a household
on a floor a ratio of 0.1 when the staple is at ten times its reference, which says it is
starving when it is fed.

The price effect is the loop STB-L18 (in-kind valuation): a rise of the price raises the
measured income of the in-kind part. It acts on the target-income rule only, and in the mode
the work share is 10 000 whatever the ratio is (5.5), so it cannot make a household rest
while the mode holds (merged into L18, section 18).

### 7.3 Excluded from the observed wage

The observed wage is the occupation signal of the spawner (`spec-spawner.md` 6.1), the
pooled regional wage of the cost anchor (`spec-production.md` 9.3), and the premium of the
skill specs. It is the return to **market labour**. An own-use job is not market labour. So
both the receipt and the ticks are left out:

```
observed_wage = sum( income_t - F_t - G_t - K^own_t ) / sum( working household-ticks_t - own-use household-ticks_t )
```

where the own-use household-ticks of a tick are `k` for an own-use job of `k` executions in
progress (started, continued or completed), the same count `spec-welfare-and-wages.md` 4.3
uses for any job. Without the exclusion a household that spends a window on the floor would
show a large number of working ticks and little cash income, and pull the regional wage, and
with it every cost anchor, toward 0 in exactly the region where the floor is in use; with the
receipt in the numerator and the ticks in the denominator it would show the floor's
productivity as a wage. An own-use tick **is** a working tick for the work floor and for the
target-income rule: the household worked (`spec-welfare-and-wages.md` 5.3 reads the same
ring).

The record of a tick gains two integers, `K^own_t` and the own-use household-ticks, kept
with running sums like the others (`spec-welfare-and-wages.md` 2.3.4). A tick of the window
contributes to the wage only through the rest.

### 7.4 What follows

| Reader | Effect |
|--------|--------|
| Relief (`spec-estates.md` 2.6) | the ratio includes the receipts, so a household fed by its floor is below the relief line by less, or not at all; relief still tops up a household whose floor feeds it only in part |
| MIG (`spec-migration.md`) | the pressure to move, if it reads the ratio, is lower for a household on a floor than for an unfed household with no floor (5.8) |
| POP (`spec-population.md` 3.4) | a fed household is exposed to the schedules at its ratio; a household on a floor that feeds it reads a ratio near what the floor covers, below the replacement ratio of a market-fed one, so it forms households less and dissolves them more: a gentle check in place of the ladder's. `r_full` of N-109 for a floor-fed household is the floor's ratio |
| The spawner's signals `E`, `V` (`spec-spawner.md` 6.1) | unchanged: the wage excludes the floor, so a role's signal is its market wage |
| The cost anchor (`spec-production.md` 9.3) | unchanged for the same reason, and an own-use recipe never sets an anchor (3.6) |
| The work floor (`spec-welfare-and-wages.md` 5.3) | own-use ticks count as working ticks |

### 7.5 A sale of the surplus is counted twice

A household that makes more than it eats and sells the surplus counts the units once at
creation (`K^own`) and the money once when they are sold (`R`). The ratio of that window is
overstated by the value of the units sold, which is not conservative (it lowers relief and the
signal to move). The overstatement is bounded: the executions of a job are bounded by the
shortfall (5.5 item 4) and the mode ends at the exit cover, so the surplus is at most the overshoot
of a job and what the household holds above its `Keep`. The exact alternatives are a
provenance on each lot (state per lot, refused) and a ban on selling the output of an own-use
job (not enforceable on fungible stock). Open question 2.

### 7.6 What WEL changes (handoff h)

`spec-welfare-and-wages.md` 2.1 (the term `K^own_t` and 7.2 here), 2.3.3 (the longest
`Duration` that sets the effective window includes the recipes on the role's own-use list, so
the window holds the lump of an own-use job), 4.3 (the observed-wage formula of 7.3), 13 (two
more integers per record) and 12 (nothing). The requirements of WEL for the income and the wage are edited in the same change.

## 8. The failure ladder

The ladder is SPN's and none of its rules changes (`spec-spawner.md` 5). The mode is the
household's own response, before the world acts on it. The table says where it lands.

### 8.1 The rungs

| Rung | Trigger | What the subsistence floor adds |
|------|---------|----------------------------------|
| 0 | `Hunger > 0` or money below the reserve: work full time (`spec-agents.md` 7.1.3) | the survival rule made the household work harder at its role. In the mode the household works full time at an activity that does not need the market. The mode is entered on `Secure`, which is before the first unfed tick when the household has seen its stock and money run short |
| 1 | `Privation >= ChangePrivation`: change to a better viable role | unchanged. A role change abandons an own-use job (5.8). The rung is reached later or not at all, because the household is fed |
| 2 | ratio below `ReliefBelow`: relief from the pool | unchanged. The ratio includes the receipts (7), so a household that is fed on the floor draws less |
| 3 | `Privation >= RemovePrivation`: reduce by the unfed share, then remove | unchanged. A household that the floor feeds in part is partly unfed, and the ladder reduces that part |

`Privation` and `Hunger` are computed by the world from `Unfed` (`spec-agents.md` 5.4.1),
whatever fed the household: a unit eaten from the floor counts as eaten, a tick in which the
floor covered the upkeep resets `Hunger` and lowers `Privation` by one.

### 8.2 The floor can be too slow for the ladder

A household that enters the mode with `Cover` `c` is fed from stock for `c` ticks and, if the
job lasts `d` ticks, is unfed for `max(0, d - 1 - c)` ticks before the first output arrives
(the job started in tick `s` completes in the production phase of tick `s + d - 1` and the
output is eaten that tick, `spec-production.md` 5.2). In the worst case, `c = 0`, that is
`d - 1` unfed ticks, and `Privation` reaches `d - 1`. If that reaches `ChangePrivation`, rung 1
can change the household's role and abandon the job a tick before it completes; if it reaches
`RemovePrivation`, rung 3 can reduce it. The warning `own_use_slower_than_ladder` fires for an
own-use recipe with `Duration - 1 >= ChangePrivation` (13.3). The remedies are data: a shorter
recipe, a larger `EnterTicks` so that the household enters with more cover left, or a larger
`ChangePrivation`. This is the register entry SUB-A5.

### 8.3 Rung 1 and a running own-use job

The default spawner does not look at the mode: a household in the mode with `Privation` at
`ChangePrivation` is a candidate for a role change like any other (`spec-spawner.md` 6.4), and
the role change abandons the own-use job (`role_changed`). That is a cost, because the floor
was working. The alternative is to exclude from rung 1 a household that has a running own-use
job whose output will cover its need, which needs a field in the spawner's view and a rule
there. It is not made here; open question 4, and the view of the spawner gains `OwnUse` (a
read-only field) so that a consumer's spawner can.

### 8.4 Relief and the floor are substitutes

Both feed a household whose market livelihood fails: relief with money from the pool, the
floor with goods from the household's own labour and the endowment. The floor relieves the
pool (STB-L39, L63) and makes the household's collapse slower, and does not need the pool to
be full. Where both apply the household receives both: relief tops it up to `ReliefBelow` on a
ratio that already includes the floor. The unresolved
question is whether relief should be paid to a household that is fed by its own labour
(open question 8).

## 9. Resources, land and the commons

### 9.1 A draw like any other

An own-use recipe with `Draws` takes units of an endowment of its region through the seam of
`spec-resources.md` 5: the executions of a request are limited by the ration of the tick
(`spec-resources.md` 5.3), the take is all or nothing, and the abundance of the endowment is a
factor of the effective yield (`spec-production.md` 6.3). Nothing in the module changes any of
it (N-108). The recipe needs the resources module on (`ErrModuleDependency`,
`spec-production.md` 10.2).

### 9.2 Open access, and no priority

Every endowment is open access in version 1 (`spec-resources.md` 10.1). The requests of own-use
jobs are in the ration with all others; **they have no priority and no quota**. When the stock
is short, a household in the mode is cut in proportion to what it asks, like any household that
draws. This is the baseline against which tenure is measured, and it means that in a region
whose commons is exhausted the floor is exhausted with it: the floor is a share of the
commons, not a source beside it. A household that is cut to 0 has no runnable own-use recipe
(`no_endowment`) and is counted as blocked (5.5 item 6, 14.1).

### 9.3 A thin commons lowers the floor's yield

The abundance factor enters `E`, so an own-use recipe's `Cov` (4.1) falls as the stock does.
The bound of 4.2 is evaluated at the plan share of the endowment (`spec-resources.md` 11.2) in
the solver, and at the abundance of the tick in the mode (the view's `E`). A floor that
covers a household at a full stock may cover half of it at a thin one.

### 9.4 The commons loop

More households fail, more take the mode, the commons is drawn down, its abundance falls, the
floor yields less, more households are unfed. This is a reinforcing loop through the endowment
(SUB-A4, section 18), a third path to the open-access draw-down of STB-L90 beside recipe
choice and entry. The dampers are the ration (D53, which caps the draw at the stock), the
abundance curve (D54, a soft landing), the refuge of a logistic stock (D56) and the exit rule
(D1: a household leaves the mode when it holds stock, so it does not draw beyond its need). The
condition is that the commons regenerates at least what the households in the mode draw at the
population the region can feed: the same inequality as the Malthusian fixed point of
`spec-resources.md` 8, and `resource_unstable_plan` and `resource_overshoot` are its warnings.

### 9.5 The hook for LND

SUB asks nothing of RES. The hooks that LND will use (`spec-resources.md` 10.2) apply to a
request of an own-use job as to any: a refusal after `no_endowment`, an entitlement applied to
the request before the ration, a rent as a named money flow. Two things are stated here so
that LND does not have to guess. (1) An own-use recipe is identified for a role by the pair of
3.4.3; the request carries the recipe, so an entitlement may treat a request for an own-use
recipe differently (a right of the landless to draw for their own use). (2) SUB itself charges nothing and
reserves nothing: if LND closes the commons to a household, its own-use recipe is `no_endowment`
and the household is blocked, and the warning `own_use_not_available` (13.3) is the static
form of that.

## 10. Seasons, shocks and the lean season

10.1. **Seasons.** The yield of an own-use recipe by season is its row in the yield table
(`spec-production.md` 6.1, `spec-resources.md` 7.3): complete by season, an unlisted season is
0. A recipe that yields only in autumn has `zero_yield` in winter and the household is blocked
for the winter (5.5 item 6). Cover and the thresholds are in ticks and are not rescaled by the
calendar (`spec-calendar.md` 6.1).

10.2. **Shocks.** A failed harvest is the yield shock of `spec-production.md` 6.8 on the
own-use recipe, and it lowers `Cov` for the household at once; a blight of the commons is the
adjustment slots of `spec-resources.md` 6.3. SUB adds no slot.

10.3. **The lean season.** A household that is in the middle of a long role job cannot take the
mode until the job ends (5.7). For a role whose recipes last a season, the time at which the
household is most in need, the end of the stores before the next yield, is the time at which it is
busy. The remedies are the starting stock and money that the warm start gives for the longest
job (`spec-viability.md` 8.3.1, STB-L19) and, with more work than this spec does, an
interruption or a second job slot. They are open question 1. A computed check of the longest
gap in the year in which none of a role's own-use recipes can run, against the cover a
household can hold, in the style of V4 (`spec-viability.md` 4.5), is open question 5.

## 11. What it does to markets

### 11.1 A staple's demand gains a response

`spec-agents.md` 11.6 finds that a staple's demand elasticity is between 0 and 1: the
quantity is fixed while cash suffices, and the expenditure is fixed when cash binds. That is
the reason a staple's labour loop has no stable interior (STB-L14, L51) and why N-043 asks for
a price-responsive flow that is not invented. The mode is one. As the price of a covered good
rises, more households have `Secure` below `EnterTicks` (their stock runs down and their money
buys fewer ticks); they work the floor, whose output builds their stock; with stock at target
their bids for the covered good are zero until the stock falls. The quantity demanded then
falls faster than the fixed expenditure of a cash-bound household, over the range where the
floor covers, and the price at which a household stops bidding is set by its own state. It is
not a rationing of upkeep (refused by `spec-demand.md` 17, N-052): the household is fed, by a
different source. The response has a delay, the job's duration and the exit band, and a delay
in a balancing loop can overshoot (SUB-A2, section 18).

### 11.2 The staple model (a model, not the library)

`models/sub/staple.py` (17.1) is the labour-loop model of `spec-stability.md` 6.1 (1 000
self-employed producers under the target-income rule, supply unit-elastic in the price) with
1 370 consumer households of three units of upkeep a tick who earn a wage when not in the
mode, buy with a hard budget and follow the rule of 5. It starts 10% above the reference price
and runs 600 ticks; the table is the median seed of five. "Unfed" is household-ticks without
the full upkeep; `f` is the floor's output a tick, `Cov = 3 333 f` basis points here (upkeep
3).

| Case | Final price | Unfed household-ticks | Mode changes |
|------|-------------|-----------------------|--------------|
| No work floor (DMIN 1). Floor off | 10 000 (the ceiling) | 796 402 | 0 |
| `f` = 2 (`Cov` 6 667), no retry | 100 (the floor) | 503 926 | 3 049 |
| `f` = 2, retry 10 | 100 | 129 394 | 3 482 |
| `f` = 3 (`Cov` 10 000), no retry | 100 | 9 603 | 20 121 |
| `f` = 3, retry 10 | 100 | 16 299 | 21 168 |
| `f` = 4 (`Cov` 13 333), retry 10 | 2 650 (tail 2 526 to 2 857) | 23 007 | 188 706 |
| With the work floor of 4 000 bp (DMIN 146). Floor off | 1 502 | 82 665 | 0 |
| `f` = 2, no retry | 100 | 167 239 | 311 |
| `f` = 2, retry 10 | 100 | 41 232 | 622 |
| `f` = 3 or 4, retry 10 | 100 or 1 386 | 0 | 622 or 12 910 |
| An outage (no seller for 100 ticks at a held price). Floor off | 1 000 | 133 390 | 0 |
| `f` = 3 / 2 / 4, retry 10 | 1 000 | 12 330 / 132 990 / 10 960 | 2 740 / 2 740 / 21 920 |

What the model shows. (1) When the floor can feed a household (`Cov` of 10 000 or more) it
removes 97% to 100% of the unfed household-ticks of the pinned shortage (796 402 to 9 603
through 23 007; 82 665 to 0) and 91% to 92% of those of the outage (133 390 to 12 330 and
10 960). (2) A floor below the
upkeep (`f` = 2) helps only with the retry: without it the unfed household-ticks are 167 239
against 82 665 with no floor at all, because households stay on a floor that cannot build
their cover and earn nothing (the trap, SUB-A6). With the retry 41 232. In an outage a floor
below the upkeep does nothing (132 990 against 133 390). (3) The mode does not anchor the
interior in general. With `f` of 3 or less the price falls through the repeller to the lower
pin (the glut, all producers working every day): the floor replaces the shortage pin by the
other regime of STB-L14, and F5 is not removed. With `f` of 4 the price stays inside the band
(2 650 with 2 526 to 2 857 in the tail; 1 386 with the work floor), with each household
changing mode about every four ticks (a time share between floor and role, not a fault but a
volume of events). (4) The retry costs a little when the floor is good: 16 299 against 9 603
unfed household-ticks at `f` = 3, because a household tests a dead market. (5) A variant of
the exit that also leaves when the last bid was filled and the money buys the exit cover
(refused, 5.3.2) multiplies the unfed household-ticks by four and the mode changes by five.
(6) An `ExitTicks` above what purchases can build (cover 4 against the stock a household
holds) leaves households on the floor for good (1 069 stuck): 5.3.3.

What the model leaves out, so that none of the above is read as a prediction: it has no income
feedback (a consumer's wage does not depend on the price), the producers are not consumers
and do not withdraw labour to the floor (11.5), money does not circulate, the floor has no
duration and no commons, beliefs are replaced by the posted price, and the cohort is
weight 1. It shows the structure of the rule, which is why these are the claims of the spec:
the floor feeds, it removes the pin of a shortage, it needs the retry when it cannot feed
fully, and it does not remove the pinned regimes.

### 11.3 The pinned staple and the register

S04 records which outcome a staple role reaches (`spec-stability.md` 8). With the module on
the upper outcome of STB-L14 (the ceiling, or the price set by the work floor) is replaced,
in the model, by a price that depends on what the floor covers: the lower pin if the floor
covers up to the upkeep, an interior band if it covers more. N-028 stays open until the
scenarios have run on the library. The module is D15 as specified, not a damper of L14: it
changes what the households that cannot afford the staple do, and not the elasticity of the
labour supply of those that can.

### 11.4 The closed economy

With upkeep as the only demand and no faucet or sink, household cash is conserved, so the
cash income net of inputs equals the money spent on upkeep, the value of the upkeep bought. The
cost of the tick is the value of all the upkeep eaten, bought or made, and the receipts
`K^own` are the value of what was made (up to the change in stock). The mean welfare ratio
`(bought + made) / (bought + made)` is 1 again: the floor creates goods and not a ratio above
1 (STB-L46, N-054). The glut that the target-income rule makes in that economy is bounded by
the halt, and an own-use recipe is exempt from the halt (5.5 item 7) because its stop is the exit
cover. A ratio above 1 still needs spending beyond upkeep (`spec-demand.md` 7), a faucet or the
outside.

### 11.5 Labour withdrawn to the floor

A household in the mode does not work its role. If the role's output is the good in short
supply, its withdrawal makes the shortage worse and can raise the price that put it on the
floor (SUB-A3, a reinforcing loop). The model does not have it. The dampers are the exit
(the household leaves when it holds stock), the retry (it works the role every `RetryJobs` jobs)
and the proportional ramp for cohorts, and the condition is that the share of a role's labour
in the mode is small against the share that works, or that the exit works. For a staple role
that produces its own upkeep the loop is weak: such a household is fed by its role and its
cover is high.

### 11.6 What the module is not

It is not an anchor of the price in the interior, not a damper of the target-income loop, not
a source that cannot run out (it draws on the commons and a season), not a livelihood (the
bound of 4.2) and not relief. It lowers the pressure that reaches the ladder, the pool and
migration, and it feeds a household that the market does not.

## 12. Parameters

All are tunables (architecture.md 9.5), fields of `Config.Modules.Subsistence`, each with a
default, a bound, an owner and a doc line, and a row in the parameter-owner table (handoff b).
**Every default is a placeholder** (N-015, N-027); none comes from a source.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `Recipes` | the default own-use recipes of every role (3.2) | none | at most `MaxOwnUseRecipes`; each a known recipe meeting 3.4 |
| Module | `Roles` | role rows that replace the default list (3.2.1) | none | one row per role, at most `MaxRoles`; each list at most `MaxOwnUseRecipes` (an empty list is legal) |
| Role row | `Role`, `Recipes` | the key and the list of a row | required, none | an existing role; known recipes |
| Module | `EnterTicks` | the mode starts when secure ticks fall below this (5.3) | 2 | 1 to `MaxOwnUseTicks` |
| Module | `ExitTicks` | the mode ends when stock cover reaches this (5.3) | 3 | `EnterTicks + 1` to `MaxOwnUseTicks` |
| Module | `RetryJobs` | consecutive own-use jobs before a role job is tried; 0 never (5.5) | 10 | 0 to `MaxRetryJobs` |

Why these defaults, and what they are not. `EnterTicks` 2 and `ExitTicks` 3 are tied to the
default `CoverTicks` of 3 (`spec-agents.md` 12): after eating, a household that has bought up
to its target holds 3 ticks of upkeep, so 3 is the highest exit that purchases alone can reach
(5.3.3), and 2 leaves a band of one tick below the exit. They
are not derived from a source and they are not claimed to be stable. `RetryJobs` 10 is from
the model of 17.1, where it cost little for a floor that feeds and rescued one that does not;
a household with 1-tick jobs tests the market every ten ticks, one with 30-tick jobs every
three hundred. A pack with jobs of different lengths may want a different value. The module
has no default list: on with no recipe it does nothing (the warning `subsistence_empty`).

12.1. **No tunable of another spec is restated.** `CoverTicks`, `ChangePrivation`,
`RemovePrivation`, `MinWorkShare` and the others are read where they are owned.

## 13. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9); errors name the path, for
example `modules.subsistence.roles.cutter.recipes[0]`.

### 13.1 Bounds

Provisional, in `bounds.go` (`spec-config.md` 4.1), placeholders until the slice runs:

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxOwnUseRecipes` | 8 | entries in one list; the default `Work` scans the list of a free household once a tick, so the work per household is bounded by it |
| `MaxOwnUseTicks` | 1 000 | `EnterTicks`, `ExitTicks` and the saturation of `Secure`; the same bound as `CoverTicks` (`spec-agents.md` 12), so a cover that purchases can build is inside it |
| `MaxRetryJobs` | 1 000 | `RetryJobs`; the count in an `int64` agent field |

`MaxRoles` and the money bounds are `spec-config.md` 4.1. The products of 5.2 add no bound:
`Cost` is a sum of `U_c * Price_c` over at most `MaxUpkeepItems` goods, each below
`MaxQuantity * MaxPrice`, which `ErrOverflow` of `spec-agents.md` 13 already keeps inside
`MaxMoney` at the largest weight.

### 13.2 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | `EnterTicks`, `ExitTicks` or `RetryJobs` outside the bounds of 12; a list longer than `MaxOwnUseRecipes` |
| `ErrOwnUseBand` | `ExitTicks` not above `EnterTicks` (5.3.1); **new** |
| `ErrUnknownRecipe` | a list that names an undefined recipe |
| `ErrUnknownRole` | a row that names an undefined role |
| `ErrDuplicateID` | a recipe twice in one list; two rows for one role |
| `ErrOwnUseNotSource` | an own-use recipe with `Inputs` or `Requires` (3.4.1); **new** |
| `ErrOwnUseNotSited` | an own-use recipe without `Sited` (3.4.2); **new** |
| `ErrOwnUseOverlap` | a recipe on both the effective own-use list and `Role.Recipes` of one role (3.4.3); **new** |
| `ErrModuleDependency` | (`spec-production.md` 10.2, unchanged) an own-use recipe with `Draws` while the resources module is off |

The three new sentinels need a root mapping (N-075).

### 13.3 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a path.
`FloorBP` in the table is the figure of 4.2 at the mean factor of the recipe in the region
(`spec-viability.md` 3.1: the mean over the starts of a year, with the abundance at the plan
share of a drawn endowment). The one that needs the solver is marked.

| Warning | Condition |
|---------|-----------|
| `subsistence_empty` | the module is on and no role has a non-empty effective own-use list |
| `own_use_no_upkeep` | a role has a non-empty list and an empty covered set (3.3.2): none of its own-use recipes makes an upkeep good |
| `own_use_beats_market` | `FloorBP` of an own-use recipe in a region is at least the `SurplusBP` the solver reports for that region (4.2). Size-gated like V3 (`spec-viability.md` 4.6); never an error |
| `own_use_partial` | `FloorBP` below 10 000 in some region: the floor feeds a household only in part (4.5). Informational |
| `own_use_trap` | `RetryJobs` is 0 and `FloorBP` is below 10 000 in some region: a household on the floor cannot build its cover and has no way back to the market (11.2) |
| `own_use_exit_unreachable` | `ExitTicks` above the `CoverTicks` of `spec-agents.md` 12: purchases cannot build the exit cover (5.3.3) |
| `own_use_slower_than_ladder` | an own-use recipe with `Duration - 1` at or above `ChangePrivation` (8.2) |
| `own_use_not_available` | a role has a group in a region where none of its own-use recipes can run: a site factor of 0 in every season, or a draw above the effective capacity of its endowment (`spec-resources.md` 13.3, `draw_impossible`) |

## 14. Observability

### 14.1 Events

One new event, plain data with no map and no pointer, emitted after the transition it
describes (observability-and-logging.md 4.1, 4.4):

| Event | Emitted when | Fields (besides `Tick`) |
|-------|--------------|-------------------------|
| `OwnUseChanged` | once per region and role per tick in phase 2, after the starts, in ascending `RegionID` then role order, **if any of `Entered`, `Left`, `Started`, `Blocked`, `Retried` or `Output` is non-zero**; never otherwise | `Region`, `Role`, `Households` (the sum of the weights of the role in the region), `OnOwnUse` (households with the flag at the end of the flag step), `Entered` and `Left` (households whose flag changed in the tick), `Started` (executions of own-use recipes started), `Blocked` (households in the mode that were free and had no own-use recipe they could run), `Retried` (households for which the retry chose a role recipe over a runnable own-use recipe), `Output` (a slice of (`Commodity`, `Units`) in commodity order: the units created at the completion of own-use jobs in the tick) |

`OwnUseChanged` is an aggregate bounded by regions times roles
(observability-and-logging.md 4.3, the paragraph of aggregates). It is off when the module is
off. A change of `OnOwnUse` always carries `Entered` or `Left`, so the level is known at every
tick from the last event (the adapter holds the last value, as for `economysim.endowment.stock`).
No event is per agent: the mode of a household is a state, not a fact worth an event of its
own, and a role with many households flipping every few ticks (11.2) would make one per agent
per few ticks. The units created are also in `BoundaryFlow` under `recipe_output`; the event is
the only place where those of an own-use job are told from the rest.

### 14.2 Logs

All via the injected logger, constant messages, attributes in `snake_case`.
`Debug` (guarded by `Enabled`, per agent): "own use entered" (`agent`, `role`, `reason`:
`unfed` when `Hunger` decided it, `short_cover` otherwise), "own use left" (`agent`, `role`,
`reason`: `cover`), "own use retry" (`agent`, `role`), "own use blocked" (`agent`, `role`,
`reason`: the production reason, `no_site`, `zero_yield` or `no_endowment`). `Warn`, bounded
per region and role (observability-and-logging.md 3.2.2):

- "role on own use": more than half of a role's households have been in the mode for ten
  consecutive ticks, at the tenth tick and every tenth after (`region`, `role`, `households`,
  `own_use`). Actionable: the role's market has failed in that region, or the floor is more
  generous than the pack means (`own_use_beats_market`).
- "own use blocked": at least one household in the mode has been blocked for ten consecutive
  ticks, at the tenth and every tenth after (`region`, `role`, `households`, `reason`, the
  commonest production reason).

No `Info` and no `Error`. One new log key, `own_use` (`int64`: households in the mode), to be
added to the table of observability-and-logging.md 3.3.2; the others (`region`, `role`,
`households`, `reason`, `agent`) exist.

### 14.3 Metrics

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.household.own_use` | Gauge | `{household}` | `economysim.role`, `economysim.region` | `OwnUseChanged` (`OnOwnUse`); the adapter keeps the last value for a tick with no event |
| `economysim.own_use.units` | Counter | `{unit}` | `economysim.commodity` | `OwnUseChanged` (`Output`, per line) |

Series added for the five-region proposal: at most regions times roles for the gauge (5 regions
and 8 roles are 40), and at most the commodities that an own-use recipe makes for the counter.
Both attributes are bounded by the configuration (observability-and-logging.md 5.4.1). A new
metric is derivable from the event, as 5.3.1 requires.

### 14.4 Snapshot

Per region and role: the households in the mode and the households blocked in the last tick;
per agent, in a query (INT): `OwnUse`, `OwnUseJobs`, `Cover` and `Secure`. The names are OBS's.
Bounded by the configuration for the aggregates.

### 14.5 Cost with no observer

The flag step is a loop over eligible agents with a bounded inner loop over the covered set; it
allocates nothing (NFR-SUB-001). The event is constructed only when an observer was supplied
(observability-and-logging.md 4.5). Observation never changes results: the module takes no
draw (FR-SUB-035).

### 14.6 The `Observability` entry (observability-and-logging.md 9)

1. *New failure path or rejected input:* configuration errors are returned (`ErrOwnUseBand`,
   `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap`, and the existing ones); a
   household with nothing it can run is survived and counted in `Blocked`, with a bounded
   `Warn`. 2. *New state transition:* entering and leaving the mode, a retry, a blocked tick, an
   output: the event above, no map, no pointer, integers. 3. *New decision point in a
   strategy:* the choice of recipe in the mode; a `Debug` line with the reason. 4. *New loop:*
   the flag step is per agent per tick, guarded by `Enabled` for any log, and the only `Warn` is
   bounded by regions times roles; event construction allocates nothing with the no-op observer.
   5. *Changed event, metric or attribute:* none; one event, one log key and two metrics are
   added. 6. *New metric attribute:* none (`role`, `region`, `commodity` exist and are bounded).
   7. *`Error` or `Info`:* neither. 8. *Error logged and returned:* none. 9. *Tests:* a recording
   observer for the event and its order, a capturing handler for the two `Warn`s at ticks 10 and
   20, non-interference, the no-op allocation test (FR-SUB-040 to 045, NFR-SUB-001).

## 15. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). All plain data in a fixed order, no maps:

- for each agent, in agent-ID order: `OwnUse` (a bool) and `OwnUseJobs` (an `int64`). Hosted in
  the agent (`spec-agents.md` 3.1, 15), defined here;
- for each (region, role), in region then role order: the two runs of consecutive ticks of the
  two `Warn`s of 14.2 (diagnostic: they affect no result and are saved so that a resumed run
  logs as the original did);
- nothing else. The ring of the welfare record gains two integers per tick (7.3), which the
  welfare spec lists with its own state.

**Not saved:** the effective own-use lists and the covered sets (derived from the
configuration), `Cover`, `Cost`, `Afford` and `Secure` (derived from the saved state and the
configuration each tick), the ramp, the candidate recipes, the event. The configuration hash
covers the module, its rows and its three tunables. A restored world evaluates the flag,
chooses and reports as an uninterrupted one (FR-SUB-044).

## 16. Worked examples

Two settings of the author's invention, flagged as examples (N-017). The numbers are
placeholders; the arithmetic is checked in `models/sub/worked.py` (17.2). Neither claims to
describe a real land or habitat.

### 16.1 Example A: a lowland with a common (an example)

One tick is a day. Upkeep of a household: `grain` 3 and `fuel` 1. Reference prices: both 100.
Two market recipes, `farm_grain` (no inputs, 8 `grain`, duration 1) and `cut_fuel` (no inputs,
3 `fuel`, duration 1); the roles `farmer` and `cutter` list one each. The own-use recipe, the
common it draws on, its yield in the one region that has it and the module (excerpt, not a
complete valid pack):

```toml
[recipe.glean]                      # an own-use recipe: a sited source with no inputs
duration = 1
sited = true
outputs = [ { commodity = "grain", quantity = 5 } ]
draws = [ { resource = "common", quantity = 1 } ]

[region.lowland.yield.glean]
all = 7500                          # a poor yield: 3.75 grain a tick (5 * 7 500 / 10 000)
                                    # highland has no row, so glean cannot run there

[resources.resource.common]
shape = "refill"
rate_bp = 10000
[resources.endowment.lowland.common]
capacity = 4000

[modules.subsistence]
recipes = ["glean"]                 # every role's default
enter_ticks = 2
exit_ticks = 3
retry_jobs = 10
[modules.subsistence.role.farmer]
recipes = []                        # a farmer makes its own grain: no own-use recipe needed
```

**The covered set and the bound.** For `cutter`, `F` is `{grain}` (glean makes grain, an upkeep
good; fuel is not made by it and the cutter makes its own). `Cov(glean, grain) = 5 * 7 500 /
(1 * 3) = 12 500` basis points: one tick of gleaning makes 1.25 ticks of grain. The market:
`farm_grain` makes 8 grain, `cut_fuel` 3 fuel; the labour that makes the basket is `3/8 + 1/3 =
17/24` of a tick per household-tick, so `lambda* = 24/17` and `SurplusBP = 14 117`. The bound
holds, `12 500 < 14 117` (the floor covers 89% of what the market chain does), and the warning
`own_use_beats_market` is silent. The labour share to subsist is `ceil(10^8 / 14 117) = 7 084` basis points in the market and `ceil(10^8 /
12 500) = 8 000` on the floor.

**A household in a failed fuel market (checked in `worked.out`).** A `cutter` household cannot
sell fuel and earns nothing; it holds 7 grain and 150 money, the grain price is 100, so
`Cost = 3 * 100 = 300`. Its residue starts at 4 000.

| Tick | Cover | Afford | Secure | Mode | Made | `K^own` | Stock after eating |
|------|-------|--------|--------|------|------|---------|--------------------|
| 1 | 2 | 0 | 2 | out (2 is not below `EnterTicks` 2) | 0 | 0 | 4 |
| 2 | 1 | 0 | 1 | **enters** (1 < 2) | 4 | 400 | 5 |
| 3 | 1 | 0 | 1 | in | 3 | 300 | 5 |
| 4 | 1 | 0 | 1 | in | 4 | 400 | 6 |
| 5 to 8 | 2 | 0 | 2 | in | 4, 4, 3, 4 | 400, 400, 300, 400 | 7, 8, 8, 9 |
| 9 | 3 | 0 | 3 | **leaves** (cover 3 reaches `ExitTicks`) | 0 | 0 | 6 |

`Made` is the yield with its carried remainder: `raw = 5 * 7 500 + residue` each tick (37 500
plus 4 000 gives 4 and a residue of 1 500; then 39 000 gives 3 and 9 000; and so on), 3.75 a
tick on average. `Afford` is 0 throughout: with 150 money the cutter buys 0 ticks of 300. It
bids for nothing, because its reserve money is `3 * 400 = 1 200`, above its 150 (the saving
rule, `spec-agents.md` 7.4.4). The cutter works the floor seven ticks, feeds itself at
`Hunger` 0 throughout, and goes back to cutting; `OwnUseJobs` is 7, below `RetryJobs`. Its `K^own` is
100 a unit made (400, 300 and 400 in ticks 2 to 4) against a cost of 400 a tick. A
`farmer` household never takes the mode: its row is empty and its own output covers it.

**A cohort.** A group of 40 `cutter` households of one agent with cover 1: `ramp = ceil(40 * (3
- 1) / 3) = 27` executions, with cover 0 all 40, with cover 2 `ceil(40 / 3) = 14`. Its 27
requests draw 27 units of `common`; a stock of 4 000 grants them. The other 13 households rest
during the job (5.5).

### 16.2 Example B: an orbital habitat (an example)

One tick is a shift. Upkeep: `ration` 4 a tick, reference price 12. The market recipe
`synthesise` makes 18 ration in 2 ticks, with no inputs (a placeholder, for the arithmetic):
`lambda* = 18 / (2 * 4) = 9/4`, `SurplusBP = 22 500`. The own-use recipe tends a tray:

```toml
[recipe.tend_tray]
duration = 3
sited = true
outputs = [ { commodity = "ration", quantity = 15 } ]
draws = [ { resource = "tray_volume", quantity = 2 } ]

[region.ring_two.yield.tend_tray]
all = 8000                          # 15 * 8 000 / 10 000 = 12 ration an execution, 4 a tick
[resources.resource.tray_volume]
shape = "refill"
rate_bp = 20000
[resources.endowment.ring_two.tray_volume]
capacity = 80

[modules.subsistence]
recipes = ["tend_tray"]
```

`Cov(tend_tray, ration) = 15 * 8 000 / (3 * 4) = 10 000`: exactly the upkeep, so
`own_use_partial` is silent and `own_use_trap` cannot fire, against the bound `10 000 < 22 500`.
The ladder: `Duration - 1 = 2` is below `ChangePrivation` 3, so `own_use_slower_than_ladder` is
silent; a household entering with cover 1 is unfed for `3 - 1 - 1 = 1` tick before the first
ration (8.2), and with cover 0 for 2.

**A cohort of 40 households in the mode.** `U = 160` ration a tick, stock 200, money 300, price
12: cover `floor(200 / 160) = 1`, `Cost = 160 * 12 = 1 920`, afford `floor(300 / 1 920) = 0`,
secure 1, below 2: it enters. The ramp is `ceil(40 * 2 / 3) = 27` executions; the draws are
`2 * 27 = 54` units of `tray_volume`, granted if the tray holds 54 or more (a stock of 40 grants
`20` executions). After the three ticks of the job it makes `27 * 15 * 8 000 / 10 000 = 324`
ration, valued at the completion tick's price of 12 as `K^own` = 3 888. The ramp puts only 27 of the
40 households on the floor, so the cohort makes 108 ration a tick against the 160 it eats: its
stock falls, its cover falls, the ramp rises toward 40, and the floor covers the cohort fully
only as the cover reaches 0. A cohort is a weaker floor than the same households at weight 1,
which is the cost of the one-job-per-agent model (5.5 item 4).

## 17. What was run

Two small Docker Python scripts in `models/sub/` (`docker run --rm -e PYTHONDONTWRITEBYTECODE=1
-v "$PWD/doc/specs/models/sub":/w -w /w python:3.12-slim python staple.py`). They are
**models of the rules of this spec, not of the library** (there is no library); they use
integers throughout, with a random source only for the price gate and the wages, and they use
placeholder parameters (N-015). They show the structure of a rule and its sensitivity; they do
not validate a default and they do not support a claim that any default is stable (N-027).

### 17.1 `staple.py`: the pinned staple market with and without the mode

The model, its cases and its results are in 11.2; `staple.out` holds the full output. It is
the labour-loop model of `spec-stability.md` 6.1 with the consumer side added (1 370 consumers,
three units of upkeep a tick, wages uniform over 3 600 to 7 200, the price rule with a gate of
one half and steps of 2%, bounds of 100 and 10 000). The cases are the floor off and on at
`f` = 2, 3 and 4 units a tick, with the retry at 10 jobs and without it, with the work floor
of `spec-welfare-and-wages.md` 5.3 and without it, an outage of the market for 100 ticks, an
exit also by purchase (refused) and an exit cover above what purchases build (the hazard of
5.3.3). Each case is run over five seeds and the table reports the median-price seed. What it
leaves out is listed at the end of 11.2.

### 17.2 `worked.py`: the arithmetic of the examples

The integer rules of 4.1, 5.2, 5.3 and 5.5 applied to the examples of 16: the hand-solved
`lambda*` of both chains, `Cov` for each recipe, the nine ticks of the household of 16.1 with
its carried remainder, the ramp for a cohort of 40 at covers 0, 1 and 2, the draws of the
cohort of 16.2, and the unfed ticks before a first output. `worked.out` holds the tables
quoted in 16.

## 18. Entries for the stability register

Written against the checklist of `spec-stability.md` 9 (the entry format is its section 3).
The IDs are provisional (`SUB-A1` for an entry, `SUB-D1` for a damper, `SUB-S1` for a scenario);
the supervisor assigns the final numbers (N-103). Nothing here has been run on the library; the
status says what was. No default is claimed stable (N-027).

**Merges into existing entries.**

| Entry | Merged into | What is added there |
|-------|-------------|---------------------|
| SUB-A1 the floor under the budget loop | L11, and with it L39 and L63 | D15 is specified: the mode, SUB-D1 to SUB-D5; the condition is a runnable own-use recipe |
| SUB-A7 ratio effects of the receipts | L18, L40, L46 | the receipts are valued at the prices of `cost_t` (L18); the floor raises the ratio that relief and MIG read (L40, with L75 for births); the closed-economy identity is unchanged (L46) |

**SUB-A1 The floor under the budget loop (L11).** *Kind:* loop (L11's). *Loop:* no cash (+) no
bids (+) no inputs (+) no production (+) no income (+) no cash; R. The own-use recipe is a path
from no stock to stock that needs no cash and no input (3.4.1), so the household whose
livelihood has failed still produces. *Delay:* the own-use job's duration. *Owner:* SUB 3, 5;
CLR 13 and STB-L11, L39 name D15. *Damper:* SUB-D1 to D5; D15 itself; D11 beside it.
*Undamped:* a household with nothing it can run (a zero yield in the season, no site, no
endowment: `Blocked`); a floor that feeds in part (4.5). *Condition:* a covered set that is not
empty and an own-use recipe that can run where the household lives. *Test:* SUB-S1, SUB-S2.
*Status:* model for the unfed counts (11.2: 97% to 100% fewer unfed household-ticks when the
floor covers the upkeep).

**SUB-A2 The floor makes a staple's demand price-responsive.** *Kind:* loop. *Loop:* the price
of the covered good (+) households whose secure ticks fall below `EnterTicks` (+) their own-use
output and stock (-) their bids for the good (+) the price; one negative link, so B. The flag
is the price-responsive flow that N-043 asked for; it is not a rationing of upkeep (11.1).
*Delay:* the own-use job's duration, plus the time to build the exit cover; the neighbours are
the welfare window `T` (30) and `ExitTicks` (3), 9.6 below. *Owner:* SUB 5, 11.1; touches L14,
L51 (it is the price-responsive flow that L51 lacks, for the households that cannot afford the
staple), L48, L15 (chatter). *Damper:* SUB-D1 (the band), SUB-D3 (the retry), SUB-D5 (the
ramp), D04 (the price bounds). *Undamped:* an overshoot: in the model the price falls through
the repeller to the lower pin when the floor covers only the upkeep (`f` = 3) and stays in an
interior band when it covers more (`f` = 4); with a long own-use job the cycle can grow.
*Condition:* the own-use job is short against the welfare window; the band is not 0; the floor
covers more than the upkeep for an interior price. *Test:* SUB-S1. *Status:* model (11.2), a
stylised one with no income feedback.

**SUB-A3 Labour withdrawn to the floor.** *Kind:* loop. *Loop:* households in the mode (+) the
labour they supply to the market (-) its supply of the role's good (+) the price (+) households
short of secure ticks (+) households in the mode. Links in order: mode to labour supplied (-),
labour to supply (+), supply to price (-), price to short households (+), short households to
mode (+): two negative links, so R. *Delay:* the job's duration and the welfare window.
*Owner:* SUB 11.5; touches L14, L36. *Damper:* SUB-D1 (a household leaves when it holds
stock), SUB-D3 (it works its role every `RetryJobs` jobs), SUB-D5. *Undamped:* a role that is
most of the supply of a good that its own households cannot afford. *Condition:* the share of
a role's labour in the mode is small, or the exit works. *Test:* SUB-S1 with consumers who are
also producers (the model has none). *Status:* hand.

**SUB-A4 The floor on the commons.** *Kind:* loop. *Loop:* unfed households (+) households in
the mode (+) draws on the endowment (-) its stock (+) its abundance (+) the effective yield of
the own-use recipe (-) unfed households. Draws to stock (-) and yield to unfed (-): two
negative links, so R. A third path to the open-access draw-down of L90, beside recipe choice and
entry; it closes on L87. *Delay:* the regeneration timescale of the endowment and the welfare
window. *Owner:* SUB 9.4; RES 10. *Damper:* D53 (the ration caps the draw at the stock), D54
(the soft landing), D56 (a refuge), SUB-D1 (a household draws only to its exit cover). *Undamped:*
a commons whose flow is smaller than what the failing households draw: tenure is LND's.
*Condition:* the endowment regenerates at least what the households in the mode draw at the
population it can feed (`spec-resources.md` 8; `resource_overshoot`, `resource_unstable_plan`).
*Test:* SUB-S4. *Status:* hand (the structure of the models of `spec-resources.md` 17.3
applies).

**SUB-A5 The floor can be too slow for the ladder.** *Kind:* hazard. *Loop:* none. An own-use
recipe whose `Duration - 1` reaches `ChangePrivation` lets rung 1 change the role of a
household and abandon the job a tick before it completes (8.2). *Owner:* SUB 8.2; SPN 5.
*Damper:* the warning `own_use_slower_than_ladder`; `EnterTicks` (a larger value enters with
more cover left); a spawner that skips a household with a running own-use job (open
question 4). *Undamped:* the case the warning flags and the pack keeps. *Condition:* `Duration
- 1 < ChangePrivation`. *Test:* SUB-S3. *Status:* hand.

**SUB-A6 A household held on the floor.** *Kind:* hazard. *Loop:* in the mode (+) no role
income (-) cash (+) afford (+) secure ticks (-) in the mode: the negative links are mode to
income and secure to mode, so R. A household on a floor that does not cover its upkeep cannot
build the cover that lets it leave, and earns nothing meanwhile. *Delay:* the retry period.
*Owner:* SUB 5.3, 5.5 item 5. *Damper:* SUB-D3 (the retry, which is a market test every
`RetryJobs` jobs), SUB-D1 (the exit by stock, reachable if `ExitTicks <= CoverTicks`).
*Undamped:* `RetryJobs` 0 with `Cov` below 10 000; a retry into a market that is dead.
*Condition:* `RetryJobs > 0`, or a floor that covers the upkeep, and `ExitTicks <= CoverTicks`.
*Test:* SUB-S1: the cases `f` = 2 (167 239 unfed household-ticks against 82 665 with no floor,
41 232 with the retry) and the exit above the cover (1 069 households stuck). *Status:* model.

**SUB-A7 Ratio effects of the receipts (merged).** *Kind:* bias. In L18: the receipts are
valued at the prices of `cost_t`, so a household that makes its upkeep reads 10 000 at any
price, and the mode forces the work share to 10 000, so the loop of L18 does not act on a
household in the mode. In L40 and L75: the floor raises the ratio that relief and MIG read
and a floor-fed household is exposed to the birth and death schedules at that ratio. In L46:
the mean ratio of a closed upkeep-only economy is 1 with the floor, as without (11.4). The sale
of a surplus is counted twice (7.5). *Test:* SUB-S2.

### 18.1 Dampers this spec adds or specifies

| ID | Damper | Mechanism | Home | Status |
|----|--------|-----------|------|--------|
| SUB-D1 | Hysteresis by stock | the mode ends only when the stock covers `ExitTicks`, above `EnterTicks`, never by money. Cost: a household may stay on the floor after the market has recovered, until it holds stock or the retry brings it back | SUB 5.3 | specified (counted by `OwnUseChanged.Left` and `Entered`) |
| SUB-D2 | Market first | the mode starts only when the household's own secure ticks are short; a household fed by the market never takes it. Cost: one unfed tick when the market is empty at a normal price | SUB 4.4, 5.2 | specified |
| SUB-D3 | The market retry | after `RetryJobs` own-use jobs the next free decision is a role job, if one can run. Cost: a role job started into a failed market | SUB 5.5 item 5 | specified (counted by `OwnUseChanged.Retried`) |
| SUB-D4 | The productivity bound and the source-only rule | `FloorBP < SurplusBP` (a warning) and no inputs or tools on an own-use recipe. Cost: a pack with a generous floor gets a warning, not an error | SUB 3.4.1, 4.2 | specified (a warning; the rule is an error) |
| SUB-D5 | Proportional ramp for cohorts | a cohort puts on the floor the share of its households that is short of cover, so labour moves in steps of `1/w` (a form of D28). Cost: the rest of the cohort rests during the job | SUB 5.5 item 4 | specified |

Each is a tunable or a rule constant with a home (checklist 9.7; `spec-stability.md` 5.1.2):
SUB-D1 is `EnterTicks` and `ExitTicks`, SUB-D3 is `RetryJobs`, SUB-D2 and SUB-D5 are rule
constants, SUB-D4 is a rule of validation. None reads a macro aggregate (5.1.3): every one reads
the household's own state.

### 18.2 Delays against neighbours (checklist 9.6)

The own-use job's duration against the welfare window `T` (the window is derived to hold twice
the longest job on the role's list, `spec-welfare-and-wages.md` 2.3.3; the own-use recipes
join that list by the edit of 7.6, so the lump of a long job stays inside it); against
`ChangePrivation` and `RemovePrivation` (8.2); the retry period (`RetryJobs` times the job's
duration) against the season length (a floor that yields only in a season); `ExitTicks`
against `CoverTicks` (5.3.3); the commons' regeneration timescale against the window and the
season (as `spec-resources.md` 19.3). None coincides at the placeholder values; each is a
warning or a stated condition.

### 18.3 Scenarios proposed

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| SUB-S1 | A staple role with a shortage injected (the pack of S04) and an outage of the staple market, with the module off and on, an own-use recipe of coverage 6 667, 10 000 and 13 333, the retry on and off, the work floor on and off (the model of 11.2 is the oracle of shape) | the unfed household-ticks fall when the floor covers the upkeep; the price is reported as the pinned shortage, the pinned glut or an interior band (F5, F6); a floor below the upkeep without the retry is no better than none | A1, A2, A3, A6, L14, L51 |
| SUB-S2 | A region that loses its income (`famine` of S14, or S27's variant) with the pool, relief and the ladder on, and the module off and on | households unfed, removals and the pool's balance are lower with the mode; the ratio of a household on the floor is near what the floor covers and not 0; the mean ratio of the closed upkeep-only pack is 1 | A1, A7, L39, L46, L63 |
| SUB-S3 | Own-use recipes of duration 1, 3 and 8 against `ChangePrivation` 3 | rung 1 abandons a running own-use job only when `Duration - 1` reaches `ChangePrivation`, and the warning fires for exactly those | A5 |
| SUB-S4 | A scarce `refill` commons that own-use recipes draw on, a thousand agents of weight 1 and twenty cohorts of weight 50, the module off and on | the ration holds; `Blocked` is non-zero when the stock is cut; cohorts and units get the same mean; the commons recovers or does not as `spec-resources.md` 8 predicts; the `Warn`s are bounded | A4, L87, L90 |
| SUB-S5 | The invariant run: the reference chain with an own-use overlay over a range of seeds with `WithAudit`, a checkpoint at the middle of a mode, and a relocation of a household in the mode | the audit holds; the recorded events are the same as the state; the resumed run is bit-identical; the relocated household's flag is dropped; no ledger reason beyond the registry | determinism, A1 |

## 19. Notes settled

| Note | Disposition | Where |
|------|-------------|-------|
| N-046 (register entries owed by SUB) | **applied** for SUB | section 18: SUB-A1 to SUB-A7, SUB-D1 to SUB-D5, SUB-S1 to SUB-S5 |
| N-104 (region-keyed tables use the one mechanism) | **applied** for SUB: no region-keyed table or field; the role-keyed rows follow the replace mode and the one-row-per-key rules; regional differences are the yield table and the endowment rows | 3.2.1, 3.5 |
| N-108 (what RES leaves to SUB) | **applied** for SUB: an own-use recipe draws through `Draws` like any recipe, open access, the ration, no priority, and the hooks for LND are stated | 9 |
| N-016 (the parameter registry) | updated: five rows (the module, its default list, the role rows, three tunables), each with a default or "required", a bound and an owner | 12, handoff b |
| N-017 (words and names) | updated: terms fixed in the handoff c; setting words only in the flagged examples of 16 | 16 |
| N-027 (no default macro constant is stable) | updated: every number is a placeholder; the model shows structure | 17 |
| N-103 (running several writers) | updated in the handoff (process slips included) | handoff l |
| N-028, N-043, N-052 (the labour loop and a price-responsive staple flow) | updated, left open: the mode is a price-responsive flow for the households that cannot afford a staple and is not a damper of the labour supply of those that can | 11, 18 |
| N-045, N-065 (silent dampers need counters) | updated: the mode's hysteresis, the retry, the blocked and the ramp are counted in `OwnUseChanged` | 14.1 |
| N-082, N-096 (the pool is shared) | updated: the floor lowers the draw on the pool; it adds none | 8.4 |
| N-090, N-097, N-110 (MIG) | updated: the seam of 5.8 | 5.8 |
| N-109 (closed forms need `r_full`) | updated: the ratio of a floor-fed household is the floor's | 7.4 |
| N-009 (phases) | updated: the flag step of phase 2 | 5.4 |
| N-012, N-019, N-020, N-075, N-092, N-094 | updated: events, code homes, state, sentinels, overlay, snapshot | handoff |
| N-107 (new faucets and sinks start from MON) | applied: SUB adds no faucet, no sink and no ledger reason | 6 |

The exact "Update" text of each is in the handoff.

## 20. Requirements

IDs are stable and unique across the module (testing.md 4.1). Each is one testable statement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-SUB-001 | The subsistence module MUST be on exactly when `Config.Modules.Subsistence` is non-nil; with it off no agent has a mode, no event of the module is emitted and the run is identical to a run without the module. |
| FR-SUB-002 | The effective own-use list of a role MUST be the list of its row if it has one (an empty list meaning none) and the module's `Recipes` otherwise, sorted by recipe ID, and a role with two rows or a recipe twice in a list MUST be rejected with `ErrDuplicateID`. |
| FR-SUB-003 | The module MUST add no ledger reason and no account kind: the output of an own-use job MUST be created by `recipe_output` into the executor's account, and with the module on the identities of `spec-ledger.md` 9.1 and the audit MUST hold. |
| FR-SUB-004 | Validation MUST reject, with `ErrOwnUseNotSource`, an own-use recipe that has `Inputs` or `Requires`. |
| FR-SUB-005 | Validation MUST reject, with `ErrOwnUseNotSited`, an own-use recipe that is not `Sited`. |
| FR-SUB-006 | Validation MUST reject, with `ErrOwnUseOverlap`, a recipe that is on both the effective own-use list and `Role.Recipes` of one role. |
| FR-SUB-007 | Validation MUST reject, with `ErrOwnUseBand`, an `ExitTicks` not above `EnterTicks`, and with `ErrOutOfBounds` a tunable outside the bounds of 12 or a list longer than `MaxOwnUseRecipes`. |
| FR-SUB-008 | Validation MUST reject an unknown recipe in a list with `ErrUnknownRecipe` and an unknown role in a row with `ErrUnknownRole`. |
| FR-SUB-009 | The module MUST have no region-keyed table and no per-region field; whether a region offers subsistence MUST be decided by the yield table and the endowment rows alone. |
| FR-SUB-010 | The covered set of a role MUST be the upkeep commodities that are a non-discard output of an own-use recipe of the role, and an agent MUST be evaluated for the mode only if it is not a pseudo-agent, not a trainee, not away, has a non-empty covered set and a positive effective upkeep; any other agent MUST have `OwnUse` false and `OwnUseJobs` 0. |
| FR-SUB-011 | `Cover`, `Cost`, `Afford` and `Secure` MUST be computed as in 5.2 with checked integer arithmetic, `Secure` saturating at `MaxOwnUseTicks`. |
| FR-SUB-012 | `Afford` MUST be 0 for an agent with `Hunger` above 0. |
| FR-SUB-013 | An agent not in the mode MUST enter it when its `Secure` is below `EnterTicks`, and `OwnUseJobs` MUST be 0 on entry. |
| FR-SUB-014 | An agent in the mode MUST leave it when its `Cover` is at least `ExitTicks` and in no other case, whatever its money. |
| FR-SUB-015 | The flag MUST depend on nothing but the agent's own stock, money, `Hunger`, upkeep, counter and the prices of its own region; a test that changes every other agent, the regional wage and every other region MUST find it unchanged. |
| FR-SUB-016 | A cohort of weight `w` and `w` agents of weight 1 in the same circumstances MUST have the same `Cover`, `Afford`, `Secure` and flag. |
| FR-SUB-017 | The flag step MUST run in phase 2 after the continuations and completions and the snapshot of the endowments and before the first `Work`, in agent-ID order, and its result MUST NOT depend on the order of the agents. |
| FR-SUB-018 | The view of an agent MUST carry `OwnUse`, `OwnUseJobs`, `Cover` and `Secure`, and mark each own-use recipe in the per-recipe group. |
| FR-SUB-019 | The default `Work` MUST use a work share of 10 000 for an agent in the mode. |
| FR-SUB-020 | The default `Work` in the mode MUST request the own-use recipe with the greatest sum of `Cov` over the covered commodities whose cover is below `ExitTicks`, compared as exact fractions, ties to the lower recipe ID, and MUST NOT request a recipe whose sum is 0. |
| FR-SUB-021 | The executions of the request of FR-SUB-020 MUST be `min(w, MaxExecutions, max(1, ceil(w * (ExitTicks - Cover) / ExitTicks)))`. |
| FR-SUB-022 | `OwnUseJobs` MUST increase by 1 when an own-use job starts and MUST reset to 0 when a job of another recipe starts or the mode changes; when `RetryJobs` is above 0, `OwnUseJobs` has reached it and a role recipe is eligible, the default `Work` MUST request the role recipe. |
| FR-SUB-023 | When no own-use recipe has `MaxExecutions` of 1 or more, the default `Work` MUST act as if the agent were not in the mode, the flag MUST stay, and the agent MUST be counted in `Blocked`. |
| FR-SUB-024 | The production halt of `spec-agents.md` 7.1.5 MUST NOT apply to an own-use recipe. |
| FR-SUB-025 | A role with an empty `Recipes` list and a non-empty effective own-use list MUST request an own-use job in the mode. |
| FR-SUB-026 | Production MUST accept a request for a recipe on the role's effective own-use list, and MUST still refuse any other recipe not on `Role.Recipes` with `not_on_role` before any other check. |
| FR-SUB-027 | An own-use recipe MUST NOT set the cost anchor of any commodity. |
| FR-SUB-028 | The income of a tick MUST include `K^own_t`, the units created at the completion of own-use jobs in the tick valued at the prices used for `cost_t`, the effective reference price where a commodity has never traded. |
| FR-SUB-029 | The observed wage MUST exclude `K^own_t` from the income and the own-use household-ticks from the working household-ticks. |
| FR-SUB-030 | An own-use tick MUST count as a working tick for the work floor and the target-income rule. |
| FR-SUB-031 | A relocation MUST set `OwnUse` false and `OwnUseJobs` 0, and the next evaluation MUST be in the new region at the next phase 2. |
| FR-SUB-032 | A role change MUST reset `OwnUseJobs`, keep `OwnUse` until the next evaluation and abandon an own-use job with `role_changed`; a weight change MUST leave both unchanged; a removal MUST drop both. |
| FR-SUB-033 | The mode MUST NOT alter the computation of `Hunger` and `Privation` from `Unfed`. |
| FR-SUB-034 | An external agent MUST be evaluated like any other, and the directive of the `Directed` strategy MUST override the default `Work`. |
| FR-SUB-035 | The module MUST draw no number from the world's random source: the stream of a world is the same with the module on and off. |
| FR-SUB-036 | The request of an own-use job with `Draws` MUST enter the ration of `spec-resources.md` 5.3 as any other request, with no priority and no quota. |
| FR-SUB-037 | A recipe that is on an own-use list and on no role's `Recipes` MUST NOT be an activity of the reference programme and MUST NOT make a commodity producible for V1. |
| FR-SUB-038 | Validation MUST warn, and MUST NOT fail, with `subsistence_empty`, `own_use_no_upkeep`, `own_use_beats_market`, `own_use_partial`, `own_use_trap`, `own_use_exit_unreachable`, `own_use_slower_than_ladder` and `own_use_not_available` under the conditions of 13.3. |
| FR-SUB-039 | `Cov(r, c)` MUST be `q * E / (Duration * u_c)` in basis points with truncation and 128-bit intermediates, and `FloorBP` of a recipe in a region MUST be its maximum over the covered commodities at the mean factor of the region. |
| FR-SUB-040 | The event `OwnUseChanged` MUST be emitted once per region and role per tick in which any of `Entered`, `Left`, `Started`, `Blocked`, `Retried` or `Output` is non-zero, and never otherwise. |
| FR-SUB-041 | The events of a tick MUST be in ascending `RegionID` then role order, after the starts of phase 2, with no map and no pointer in any field and `Output` in commodity order. |
| FR-SUB-042 | The `Warn` "role on own use" MUST be emitted at the tenth consecutive tick on which more than half of a role's households in a region are in the mode and every tenth after, and the `Warn` "own use blocked" at the tenth consecutive tick on which a household in the mode is blocked and every tenth after, each at most once per region and role per tick. |
| FR-SUB-043 | The log key `own_use` and the metrics `economysim.household.own_use` and `economysim.own_use.units` MUST be defined as in 14.2 and 14.3, with attributes bounded by the configuration. |
| FR-SUB-044 | A checkpoint MUST save `OwnUse` and `OwnUseJobs` of every agent and the two diagnostic runs, and a world restored between ticks MUST evaluate, choose and report as an uninterrupted one. |
| FR-SUB-045 | A run with a logger and an observer attached MUST produce exactly the same results as one without. |
| FR-SUB-046 | The snapshot MUST report per region and role the households in the mode and the households blocked in the last tick, and per agent `OwnUse`, `OwnUseJobs`, `Cover` and `Secure`. |
| FR-SUB-047 | A pack with the module, its recipes, its yield rows and its endowment rows MUST run from configuration alone, with no Go specific to it (architecture.md 9.4). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SUB-001 | With the default logger and observer the flag step, the choice and the event construction MUST allocate nothing in the tick loop (`testing.AllocsPerRun`). |
| NFR-SUB-002 | `internal/ownuse` MUST use the standard library only, take and return primitive values (architecture.md 2.9), and use no floating point; every product MUST be checked. |
| NFR-SUB-003 | The work of the flag step per agent per tick MUST be bounded by the covered set (at most `MaxUpkeepItems` goods), and the module SHOULD have a benchmark with `b.ReportAllocs()`. |

## 21. Open questions

1. **The lean season.** A household in a long role job cannot take the mode (5.7), and the end
   of the stores before a harvest is when it needs it. Either a second job slot for light
   own-use work, which changes `spec-production.md` 5.1.2, or an interruption of a role job
   with an abandon reason that costs the inputs. Decide with PRO, with the first scenario that
   shows the case.
2. **The surplus.** The output of an own-use job is fungible stock, may be sold, and is then
   counted twice in the ratio (7.5). A per-lot provenance, a ban on selling, or counting the
   receipt at consumption instead of creation.
3. **Seed cycles.** An own-use recipe that needs seed is refused (3.4.1). A recipe that returns
   its own seed is a cycle of its own and is sound where the seed is held; allowing it needs a
   check that the net is positive and a rule for the first stock.
4. **Rung 1 and a running own-use job.** The spawner's view gains `OwnUse`; the default
   spawner does not use it (8.3). Whether a household with a job that will cover its need should
   be skipped by rung 1, and the cost to the occupation signal.
5. **A computed check of the lean season** in the style of V4: the longest gap in the year in
   which none of a role's own-use recipes can run, against the cover that a household can hold
   (`CoverTicks`).
6. **The retry.** In jobs, so that it is independent of durations. A retry in ticks or a retry
   only when the price of the covered good has fallen would test the market less often when it
   is dead and more when it has recovered. The latter needs a price memory (state).
7. **Cohorts.** One job per agent means the part of a cohort not on the floor rests (5.5 item
   4). Exact only at weight 1. A split of the cohort into two agents is `spec-scale.md` 8's.
8. **Relief to a household that feeds itself.** Whether relief should pay a household whose
   ratio is below the line only because its floor covers less than its upkeep (8.4).
9. **An own-use recipe that makes a want** (a non-upkeep good): out of scope; the covered set is
   upkeep only (3.3.2).
10. **The priority on the commons.** Open access gives the floor no priority (9.2). A right to
    glean is LND's entitlement; whether the default of LND is a priority for own-use requests.
11. **Measuring the duty cycle.** The share of a household's jobs that are own-use is a measure
    (IND), derivable from `OwnUseChanged.Started` and the recipe executions.
12. **A mode that changes the target cover.** Raising the household's `Keep` while in the mode,
    so that purchases can build the exit cover, would remove the unreachable exit of 5.3.3 and
    couple the module to `Trade`.

## 22. Where this lives in the code

Nothing is written. When it is, in the layout of architecture.md 2.6 and `AGENTS.md`:

- `subsistence.go` in the root package: `SubsistenceConfig`, `RoleOwnUse`, the validation of
  13, the effective lists and covered sets resolved at construction, the call of the flag step
  and the field `OwnUse` of the agent;
- `internal/ownuse/`: the flag step (`Secure`, the enter and leave rules), `Cov`, the ramp, over
  primitive slices (a covered good is a stock, an upkeep total and a price; architecture.md
  2.9), standard library only, with the checked 128-bit helper of `spec-currency.md` 7. The
  root converts inline at the call. The package does not know what a recipe, a role or a
  commodity is;
- the default strategy's `Work` (`strategy` is not needed: the default stays in the root
  package, architecture.md 2.6) gains the rule of 5.5;
- the new sentinels `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited` and
  `ErrOwnUseOverlap` and their mapping (N-075), the log key `own_use` (observability-and-logging.md
  3.3.2) and the glossary terms in `doc.go`;
- the Python models are archived in `doc/specs/models/sub/` (`staple.py`, `worked.py` and their
  outputs, with a README).

