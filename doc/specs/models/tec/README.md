# models/tec: prototype models behind spec-technology.md

Stylised models of the RULES of `spec-technology.md`, not of the library, and not claims that any default is stable
(notes N-015, N-027). Run each with Docker, never a local interpreter, and keep Python from writing bytecode (a
root-owned `__pycache__` from a container blocks removing a worktree). The scripts import each other and
`models/res/coupled.py`, so mount the whole `models` directory read-only and write the output with the shell:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models":/m:ro -w /m/tec python:3.12-slim python law.py > doc/specs/models/tec/law.out
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `law.py`, `law.out` | 3.3, 4.2 to 4.6, 6.3 | The integer rules: the piecewise-linear log2 in 12-bit fixed point (exact at a doubling, error at most 0.086 of a doubling between, monotone, no overflow at the bounds); the trend as a carried remainder (exact over whole years); the keyed discovery (the hit rate of the stateless hash against `DiscoveryBP`, independence across keys and cells); the diffusion step (approach from below to the weighted frontier, no order effect over 6 permutations, no contact over a closed edge); the identity of `TechnologyAdvanced`. |
| `malthus.py`, `malthus.out` | 10.3, 13.2, 13.3 | A gain in the Malthusian loop of `spec-population.md` 5 closed by an endowment (`models/res/coupled.py`): the tracking premium of a trend against `10 000 * TrendBP / ((10 000 + G) g' eta)` (within 4% at year 40, 9% at year 60 and 22% to 27% below it at year 20), the e-fold time of a step against `1 / kappa`, the paid-back premium after a cap, the stock share `sigma*(T)` of a refilling stock, and a logistic stock pushed past its maximum-yield point (a step and a ramp). The ratio is proportional to output (RES 8.2), so the premium is an upper bound (see `farmer.py`). |
| `farmer.py`, `farmer.out` | 13.5 | An analysis (closed form, not a run) of the target-income rule against a demand curve of elasticity `e`: a gain becomes rest first, then output; income per household falls as `T^(1 - 1/e)` once the work floor binds when `e` is below 1. |
| `worked.py`, `worked.out` | 9.5, 16 | The arithmetic of the worked examples (three cells over five years, one step of P8.3c by hand) and the bounds of the module's products (the chain of PRO 6.3 with a hired factor of `MaxYieldBP`, 1.1 * 10^25, and the contact weight product, 10^13). |

`law.py` is imported by `worked.py`; `models/res/coupled.py` is imported by `malthus.py`.
