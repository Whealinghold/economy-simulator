package economysim

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestStream_BareSeedVectors verifies the generator behind the world's source:
// the PCG seeded from a bare seed (the empty fold) gives the first two words
// of spec-world.md 7.1.5 (a). It pins the finaliser and the generator
// together, so a toolchain that changed the generator would fail here.
//
// Requirement: FR-WLD-060
func TestStream_BareSeedVectors(t *testing.T) {
	t.Parallel()
	// Source: spec-world.md 7.1.5 (a), produced by models/wld/seed.py.
	tests := []struct {
		seed   uint64
		w0, w1 uint64
	}{
		{0, 0xb18c97843f57bb91, 0xd7046f21f79eba31},
		{1, 0xaf9d32bf75779748, 0xb7672df032bf0473},
		{2, 0xa28db69c539314fb, 0xfe57a921339b50b9},
		{42, 0x61c88529c9612c1b, 0x2608d8a3075aa168},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			s := newStream(tt.seed)
			// The empty fold: hi = finalise(seed + g), lo = finalise(seed + g + g).
			s.pcg.Seed(ordering.Finalise(tt.seed+ordering.Increment), ordering.Finalise(tt.seed+ordering.Increment+ordering.Increment))
			if w0, w1 := s.word(), s.word(); w0 != tt.w0 || w1 != tt.w1 {
				t.Errorf("bare seed %d words = %016x %016x, want %016x %016x", tt.seed, w0, w1, tt.w0, tt.w1)
			}
		})
	}
}

// TestStream_KeyedVectors verifies the keyed streams k1 to k9 of
// spec-world.md 7.1.5 (b): the first two words, the draw of bound 1 on the
// first word (0, which still consumes it) and the draw of bound 100 on the
// second. Rows k6 to k8 pin the two's-complement conversion of a negative tick
// or ID.
//
// Stage 0 exit test.
//
// Requirement: FR-WLD-060, FR-WLD-061, FR-WLD-068
func TestStream_KeyedVectors(t *testing.T) {
	t.Parallel()
	// Source: spec-world.md 7.1.5 (b), produced by models/wld/seed.py.
	tests := []struct {
		name        string
		seed, site  uint64
		tick, id    int64
		sub         int64
		pair        bool
		w0, w1      uint64
		wantDraw100 int64
	}{
		{"k1", 0, siteWork, 0, 0, 0, false, 0x264d986499a1a1fa, 0x0fb3474a976f0acf, 6},
		{"k2", 1, siteTrade, 5, 17, 0, false, 0xe87e37a8f2ed3a12, 0x46c2cfef3d61d69c, 27},
		{"k3", 42, siteClearing, 100, 2, 3, true, 0x366090c9bf387982, 0xfe2c271dd54ed665, 99},
		{"k4", 42, siteOutcome, 100, 17, 3, true, 0x30967dc8ee4ec36e, 0xd5b237e50696f3bf, 83},
		{"k5", 1<<64 - 1, siteConstruction, 0, 4, 0, false, 0x64d9772cd475bf00, 0x4dd6b1aa70460b76, 30},
		{"k6", 7, siteCreation, -1, 12, 0, false, 0x1b6b4a321827daf5, 0x240b83bc1e00988b, 14},
		{"k7", 7, siteSpawner, 3, -1, 0, false, 0xeeeaa6fd69c7b339, 0x3355b2e3c0d174dd, 20},
		{"k8", 7, siteSpawner, -1, -1, 0, false, 0xf5455bb2a283bc00, 0x85eba190e6f38b27, 52},
		{"k9", 1 << 63, siteTrade, 1 << 40, 1<<40 + 1, 0, false, 0x3c277642ea207186, 0x510c6f3a642bcd9d, 31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStream(tt.seed)
			key := func() {
				if tt.pair {
					s.keyedPair(tt.site, tt.tick, tt.id, tt.sub)
				} else {
					s.keyed(tt.site, tt.tick, tt.id)
				}
			}
			key()
			if w0, w1 := s.word(), s.word(); w0 != tt.w0 || w1 != tt.w1 {
				t.Errorf("%s words = %016x %016x, want %016x %016x", tt.name, w0, w1, tt.w0, tt.w1)
			}
			key()
			if got := s.draw(1); got != 0 {
				t.Errorf("%s draw(1) = %d, want 0", tt.name, got)
			}
			// The draw of bound 1 consumed the first word, so this one sees the second.
			if got := s.draw(100); got != tt.wantDraw100 {
				t.Errorf("%s draw(100) after draw(1) = %d, want %d", tt.name, got, tt.wantDraw100)
			}
		})
	}
}

// TestStream_Draw verifies the world's bounded draw: the high word of the
// source word times the bound, in [0, n), one word per draw whatever the
// bound, and a range as lo + draw(hi - lo + 1).
//
// Requirement: FR-WLD-061
func TestStream_Draw(t *testing.T) {
	t.Parallel()
	// Key k1: words 264d986499a1a1fa then 0fb3474a976f0acf (spec-world.md 7.1.5 b).
	const w0, w1 uint64 = 0x264d986499a1a1fa, 0x0fb3474a976f0acf
	half := uint64(1) << 63 // lo of the whole range, as uint64
	hiOf := func(w uint64, n int64) int64 { h, _ := bits.Mul64(w, uint64(n)); return int64(h) }
	tests := []struct {
		name string
		take func(s *stream) int64
		want int64 // for the first word w0
	}{
		{"bound 1", func(s *stream) int64 { return s.draw(1) }, 0},
		{"bound 100", func(s *stream) int64 { return s.draw(100) }, hiOf(w0, 100)},
		{"bound 2^62", func(s *stream) int64 { return s.draw(1 << 62) }, hiOf(w0, 1<<62)},
		{"bound max", func(s *stream) int64 { return s.draw(1<<63 - 1) }, hiOf(w0, 1<<63-1)},
		{"bound 0 has no range", func(s *stream) int64 { return s.draw(0) }, 0},
		{"negative bound has no range", func(s *stream) int64 { return s.draw(-7) }, 0},
		{"range 5 to 9", func(s *stream) int64 { return s.drawRange(5, 9) }, 5 + hiOf(w0, 5)},
		{"range of one value", func(s *stream) int64 { return s.drawRange(-3, -3) }, -3},
		{"range below zero", func(s *stream) int64 { return s.drawRange(-10, -1) }, -10 + hiOf(w0, 10)},
		{"empty range returns lo", func(s *stream) int64 { return s.drawRange(9, 5) }, 9},
		{"whole int64 range", func(s *stream) int64 { return s.drawRange(-1<<63, 1<<63-1) }, int64(w0 + half)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStream(0)
			s.keyed(siteWork, 0, 0)
			if got := tt.take(s); got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, got, tt.want)
			}
			// Every draw takes exactly one word, whatever the bound.
			if got := s.word(); got != w1 {
				t.Errorf("%s: next word = %016x, want the second word %016x (one word consumed)", tt.name, got, w1)
			}
		})
	}
}

// TestStream_StreamsAreIndependent verifies that two streams of different
// site, tick, id or sub do not depend on each other's draw count: the stream
// of A is the same whether B was drawn from, and for how many words, before
// or after it. It also checks that the keys differ from one another.
//
// Requirement: FR-WLD-069
func TestStream_StreamsAreIndependent(t *testing.T) {
	t.Parallel()
	type key struct {
		site          uint64
		tick, id, sub int64
	}
	a := key{siteTrade, 7, 3, 0}
	others := []struct {
		name string
		k    key
	}{
		{"other site", key{siteWork, 7, 3, 0}},
		{"other tick", key{siteTrade, 8, 3, 0}},
		{"other id", key{siteTrade, 7, 4, 0}},
		{"other sub", key{siteTrade, 7, 3, 1}},
	}
	take := func(s *stream, k key, n int) []uint64 {
		s.keyedPair(k.site, k.tick, k.id, k.sub)
		out := make([]uint64, n)
		for i := range out {
			out[i] = s.word()
		}
		return out
	}
	for _, tt := range others {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStream(42)
			alone := take(s, a, 4)
			for _, drawn := range []int{0, 1, 17, 1000} {
				// Draw `drawn` words from the other stream, then take A again.
				take(s, tt.k, drawn)
				again := take(s, a, 4)
				for i := range alone {
					if again[i] != alone[i] {
						t.Fatalf("stream %+v word %d = %016x after %d words of %+v, want %016x", a, i, again[i], drawn, tt.k, alone[i])
					}
				}
			}
			if other := take(s, tt.k, 1); other[0] == alone[0] {
				t.Errorf("streams %+v and %+v share the first word %016x", a, tt.k, other[0])
			}
		})
	}
}

// TestReseed_Vectors verifies the pure reseed function seed' =
// finalise(seed xor finalise(salt)) and the first word of the stream
// (siteWork, 0, 0) under the new seed, which is what a restore with a reseed
// would draw first. Seed 0 with salt 0 is the identity (note N-215).
//
// Requirement: FR-CHK-070, FR-CHK-071
func TestReseed_Vectors(t *testing.T) {
	t.Parallel()
	// Source: spec-checkpoint.md 6.8.2 and 17.5, models/chk/worked.py.
	const max = 1<<64 - 1
	tests := []struct {
		seed, salt, want, firstWord uint64
	}{
		{0, 0, 0x0000000000000000, 0x264d986499a1a1fa},
		{0, 1, 0x7ab40e090f363a7d, 0x9fea3673fd1e605f},
		{0, 2, 0x52ead7e36ea7fea8, 0x73a2087f8eae98d5},
		{0, 42, 0x97ea87f7e45c00a5, 0x1b4989fe2e01283b},
		{1, 0, 0x5692161d100b05e5, 0x3a90ed51beb4e41d},
		{1, 1, 0x83ec686c1600460a, 0xa5d0a8b51be8378f},
		{1, 2, 0x3ec2d42f3a45cc6e, 0x47f38e4d9854e8fd},
		{1, 42, 0x901883d2427428f4, 0x00c073c4d5d6c31a},
		{42, 0, 0xa759ea27d4727622, 0xe3082397f3bc8ba6},
		{42, 1, 0xc2a6eebdf3976ad0, 0xcd5e38f87f0ec634},
		{42, 2, 0x7bfa87c92aa0cff0, 0x5b73a5b88ec53d5a},
		{42, 42, 0x36cdb0593534a37b, 0x472c27560611be0f},
		{max, 0, 0xb4d055fcf2cbbd7b, 0x9cbc3a9d395aed3d},
		{max, 1, 0xf3759af4852e5904, 0xe6b61a2920e28083},
		{max, 2, 0xab47721b00f39ddb, 0x649e9d647ec06743},
		{max, 42, 0xeab06b8e3b2074d7, 0x71829937eb038fa1},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := reseed(tt.seed, tt.salt)
			if got != tt.want {
				t.Fatalf("reseed(%d, %d) = %016x, want %016x", tt.seed, tt.salt, got, tt.want)
			}
			s := newStream(got)
			s.keyed(siteWork, 0, 0)
			if w := s.word(); w != tt.firstWord {
				t.Errorf("first word of (siteWork, 0, 0) under reseed(%d, %d) = %016x, want %016x", tt.seed, tt.salt, w, tt.firstWord)
			}
		})
	}
}

// TestReseed_Composes verifies that a second reseed is a reseed of the first
// result, not a function of the original seed alone.
//
// Requirement: FR-CHK-070
func TestReseed_Composes(t *testing.T) {
	t.Parallel()
	once := reseed(42, 1)
	twice := reseed(once, 2)
	if twice != reseed(reseed(42, 1), 2) || twice == reseed(42, 2) || twice == once {
		t.Errorf("reseed(reseed(42, 1), 2) = %016x, once = %016x, reseed(42, 2) = %016x: want a composition distinct from both", twice, once, reseed(42, 2))
	}
}

// TestSiteConstants_Guard verifies the literal site constants: pairwise at
// least 2^32 apart (modulo 2^64), and each the first eight bytes, big-endian,
// of SHA-256 of "economy-sim/site/" and its Go name (the record of origin of
// spec-world.md 7.1.4).
//
// Guard: the first fold step adds the site to the seed, so (seed a, site X)
// and (seed b, site Y) are one stream iff a - b = Y - X; a site near another
// or near zero would make two small seeds collide across sites. A module that
// appends a site must append it here too.
//
// Requirement: FR-WLD-067
func TestSiteConstants_Guard(t *testing.T) {
	t.Parallel()
	sites := []struct {
		name  string
		value uint64
	}{
		{"siteConstruction", siteConstruction},
		{"siteWork", siteWork},
		{"siteTrade", siteTrade},
		{"siteClearing", siteClearing},
		{"siteOutcome", siteOutcome},
		{"siteSpawner", siteSpawner},
		{"siteCreation", siteCreation},
	}
	for i, a := range sites {
		sum := sha256.Sum256([]byte("economy-sim/site/" + a.name))
		if got := binary.BigEndian.Uint64(sum[:8]); got != a.value {
			t.Errorf("%s = %016x, want %016x (first 8 bytes of SHA-256 of its name)", a.name, a.value, got)
		}
		for _, b := range sites[i+1:] {
			d := a.value - b.value
			if d > 1<<63 {
				d = -d
			}
			if d < 1<<32 {
				t.Errorf("%s and %s differ by %d, want at least 2^32", a.name, b.name, d)
			}
		}
	}
}

// FuzzStream_Draw checks that the keyed reseed, the bounded draw and the
// range draw never panic and stay in range for every key and bound.
//
// Requirement: NFR-WLD-005
func FuzzStream_Draw(f *testing.F) {
	f.Add(uint64(0), uint64(siteWork), int64(0), int64(0), int64(1), int64(0), int64(0))
	f.Add(uint64(1<<64-1), uint64(siteClearing), int64(-1), int64(1<<63-1), int64(1<<63-1), int64(-1<<63), int64(1<<63-1))
	f.Add(uint64(42), uint64(0), int64(100), int64(17), int64(0), int64(5), int64(-5))
	f.Add(uint64(7), uint64(siteSpawner), int64(3), int64(-1), int64(-1<<63), int64(-3), int64(-3))
	f.Fuzz(func(t *testing.T, seed, site uint64, tick, id, n, lo, hi int64) {
		s := newStream(seed)
		s.keyed(site, tick, id)
		d := s.draw(n)
		switch {
		case n >= 1 && (d < 0 || d >= n):
			t.Fatalf("draw(%d) = %d under seed %d site %d tick %d id %d, want [0, %d)", n, d, seed, site, tick, id, n)
		case n < 1 && d != 0:
			t.Fatalf("draw(%d) = %d, want 0", n, d)
		}
		s.keyedPair(site, tick, id, n)
		r := s.drawRange(lo, hi)
		switch {
		case lo <= hi && (r < lo || r > hi):
			t.Fatalf("drawRange(%d, %d) = %d, want within the range", lo, hi, r)
		case lo > hi && r != lo:
			t.Fatalf("drawRange(%d, %d) = %d, want lo for an empty range", lo, hi, r)
		}
		// One word per draw: after the draw of n, the next word is the stream's
		// second word, whatever n was.
		s.keyed(site, tick, id)
		s.word()
		w1 := s.word()
		s.keyed(site, tick, id)
		s.draw(n)
		if got := s.word(); got != w1 {
			t.Fatalf("after draw(%d) next word = %016x, want the second word %016x", n, got, w1)
		}
	})
}

// TestStream_KeyedDrawNoAllocation verifies that a keyed reseed followed by a
// draw allocates nothing, for a three-word and a four-word key, so a site
// call costs no garbage (spec-world.md 7.1.3, NFR-WLD-001).
//
// Requirement: NFR-WLD-001
func TestStream_KeyedDrawNoAllocation(t *testing.T) {
	// No t.Parallel: AllocsPerRun must not run beside other allocating tests.
	s := newStream(42)
	var sink int64
	tests := []struct {
		name string
		run  func()
	}{
		{"three words", func() { s.keyed(siteWork, 100, 17); sink += s.draw(100) }},
		{"four words", func() { s.keyedPair(siteClearing, 100, 2, 3); sink += s.drawRange(-5, 5) }},
		{"through the Rand", func() { s.keyed(siteTrade, 1, 2); sink += s.rnd.Int64N(1000) }},
	}
	for _, tt := range tests {
		if n := testing.AllocsPerRun(100, tt.run); n != 0 {
			t.Errorf("%s: keyed reseed and draw allocated %v times, want 0 (sink %d)", tt.name, n, sink)
		}
	}
}

// BenchmarkStream_KeyedDraw reports the cost of one site call: a keyed reseed
// and one bounded draw.
func BenchmarkStream_KeyedDraw(b *testing.B) {
	s := newStream(42)
	b.ReportAllocs()
	var sink int64
	var tick int64
	for b.Loop() {
		tick++
		s.keyed(siteTrade, tick, 17)
		sink += s.draw(100)
	}
	_ = sink
}
