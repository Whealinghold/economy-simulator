# Models of `spec-stylised-facts.md`

Standard-library Python, run only in Docker, never on the host
(`docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD":/w -w /w python:3.12-slim python <script>`).
They are prototypes of the **definitions** of the spec (a detector, a seed rule, a
band-setting rule) run on synthetic series, **not** of the library and not runs of it
(note N-027). Nothing in them is a default or a band.

| Script | Output | What it does |
|--------|--------|--------------|
| `seeds.py` | `seeds.out` | (A) the probability that a build passes under SR1 (all seeds) and SR2 (nine in ten) for 10 and 50 seeds by the chance `q` that one seed is outside the band, exact with fractions; (B) the chance that a new seed lies outside the envelope of `m` exchangeable seeds, `2 / (m + 1)`, by enumeration for small `m`; (C) the pass rate of 10 new seeds against the band-setting rule (headroom `h` of the observed range, rounded outward to two significant digits) on a bell-like and a one-sided integer distribution, by `m` and `h`. Quoted in sections 8.3 and 8.4 |
| `detectors.py` | `detectors.out` | the detectors R1 to R9 of section 6 in integer arithmetic on built series: the verdict of each on its regime, on a healthy seasonal series, on a series that is too short and on one with stale prices; the ordering rule of 6.5.2 on the priors; year windows against 120-tick windows on a stationary seasonal series. Quoted in 4.1.3, 6 and 17.8 |
| `worked.py` | `worked.out` | every number of the worked examples of section 17, asserted (imports the two others): a band set from 50 setting seeds, the seed rules on one gate run, the classification of an inflating series, the speed limit of a recovery by integer iteration, the historical premium range, a gap after freight, the earliest unfed tick |

Rerun from this directory. The scripts import one another (`worked.py` imports
`detectors.py` and `seeds.py`), so they must be run from here.
