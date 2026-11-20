package economysim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// insuranceHashConfig is a Config with only the insurance module and two role blocks:
// the tests below read its bytes, not a world.
func insuranceHashConfig() Config {
	return Config{
		Roles: []Role{
			{ID: "insured", Cover: &CoverTerms{Insure: &InsureTerms{
				MinLossBP: 1_000, RuinShareBP: 3_000, MaxLoadingBP: 15_000, MinCoverBP: 2_000, StoreShareBP: 5_000, MaxStoreRateBP: 200}}},
			{ID: "underwriter", Cover: &CoverTerms{Underwrite: &UnderwriteTerms{
				LoadingBP: 12_000, StoreRateBP: 50, CoverBP: 8_000, CapBP: 5_000, ShareBP: 5_000}}},
			{ID: "zplain"},
		},
		Modules: Modules{Insurance: &Insurance{ValueCapBP: 20_000, PayoutTicks: 1, MinSum: 100, MaxPolicies: 1_000, MaxCoverRequests: 4}},
	}
}

// TestHash_InsuranceFieldIds verifies the bytes of the insurance module under id 18
// of the Modules row (after Infrastructure 17 and before Polity 19), that the pointer
// is written even when empty, that the fields have the ids spec-config.md 8.2.2 gives,
// and the bytes of a role block under id 28 of Role.
//
// Requirement: FR-INS-008, FR-CFG-091
//
// Guard: a pair out of id order, an empty module that writes nothing, or a block with
// the wrong ids would hash a configuration differently from the one the spec defines.
func TestHash_InsuranceFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }

	e := &encoder{}
	pairsModules(e, Modules{Infrastructure: &Infrastructure{}, Insurance: &Insurance{}, Polity: &PolityConfig{}})
	at := bytes.Index(e.buf, append(id(18), 0, 0))
	if at < 0 || bytes.Index(e.buf, append(id(17), 0, 0)) > at || at > bytes.Index(e.buf, append(id(19), 0, 0)) {
		t.Errorf("pairsModules(Infrastructure, Insurance, Polity) = % x, want the empty pair 0012 0000 after id 17 and before id 19", e.buf)
	}

	e = &encoder{}
	pairsInsurance(e, Insurance{ValueCapBP: 1, PayoutTicks: 2, MinSum: 3, MaxPolicies: 4, MaxCoverRequests: 5})
	want := bytes.Join([][]byte{id(1), u64(1), id(2), u64(2), id(3), u64(3), id(4), u64(4), id(5), u64(5)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsInsurance = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsCoverTerms(e, CoverTerms{
		Underwrite: &UnderwriteTerms{LoadingBP: 5, StoreRateBP: 6, CoverBP: 7, CapBP: 8, ShareBP: 9},
		Insure:     &InsureTerms{MinLossBP: 10, RuinShareBP: 11, MaxLoadingBP: 12, MinCoverBP: 13, StoreShareBP: 14, MaxStoreRateBP: 15},
	})
	want = bytes.Join([][]byte{
		id(1), id(1), u64(5), id(2), u64(6), id(3), u64(7), id(4), u64(8), id(5), u64(9), {0, 0},
		id(2), id(1), u64(10), id(2), u64(11), id(3), u64(12), id(4), u64(13), id(5), u64(14), id(6), u64(15), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsCoverTerms = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsRole(e, Role{Cover: &CoverTerms{}}, false, false)
	if !bytes.HasSuffix(e.buf, append(id(28), 0, 0)) {
		t.Errorf("pairsRole of an empty block = % x, want it to end with the empty pair 001c 0000", e.buf)
	}
}

// TestHashConfig_InsuranceOffLeavesNoTrace verifies that a configuration with the
// module off hashes as it did before the module existed: a nil module and a role
// with no block write no pair, so the hash of the all-off reference does not move.
//
// Requirement: FR-INS-001, FR-INS-008
//
// Guard: HashConfig of a Config with neither the module nor a block equals that of the
// same Config built without the fields, which is every existing hash and golden.
func TestHashConfig_InsuranceOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	off := insuranceHashConfig()
	off.Modules.Insurance = nil
	for i := range off.Roles {
		off.Roles[i].Cover = nil
	}
	e := &encoder{}
	pairsModules(e, off.Modules)
	if len(e.buf) != 0 {
		t.Errorf("pairsModules(module off) = % x, want nothing", e.buf)
	}
	if HashConfig(off) == HashConfig(insuranceHashConfig()) {
		t.Error("HashConfig of the module off = that of the module on, want different")
	}
}

// TestHash_InsuranceNilIsByteIdentical verifies, on the bytes, that a nil module and a
// role with no block add nothing to the pairs: the bytes of the Modules row with the
// module off equal those of the same row built without the field's neighbours moving,
// the bytes of a role with no block are those of a role that never had the field, the
// bytes of a role with a block are exactly those plus the block's pair at id 28 as the
// last pair, and the canonical copy of a configuration with neither keeps both nil.
//
// Requirement: FR-INS-001, FR-INS-008
//
// Guard: the pinned hash of the all-off reference only proves the case it pins. This
// proves the rule for any configuration: the module's id 18 and the block's id 28 are
// written only when present, and never as an empty pair, so every existing hash and
// checkpoint stays valid when a module is added (architecture.md 2.10.5).
func TestHash_InsuranceNilIsByteIdentical(t *testing.T) {
	t.Parallel()
	neighbours := Modules{Infrastructure: &Infrastructure{}, Polity: &PolityConfig{}}
	on := neighbours
	on.Insurance = &Insurance{}
	off, withModule := &encoder{}, &encoder{}
	pairsModules(off, neighbours)
	pairsModules(withModule, on)
	empty := []byte{0, 18, 0, 0} // id 18 and the end of an empty module
	at := bytes.Index(withModule.buf, empty)
	if at < 0 || bytes.Contains(off.buf, empty) {
		t.Fatalf("pairsModules: on = % x, off = % x, want the pair 0012 0000 only with the module on", withModule.buf, off.buf)
	}
	if rest := slices.Concat(withModule.buf[:at], withModule.buf[at+len(empty):]); !bytes.Equal(rest, off.buf) {
		t.Errorf("pairsModules(module on) less its pair = % x, want the bytes with the module off % x", rest, off.buf)
	}

	plain := Role{ID: "r", Recipes: []string{"x"}, StartMoney: 5}
	withBlock := plain
	withBlock.Cover = &CoverTerms{Insure: &InsureTerms{MinLossBP: 1}}
	p, q := &encoder{}, &encoder{}
	pairsRole(p, plain, false, false)
	pairsRole(q, withBlock, false, false)
	block := &encoder{}
	pairsCoverTerms(block, *withBlock.Cover)
	want := slices.Concat(p.buf, []byte{0, 28}, block.buf, []byte{0, 0})
	if !bytes.Equal(q.buf, want) {
		t.Errorf("pairsRole(with block) = % x, want the bytes of the role with none, then id 28, the block % x and its end", q.buf, block.buf)
	}
	if bytes.HasSuffix(p.buf, []byte{0, 28, 0, 0}) {
		t.Errorf("pairsRole(no block) = % x, want no pair at id 28", p.buf)
	}

	c := Config{Roles: []Role{plain}, Modules: neighbours}
	cp := c.canonical()
	if cp.Modules.Insurance != nil || cp.Roles[0].Cover != nil {
		t.Errorf("canonical(no insurance) has Insurance %v and Cover %v, want both nil", cp.Modules.Insurance, cp.Roles[0].Cover)
	}
	if !reflect.DeepEqual(cp.Modules, c.Modules) {
		t.Errorf("canonical(no insurance).Modules = %+v, want %+v", cp.Modules, c.Modules)
	}
}

// TestHashConfig_Insurance verifies that the module on, even empty, differs from the
// module off, that an empty block differs from none, that a block with every field
// zero differs from an empty block, that an underwriter differs from an insured, that
// every field of the module and of the blocks is hashed, and that the order the roles
// are written in changes nothing.
//
// Requirement: FR-INS-008, FR-CFG-091
func TestHashConfig_Insurance(t *testing.T) {
	t.Parallel()
	if HashConfig(Config{}) == HashConfig(Config{Modules: Modules{Insurance: &Insurance{}}}) {
		t.Error("HashConfig of the module on and empty = that of the module off, want different")
	}
	base := insuranceHashConfig()
	hash := HashConfig(base)
	changes := map[string]func(c *Config){
		"ValueCapBP":       func(c *Config) { c.Modules.Insurance.ValueCapBP++ },
		"PayoutTicks":      func(c *Config) { c.Modules.Insurance.PayoutTicks++ },
		"MinSum":           func(c *Config) { c.Modules.Insurance.MinSum++ },
		"MaxPolicies":      func(c *Config) { c.Modules.Insurance.MaxPolicies++ },
		"MaxCoverRequests": func(c *Config) { c.Modules.Insurance.MaxCoverRequests++ },
		"LoadingBP":        func(c *Config) { c.Roles[1].Cover.Underwrite.LoadingBP++ },
		"StoreRateBP":      func(c *Config) { c.Roles[1].Cover.Underwrite.StoreRateBP++ },
		"CoverBP":          func(c *Config) { c.Roles[1].Cover.Underwrite.CoverBP++ },
		"CapBP":            func(c *Config) { c.Roles[1].Cover.Underwrite.CapBP++ },
		"ShareBP":          func(c *Config) { c.Roles[1].Cover.Underwrite.ShareBP++ },
		"MinLossBP":        func(c *Config) { c.Roles[0].Cover.Insure.MinLossBP++ },
		"RuinShareBP":      func(c *Config) { c.Roles[0].Cover.Insure.RuinShareBP++ },
		"MaxLoadingBP":     func(c *Config) { c.Roles[0].Cover.Insure.MaxLoadingBP++ },
		"MinCoverBP":       func(c *Config) { c.Roles[0].Cover.Insure.MinCoverBP++ },
		"StoreShareBP":     func(c *Config) { c.Roles[0].Cover.Insure.StoreShareBP++ },
		"MaxStoreRateBP":   func(c *Config) { c.Roles[0].Cover.Insure.MaxStoreRateBP++ },
		"block removed":    func(c *Config) { c.Roles[0].Cover = nil },
		"empty block":      func(c *Config) { c.Roles[2].Cover = &CoverTerms{} },
		"insured to underwriter": func(c *Config) {
			c.Roles[1].Cover = &CoverTerms{Insure: &InsureTerms{}}
		},
		"zero underwriter": func(c *Config) { c.Roles[2].Cover = &CoverTerms{Underwrite: &UnderwriteTerms{}} },
	}
	seen := map[[32]byte]string{hash: "base"}
	for name, change := range changes {
		c := insuranceHashConfig()
		change(&c)
		h := HashConfig(c)
		if prior, dup := seen[h]; dup {
			t.Errorf("HashConfig after %q = that after %q, want every change hashed and each different", name, prior)
		}
		seen[h] = name
	}
	zero, empty := insuranceHashConfig(), insuranceHashConfig()
	zero.Roles[2].Cover = &CoverTerms{Underwrite: &UnderwriteTerms{}}
	empty.Roles[2].Cover = &CoverTerms{}
	if HashConfig(zero) == HashConfig(empty) {
		t.Error("HashConfig of an all-zero underwriter = that of an empty block, want different")
	}
	swapped := insuranceHashConfig()
	swapped.Roles[0], swapped.Roles[1] = swapped.Roles[1], swapped.Roles[0]
	if HashConfig(swapped) != hash {
		t.Error("HashConfig changed with the order of the roles, want canonical order")
	}
}

// TestCanonical_InsuranceCopiesBlocks verifies that the canonical copy holds blocks of
// its own: changing the consumer's Config after canonical moves nothing in the copy.
//
// Requirement: FR-INS-008
func TestCanonical_InsuranceCopiesBlocks(t *testing.T) {
	t.Parallel()
	c := insuranceHashConfig()
	cp := c.canonical()
	c.Modules.Insurance.MaxPolicies = 1
	c.Roles[1].Cover.Underwrite.CapBP = 1
	c.Roles[0].Cover.Insure.MinCoverBP = 1
	if cp.Modules.Insurance.MaxPolicies != 1_000 || cp.Roles[1].Cover.Underwrite.CapBP != 5_000 || cp.Roles[0].Cover.Insure.MinCoverBP != 2_000 {
		t.Errorf("canonical copy changed with the original: %+v %+v %+v", *cp.Modules.Insurance, *cp.Roles[1].Cover.Underwrite, *cp.Roles[0].Cover.Insure)
	}
	if (*Insurance)(nil).canonical() != nil || (*CoverTerms)(nil).canonical() != nil {
		t.Error("canonical of a nil module or block is not nil")
	}
}

// TestInsurance_LedgerReasonsDeclaredWhenOn verifies that with the module on the ledger
// declares the premium and payout reasons, so an agent-to-agent transfer with either
// is allowed, and that with the module off the same transfer is refused with
// ErrReasonMismatch because the reason is not declared.
//
// Requirement: FR-INS-002, FR-INS-003
func TestInsurance_LedgerReasonsDeclaredWhenOn(t *testing.T) {
	t.Parallel()
	for _, on := range []bool{true, false} {
		cfg := Config{Regions: []Region{{ID: "north"}}}
		if on {
			cfg.Modules.Insurance = &Insurance{ValueCapBP: 1, PayoutTicks: 1, MinSum: 1, MaxPolicies: 1, MaxCoverRequests: 1}
		}
		l := mustLedger(t, cfg)
		a, err := l.Open(ledger.KindAgent, 0, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		b, err := l.Open(ledger.KindAgent, 1, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if err := l.EndConstruction(); err != nil {
			t.Fatalf("EndConstruction: %v", err)
		}
		from, ok := l.Boundary(ledger.Initial, ledger.Money)
		if !ok {
			t.Fatal("no Initial boundary account for money")
		}
		if err := l.Transfer(from, a, 1_000, ledger.Initial); err != nil {
			t.Fatalf("Transfer: %v", err)
		}
		for _, r := range []ledger.Reason{ledger.Premium, ledger.Payout} {
			err := l.Transfer(a, b, 1, r)
			if (err == nil) != on {
				t.Errorf("Transfer with reason %v and the module on = %v: error %v, want an error only with it off", r, on, err)
			}
		}
	}
}
