package economysim

// The test-only seam of the per-position state hash (plan 1-12, phase 4,
// D174): one-line wrappers over poshash_internal_test.go for the external
// package. Nothing is exported from production code (architecture.md 10.2).

// HashWorld is the state hash of the whole world (D177).
func HashWorld(w *World) [32]byte { return hashWorld(w) }

// ReplayStep runs one step through the position methods, as Step does.
func ReplayStep(w *World) error { return replayStep(w, func(string) {}) }

// PositionHashes replays ticks steps and returns the lines
// "t=<tick> <position> <16 hex>".
func PositionHashes(w *World, ticks int) ([]string, error) { return positionHashes(w, ticks) }

// PositionHashesPerturbed is PositionHashes with a change of inert state made
// just before the line of the given tick and position is hashed: the world
// seed recorded in worldState is flipped, which no position reads.
func PositionHashesPerturbed(w *World, ticks int, tick int64, position string) ([]string, error) {
	return positionHooked(w, ticks, func(t int64, p string) {
		if t == tick && p == position {
			w.state.seed ^= 1
		}
	})
}

// FirstDivergence names the tick and position of the first line at which two
// recorded sequences differ; ok is false when they are equal.
func FirstDivergence(a, b []string) (tick int64, position string, ok bool) {
	d := firstDivergence(a, b)
	return d.Tick, d.Position, d.Line >= 0
}
