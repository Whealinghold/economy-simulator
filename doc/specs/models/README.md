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

## Rules for this folder

- A script that supports a number in a spec is kept; scratch and debugging
  scripts are not.
- A script is changed together with the spec section that quotes it, in the same
  change.
- No dependency outside the Python standard library.
