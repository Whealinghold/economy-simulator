package welfare

// NextRun returns a diagnostic run counter advanced by one tick, saturating at
// math.MaxInt64 instead of wrapping (spec-welfare-and-wages.md FR-WEL-033;
// spec-observability.md 7.3). It uses integers only.
func NextRun(run int64) int64 { return satAdd(run, 1) }
