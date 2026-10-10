# models/res: prototype models behind spec-resources.md

Stylised models of the RULES of `spec-resources.md`, not of the library, and not claims that any default is stable
(notes N-015, N-027). Run each with Docker, never a local interpreter, and keep Python from writing bytecode (a
root-owned `__pycache__` from a container blocks removing a worktree):

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/res":/w -w /w python:3.12-slim python coupled.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `regen.py`, `regen.out` | 4.2, 4.3, 4.6 | The integer regeneration rules (three shapes, the carried remainder, the round-up logistic factor): the units added over a run equal `floor((carry0 + sum of numerators) / D)` on 400 random runs; a small logistic stock regrows with the carry and is stuck without it; the largest numerator and denominator fit their types. |
| `allocate.py`, `allocate.out` | 5.3 | The ration of a scarce endowment: first come first served in the rotation of `spec-production.md` 5.5.1, proportional with a rotating tie-break and the spec's proportional rule with a hash key, on 1 000 agents of weight 1, on cohorts, and with mixed draws. |
| `coupled.py`, `coupled.out` | 8.2 to 8.4 | The Malthusian loop of `spec-population.md` 5 closed by a refilling, a logistic and a constant-inflow stock, and by a pure quota: the approach, the overshoot threshold against the linearised prediction, the basin of the logistic stock, the neutral cycle. The economy is one assumed function of the granted share and the abundance. |
| `recovery.py`, `recovery.out` | 6.1, 8.4 | A blight that sets the stock to 5% of the capacity, for the four recoverable and permanent cases; the run-down of a finite deposit; the basin of the logistic stock by bisection. |
| `worked.py`, `worked.out` | 16, 18 | The integer arithmetic of the two worked examples (the stock of an arable resource through a year, a logistic woodland at three loads, the runway of a finite ice field, the ration of 5.3 by hand) and the sizing of the five-region proposal. |

`coupled.py` is imported by `recovery.py` and `worked.py`; `allocate.py` by `worked.py`.
