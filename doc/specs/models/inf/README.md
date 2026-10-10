# Models behind `spec-information.md`

Prototypes of the spec's rules, not of the library (see `../README.md`). Standard
library only. Run each in Docker, never with a local interpreter, and without
writing bytecode:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/inf":/w -w /w python:3.12-slim python lag.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/inf":/w -w /w python:3.12-slim python shippers.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/inf":/w -w /w python:3.12-slim python worked.py
```

| Script | Output | What it shows |
|--------|--------|---------------|
| `lag.py` | `lag.out` | Section 11.1 and register entry INF-A1: the largest gain at which price-gap shipping decays when the shipper reads a rumour of lag `L` and the goods take `d` ticks, with a share of the effect at dispatch, and with a spread of vintages over the shippers. A linearised gap, floating point inside the model only. Extends `../trn/pipeline.py` |
| `shippers.py` | `shippers.out` | Section 11.2 and register entry INF-A2: threshold shippers that ship a lot when the rumoured gap exceeds the freight, with identical shippers, a spread of vintages, an independent error per shipper, an error shared by all, a static spread of the shippers' own thresholds, the number of shippers and a cap on what can be shipped. Seeded `random.Random`; the library draws from the world's source (spec 7) |
| `worked.py` | `worked.out` | Section 12: the integer arithmetic of the two worked examples (lags from distances, the vintage a reader sees, the bounds of a rumour with the error). Every number is a labelled placeholder |
