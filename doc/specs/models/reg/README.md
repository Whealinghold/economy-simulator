# Models behind `spec-regions.md`

`paths.py` is a prototype of the graph queries of `spec-regions.md` section 4, not of the
library. It checks, in integer Python, that the definition of `NextHop` (the smallest
region index among the out-neighbours that lie on a shortest path) gives the
lexicographically smallest shortest path, on 3 000 random directed graphs with
asymmetric lengths, ties and unreachable pairs; that the strongly connected components
are labelled as 4.5 says; and it prints the tables of the three worked examples of
section 12. `paths.out` is its output.

```sh
docker run --rm -v "$PWD/doc/specs/models/reg":/w -w /w python:3.12-slim python paths.py
```

Standard library only. It measures no speed: the cost figures of the spec are
operation counts, not timings (note N-015).
