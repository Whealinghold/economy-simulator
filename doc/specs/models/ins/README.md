# models/ins: the arithmetic of `spec-insurance.md`

One script, `cover.py`, with its output `cover.out`. Standard library only, no
bytecode, run in Docker (never a local interpreter):

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/ins":/w:ro -w /w python:3.12-slim python cover.py
```

It is a model of the **rules** of sections 6 to 8 of the spec with integers and
placeholder numbers, and of two stylised loops (INS-A2 and INS-A4 in part 6, INS-A1
in part 7). It does not show that any default is stable (note N-027).

| Part | What it shows | Spec |
|------|---------------|------|
| 1 | the premium of a route cover on the lane of `spec-risk.md` 18.1 | 6.2, 6.3, 6.4, 19.1 |
| 2 | the claim at the arrival: loss value at the capped unit values, the due | 8.2, 19.1 |
| 3 | one underwriter's policies at one position: pot, shield, pro rata, shortfall; a removal's waterfall | 8.3, 8.4, 19.1, 19.3 |
| 4 | a store cover on the held goods of `spec-risk.md` 18.2, the maximum binds | 6.3.3, 8.2, 19.2 |
| 5 | the value cap against a rising published price | 6.2, INS-A3 |
| 6 | an underwriter's capital under unit and common rows, `CapBP` 5 000 to 40 000 | INS-A2, INS-A4 |
| 7 | the shipper that ignores the loss, reads it, or nets the recovery: totals and the largest gain that decays | INS-A1 |
