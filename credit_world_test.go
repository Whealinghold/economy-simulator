package economysim_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditPackDoc is the credit tables of spec-credit.md 4.1 and 4.2 as a pack layer:
// the module and a Borrow and a Lend block on two roles of the pack one.
const creditPackDoc = `
[modules.credit]
min_principal = 5000
shield_ticks = 2
write_off_ticks = 7
max_claims = 1000

[role.grower.credit.borrow]
max_rate_bp = 1500
max_debt_ticks = 90
slack_ticks = 4

[role.cook.credit.lend]
rate_bp = 600
max_term_ticks = 90
keep_ticks = 3
share_bp = 5000
`

// TestCredit_PackFileToWorld verifies the path from a pack file to a world: the
// credit tables load on top of the pack one, validate, build a world, and the world
// reports the module, the two blocks and the configuration hash of the loaded value,
// which differs from the pack with the module off.
//
// Requirement: FR-CRD-001, FR-CRD-011, FR-CRD-012
func TestCredit_PackFileToWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditPackDoc), packOne...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(pack one + credit tables) error = %v, want nil", err)
	}
	got := w.Config()
	wantModule := &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, WriteOffTicks: 7, MaxClaims: 1_000}
	if !reflect.DeepEqual(got.Modules.Credit, wantModule) {
		t.Errorf("World.Config().Modules.Credit = %+v, want %+v", got.Modules.Credit, wantModule)
	}
	wantBlocks := map[string]economysim.CreditTerms{
		"grower": {Borrow: &economysim.BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}},
		"cook":   {Lend: &economysim.LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}},
	}
	for _, r := range got.Roles {
		want, has := wantBlocks[r.ID]
		if has != (r.Credit != nil) || (has && !reflect.DeepEqual(*r.Credit, want)) {
			t.Errorf("World.Config() role %q Credit = %+v, want %+v (present: %v)", r.ID, r.Credit, want, has)
		}
	}
	if w.ConfigHash() != economysim.HashConfig(cfg) {
		t.Errorf("World.ConfigHash() = %x, want HashConfig of the loaded configuration %x", w.ConfigHash(), economysim.HashConfig(cfg))
	}
	if off := loadValidPack(t, packOne...); economysim.HashConfig(off) == w.ConfigHash() {
		t.Error("hash of the pack with the module = that of the pack without, want different")
	}
}

// TestCredit_NewWorldRejectsABadBlock verifies that construction refuses a pack
// whose credit tables break a rule of 15.1, with the sentinel of the rule and no
// world, so a file that breaks the module never reaches a run.
//
// Requirement: FR-CRD-010, FR-CRD-011
func TestCredit_NewWorldRejectsABadBlock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		name string
		set  func(c *economysim.Config)
		want error
	}{
		{"a rate above the bound", func(c *economysim.Config) {
			creditRole(c, "cook").Credit.Lend.RateBP = economysim.MaxCreditRateBP + 1
		}, economysim.ErrOutOfBounds},
		{"a block with both sides", func(c *economysim.Config) {
			creditRole(c, "grower").Credit.Lend = &economysim.LendTerms{MaxTermTicks: 1, ShareBP: 1}
		}, economysim.ErrCreditShape},
		{"a block with the module off", func(c *economysim.Config) { c.Modules.Credit = nil }, economysim.ErrModuleDependency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPackBytes(t, []byte(creditPackDoc), packOne...)
			tt.set(&cfg)
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
			if !errors.Is(err, tt.want) || w != nil {
				t.Errorf("NewWorld(%s) = %v, %v, want a nil world and %v", tt.name, w, err, tt.want)
			}
		})
	}
}

// TestCredit_InertOverARun verifies, over a run of the pack one with the audit on
// and a recording observer, that a world built with the credit module on behaves
// exactly as one built with it off: the same events in the same order (but for the
// configuration hash of WorldCreated, which the module moves) and the same final
// snapshot, for two seeds. The module has no state, no position and no event yet,
// so its presence, with or without a lender and a borrower, moves nothing.
//
// Requirement: FR-CRD-001
//
// Guard: the module is inert until its positions are built (plan 3-crd Phases 4 to
// 7). The empty module and the module with blocks must stay identical to off until
// then; the PR that builds P1.7a and P3.3b replaces the block case with the
// invariants of spec-credit.md 12.
func TestCredit_InertOverARun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	variants := []struct {
		name string
		cfg  economysim.Config
	}{
		{"the empty module", func() economysim.Config {
			c := off
			c.Modules.Credit = &economysim.Credit{MinPrincipal: 1, MaxClaims: 1}
			return c
		}()},
		{"a lender and a borrower", loadPackBytes(t, []byte(creditPackDoc), packOne...)},
	}
	for seed := uint64(1); seed <= uint64(seedsFor(raceEnabled, 2)); seed++ {
		base := runPack(t, off, runSpec{Seed: seed, Ticks: 30, Observe: true, Audit: true})
		for _, v := range variants {
			t.Run(v.name, func(t *testing.T) {
				raw := runPack(t, v.cfg, runSpec{Seed: seed, Ticks: 30, Observe: true, Audit: true})
				if raw.World.ConfigHash() == base.World.ConfigHash() {
					t.Errorf("seed %d: the configuration hash with %s = that of the module off, want different", seed, v.name)
				}
				res := sameButHash(raw, base.World.ConfigHash())
				// The module names itself among the modules that are on (spec-credit.md
				// 16.2), the one thing besides the hash that it may change while it has no
				// claim; the lists are compared without it.
				res.Final.Modules = slices.DeleteFunc(slices.Clone(res.Final.Modules), func(m string) bool { return m == "credit" })
				if !bytes.Equal(renderEvents(res), renderEvents(base)) {
					t.Errorf("seed %d: the run with %s renders differently from the run with the module off, want identical", seed, v.name)
				}
				if !reflect.DeepEqual(res.Events, base.Events) {
					t.Errorf("seed %d: events with %s differ from the module off (%d and %d events), want identical", seed, v.name, len(res.Events), len(base.Events))
				}
				if !reflect.DeepEqual(res.Final, base.Final) {
					t.Errorf("seed %d: the final snapshot with %s differs from the module off, want identical", seed, v.name)
				}
			})
		}
	}
}
