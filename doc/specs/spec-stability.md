# Spec: The stability register

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-STB-NNN` and
`NFR-STB-NNN`, which tests cite (testing.md section 4). They are in section 14.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs nothing; it is cross-cutting and grows with every spec. It is a
**register**: it collects the feedback loops of the design, says how each is
damped and how a test would show it, and fixes the rule by which every later spec
adds its own.

**Nothing here claims that any default is stable.** The numeric analyses of
section 6 are runs of small stylised models of the rules of other specs, with
placeholder parameters (notes N-015, N-027). They show how a loop can behave and
which parameters it is sensitive to; they do not validate a default, and they are
not runs of the library, which does not exist. Every threshold in sections 7 and 8
is a placeholder for `spec-stylised-facts.md` to replace.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). Two of them
(feedback loop, damper) are already on the glossary list in `README.md`.

## 1. Purpose and scope

Stability is a requirement, not a hope (overview.md principle 7; roadmap.md
"Stability"). A simulation of this kind fails in a few characteristic ways: a price
runs to a bound, a market freezes, a role oscillates between full and empty, a
region collapses and cannot recover. Each failure comes from a **feedback loop**: a
chain of variables in which a change comes back to the variable that started it.
This spec:

- fixes the format of a register entry and lists every loop the written specs
  create, with its sign, its delay, its **damper** and the test that would show it
  (sections 3 and 4);
- catalogues the dampers, says where each lives, and fixes the rule for adding one
  (section 5);
- analyses the three most consequential loops numerically (section 6);
- defines the **failure regimes** a stability test looks for (section 7) and the
  scenarios the acceptance bands must cover (section 8);
- gives the checklist a spec author follows so that every new loop is registered
  (section 9) and lists the loops that no spec yet dampens (section 10).

Out of scope: the acceptance bands themselves and the failure thresholds
(`spec-stylised-facts.md`), the measures (`spec-indicators.md`), the reference
chain (`spec-reference-chain.md`), the steady-state solver and the warm start
(`spec-viability.md`), the experiment harness (`spec-experiments.md`). This spec
names what they must test; they decide the numbers.

## 2. Principles

2.1. **Every loop is named, damped and tested.** A feedback loop that is in no
register entry has no owner when it fails. A loop that has an entry but no damper
says so in the entry (field "Undamped") instead of being left out.

2.2. **A damper is a mechanism, and it exists.** An entry names the damper by the
spec section and the parameter or rule constant that implements it. The catalogue
(section 5) is the index. A damper that is only an intention is marked
*proposed* and counts as undamped until a spec adopts it.

2.3. **Dampers change dynamics; bounds only cap them.** Several loops in this
design are held only by a price bound or a validation bound. A bound stops a loop
from running away without removing it: the system then sits on the bound. The
register distinguishes the two, because a market pinned to a bound is a failure
regime (section 7), not a stable state.

2.4. **The register states conditions, not guarantees.** A loop is stable under a
stated condition (for example "demand more elastic than supply"). Whether the
condition holds in a given pack is data. The register records the condition, and
the acceptance scenarios test the packs that ship.

2.5. **Honest status.** Each entry says how it is known: derived by hand, run in a
stylised model (section 6), or untested. "Untested" is a legitimate status and the
majority at this stage.

2.6. **The register is documentation, not code.** The library carries no runtime
stability controller. Macro aggregates are measured and never fed back to agents
(roadmap.md "Macro is measured, never set"); an adaptive controller that read them
would break that rule. All stability comes from local rules and their parameters.

## 3. The entry format

Every entry has an ID, a name and these fields. A spec that contributes an entry
writes the fields it knows and leaves the others as "to be set"; the register
owner (this spec) completes it.

| Field | Content |
|-------|---------|
| ID | `STB-Lnn`, never reused. Dampers are `STB-Dnn`, failure regimes `STB-Fn`, scenarios `STB-Snn` |
| Kind | `loop` (a closed chain of causes), `amplifier` (a property that makes other loops worse, such as noise or lockstep), `bias` (a systematic offset with no loop), `constraint` (a hard rule that bounds but does not stabilise), `hazard` (a configuration or process trap) |
| Loop | The variables in order, the sign of each link (+ same direction, - opposite), and the polarity of the whole: **R** reinforcing (an even number of - links; it amplifies), **B** balancing (an odd number; it corrects) |
| Delay | Where the loop waits, in ticks or as a named parameter |
| Owner | The spec that creates the loop and fixes its mechanism |
| Damper | The mechanisms in the design that limit it, by `STB-Dnn` and spec section |
| Undamped | What remains: the part no mechanism covers |
| Condition | When the loop is stable, as a statement that can be checked |
| Test | The scenario (`STB-Snn`, section 8) that would show it |
| Status | `hand` (derived by hand), `model` (run in a model of section 6), `untested` |

A balancing loop with a delay can still oscillate, and a reinforcing loop is
harmless if its gain is below one or a bound meets it first. The polarity says
where to look, not that a loop is good or bad.

## 4. The register

Entries are grouped by area. Where a source spec already left a labelled entry,
the spec and section are given in the Owner field (`spec-clearing.md` section 13
is written "CLR 13"). Entries that two specs listed under different words are
merged and say so.

### 4.1 Price formation (CLR, BEL)

**STB-L01 Price, belief and limit loop.** *Kind:* loop. *Loop:* market price
(+) evidence for each agent's belief centre (+) limit price drawn from the centre
(+) trade price (midpoint of limits) (+) market price. All links positive, so the
loop looks reinforcing, but each agent moves its centre toward a target and never
past it, so dispersion between agents falls (balancing); the common level of all
centres has no restoring force from evidence alone (section 6.2). *Delay:* one
tick for the outcome, and a mean of two ticks for the gate (`BeliefAdjustProb`
50%). *Owner:* CLR 13, BEL 17. *Damper:* D01, D02, D03, D04, D05, D06.
*Undamped:* the level of the price is anchored only by the stock component, the
cost floor and the reference band (L02, L04). *Condition:* the stock component or
a flow response (L06) must supply a restoring force on the level. *Test:* S01.
*Status:* model.

**STB-L02 Evidence offset.** *Kind:* bias. *Loop:* none. In the stylised model of
section 6.2 the first draft of the evidence rule of `spec-beliefs.md` 5.4 settled the
price 11% below the price at which supply equals demand and left about 12% of the
buyers' need unmet, while the stock component alone settled within 0.3% of it. BEL
has revised the rule in round 3 (the target is compared to the order's limit, a move
that points the wrong way is dropped, and the step is `BeliefEvidenceShare` of `Δ`,
default 5 000); a fresh run of this register's own model with the revised rule gives
944 (940 to 949) against 1 000, about 5.6% low, with 5.9% of need unmet (6.2, run
2b), which agrees with BEL's own run. **The offset is reduced, not removed.** *Delay:* none.
*Owner:* BEL 5.4. *Damper:* the guard and the share of 5.4 (a partial damper: they
shrink the offset). *Undamped:* the remaining 5.7%, which BEL attributes to the
quantity rule (sellers ask all their stock, buyers bid the shortfall) and to the
pairing of highest bids with lowest asks (BEL open question 12). *Condition:* the
long-run price of a market with price-responsive flows is within a stated band of the
clearing price. *Test:* S01 with a clearing-residual check (FR-STB-018). *Status:*
model. *Note:* a finding about a stylised model, not a proven defect of the design.
N-042 stays open for BEL's question 12. AGT-A10
merged: the level of unsold stock that AGT's rules settle at (about
`OutputCoverTicks` ticks of output) is a property of the rules as written, which
include this offset, and is not a result.

**STB-L03 Stale price loop.** *Kind:* loop. *Loop:* published price (+) beliefs
anchor to it (+) limits stay uncrossed (+) nobody trades (+) price stays published.
Polarity R. *Delay:* `BeliefMaxAge` (3 ticks) before the price stops counting as
evidence. *Owner:* CLR 13, BEL 17. *Damper:* D05 (price age and the best unmatched
quotes); a bid and an ask close the gap by one `Δ` per adjustment (FR-BEL-034).
*Undamped:* a seller at its cost floor cannot cross a buyer below it (L09); the
real release is exit. *Condition:* the gap closes in at most `ceil(gap / Δ)`
adjustments unless a bound holds a side. *Test:* S02, S03. *Status:* hand.

**STB-L04 Stock-ratio integrator.** *Kind:* loop. *Loop:* seller stock (+) stock
ratio (+) downward push on the belief centre (-) price (-) buyers' purchases and
(+) producers' output are the flows into stock (the stock is the integral of the
flow imbalance). Because the signal is the integral, not the imbalance itself, the
loop is a balancing loop with a lag and can oscillate. *Delay:* the time to build
or run down a stock of one target (`BeliefStockLow` to `BeliefStockHigh` is a band
of 100% of the target). *Owner:* BEL 5.5, 17. *Damper:* D02 (bounded step), D03
(gate), D07 (the buffer itself), D17 (width bounds do not help; only the step
does). *Undamped:* with the stock signal alone, a sustained limit cycle of 6% to
29% of the price in the stylised model (section 6.1), whose amplitude scales with
`BeliefStep`. *Condition:* a proportional signal (the evidence rule) acts as well;
the combination was run in 6.2 (runs 2 and 2b) and is stable there, with a price
noise of 3.6% to 4.8%; BEL keeps both signals (5.5, N-044). *Test:* S01, S04.
*Status:* model.

**STB-L05 Shortage with no supply.** *Kind:* loop. *Loop:* buyers short of stock
(+) stock component pushes centre up (+) price (-) quantity bought and (+)
production response; with no response, nothing opposes the rise. Polarity R until
a bound. *Delay:* the gate. *Owner:* BEL 5.5, 17. *Damper:* D04 (ceiling), D01
(buyers' money). *Undamped:* the price rises each adjustment to the ceiling; this is
intended for a true shortage, and the register records that the ceiling is the only
stop in the market itself. *Condition:* production or imports respond before the
buyers' money runs out. *Test:* S09. *Status:* model (section 6.2: with a stop on
production, the price rises to 3 200 to 8 000 against a reference of 1 000).

**STB-L06 Overhang ratchet and the missing anchor.** *Kind:* loop. *Loop:* seller
stock above its band (+) downward push (-) price; if neither production nor demand
responds to price, the overhang never clears, so the push never stops. Polarity R
until the floor. The mirror case (a shortage that production does not answer)
reaches the ceiling (L05). With price-inelastic flows there is no interior
attractor: the sign of the accumulated imbalance decides which bound the price
reaches (section 6.2). *Delay:* the time to accumulate the overhang. *Owner:*
BEL 5.5; the missing response is AGT and PRO's, and AGT has answered that a
staple's flows stay inelastic (L51); DEM (round 5) gives a non-staple good a
price-responsive demand (D30). *Damper:* D04 (floor), D07
(inventory), D24 (production stop, specified: L44), D16 (spoilage, which removes the
overhang), D30 (a schedule, non-staples only). *Undamped:* the whole interior of a
staple; the market needs a price-responsive flow somewhere: a budget (CLR 3), a
production choice (AGT), exit (SPN 5, capped by `EntryShare` and `ExitShare`, so slow:
D35 and D36 are not an anchor, SPN 16). *Condition:*
the sum of the elasticities of supply and demand is positive and the response is
faster than the accumulation. *Test:* S16. *Status:* model. N-043.

**STB-L07 Width collapse and blow-up.** *Kind:* loop. *Loop:* many full fills
(+) narrower width (-) less exploration (-) fewer misses; the mirror for misses.
Polarity R within bounds. *Delay:* one adjustment. *Owner:* BEL 5.7, 17. *Damper:*
D17 (`BeliefWidthMin`, `BeliefWidthMax`). *Undamped:* widths pinned at an extreme
for a long run. In the inelastic run of section 6.2 the widths ended near the
maximum (4 925 to 4 962 of 5 000) in a market in disequilibrium. *Condition:* widths are not
all at one extreme in a long run of the reference chain. *Test:* S01, S10.
*Status:* model (observed, not isolated).

**STB-L08 Overconfident reference band.** *Kind:* hazard. *Loop:* none. A band
(`RefBandLow`, `RefBandHigh`) that is too narrow caps a famine price below the
clearing price; one that is too wide never binds and so is no anchor. *Owner:*
BEL 3.2.1, 17, open question 3. *Damper:* D04. *Undamped:* the choice of width.
*Condition:* a commodity in a shortage is not flat at the ceiling for long. *Test:*
S09. *Status:* untested.

**STB-L09 Hard cost floor and the frozen market.** *Kind:* loop. *Loop:* seller's
floor above what buyers can pay (+) no trades (+) unsold stock; with a perishable,
the stock spoils (the stock component would lower the price, but the floor blocks
it) (+) loss for the seller, which in the end leaves through exit. Polarity R until
exit. *Delay:* the shelf life, then the time to bankruptcy. *Owner:* BEL 3.2.2, 18.1;
N-039. *Damper:* D16 (oldest first; spoilage as a sink), the lower default
`CostFloor` of 7 000 on a cost that includes labour (BEL 3.2.2, FR-BEL-038 warns at
10 000 or more), exit by the failure ladder (D35, SPN 5; slow by design). *Undamped:* the freeze itself when the
market price falls below the floor, until the seller leaves. In the stylised model a
floor of 100% of cost against buyers capped at 91% of it gives no trade in any
tick and spoils 99% of the output, and a floor of 70% trades in every tick (section
6.2, run 5). BEL's reading is that the floor now binds only in a downturn, not in the
steady state, and tolerates a wage up to about 40% too high. *Condition:* `CostFloor`
low enough that a seller can sell at the buyers' price in a downturn, or exit fast;
the wage lag of PRO 9.3 (the cost is measured over the previous window) moves the
threshold. *Test:* S07, S03. *Status:* model.

**STB-L10 Cost-push ratchet.** *Kind:* loop. *Loop:* input price (+) unit cost
(+) the seller's floor (+) output price, which is an input of the next stage. In a
chain the loop closes around a recipe cycle. Since round 3 the unit cost includes
labour at the observed wage of the previous window (`spec-production.md` 9, N-055), so
there are two loops. (a) The material loop: around a cycle the gain is
`g = CostFloor / 10 000 * s`, where `s` is the full-cost share of the cycle's own
inputs. At the solver's prices `s` is `q_in / q_out` (input quantity over output
quantity), which is what validation computes exactly, since it has no prices
(`spec-config.md` 7.6 item 3, N-066 closed). (b) The wage loop: price (+) income (+)
wage (+) the cost, which includes labour (+) the floor (+) price, with gain
`CostFloor / 10 000` times the labour share where the floor binds (BEL 3.2.2). It
needs no recipe cycle, so it is flagged by its own warning, not by `cycle_cost_gain`:
gain 1 at `CostFloor` 10 000, a wage-price spiral that nothing damps, and 0.7 at the
default 7 000. Polarity R; stable if the gain is below 1, neutral at 1, a ratchet
above. *Delay:* the production duration and the wage window (PRO 9.3). *Owner:* BEL
17, PRO 9. *Damper:* D04 (cost ceiling and reference band), D23: the error
`cycle_not_productive` (V2, FR-VIA-006) for a cycle with no material surplus, the
warning `cycle_cost_gain` for `g >= 1` at the configured floor, and the warning
`cost_floor_at_full_cost` (FR-BEL-038, FR-CFG-056) for the wage loop at `CostFloor`
10 000 or more; the default `CostFloor` of 7 000. *Undamped:* a cycle with `g >= 1` at
the configured floor only warns, it is not rejected. *Condition:* `g < 1` on every
cycle and `CostFloor` below 10 000. *Test:* S08. *Status:* hand.

**STB-L11 Budget loop.** *Kind:* loop. *Loop:* no cash (+) no bids (+) no inputs
(+) no production (+) no income (+) no cash. Polarity R. *Delay:* one production
cycle. *Owner:* CLR 13. *Damper:* D01 (the hard budget stops the negative balance
of the original; it is the cost of this loop), D11 (relief), D15 (the floor under
market failure, `spec-subsistence.md`, not written). *Undamped:* an agent without
money stays without. *Condition:* relief or subsistence output keeps an agent above
zero. *Test:* S14. *Status:* hand. AGT-A3 merged: below the reserve money no
buffers are bought (less stock, less cash left for inputs), while the second loop
(money, inputs, output, income) is not blocked by the saving rule; the damper is also
D27, and an agent with no money, no goods and no relief stays undamped.

**STB-L12 Pairing bias.** *Kind:* bias. Greedy best-to-best matching with a random
tie-break is neutral between agents with equal limits. *Owner:* CLR 13. *Damper:*
the seeded tie-break key (`spec-clearing.md` 4.2). *Condition:* over many seeds no
agent position wins more often than the others. *Test:* a unit test over many seeds
(FR-STB-016). *Status:* untested.

**STB-L13 Herding of beliefs.** *Kind:* amplifier. Every agent in a market sees the
same report and uses the same rule, so beliefs move together. *Delay:* none.
*Owner:* BEL 17. *Damper:* D03 (the gate decorrelates), the random draw inside the
range, D10 (the floor `K` of agents per group). A cohort has one gate draw
(`spec-beliefs.md` 9.3), so it multiplies the effect (L35). *Undamped:* a cohort of
large weight. *Condition:* the dispersion of belief centres stays above a minimum
in a long run. *Test:* S10. *Status:* model (section 6.2: price noise rises from
4.7% to 9.1% as the weight goes from 1 to 10 at constant households).

### 4.2 Labour, wages and carriage (WEL, CAR)

**STB-L14 The target-income labour loop.** *Kind:* loop. *Loop:* wage `W` (+)
income per working day (-) days worked `D = K / W` (the target `K` is reached
sooner) (-) labour supplied (+) excess demand for labour or for the produced good
(+) price. Two negative links in the chain make the loop reinforcing: a shortage
raises the price, a higher price lowers the supply, and the shortage deepens.
*Delay:* the welfare window `T` (30 ticks) and the gate. *Owner:* WEL 5 (N-028); DEM 6.3
for a luxury role (merged: DEM-A2).
*Damper:* the price bounds (D04); the cap of one tick of work per tick (`DMAX`);
the one-tick minimum; households running out of money (D01). *Undamped:* the
interior equilibrium itself. *Condition:* demand elasticity above 1 at the
equilibrium (section 6.1). *Test:* S04. *Status:* model; the full analysis is in
6.1 and decides the question of N-028. **AGT's answer:** a staple's demand
elasticity is 0 to 1 and never above 1 (`spec-agents.md` 11.6, L51), so the
condition fails for a staple and the interior equilibrium is unstable. The default
strategy applies the rule with a rest band and a proportional work share for a
cohort (D09, D28), which soften steps and keep the unit-elastic supply; the survival
rule D26 does not change the elasticity. **D13 is adopted** by WEL as `MinWorkShare`
(`spec-welfare-and-wages.md` 5.3, default 4 000 basis points of the window, a
placeholder): below that share of working ticks the work share is 10 000. It moves the
upper stop from the price ceiling to a price set by the parameter and does not remove
the instability (WEL 5.4 says so). The register's position: pinned outcomes remain
possible for a staple role (F5), and S04 must record which one a pack reaches, with
`MinWorkShare` at its default and at 0. **DEM's answer for a non-staple (round 5,
DEM-A2):** a want has a schedule elasticity `e` (`Elasticity`, 10 000 is unit-elastic), so
a self-employed *luxury* role has excess-demand slope `(1 - e) Q / p` as in 6.1 and is
stable when `e` is above 10 000, but only with **budget headroom**: the household's
discretionary budget about 10% above the spending at the reference price, because with a
fixed budget the elasticity is exactly 1 and the loop is neutral. In DEM's model
(`spec-demand.md` 6.3, 14.1; not re-run here) `Elasticity` 15 000 to 30 000 with
headroom converges to 988 to 1 015 of 1 000, 5 000 ends at the work floor's stop (1 033
to 1 074) from above and at the band floor (100) from below, 10 000 is neutral, and the
lower edge of the basin is between 60% and 70% of the reference. DEM cannot make a
**staple** stable: its demand is upkeep (L51). The condition for a luxury is data
(`Elasticity`, the cap `SpendMaxShare`), checked by the warnings `want_inelastic` and
`wants_exceed_spend_cap` and by scenario S23; the damper is D30. *Test (added):* S23.

**STB-L15 Work and rest chattering.** *Kind:* hazard. A rule "work below the
target, rest at or above" switches every time the ratio crosses the target.
*Owner:* WEL 5.1. *Damper:* D08 (`TargetSpread`), D09 (the rest band `RestBand`
of the default strategy, `spec-agents.md` 7.1.2, which answers the "left to the
strategy" of WEL 5.1), D28. *Undamped:* a consumer's own strategy that applies no
band. *Condition:* the strategy applies a band. *Test:* S04. *Status:* hand.

**STB-L16 A carrier's fee loop.** *Kind:* loop. The same as L14 for hired
carriage: a carrier at its target stops taking contracts, which pushes fees up
(`spec-carriage.md` 2.4), and a higher fee lowers the days the carrier needs.
*Owner:* CAR 2.4, WEL 5. *Damper:* the merchant's ceiling `MerchantFeeShare` (a
share of its expected margin), D08. *Undamped:* as L14. *Condition:* the demand for
carriage is elastic in the fee: a merchant stops hiring when the fee exceeds its
margin share, which is a strong elasticity above the ceiling. *Test:* S04 with a
carrier role. *Status:* hand.

**STB-L17 Carriage cost and the basket.** *Kind:* loop. *Loop:* delivered food
price (+) the carrier's upkeep and feed cost (+) the cost of a haul unit (`UnitCost` of the
haul recipe, `spec-transport.md` 8.3, which replaced the reservation fee `fee_min`) (+) the
freight (+) delivered food price. Polarity R across regions; the gain is the share of the
delivered price that the carrier's food cost represents, which is below 1. *Delay:* a trip.
*Owner:* CAR 2.2; TRN 8.3 [TRN-A4]. *Damper:* none, but the gain is below
one for any real fee share. *Undamped:* a long route with a high basket price.
*Condition:* the carrier's upkeep is a small share of the delivered price (in
`spec-transport.md` 17.4: 6% for a full unit, 24% for a small cargo, placeholders). *Test:*
S04 on a two-region chain (REF), X1. *Status:* hand. The double count of the round trip
that `spec-carriage.md` 5.1 found concerned the `fee_min` form, which `spec-transport.md` 8
replaces; it is restated as open in `spec-transport.md` 20.2.

**STB-L18 In-kind valuation.** *Kind:* loop (weak). In-kind receipts count in
income at the prices of the region (`spec-welfare-and-wages.md` 6.2); a price rise
raises the measured income, so the agent reaches its target sooner and works less.
*Owner:* WEL 6. *Damper:* `InKind` is a small share by default (0). *Condition:*
`InKind` small, or the target-income rule off. *Test:* S04. *Status:* hand.

**STB-L19 A job is a gap in income.** *Kind:* loop. *Loop:* a job longer than
the welfare window `T` shows costs and no income (+) the ratio reads below 1 (+)
relief, migration and bankruptcy for a healthy producer (-) producers (-) supply.
Polarity R through the failure ladder. *Delay:* the job duration. *Owner:* PRO 16,
N-032; TRN 3.3.1 [TRN-A6] for a self-carried trip. *Damper:* D29, the derived window (`spec-welfare-and-wages.md` 2.3.3):
`T = max(WelfareWindow, WindowCycles * longest Duration on the role's list)`, so the
window always exceeds the longest job, and the PRO warning `job_longer_than_window`
(FR-PRO-040) can no longer occur. *Undamped:* the response of a long-job role to
relief, migration and the target-income rule is slower by design (over `T` ticks),
and a farm that cannot pay for its next job is still undamped (below); credit
(`spec-credit.md`) is the real remedy and version 1 has none. *Condition:* `T`
exceeds the longest job plus the selling time, which 2.3.3 makes true by
construction when `WindowCycles` is at least 1 and the selling time is less than the
job. *Test:* S11. *Status:* hand. Merged: AGT-A12 (tier 2, the job lump,
has no reserve; the household protects its food in stock, not its money) and VIA
8.3.1 (starting money `U * (d_max + buffer)`, so a warm-started household can pay
for the longest job). The starting money is a damper only for a household that starts
at that level; a household that falls below it later is undamped, apart from D11.
**A self-carried trip is also a gap in income** (TRN-A6): no income until the goods sell at the destination, so a trip longer than the effective window reads as starvation and can
trigger relief or a role change (rungs 1 and 2). D29 does not cover it (the window derives from recipes only); the warning `trip_longer_than_window` is the only guard, and a traveller that leaves with enough money and provisions (`unprovisioned`, `spec-transport.md` 5.4) survives it.
*Condition:* the longest trip of a travelling role is below its effective window. *Test:* S11's long-job pack with a travelling role. A trader's labour is also lost to production while it is away. `spec-welfare-and-wages.md` open question 9 (a window that counts trips) stays open.

**STB-L20 An uninterruptible job overshoots the target.** *Kind:* amplifier. A
started job cannot be interrupted (`spec-production.md` 5.2.2), so the target-income
rule overshoots its target by up to `Duration - 1` ticks; a cohort makes it `w`
households at once. *Owner:* PRO 16. *Damper:* D08. *Condition:* durations short
against the window. *Test:* S11. *Status:* hand.

### 4.3 Production, goods and money (PRO, GDS, LED, CAL)

**STB-L21 The production-delay cobweb.** *Kind:* loop. *Loop:* price (+) expected
price (+) starts of production (delay `d`) (+) later supply (-) price. Polarity B
with a delay, which is the cobweb. With naive expectations and `d` interleaved
independent chains, it converges if supply is less elastic than demand at the
equilibrium and diverges otherwise; the delay sets the period, not the threshold
(section 6.3). *Delay:* the recipe `Duration`, travel, the traineeship.
*Owner:* PRO 16, BEL 17; TRN 8.4 [TRN-A3] for the cobweb of the price of haul units (the delay is the haul recipe's `Duration`, the round trip; the short `ShelfLife` of the haul commodity ends banked capacity; a shipper's demand for haul units is derived and inelastic over the range where the price gap exceeds the freight, so the condition below is likelier to fail there, and the measured elasticity should be recorded; test S32). *Damper:* D02, D03 (beliefs are an averaging
expectation), D07, D14 (entry limits: `EntryShare` of the spawner, SPN 6.3, specified).
*Undamped:* with an unstable parameter set
the price crosses its whole band (a range of 100 to 1 between the bounds of the
model). *Condition:* the supply elasticity of the starts to the expected price is
below the demand elasticity, or the expectation is slow enough (section 6.3). A
staple fails the first form (L51), so for a staple the damper is the slow
expectation and the entry limit. *Test:* S05. *Status:* model.

**STB-L22 Lockstep of jobs.** *Kind:* amplifier. Agents free at tick 0 start
together, and all complete together. A seasonal glut is realistic; a glut of a
year-round recipe is an artefact. The random starting residue decorrelates yield
fractions, not start times. *Owner:* PRO 16. *Damper:* D22 (the warm start staggers
initial jobs, tool ages and lots at fixed phases, `spec-viability.md` 8.4.3 and
8.5, which answers `spec-production.md` open question 2). *Undamped:* a world not
started from the warm start. Merged: AGT-A8, the synchronised purchase: jobs complete
in one tick, the agents are free in one tick and buy their inputs in that tick, a
pulse of demand and then of price (`spec-agents.md` 7.3.2); and VIA's "lockstep of
jobs". *Condition:* completions are spread. *Test:* S18 (VIA A8). *Status:* hand.

**STB-L23 Seasons are a periodic forcing.** *Kind:* amplifier. Seasonal yields make
supply swing with a fixed period, exciting any loop with a delay close to it
(L21, L25). *Owner:* CAL 9. *Damper:* D07, D16. *Condition:* the reference chain
has a seasonal scenario in the acceptance bands. *Test:* S12. *Status:* untested.
*Note:* CAL's "a season shorter than a delay" does not arise for production, which
reads the season at job start (N-031 update); it remains for the coarse-tick case
below.

**STB-L24 A coarse tick hides a season.** *Kind:* hazard. With `DaysPerTick` above
1 a short season may not be sampled (`spec-calendar.md` 7.3). *Owner:* CAL 9.
*Damper:* the validation warning FR-CAL-016. *Test:* S12. *Status:* untested.

**STB-L25 Spoilage and stockpiling.** *Kind:* loop. *Loop:* a glut (+) spoilage of
whole lots at the shelf life (+) wealth destroyed (-) the next supply (+) the next
price. Whole-lot spoilage makes a sawtooth (`spec-goods.md` 4.5). A short shelf life
on a staple is a destabiliser. *Delay:* the shelf life. *Owner:* GDS 10, LED 14.
*Damper:* D16 (oldest first), the choice of shelf life (calibration). *Undamped:*
the sawtooth. *Condition:* the staple's shelf life is long against its production
cycle. *Test:* S07. *Status:* untested for the sawtooth; model for the freeze (L09).
The ledger entry "hoarding by spoilage" is merged here.

**STB-L26 Durables and the replacement cycle.** *Kind:* loop. A slowly wearing tool
is capital: replacement demand is low and lumpy until the stock ages together, then
spikes. *Delay:* the durability. *Owner:* GDS 10, SKL 2.6. *Damper:* D19 (different
ages across holders, set by the starting inventories of the chain's data).
*Condition:* ages are spread at the start. *Test:* S17. *Status:* untested. Transport
adds wear charged per tick of travel at dispatch (`spec-transport.md` 3.3.2), so a fleet
bought together wears and is replaced together (TRN-A7, merged).

**STB-L27 A single tool.** *Kind:* hazard. A recipe that requires a tool stops when
it breaks; the toolmakers' capacity is the damper (`spec-skill-barriers.md` 2.6).
*Owner:* PRO 16. *Test:* S17. *Status:* untested.

**STB-L28 Self-feeding recipes.** *Kind:* loop. Seed that makes seed: a poor yield
(-) next seed (-) next yield; positive loop that a poor yield breaks (a famine of
seed). *Owner:* PRO 16. *Damper:* none in PRO; the yield shock of 6.8 is how STY
tests it. *Condition:* a yield above the seed share with margin. *Test:* S08, S12.
*Status:* untested.

**STB-L29 Contested endowments.** *Kind:* hazard. The rotation of
`spec-production.md` 5.5.1 gives fairness over time, not within a tick; a deposit
that runs out takes its role with it. The depletion dynamics are RES's. *Owner:*
PRO 16. *Test:* in RES. *Status:* untested.

**STB-L30 Money that cannot leave.** *Kind:* hazard. With no sink a faucet's
cumulative total grows without bound; the boundary balances show it per reason
(`spec-ledger.md` 3.4). *Owner:* LED 14; MON 9.1 [MON-A2]. *Damper:* D41 (a cumulative
`Limit` ends the flow), D42 (a holding charge with a derived bound `m* = E + g/rho` on a
hoarding class), and the warning `faucet_without_sink` (a grant and no levy at all;
a grant with a `Limit` is exempt). *Undamped:* a pack that ignores the warning.
*Condition:* every grant has a limit or a sink (a rate levy on a class that holds the
money, or a matched fixed levy). *Test:* S13 (defined by `spec-money-supply.md` 17.6: the
faucet alone, report-only on the price until STY sets a band), S33 (the sink and the
bound). *Status:* hand for the hazard, model for the bound `m*` of a hoarder.

**STB-L31 Wealth held out of circulation.** *Kind:* hazard. The estate pool and
escrow hold money out of circulation. *Owner:* LED 14, EST 2.6. *Damper:* D12
(`PoolRelease`); for a hoarding class also D41 and D42 (`spec-money-supply.md` 9.3). *Test:* S14, S33. *Status:* untested.

**STB-L32 Non-negativity is a constraint, not a stabiliser.** *Kind:* constraint.
The ledger enforces non-negative balances by refusing an operation, so a shortage
shows as a refused order and a starving agent, never as a negative balance.
*Owner:* LED 14. *Damper:* the floor under it, D15 and D11. *Test:* the ledger
invariant (testing.md 6.3). *Status:* hand.

### 4.4 Scale (SCL)

**STB-L33 Noise from fewer traders.** *Kind:* amplifier. Fewer agents mean larger
swings from one belief update. *Owner:* SCL 10. *Damper:* D10 (the floor `K`),
the scale-consistency check (`spec-scale.md` 7.2). *Condition:* the price at scale
4 and 10 is within the tolerance of scale 1. *Test:* S10. *Status:* model (6.2).

**STB-L34 Lockstep of a cohort.** *Kind:* amplifier. The households of one agent
cross thresholds together, adding to the lockstep of the target-income rule.
*Owner:* SCL 10, N-014. *Damper:* D08 across at least `K` agents. *Undamped:* a
cohort of large weight with spread 0 for the belief gate (one draw per cohort). The
first consumer's world needs no weight above 1 (N-030). A cohort also places a want in
one pulse (L59). AGT-A13 merged: with the
proportional work share (D28, `spec-agents.md` 7.1.2, 9.3) the cohort's households
move in steps of `1 / w` as the ratio crosses the band, not in one step, which makes
the target-income lockstep smaller. *Test:* S10. *Status:* model for beliefs (6.2),
hand for the work share.

**STB-L35 Synchronised failure.** *Kind:* hazard. A cohort whose money reaches 0
fails whole; more generally a shock to the income of a role brings all its agents to the
same privation at once and they leave together (SPN-A1 merged). *Loop:* a shock lowers
the income of all agents of a role (+) they reach the same privation together (+) they
leave together (-) supply and demand of the role. *Delay:* the privation threshold
(`RemovePrivation`). *Owner:* SCL 10, SPN 5. *Damper:* the failure ladder of
`spec-spawner.md` 5: AGT's survival rule (rung 0), a role change at `ChangePrivation`
(rung 1), relief (rung 2, EST), and a partial weight reduction by the unfed share before
removal (rung 3), with graded exit D35 (`DeathStep`, `ExitShare`) counted by
`EntryLimited`. *Undamped:* a sudden loss of all food in a region with no pool leaves
exits at the cap while the rest starve; the ladder pays in long hunger (SPN 16.2: 36 000
unfed household-ticks against 316 for the "bazaar" policy that removes an agent after
one unfed tick, in a model with no goods). *Condition:* `ExitShare` small against the
share failing at once. *Test:* S14, S26. *Status:* model (SPN 16.2: the bazaar policy
empties the failing role in one tick; without an exit cap 130 of 300 leave in one tick,
with a cap of 1% to 2% 2 to 6).

### 4.5 Skills and entry (SKL, EST)

**STB-L36 The occupation-choice cobweb.** *Kind:* loop. *Loop:* a premium or an
income gap (+) entry into the role (-) the role's income per head (-) the gap.
Polarity B with delays: the welfare window `T`, the tool (`spec-skill-barriers.md`
2.3), the traineeship (`TrainingTicks`, 2 555 ticks for a seven-year course) and
the decision rate. The longer the delay and the larger the share that re-decides
each tick, the larger the swings (section 6.3). SPN-A2 merged. *Owner:* SKL 2.5, 4;
SPN 6. *Damper:* D14 (`EntryShare`: the most households that may change role per tick, as
a share of the region and of the destination, SPN 6.3; `MaxTrainees`, the queue), D09
(hysteresis for a change of role: `RoleHysteresis`, default 1 000 basis points, SPN 6.2),
D33 (staggered review: an agent reviews every `ReviewTicks`, by ID, no draw), D34
(cooldown `RoleCooldown`), D20 (the throttle `trainers * MaxTrainees /
TrainingTicks`). *Undamped:* an unthrottled free role (no tool, no training): in
the mean-field model a role loses and regains most of its members. In SPN's model with
discrete agents (SPN 16.1, 1 000 agents in 3 roles, `e` 0.5, window 30) review
rotation and a 10% hysteresis band, alone or stacked, **leave the swing at the whole
population** (amplitude 1.00); a cooldown of 30 alone brings it to 0.35 and an entry
share of 200 basis points alone to 0.30; with rotation, hysteresis and cooldown the
entry share gives 0.29 at 200 and 0.00 at 50 basis points. The condition is on `EntryShare` times the effective window:
the pairs that settle have the product at about 1 500 to 2 400 or less, those that do
not 6 000 or more; the warning `entry_share_window` fires above 2 500 (a placeholder).
The cost is slow adjustment. *Condition:* the share re-deciding per tick times the
window length is small (`EntryShare * T` below about 2 000), or a hysteresis band is
applied (a band alone is not enough). *Test:* S06, S26. *Status:* model (stylised;
mean-field in 6.3, discrete agents in SPN 16.1, not re-run here). SKL (round 3) adds the
expected wait in the queue to the application rule (4.5), which lengthens the delay
by a term and should shrink the swings of a role with a throttle and a queue; not
re-run here (neither model has a queue). SKL (round 3) adds the
expected wait in the queue to the application rule (4.5), which lengthens the delay
by a term and should shrink the swings of a role with a throttle and a queue; not
re-run here (the model of 6.3 has no queue).

**STB-L37 The trainer's incentive cliff.** *Kind:* hazard. If the gain per
trainee-tick is negative no places open and entry stops (`spec-skill-barriers.md`
4.3); a small price change flips it. *Owner:* SKL 4.3. *Damper:* the gain closes
new places and does not end a traineeship already in progress (SKL 4.3.1), and a
bounded `Warn` reports a role whose gain has been negative for ten ticks (SKL 8.3).
*Undamped:* it is still a cliff for new entry. *Test:* S06. *Status:* hand.

**STB-L38 Estate listing markdown.** *Kind:* loop. Unsold listings fall each tick by
`EstateMarkdown` until they sell; the dumped goods lower the local price.
*Owner:* EST 2.7.3. *Damper:* the resolution floor; the volume is bounded by the
estate. *Test:* S14. *Status:* hand.

**STB-L39 Pool-dry collapse.** *Kind:* loop. *Loop:* a collapsing region (+) a dry
pool (+) new agents start with no money (+) they fail fast (+) the region collapses
further. Polarity R. *Delay:* the time for a new agent to fail. *Owner:* EST 2.4.
*Damper:* D11 and D12 (relief from what remains), D37 (the spawner funds entrants
from at most `EntryPoolShare` of the pool per tick and keeps the rest for relief; see
L63), immigrants carrying money, D15 (not written). *Undamped:* once the pool is empty
and nothing arrives. *Condition:* a floor under market failure. *Test:* S14. *Status:*
hand.
With the population module (POP-A3, L77) the dead's holdings fill the pool, so the pool is
fullest when the dead are most numerous and funds the births and the relief that follow
(D47, D51); the loop still has no way back once a region is extinct.

**STB-L40 Relief suppresses migration.** *Kind:* hazard. Relief raises the ratio
that triggers migration (`specs/README.md` "Still open" 4). *Owner:* EST 2.6.
*Test:* an experiment (STY). *Status:* untested. With the population module relief also raises births and lowers deaths
(`spec-population.md` open question 4, L75); the same experiment covers it.

**STB-L41 The traineeship queue.** *Kind:* bias, **resolved in design**. The earlier
application rule did not count the time in the queue and so overstated the appeal of
a long queue (`specs/README.md` "Still open" 2). SKL 4.5 now includes the expected
wait in the rule, so a long queue lowers the appeal of joining. *Owner:* SKL 4.5.
*Damper:* the expected wait. *Test:* S06. *Status:* hand.

### 4.6 Configuration (CFG)

**STB-L42 A stable default can be undone by a pack.** *Kind:* hazard. A pack may
override any parameter, including ones whose defaults keep the economy inside the
bands. Validation cannot detect it. *Owner:* CFG 12. *Damper:* each shipped pack is
tested against the bands (FR-STB-015). *Test:* S01 to S14 on each pack.
*Status:* untested.

**STB-L43 A reachable parameter space is not a stable one.** *Kind:* hazard. Every
value inside its bounds is accepted; bounds prevent overflow, not instability.
*Owner:* CFG 12. *Damper:* the experiment harness and the viability check (D23).
*Status:* untested.

### 4.7 Agents (AGT), viability (VIA), demand (DEM) and spawner (SPN)

These entries come from `spec-agents.md` 18 (thirteen, named `AGT-A1` to `AGT-A13`
there), `spec-viability.md` 19 and 4.7, `spec-demand.md` 13 (eight, `DEM-A1` to
`DEM-A8`) and `spec-spawner.md` 17 (nine, `SPN-A1` to `SPN-A9`), merged by the
checklist of section 9 (9.9: one entry per loop). Where an AGT, VIA, DEM or SPN entry
describes a loop already registered, it is merged into that entry (listed next) and has
no new ID; the others are the entries below, L44 to L56 from AGT and VIA (round 3) and
L57 to L65 from DEM and SPN (round 5). Nothing here has been run except where a status
says `model`. No default is claimed stable (N-027).

| Source entry | Merged into | What was added there |
|--------------|-------------|----------------------|
| AGT-A3 poverty trap and the reserve | L11 | the reserve and the saving rule (D27) |
| AGT-A8 synchronised purchase; VIA "lockstep of jobs" | L22 | the stagger is now specified (D22); the purchase pulse |
| AGT-A10 level of unsold stock and the evidence offset | L02 | a note that the unsold-stock level is a property of the rules as written |
| AGT-A12 a job longer than the reserve; VIA 8.3.1 | L19, L20 | starting money covers the longest job; tier 2 has no reserve |
| AGT-A13 cohort work share | L34 | the proportional share (D28) |
| VIA "a glut at `lambda* > 1`" | L46 | the same loop as the closed economy |
| DEM-A2 a luxury market with an elastic demand and a target-income labour supply | L14 | the luxury case: stable if `Elasticity` is above 10 000 with budget headroom (D30); the lower edge of the basin; test S23 |
| DEM-A3 food first and the fall of wants in a staple shortage | L47 | wants fall first in a famine, by design (D31) |
| DEM-A6 a target the wants cannot absorb | L46 | with wants the mean ratio is `1 + W / U`, at most `1 + Wmax / U`; the warning `target_above_demand`; test S24 |
| DEM-A7 money below the sum of the reserves | L45 | with total money below the reserves plus `e*` no want is funded; test S25 |
| SPN-A1 synchronised failure | L35 | the failure ladder and graded exit (D35); test S26 |
| SPN-A2 the occupation cobweb | L36 | the entry share, hysteresis, review and cooldown (D14, D09, D33, D34); the condition on `EntryShare * T`; test S26 |
| SPN-A6 census imbalance | L54 | the transient is bounded by D14 and slowed by the window |
| SPN-A7 hunger, removal and demand (the SPN part) | L50 | the ladder supplies the exit; the rates are POP's |

**STB-L44 The production halt.** *Kind:* loop (AGT-A2). *Loop:* output stock (+)
halt of the recipe (-) supply (-) stock. Polarity B. *Delay:* the recipe `Duration`
(a halted recipe is not started; a started job completes). *Owner:* AGT 7.1.5.
*Damper:* D24 (`ProduceHaltRatio`, 30 000 basis points of the output target), D07
(the belief stock rule acts at 15 000, the halt at 30 000), D16. *Undamped:* where
the threshold sits decides whether an overhang ends at the floor or the ceiling in a
model with inelastic flows (6.2, run 1); a halt that every agent of a role reaches in
one tick stops them together (a cobweb source, L21); a halted household earns nothing
that tick. *Condition:* the halt is reached before the price reaches the floor of the
reference band, and the restart is gradual. *Test:* S16, S05, S19. *Status:* model
for the structure, hand for the threshold (a guess, N-015). In the model with
inelastic flows (6.2, run 1b) the default halt removes the floor pin and leaves a
shortage cycle: price 1.5 to 3.1 times the reference, a relative standard deviation
of 23%, 12% of need unmet. So the halt bounds the glut (L46) but does not by itself
anchor the price of a staple (L51).

**STB-L45 Saving, money out of circulation and the paradox of thrift.** *Kind:* loop
(AGT-A4). *Loop:* households' reserve money (+) money held (-) money in circulation
(-) sales and income (-) ability to hold the reserve. Polarity R, bounded: a
household holds at most `ReserveMoney` plus what it chooses to hold above it, and the
cap of `spec-agents.md` 7.4.4 lets the excess go. *Delay:* a tick. *Owner:* AGT 7.4;
touches L31. *Damper:* D27, D11, D12. *Undamped:* if the sum of the reserves exceeds
the money in the world, no household reaches its reserve, buffers are never bought
and the work rule runs at full. *Condition:* total household money at least the sum
of the reserve money. The warning `reserve_exceeds_money` (`spec-agents.md` 13)
checks a role's `StartMoney` against the reserve at the reference prices, which is
a check of the start, not of the total. DEM-A7 merged: with wants the same condition
has a second part, **total money at least the sum over households of the reserve plus
`e*`** (the excess a household needs to buy its wants, `spec-demand.md` 7.2); below the
sum of the reserves no household has an excess, no want is funded, the sellers of want
goods earn nothing and the money does not move. In DEM's model (`spec-demand.md` 14.2,
not re-run here) a total of 450 per household against a reserve gave a mean ratio of
0.5, and the money threshold lay between 900 and 1 800 per household; the model is
stylised and the numbers are placeholders. *Test:* S20, S25. *Status:* hand for the
reserve, model for the want threshold.

**STB-L46 The closed economy fixes the mean welfare ratio at 1.** *Kind:* hazard
(AGT-A5; VIA 4.7; N-054). *Loop:* none; an identity. With no faucet or sink and upkeep
as the only purchase, total income equals total upkeep spending, so the
household-weighted mean welfare ratio is 1 whatever `TargetRatio` says. A
`TargetRatio` of 15 000 is then unreachable by every role at once: the
target-income rule never rests, every household works, and the surplus `lambda* - 1`
that the steady state produces is unsold. The feedback it adds is positive (more
work, more supply, lower price, lower income, more work): it is L14 seen from the
other side. *Delay:* none in the identity. *Owner:* VIA 4.7; AGT 11.7 accepts the
glut; DEM 7 supplies the spending (DEM-A6 merged). *Damper:* D24 (the halt bounds the
glut), D16 (spoilage); with the demand module, D30 and D31 (wants are spending beyond
upkeep and are income to their sellers). *Undamped:* the glut until the halt, and a rule
that never rests. **With the demand module** (`spec-demand.md` 7.1) the identity becomes
mean welfare ratio `= (U + W) / U = 1 + W / U`, where `W` is the household spending on
want goods, so a ratio above 1 exists with no faucet; it cannot exceed `1 + Wmax / U`
(`Wmax` the spending with every want fully funded at its ceiling). A `TargetRatio` above
that ceiling is still never met (the work share stays at 10 000 and the sellers' surplus
is unsold): the warning `target_above_demand` flags it and D24 bounds the glut. In DEM's
model (`spec-demand.md` 14.2, not re-run here) the closed economy gave 1.000 with upkeep
only and 1.53 with wants at the target 15 000. The default `TargetRatio` stays 15 000
(decided, N-054; revisit when MON or EXT exist). *Condition:* a world where upkeep is
the only demand either sets targets at 10 000 or accepts the halt as the only brake; a
world with the demand module keeps the target within its ceiling. *Test:* S19 (no
wants), S24 (with wants). *Status:* hand (an identity), model for the figure 1.53. **For the population module** (`spec-population.md` 5.3, L81): with upkeep as the only demand a fed
household reads a ratio near 1 and a surplus is a glut, not a higher ratio, so the replacement ratio of the
schedules must lie below 10 000 (a fixed point at chronic hunger) or the population dies out;
with wants it can lie between 10 000 and `1 + Wmax / U`.

**STB-L47 Food first and the urgency bid.** *Kind:* amplifier (AGT-A6). *Loop:* hunger
(+) a bid at the top of the belief range (+) staple price (+) cash spent on the
staple (-) cash for other goods; and the staple price (+) hunger. Polarity R,
bounded by the cash limit and the belief ceiling. *Delay:* a tick for the hunger
counter. *Owner:* AGT 7.3, 7.5.3. *Damper:* D04, D01. *Undamped:* demand for other
goods falls in a staple shortage, by design. DEM-A3 merged: wants are funded from the
part of the saving rule's cap that buffers leave (D31), so in a staple shortage the
upkeep spending rises, the excess and the cap fall, and wants spending falls first; the
luxury sellers' income falls and with it their staple purchases (R, bounded by the cap
reaching 0). *Condition:* the staple's price stays below the ceiling in a shock.
*Test:* S04, S09. *Status:* hand.

**STB-L48 Recipe choice by margin is an occupation switch inside one agent.** *Kind:*
loop (AGT-A7; N-034). *Loop:* the net value of a recipe (+) its choice (+) its output
(-) its price (-) its net value. Polarity B with a delay; the rule is a maximum, so it
switches all at once when two values cross. *Delay:* the job's duration (an agent
chooses only when free). *Owner:* AGT 7.1.4; cousin of L36, inside a role. *Damper:*
D03 (beliefs move at different times, so agents cross at different times); D08
changes whether they work, not what. *Undamped:* no hysteresis in the recipe choice:
two recipes of nearly equal value flip with each belief move, and a long job holds the
flip for its length. *Condition:* recipes differ in value by more than the belief
noise, or few roles have several recipes. *Test:* S06 (a role with two recipes).
*Status:* hand.

**STB-L49 The buffer pulse (a bullwhip).** *Kind:* amplifier (AGT-A9). *Loop:* a rise
in use (+) the target `U * CoverTicks + Job` (+) a purchase larger than the rise, once
(+) the input's price. *Delay:* one tick. *Owner:* AGT 7.3. *Damper:* D27 (the spend
cap limits tier 3), favourability (7.5.2). *Undamped:* tier 2, the job lump, is not
capped. *Condition:* cover is small against the shock. *Test:* S01 (a step in upkeep).
*Status:* hand.

**STB-L50 Hunger, removal and demand.** *Kind:* loop (AGT-A11). *Loop:* unfed
households (+) removal or loss of weight (POP, SPN) (-) demand for the staple (-) its
price (-) hunger. Polarity B, Malthusian by design, with a long delay and a source of
cycles. SPN-A7 merged. *Owner:* AGT 5.5 supplies the signal; SPN 5 and 7 supply the
exit (rung 3, `AgentBankrupted` with a `Cause`) and the replacement (`Renewal`, default
10 000, which restores households and reopens the loop); the demographic rates are POP's
(the hook is `PopDelta`, SPN 8). *Damper:* D11, D35 (graded exit), D36 (an entrant
enters only a viable role). *Undamped:* POP's rates and the interaction of `Renewal`
with births (the warning `renewal_with_population`). *Condition and test:* the POP half
is L75 to L83 (`spec-population.md` 14: the rates are specified, `PopBirths` and `PopDeaths` replace the
signed `PopDelta`, `Renewal` is ignored while the module is on, D47, D48), tests S37 to S40; the SPN half is S14.
*Status:* model for the POP half (`spec-population.md` 13), which needs a fixed factor from RES
(L80) for its feedback; hand for the SPN half.

**STB-L51 A staple's demand is inelastic.** *Kind:* hazard (AGT-A1; N-043). *Loop:*
price (+) spending on the staple at a fixed quantity and (-) cash left; when cash
binds, spending is fixed, so the elasticity is 1; otherwise 0 (`spec-agents.md`
11.6). So a staple's demand elasticity is 0 to 1 and never above 1: the stability
conditions `e > 1` (L14) and `es < ed` (L21) cannot be met by a staple, and the
belief rules have no interior anchor (L05, L06). *Delay:* none in the rule. *Owner:*
AGT 7.3, 7.4, 11.6. *Damper:* D04, D01, D24 (L44), and exit (D35, D36: slow by design,
capped by `EntryShare` and `ExitShare`, so not an anchor; SPN 16, N-043). *Undamped:*
the interior of the price range of a staple. *Condition:* a price-responsive flow acts
before a bound is reached: the halt, exit, or an elastic demand. DEM (round 5) rejected
tier-1 rationing as an unsupported guess (`spec-demand.md` 17, N-052 closed) and
supplies the elastic demand for discretionary goods only, including an elastic
**increment on a staple** listed as a want item (L60). A pure staple is unchanged.
*Test:* S04 (records the elasticity), S16. *Status:* model (6.1 and 6.2 for the
structure; the AGT rules themselves are not modelled).

**STB-L52 The belief cost ceiling against a labour-dominated price.** *Kind:* hazard
(VIA 8.2.3; N-055). **Resolved in design in round 3.** *Loop:* none at first; it
became a pin. `UnitCost` excluded labour, so the zero-profit price added the wage: in
the VIA prototype the seed was 3.7, 9.7, 8.4 and 19 times unit cost for flour, dough,
a loaf and a tool, against `CostCeiling` of 3 times, so a seller with a labour share
above about two thirds was capped below its equilibrium price, could not earn the
wage and left (a pin, F5, then exit). PRO 9 now counts the recipe's labour at the
observed wage of the region over the previous window, and BEL 3.2.2 sets `CostFloor`
7 000 on that cost; the prototype's seed is then 1.00 of the unit cost, inside the
band of 0.70 to 3.00. The freeze of L09 becomes likelier at the old floor and so the
two were one decision (made). *Delay:* until the seller's money runs out. *Owner:*
BEL 3.2.2, PRO 9. *Damper:* the labour-inclusive cost; the solver still reports
`SeedAboveCostCeiling` and `SeedBelowCostFloor` (FR-VIA-027). *Undamped:* a region
with no observed wage yet has unit cost 0 (unknown), so a role's first ticks have no
cost band (PRO 9.1.1); the wage lag; a labour share so large that the wage itself
moves with the price (L10 wage loop). *Condition:* every seed lies inside the cost
band at the seed prices. *Test:* S21, S07. *Status:* hand (VIA's prototype numbers
as quoted by BEL and PRO; not re-run here).

**STB-L53 A warm start is not a stable start.** *Kind:* hazard (VIA 19). Starting at
the solver's point removes the transient of bad beliefs, not the dynamic
instability: a loop with an unstable equilibrium (L14, L21) leaves the point at the
first perturbation. *Owner:* VIA 13.1. *Damper:* none; the warm start is a start.
*Condition:* the loops of the register hold. *Test:* S22 (new): the same chain from
the warm start and from a cold start, perturbed. *Status:* untested.

**STB-L54 Census imbalance.** *Kind:* hazard (VIA 19, FR-VIA-028). A configured census
that differs from the balanced one starts a transient of role changes. *Owner:* VIA 5; SPN 6 (SPN-A6 merged).
*Damper:* D14 and D09 for roles (`EntryShare`, `RoleHysteresis`, SPN 6: they bound the
transient per tick and the window slows it). *Condition:* occupation shares return
after the transient. *Test:* S06, S26 and the check of VIA A1 (the balanced census and
one 20% off). *Status:* model (the transient from shares 0.4, 0.3, 0.3 in SPN 16.1, not
re-run here).

**STB-L55 Seasonal storage and the supply gap.** *Kind:* hazard (VIA V4). A gap in
supply longer than the shelf life of a good needed every tick makes a famine that no
price can cure. *Owner:* VIA 4.5; GDS 4, CAL. *Damper:* the error `storage_gap`
(FR-VIA-008, D23). *Condition:* the shelf life of a staple covers its supply gap.
*Test:* S12. *Status:* hand.

**STB-L56 Rounding residual of a free good.** *Kind:* bias (VIA 19, 6.5). A good priced
at the resolution floor adds revenue the exact solution lacks and changes the wage
balance of its producer (+10% on one activity in the prototype). *Owner:* VIA 6.
*Damper:* the report `ResidualWarnBP` (FR-VIA-025). *Condition:* the residual stays
under the stated limit for the reference chain. *Test:* VIA A1. *Status:* hand.

The entries below come from the demand and spawner specs (round 5). The entries of
those specs that describe a loop already registered are merged above and not repeated:
DEM-A2 into L14, DEM-A3 into L47, DEM-A6 into L46, DEM-A7 into L45, SPN-A1 into L35,
SPN-A2 into L36, SPN-A6 into L54, SPN-A7 into L50.

**STB-L57 The wage and demand loop.** *Kind:* loop (DEM-A1). *Loop:* wants spending (+)
income of the sellers of want goods (+) their money above the reserve (+) their wants
spending. Polarity R, with a gain equal to the marginal propensity to spend, at most
`SpendMaxShare / 10 000` and falling with the excess (`spec-demand.md` 8.4). *Delay:*
a tick for the money, the welfare window for the ratio. *Owner:* DEM 4.5, 8.4; replaces
the DEM part of the hook row "DEM, EMP" (the employment and wage-setting side stays with
EMP); touches L45. *Damper:* D27 (the cap is concave and bounded by the excess), D31
(wants are funded only from what the buffers leave). *Undamped:* nothing beyond the cap:
the multiplier is finite by construction. *Condition:* the marginal propensity to spend is
below 1, which the cap guarantees for `SpendMaxShare` below 10 000. *Test:* S24.
*Status:* model (DEM 14.2: the closed-economy ratio converges to 1.53 and not
unboundedly; not re-run here).

**STB-L58 Substitution flips.** *Kind:* amplifier (DEM-A4). *Loop:* an item's price (+)
its substitution share (-) demand for it (-) its price. With a high `Substitution`
exponent a small price gap moves most of a want's demand to the cheaper alternative, and
with the belief lag the shares can swing between the items. *Delay:* the belief step.
*Owner:* DEM 4.2. *Damper:* D03, D02, an integer exponent of at most 4 (`MaxSubstitution`).
*Undamped:* a pair of identical goods with `Substitution` 4. *Condition:* the want's
alternatives differ in weight, or the exponent is small. *Test:* S25. *Status:* hand.

**STB-L59 Periodic wants and cohort pulses.** *Kind:* amplifier (DEM-A5). *Loop:* none. A
want with a long `Period` is placed once per `Period` ticks from a per-agent carry, and a
cohort places `w` units at once. *Owner:* DEM 4.4, 4.8. *Damper:* the carry start spread
(`(AgentIndex * 7 919 + position * 2 777) mod 10 000`, no draw), the floor `K` of agents
per group (D10), different `Period`s. *Undamped:* a world with few agents and a long
period has pulses. *Condition:* many agents of weight 1, or periods that differ. Delays
that match a neighbour (checklist 9.6): none found; a want bought in a tick is consumed
in it, so shelf life does not interact. *Test:* S10. *Status:* hand.

**STB-L60 A staple that is also a want item.** *Kind:* bias (DEM-A8). *Loop:* none. If a
commodity that is upkeep for one role is a want item for another, or an elastic
increment of the same role, its demand is the inelastic base plus the elastic increment,
so its elasticity is above AGT's 0 to 1 by the share of the increment and no more
(`spec-demand.md` 6.1). It does not make the staple's labour loop (L14) stable; it
softens the pin by that share. *Owner:* DEM 6.1. *Damper:* none needed. *Condition:* the
increment's share of demand is large enough to matter; otherwise L51 holds. *Test:* S04
with the staple as a want item. *Status:* hand.

**STB-L61 Replacement by the most profitable role (the bazaarBot herd).** *Kind:* hazard
(SPN-A3). *Loop:* the best role's expected ratio (+) all replacements go into it (-) its
income per head (-) its ratio, with the window as delay. Polarity B with a delay, so a
cobweb: bazaarBot replaces every leaver by the role with the best signal and empties the
failing role in one tick (SPN 16.2: share 0 to 1). *Owner:* SPN 7.2. *Damper:* D36
(like for like first, and a newcomer enters only a viable role), D14 (the cap on
cross-role entry). *Undamped:* a role that is both the best and the only viable one.
*Condition:* as L36 (`EntryShare * T` small). *Test:* S26. *Status:* model (SPN 16.2).

**STB-L62 The churn of a famine.** *Kind:* hazard (SPN-A4). *Loop:* an unviable region
(+) deaths (+) replacements funded from the dead agents' money (+) deaths. Polarity R, if
replacements enter. *Owner:* SPN 7.2. *Damper:* D36 (no viable role, no entrant), D37
(funding limit). *Undamped:* a role that looks viable on a stale wage, because the
viability `V` of a role is a window mean. *Condition:* the window is short against the
collapse. *Test:* S27. *Status:* hand.

**STB-L63 The pool serves two masters.** *Kind:* loop (SPN-A5; extends L39). *Loop:* a dry
pool (+) unfunded entrants (-) agents (-) households to relieve; and relief (-) pool
(-) entrants. *Owner:* SPN 7.3; EST 2.6. *Damper:* D37 (`EntryPoolShare`, default
5 000, so at least half of the pool stays for relief), D11, D12, D35. *Undamped:* a pool
that is empty when nothing arrives. *Condition:* money enters the region (immigrants,
merchants). *Test:* S14. *Status:* hand. The population module's births draw the stake from the same pool (L76, D51), money flows to a
pool come from `spec-money-supply.md` (L73), and cargo whose consignee is gone is diverted to the
destination estate and so to the pool (TRN-A8, `spec-transport.md` 5.8; bounded by the shipments in flight at a
removal; D37).

**STB-L64 An extinct role is never re-seeded.** *Kind:* hazard (SPN-A8). A role with no
agents has no wage and no signal, so no one enters it (SPN 6.5); a role that never had a
group and is closed (the reference chain's `visitor`) is not extinct. *Owner:* SPN 6.5.
*Damper:* none by design; a bounded `Warn` "role extinct" (`spec-spawner.md` 13.3).
*Condition:* a pack whose roles can all die has no way back except an external agent or a
consumer's action. *Test:* S27. *Status:* hand. The population module's floor `MinHouseholds` (D49) keeps the schedule from extinguishing a small group;
the ladder alone can end a role, and an extinct region is absorbing (`spec-population.md` 6.4).

**STB-L65 The cost of switching.** *Kind:* bias (SPN-A9). Goods outside the new role's
tradable set go to the estate and the job is abandoned (`spec-spawner.md` 4.2), so a
switch costs the agent; the cost works as hysteresis in effect and shows in the event's
`GoodsListed`. *Owner:* SPN 4.2. *Damper:* none needed. *Condition:* none. *Test:* S06.
*Status:* hand.

### 4.8 Hooks for specs not yet written

These entries do not exist yet. Each future spec adds its own by the checklist of
section 9. The list says what each is expected to bring, from the notes and the
roadmap, so that no loop is forgotten. The agents, viability, demand and spawner specs
have been merged (4.7), and the region, money, population and transport specs (4.9), and no longer have a row. The reference chain (REF) has no row:
it adds no rule and so no loop (`spec-reference-chain.md` 13, FR-STB-003); it builds the
scenarios of section 8.

| Spec | Expected entries | Where it is known from |
|------|------------------|------------------------|
| MIG, INF | migration on rumoured ratios: a loop with the rumour delay (the distances of REG feed it); a flow toward a region lowers its ratio; relief and births raise it (L40, L75); whether a migrant keeps or reseeds its beliefs (SPN leaves it to MIG) | roadmap; SPN 7.4; REG 6 |
| EMP | the employment and wage-setting side of the wage and demand loop (the demand side is L57); wages are demand, demand raises prices and wages; and DEM's need for a recurring named transfer to a non-producing class with a paired sink or return flow (`spec-demand.md` 7.3) | roadmap "Macro is measured"; DEM 7.3 |
| EXT, SAV, CRD | the outside world's flows (the primitives reserved by MON 10), saving and investment, credit cycles and default | roadmap; MON 10 |
| MER, STO, RSK | the merchant's decision to ship (the loop of L84 is closed by its response), storage and its cost, loss in transit (the `loss` reason is RSK's) | roadmap; TRN 5, 10 |
| FIS, PLC | policy rules that react to measures (a controller in the world, which `ADR 0001` allows only as a policy actor) | ADR 0001 |
| RES, LND, SUB | depletion and regeneration, rent, subsistence output as the floor (D15) | roadmap |
| WLD | the decision rates (D25) and the phase order the register assumes (`spec-agents.md` 4) | roadmap; N-009 |

### 4.9 Regions (REG), money (MON), population (POP) and transport (TRN)

These entries come from `spec-regions.md` 13 (three, `REG-A1` to `REG-A3`),
`spec-money-supply.md` 16 (seven, `MON-A1` to `MON-A7`), `spec-population.md` 14 (nine,
`POP-A1` to `POP-A9`) and `spec-transport.md` 15 (eight, `TRN-A1` to `TRN-A8`), merged by
the checklist of section 9 (9.9). The new IDs are L66 to L86; the writers' provisional IDs
stay in the Owner field in brackets. The dampers are D38 to D52 (section 5) and the
scenarios S28 to S40 (section 8). A section number in an entry that names no spec is a
section of the spec named in its Owner field. REG closes no loop of its own (checklist
9.1): its distances are inputs to the delays of the loops of TRN, MER, MIG and INF, which
enter them. Nothing here has been run except where a status says `model`. No default is
claimed stable (N-027).

| Source entry | Merged into | What was added there |
|--------------|-------------|----------------------|
| MON-A2 money that cannot leave | L30 | the limit, the holding charge and the warning `faucet_without_sink` (D41, D42); the faucet scenario S13 is now defined |
| TRN-A3 the haul capacity cobweb | L21 | a case of the production-delay cobweb: the delay is the haul recipe's `Duration`; the shipper's demand for haul units is derived and inelastic over the range where the gap exceeds the freight; test S32 |
| TRN-A4 freight and the delivered price | L17 | the `fee_min` form is replaced by the price of haul units (`spec-transport.md` 8.3) |
| TRN-A6 a trip is a gap in income | L19 | a self-carried trip is not covered by D29; the warning `trip_longer_than_window` |
| TRN-A7 vehicles are capital | L26 | wear charged per tick of travel at dispatch |
| TRN-A8 diversion transfers wealth to the pool | L63 | cargo whose consignee is gone goes to the destination estate |
| POP-A1 to A9 | L50, L46, L39, L40, L64 (edited) and L75 to L83 | see each entry |

Cross-references between owners (R18): money flows and births draw on the same regional
pool (L72, L73, L76, L77); a per-household grant couples MON and POP (L72).

**STB-L66 A region that cannot be reached.** *Kind:* hazard. *Loop:* none. A region that imports an upkeep good (VIA `region_not_self_sufficient`) and has no path from a region that produces it is a closed economy that starves, and the cause is the graph, not the economy. A closure (7) can create the situation in a run. *Owner:* REG 4.5, 7 [REG-A1]. *Damper:* none by design; the warnings `region_isolated` and `graph_not_strongly_connected` (9.2), VIA's warning, and the event `EdgeStatusChanged` make it visible. *Undamped:* a closure that partitions the realm. *Condition:* every region that imports has a path from a region that supplies it, at construction; in a run the condition is the shock's. *Test:* S28. *Status:* hand.

**STB-L67 A small market is a thin market.** *Kind:* hazard (the cost of N-026). *Loop:* none of its own. Splitting a town from its hamlets gives each hamlet a market with a few agents; a thin market's price is mostly its carried-forward price (`spec-clearing.md` 5.4), so the belief and price loops (L01 to L09) run on few orders, and each hamlet's pool and failure ladder are its own (L39, L63). Trade between them needs a carrier and a margin (TRN, MER), which do not exist yet. *Owner:* REG 3.4 [REG-A2]. *Damper:* none in this spec; the short edge makes the arbitrage cheap once TRN exists, and a setting that does not want the effect writes one region. *Condition:* a region holds enough agents to keep the group floor `MinAgents` (D10) or is accepted as thin. *Test:* S29. *Status:* hand.

**STB-L68 A household takes its money with it.** *Kind:* bias. *Loop:* none of its own. A relocation moves money and goods between regions with no flow (6.3). The money of an arriving household enters the destination's demand and the sender's loses it; the loop that follows (migration raises demand and prices in the destination, which lowers the ratio that triggered it) is MIG's, and the hook is in the register (4.8, MIG row). The bias here is that a report of money by region changes with no flow, which the relocation totals (6.4) reconcile. *Owner:* REG 6 [REG-A3]. *Damper:* none needed. *Condition:* none. *Test:* S14 with a household that moves. *Status:* hand.

**STB-L69 The fixed nominal faucet and the price level.** *Kind:* loop with drift. *Loop:* price level (-) real value of a fixed nominal grant (+) real demand of its recipients (+) price level. One `-` link, so balancing (B). The drift is not a loop: `M` rises by `g` a tick and, at constant real volume and turnover, the price level rises at `g / M(t)`, a rate that falls as `1/t` (9.1). *Delay:* the belief adjustment (`spec-beliefs.md` 5.2, a few ticks) and, for an income grant, the welfare window `T`. *Owner:* MON 9.1 [MON-A1]. *Damper:* D04 (the reference band bounds the price), D41, D42. *Undamped:* the nominal drift of an unbalanced faucet, by design, and measured. *Condition:* real volume and turnover change slowly against `M`. *Test:* S13, S33. *Status:* hand (a quantity-theory reading, untested in the library).

**STB-L70 The holding charge.** *Kind:* loop. *Loop:* the class's money (+) the amount collected (+) money removed (-) the class's money. B. *Delay:* the time constant `1/rho` ticks (730 in the example of 9.3), against the welfare window `T` (30 to 120 ticks) and a year; a charge fast enough to be near `T` (`rho` above about 1/30) acts within a window and can take a household's savings for a long job (L19, L20), which the exemption is for. *Owner:* MON 6.3 [MON-A3]. *Damper:* itself; D43 protects the savings. *Undamped:* none known; a side effect is that holding money costs, so a household that saves for a tool pays for it. *Condition:* `Exempt` covers the reserve and the longest job's cash. *Test:* S33. *Status:* model for a hoarder (the strategy that spends is untested).

**STB-L71 A levy that takes upkeep money.** *Kind:* hazard. *Loop:* levy (+) money held (-) upkeep bought (-) unfed households (+) privation (+) the ladder (role change, relief, exit; `spec-spawner.md` 5) (-) the levied base. B in the end, but through hunger. *Delay:* the privation count. *Owner:* MON 9.4 [MON-A4]. *Damper:* D43, D11 (relief refills the poor from the pool). *Undamped:* a fixed levy without an exemption on a class with no income above its upkeep. *Condition:* `Exempt` at least one tick's upkeep cost (the warning `levy_below_upkeep`). *Test:* S34. *Status:* hand.

**STB-L72 A per-household grant and the population.** *Kind:* loop, owned jointly. *Loop:* grant (+) income and welfare ratio (+) births (POP) (+) households (+) the issue (the grant is per household). R, bounded by POP's carrying capacity and the food loop (L50). *Delay:* the welfare window and POP's. *Owner:* MON 4.3, 7.2; the rate is POP's [MON-A5]. *Damper:* `NotIncome`, D41, `Scale` (10.4), or a grant to the pool instead of to households, whose total does not depend on `H`. *Undamped:* POP's side. *Condition:* the decision of POP on whether income from a grant raises births. *Test:* in POP. *Status:* hand-off.

**STB-L73 Growing households in a closed economy.** *Kind:* hazard (neighbours STB-L31, L45, regime F4). With `M` fixed and `H` rising, `M_h / H` crosses the spending threshold and then the liquidity floor (8.5, 8.6). *Delay:* the entry rate. *Owner:* MON 8 [MON-A6]. *Damper:* D11, D12 (the pool recycles what the dead leave), D31 (wants fall first, by design), and a pool grant sized to the expected entry (8.6). *Undamped:* a closed economy whose households grow past the thresholds. *Condition:* `M_h / H` at least the spending threshold; the measure is in the Snapshot (13.4). *Test:* S35. *Status:* hand.

**STB-L74 A top-up peg.** *Kind:* amplifier (for EXT). A class topped up to `Level` has a budget fixed in money per tick, so its demand is unit-elastic in price (spending constant). A seller facing it has the neutral case `e = 1` of `spec-stability.md` 6.1. *Owner:* MON 10.2, EXT [MON-A7]. *Damper:* none in MON; the price bounds. *Condition:* the seller side is EXT's. *Test:* in EXT. *Status:* hand.

**STB-L75 The Malthusian loop.** *Kind:* loop. *Loop:* households (+) crowding of the fixed factor (-) the ratio (+) net growth `birth - death` (+) households. Polarity B (an odd number of negative links), Malthusian by design, with a long delay. *Delay:* the welfare window `T` (mean `T / 2`), the economy's own lag `tau_e`, and nothing else (no generation lag, 3.3). *Owner:* POP 5; the third assumption is RES's; extends L50 and L46 [POP-A1]. *Damper:* D46, D47, D52, and the rate bounds (a rate is at most 10 000). *Undamped:* a long `tau_e`: the warning sees only the window. *Condition:* `k * tau` below about 1 (model: monotone to 0.7, overshoot from 0.9 to 1.4, a cycle from about 2); `k` from the schedules and `eta` of the pack. *Test:* S37. *Status:* model (13.1). *Needs a fixed factor (5.2); without one the loop has no feedback.*

**STB-L76 The stake loop.** *Kind:* loop. *Loop:* households (+) money per household (-) funded births (+) households. Polarity B, a spurious Malthusian limit set by money. *Delay:* the pool's turnover (a death to a birth). *Owner:* POP 4.3, 5.5; touches EST 2.3 and SPN 7.3, extends L63 [POP-A2]. *Damper:* D51, D7; money from outside (MON, MIG). *Undamped:* a closed region cannot grow past `M / stake`. *Condition:* `M / stake` above the food-limited `N*`. *Test:* S39. *Status:* model (13.2).

**STB-L77 Recovery needs the money of the dead.** *Kind:* loop (damper of L39). *Loop:* collapse (+) deaths (+) the pool's money (+) funded births and relief (-) the collapse. Polarity B, with the delay of the births. *Owner:* POP 6.2, 6.3; EST 2.1, 2.6 [POP-A3]. *Damper:* D47, D51; D11, D12, D37. *Undamped:* an extinct region (absorbing). *Condition:* households remain. *Test:* S14, S38. *Status:* hand.

**STB-L78 Selection by role.** *Kind:* loop. *Loop:* a role's ratio (+) its births, net of deaths (+) its share of households (-) its ratio (a wage falls with supply). Polarity B, much slower than the spawner's role change (L36). Births enter the parents' role (3.2), so roles with a higher ratio grow faster and the spawner's damped switching acts on a population whose composition also drifts. *Owner:* POP 3.2; SPN 6 [POP-A4]. *Damper:* D14, D09 for the switching; rates bounded. *Condition:* the two rates of change are far apart (births 2.5% a year against switching per tick). *Test:* S06, S37. *Status:* hand.

**STB-L79 Renewal refills the check.** *Kind:* hazard. *Loop:* ladder removals (-) households (+) renewal (+) households. Polarity R if enabled: a second positive loop on the same flow. *Owner:* POP 4.6; SPN 7.1 [POP-A5]. *Damper:* D48. *Undamped:* none while the module is on. *Test:* S40. *Status:* hand.

**STB-L80 No fixed factor.** *Kind:* hazard. Without a fixed factor the ratio does not fall with households and the rate has the same sign for any `N` (5.2). *Owner:* POP 5.2; RES [POP-A6]. *Damper:* D50, D51 (the money). *Condition:* a fixed factor (RES). *Test:* S37 with a double. *Status:* hand.

**STB-L81 The closed ceiling of the welfare ratio.** *Kind:* hazard (N-054; L46). In a closed economy with upkeep only the mean ratio is at most 1 (5.3): the schedule cannot separate abundance from sufficiency, so the fixed point is at chronic hunger (`r*` below 10 000) or the population dies out (`r*` above it). *Owner:* POP 5.3; DEM 7 [POP-A7]. *Damper:* wants (DEM) or a money source; the warning `replacement_above_closed_ceiling`. *Condition:* the pack's ceiling `1 + Wmax/U` above `r*`. *Test:* S40. *Status:* hand (a derivation from the cited identity).

**STB-L82 A low-density trap.** *Kind:* hazard. At low density the ratio may fall with households (a chain without enough households to run), making a low fixed point stable and `N*` unstable above it (6.7). *Owner:* POP 6.7 [POP-A8]. *Damper:* D49 only. *Condition:* a pack whose chain survives at the smallest group. *Test:* S38. *Status:* hand.

**STB-L83 The ladder's cap bounds the positive check.** *Kind:* constraint. A severe shortage is checked by the ladder, capped at `ExitShare` a tick and delayed by `RemovePrivation` (4.5.1); the schedule does not see unfed households. *Owner:* POP 4.5; SPN 5 [POP-A9]. *Damper:* D35. *Condition:* a pack wanting a faster check sets the ladder. *Test:* S14. *Status:* hand.

**STB-L84 Price-gap shipping with a transit delay.** *Kind:* loop. *Loop:* the price gap between a dear and a cheap region (+) the quantity shipped (+) the price in the cheap region, at dispatch, where the shipper bought (-) the gap; and the gap (+) the quantity shipped (+, after `k` ticks) the arrivals in the dear region (-) the price there (+) the gap. Each path has one negative link, so each is balancing (B); the first is immediate and the second has the delay. *Delay:* the travel time `k` (section 4). *Owner:* TRN 5; the decision to ship is MER's [TRN-A1]. *Damper:* the freight is a floor on the gap that triggers a shipment, so the loop closes the gap to the freight cost and not to zero (L17); a shipper that counts `Inbound` (9.1) removes the delay from its signal; a limit on what can be shipped per tick (the capacity of one carrying unit, `MaxDispatchPerTick`) bounds the swing. *Undamped:* shippers that ignore their pipeline and respond strongly. *Condition:* in the linearised model of section 16, with `g` the fraction of the gap a shipper tries to close per tick and none of the effect at dispatch, the gap decays if `g < 2 sin(pi / (2 (2k + 1)))`, about `pi / (2k + 1)`: 1.0 at `k` = 1, 0.62 at 2, 0.35 at 4, 0.18 at 8; with at least half of the effect at dispatch (the purchase in the origin region moves its price at once) or with a shipper that counts `Inbound`, the limit is at least about 2 for any `k`. A cap on the shipment per tick bounds the swing of the gap to about `k` times the cap. *Test:* S30. *Status:* model (section 16). It does not show the library: it has no inventory, no belief and no market, and the shares of the effect at dispatch and at arrival are not derived for any pack.

**STB-L85 The pipeline is invisible.** *Kind:* amplifier. *Loop:* a consignee's stock (excludes goods on the way) (-) its orders for the good (+) the goods on the way (+) its stock on arrival: a buyer that sees only its stock orders again while its goods are in transit, then holds twice what it wanted `k` ticks later and stops ordering, an oscillation of period about `2k` that grows with the delay. R in the stock signal, with the delay `k`. *Owner:* TRN 9.1 (the `Inbound` field); AGT's `Target` (`spec-agents.md` open question 20) [TRN-A2]. *Damper:* the buffer cap and the production halt (D24) bound the overhang once it exists; `Inbound` is the means to avoid it. *Undamped:* a strategy that does not count `Inbound`. *Condition:* the stock a strategy sizes against includes `Inbound`. *Test:* S30 with a consignee that buys and receives. *Status:* hand.

**STB-L86 Perishables cannot cross.** *Kind:* hazard. A commodity with `ShelfLife` `S` cannot be delivered by a route of `k >= S` ticks (5.6.2), so a price gap in it never closes and its shipments, if a strategy makes them, spoil whole. Not an instability: a pinned gap, counted by `ShipmentArrived.Spoiled`. *Owner:* TRN 5.6 [TRN-A5]. *Damper:* the warning `route_too_slow_for_shelf_life` and the strategy's own use of `Routes`. *Condition:* a strategy ships a perishable only if `k <= S - 1 - age`. *Test:* a unit test of 5.6.2 and S07's perishable in a two-region pack. *Status:* hand.

## 5. The catalogue of dampers

A **damper** is a rule or parameter whose job is to limit a feedback loop. This
catalogue is the index of the dampers in the design. "Home" is the spec and section
that fixes the mechanism; "Status" is *specified* (a written spec fixes it) or
*proposed* (this spec suggests it and no spec owns it). Counters in events make a
damper observable (FR-STB-013).

| ID | Damper | Mechanism | Home | Status |
|----|--------|-----------|------|--------|
| D01 | Hard budget and limit price | every order is backed by a reservation and rejected whole otherwise; no negative balance | CLR 3 | specified |
| D02 | Bounded step | a belief moves by at most `2 * BeliefStep` of its centre per adjustment | BEL 5.6 | specified |
| D03 | Asynchronous adjustment | an agent adjusts with probability `BeliefAdjustProb` | BEL 5.2 | specified |
| D04 | Price bounds | the reference band and the cost band clamp every belief | BEL 3.2 | specified |
| D05 | Evidence from age and quotes | a stale price is not evidence; the best unmatched quotes move a belief off it | BEL 5.4; CLR 5.4, 5.5 | specified |
| D06 | Midpoint pricing | the trade price is the midpoint of the two limits, half to even | CLR 5.1 | specified |
| D07 | Inventory buffer | the stock ratio moves price and delays a supply interruption; it is an integrator (L04) | BEL 5.5 | specified |
| D08 | Spread of targets | `TargetSpread` gives agents different targets so supply responds smoothly | WEL 5.1 | specified |
| D09 | Hysteresis band | an agent rests only when its ratio is `RestBand` above its target, and a cohort answers in proportion; a margin before switching | AGT 7.1.2, 9.3 (`RestBand`); SPN 6.2 (`RoleHysteresis`, default 1 000 basis points) | specified for the work decision (AGT) and for a change of role (SPN; not enough alone, SPN 16.1); proposed for recipe choice (L48) |
| D10 | Floor of agents | at least `K` agents per group (`MinAgents`) | SCL 5.2 | specified |
| D11 | Relief | a fraction of the pool is paid to agents below subsistence | EST 2.6 | specified |
| D12 | Pool release rate | `PoolRelease` keeps the pool from sitting idle | EST 2.6 | specified |
| D13 | Work floor | if the working ticks in an agent's window are fewer than `MinWorkShare` of it, the work share is 10 000, so supply cannot fall below a stated level as the wage rises; it moves where a pinned market stops and does not remove the instability | WEL 5.3 (`MinWorkShare`) | specified (a placeholder default; counted by a snapshot figure and a bounded `Warn` "work floor binding") |
| D14 | Entry limit | `EntryShare` (default 50 basis points, a placeholder): a cap on the households that may change role per tick, as a share of the region's households and of the destination's; `MaxTrainees`; a slew limit on starts. The effective occupation damper in SPN's model; the condition is on `EntryShare` times the effective window (L36) | SPN 6.3 (`EntryShare`); SKL 4.4 | specified (counted by `EntryLimited`) |
| D15 | A floor under market failure | household production outside the market, and relief | SUB (not written); EST 2.6 | proposed |
| D16 | Spoilage and oldest first | goods spoil whole at a fixed age and are sold oldest first; spoilage removes an overhang | GDS 4; LED 5 | specified |
| D17 | Width bounds | `BeliefWidthMin`, `BeliefWidthMax` | BEL 5.7 | specified |
| D18 | Conservation and non-negativity | the ledger refuses what it cannot apply | LED | specified |
| D19 | Spread of ages | different ages across holders of a durable, set by the starting inventories | SKL 2.8; GDS 10 | specified (data) |
| D20 | Training throttle | `skilled_per_tick <= trainers * MaxTrainees / TrainingTicks` | SKL 4.4 | specified |
| D21 | Resolution floor | every tradable commodity's reference price is at least `ceil(10 000 / BeliefStep)` | CUR 3.1; BEL 5.3 | specified |
| D22 | Staggered warm start | initial jobs, tool ages and perishable lots placed at fixed phases (agent `i` of `n` at `floor(i * P / n)`, no random number) so they do not complete together | VIA 8.4.3, 8.5; FR-VIA-037, 038 | specified (only when a warm start is configured) |
| D23 | Viability check | reject a chain with an unproductive cycle, `labour_short`, `storage_gap` or an unsourced good (V1 to V4); `cycle_not_productive` is the cycle gain `g < 1` of L10 at `CostFloor` 10 000, and the warning `cycle_cost_gain` flags `g = CostFloor / 10 000 * s >= 1` at the configured `CostFloor` (`s` the full-cost share of the cycle's own inputs); the warning `cost_floor_at_full_cost` flags the wage loop at `CostFloor` 10 000 or more | VIA 4; CFG 7.6 item 3; FR-VIA-004 to 008; FR-CFG-044, 056 | specified (error for the material cycle, warnings for the cost gain and the wage loop) |
| D24 | Production stop | a recipe is not started when the stock of an output with a positive cost share is at or above `ProduceHaltRatio` of its target, so an overhang cannot grow without bound; no inventory capacity is added (N-002) | AGT 7.1.5; FR-AGT-036, 070 | specified |
| D25 | Decision rates | trade each tick, price and hiring and migration decisions at slower stated rates, as data | WLD (not written) | proposed |
| D26 | Survival rule | work while unfed or below the reserve money | AGT 7.1.3 | specified, but unit-elastic (6.1), so it is a household rule and **not a damper of L14** |
| D27 | Budget stack and saving rule | upkeep first, job inputs second, buffers only above a reserve and under a concave cap | AGT 7.4 | specified |
| D28 | Proportional work share | a cohort of `w` households works `k = ceil(w * s / 10 000)` households, so it moves in steps of `1 / w` as the ratio crosses the band, not all at once | AGT 7.1.2, 9.3 | specified (a form of D09 and D08) |
| D29 | Derived welfare window | the effective window of a role is `max(WelfareWindow, WindowCycles * longest Duration)`, never configured per role, so a healthy long-job producer is not read as starving | WEL 2.3.3 (`WelfareWindow`, `WindowCycles`) | specified (a placeholder default of 2 cycles) |
| D30 | Demand schedule | a want falls with the price of its items to a choke price (the schedule reaches 0) and rises with a fall to a ceiling; the response is through the belief centre, so it is local and lagged. Cost: a want vanishes at a dear price | DEM 4.3 (`Elasticity`, `Ceiling`) | specified (counted by `WantShortfall.Choked`) |
| D31 | Ladder and shared cap | a want is funded only from the part of the saving rule's cap that buffers leave, in rank order, so wants never touch the reserve, upkeep or job inputs. Cost: in a poor household higher wants are never bought | DEM 4.5 (`Rank`; the cap is AGT's) | specified (counted by `WantShortfall`: `Planned` less `Funded`) |
| D32 | Consumption guard | a want never eats into the next tick's upkeep of the same commodity | DEM 4.7 | specified (a rule constant) |
| D33 | Staggered occupation review | an agent reviews its role only every `ReviewTicks` ticks, by agent ID, with no draw (a form of D25); alone it does not remove the cobweb | SPN 6.2 (`ReviewTicks`, default 10) | specified |
| D34 | Cooldown after a role change | `RoleCooldown` ticks (default 30) after entering a role or being created before an agent may change role | SPN 6.2 (`RoleCooldown`) | specified (counted by `EntryLimited.Cooldown`) |
| D35 | Graded exit | the privation count, the cohort step `DeathStep` and the regional `ExitShare` make a leaving of households gradual, so a role does not empty in one tick. Cost: long hunger before the rungs apply | SPN 5 (`ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`) | specified (counted by `EntryLimited.ExitCap`) |
| D36 | Entry gate | a newcomer enters only a viable role (`EntryRatio`, default 8 000 basis points); a replacement goes to the leaver's role first | SPN 6.1, 7.2 (`EntryRatio`) | specified (counted by `EntryLimited.NoViableRole`) |
| D37 | Funding limit | entrants are funded from at most `EntryPoolShare` of the pool per tick, in whole households, so relief keeps a share. Cost: some entrants are not created | SPN 7.3 (`EntryPoolShare`, default 5 000) | specified (counted by `EntryLimited.Unfunded`) |
| D38 | The pipeline view | `Inbound` lets a consignee or shipper count goods on the way; its cost is the field in every view | TRN 9.1 | specified (the field); the use is AGT's (`spec-agents.md` open question 20) |
| D39 | The dispatch cap | a plan carries at most `MaxDispatchPerTick` requests and a traveller makes one trip at a time; its cost is that a large shipment must be one request | TRN 5.4, 6.2 | specified (a bound) |
| D40 | Freight as a floor on the gap | a good is shipped only if the gap exceeds the haul price, so shipping closes the gap to the freight and no further; its cost is a persistent gap | TRN 3.3.3, 8 | specified (emergent from the market price of haul units) |
| D41 | Cumulative limit | `Limit` and the effective `MaxFlowTotal`: a flow that reaches it ends and says so | MON 4.4.2 [MON-D1] | specified (counted by `MoneyFlowEnded`, `MoneyFlowLimited`) |
| D42 | Holding charge | a rate levy on the balance above an exemption, a balancing sink with a derived bound `m* = E + g/rho` | MON 6.3, 9.3 [MON-D2] | specified; the bound is `model` for hoarders |
| D43 | Exemption on a levy | the levy never takes the balance below `w * Exempt`; warning `levy_below_upkeep` | MON 5.3, 9.4 [MON-D3] | specified |
| D44 | A closed set of shapes | no faucet reads an aggregate or grows with a balance (a rule constant: the shape set) | MON 2, 9.5 [MON-D4] | specified |
| D45 | Payment spread by a carry | the carry starts from the account ID: no lump, no pulse of every recipient at one tick, no draw | MON 6.1, 6.2 [MON-D5] | specified |
| D46 | Bounded gain of the population loop | the slopes of the two schedules set the gain; the warning `population_gain` fires when the half-window alone makes gain times delay exceed 1/2 | POP 3.5, 5.4 [POP-D1] | specified (a warning; the economy's lag is not known to it) |
| D47 | Disjoint rules, fed against unfed | the schedule acts only on fed households and the failure ladder only on unfed ones, so no household is counted twice and the positive check is graded | POP 3.7, 4.5 [POP-D2] | specified (a rule constant; counted by `Unfed` and `NoRatio` of `PopulationChanged`) |
| D48 | No renewal | `Renewal` is 0 while the population module is on, so the ladder's removals are not refilled | POP 4.6 [POP-D3] | specified (a rule constant; a non-default value is warned) |
| D49 | Floor of households | `MinHouseholds` against the schedule's deaths. Cost: a small group never dies out by the schedule | POP 4.4.1 [POP-D4] | specified (counted by `DeathsHeld`) |
| D50 | Cap of households | `HouseholdCap`. Cost: births dropped at the cap | POP 4.4.2 [POP-D5] | specified (counted by `BirthsCapped`) |
| D51 | Funding limit by the stake | births only as the pool pays, `BirthStake`. Cost: the money-limited population `M / stake` | POP 4.3, 5.5 [POP-D6] | specified (counted by `BirthsUnfunded`) |
| D52 | Death in proportion | deaths fall on each agent in proportion to its own rate, by a per-agent carry with a spread start: neither synchronous nor concentrated on the poor | POP 3.8 [POP-D7] | specified (model 2 of `spec-population.md` 13.2) |

### 5.1 The rule for adding a damper

A damper may be added, changed or removed only if the change meets all of these.
Each can be checked by reading the change.

5.1.1. **It is named in the register.** The change adds or edits the `STB-Dnn`
row and lists the `STB-Lnn` entries it affects. A damper in no entry is a number
in a rule that nobody can explain.

5.1.2. **It is a parameter or a rule constant with a home.** A damper whose value
a user could reasonably vary is a configuration field with a default, an owner, a
bound and a doc line (architecture.md 9.5 to 9.7), and a row in the
parameter-owner table. A fixed one is a rule constant beside its code (9.6).

5.1.3. **It acts locally.** A damper uses information the agent or the market has.
It does not read a macro aggregate and feed it back (roadmap.md "Macro is measured,
never set"; ADR 0001). A rule that reacts to measures is a policy actor (PLC), not
a damper of the core.

5.1.4. **It is bounded and deterministic.** It is integer arithmetic, uses the
world's random source only through a documented draw, and has a bound on its own
effect (architecture.md 3.5, 7.3).

5.1.5. **It is observable.** If it clamps, refuses, floors or limits, there is a
counter in an event, or a `Warn` with a volume bounded by the configuration
(observability-and-logging.md 3.2.2), so that a market pinned by a damper shows.
`BeliefsUpdated` (`AtFloor`, `AtCeiling`, `Clamped`) is the model.

5.1.6. **It has a test.** The damper's own unit test, and the scenario of the entry
it serves (FR-STB-005).

5.1.7. **It states what it costs.** A damper trades one thing for another (a hard
budget gives a refused order; a bound gives a pinned market). The entry names the
cost.

5.1.8. **A removal is a behaviour change.** Removing or weakening a damper changes
results and the golden file, and the PR says which entry is now undamped
(architecture.md 9.8; testing.md 7.2).

## 6. Numeric analyses

Three loops were analysed by running small stylised models in Docker, as the user's
instructions require for Python. They are **models of the rules of the specs, not
of code**. Floating point is used inside the models where a curve needs a power; the
rules themselves are integer. They use placeholder parameters (N-015). They show
the qualitative structure and sensitivity of a loop; they are not tuned and do
not support a claim that any default is stable (N-027). The scripts were written in
`doc/specs/models/stb/` (`a_labour.py`, `a2.py`, `a3.py`, `b_bel.py`, `b1.py`,
`b3.py` to `b6.py`, `c_cobweb.py`, `*.out`); open question 1 asks where they should
live.

### 6.1 The target-income labour loop (STB-L14, N-028)

**Model.** `N` = 1 000 identical self-employed households, each producing
`A` = 10 units per working day. The rule of `spec-welfare-and-wages.md` section 5
with `K` as the income target per year in money: days worked
`D = clamp(ceil(K / (A * p)), DMIN, DMAX)`, so supply `S = N * A * D`. `DMAX` = 365,
`DMIN` = 1. `K` = 1 500 000 and a reference price `P0` = 1 000 give `D0` = 150 days
at the equilibrium, as in a role with `ExpectedWorkTicks` 150. Demand is
`Q = S0 * (P0 / p)^e` with a constant elasticity `e` and `S0` the supply at `P0`,
so `P0` is an equilibrium. The price moves as the belief rule does: with
probability one half (the gate) by 2% of itself (`BeliefStep` 200) in the direction
of excess demand, clamped to `[100, 10 000]` (a tenth to ten times the reference
price, the band of `RefBandLow` and `RefBandHigh`). The run is 400 ticks from a
perturbation of the price, 30 seeds.

**The claim `D = K / W`.** Confirmed by construction: the rule gives
`D * W` = `K` where it is not capped, so labour supply `N * K / W` is unit-elastic
in the wage, and for a self-employed role supply of the good `N * A * D = N * K / p`
is unit-elastic in the good's price. Rounding up (`ceil`) adds at most one tick of
supply per household, about 0.7% at 150 days.

**The elasticity condition.** Excess demand `Q_d - S` has slope
`(1 - e) * Q / p` at the equilibrium (`Q = N K / p`), which is the
`(N K / p^2)(1 - e)` of N-028. A numerical derivative agrees with `(1 - e)` to four
places at each tested `e`:

| `e` | 0.3 | 0.5 | 0.9 | 1.0 | 1.1 | 1.5 | 2.0 |
|-----|-----|-----|-----|-----|-----|-----|-----|
| slope of excess demand (normalised) | +0.70 | +0.50 | +0.10 | 0.00 | -0.10 | -0.50 | -1.00 |
| price tatonnement | unstable | unstable | unstable | neutral | stable | stable | stable |

A positive slope means a price rise creates more excess demand, so the price keeps
rising: the equilibrium is unstable when `e < 1` (the staple case) and stable when
`e > 1`. At `e` = 1 exactly, which is a buyer with a fixed budget, expenditure
`p * Q` is constant and the loop is neutral: the sign is decided by what the model
leaves out (income effects). **The condition for stability is demand elasticity
above 1 at the equilibrium.**

**Results of the discrete runs** (30 seeds each; the price after 400 ticks):

| `e` | Case | Result |
|-----|------|--------|
| 0.5 | +10% start | every seed ends at 10 000, the price ceiling |
| 0.5 | -10% start | every seed ends at 167 to 170, where all households work the full year: `P0 * (150/365)^(1/e)` = 169 |
| 0.5 | +10% start, with an inventory buffer, or with `TargetSpread` 3 000 | every seed still ends at the ceiling: neither removes the instability |
| 1.5 | +10% start, no buffer | every seed ends at 977 to 996 |
| 1.5 | spread 3 000 | 977 to 1 032 |
| 1.5 | inventory-ratio rule alone | no convergence; a sustained oscillation (see below) |

For `e` = 0.5 the equilibrium is a repeller and the basin boundary is at the
equilibrium (starts at 900 to 995 go to the low outcome, 1 020 and above go to the
ceiling). A start of 1 005 goes down, because the rounding up of days makes supply
flat for a one-day change; this is an artefact of the rule that shifts the boundary
by about one day's worth (1%). The recovery rate is set by the step, not by `e`:
for `e` between 1.05 and 3 a +20% shock closes in about 20 ticks at a mean
movement of 1% a tick.

**The two outcomes of an unstable role.** The loop does not blow up; the two bounds
of the rule catch it. At the top, the price bound (here 10 000) holds a shortage
that the price cannot clear. At the bottom, the cap of a full year of work holds a
glut of cheap output that the households still produce. Both are the "pinned"
regime of section 7.

**A work floor.** Add a minimum number of working days `DMIN` (D13). Supply is
then flat above `p_f = K / (A * DMIN)`, so excess demand is `Q_d(p) - N A DMIN`,
which is negative at high prices and gives a stable upper outcome at
`p = P0 * (S0 / (N A DMIN))^(1/e)`. The runs agree with the formula:

| `DMIN` (days) | 100 | 120 | 140 |
|---------------|-----|-----|-----|
| upper outcome, formula | 2 250 | 1 562 | 1 148 |
| upper outcome, run (30 seeds) | 2 250 | 1 561 to 1 592 | 1 127 to 1 149 |

The low outcome and the unstable interior are unchanged. So a work floor does not
remove the instability; it replaces the ceiling as the stop with a price set by a
parameter, below which supply cannot fall. A floor of two thirds of the expected
work (`DMIN` 100 of 150) holds the staple at 2.25 times its reference price in
this model. **WEL has adopted it** as the role parameter `MinWorkShare`
(`spec-welfare-and-wages.md` 5.3): a share of the window's ticks, in basis points,
default 4 000 (a placeholder), validated to be at most `ExpectedWorkTicks` of the
year. In the model's terms `DMIN = MinWorkShare * 365 / 10 000`, so the upper
outcome is `P0 * (D0 / DMIN)^(1/e)` where `D0` is the days worked at the equilibrium:
with `D0` = 150 and the default floor (146 days) it is 1.06 times the reference price,
and with the 250 days of WEL's example it is 2.9 times (WEL 5.3 quotes the second).
If the floor lies at or above `D0`, supply is flat around the equilibrium and the
target-income rule is moot there (hand analysis; not run); validation keeps the floor
at or below `ExpectedWorkTicks`, not below the days that emerge. The runs above used
a fixed number of days; they have not been re-run with the share form, which differs
only in the unit (days of a year against a share of a window). The survival floor
discussed in N-028 (work while the household's upkeep is unmet) is a different rule,
worth `K_subsistence / (A * p)` days, which is also unit-elastic and so is not a
damper by itself.

**The inventory buffer as an integrator.** With a rule that moves the price only on
the stock ratio (`BeliefStockLow` 5 000, `BeliefStockHigh` 15 000 of a target of
10% to 300% of one tick's supply), `e` = 1.5 does not converge: the relative
standard deviation of the price over the last 500 ticks is 6% to 29% in a grid of
24 settings, against 1% with the excess-demand rule. The amplitude scales with the
step: 6% to 12% at `BeliefStep` 50, 9% to 29% at 200. The cause is that the stock is
the integral of the imbalance, so the rule responds late (L04). The evidence rule
is the proportional signal that `spec-beliefs.md` adds; their combination is
analysed in 6.2.

**What this shows.** (1) N-028's hand analysis is right and quantitatively exact in
the model: the sign of the slope is `1 - e`, the corner prices follow the formulas.
(2) For a staple (`e < 1`) the interior equilibrium is unstable under the rule as
written, and the design's only dampers are bounds: the price ceiling above, the
full year of work below. Neither the buffer nor the spread of targets removes it.
(3) The failure is a regime (a pinned market), not a numerical blow-up.

**What it does not show.** Real household demand is not a fixed-elasticity curve:
income feeds back into demand. `spec-agents.md` 11.6 has since fixed a staple's
elasticity at 0 to 1 (never above 1), which is the unstable case of this model, but
the model does not run AGT's rules (the budget stack, the rest band, the proportional
share). The model has
no wage labour (only self-employed output), no migration and no relief. The price
rule is a one-signal caricature of `spec-beliefs.md`. It does not show that any
real pack is unstable; it shows the condition under which one is not.

**Decision for N-028.** Keep the rule (the backward bend is evidenced for
labourers, `evidence.md`). The damper is a condition and a test: S04 injects a
shortage into a self-employed staple role and asserts that the price stays within
its bounds and that the role ends in a stated condition, and a pack author checks
the staple's demand elasticity. The work floor D13 was recommended as an option and
WEL adopted it as `MinWorkShare` (5.3 there); it is a stop, not a cure.

### 6.2 The belief, price and inventory loop (STB-L01 to L09)

**Model.** One commodity in one region: `NS` seller agents and `NB` buyer agents,
default 20 each, weight `w` (default 1). Beliefs follow `spec-beliefs.md` sections
3 to 5 exactly with the 15 placeholder parameters of section 11 (`BeliefStep` 200,
`BeliefAdjustProb` 5 000, `BeliefSignificance` 500, `BeliefStockLow` 5 000,
`BeliefStockHigh` 15 000, `BeliefMaxAge` 3, `BeliefWidthInit` 2 000,
`BeliefWidthMin` 200, `BeliefWidthMax` 5 000, `BeliefWidthStep` 250, `CostFloor`
10 000, `CostCeiling` 30 000, `RefBandLow` 1 000, `RefBandHigh` 100 000) with
reference price 1 000. Clearing is the average clearing of `spec-clearing.md`
(greedy best-to-best, crossing check, midpoint half to even, volume-weighted
market price, price age, unmatched quotes). Sellers produce `Q` units per
household per tick and hold the stock; they ask all of it. Buyers consume `D` per
household per tick, hold a target of three ticks of consumption, and bid for the
shortfall. The seller's target is two ticks of production. Money is not modelled:
a buyer's willingness to pay is an optional cap standing in for a budget. In the
elastic runs production is `Q * (p/Pref)^es` and consumption `D * (Pref/p)^ed` on
the last market price, with no delay. Randomness is Python's, used only as the
model's source of uint64 draws. 8 seeds unless stated; statistics over the second
half of the run.

**Run 1: price-inelastic flows (`Q = D`, no response to price).** Without a cost,
the market price falls to the floor of the reference band: 107 to 110 against a
floor of 100 and a reference of 1 000, in every seed, with the sellers holding a
growing overhang (1 890 units after 600 ticks against a production of 200 a tick)
while the buyers hold about 60% of their target stock and miss about 2% of their
consumption. A
cost of 700 stops the fall at the cost floor: 745 to 751. With a stop on
production above a stated share of the target (D24), the outcome depends on where
the stop is: at 300% the price still falls to 128 to 138, at 150% it rises to 3 207
to 6 367 with about 13% of need unmet, and at 100% it reaches 7 845 to 8 006. The price
goes to a bound; which one is decided by the sign of the accumulated imbalance, and
a small change in a threshold flips it (L05, L06). The first fall begins in the
early ticks, when only about half of the volume trades because the limits overlap
partly, and the unsold half accumulates at the sellers.

**Run 1b: the halt with the agents' own target** (a fresh run after
`spec-agents.md` was written). Run 1 used a seller target of two ticks of
production; `spec-agents.md` 7.3 sets the target of a good that is only made at
`OutputCoverTicks` = 1 tick of output, and 7.1.5 halts a recipe at `ProduceHaltRatio`
of it (default 30 000). With that target (same model, `Q = D` = 10, 8 seeds, ticks 400
to 800):

| Halt at (basis points of the target) | none | 30 000 (default) | 20 000 | 15 000 | 10 000 |
|--------------------------------------|------|------------------|--------|--------|--------|
| mean price (median; range) | 107 (107 to 108) | 1 934 (1 548 to 3 144) | 7 927 (7 897 to 8 049) | 7 927 | 8 773 |
| relative std of the price | 2.0% | 22.9% | 3.3% | 3.3% | 2.5% |
| unmet need per tick (of 200) | 3.4 | 24.6 | 50.0 | 50.0 | 100.0 |

With the default halt the price no longer falls to the floor: it sits at 1.5 to 3.1
times the reference, swings by about a fifth, and 12% of the need goes unmet. The
halt turns the overhang ratchet into a shortage: when the sellers' stock reaches the
halt they stop producing, the fixed demand then exceeds supply, and the price rises
until the stock falls and production restarts. Tighter halts pin the price near the
ceiling and leave a quarter to a half of the need unmet. So in this model the halt
removes the floor pin and replaces it by a shortage cycle whose size depends on the
threshold; it is not a cure for inelastic flows (L44, L51). Run again with the
revised evidence rule of BEL 5.4 (round 3), the same default halt gives a mean price
of 4 023 (2 964 to 4 763), a relative standard deviation of 26% and 11% of need
unmet, and no halt gives 132 (126 to 137): the shape is the same and the size of the
shortage cycle depends on the evidence rule as well as on the threshold.

**Run 2: price-responsive flows (`ed` = `es` = 0.5, `Q = D = 100` per household).**
The market price settles in an interior band, but not at the price where supply
equals demand (1 000):

| Variant | Mean price (median of 8 seeds, range) | Relative std | Unmet need per tick (of 2 000) |
|---------|----------------------------------------|--------------|--------------------------------|
| as written | 888 (883 to 894) | 4.8% | 242 |
| stock component off | 870 (863 to 877) | 5.1% | 284 |
| evidence rule off (stock only) | 1 001 (1 000 to 1 002) | 4.8% | 0 |
| direction-restricted evidence | 907 (903 to 911) | 4.4% | 197 |
| fixed width | 909 (902 to 913) | 3.0% | 191 |
| `BeliefSignificance` 0 | 884 (876 to 887) | 4.7% | 253 |
| elasticities 0.3 / 0.3 | 824 (791 to 829) | 4.9% | 238 |
| elasticities 1.0 / 1.0 | 942 (937 to 944) | 4.5% | 247 |
| elasticities 0.5 / 1.5 | 941 (935 to 944) | 4.6% | 239 |

The evidence rule of `spec-beliefs.md` 5.4 produces the offset: with it off, the
price is at the clearing price (1 001) and nothing is unmet. A trace of the direction
of the moves, with the stock component off, shows that unfilled bids on balance
move down (128 of 160 moves) and unfilled asks move down (364 of 399): an unfilled
order's centre moves toward the other side's best unmatched quote, which can lie on
the wrong side of the centre when the limit was drawn far from it, so a failing bid
can lower its own centre. Restricting the sign of the evidence move to the direction
of the failure (the "direction-restricted" row) removes only part of the offset
(907). This is a finding about the model of the rule as written, and N-042 passes it
to BEL; a different reading of 5.4 may not show it.

**Run 2b: the revised evidence rule** (a fresh run in round 3, after BEL revised 5.4
and added `BeliefEvidenceShare`). The model's update function was changed to the
revised rule as written (the target compared to the limit; no move when the limit and
the centre lie on different sides of the target; a step of at most `Δe`), everything
else as in run 2:

| Variant | Mean price (median of 8 seeds, range) | Relative std | Unmet need per tick (of 2 000) |
|---------|----------------------------------------|--------------|--------------------------------|
| first draft (run 2, "as written") | 888 (883 to 894) | 4.8% | 242 |
| **revised, `BeliefEvidenceShare` 5 000 (the default)** | **944 (940 to 949)** | **3.6%** | **118** |
| revised, share 10 000 | 903 (895 to 906) | 4.5% | 218 |
| revised, share 2 500 | 966 (964 to 970) | 3.6% | 73 |
| evidence off (stock only) | 1 001 (1 000 to 1 002) | 4.8% | 0 |

This reproduces BEL's own table for the default (944, 118 unmet; BEL ran a copy of
the same model). The offset falls from 11.2% to 5.6% and the unmet need from 12% to
5.9%; a smaller share removes more of it. A shock of consumption up 50% for 100 ticks
(6 seeds as in run 3) peaks at 1.56 times the pre-shock mean and returns in a median
30.5 ticks (first draft: 1.69 and 23.5; evidence off: 1.62 and 99), so the evidence
rule is what makes the recovery fast and it is still the source of a smaller offset.
**The offset is reduced, not removed**, and the cost of the revision is a slower
recovery (30.5 ticks against 23.5).

**Run 3: a shock** (`ed` = `es` = 0.5; consumption up 50% for ticks 400 to 500; 6
seeds). The price rises to 1.69 times the pre-shock mean at its peak and returns
within 10% of it a median 23.5 ticks after the shock ends (as written); with the
evidence rule off the peak is 1.61 and the return takes 98 ticks. Sweeping the gate
trades speed for overshoot: `BeliefAdjustProb` 1 000 gives a peak of 1.37 and a
return of 82 ticks; 10 000 gives 1.74 and 15.5 ticks. Sweeping `BeliefStep` from 50
to 1 000 gives peaks of 1.54 to 1.78 and returns of 24 to 42 ticks with no clear
trend at 6 seeds (noise).

**Run 4: cohorts** (20 households per side, fixed; `w` = 1, 2, 4, 10 gives 20, 10,
5, 2 agents per side; `K` is not applied):

| `w` | 1 | 2 | 4 | 10 |
|-----|---|---|---|----|
| relative std of the price, before the shock | 4.7% | 5.5% | 6.9% | 9.1% |
| peak, relative to the pre-shock mean | 1.69 | 1.72 | 1.78 | 1.83 |
| return after the shock (ticks, median) | 23.5 | 63 | 103 | 169 |

One gate draw per cohort multiplies the lockstep (L13, L34). The floor `K` of
`spec-scale.md` 5.2 would keep the agents at no fewer than 10 per group here
(`w` of at most 2 for 20 households). The pre-shock level also shifts from 883 to
974 as `w` rises, so the return times are indicative only. With the revised
evidence rule the noise before the shock is 3.6%, 7.0% and 9.7% at `w` = 1, 4 and 10
(fresh run, 6 seeds), so the cohort effect is unchanged.

**Run 5: the hard floor with a perishable** (STB-L09, N-039). Sellers with unit
cost 1 100, buyers capped at 1 000 (a budget stand-in), a shelf life of 3 ticks,
`Q` = `D` = 100 per household, ticks 100 to 300:

| `CostFloor` | Floor | Ticks with a trade | Share of output spoiled | Mean price |
|-------------|-------|--------------------|-------------------------|------------|
| 10 000 | 1 100 | 0% | 99% | 1 000 |
| 9 000 | 990 | 7% | 98% | 995 |
| 8 000 | 880 | 97% | 69% | 941 |
| 7 000 (the new default of `spec-beliefs.md` 3.2.2; revised evidence rule) | 770 | 100% | 45% | 895 |
| 5 000 | 550 | 100% | 21% | 780 |

At a floor above what buyers can pay the market freezes, and a perishable rots
(`spec-beliefs.md` open question 1, option (a) with no release). The spoilage share
stays high even when trades resume because the model's flows are not matched to the
buyers' appetite. The rows for 10 000, 9 000, 8 000 and 5 000 were run with the first
draft of the evidence rule; a re-run of 10 000, 9 000, 7 000 and 5 000 with the
revised rule gave 0% and 99%, 7% and 98%, 100% and 45%, and 100% and 29% (the 5 000
row is 29% spoiled, not 21%, with the revised rule). The unit cost of 1 100 stands
for the full cost with labour since round 3, so a floor of 7 000 is 770, below what
buyers can pay (1 000), and the market trades.

**What this shows.** (1) The belief rules do not by themselves fix the level of
the price: with inelastic flows there is no interior anchor, and the price goes to
a bound whose choice depends on the sign of an imbalance. A price-responsive flow
(a budget, a production choice, exit) is what anchors it, and AGT, PRO and SPN own
those (N-043). (2) With responsive flows the rule settles near, but below, the
clearing price in the model, and the evidence rule is the source (N-042); BEL's
revision (the limit, the guard, the share) halves the offset to 5.6% and does not
remove it (run 2b). (3) The
stock rule alone converges to the clearing price but oscillates with the
integrator lag (6.1); the combination is stable here. (4) The gate and step trade
speed for overshoot. (5) A cohort multiplies noise and slows recovery. (6) A hard
cost floor freezes a market whose buyers cannot pay it.

**What it does not show.** There is no money loop (a buyer's income is not the
seller's spending), no production delay, no failure ladder, no price window, no
seeded beliefs, one commodity and one region, 20 agents a side. The "evidence
offset" may depend on the model's quantity rules (a bid for the shortfall, an ask
for the whole stock), and on only one quantity-sizing rule. The results are for the
placeholder parameters. They do not show that the defaults of `spec-beliefs.md`
section 11 are good or bad.

### 6.3 The production-delay cobweb and the occupation cobweb (STB-L21, L36)

**Model 1: the production-delay cobweb.** Demand is `p = Q^(-1/ed)` with `ed` =
0.5, normalised so the equilibrium is `p` = 1, `Q` = 1. Producers start production
at tick `t` at a rate `(expected price)^es`, and it arrives `d` ticks later. The
price is bounded to `[0.1, 10]` (a tenth to ten times the reference, like the
reference band). The "amplitude" is the maximum over the minimum price in the last
200 of 600 ticks after a 10% initial price perturbation. A value of 100 means the
price runs between the two bounds.

*Naive expectations* (expected price is the last price):

| `es` | 0.25 | 0.5 | 0.75 | 1.0 | 2.0 |
|------|------|-----|------|-----|-----|
| amplitude, `d` = 1, 5 and 20 (identical) | 1.00 | 1.21 | 100 | 100 | 100 |

Convergence holds for `es` < `ed`, and the case `es` = `ed` is a neutral cycle that
neither grows nor decays (1.21). Above it the price saturates the bounds. The delay
does not change the threshold: with a delay `d` the process is `d` interleaved
independent copies of the one-tick chain. It sets the period (`2d` ticks) and the
time the cycle takes to show.

*Averaging expectations* (Nerlove: the expected price moves a fraction `λ` of the
way to the last price), `es` = 2.0, which diverges with naive expectations:

| `λ` | 0.5 | 0.2 | 0.05 | 0.02 |
|-----|-----|-----|------|------|
| `d` = 1 | 100 | 1.00 | 1.00 | 1.00 |
| `d` = 5 | 100 | 100 | 1.00 | 1.00 |
| `d` = 20 | 100 | 100 | 100 | 1.14 |

The maximum `λ` that converges falls roughly as one over the delay (0.2 at `d` = 1,
between 0.2 and 0.05 at 5, about 0.02 at 20): a long delay needs slow expectations.

*Bounded asynchronous expectations* (the belief rule: with probability one half the
expectation moves by at most `step` of itself toward the price), `es` = 2.0, median
of 10 seeds:

| `step` | 5% | 2% | 0.5% |
|--------|----|----|------|
| `d` = 1 | 1.87 | 1.29 | 1.06 |
| `d` = 5 | 4.87 | 1.83 | 1.16 |
| `d` = 20 | 52.1 | 5.56 | 1.51 |

The bounded step does not remove the instability; it caps the swing. The amplitude
depends on the step and on the delay, so a long delay with a step of a few percent
cycles over most of the band.

*An entry limit* (the rate of starts changes by at most `slew` a tick), naive
expectations, `es` = 2.0:

| `slew` | none | 20% | 5% | 1% | 0.2% |
|--------|------|-----|----|----|------|
| `d` = 1 | 100 | 3.41 | 1.38 | 1.06 | 1.01 |
| `d` = 5 | 100 | 40.7 | 2.55 | 1.20 | 1.04 |
| `d` = 20 | 100 | 100 | 27.3 | 1.95 | 1.14 |

Both together (step 2%, slew 1%): 1.20 at `d` = 1, 1.40 at 5, 2.43 at 20.

**Model 2: the occupation-choice cobweb.** A share `x` of a population is in a
role whose income per head is `p = (x / 0.5)^(-1/e)`; the alternative pays a
constant 1, so `x` = 0.5 is the balance. Each tick a fraction `r` of agents
re-decides, comparing a moving average over `T` ticks of the role's past income with
1: all re-deciders enter if the average exceeds `1 + h`, leave if below `1 - h`,
stay between (`h` is a hysteresis band). Mean-field and deterministic, 3 000 ticks.
Amplitude of `x` over the last 1 000 ticks:

| `e` | `T` | `r` | `h` = 0 | `h` = 0.1 |
|-----|-----|-----|---------|-----------|
| 0.5 | 1 | 0.2 / 0.05 / 0.01 | 0.111 / 0.026 / 0.005 | 0.111 / 0 / 0 |
| 0.5 | 30 | 0.2 | 0.577 | 0.831 |
| 0.5 | 30 | 0.05 | 0.547 | 0 |
| 0.5 | 30 | 0.01 | 0.147 | 0 |
| 2.0 | 1 | 0.2 / 0.05 / 0.01 | 0.111 / 0.026 / 0.005 | 0 |
| 2.0 | 30 | 0.2 | 0.929 | 0 |
| 2.0 | 30 | 0.05 | 0.545 | 0 |
| 2.0 | 30 | 0.01 | 0.147 | 0 |

With a window of 30 ticks (the default `T`), re-decision of a fifth of the agents
each tick swings the role between 7% and 100% of the population. Reducing the rate
to 1% of the agents a tick brings the swing to 15 points, and a hysteresis band of
10% removes it in this model (except when both a large rate and a large window
meet an inelastic role, where the band lets the share run farther before it
switches). Where the band removes the swing it leaves the share at an offset from
the balance (0.50 to 0.55 in the runs), the price of not switching inside the band.
With `T` = 1 the swing is just the bang-bang step of size `r / 2`.

**What this shows.** (1) The cobweb threshold is the elasticity condition of the
labour loop, in another form: supply must be less price-responsive than demand
(`es < ed`), or the expectation slow enough, and the price bounds are the only stop
otherwise. (2) The delay moves the period; it multiplies the amplitude of
bounded-step expectation. (3) Entry limits and hysteresis are the effective dampers
for the occupation choice. Both have since been specified by the spawner (D09 as
`RoleHysteresis`, D14 as `EntryShare`, with D33 to D35), and SPN's model with discrete
agents (`spec-spawner.md` 16.1, section 6.5 below) agrees that the entry limit is the
damper that works and a hysteresis band alone is not enough.

**What it does not show.** The occupation model is mean-field and bang-bang, has no
cost of switching, no tool, no queue and no noise. The production model has no
inventory, which the real design has and which delays the price effect (it can
also stabilise or destabilise, 6.1). The parameters are not those of any pack.
`es` is the elasticity of the starts to the expected price, which the real design
does not state anywhere: it will come from AGT's choice of recipe.

### 6.4 Summary of what was run

| Entry | Model | Headline |
|-------|-------|----------|
| L14 | 6.1 | stable only if `e > 1`; otherwise two pinned outcomes; a work floor moves the upper one |
| L04 | 6.1, 6.2 | the stock signal alone oscillates (6% to 29%); it converges when combined |
| L01, L02 | 6.2 | the level is anchored only by the stock rule; the evidence rule offsets it by 11% (first draft) or about 5.6% (revised, run 2b) |
| L05, L06 | 6.2 | inelastic flows: no interior; the price goes to a bound |
| L44 | 6.2 run 1b | the default halt replaces the floor pin by a shortage cycle (price 1.5 to 3.1 times the reference, 12% unmet) |
| L09 | 6.2 | a hard floor above the buyers' budget freezes the market and spoils 99% |
| L13, L34 | 6.2 | a cohort of weight 10 doubles the noise and slows recovery |
| L21 | 6.3 | cobweb threshold `es` < `ed`; slow or limited expectations needed for long delays |
| L36 | 6.3 | window 30 and a fifth re-deciding swings the role by up to the whole population |
| L14 (luxury) | DEM 6.3, 14.1 (not run by this spec) | stable only with `Elasticity` above 10 000 and about 10% budget headroom |
| L46, L57 | DEM 7.2, 14.2 (not run by this spec) | closed economy ratio 1.000 with upkeep only, 1.53 with wants at the target 15 000 |
| L36, L35, L61 | SPN 16.1, 16.2 (not run by this spec) | the entry share removes the occupation swing, rotation and hysteresis do not; graded exit bounds the exit in a tick |

### 6.5 Models run by the demand and spawner specs (not re-run here)

Section 6 so far is this spec's own work. `spec-demand.md` 14 and `spec-spawner.md` 16
each ran small stylised models in Docker (scripts outside the repository, N-068) and
the register quotes their headline results, **without having re-run them**. The status
of the entries that rest on them is `model` in the sense of "a model exists", not "this
register reproduced it". They carry the same caveats as section 6 and nothing in them
supports a claim of stability (N-027).

- **DEM, the luxury labour loop (14.1).** The setting of 6.1 (1 000 producers, 150 days
  at the reference, a work floor at 146 days, 30 seeds) with a schedule elasticity in
  place of a constant `e`. Stable only with `Elasticity` above 10 000 **and** budget
  headroom of about 10% over the spending at the reference price; at exactly 10 000 or
  with a fixed budget it is neutral and falls to the band floor; the lower edge of the
  basin is between 60% and 70% of the reference price. It has no income feedback from
  the luxury role's customers. Entry L14.
- **DEM, the closed economy (14.2).** 400 households in two roles: mean welfare ratio
  1.000 with upkeep only and 1.53 with wants at the target 15 000, bounded by
  `1 + Wmax / U`; money below the sum of the reserves buys no want (ratio 0.5 at 450 per
  household), and the threshold lies between 900 and 1 800 per household. A first model
  (`b_closed.py`) was dropped (DEM 14.2). Entries L45, L46, L57.
- **SPN, the occupation cobweb with discrete agents (16.1).** 1 000 weight-1 agents in
  three roles under the rules of SPN 6. Rotation and a hysteresis band do not remove the
  swing; the entry share does (amplitude 0.00 at 50 basis points, window 30; 0.35 at 50
  with window 120); the pairs that settle have `EntryShare * T` below about 2 400. It has
  no switching cost beyond the cooldown, no tool or queue, and a demand curve per role
  instead of a market. Entries L36, L54, L61.
- **SPN, the ladder, exit and funded entry (16.2).** 300 households, two roles, an income
  shock to 35% or 15%. The "bazaar" policy removes the failing role in a tick; the
  ladder replaces deaths with role changes and hunger (36 094 unfed household-ticks
  against 316 for the bazaar policy at 35%, with no deaths), and an exit cap of 1% to 2% bounds the exit in a tick to 2
  to 6. No goods, no market clearing, no relief, incomes set by a formula. Entries L35,
  L61, L62.

## 7. Failure regimes

A **failure regime** is a way a run goes wrong that a stability test must be able to
recognise. Each is defined by a measure that can be computed from events and the
snapshot alone (FR-STB-012), so that `spec-indicators.md` and
`spec-stylised-facts.md` can implement it. Thresholds are placeholders that STY
sets.

| ID | Regime | Definition (placeholder thresholds) | Entries that can cause it |
|----|--------|--------------------------------------|---------------------------|
| STB-F1 | Runaway inflation | the price index of a commodity class rises by more than a stated share a window and keeps rising for more than a stated number of windows; money in the economy exceeds its start by a stated factor | L05, L10, L30, L14 (upper outcome), L69 |
| STB-F2 | Hoarding | money or goods held by a sector or by the pool rise above a stated share of the total for a stated time, and trade volume falls | L31, L11, L25, L45, L30, L73 |
| STB-F3 | Collapse | households fall more than a stated share in a window, or a role with a positive start disappears, with no recovery within a stated horizon | L39, L35, L19, L11, L62, L63, L64, L66, L77, L82 |
| STB-F4 | Stagnation | trade volume stays below a stated share of its start for a stated time with no collapse: markets open, nothing happens | L03, L09, L11, L45, L46, L73 |
| STB-F5 | Pinned market (frozen at a bound) | more than a stated share of a market's outcomes end on a floor or ceiling for a stated number of ticks (the `Warn` of `spec-beliefs.md` 12.2); or no trade in a market that has unmatched orders on both sides | L05, L06, L09, L14 (a staple, or a luxury below the basin of 6.5), L51, L86 |
| STB-F6 | Cobweb blow-up | the price range (maximum over minimum) in a stated window exceeds a stated factor and does not shrink over successive windows; or a role's share swings by more than a stated fraction | L21, L36, L04, L58, L61, L84, L85 |
| STB-F7 | Synchronisation | the dispersion of belief centres or of targets falls below a stated minimum, or a role works and rests in lockstep | L13, L15, L22, L34, L35, L59 |
| STB-F8 | Offset | a market's long-run price differs from its clearing price by more than a stated band, with unmet need | L02 |

The paper's stability criteria (`spec-beliefs.md` 19: every commodity produced and
consumed, none untraded for more than 20% of rounds, mean volume above one unit,
bounded variance and inflections) are a lead for the first four, not a calibration.
F5 is the regime that the two rules of this spec's analysis keep meeting: in 6.1
and 6.2 the runs end pinned, not oscillating.

## 8. Scenarios

A **scenario** is a configuration and a script of injected shocks that a stability
test runs and asserts on. This list is what `spec-stylised-facts.md` and
`spec-reference-chain.md` must cover. A scenario is written by those specs, with
bands; here it is named, tied to the entries it shows, and described just enough to
be built. All use fixed seeds and report the failing seed (testing.md 6.2).

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| S01 | Shock to one commodity's reference price or a consumption step, one region | the price returns to within a band of its pre-shock level within a stated time; the long-run price is within a band of the clearing price (F8) | L01, L02, L04 |
| S02 | A market with a stale price and a bid and an ask | they cross within `ceil(gap / Δ)` adjustments | L03 |
| S03 | A market pinned at a cost floor | the `Warn` is emitted at the tenth tick and each tenth after, once per market | L03, L09 |
| S04 | A staple role of self-employed households and a carrier role, with a shortage injected | the price stays within its bounds; the role ends in a stated condition, not pinned for longer than a stated time (F5); the staple's measured demand elasticity is recorded | L14, L15, L16, L17, L18, L51, L60 |
| S05 | A delayed supply response with a staple: a long recipe or a long trip | an oscillation decays: the range of each successive window shrinks, or stays within a stated factor (F6) | L21 |
| S06 | An income gap between two roles, with the tool and the traineeship on and off | the share of the role stays within a stated band of its mean (F6) | L36, L37, L48, L54, L65 |
| S07 | A perishable glut with a hard cost floor (N-039), `CostFloor` at the default and at lower values | the share spoiled is below a stated limit, or the seller leaves within a stated time; the market does not stay frozen (F5) | L09, L25 |
| S08 | A cost shock to a raw input passes through a chain | the output prices settle; cycle gain `g < 1` on every cycle | L10, L28 |
| S09 | A market with no sellers | the price reaches the ceiling and no further; the `Warn` is emitted once per ten ticks | L05, L08, L47 |
| S10 | A long run of the reference chain at scales 1, 4 and 10 | the dispersion of belief centres stays above a minimum; the scale-consistency tolerance of `spec-scale.md` 7.2 holds | L13, L33, L34, L59 |
| S11 | A producer with a job longer than the welfare window | a healthy producer is not flagged as starving, and gets no relief or migration trigger by the job alone | L19, L20 |
| S12 | A seasonal run with a delay near the season length, and a coarse tick | the prices stay within bands over several years | L23, L24, L28 |
| S13 | A run with a faucet and no sink (defined by `spec-money-supply.md` 17.6) | the boundary balance of the flow's reason equals `-Total` exactly, `M` equals `M_init` plus it, and the audit holds; the price drift is report-only until STY sets a band | L30, L69 |
| S14 | A region that loses its income, with the pool, relief and the failure ladder on | the region recovers from a stated depth of collapse, or collapses in a stated way and the immigrants restore it (F3) | L11, L31, L35, L38, L39, L50, L62, L63 |
| S16 | Price-inelastic flows with an initial overhang | the price does not end pinned; or the run is reported as F5 and the pack is rejected | L06 |
| S17 | A durable's replacement demand from a uniform starting age | the replacement orders are spread, not one spike | L26, L27 |
| S18 | The warm start of a long-job chain (VIA A8) | the output does not complete in a single tick; completions per tick over the first `d` ticks are within a stated spread of their mean | L22 |
| S19 | The default `TargetRatio` in a closed economy with upkeep as the only demand, run with the halt on and off (`spec-agents.md` 18) | the mean welfare ratio is near 1; the unsold stock is bounded and the halt stops output; with the halt off the glut grows (F2, F4) | L46, L44 |
| S20 | Starting money below `ReserveTicks` times the upkeep cost for every household (`spec-agents.md` 18) | households work and reach the reserve, or the world reports the unreachable reserve | L45, L11 |
| S21 | Seed prices of the reference chain against the cost band | every seed lies inside `[UnitCost * CostFloor, UnitCost * CostCeiling]` at the seed prices, or the solver reports it (FR-VIA-027) | L52, L09 |
| S22 | A perturbed chain started from the warm start and from a cold start | the warm start does not hide an instability: both end in the same regime (VIA A1) | L53, L14, L21 |
| S23 | A self-employed luxury role under the target-income rule with a want of `Elasticity` 5 000, 10 000, 15 000 and 30 000 and budget headroom of 100%, 110% and 160%, perturbed 10% up and 10% down (`spec-demand.md` 13) | the price converges only when both conditions of `spec-demand.md` 6.3 hold (schedule elasticity above 10 000 and headroom of about 10%); the lower edge of the basin is reported; otherwise the run is reported as F5 | L14, L57 |
| S24 | A closed economy with a ladder: the default `TargetRatio` with the wants on and off, and a target above the ceiling (`spec-demand.md` 13) | the mean ratio is near 1 with the wants off, above 1 with them on and at most `1 + Wmax / U`; with a target above the ceiling the glut is bounded by the halt (F2, F4) | L46, L57, L44 |
| S25 | Total money at 0.5, 1 and 2 times the sum of the reserves, and a want of two alternatives with `Substitution` 4 and a price gap (`spec-demand.md` 13) | wants are not bought below the reserves; the shares of the two alternatives do not swing between the items without damping | L45, L58 |
| S26 | A role-income shock with the spawner's caps on and off (`EntryShare` at its default and at 0, `ExitShare` at 0; `spec-spawner.md` 17) | the role's share stays within a stated band, the role changes per tick do not exceed `ceil(Households * EntryShare / 10 000)`, and with the caps off the share swings more, so the measure can see it (F6) | L35, L36, L54, L61 |
| S27 | A region where no role is viable (`spec-spawner.md` 17) | no entrant is made; the `Warn` "role extinct" or the counter `NoViableRole` shows; households fall and the run is reported as F3 | L62, L64 |
| S28 | A realm of three regions in a chain, with a closure of the middle link for a stated number of ticks and then a reopening, on a pack with trade on (`spec-regions.md` 13) | the cut region's households fall no faster than in the closed-region variant (F3) while cut; the price gap between the two ends rises while cut and returns inside a stated band after the reopening; the event `EdgeStatusChanged` is emitted at both changes. Waits for TRN, MER and INT | L66, L62 |
| S29 | A town as one region, and the same town as a town and two hamlets joined by a short edge, with the same total households (`spec-regions.md` 13) | the thin hamlet markets are reported (price age, unmatched units); the dispersion of the three prices stays within a stated band once trade is on; the one-region run is the control. Waits for TRN and MER | L67 |
| S30 | Two regions with a route and a merchant that ships on a price gap, with the shipper counting `Inbound` and not (`spec-transport.md` 15) | the gap closes to within the freight plus a band and does not oscillate with a growing amplitude for `k` in a stated range | L84, L85 |
| S31 | A perishable with `ShelfLife` equal to the travel time and one less (`spec-transport.md` 15) | the first is never delivered, the second always | L86 |
| S32 | A hired mode with a haul recipe whose `Duration` is the round trip, at three elasticities of the carriers' starts (`spec-transport.md` 15) | the price of haul units converges or is reported as F6 | L21 |
| S33 | A rate levy bounds a hoarding class; a pair of a grant and a fixed levy drifts by its difference (`spec-money-supply.md` 16, MON-S1) | the class's money converges below `m*` and never above it; the net of the pair equals the difference of the fixed amounts times the ticks, exactly | L30, L70 |
| S34 | A levy equal to a household's upkeep cost, without and with `Exempt` (`spec-money-supply.md` 16, MON-S2) | with the exemption no household goes unfed because of the levy; without it the warning fires and the unfed households are counted | L71 |
| S35 | Growing households with and without a pool grant (`spec-money-supply.md` 16, MON-S3) | `M_h / H` crosses the thresholds `ReserveTicks * U_c` and `U_c` at the derived ticks without the grant and stays above the spending threshold with it | L73 |
| S36 | The scale seam: `SetFlowScale` between steps (`spec-money-supply.md` 16, MON-S4) | only that flow's amounts change, from the next step; nothing changes without a call; the audit holds | L69 |
| S37 | A region with a fixed factor (RES, or a test double that makes the ratio a function of households), the demand module on, started at 0.6 and at 1.6 times `N*` (`spec-population.md` 14.4, POP-S1) | the population approaches `N*` within a band and, for gain times delay below 0.5, without crossing it more than once; the mean ratio at the end is within a band of `r*` | L75, L78, L80 |
| S38 | A productivity shock that cuts the region's income for a time, then restores it, with the pool, relief and the ladder on, and a chain whose role disappears (POP-S2) | households fall and regrow, the ratio rises while they are low, the extinct-region `Warn` does not fire; with the chain whose role disappears the run is reported as F3 and the trap is visible | L77, L82 |
| S39 | A closed region with the pool empty, `BirthStake` 10 000, 5 000 and 0, and births above deaths (POP-S3) | the population stops at about `M / stake`; `BirthsUnfunded` is non-zero when it does | L76 |
| S40 | The default schedules with the demand module off and `r*` above 10 000 and just below it, and a pack with `Renewal` set to a non-default value above 0 (POP-S4) | the first dies out or is flagged, the second sits at a bounded hunger that the ladder's counters show; `Renewal` is ignored, and warned only at the non-default value | L79, L81 |

S15 is not used and the ID is not reissued. The scenarios S28 and S29 are REG's, S30 to S32
TRN's (proposed as S28 to S30 there), S33 to S36 MON's (MON-S1 to MON-S4) and S37 to S40 POP's
(POP-S1 to POP-S4); S13 existed and is now defined (MON). The scenarios S23 to S25 are DEM's and S26
and S27 are SPN's; the specs proposed them as S23 to S25 and as S23 and S24, which
collided, and the numbers above are final (the spawner's S23 and S24 became S26 and S27).

The reference chain must contain at least one staple role with a demand elasticity
below 1 (S04), one perishable (S07), one chain with a cycle (S08), a long recipe
(S05), a durable (S17), a role with a job longer than the base welfare window (S11),
a warm start (S18, S22), a world with upkeep as the only demand (S19), a self-employed
luxury role, a want with two alternatives and a taste table for two roles (S23 to S25),
and a closed role that only external agents fill, a role-income shock and a region where
no role is viable (S26, S27), or these scenarios cannot be built from it (note N-047).
`spec-reference-chain.md` now provides them (features C01 to C17 of its section 3; the
demand overlay `one-demand` of its 4.4.6; N-047 closed by REF).

**Where REF builds each scenario** (`spec-reference-chain.md` 7; the scenarios of that
spec named X1 to X8 and the viability scenarios A1 to A10 are REF's and VIA's, not part
of this register, and exercise it as noted). Bands are STY's: until `spec-stylised-facts.md`
exists a band key with no value makes a scenario report its measure and assert only its
exact checks (REF 7.1.5, FR-REF-022, FR-REF-023).

| Scenario | REF pack | REF section |
|----------|----------|-------------|
| S01, S03, S05, S08, S09, S18 | `one` | 7.2 |
| S02 | none: a world built in Go in the test | 7.2 |
| S04 | `one`, and `one` + `overlay/nofloor` | 7.2 |
| S06 | `one` + `overlay/imbalance-cook`, with `overlay/skill` and `overlay/notool` | 7.2 |
| S07 | `one` + `overlay/glut-prepared` at four `cost_floor` values | 7.2 |
| S10 | `one` + `overlay/h200` at scales 1, 4 and 10 | 7.2 |
| S11 | `one` + `overlay/estates` | 7.2 |
| S12 | `one` + `overlay/long-grow`; the coarse-tick leg waits (REF 7.6) | 7.2, 7.6 |
| S13 | `one` + `overlay/faucet` (MON 17.6; REF to map) | to be assigned by REF |
| S14 | `famine` | 7.2 |
| S16 | `one` + `overlay/overhang` | 7.2 |
| S17 | `one` warm, with `one` cold as the control | 7.2 |
| S19 | `one` with the halt on and off | 7.2 |
| S20 | `one` + `overlay/poor` | 7.2 |
| S21 | every variant except `famine`, and the defect packs | 7.2 |
| S22 | `one` + `overlay/perturbed`, with `warm-one` and without | 7.2 |
| S23, S24, S25 | `one-demand` (and `one` for comparison in S24) | 7.7 |
| S26 | `one`, with `entry_share` at its default and at 0 and `exit_share` at 0 | 7.7 |
| S27 | `famine` with the yield of `grow_staple` and of the foresters and diggers at 0 | 7.7 |
| S28, S29 | pack: to be assigned by REF (waits for TRN, MER, INT) | to be assigned by REF |
| S30, S31, S32 | pack: to be assigned by REF | to be assigned by REF |
| S33, S34, S35, S36 | pack: to be assigned by REF (`overlay/faucet`, `overlay/faucet-sunk` of MON 17.6 for S13 and S33) | to be assigned by REF |
| S37, S38, S39, S40 | pack: to be assigned by REF (needs RES or a fixed-factor double, the demand overlay on, `Renewal` 0) | to be assigned by REF |

FR-STB-005 can now be checked against this table and REF 7: REF's 7.2 has an entry for
every scenario of this section except S15, or lists it in 7.6 with the spec it waits for
(FR-REF-035, FR-REF-037). REF 7.7 still names the scenarios of DEM and SPN by owner
(`S23-SPN`, `S23-DEM` and so on); REF is updating that text to the final numbers.

The exact checks of S02, S03, S09, S11, S17, S18 and S21 are asserted whether or not a
band exists (FR-REF-023); the other scenarios report until STY sets bands.

## 9. How a spec adds entries: the checklist

Every spec that creates, changes or removes a feedback loop follows this. It is part
of "done" for a spec (it joins the reviewer's list of `AGENTS.md`'s spec work).

9.1. **Look for loops.** For each new rule, write the chain of variables it
creates: what does it read, what does it change, does the change reach what it
read, through which delay? A rule that reads a price, an inventory, an income, a
count of agents or a share and changes any of these is a loop candidate.

9.2. **Write the entry.** Add the fields of section 3 to the spec's own "Entries for
the stability register" section (as `spec-clearing.md` section 13 does), with the
sign of each link and the polarity.

9.3. **Name the damper or say there is none.** Reference the damper by `STB-Dnn`
and section. If it is new, add it to the catalogue (5.1). If nothing damps the loop,
write "Undamped".

9.4. **State the condition and the test.** The condition under which the loop is
stable, and the scenario of section 8 (or a new one) that would show it. If a new
scenario is needed, propose it in the handoff.

9.5. **Give a status.** `hand`, `model` or `untested`. If a model was run, give its
parameters and what it does not show, as 6 does.

9.6. **Check the delay against the neighbours.** A delay near a season length, a
window, a shelf life or another loop's delay is where cycles form (L23). List any
that match.

9.7. **Check the tunables.** Each damper parameter is a tunable with a default,
owner, bound and doc line, a row in the parameter-owner table, and a placeholder
note if not calibrated (N-015, N-016).

9.8. **Hand over.** The spec's handoff or PR lists the new entries so the register
owner merges them here. A spec may not be called done if it adds a loop and no entry.

9.9. **Merge duplicates.** If two specs describe one loop in different words, the
register keeps one entry and names both owners (as L03 and L25 do).

## 10. Cross-check: loops no spec dampens

This is the result of reading the written specs against the register (section 4).
A loop appears here when its Damper field is empty, or when the damper is only a
bound, or when the damper is in a spec that does not exist yet. The list is the
register's honest backlog.

| Loop | What is missing | Who must act |
|------|-----------------|--------------|
| L14 target-income labour loop, `e < 1`; L51 a staple's demand is inelastic | AGT states that a staple's elasticity is 0 to 1, so the condition `e > 1` fails; only the price bounds, the cap on work, the halt (D24) and exit hold it. The rest band and the proportional share (D09, D28) soften steps and do not change the elasticity. The work floor (D13, `MinWorkShare`) is adopted and moves the upper stop from the ceiling to a price set by the parameter; the interior stays a repeller. DEM (round 5) makes a *luxury* role stable (elasticity above 10 000 with headroom, L14, D30) but not a staple; exit (SPN, D35) is slow by design and not an anchor | S04 (floor at default and at 0), S16, S23; rationing was rejected (N-052 closed) and returns only if S04 or S16 shows a pinned staple that the halt, exit and the floor do not cure |
| L02 evidence offset | BEL's revised rule halves the offset (5.6% in the model) and does not remove it; the rest is attributed to the quantity rule and the pairing | BEL (N-042, open question 12) |
| L05 shortage with no supply, L06 overhang | the halt (D24) now caps the overhang, but its threshold decides which bound the price reaches (L44), and nothing makes a staple's flow price-responsive in the interior (exit, D35, is capped and slow) | AGT (S16, threshold) |
| L44 production halt | one threshold, a guess; a halt reached by every agent in one tick stops them together | AGT, S16, S05 |
| L46 closed economy fixes the mean ratio at 1 | without the demand module the glut is bounded by the halt only (the default `TargetRatio` stays 15 000, decided); with it the ratio is `1 + W / U` up to a ceiling and a target above the ceiling is still unmet and flagged (`target_above_demand`). DEM needs a recurring named transfer to a non-producing class with a paired sink (`spec-demand.md` 7.3) | MON, EXT, EMP; S19, S24 |
| L52 cost ceiling against a labour-dominated price | sellers whose labour share exceeds about two thirds of the price are capped below their equilibrium; the fix (labour in `UnitCost`, a higher ceiling, or none where labour dominates) is not decided, and the labour option makes the freeze of L09 likelier | BEL, PRO (N-055), S21 |
| L48 recipe choice by margin | no hysteresis in the choice; a switching margin would need the previous recipe as state | AGT, S06 |
| L36 occupation cobweb, L54 census imbalance, L61 replacement herd | specified by the spawner (D14 `EntryShare`, D09 `RoleHysteresis`, D33, D34, D36); what remains is a condition, not a missing mechanism: `EntryShare * T` below about 2 000 (the default 50 gives 1 500 at the base window of 30, and 6 000 at a window of 120, where SPN's model still swings by 0.35 and the warning `entry_share_window` fires above 2 500); every number is a placeholder from a stylised model; the traineeship throttle covers only skilled roles | S06, S26 |
| L21 production-delay cobweb | beliefs are an averaging expectation, but the production response `es` to price comes only from recipe choice (L48) and entry (D14, slow by design); a staple fails `es < ed` (L51) | AGT |
| L19 a job is a gap in income | only a warning, and starting money for the longest job in a warm start; the window or the income attribution is not fixed | WEL (N-032) |
| L39 pool-dry collapse, L63 the pool serves two masters, L11 budget loop, L45 saving | relief is bounded by the pool and entrants share it (D37 keeps at least half for relief by default); an agent with no money, no goods and no relief is undamped; the floor under market failure (subsistence) is not written | SUB |
| L09 hard cost floor | exit through the ladder (D35) is the only release and is slow by design | BEL (open question 1) |
| L22 lockstep of jobs | the stagger exists only when a warm start is configured | VIA (done), WLD for a cold start |
| L35 synchronised failure, L50 hunger and removal, L75 the Malthusian loop | the failure ladder is specified (SPN 5, D35); it pays in long hunger (SPN 16.2), and the Malthusian loop is now specified (POP, D46 to D52) but has no feedback until a fixed factor exists (L80): the endowment is RES's and RES is not written | RES, S37 |
| L81 the closed ceiling, L82 a low-density trap, L66 an unreachable region | no mechanism beyond the warnings (`replacement_above_closed_ceiling`, `region_isolated`) and `MinHouseholds`; L66 and L82 are hazards by design | S40, S38, S28 |
| L58 substitution flips, L59 periodic wants and cohort pulses, L62 the churn of a famine, L64 an extinct role | no mechanism beyond the exponent bound, the carry spread and the entry gate; L64 is undamped by design | S25, S10, S27 |
| L13, L34 cohort lockstep | the floor `K` and the proportional share for the work decision; the belief gate has one draw per cohort; the first consumer needs no weight above 1 | SCL, N-030 |
| L53 a warm start is not a stable start | no mechanism; the scenario S22 is the only check | STY, REF |
| L23 seasonal forcing, L55 seasonal storage | the storage gap is an error (D23); the forcing has no scenario yet | STY |
| L40 relief and migration | an experiment is the only check | STY, MIG |

## 11. Parameters

This spec adds no configuration field. It indexes the dampers that other specs own
(section 5). The four parameters this spec proposed in its first version have all been
adopted by other specs: `ProdStopRatio` as AGT's `ProduceHaltRatio`, `MinWorkTicks` as
WEL's `MinWorkShare`, and `EntryShare` and `RoleHysteresis` by the spawner under the
same names. None is a proposal any more. The first rows below are AGT's and WEL's:

| Damper | Parameter | Owner | Default (placeholder) |
|--------|-----------|-------|-----------------------|
| D24 (was `ProdStopRatio`) | `ProduceHaltRatio` | `spec-agents.md` 7.1.5, 12 | 30 000 basis points of the output target |
| D09 for the work decision | `RestBand` | `spec-agents.md` 7.1.2, 12 | 500 basis points of the target |
| D13 (was `MinWorkTicks`) | `MinWorkShare` | `spec-welfare-and-wages.md` 5.3, 4.2 | 4 000 basis points of the window (a placeholder) |
| D29 | `WelfareWindow`, `WindowCycles` | `spec-welfare-and-wages.md` 2.3 | 30 ticks, 2 cycles (placeholders) |

The parameters of the spawner's and the demand module's dampers, with `EntryShare` and
`RoleHysteresis` adopted from the first version's proposals; no proposal is left:

| Damper | Parameter | Owner | Default (placeholder) |
|--------|-----------|-------|-----------------------|
| D14 | `EntryShare` | `spec-spawner.md` 6.3, 11 (World) | 50 basis points of the region's households and of the destination's (0 turns role change off) |
| D09 for roles | `RoleHysteresis` | `spec-spawner.md` 6.2, 11 (World) | 1 000 basis points |
| D33 | `ReviewTicks` | `spec-spawner.md` 6.2, 11 (World) | 10 ticks |
| D34 | `RoleCooldown` | `spec-spawner.md` 6.2, 11 (World) | 30 ticks |
| D35 | `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare` | `spec-spawner.md` 5, 11 (World) | 3, 10, 1 000 basis points, 100 basis points |
| D36 | `EntryRatio`; `Role.Closed` | `spec-spawner.md` 6.1, 11 | 8 000 basis points; false |
| D37 | `EntryPoolShare` | `spec-spawner.md` 7.3, 11 (World) | 5 000 basis points |
| D30 | `Elasticity`, `Ceiling` (per want) | `spec-demand.md` 4.3, 9 | 15 000 and 15 000 basis points |
| D31 | `Rank` (per want); the cap is AGT's `SpendMaxShare` | `spec-demand.md` 4.5, 9 | 0; see AGT |
| D32 | none: a rule constant | `spec-demand.md` 4.7 | not applicable |
| D38 to D40 | `MaxDispatchPerTick` (D39); the field `Inbound` (D38) | `spec-transport.md` 5.4, 9.1, 11 | see TRN |
| D41, D42, D43, D45 | `Limit`, `MaxFlowTotal`; `RateBP`, `Exempt`; the carry | `spec-money-supply.md` 4.4, 6, 9, 11.1 | none by default (a flow has no default) |
| D46 | the slopes of `BirthSchedule` and `DeathSchedule` | `spec-population.md` 3.5, 5.4, 8 | flat birth 250, death 450 to 50 basis points a year between ratios 8 000 and 16 000 (placeholders) |
| D49, D50, D51 | `MinHouseholds`; `HouseholdCap`; `BirthStake` | `spec-population.md` 4.3, 4.4, 8 | 1; 0 (none); 10 000 basis points |
| D47, D48, D52 | none: rule constants | `spec-population.md` 3.7, 4.6, 3.8 | not applicable |

Each parameter has its row in the parameter-owner table of `README.md` once the owner
specs' rows are added (the owners' handoffs list them). Nothing here is a default that
has been shown to work: every number is a placeholder from the runs of section 6, from
the models quoted in 6.5 or from a guess (N-015). `RoleHysteresis` is not enough alone
(SPN 16.1), and the entry share is what the model needs; its default of 50 replaced a
first guess of 200 that failed in the model.

## 12. Observability

Everything follows observability-and-logging.md. The register is documentation and
adds no event, no log line and no metric of its own. It makes demands on the
events of other specs:

12.1. **Regimes are computed from events and the snapshot.** The measures of section
7 use `MarketCleared` (price, volume, unmatched), `BeliefsUpdated` (`AtFloor`,
`AtCeiling`, `Clamped`, `MeanCentre`, `LowCentre`, `HighCentre`), `AgentSpawned` and
`AgentBankrupted` with weights, `BoundaryFlow` and the snapshot. A regime that
needs a number no event carries is a request to OBS (FR-STB-012, note N-012).

12.2. **A damper that acts must be seen to act** (FR-STB-013). A clamp, a refusal,
a floor or a limit has a counter in an event or a bounded `Warn`. Today:
`BeliefsUpdated` counts the clamps of beliefs; the rejection reasons of
`OrderRejected` count the refusals of the hard budget; the `Warn` "price beliefs at
bound" reports a pinned market; `UpkeepShortfall` and the `Warn` "households unfed"
(`spec-agents.md` 14) report hunger. Not yet counted: the production halt (L44) has
only a per-agent `Debug` line ("work declined", reason `halted`), no aggregated
counter and no bounded `Warn`. Counted since round 3 (N-065): the work floor D13
(`spec-welfare-and-wages.md` 5.3: the `Snapshot` reports, per region and role, the
households held to work by `MinWorkShare` in the last tick, and the `Warn` "work floor
binding" is emitted once per region and role after ten ticks with more than half held),
and the traineeship throttle and queue D14, D20 (`spec-skill-barriers.md` 8.3: the
queue, the places and the `Warn` "no places open"). Counted since round 5: the demand
schedule and the ladder D30, D31 (`WantShortfall`: `Choked` and `Planned` less `Funded`,
and the `Warn` "wants choked", `spec-demand.md` 11), and the spawner's limits D14, D34,
D35, D36, D37 (`EntryLimited`: `RegionCap`, `RoleCap`, `Cooldown`, `Hysteresis`,
`ExitCap`, `NoViableRole`, `Unfunded`, `Refused`, and the `Warn`s "role extinct", "entry
unfunded" and "group below floor", `spec-spawner.md` 13). These events
(`WantShortfall`, `EntryLimited`, `WeightChanged`, `RoleChanged`) are not yet in the
table of observability-and-logging.md 4.3: their owners list the rules change.
Still not counted: the production halt (L44, N-067), whose owners add an aggregate in an
event per region and role per tick in the style of `UpkeepShortfall` (note N-045).
The regime measures of section 7 may also use `WeightChanged`, `RoleChanged` and
`EntryLimited` for the occupation share and the changes per tick (S26). Counted since
round 6: the money flows' limits D41 (`MoneyFlowLimited`, `MoneyFlowEnded`,
`MoneyFlowScaled`, `spec-money-supply.md` 13), the population module's floor, cap, funding
limit and clamps D49, D50, D51 (`PopulationChanged`: `DeathsHeld`, `BirthsCapped`,
`BirthsUnfunded`, `RateClamped`, `spec-population.md` 10.1), the dispatch cap D39
(`ShipmentRejected`, reason `too_many_requests`, `spec-transport.md` 5.4) and an edge closure
(`EdgeStatusChanged`, `spec-regions.md` 10).

12.3. **Stability tests may add no event of their own.** A test that needs an
internal number reads it from the snapshot or an event; it does not add a back door
(architecture.md 10.2).

## 13. State contributed to a checkpoint

None, because the register is documentation. The dampers' state (for example the
`BoundRun` counter and the price window of `spec-beliefs.md` 13) is contributed by
the specs that own them. A damper added by section 5.1 lists its state in its own
spec, as ADR 0002 requires.

## 14. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. The
register has two kinds of requirement: those on this document, checked by a text
parse of `doc/specs/` like the parameter registry test (testing.md 6.7), and those
on the library, checked by scenarios.

### Functional

| ID | Requirement |
|----|-------------|
| FR-STB-001 | Every register entry MUST have an ID of the form `STB-Lnn`, a kind from the list of section 3, a polarity or "none", an owner, a damper or the word "Undamped", a condition, a test and a status of `hand`, `model` or `untested`, and an ID MUST NOT be reused. |
| FR-STB-002 | Every spec that has a section "Entries for the stability register" MUST have each of its entries present in this register, by name or by a merge noted in the entry. |
| FR-STB-003 | Every spec with a Requirements section that adds a rule reading a price, a stock, an income, a count or a share and changing one MUST have a register entry for the loop or state in its own register section why there is none. |
| FR-STB-004 | Every damper named in a register entry MUST appear in the catalogue of section 5 with a home and a status, and a damper with the status *specified* MUST name a parameter in the parameter-owner table of `README.md` or a rule constant at the section it names. |
| FR-STB-005 | Every register entry with a status of `untested` or `hand` MUST name a scenario of section 8, and every scenario MUST be covered by a test in `spec-stylised-facts.md` or `spec-reference-chain.md` when those specs exist. |
| FR-STB-006 | A damper that is *proposed* MUST NOT be counted as a damper in an entry's Damper field: the entry MUST list the loop as undamped until a spec adopts it. |
| FR-STB-007 | A scenario MUST use fixed seeds, MUST report the failing seed, and MUST be an integration or end-to-end test skipped with `-short` (testing.md 10.5). |
| FR-STB-008 | In every stability scenario no tradable commodity's published price MUST leave the interval allowed by its reference band and cost band, and no balance or inventory MUST be negative (the invariants of testing.md 6.3 hold at every tick). |
| FR-STB-009 | Scenario S04 MUST inject a shortage into a self-employed staple role and MUST assert that the price stays within its bounds and that the role is not pinned at a bound for longer than a stated number of ticks. |
| FR-STB-010 | Scenario S05 MUST run a staple with a delayed supply response and MUST assert that the range of the price over successive windows does not grow without bound, or report the pack as F6. |
| FR-STB-011 | Scenario S07 MUST run a perishable glut with the default `CostFloor` and MUST assert that the market does not stay with no trade while both sides have unmatched orders for longer than a stated number of ticks. |
| FR-STB-012 | The measures of the failure regimes of section 7 MUST be computable from the events and the snapshot alone, without access to unexported state. |
| FR-STB-013 | Every damper that clamps, refuses, floors or limits MUST have a counter in an event or a `Warn` whose volume is bounded by the configuration, and a test that the counter is emitted. |
| FR-STB-014 | The shocks of a scenario MUST be injected only through the integration interface or the configuration (INT, `spec-production.md` 6.8) and MUST NOT be applied by editing state. |
| FR-STB-015 | Each pack that ships MUST be run through the scenarios of section 8 that its content allows, and the result MUST be recorded with the pack. |
| FR-STB-016 | A test over many seeds MUST show that no agent position wins more often than another when limits are equal (STB-L12). |
| FR-STB-017 | A validation or a scenario MUST check, for every cycle of recipes, that the gain `g = CostFloor / 10 000 * s` is below 1, `s` being the full-cost share of the cycle's own inputs (`q_in / q_out` at the solver's prices, which validation computes exactly); a cycle that is not productive in material MUST be rejected (`cycle_not_productive`), a productive cycle with `g` of 1 or more at the configured `CostFloor` MUST produce the warning `cycle_cost_gain`, and a `CostFloor` of 10 000 or more MUST produce the warning `cost_floor_at_full_cost` (the wage loop has gain 1 there) (`spec-config.md` 7.6 item 3, FR-CFG-044, FR-CFG-056). |
| FR-STB-018 | In a scenario with price-responsive flows, the long-run price of a market MUST be within a stated band of the price at which the flows balance, and the unmet need MUST be below a stated share, or the run MUST be reported as F8. |
| FR-STB-019 | A change that adds, removes or weakens a damper MUST update the catalogue and the affected entries in the same change (section 5.1.1, 5.1.8). |
| FR-STB-020 | The library MUST NOT read a macro aggregate and feed it back into an agent's decision or a market's price; a stability mechanism MUST use local information only (section 5.1.3). |
| FR-STB-021 | Scenario S19 MUST run the default `TargetRatio` in a closed economy whose only demand is upkeep, with the production halt on, and MUST assert that the unsold stock of each produced good stays below a stated bound and that the mean welfare ratio is near 1; with the halt off the same run MUST show the stock growing beyond that bound. |
| FR-STB-022 | Scenario S20 MUST start every household below `ReserveTicks` times its upkeep cost and MUST assert that households reach the reserve within a stated number of ticks, or that the world reports the unreachable reserve. |
| FR-STB-023 | For the reference chain, every seed price MUST lie inside the cost band at the seed prices, or the report MUST list it (FR-VIA-027), and scenario S21 MUST fail the pack when a good sold at its seed price is capped below the price at which its seller earns the wage. |
| FR-STB-024 | Scenario S22 MUST run a perturbed chain from the warm start and from a cold start and MUST assert that the warm start does not hide an instability: both runs end in the same regime of section 7. |
| FR-STB-025 | Every entry merged from another spec's register section MUST name its source entry (for example `AGT-A8`, `VIA 19`) in the merged text, so that FR-STB-002 can be checked in both directions. |

| FR-STB-026 | Scenario S23 MUST run a self-employed luxury role under the target-income rule with a want of schedule elasticity at, below and above 10 000 and with budget headroom at 100% and above 110%, perturbed up and down, and MUST assert that the price converges to the reference only when the elasticity is above 10 000 and the headroom is at least about 10%, and MUST report the lower edge of the basin; any other outcome MUST be reported as F5. |
| FR-STB-027 | Scenario S24 MUST run a closed economy with a ladder with the wants on and off and a `TargetRatio` above the ceiling, and MUST assert that the mean welfare ratio is near 1 with the wants off, above 1 with them on and at most `1 + Wmax / U`, and that the unsold stock stays below a stated bound when the target is above the ceiling. |
| FR-STB-028 | Scenario S25 MUST run a want with total money at 0.5, 1 and 2 times the sum of the reserves and a want of two alternatives with `Substitution` 4 and a price gap, and MUST assert that no want is bought by a household below its reserve and that the shares of the alternatives do not swing between the items undamped. |
| FR-STB-029 | Scenario S26 MUST apply a role-income shock with the spawner's caps on and off and MUST assert that the role changes per tick do not exceed `ceil(Households * EntryShare / 10 000)` per region with the caps on, that the role's share stays within a stated band, and that with the caps off the share swings by more than with them on. |
| FR-STB-030 | Scenario S27 MUST run a region where no role is viable and MUST assert that no entrant is created, that the count `NoViableRole` or the `Warn` "role extinct" shows, and that the households fall. |
| FR-STB-031 | The ID of a loop, damper or scenario that two specs proposed under one number MUST be assigned once by this register: loops from L57, dampers D30 to D37 and scenarios S23 to S27, with the owner's local entry ids (`DEM-A1` and so on) kept in the merged text (FR-STB-025). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-STB-001 | The text checks of FR-STB-001 to FR-STB-005 MUST run without a network and in under a stated time, and MUST fail with the offending ID. |
| NFR-STB-002 | The scenarios MUST be reproducible: the same seed and configuration give the same series on every platform (architecture.md 7.4). |

## 15. Open questions

Closed in the revision after the agents and viability specs landed:

- **Where do the models live? Closed (N-068).** The scripts of section 6 are archived in
  `doc/specs/models/stb/` (with the other areas' prototypes; see
  `doc/specs/models/README.md`), run only in Docker, and section 6 also describes each
  model fully enough to rebuild it. The numbers are placeholders from stylised runs. A
  script changes together with the section that quotes it.
- **What anchors the level? Closed (N-043).** `spec-agents.md` 11.6 states that a
  staple's flows stay inelastic and names the anchors: the belief bounds, the halt
  (D24), buffer buying, recipe choice by margin and exit. An elastic subsistence demand
  is not invented. What remains is a test, not a question: S04 and S16 show whether
  the halt and exit cure a pinned market, and N-052 holds the rationing option if not.
- **A controller. Closed.** No in-run controller (2.6). A rule that reacts to measures
  is a policy actor (PLC, ADR 0001).
- **How to count loops. Closed.** Merging entries is by judgement; section 4.7
  records every merge, and IDs are stable once published.

Still open, with the next step:

1. **The work floor is adopted; is it enough? (N-028)** WEL adopted D13 as
   `MinWorkShare` (5.3 there). It sets where a pinned staple market stops and does not
   remove the instability. *Next step:* scenario S04 records the pack's outcome with
   the floor at its default and at 0. N-052 (rationing above a dear price, or an elastic
   demand in DEM) is closed: DEM rejected rationing and supplies elastic demand for
   discretionary goods only (L14, L51, L60); if S04 or S16 shows a pinned *staple* that
   the halt, exit and the floor do not cure, a new note goes to AGT (upkeep
   alternatives, `spec-demand.md` open question 1).
2. **How much of the evidence offset is real? (N-042.)** BEL revised 5.4 and the
   register's model, re-run on the revised rule (run 2b), shows 5.6% low against
   11.2% before, and 5.9% of need unmet. The rest probably comes from the quantity
   rule and the pairing, not from the evidence rule (BEL open question 12). *Next
   step:* BEL's question 12 (a variant of the model with another quantity rule); FR-STB-018
   becomes a hard requirement with a band set by STY, because the offset is reduced, not
   removed.
3. **The thresholds of the regimes.** Section 7 has none. *Next step:* STY sets them
   from the first run of the reference chain.
4. **The coupling of loops.** The register treats loops one by one. The interactions
   (a cobweb on top of a backward-bending labour supply, a freeze feeding a collapse,
   the halt of L44 against the cobweb of L21) are not analysed. *Next step:* S22 and
   S05 with the halt on; no earlier.
5. **Models of the agent rules.** Section 6.1 models the target-income rule without
   the rest band and the proportional share. By hand the band only rescales the
   target `K` by `1 + RestBand / 10 000` and the share equals the closed form for a
   cohort, so neither changes the unit elasticity; 6.1 has not been re-run for them.
   The halt is in 6.2 (run 1b). Neither the budget stack nor the belief inputs of
   `spec-agents.md` 7.3 are modelled. *Next step:* re-run 6.2 against the revised
   BEL when it lands (open question 2).
6. **The models quoted in 6.5 are not re-run here.** The headline numbers of DEM and SPN
   (the luxury basin, the closed-economy ratio 1.53, the entry-share condition) come from
   their models. *Next step:* re-run the occupation model of 6.3 with SPN's discrete rules
   and the luxury model of 6.1 with DEM's schedule when STY or EXP exist, and set the
   threshold of `entry_share_window` from the run, not from the guess.
7. **A cobweb on top of the ladder.** The interaction of the ladder's long hunger
   (L35), the entry share (L36) and relief (L63) is analysed only in SPN's model, which has
   no goods and no market. *Next step:* S14 and S26 together on `famine`, then S22.
8. **Wants and the halt.** Whether the production halt (L44) cuts the supply of want goods
   in a downturn and deepens the loop of L57 is not analysed. *Next step:* S24 with the
   halt on and off.
9. **The generation lag of the population loop (L75).** The model of `spec-population.md` 13.1 has
   no generation lag; with one (a maturity pipeline for births, POP open question 1) the
   approach to the replacement ratio becomes a cycle when gain times delay reaches about 1.
   *Next step:* decide when a scenario needs a cycle (STY).
10. **Shippers that count the pipeline (L84, L85).** The model of `spec-transport.md` 16 has no
    inventory, belief or market; the shares of a shipper's effect at dispatch and at arrival are
    not derived for any pack. *Next step:* S30 with the first merchant strategy (MER).

## 16. Sources

Marked "read in full" only where the whole source was opened. No default is set
from any source in this spec.

| Source | What was used | Status |
|--------|---------------|--------|
| The specs and rules in this repository | the loops of section 4 and the rules modelled in section 6 | **read in full:** the three rule files, `notes.md`, `evidence.md`, the overview, vision, roadmap, `specs/README.md`, the documentation index, `spec-beliefs.md`, `spec-welfare-and-wages.md`, `spec-skill-barriers.md`, `spec-estates.md`, `spec-carriage.md`. **Read in part:** the register sections (and for clearing, scale and production the sections around prices, weights and jobs) of `spec-clearing.md`, `spec-scale.md`, `spec-config.md`, `spec-goods.md`, `spec-ledger.md`, `spec-calendar.md` and `spec-production.md`, plus a search of all of them for loop vocabulary (feedback, oscillation, ratchet, spiral, lockstep, hoard), which found nothing beyond the labelled entries. **Added in the revision (round 3):** `spec-agents.md` section 18 (all thirteen entries), 7.1.2, 7.1.5, 11.6, 12, 13 and 14 read; `spec-viability.md` 4.7, 8.2.3, 13 and 19 read, and its requirements table scanned; `spec-welfare-and-wages.md` 2.3 and the parameter table; `spec-config.md` 7.6. **Added in round 5:** `spec-demand.md` sections 6.3, 7, 9, 11, 12, 13, 18 and the dampers and scenarios of 13 read; `spec-spawner.md` sections 11, 12, 13, 16, 17 and the requirements table read; `spec-reference-chain.md` sections 7, 13 and the requirements table read (the packs of 4 were not read); the round-4 handoffs of `dem`, `spn`, `ref`, `sync` and `rules`. **Not read:** `spec-currency.md`, the rest of `spec-demand.md` and `spec-spawner.md` outside the sections named, and the two ADRs. A loop that lives only in an unread part could be missing from section 4 |
| Cobweb model, Wikipedia. <https://en.wikipedia.org/wiki/Cobweb_model> | the stability inequality under naive expectations: convergent if the supply curve is steeper than the demand curve in price terms, equivalently demand more elastic than supply at the equilibrium | opened through a summarising fetch, not read as a whole; the fetch reports no formula for adaptive expectations |
| Nerlove (1958) adaptive expectations in the cobweb, and the finding that they widen the stable region and can produce chaos | the use of an adaptive expectation in 6.3 | **snippet only** (search result text); the paper was not opened. The runs of 6.3 are this spec's own, not Nerlove's results |
| Backward-bending supply curve of labour, Wikipedia (income and substitution effects) | the mechanism behind the backward bend | opened through a summarising fetch; the page says nothing about stability or multiple equilibria (the claim in N-028 that inflection points create multiple unstable equilibria stays **unverified**) |
| Lengnick (2013), Doran and Parberry (2012) | the belief rules and the stability findings, as recorded in `spec-beliefs.md` 19 | not reopened for this spec; see that section for the status |
| Humphries and Weisdorf (2019), Dyer | the backward-bending labour supply of medieval labourers | not reopened; see `evidence.md` (read directly there) |
| The numeric analyses of section 6 | this spec's own work | stylised models run in Docker (`python:3.12-slim`); not validated against any data |

The user's instruction to run Python only in Docker was followed; no local
interpreter was used for the analyses.

## 17. Where this lives

| Concern | Home |
|---------|------|
| The register, the catalogue, the checklist | this file |
| The text checks of FR-STB-001 to 005 | a test in the root package's external test package that parses `doc/specs/` (like the parameter registry test, testing.md 6.7); written when `spec-config.md` fixes the registry test |
| The scenarios S01 to S40 (S15 unused) | `spec-stylised-facts.md` (bands) and `spec-reference-chain.md` (configurations, section 7; the table after section 8 here), as integration and end-to-end tests |
| The regime measures | `spec-indicators.md`, computed from `spec-observability.md` events |
| The damper counters | the events of the specs that own the dampers |
