# Models of spec-world.md

Small Python scripts (standard library only) and their captured outputs, run in
Docker with `PYTHONDONTWRITEBYTECODE=1`. They are prototypes of the rules of
`doc/specs/spec-world.md`, not of the library (note N-027). See
`doc/specs/models/README.md` for how to run them:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/wld":/w -w /w python:3.12-slim python seed.py
```

| Script | Output | What it does |
|--------|--------|--------------|
| `seed.py` | `seed.out` | The seeding rule of section 7.1 (SplitMix64 expansion into the 128-bit state of Go's PCG-DXSM), asserted against the values Go 1.25 printed for seeds 0, 1, 2 and 42, for the mix and for the plain rule; the bits that differ between the first outputs of seeds `s` and `s + 1` |
| `rotation.py` | `rotation.out` | A cap that binds on tied candidates, under the lower ID, the keyed tie `K(t, id)` and a rotation (10.1.1, WLD-S1) |
| `cadence.py` | `cadence.out` | The reviews per tick of a rotation cadence by raw ID, `Weyl32 mod P`, the Weyl32 bucket (correlated with MIG's margin), the hash bucket (not adopted) and the rule bucket of 9.1.1 (a second multiplier, `0x6A09E667`), over strides 1 to 31 and periods 10 and 30; the correlation of each with the migration margin; the share of coinciding 30 and 10 tick reviews; a group smaller than its period; the drift against a year (9, WLD-S2) |
| `stream.py` | `stream.out` | The share of later draws that differ after one extra draw, with one stream and with a stream per tick (7.6, WLD-A5, WLD-S4) |
| `worked.py` | `worked.out` | The arithmetic of the worked examples of section 19: the rule bucket of ids 10 to 15 and the words of one tick |

The Go check of 7.1 (a stdlib-only `go run` of the seeding, `MarshalBinary` and the
words consumed by `Int64N`, in `golang:1.25`) is not archived.
