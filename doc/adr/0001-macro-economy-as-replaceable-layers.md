# 0001. The macro economy is a replaceable layer with stable defaults

- Status: Accepted
- Date: 2026-10-08
- Supersedes: none

## Context

The library models a market economy for any setting: a fantasy role-playing
game, a space game, or something else (`AGENTS.md`, "Setting-agnostic"). Micro
behaviour (agents, orders, clearing, production) is the core. Macro behaviour
(price levels, money supply, saving and investment, unemployment, growth,
government) is meant to emerge from it, but some of it cannot emerge without a
structural piece. Without money creation a growing economy deflates, without
hired labour there is no unemployment, and without a boundary there is no
trade with the outside (`doc/roadmap.md`, "Macro is measured, never set").

Two risks pull in opposite directions:

- If the library hard-codes macro mechanisms (a mint, a tax scheme, a credit
  system, a growth law), it carries one setting's opinions and a consumer with
  a different setting must fight it.
- If it provides nothing, every consumer rebuilds the same basics, and a new
  user gets an unstable economy by default.

A reference game, EVE Online, shows the value of separating money creation and
destruction from goods creation and destruction, naming every faucet and sink,
and having someone tune the faucets over time. It is an inspiration, not a design to copy. The facts used here come from a
2010 Engadget article
(<https://www.engadget.com/2010-10-24-eve-evolved-isk-sinks-and-faucets.html>)
and CCP's monthly economic reports (<https://www.eveonline.com/news/view/monthly-economic-report-february-2026>),
read as summaries; CCP's own 2010 newsletter PDF could not be read.

## Decision

1. **The core provides primitives, not macro policy.** The core knows accounts
   that hold money and goods, and transfers between them that carry a reason
   code. A **boundary** is an account outside conservation (a source or sink of
   money or goods). A **faucet** is a transfer from a boundary into the economy
   and a **sink** is the reverse. No other operation creates or destroys money,
   so conservation tests can state exactly which faucets and sinks a
   configuration declares.

2. **Macro mechanisms are optional modules, off unless configured.** Credit,
   taxation, saving and investment, hired labour, technology, risk, the outside
   world and similar mechanisms are modules inside the core module. Each is a
   plain struct built from configuration and called by the scheduler in a fixed,
   documented order. There is no plugin registry and no package-level state
   (`doc/rules/architecture.md` 7.1). A module with no configuration does not
   run. The zero configuration is a closed economy with no faucets, taxes,
   credit or growth, and it is the tested baseline.

3. **The consumer can tweak, adjust or replace the macro layer.** There are
   three levels, from least to most change:
   - **Tweak:** change a parameter of a built-in module (a rate, a cap, a
     threshold). Parameters are configuration, each with one owner spec.
   - **Adjust:** enable, disable or compose modules, and choose among the
     built-in rule shapes for a flow (fixed, proportional to a measure,
     threshold, scheduled).
   - **Replace:** disable the built-in macro modules and supply the whole macro
     layer through an extension interface (point 4).

4. **One hook carries custom macro logic.** A `Policy`-style extension reads a
   read-only, deep-copied set of measures and the world's random source and
   returns a closed set of typed actions (transfer between accounts, change a
   rate, add stock to a boundary). It cannot mutate state. The ledger validates
   every action exactly as it validates any transfer, so a custom policy cannot
   create money without a declared faucet and cannot break determinism. The
   default does nothing. Whether this is a new fifth extension point or reuses
   `Strategy` is open (see below).

5. **Defaults are stable macro constants.** The library ships a named default
   for every macro parameter, held in one place per owner spec, chosen so that
   the reference chain stays inside the acceptance bands of
   `spec-stylised-facts.md` over a long run: no runaway inflation or deflation,
   no hoarding trap, no collapse that the configuration did not cause. A new
   user who enables a module without tuning it gets a stable economy. A default
   is a constant, documented with its reason, and changing one is a behaviour
   change that moves the golden files and is explained in its PR.

6. **Opinions live in packs, not the engine.** A pack is configuration (data
   only): commodities, roles, recipes, regions, boundary and faucet settings,
   module switches and rates, and an optional display ladder. An RPG pack and a
   space pack are examples. The welfare-ratio wage model, the target-income
   rule and the tool and traineeship barriers are optional modules or
   calibration tools, not engine behaviour.

7. **Macro is measured, never set.** The engine holds no aggregate state that
   agents react to. Aggregates are computed from the ledger by generic
   aggregators over commodity classes and account sets, and the standard
   indicators (price index, value added, inequality) are presets built on them.
   A consumer may define its own.

8. **Vocabulary stays neutral.** Mechanisms are expressed as upkeep unmet for a
   number of ticks (not "starvation"), a tick of configurable length with
   optional seasons (not "a day"), and a household as whatever one agent
   represents (a family, a crew, a colony).

## Consequences

- More configuration surface and a larger test matrix. We do not test every
  module combination. We test the zero configuration, each module alone, and
  each shipped pack against the stylised-facts bands.
- Every spec says whether it describes core, an optional module, or a pack. The
  roadmap gains a layer column and the specs for welfare, skill barriers and
  currency are reclassified accordingly.
- `spec-money-supply.md`, `spec-indicators.md`, `spec-experiments.md` and
  `spec-stylised-facts.md` are core, because stable defaults and tuning depend
  on measuring and sweeping.
- The ledger needs transfers with reason codes and boundary accounts from the
  start, and observability needs a faucet-and-sink event with the reason, so a
  report can total each source and drain.
- Stable defaults are a requirement we have not yet met. They cannot be claimed
  until the reference chain exists and the acceptance bands are written.
- A custom policy gives a consumer the ability to make an economy unstable.
  That is accepted: the guarantee is conservation and determinism, not
  stability.

## Alternatives considered

- **A fixed macro model in the engine** (one mint, one tax scheme, one growth
  law). Rejected: it is the opinion this decision exists to avoid.
- **Nothing provided; consumers build their own.** Rejected: every consumer
  rebuilds the same basics and a new user starts with an unstable economy.
- **A scripting language for rules.** Rejected for now: it adds a parser,
  sandboxing and a determinism surface. Rule shapes in configuration plus a Go
  extension interface cover the same need.
- **A plugin registry or global module list.** Rejected: it needs package-level
  state, which the architecture rules forbid.

## Open points

1. Does the `Policy` hook reuse the `Strategy` extension point or become a fifth
   extension point? A fifth means a change to the table in
   `doc/rules/architecture.md` 4.3 in the same change.
2. Which rule shapes ship for flows, and which parameters does each take?
3. Which modules are in the first release (`doc/roadmap.md`, Open, item 1)?
4. What numeric acceptance bands define "stable", and what are the default
   constants? These belong to `spec-stylised-facts.md` and
   `spec-calibration.md`.

## Related

- `AGENTS.md`, "Setting-agnostic"
- `doc/roadmap.md`, "Macro is measured, never set"
- `doc/specs/README.md`, parameter owners
- `doc/specs/spec-currency.md`, section 4 (the optional formatter, an earlier
  application of the same idea)
