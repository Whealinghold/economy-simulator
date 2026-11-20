package economysim_test

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the saving module's configuration surface (spec-saving-and-investment.md
// 10, 11, FR-SAV-034, 035, FR-CFG-096): the failing case of each rule of 11.1, their
// order, the warnings of 11.2 and the hash. The bytes of the hash are in
// saving_internal_test.go and the loader in config/saving_test.go.

// savingConfig returns infrastructureConfig with the saving module on and the growers
// as savers. The growers' list is a seed gatherer, a grinder that needs a mill, and the
// build and repair of a mill, so the one project, a mill, gains over the gatherer
// about eleven times its cost: it passes the hurdle of 15 000 and the spread of
// 100 000 leaves no agent above the gain. The configuration raises none of the module's
// warnings. Every number is a placeholder (N-015).
func savingConfig() economysim.Config {
	c := infrastructureConfig()
	grind := recipeOf(&c, "grind")
	grind.Outputs = []economysim.Output{{Commodity: "staple", Quantity: 8, CostShare: 10_000}}
	m := roleOf(&c, "grower")
	m.Recipes = []string{"gather_seed", "grind", "build_mill", "repair_mill"}
	s := economysim.DefaultSaving()
	s.HurdleSpreadBP = 100_000
	c.Modules.Saving = &s
	b := economysim.DefaultSaver()
	m.Saver = &b
	return c
}

// savingWarningsOf returns the rule@path strings of the warnings about the module or a
// Saver block.
func savingWarningsOf(c economysim.Config) []string {
	var got []string
	for _, w := range c.Warnings() {
		p := strings.Join(w.Path, ".")
		if strings.HasPrefix(p, "modules.saving") || strings.Contains(p, ".saver") {
			got = append(got, w.Rule+"@"+p)
		}
	}
	return got
}

// TestValidateSaving_ConfigIsValid verifies that the configuration the saving tests
// start from validates and raises none of the module's warnings, that the module with
// no block validates (it is on and inert), and that a block with every field zero is
// out of bounds only at the fields whose bound is above 0.
//
// Requirement: FR-SAV-001, FR-SAV-034
//
// Guard: the field cases start from savingConfig; if it did not validate, each case
// would pass for the wrong reason.
func TestValidateSaving_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := savingConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("savingConfig().Validate() = %v, want nil", err)
	}
	if got := savingWarningsOf(c); len(got) != 0 {
		t.Errorf("savingConfig().Warnings() has %v, want none of the module's", got)
	}
	bare := referenceConfig()
	bare.Modules.Saving = &economysim.Saving{}
	if err := bare.Validate(); err != nil {
		t.Errorf("the module with no block .Validate() = %v, want nil", err)
	}
}

// savingField builds a failing case from the valid configuration.
type savingField func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase

// savingFieldCases are the failing cases of the rows of the saving module in the
// parameter-owner table (spec-saving-and-investment.md 10.2, 11.1, FR-CFG-096): each
// breaks one field of savingConfig. buildFieldCases appends them to the table that the
// registry test reads.
func savingFieldCases() []fieldCase {
	field := savingField(func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = savingConfig()
				set(c)
			}}
	})
	cs := savingModuleCases(field)
	cs = append(cs, savingBlockCases(field)...)
	return append(cs, savingWorthCases(field)...)
}

// savingModuleCases are the cases of the three fields of the module.
func savingModuleCases(field savingField) []fieldCase {
	at := func(key string) []string { return []string{"modules", "saving", key} }
	str := strconv.Itoa
	return []fieldCase{
		field("Saving.SafetyTicks", "above MaxSafetyTicks", economysim.ErrOutOfBounds, at("safety_ticks"), str(economysim.MaxSafetyTicks+1), str(economysim.MaxSafetyTicks),
			func(c *economysim.Config) { c.Modules.Saving.SafetyTicks = economysim.MaxSafetyTicks + 1 }),
		field("Saving.SafetyTicks", "below 0", economysim.ErrOutOfBounds, at("safety_ticks"), "-1", "0",
			func(c *economysim.Config) { c.Modules.Saving.SafetyTicks = -1 }),
		field("Saving.HurdleSpreadBP", "above MaxHurdleSpreadBP", economysim.ErrOutOfBounds, at("hurdle_spread_bp"), str(economysim.MaxHurdleSpreadBP+1), str(economysim.MaxHurdleSpreadBP),
			func(c *economysim.Config) { c.Modules.Saving.HurdleSpreadBP = economysim.MaxHurdleSpreadBP + 1 }),
		field("Saving.CommitBandBP", "above MaxCommitBandBP", economysim.ErrOutOfBounds, at("commit_band_bp"), str(economysim.MaxCommitBandBP+1), str(economysim.MaxCommitBandBP),
			func(c *economysim.Config) { c.Modules.Saving.CommitBandBP = economysim.MaxCommitBandBP + 1 }),
	}
}

// savingBlockCases are the cases of the block of a role: the module off and the six
// fields.
func savingBlockCases(field savingField) []fieldCase {
	block := []string{"roles", "grower", "saver"}
	at := func(key string) []string { return append(slices.Clone(block), key) }
	str := strconv.Itoa
	set := func(f func(b *economysim.Saver)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(roleOf(c, "grower").Saver) }
	}
	return []fieldCase{
		field("Role.Saver", "a block with the module off", economysim.ErrModuleDependency, block, "saver block", "modules.saving",
			func(c *economysim.Config) { c.Modules.Saving = nil }),
		field("Saver.PrecautionShareBP", "above MaxPrecautionShareBP", economysim.ErrOutOfBounds, at("precaution_share_bp"), str(economysim.MaxPrecautionShareBP+1), str(economysim.MaxPrecautionShareBP),
			set(func(b *economysim.Saver) { b.PrecautionShareBP = economysim.MaxPrecautionShareBP + 1 })),
		field("Saver.PrecautionCapTicks", "above MaxPrecautionCapTicks", economysim.ErrOutOfBounds, at("precaution_cap_ticks"), str(economysim.MaxPrecautionCapTicks+1), str(economysim.MaxPrecautionCapTicks),
			set(func(b *economysim.Saver) { b.PrecautionCapTicks = economysim.MaxPrecautionCapTicks + 1 })),
		field("Saver.HurdleBP", "below break-even", economysim.ErrOutOfBounds, at("hurdle_bp"), "9999", "10000",
			set(func(b *economysim.Saver) { b.HurdleBP = 9_999 })),
		field("Saver.HurdleBP", "above MaxHurdleBP", economysim.ErrOutOfBounds, at("hurdle_bp"), str(economysim.MaxHurdleBP+1), str(economysim.MaxHurdleBP),
			set(func(b *economysim.Saver) { b.HurdleBP = economysim.MaxHurdleBP + 1 })),
		field("Saver.PaybackTicks", "below 1", economysim.ErrOutOfBounds, at("payback_ticks"), "0", "1",
			set(func(b *economysim.Saver) { b.PaybackTicks = 0 })),
		field("Saver.PaybackTicks", "above MaxPaybackTicks", economysim.ErrOutOfBounds, at("payback_ticks"), str(economysim.MaxPaybackTicks+1), str(economysim.MaxPaybackTicks),
			set(func(b *economysim.Saver) { b.PaybackTicks = economysim.MaxPaybackTicks + 1 })),
		field("Saver.RepairBelowBP", "above 10 000", economysim.ErrOutOfBounds, at("repair_below_bp"), "10001", "10000",
			set(func(b *economysim.Saver) { b.RepairBelowBP = 10_001 })),
	}
}

// savingWorthCases are the cases of the Worth rows: the count, the numbers of a row, the
// module it needs, an unknown class and a class twice.
func savingWorthCases(field savingField) []fieldCase {
	block := []string{"roles", "grower", "saver"}
	row := func(i int, key string) []string {
		return append(slices.Clone(block), "worth["+strconv.Itoa(i)+"]", key)
	}
	str := strconv.Itoa
	rows := func(w ...economysim.ProjectWorth) func(c *economysim.Config) {
		return func(c *economysim.Config) { roleOf(c, "grower").Saver.Worth = w }
	}
	good := economysim.ProjectWorth{Class: "road", PerTick: 10, Count: 1}
	return []fieldCase{
		field("Saver.Worth", "more than MaxWorthRows", economysim.ErrOutOfBounds, append(slices.Clone(block), "worth"), str(economysim.MaxWorthRows+1), str(economysim.MaxWorthRows),
			rows(slices.Repeat([]economysim.ProjectWorth{good}, economysim.MaxWorthRows+1)...)),
		field("Saver.Worth", "a row with the infrastructure module off", economysim.ErrModuleDependency, append(slices.Clone(block), "worth"), "worth row", "modules.infrastructure",
			func(c *economysim.Config) {
				roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{good}
				c.Modules.Infrastructure = nil
				for _, r := range []string{"build_mill", "repair_mill"} {
					recipeOf(c, r).Builds, recipeOf(c, r).Repairs = nil, nil
				}
				recipeOf(c, "grind").Needs = nil
			}),
		field("Saver.Worth", "a class not defined", economysim.ErrUnknownFacilityClass, []string{"roles", "grower", "saver", "worth", "bridge", "class"}, "bridge", "",
			rows(economysim.ProjectWorth{Class: "bridge", PerTick: 10, Count: 1})),
		field("Saver.Worth", "a class twice", economysim.ErrDuplicateID, []string{"roles", "grower", "saver", "worth", "road", "class"}, "road", "",
			rows(good, good)),
		field("ProjectWorth.Class", "a class not defined", economysim.ErrUnknownFacilityClass, []string{"roles", "grower", "saver", "worth", "bridge", "class"}, "bridge", "",
			rows(economysim.ProjectWorth{Class: "bridge", PerTick: 10, Count: 1})),
		field("ProjectWorth.PerTick", "below 1", economysim.ErrOutOfBounds, row(0, "per_tick"), "0", "1",
			rows(economysim.ProjectWorth{Class: "road", PerTick: 0, Count: 1})),
		field("ProjectWorth.PerTick", "above MaxWorthPerTick", economysim.ErrOutOfBounds, row(0, "per_tick"), str(economysim.MaxWorthPerTick+1), str(economysim.MaxWorthPerTick),
			rows(economysim.ProjectWorth{Class: "road", PerTick: economysim.MaxWorthPerTick + 1, Count: 1})),
		field("ProjectWorth.Count", "below 1", economysim.ErrOutOfBounds, row(0, "count"), "0", "1",
			rows(economysim.ProjectWorth{Class: "road", PerTick: 1, Count: 0})),
		field("ProjectWorth.Count", "above MaxWorthCount", economysim.ErrOutOfBounds, row(0, "count"), str(economysim.MaxWorthCount+1), str(economysim.MaxWorthCount),
			rows(economysim.ProjectWorth{Class: "road", PerTick: 1, Count: economysim.MaxWorthCount + 1})),
	}
}

// TestValidateSaving_Errors verifies each row of 11.1 that Validate checks: every case of
// savingFieldCases, which breaks only its own field of a valid configuration, is refused
// with its rule (errors.Is) at its path with its value and bound.
//
// Requirement: FR-SAV-034
func TestValidateSaving_Errors(t *testing.T) {
	t.Parallel()
	for _, tc := range savingFieldCases() {
		t.Run(tc.field+" "+tc.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tc.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tc.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tc.rules[0])
			}
			var found bool
			for _, e := range unwrapAll(err) {
				var ve *economysim.ValidationError
				if errors.As(e, &ve) && errors.Is(e, tc.rules[0]) && strings.Join(ve.Path, ".") == strings.Join(tc.paths[0], ".") {
					found = found || (ve.Value == tc.value && ve.Bound == tc.bound)
				}
			}
			if !found {
				t.Errorf("Validate() = %v, want %v at %v with value %q and bound %q", err, tc.rules[0], tc.paths[0], tc.value, tc.bound)
			}
		})
	}
}

// TestValidateSaving_Order verifies the order of 11.1: two mistakes at once report the
// earlier rule only, the bounds before the dependencies and the dependencies before the
// class names, and no module error is raised for a Worth row's class when the
// infrastructure module is off.
//
// Requirement: FR-SAV-034
func TestValidateSaving_Order(t *testing.T) {
	t.Parallel()
	worth := []economysim.ProjectWorth{{Class: "bridge", PerTick: 1, Count: 1}}
	tests := []struct {
		name   string
		break_ func(c *economysim.Config)
		want   error
		not    error
	}{
		{"bounds before dependency", func(c *economysim.Config) {
			c.Modules.Saving.SafetyTicks = -1
			c.Modules.Saving = nil
			roleOf(c, "grower").Saver.PaybackTicks = 0
		}, economysim.ErrOutOfBounds, economysim.ErrModuleDependency},
		{"dependency before class", func(c *economysim.Config) {
			c.Modules.Saving = nil
			roleOf(c, "grower").Saver.Worth = worth
		}, economysim.ErrModuleDependency, economysim.ErrUnknownFacilityClass},
		{"module bound before block bound is one phase", func(c *economysim.Config) {
			c.Modules.Saving.CommitBandBP = economysim.MaxCommitBandBP + 1
			roleOf(c, "grower").Saver.Worth = worth
		}, economysim.ErrOutOfBounds, economysim.ErrUnknownFacilityClass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := savingConfig()
			tt.break_(&c)
			err := c.Validate()
			if !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
			if errors.Is(err, tt.not) {
				t.Errorf("Validate() = %v, want %v not reported while an earlier rule fails", err, tt.not)
			}
		})
	}
}

// TestSentinels_Saving verifies that ErrNotSaver is its own sentinel, matched with
// errors.Is through a wrap and by no other sentinel of the module.
//
// Requirement: FR-SAV-037
func TestSentinels_Saving(t *testing.T) {
	t.Parallel()
	if economysim.ErrNotSaver.Error() == "" {
		t.Fatal("ErrNotSaver has no message")
	}
	wrapped := errors.Join(errors.New("Saving"), economysim.ErrNotSaver)
	if !errors.Is(wrapped, economysim.ErrNotSaver) {
		t.Errorf("errors.Is(wrapped, ErrNotSaver) = false, want true")
	}
	for _, other := range []error{economysim.ErrOutOfBounds, economysim.ErrModuleDependency, economysim.ErrUnknownFacilityClass, economysim.ErrDuplicateID} {
		if errors.Is(other, economysim.ErrNotSaver) {
			t.Errorf("ErrNotSaver matches %v, want a sentinel of its own", other)
		}
	}
}

// TestHash_SavingOnOff verifies that the module off leaves the pinned hash of the
// reference configuration, that the module on and empty, and an empty block, differ
// from none, that every field of the module and the block and every row is hashed and
// each change moves the hash differently, and that a zero field spelled out hashes as
// the field left out.
//
// Requirement: FR-CFG-096, FR-SAV-001
func TestHash_SavingOnOff(t *testing.T) {
	t.Parallel()
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Fatalf("hash of the reference configuration = %s, want the pinned %s", got, pinnedReferenceHash)
	}
	on := referenceConfig()
	on.Modules.Saving = &economysim.Saving{}
	if hexHash(on) == pinnedReferenceHash {
		t.Errorf("an empty module that is on hashes as the module off")
	}
	base := hexHash(savingConfig())
	changes := map[string]func(c *economysim.Config){
		"SafetyTicks":        func(c *economysim.Config) { c.Modules.Saving.SafetyTicks++ },
		"HurdleSpreadBP":     func(c *economysim.Config) { c.Modules.Saving.HurdleSpreadBP-- },
		"CommitBandBP":       func(c *economysim.Config) { c.Modules.Saving.CommitBandBP++ },
		"PrecautionShareBP":  func(c *economysim.Config) { roleOf(c, "grower").Saver.PrecautionShareBP++ },
		"PrecautionCapTicks": func(c *economysim.Config) { roleOf(c, "grower").Saver.PrecautionCapTicks++ },
		"HurdleBP":           func(c *economysim.Config) { roleOf(c, "grower").Saver.HurdleBP++ },
		"PaybackTicks":       func(c *economysim.Config) { roleOf(c, "grower").Saver.PaybackTicks++ },
		"RepairBelowBP":      func(c *economysim.Config) { roleOf(c, "grower").Saver.RepairBelowBP++ },
		"a Worth row": func(c *economysim.Config) {
			roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "road", PerTick: 1, Count: 1}}
		},
		"the block removed": func(c *economysim.Config) { roleOf(c, "grower").Saver = nil },
		"an empty block on another role": func(c *economysim.Config) {
			roleOf(c, "cook").Saver = &economysim.Saver{}
		},
		"the module removed": func(c *economysim.Config) { c.Modules.Saving = nil },
	}
	seen := map[string]string{base: "base"}
	for name, change := range changes {
		c := savingConfig()
		change(&c)
		h := hexHash(c)
		if prior, dup := seen[h]; dup {
			t.Errorf("hash after changing %s = the hash after %s, want every change hashed and each different", name, prior)
		}
		seen[h] = name
	}
	zero, left := savingConfig(), savingConfig()
	zero.Modules.Saving.SafetyTicks, left.Modules.Saving.SafetyTicks = 0, 0
	roleOf(&zero, "grower").Saver.Worth = []economysim.ProjectWorth{}
	if hexHash(zero) != hexHash(left) {
		t.Errorf("hash with an empty Worth slice differs from a nil one")
	}
}

// TestSaving_ModuleOnBuildsAWorldThatIgnoresIt verifies that a configuration with the
// module on and a block on a role builds a world and runs a tick: this change adds no
// behaviour (FR-SAV-001), the module's data is accepted and nothing reads it yet.
//
// Requirement: FR-SAV-001
func TestSaving_ModuleOnBuildsAWorldThatIgnoresIt(t *testing.T) {
	t.Parallel()
	on, off := savingConfig(), savingConfig()
	off.Modules.Saving = nil
	roleOf(&off, "grower").Saver = nil
	a, err := economysim.NewWorld(on, economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld(saving on) = %v", err)
	}
	b, err := economysim.NewWorld(off, economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld(saving off) = %v", err)
	}
	for range 5 {
		if err := a.Step(); err != nil {
			t.Fatalf("Step with the module on = %v", err)
		}
		if err := b.Step(); err != nil {
			t.Fatalf("Step with the module off = %v", err)
		}
	}
	sa, errA := a.Snapshot()
	sb, errB := b.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot = %v, %v", errA, errB)
	}
	sa.ConfigHash, sb.ConfigHash = [32]byte{}, [32]byte{} // the hash is the one thing the configuration changes
	if !reflect.DeepEqual(sa, sb) {
		t.Error("the module on changed the snapshot, want it to be ignored")
	}
}

// TestSavingWarnings verifies the eight warnings of spec-saving-and-investment.md 11.2:
// each case changes the valid configuration so that exactly its warning is raised, once,
// with the rule and the path, and the configuration still validates where the change is
// not an error (a warning never fails validation, FR-SAV-035). The quiet case is the
// configuration itself, which TestValidateSaving_ConfigIsValid pins as raising none.
//
// Requirement: FR-SAV-035
func TestSavingWarnings(t *testing.T) {
	t.Parallel()
	block := "roles.grower.saver"
	tests := []struct {
		name   string
		change func(c *economysim.Config)
		want   []string
	}{
		{"a block and no recipe that builds or repairs", func(c *economysim.Config) {
			roleOf(c, "grower").Recipes = []string{"gather_seed", "grind"}
		}, []string{"saver_no_project@" + block}},
		{"a Worth row for a class no recipe builds or repairs", func(c *economysim.Config) {
			roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "road", PerTick: 10, Count: 1}}
		}, []string{"worth_unused@" + block + ".worth.road"}},
		{"a Worth row for a site class", func(c *economysim.Config) {
			roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "mill", PerTick: 100, Count: 1}}
		}, []string{"worth_on_site@" + block + ".worth.mill"}},
		{"no project that passes at the reference prices", func(c *economysim.Config) {
			recipeOf(c, "grind").Outputs[0].Quantity = 4
		}, []string{"project_below_hurdle@" + block}},
		{"a safety below the slack of a borrower", func(c *economysim.Config) {
			c.Modules.Credit = &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, MaxClaims: 1_000}
			roleOf(c, "grower").Credit = &economysim.CreditTerms{Borrow: &economysim.BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
		}, []string{"safety_below_slack@" + block}},
		{"a repair recipe and a gate of 0", func(c *economysim.Config) {
			roleOf(c, "grower").Saver.RepairBelowBP = 0
		}, []string{"repair_never@" + block}},
		{"a spread below the margin of the best project", func(c *economysim.Config) {
			c.Modules.Saving.HurdleSpreadBP = 1_000
		}, []string{"spread_below_margin@" + block}},
		{"a target above the money of the start", func(c *economysim.Config) {
			for i := range c.Roles {
				c.Roles[i].StartMoney = 1
			}
		}, []string{"saving_target_exceeds_money@modules.saving"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := savingConfig()
			tt.change(&c)
			got := savingWarningsOf(c)
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() about the module = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSavingWarnings_OffIsSilent verifies that the module off raises none of the
// warnings, even for a block (the block is the error ErrModuleDependency).
//
// Requirement: FR-SAV-035
func TestSavingWarnings_OffIsSilent(t *testing.T) {
	t.Parallel()
	c := savingConfig()
	c.Modules.Saving = nil
	if got := savingWarningsOf(c); len(got) != 0 {
		t.Errorf("Warnings() with the module off = %v, want none", got)
	}
}

// TestSavingWarnings_NeverFail verifies that a configuration that raises a warning of
// the module validates: a warning never fails validation (FR-SAV-035).
//
// Requirement: FR-SAV-035
func TestSavingWarnings_NeverFail(t *testing.T) {
	t.Parallel()
	c := savingConfig()
	roleOf(&c, "grower").Recipes = []string{"gather_seed", "grind"}
	roleOf(&c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "mill", PerTick: 10, Count: 1}}
	if len(savingWarningsOf(c)) == 0 {
		t.Fatal("the configuration raises no warning, want at least one for the test to mean something")
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil: a warning does not fail", err)
	}
}
