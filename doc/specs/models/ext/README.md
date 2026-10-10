# Models behind `spec-external-sector.md`

Two small integer models of the rules of `spec-external-sector.md`. They are prototypes of the
spec's rules, not of the library: Python integers (so the 128-bit intermediates are not
exercised), Python's `random` standing in for the world's seeded source (its draws are not the
library's), and placeholder numbers. They show arithmetic, conservation and the shape of an
effect, not a result or a stable default (notes N-015, N-027).

| File | What it holds | Quoted in |
|------|---------------|-----------|
| `cycle.py`, `cycle.out` | The counterparty cycle: top-up, order, a copy of the average clearing, sweep; the six reasons; a mean-reverting walk, a trend, a window, a capacity with a carry and cumulative caps. Asserts after every tick that the total of money and of each commodity is zero over accounts and boundaries, that the counterparty accounts are empty, and that the change of the realm's money equals the sum of budget less unspent less receipts | sections 6.4.2, 7.4, 18 and 19.1 |
| `corridor.py`, `corridor.out` | A production-delay cobweb with and without an outside, with a uniform price standing in for the belief and clearing dynamics | sections 11.6, 17 (EXT-A2, A5) and 19.2 |

Run them with Docker, never with a local interpreter, and without bytecode so no `__pycache__`
is left in the folder:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/ext":/w -w /w python:3.12-slim python cycle.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/ext":/w -w /w python:3.12-slim python corridor.py
```

`corridor.py` is self-contained. Standard library only.

Proposed line for `doc/specs/models/README.md` (the supervisor adds it): `| ext/ | spec-external-sector.md sections 6, 7, 17 to 19 | The counterparty cycle and its conservation, a walk, caps, and a cobweb with and without an outside (cycle.py, corridor.py, with outputs) |`
