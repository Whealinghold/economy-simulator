# models/fis: prototype models behind spec-fiscal.md

Checks of the RULES of `spec-fiscal.md` (the pro rata cut, the carry of a duty, the cash
channel of a duty, a standing bid and ask around a reference price) and the arithmetic of
its worked examples. They are prototypes of the spec's rules, not of the library, and not
claims that any default is stable (notes N-015, N-027). Run each with Docker, never a
local interpreter, and never leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/fis":/w:ro -w /w python:3.12-slim python cut.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/fis":/w:ro -w /w python:3.12-slim python laffer.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/fis":/w:ro -w /w python:3.12-slim python band.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/fis":/w:ro -w /w python:3.12-slim python worked.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `cut.py` (output `cut.out`) | 8.4, 6.3, 20 | The pro rata cut with the largest remainder and the keyed tie (the stateless key of `spec-resources.md` 5.3.3) on 20 000 random instances: the whole treasury used, no recipient above its due, the same result for any listing order, at most one unit above the floor of the proportional share; the key against "lower ID first" over 1 000 ticks; and the carry of a duty on 2 000 random runs against its closed form. |
| `laffer.py` (output `laffer.out`) | 19 (FIS-A2) | The cash channel of a duty in one sector, exact rationals: revenue per tick against the rate at several ticks for a treasury that pays nothing back, the best rate against `1 / (s (t + 1))`, and the steady state for several payout shares against its closed form. |
| `band.py` (output `band.out`) | 9.7, 19 (FIS-A4, A5) | A standing bid and ask of a treasury at 80% and 120% of a reference price on a toy linear market, integer arithmetic: iid shocks, a persistent shortage and a persistent glut; stock, cash and ticks outside the band. |
| `worked.py` (output `worked.out`) | 18 | The integer arithmetic of the worked examples: a poll tax, a holding tax, a remittance and a cut with its keys; the crossings and tolls of the kingdom of `spec-polity.md` 12.1; a duty with a carry; a tariff and an export duty at a gate. |
