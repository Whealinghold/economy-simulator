# Models of `spec-indicators.md`

Standard-library Python, run only in Docker, never on the host
(`docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD":/w -w /w python:3.12-slim python <script>`).
They are prototypes of the integer rules of the spec, not of the library (note N-027).

| Script | Output | What it does |
|--------|--------|--------------|
| `arith.py` | `arith.out` | half to even on both signs; the Gini rule of section 5.4.1 against the pairwise sum with `Fraction` on 2 000 random row sets; the maximum `(W - 1) / W`; the nearest-integer property of the Fisher square root on 20 000 random products; a 400-link chain against the exact product; the bound `10 000 * W * S < 2^127`. Quoted in sections 5 and 17 |
| `worked.py` | `worked.out` | every number of the worked examples of section 18 (imports `arith.py`): the production, expenditure and regional identities and their equality, the recipe table with its `WIP` row, freight and the delivered price, the three indices, the Gini and the quantiles, sectors and velocity, the households account and the unfed share, distributions, trade balance, the stationary world, and the calendar and season windows; each figure is asserted. Quoted in section 18 |
