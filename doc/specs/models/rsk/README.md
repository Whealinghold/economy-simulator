# models/rsk: the arithmetic of `spec-risk.md`

One script, `loss.py`, with its output `loss.out`. Standard library only, no
bytecode, run in Docker (never a local interpreter):

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/rsk":/w -w /w python:3.12-slim python loss.py
```

It is a model of the **rules** of sections 3 to 6 and 8 of the spec with integers and
placeholder numbers (the PCG and its seeding are those of `models/wld/seed.py`, which
matches Go 1.25). It does not show that any default is stable (note N-027).

| Part | What it shows | Spec |
|------|---------------|------|
| 1 | the first words of seeds 0, 1 and 42 reduced by the world's draw, and the strike test | 3.4.2 |
| 2 | example A: two rows, one common, one unit, the combined share, the loss of four lines (one with a spoiled part, a tie to even), a contract's pay and refund | 4.1, 4.3, 5.2, 5.3, 5.6, 18.1 |
| 3 | example B: a limited owner with two store classes, the stored fraction `phi`, the share and the loss by class | 6.1, 6.3, 18.2 |
| 4 | example C: the traveller's own account, a `Vehicle` among its holdings (not exposed) | 5.4, 18.3 |
| 5 | the rounding bias of half to even on small lines | 5.3.2 |
| 6 | the expected loss of a route against 500 000 simulated dispatches, and the frequency of a strike | 3.4.2, 8.1.1 |
| 7 | the price-gap shipper with a loss: the totals shipped, arrived and wasted when the loss is read and when it is not; the largest gain that decays with a delay of 3, with none and with half of the effect at dispatch | 16 (RSK-A1), 17 |
