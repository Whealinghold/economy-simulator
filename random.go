package economysim

import (
	"math/bits"
	"math/rand/v2"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// The draw sites of the slice (spec-world.md 7.1.4). Each constant is a
// literal that does not depend on the position of its row in the table of 7.3,
// and none ever changes: a change moves every golden output (architecture.md
// 6.6). Each equals the first eight bytes, big-endian, of SHA-256 of
// "economy-sim/site/" and its Go name, as a record of origin only. No two
// differ by less than 2^32, because the first fold step adds the site to the
// seed (a Guard: test holds this). A later module appends its own site with
// its change; the rows D2, D4, D6, D11 to D15 and D17 have none yet.
const (
	// siteConstruction keys D1: tick 0, id the agent.
	siteConstruction uint64 = 0xC103123756D13F20
	// siteWork keys D3: tick, id the agent.
	siteWork uint64 = 0xDE594838D3779A37
	// siteTrade keys D5: tick, id the agent.
	siteTrade uint64 = 0xFC14CD10E8A8EF43
	// siteClearing keys D7: tick, id the region, sub the commodity.
	siteClearing uint64 = 0xD364E6AE73E1313F
	// siteOutcome keys D8: tick, id the agent, sub the commodity.
	siteOutcome uint64 = 0xD9CC0A517EB0D9F5
	// siteSpawner keys D9: tick, id the region.
	siteSpawner uint64 = 0xBE0B1D169200E9FF
	// siteCreation keys D10: tick, id the new agent.
	siteCreation uint64 = 0x9B8FFA303CB545DC
)

// stream is the one random source of a world (architecture.md 7.3, ADR 0008):
// the seed, one rand.PCG and one *rand.Rand built on it, both created once. The
// position in a stream is not state. Before a site takes a word the world
// calls keyed or keyedPair, which reseeds the PCG from the seed and the key
// and so resets the whole stream; nothing reads the PCG between two sites.
//
// A stream is not safe for concurrent use, like the world that holds it.
type stream struct {
	seed uint64
	pcg  *rand.PCG
	rnd  *rand.Rand
}

// newStream builds the source of a world with the given effective seed. The
// PCG holds a fixed state until the first keyed call.
func newStream(seed uint64) *stream {
	pcg := rand.NewPCG(0, 0)
	return &stream{seed: seed, pcg: pcg, rnd: rand.New(pcg)}
}

// keyed starts the stream of (site, tick, id) of a row with three key words
// (spec-world.md 7.1.3). The first word then taken is its first word. It does
// not allocate.
func (s *stream) keyed(site uint64, tick, id int64) {
	hi, lo := ordering.StreamKey(s.seed, site, tick, id)
	s.pcg.Seed(hi, lo)
}

// keyedPair is keyed for a row that names a pair: the fourth word sub is
// folded after id (D7: region and commodity; D8: agent and commodity).
func (s *stream) keyedPair(site uint64, tick, id, sub int64) {
	hi, lo := ordering.StreamKeySub(s.seed, site, tick, id, sub)
	s.pcg.Seed(hi, lo)
}

// word returns the next word of the stream in use, for a key (the tie-break
// key of a market, spec-world.md 7.2.1).
func (s *stream) word() uint64 {
	return s.pcg.Uint64()
}

// draw is the world's bounded draw (spec-world.md 7.2.1): the high word of
// Uint64() * n, a value in [0, n). It always consumes exactly one word, so a
// bound of 1 still advances the stream. A bound below 1 has no range: it
// returns 0 and consumes its word too, so the stream never depends on a bound.
func (s *stream) draw(n int64) int64 {
	w := s.pcg.Uint64()
	if n < 1 {
		return 0
	}
	hi, _ := bits.Mul64(w, uint64(n))
	return int64(hi)
}

// drawRange returns a value in [lo, hi], lo + draw(hi - lo + 1), consuming one
// word. An empty range (hi below lo) returns lo and consumes its word. The
// whole int64 range is allowed: the span is taken modulo 2^64 and a span of
// 2^64 returns the raw word.
func (s *stream) drawRange(lo, hi int64) int64 {
	w := s.pcg.Uint64()
	if hi < lo {
		return lo
	}
	span := uint64(hi) - uint64(lo) + 1
	if span == 0 {
		return int64(uint64(lo) + w)
	}
	r, _ := bits.Mul64(w, span)
	return int64(uint64(lo) + r)
}

// reseed is the pure reseed function of a restore (spec-checkpoint.md 6.8.2):
// seed' = finalise(seed xor finalise(salt)). It installs nothing and takes no
// word; the result is the effective seed from the first draw on. Seed 0 with
// salt 0 gives 0, because finalise(0) is 0 (note N-215); every other pair is
// an ordinary reseed, and reseeds compose.
func reseed(seed, salt uint64) uint64 {
	return ordering.Finalise(seed ^ ordering.Finalise(salt))
}
