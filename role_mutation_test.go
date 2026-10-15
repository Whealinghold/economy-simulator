package economysim

// Mutation review of role.go (gomu): the existing role tests already kill every
// mutant that changes a result. Nothing is left to assert; the survivors and
// timeouts are recorded here (one line of why each).
//
// Equivalent mutants (cannot be killed, each hand-applied and run against the suite):
//   - line 92  `len(ExtraUpkeep) == 0` -> `<= 0`: a length is never negative.
//   - line 124 `n > 0` -> `n != 0`: n is len(merged), never negative.
//   - line 124 `merged[n-1].commodity == it.commodity` -> `>=`: out is sorted ascending,
//     so the last merged commodity is never above the next item's; `>=` agrees with `==`.
//   - line 151 `lo < hi` -> `lo != hi`: the search keeps lo <= hi, so `!=` is `<`.
//   - line 159 `lo < len(regionUpkeep)` -> `!=`: the search ends with lo <= len.
//   - line 159 `regionUpkeep[lo].region == g` -> `<=`: lo is the first row with region >= g,
//     so `<=` agrees with `==`.
//
// Hang mutants (reported TIMED_OUT, not killed, by gomu; each runs the binary into
// "test timed out" under `go test -timeout`, so the suite does detect them, but a test
// cannot fail an infinite loop without a wall-clock guard, which testing.md 10.2 forbids):
//   - line 151 `lo < hi` -> `<=`, line 152 `lo+hi` -> `lo/hi`, and line 154 `m + 1` ->
//     `m - 1`, `m * 1`, `m / 1` in upkeepIn's binary search: lo never reaches hi.
