# Models behind `spec-infrastructure.md`

One small Docker Python script. It is a **model of the rules of the spec, not of the library**
(there is no library): it uses integers and placeholder parameters (note N-015), and it does not
validate any default (note N-027).

Run, never with a local interpreter. The output is captured by the host shell, so `worked.out` is
owned by the user and not by root:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/cap":/w:ro -w /w python:3.12-slim python worked.py > doc/specs/models/cap/worked.out
```

| File | What it holds | Spec |
|------|---------------|------|
| `worked.py`, `worked.out` | eight parts: (1) the transit factor and the travel time of an edge with route facilities (a road at full condition, a built one at 1 999, half, two, a consumer's shock and a closed pair), and (1b) which pair a layer applies to (the direct edge that is the shortest path, 5.2.5); (2) the effective capacity of an endowment with a ground facility as its condition falls, and the stock lost at each step of the clamp; (3) the most-worn-first assignment of use wear over 5 000 random lists of facilities and two starts of one principal (every execution placed, no condition negative, no facility above its remaining `cap(f)`, the grouping of the starts changing nothing), (3c) two mills and two starts, and (3b) the capacity bound per principal and tick across an own start and two hires; (4) a mill of a cohort of 10 under five repair policies over 1 000 ticks; (5) the delay and the gain of an invented builder's rule against a decaying capital stock (the cobweb of CAP-A1), as a model of that rule and not of any rule in the library | 3.3, 5.1, 5.2, 5.3, 16, 18 |

The script uses only the Python standard library and writes to standard output; `worked.out` is
the captured output.
