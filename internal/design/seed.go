package design

// Seed returns the seed of replica r of point p: Seed0 + r when every point uses
// the same seeds (the default), Seed0 + p*replicas + r when each point has its
// own, in arithmetic modulo 2^64 (spec-experiments.md 4.6.1). The seeds are
// consecutive integers on purpose: a failing run is reproduced by one WithSeed
// call (4.6.2).
func Seed(seed0 uint64, perPoint bool, p, r, replicas int) uint64 {
	if perPoint {
		return seed0 + uint64(p)*uint64(replicas) + uint64(r)
	}
	return seed0 + uint64(r)
}
