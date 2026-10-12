# Models behind `spec-saving-and-investment.md`

Three small Docker Python scripts. They are **models of the rules of the spec, not of the library**
(there is no library): they use integers and placeholder parameters (note N-015), and they do not
validate any default (note N-027).

Run, never with a local interpreter. The output is captured by the host shell, so the `.out` files
are owned by the user and not by root:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sav":/w:ro -w /w python:3.12-slim python precaution.py > doc/specs/models/sav/precaution.out
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sav":/w:ro -w /w python:3.12-slim python invest.py > doc/specs/models/sav/invest.out
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sav":/w:ro -w /w python:3.12-slim python worked.py > doc/specs/models/sav/worked.out
```

| File | What it holds | Spec |
|------|---------------|------|
| `precaution.py`, `precaution.out` | the buffer of 4: (A) one household with a two-state income against unfed ticks, money held and discretionary spending at five settings of the share and the cap; (B, B2) a representative household in a circular flow with a leak and a cut in its autonomous income, the trough, the time to recover and the peak buffer; (C) the drawdown of a producer with a 20-tick job | 4, 15.1 |
| `invest.py`, `invest.out` | the project test of 5 as an entry rule over 60 identical households with a price that falls with the number of working sites: thirteen cases of funds, hurdle, spread, commit band and payback, and the ownership ticks by saving rate | 5, 14 (SAV-A3 to A6), 15.2 |
| `worked.py`, `worked.out` | the arithmetic of the examples of 16 (a mill, its repair and the build that no longer passes, reach, a job in progress and the funding tick, a buffer, a cold cell and the saturation of its row, a tool) with the rules as written | 16 |

The scripts use only the Python standard library and write to standard output; each `.out` is the
captured output. Nothing is written by the scripts themselves, and no bytecode is created
(`PYTHONDONTWRITEBYTECODE=1`).
