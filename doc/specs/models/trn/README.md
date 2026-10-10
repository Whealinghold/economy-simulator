# Models behind `spec-transport.md`

Prototypes of the spec's rules, not of the library (see `../README.md`). Standard
library only. Run each in Docker, never with a local interpreter:

```sh
docker run --rm -v "$PWD/doc/specs/models/trn":/w -w /w python:3.12-slim python pipeline.py
docker run --rm -v "$PWD/doc/specs/models/trn":/w -w /w python:3.12-slim python worked.py
```

| Script | Output | What it shows |
|--------|--------|---------------|
| `pipeline.py` | `pipeline.out` | Section 16 and register entry TRN-A1: the largest gain at which price-gap shipping decays, by transit delay, with a share of the effect at dispatch, with a shipper that counts what is in flight, and the swing under a cap on the shipment per tick. Floating point inside the model only. Numbers are from a linearised gap, not the library |
| `worked.py` | `worked.out` | Section 17: the integer arithmetic of the three-region example (shortest distances, travel times, the pair fit test, spoilage timing, the haul unit cost). Every number is a labelled placeholder |
