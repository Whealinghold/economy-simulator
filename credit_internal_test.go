package economysim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// creditHashConfig is a Config with only the credit module and two role blocks: the
// tests below read its bytes, not a world.
func creditHashConfig() Config {
	return Config{
		Roles: []Role{
			{ID: "borrower", Credit: &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}},
			{ID: "lender", Credit: &CreditTerms{Lend: &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}},
			{ID: "plain"},
		},
		Modules: Modules{Credit: &Credit{MinPrincipal: 5_000, ShieldTicks: 2, WriteOffTicks: 7, MaxClaims: 1_000}},
	}
}

// TestHash_CreditFieldIds verifies the bytes of the credit module under id 15 of the
// Modules row (after Employment 13 and before Polity 19), that the pointer is
// written even when empty, that the fields have the ids spec-config.md 8.2.2 gives,
// and the bytes of a role block under id 25 of Role.
//
// Requirement: FR-CRD-012, FR-CFG-088
//
// Guard: a pair out of id order, an empty module that writes nothing, or a block with
// the wrong ids would hash a configuration differently from the one the spec defines.
func TestHash_CreditFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }

	e := &encoder{}
	pairsModules(e, Modules{Employment: &Employment{}, Credit: &Credit{}, Polity: &PolityConfig{}})
	at := bytes.Index(e.buf, append(id(15), 0, 0))
	if at < 0 || bytes.Index(e.buf, append(id(13), 0, 0)) > at || at > bytes.Index(e.buf, append(id(19), 0, 0)) {
		t.Errorf("pairsModules(Employment, Credit, Polity) = % x, want the empty pair 000f 0000 after id 13 and before id 19", e.buf)
	}

	e = &encoder{}
	pairsCredit(e, Credit{MinPrincipal: 1, ShieldTicks: 2, WriteOffTicks: 3, MaxClaims: 4})
	want := bytes.Join([][]byte{id(1), u64(1), id(2), u64(2), id(3), u64(3), id(4), u64(4)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsCredit = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsCreditTerms(e, CreditTerms{
		Lend:   &LendTerms{RateBP: 5, MaxTermTicks: 6, KeepTicks: 7, ShareBP: 8},
		Borrow: &BorrowTerms{MaxRateBP: 9, MaxDebtTicks: 10, SlackTicks: 11},
	})
	want = bytes.Join([][]byte{
		id(1), id(1), u64(5), id(2), u64(6), id(3), u64(7), id(4), u64(8), {0, 0},
		id(2), id(1), u64(9), id(2), u64(10), id(3), u64(11), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsCreditTerms = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsRole(e, Role{Credit: &CreditTerms{}}, false, false)
	if !bytes.HasSuffix(e.buf, append(id(25), 0, 0)) {
		t.Errorf("pairsRole of an empty block = % x, want it to end with the empty pair 0019 0000", e.buf)
	}
}

// TestHashConfig_Credit verifies that the module on, even empty, differs from the
// module off, that an empty block differs from none, that a lender of rate 0
// differs from an empty block, that a lender differs from a borrower, that every
// field of the module and of the blocks is hashed, and that the order the roles are
// written in changes nothing.
//
// Requirement: FR-CRD-012, FR-CFG-088
func TestHashConfig_Credit(t *testing.T) {
	t.Parallel()
	if HashConfig(Config{}) == HashConfig(Config{Modules: Modules{Credit: &Credit{}}}) {
		t.Error("HashConfig of the module on and empty = that of the module off, want different")
	}
	base := creditHashConfig()
	hash := HashConfig(base)
	changes := map[string]func(c *Config){
		"MinPrincipal":  func(c *Config) { c.Modules.Credit.MinPrincipal++ },
		"ShieldTicks":   func(c *Config) { c.Modules.Credit.ShieldTicks++ },
		"WriteOffTicks": func(c *Config) { c.Modules.Credit.WriteOffTicks++ },
		"MaxClaims":     func(c *Config) { c.Modules.Credit.MaxClaims++ },
		"RateBP":        func(c *Config) { c.Roles[1].Credit.Lend.RateBP++ },
		"MaxTermTicks":  func(c *Config) { c.Roles[1].Credit.Lend.MaxTermTicks++ },
		"KeepTicks":     func(c *Config) { c.Roles[1].Credit.Lend.KeepTicks++ },
		"ShareBP":       func(c *Config) { c.Roles[1].Credit.Lend.ShareBP++ },
		"MaxRateBP":     func(c *Config) { c.Roles[0].Credit.Borrow.MaxRateBP++ },
		"MaxDebtTicks":  func(c *Config) { c.Roles[0].Credit.Borrow.MaxDebtTicks++ },
		"SlackTicks":    func(c *Config) { c.Roles[0].Credit.Borrow.SlackTicks++ },
		"block removed": func(c *Config) { c.Roles[0].Credit = nil },
		"empty block":   func(c *Config) { c.Roles[2].Credit = &CreditTerms{} },
		"lender to borrower": func(c *Config) {
			c.Roles[1].Credit = &CreditTerms{Borrow: &BorrowTerms{}}
		},
		"zero lender": func(c *Config) { c.Roles[2].Credit = &CreditTerms{Lend: &LendTerms{}} },
	}
	seen := map[[32]byte]string{hash: "base"}
	for name, change := range changes {
		c := creditHashConfig()
		change(&c)
		h := HashConfig(c)
		if prior, dup := seen[h]; dup {
			t.Errorf("HashConfig after %q = that after %q, want every change hashed and each different", name, prior)
		}
		seen[h] = name
	}
	zeroLender, emptyBlock := creditHashConfig(), creditHashConfig()
	zeroLender.Roles[2].Credit = &CreditTerms{Lend: &LendTerms{}}
	emptyBlock.Roles[2].Credit = &CreditTerms{}
	if HashConfig(zeroLender) == HashConfig(emptyBlock) {
		t.Error("HashConfig of a lender of rate 0 = that of an empty block, want different")
	}
	swapped := creditHashConfig()
	swapped.Roles[0], swapped.Roles[1] = swapped.Roles[1], swapped.Roles[0]
	if HashConfig(swapped) != hash {
		t.Error("HashConfig changed with the order of the roles, want canonical order")
	}
}

// TestCanonical_CreditCopiesBlocks verifies that the canonical copy holds blocks of
// its own: changing the consumer's Config after canonical moves nothing in the copy.
//
// Requirement: FR-CRD-012
func TestCanonical_CreditCopiesBlocks(t *testing.T) {
	t.Parallel()
	c := creditHashConfig()
	cp := c.canonical()
	c.Modules.Credit.MaxClaims = 1
	c.Roles[1].Credit.Lend.RateBP = 1
	c.Roles[0].Credit.Borrow.MaxRateBP = 1
	if cp.Modules.Credit.MaxClaims != 1_000 || cp.Roles[1].Credit.Lend.RateBP != 600 || cp.Roles[0].Credit.Borrow.MaxRateBP != 1_500 {
		t.Errorf("canonical copy changed with the original: %+v %+v %+v", *cp.Modules.Credit, *cp.Roles[1].Credit.Lend, *cp.Roles[0].Credit.Borrow)
	}
	var none *Credit
	if none.canonical() != nil || (*CreditTerms)(nil).canonical() != nil {
		t.Error("canonical of a nil module or block is not nil")
	}
}

// ledgerWithCredit builds the ledger of a minimal configuration, the module on or
// off, and two agent accounts, the first holding 1 000 of money.
func ledgerWithCredit(t *testing.T, on bool, maxClaims int64) (l *ledger.Ledger, a, b int) {
	t.Helper()
	cfg := Config{Regions: []Region{{ID: "north"}}}
	if on {
		cfg.Modules.Credit = &Credit{MinPrincipal: 1, MaxClaims: maxClaims}
	}
	l = mustLedger(t, cfg)
	var err error
	if a, err = l.Open(ledger.KindAgent, 0, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if b, err = l.Open(ledger.KindAgent, 1, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err = l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	from, ok := l.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("no Initial boundary account for money")
	}
	if err = l.Transfer(from, a, 1_000, ledger.Initial); err != nil {
		t.Fatalf("Move: %v", err)
	}
	return l, a, b
}

// TestCredit_LedgerReasonsDeclaredWhenOn verifies that with the module on the ledger
// declares the three loan reasons and takes MaxClaims claims, and with it off a
// claim is refused with ErrReasonMismatch (the reason is not declared) and the
// reasons are refused too.
//
// Requirement: FR-CRD-002
func TestCredit_LedgerReasonsDeclaredWhenOn(t *testing.T) {
	t.Parallel()
	l, a, b := ledgerWithCredit(t, true, 1)
	for _, r := range []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest} {
		if err := l.Transfer(a, b, 1, r); err != nil {
			t.Errorf("Transfer with reason %v and the module on = %v, want nil", r, err)
		}
	}
	id, err := l.Lend(a, b, 100, 10, 600, 5)
	if err != nil {
		t.Fatalf("Lend with the module on = %v, want nil", err)
	}
	if _, err := l.Lend(a, b, 100, 10, 600, 5); !errors.Is(err, ledger.ErrTooManyClaims) {
		t.Errorf("a second Lend with MaxClaims 1 (claim %d open) = %v, want ErrTooManyClaims", id, err)
	}

	off, a, b := ledgerWithCredit(t, false, 0)
	for _, r := range []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest} {
		if err := off.Transfer(a, b, 1, r); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Transfer with reason %v and the module off = %v, want ErrReasonMismatch", r, err)
		}
	}
	if _, err := off.Lend(a, b, 100, 10, 600, 5); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("Lend with the module off = %v, want ErrReasonMismatch", err)
	}
}

// TestClaimsLimit verifies the limit the module hands the ledger: MaxClaims when on,
// 0 when off.
//
// Requirement: FR-CRD-002
func TestClaimsLimit(t *testing.T) {
	t.Parallel()
	var off *Credit
	if got := off.claimsLimit(); got != 0 {
		t.Errorf("claimsLimit of the module off = %d, want 0", got)
	}
	if got := (&Credit{MaxClaims: 42}).claimsLimit(); got != 42 {
		t.Errorf("claimsLimit = %d, want 42", got)
	}
}

// TestClaimsLimit_Saturates verifies that a MaxClaims above MaxOpenClaims, which
// Validate refuses, is cut to the bound rather than narrowed to an int that could
// wrap on a 32-bit platform, and that a negative one reaches the ledger as it is, to
// be refused there.
//
// Requirement: FR-CRD-002
//
// Fix: dp3-crd-1. The conversion int(MaxClaims) was unbounded: on a 32-bit int,
// 1<<32 + 5 became 5, a limit that no configuration had asked for.
func TestClaimsLimit_Saturates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		max  int64
		want int
	}{
		{"at the bound", MaxOpenClaims, MaxOpenClaims},
		{"above the bound", MaxOpenClaims + 1, MaxOpenClaims},
		{"far above the bound", 1<<32 + 5, MaxOpenClaims},
		{"largest int64", 1<<63 - 1, MaxOpenClaims},
		{"negative", -1, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (&Credit{MaxClaims: tt.max}).claimsLimit(); got != tt.want {
				t.Errorf("claimsLimit(MaxClaims %d) = %d, want %d", tt.max, got, tt.want)
			}
		})
	}
}
