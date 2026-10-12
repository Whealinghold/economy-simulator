# Prototype models behind the specs' numbers

Small Python scripts (and a few captured outputs) that the spec workers wrote to
check the equations and rules of the specs. They are **prototypes of the specs'
rules, not of the library**: they are not part of the Go module, are not tested
by its suite, and nothing in the library imports them. Where a spec quotes a
number from a stylised run, the script that produced it is here. The numbers
are placeholders and models only (note N-027): no script shows that a default
is stable.

## Running

Never with a local interpreter. Use Docker, mounting the folder of the area:

```sh
docker run --rm -v "$PWD/doc/specs/models/stb":/w -w /w python:3.12-slim python r3_rev.py
```

The scripts use only the Python standard library (the viability solver uses
`fractions` for exact arithmetic). Each script writes to standard output.

## Contents

| Folder | Spec | What it holds |
|--------|------|---------------|
| `stb/` | `spec-stability.md` section 6 | The target-income labour loop (`a_labour.py`, `a2.py`, `a3.py`), the belief, price and inventory loop (`b_bel.py` with the revised evidence rule, `b_bel_old.py` with the first rule, `b1.py` to `b6.py`), the production and occupation cobwebs (`c_cobweb.py`) and the round-3 runs (`r3_halt.py`, `r3_rev.py`, `r3_off.py`) with outputs |
| `via/` | `spec-viability.md` section 15 | The exact steady-state solver prototype (`proto.py`, `proto2.py`, `proto3.py`), the three-region check (`ricardo.py`), the rescaled chains (`round.py`) and the labour-inclusive seed check (`sweep.py`) |
| `pb/` | `spec-beliefs.md` 3.2.2 and 5.4, `spec-production.md` 9 | The cost-floor freeze runs (`floor.py`, `floor.out`), the evidence-rule variants (`ev.py` to `ev5.py`, `repro.py`, `trace.py`), the labour-inclusive cost check (`cost.py`) and the belief model (`b_bel.py`) |
| `agt/` | `spec-agents.md` section 16 | The integer arithmetic of the worked examples (`check.py`) |
| `reg/` | `spec-regions.md` sections 4, 12 and 14 | The path definitions check on random directed graphs and the tables of the worked examples (`paths.py`, `paths.out`) |
| `mon/` | `spec-money-supply.md` sections 9, 17 and 18 | The integer rules of the money flows: carry, rate levy, conservation, a balanced pair (`flows.py`, `flows.out`) |
| `pop/` | `spec-population.md` sections 5, 13 | The Malthusian fixed point, the gain-times-delay sweep, the stake loop and the arithmetic of the worked examples (`fixed_point.py`, `sweep.py`, `stake.py`, `worked.py`) |
| `trn/` | `spec-transport.md` sections 16 and 17 | The price-gap shipping delay model (`pipeline.py`, `pipeline.out`) and the arithmetic of the three-region example (`worked.py`, `worked.out`) |
| `res/` | `spec-resources.md` sections 4, 5.3, 6.1, 8 and 16 to 18 | The integer regeneration rules and their carry (`regen.py`), the ration of a scarce endowment (`allocate.py`), the Malthusian loop closed by a refilling, a logistic and a constant-inflow stock (`coupled.py`), recovery after a blight and the basin of a logistic stock (`recovery.py`) and the arithmetic of the worked examples and the five-region sizing (`worked.py`), with outputs |
| `inf/` | `spec-information.md` sections 11 and 12 | The largest gain at which price-gap shipping decays when the shipper reads a rumour of a given lag (`lag.py`), threshold shippers under identical, spread, independent-error and shared-error readings (`shippers.py`) and the arithmetic of the two worked examples (`worked.py`), with outputs |
| `ext/` | `spec-external-sector.md` sections 6.4, 6.7, 7, 18 and 19 | The counterparty cycle (top-up, order, sweep) with its conservation checks, a walk, a window and caps (`cycle.py`), and a production cobweb with and without an outside (`corridor.py`), with outputs |
| `sub/` | `spec-subsistence.md` sections 11, 16 and 17 | The pinned staple market with 1 370 consumer households added, with and without the own-use mode: the floor off, three yields, the retry on and off, the work floor on and off, an outage of the market, the refused exit by purchase (`staple.py`), and the integer rules of the worked examples: the surplus factors, `Cov`, the trace of one household and the ramp of a cohort (`worked.py`), with outputs |
| `mig/` | `spec-migration.md` sections 15 and 16 | The delayed integrator of migration and its limit, each damper alone against an undamped base, the outflow and pipeline caps, herding across two equal destinations, the defaults over seeds (`loop.py`) and the integer arithmetic of the two worked examples (`worked.py`), with outputs |
| `mer/` | `spec-merchants.md` sections 5.7, 7, 9, 10, 18 and 19 | The lane of merchants with a lock, a spread, an error and a cap (`lane.py`), the contract round with its conservation checks (`contract.py`), the arithmetic of the worked examples (`worked.py`) and the backhaul double count (`backhaul.py`), with outputs |
| `wld/` | `spec-world.md` sections 7.1, 7.6, 9.1, 9.3, 10.1 and 19 | The seeding rule checked against Go 1.25 values (`seed.py`), a binding cap under the lower ID, a keyed tie and a rotation (`rotation.py`), the reviews per tick of a rotation by raw ID, `Weyl32`, a hash and the rule bucket with its model of MIG's margin (`cadence.py`), the positional stream after one extra draw (`stream.py`) and the arithmetic of the examples (`worked.py`), with outputs |
| `chk/` | `spec-checkpoint.md` sections 5, 10, 16 and 17 | The size of a checkpoint under two integer encodings (`size.py`) and the byte examples of the encoding with the reseed vectors (`worked.py`), with outputs |
| `ind/` | `spec-indicators.md` sections 5.4, 17 and 18 | Half to even, the Gini rule against the pairwise sum with exact fractions and the bound on the maximum (`arith.py`), and the arithmetic of the worked examples: identities, recipe table, indices, Gini, quantiles, sectors, trade balance (`worked.py`), with outputs |
| `sty/` | `spec-stylised-facts.md` sections 4.1.3, 6, 8.2 to 8.4 and 17 | The seed rules and the band-setting rule in exact arithmetic (`seeds.py`), the detectors R1 to R9 in integers on synthetic series (`detectors.py`) and the numbers of the worked examples (`worked.py`), with outputs |
| `exp/` | `spec-experiments.md` sections 4.4, 4.7, 7.4, 8.3 to 8.5, 10 and 19 | The counter-based Latin hypercube (`lhs.py`), the rejected Kronecker sequence (`kronecker.py`), the integer reductions (`reduce.py`), the pairing toy and its gain (`pairing.py`), the cost arithmetic (`cost.py`) and the worked points (`worked.py`), with outputs |
| `clb/` | `spec-calibration.md` sections 5.2, 7.1 to 7.4, 11.1, 18 and 19 | The registry checks as plain text over the spec, the README and `spec-config.md` (`registry.py`) and the references of the spec (`refs.py`), the grid and its cut in integers (`grid.py`), the exact binomial arithmetic of the stop rules (`stop.py`), the cost of the stages (`cost.py`), the feasible set, minimal change, Pareto set and alias test (`choose.py`) and the worked examples (`worked.py`), with outputs |

## Rules for this folder

- A script that supports a number in a spec is kept; scratch and debugging
  scripts are not.
- A script is changed together with the spec section that quotes it, in the same
  change.
- No dependency outside the Python standard library.
