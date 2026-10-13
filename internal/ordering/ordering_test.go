package ordering_test

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestFinalise_SeedVectors verifies the finaliser against the four seed rows
// of spec-world.md 7.1.3: for seed s the PCG state is (hi, lo) with
// hi = Finalise(s + Increment) and lo = Finalise(s + 2*Increment), modulo 2^64.
//
// Requirement: FR-WLD-060
func TestFinalise_SeedVectors(t *testing.T) {
	t.Parallel()
	// Source: the table of spec-world.md 7.1.3, produced with Go 1.25 and
	// reproduced by doc/specs/models/wld/seed.py.
	tests := []struct {
		name   string
		seed   uint64
		hi, lo uint64
	}{
		{"seed 0", 0, 0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4},
		{"seed 1", 1, 0x910a2dec89025cc1, 0xbeeb8da1658eec67},
		{"seed 2", 2, 0x975835de1c9756ce, 0xbfc846100bfc1e42},
		{"seed 42", 42, 0xbdd732262feb6e95, 0x28efe333b266f103},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.seed + ordering.Increment
			hi := ordering.Finalise(s)
			s += ordering.Increment
			lo := ordering.Finalise(s)
			if hi != tt.hi || lo != tt.lo {
				t.Errorf("Finalise chain of seed %d = (%016x, %016x), want (%016x, %016x)", tt.seed, hi, lo, tt.hi, tt.lo)
			}
		})
	}
}

// TestWeyl2_Vectors verifies Weyl2 on the values the specs state.
//
// Requirement: FR-WLD-080
func TestWeyl2_Vectors(t *testing.T) {
	t.Parallel()
	// Weyl2(id) = id * 0x6A09E667 mod 2^32, and 0x6A09E667 = 1779033703:
	// id 1 gives 1779033703, id 2 gives 3558067406 (< 2^32 = 4294967296),
	// and id 17 gives 17 * 1779033703 = 30243572951 = 7 * 2^32 + 178801879.
	tests := []struct {
		id   int64
		want uint32
	}{
		{0, 0},
		{1, 1779033703},
		{2, 3558067406},
		{17, 178801879},
	}
	for _, tt := range tests {
		if got := ordering.Weyl2(tt.id); got != tt.want {
			t.Errorf("Weyl2(%d) = %d, want %d", tt.id, got, tt.want)
		}
	}
}

// TestWeylAndBucket_WorkedVectors verifies Weyl32 and Bucket for the ids 10 to
// 15 against doc/specs/models/wld/worked.out.
//
// Requirement: FR-WLD-080
func TestWeylAndBucket_WorkedVectors(t *testing.T) {
	t.Parallel()
	// Source: "Weyl32 and bucket for ids 10..15" of worked.out.
	tests := []struct {
		id       int64
		weyl32   uint32
		bucket3  int64
		bucket10 int64
	}{
		{10, 774553914, 0, 1},
		{11, 3428989683, 1, 5},
		{12, 1788458156, 2, 9},
		{13, 147926629, 1, 3},
		{14, 2802362398, 2, 7},
		{15, 1161830871, 0, 2},
	}
	// Weyl2(17) = 178801879 (see TestWeyl2_Vectors), so
	// Bucket(17, 10) = 178801879 * 10 >> 32 = 1788018790 >> 32 = 0.
	if got := ordering.Bucket(17, 10); got != 0 {
		t.Errorf("Bucket(17, 10) = %d, want 0", got)
	}
	for _, tt := range tests {
		if got := ordering.Weyl32(tt.id); got != tt.weyl32 {
			t.Errorf("Weyl32(%d) = %d, want %d", tt.id, got, tt.weyl32)
		}
		if got := ordering.Bucket(tt.id, 3); got != tt.bucket3 {
			t.Errorf("Bucket(%d, 3) = %d, want %d", tt.id, got, tt.bucket3)
		}
		if got := ordering.Bucket(tt.id, 10); got != tt.bucket10 {
			t.Errorf("Bucket(%d, 10) = %d, want %d", tt.id, got, tt.bucket10)
		}
	}
}

// TestBucket_RangeAndRotationLoad verifies that Bucket stays in [0, p) for
// every period from 1 to 1 000 and a spread of ids, and that a rotation of
// six consecutive ids over a period of 3 reviews two agents a tick, as
// worked.out shows.
//
// Requirement: FR-WLD-080
func TestBucket_RangeAndRotationLoad(t *testing.T) {
	t.Parallel()
	t.Run("range for every period", func(t *testing.T) {
		t.Parallel()
		ids := []int64{-1 << 63, -12345, -1, 0, 1, 2, 17, 99, 12345, 1<<31 - 1, 1 << 31, 1<<32 + 5, 1<<63 - 1}
		for id := int64(100); id < 300; id++ {
			ids = append(ids, id)
		}
		for p := int64(1); p <= 1000; p++ {
			for _, id := range ids {
				if got := ordering.Bucket(id, p); got < 0 || got >= p {
					t.Fatalf("Bucket(%d, %d) = %d, want a value in [0, %d)", id, p, got, p)
				}
			}
		}
	})
	t.Run("period below one", func(t *testing.T) {
		t.Parallel()
		for _, p := range []int64{0, -1, -1 << 63} {
			if got := ordering.Bucket(17, p); got != 0 {
				t.Errorf("Bucket(17, %d) = %d, want 0", p, got)
			}
		}
	})
	t.Run("rotation load of six agents over three ticks", func(t *testing.T) {
		t.Parallel()
		// worked.out: "reviews per tick, ids 10..15, ticks 0..8: [2 x 9]".
		for tick := int64(0); tick < 9; tick++ {
			n := 0
			for id := int64(10); id <= 15; id++ {
				if (tick+ordering.Bucket(id, 3))%3 == 0 {
					n++
				}
			}
			if n != 2 {
				t.Errorf("reviews at tick %d = %d, want 2", tick, n)
			}
		}
	})
}

// TestBucket_LargePeriod verifies the exact shift for a period that makes the
// product wider than 64 bits.
//
// Guard: a plain (Weyl2 * p) >> 32 in uint64 wraps for p above 2^32 and would
// put the bucket outside [0, p); the 128-bit form keeps it exact.
func TestBucket_LargePeriod(t *testing.T) {
	t.Parallel()
	for _, p := range []int64{1 << 32, 1<<40 + 7, 1<<63 - 1} {
		for _, id := range []int64{1, 17, 12345} {
			want := new(big.Int).Mul(big.NewInt(int64(ordering.Weyl2(id))), big.NewInt(p))
			want.Rsh(want, 32)
			if got := ordering.Bucket(id, p); got != want.Int64() {
				t.Errorf("Bucket(%d, %d) = %d, want %d", id, p, got, want.Int64())
			}
		}
	}
}

// TestWeyl32_Evenness verifies that Weyl32 spreads consecutive ids evenly:
// 1 000 ids dropped into 10 equal bins by the high bits give counts within a
// few of the mean.
//
// Guard: spec-world.md 9.1.1 relies on Weyl32 being even for the death carry
// and the migration margin; a wrong multiplier would bunch them on a few ticks
// and nothing else would notice.
func TestWeyl32_Evenness(t *testing.T) {
	t.Parallel()
	// Weyl32 is a Kronecker sequence with the golden ratio, whose gaps take at
	// most three lengths (three-distance theorem), so a bin of width 1/10
	// holds 100 +- a few of 1 000 consecutive ids. The bound of 5 is loose; a
	// broken multiplier (an even one, for example) breaks it by a wide margin.
	for _, start := range []int64{0, 100, 12345} {
		var bins [10]int
		for id := start; id < start+1000; id++ {
			bins[uint64(ordering.Weyl32(id))*10>>32]++
		}
		for b, n := range bins {
			if n < 95 || n > 105 {
				t.Errorf("Weyl32 over ids %d..%d: bin %d holds %d, want 95 to 105 (bins %v)", start, start+999, b, n, bins)
			}
		}
	}
}

// FuzzBucket checks that Bucket never panics and stays in [0, p) for every id
// and every period of at least 1, and equals the exact big-integer value.
//
// Requirement: NFR-WLD-005
func FuzzBucket(f *testing.F) {
	f.Add(int64(17), int64(10))
	f.Add(int64(0), int64(1))
	f.Add(int64(-1), int64(1000))
	f.Add(int64(1<<63-1), int64(1<<63-1))
	f.Fuzz(func(t *testing.T, id, p int64) {
		got := ordering.Bucket(id, p)
		if p < 1 {
			if got != 0 {
				t.Fatalf("Bucket(%d, %d) = %d, want 0", id, p, got)
			}
			return
		}
		if got < 0 || got >= p {
			t.Fatalf("Bucket(%d, %d) = %d, want a value in [0, %d)", id, p, got, p)
		}
		want := new(big.Int).Mul(new(big.Int).SetUint64(uint64(ordering.Weyl2(id))), big.NewInt(p))
		want.Rsh(want, 32)
		if got != want.Int64() {
			t.Fatalf("Bucket(%d, %d) = %d, want %d", id, p, got, want.Int64())
		}
	})
}

// TestStreamKey_Vectors verifies the stream key against rows k1 to k9 of
// spec-world.md 7.1.5 (b): hi and lo, and the first two words of a
// math/rand/v2 PCG constructed from them (the generator is specified, so the
// words are the check).
//
// Requirement: FR-WLD-060, FR-WLD-068
func TestStreamKey_Vectors(t *testing.T) {
	t.Parallel()
	// Source: doc/specs/models/wld/seed.py, table (b) of spec-world.md 7.1.5.
	// The site literals are the constants of spec-world.md 7.1.4.
	const (
		siteConstruction uint64 = 0xC103123756D13F20
		siteWork         uint64 = 0xDE594838D3779A37
		siteTrade        uint64 = 0xFC14CD10E8A8EF43
		siteClearing     uint64 = 0xD364E6AE73E1313F
		siteOutcome      uint64 = 0xD9CC0A517EB0D9F5
		siteSpawner      uint64 = 0xBE0B1D169200E9FF
		siteCreation     uint64 = 0x9B8FFA303CB545DC
	)
	tests := []struct {
		name       string
		seed, site uint64
		tick, id   int64
		sub        int64
		hasSub     bool
		hi, lo     uint64
		w0, w1     uint64
	}{
		{name: "k1", seed: 0, site: siteWork, hi: 0x79ae1abc6fb3c555, lo: 0x32a6affd0dbf0521, w0: 0x264d986499a1a1fa, w1: 0x0fb3474a976f0acf},
		{name: "k2", seed: 1, site: siteTrade, tick: 5, id: 17, hi: 0xfba1decca480b034, lo: 0x59c9e7d8138ce36b, w0: 0xe87e37a8f2ed3a12, w1: 0x46c2cfef3d61d69c},
		{name: "k3", seed: 42, site: siteClearing, tick: 100, id: 2, sub: 3, hasSub: true, hi: 0x92cfbdaff0085569, lo: 0xdf1d7be0b72c91fe, w0: 0x366090c9bf387982, w1: 0xfe2c271dd54ed665},
		{name: "k4", seed: 42, site: siteOutcome, tick: 100, id: 17, sub: 3, hasSub: true, hi: 0x4c1a052c0514495c, lo: 0x1ebc9cbf8410ab07, w0: 0x30967dc8ee4ec36e, w1: 0xd5b237e50696f3bf},
		{name: "k5", seed: 1<<64 - 1, site: siteConstruction, id: 4, hi: 0x7e12e8d9e610f1f3, lo: 0x11e76e20e84424df, w0: 0x64d9772cd475bf00, w1: 0x4dd6b1aa70460b76},
		{name: "k6 negative tick", seed: 7, site: siteCreation, tick: -1, id: 12, hi: 0x290ef3893fee12ef, lo: 0xaa22a25ddaace732, w0: 0x1b6b4a321827daf5, w1: 0x240b83bc1e00988b},
		{name: "k7 negative id", seed: 7, site: siteSpawner, tick: 3, id: -1, hi: 0x11f3bdf788e5ffb9, lo: 0x4a115493bf1bf035, w0: 0xeeeaa6fd69c7b339, w1: 0x3355b2e3c0d174dd},
		{name: "k8 both negative", seed: 7, site: siteSpawner, tick: -1, id: -1, hi: 0xcc1b665a6f86d8ba, lo: 0x6901bc968a06bc0f, w0: 0xf5455bb2a283bc00, w1: 0x85eba190e6f38b27},
		{name: "k9 above 2^32", seed: 1 << 63, site: siteTrade, tick: 1 << 40, id: 1<<40 + 1, hi: 0x87e7225b79ecaea7, lo: 0xca410d7c919c53eb, w0: 0x3c277642ea207186, w1: 0x510c6f3a642bcd9d},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var hi, lo uint64
			if tt.hasSub {
				hi, lo = ordering.StreamKeySub(tt.seed, tt.site, tt.tick, tt.id, tt.sub)
			} else {
				hi, lo = ordering.StreamKey(tt.seed, tt.site, tt.tick, tt.id)
			}
			if hi != tt.hi || lo != tt.lo {
				t.Fatalf("stream key (seed %d, site %x, tick %d, id %d, sub %d) = (%016x, %016x), want (%016x, %016x)",
					tt.seed, tt.site, tt.tick, tt.id, tt.sub, hi, lo, tt.hi, tt.lo)
			}
			p := rand.NewPCG(hi, lo)
			if w0, w1 := p.Uint64(), p.Uint64(); w0 != tt.w0 || w1 != tt.w1 {
				t.Errorf("PCG words of %s = (%016x, %016x), want (%016x, %016x)", tt.name, w0, w1, tt.w0, tt.w1)
			}
		})
	}
}

// TestStreamKey_BareSeedWords verifies the four bare-seed rows of
// spec-world.md 7.1.5 (a): the first two words of rand.NewPCG(hi, lo).
//
// Requirement: FR-WLD-060
func TestStreamKey_BareSeedWords(t *testing.T) {
	t.Parallel()
	// Source: doc/specs/models/wld/seed.py, table (a) of spec-world.md 7.1.5.
	tests := []struct {
		hi, lo, w0, w1 uint64
	}{
		{0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4, 0xb18c97843f57bb91, 0xd7046f21f79eba31},
		{0x910a2dec89025cc1, 0xbeeb8da1658eec67, 0xaf9d32bf75779748, 0xb7672df032bf0473},
		{0x975835de1c9756ce, 0xbfc846100bfc1e42, 0xa28db69c539314fb, 0xfe57a921339b50b9},
		{0xbdd732262feb6e95, 0x28efe333b266f103, 0x61c88529c9612c1b, 0x2608d8a3075aa168},
	}
	for _, tt := range tests {
		p := rand.NewPCG(tt.hi, tt.lo)
		if w0, w1 := p.Uint64(), p.Uint64(); w0 != tt.w0 || w1 != tt.w1 {
			t.Errorf("NewPCG(%016x, %016x) words = (%016x, %016x), want (%016x, %016x)", tt.hi, tt.lo, w0, w1, tt.w0, tt.w1)
		}
	}
}

// TestKey_Vectors verifies K(t, id) against table (c) of spec-world.md
// 7.1.5, including negative arguments and K(-1, 0) = 0.
//
// Requirement: FR-WLD-091, FR-RES-019
func TestKey_Vectors(t *testing.T) {
	t.Parallel()
	// Source: doc/specs/models/wld/seed.py (table c); K = Finalise(t*gamma +
	// id + gamma). K(-1, 0) = Finalise(-gamma + gamma) = Finalise(0) = 0.
	tests := []struct {
		t, id int64
		want  uint64
	}{
		{0, 0, 0xe220a8397b1dcdaf},
		{1, 0, 0x6e789e6aa1b965f4},
		{0, 1, 0x910a2dec89025cc1},
		{5, 17, 0x14db9121e4c732ea},
		{100, 3, 0xa854cdf21653e5e5},
		{1 << 40, 123456789, 0x2cd5718615e1be22},
		{-1, -1, 0xb4d055fcf2cbbd7b},
		{-1, 0, 0},
	}
	for _, tt := range tests {
		if got := ordering.Key(tt.t, tt.id); got != tt.want {
			t.Errorf("Key(%d, %d) = %016x, want %016x", tt.t, tt.id, got, tt.want)
		}
	}
}

// TestKey_TieOrder verifies the use FR-WLD-090 names: for distinct ids at one
// tick the keys are distinct (Finalise is a bijection and id enters linearly),
// so a tie broken by K and then the id is a total order.
//
// Requirement: FR-WLD-090
func TestKey_TieOrder(t *testing.T) {
	t.Parallel()
	seen := make(map[uint64]int64)
	for id := int64(-50); id < 5000; id++ {
		k := ordering.Key(7, id)
		if prev, ok := seen[k]; ok {
			t.Fatalf("Key(7, %d) = Key(7, %d) = %016x, want distinct keys", id, prev, k)
		}
		seen[k] = id
	}
}

// TestSiteConstants_Spacing verifies that two site constants of spec-world.md
// 7.1.4 differ by at least 2^32 in both directions.
//
// Guard: the first fold step adds the site to the seed, so (seed a, site X)
// and (seed b, site Y) are one stream iff a - b = Y - X; a small gap would let
// two small seeds collide across sites. The constants live in random.go in a
// later phase, where this guard moves with them; here the literals are
// repeated from the spec.
func TestSiteConstants_Spacing(t *testing.T) {
	t.Parallel()
	sites := []uint64{0xC103123756D13F20, 0xDE594838D3779A37, 0xFC14CD10E8A8EF43,
		0xD364E6AE73E1313F, 0xD9CC0A517EB0D9F5, 0xBE0B1D169200E9FF, 0x9B8FFA303CB545DC}
	for i, a := range sites {
		for _, b := range sites[i+1:] {
			d := a - b
			if d > 1<<63 {
				d = -d
			}
			if d < 1<<32 {
				t.Errorf("sites %016x and %016x differ by %d, want at least 2^32", a, b, d)
			}
		}
	}
}

// TestStreamKey_NoAllocation verifies that a reseed from a key plus two draws
// allocates nothing (spec-world.md 7.1.3).
//
// Requirement: NFR-WLD-005
func TestStreamKey_NoAllocation(t *testing.T) {
	pcg := rand.NewPCG(1, 2)
	var sink uint64
	n := testing.AllocsPerRun(100, func() {
		hi, lo := ordering.StreamKeySub(42, 0xDE594838D3779A37, 100, 17, 3)
		pcg.Seed(hi, lo)
		sink += pcg.Uint64()
	})
	if n != 0 {
		t.Errorf("reseed and draw allocated %v times, want 0 (sink %d)", n, sink)
	}
}

// FuzzKey checks that Key and the stream keys never panic and are
// deterministic for every input, and that for a fixed tick Key is a bijection
// in the id: Key is Finalise of (t*Increment + id + Increment), an addition of
// a constant modulo 2^64 followed by a bijection, so two distinct ids of one
// tick never share a key and the order by Key is a strict total order on
// them, with no tie to break. Across ticks collisions are possible by
// construction and are not asserted.
//
// Requirement: NFR-WLD-005
func FuzzKey(f *testing.F) {
	f.Add(int64(0), int64(0), uint64(0), int64(1))
	f.Add(int64(-1), int64(-1), uint64(1<<64-1), int64(-2))
	f.Add(int64(1<<63-1), int64(-1<<63), uint64(0xDE594838D3779A37), int64(1<<63-1))
	f.Add(int64(-1<<63), int64(1<<63-1), uint64(7), int64(-1<<63))
	// Ids that differ only in a high bit: a key that dropped the high bits of
	// the id would tie them.
	f.Add(int64(3), int64(5), uint64(0), int64(5+1<<16))
	f.Add(int64(3), int64(5), uint64(0), int64(5+1<<32))
	f.Fuzz(func(t *testing.T, tick, id int64, seed uint64, other int64) {
		ka, kb := ordering.Key(tick, id), ordering.Key(tick, other)
		if (ka == kb) != (id == other) {
			t.Fatalf("Key(%d, %d) = %016x, Key(%d, %d) = %016x: equal keys must mean equal ids", tick, id, ka, tick, other, kb)
		}
		if id != other && (ka < kb) == (kb < ka) {
			t.Fatalf("ids %d and %d of tick %d are not strictly ordered by Key", id, other, tick)
		}
		if a, b := ordering.Key(tick, id), ordering.Key(tick, id); a != b {
			t.Fatalf("Key(%d, %d) = %016x then %016x, want deterministic", tick, id, a, b)
		}
		h1, l1 := ordering.StreamKeySub(seed, 1, tick, id, tick^id)
		h2, l2 := ordering.StreamKeySub(seed, 1, tick, id, tick^id)
		if h1 != h2 || l1 != l2 {
			t.Fatalf("StreamKeySub(%d, 1, %d, %d) not deterministic", seed, tick, id)
		}
	})
}
