package economysim_test

import (
	"errors"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the migration module's configuration surface: its nine fields, the
// region rows, the flag of a role, the bounds, the derived checks, the defaults
// and the seven warnings (spec-migration.md 11, 12, FR-MIG-001 to 004, 039, 040,
// 043, 044, FR-CFG-078, 080). Its effect on a world is tested in
// migration_internal_test.go.

// migrationConfig returns informationConfig with the migration module on at a
// speed of 20: the edges of 55 and 40 give journeys of ceil(55 / 20) = 3 ticks
// from north to south and ceil(40 / 20) = 2 from south to north.
func migrationConfig() economysim.Config {
	c := informationConfig()
	m := economysim.DefaultMigration()
	m.MoveSpeed = 20
	c.Modules.Migration = &m
	return c
}

// migrationFieldCases are the failing cases of the rows of the migration module
// in the parameter-owner table (spec-migration.md 11, 12.1, FR-CFG-078): each
// breaks one field of a valid configuration. buildFieldCases appends them to the
// table that the registry test reads.
func migrationFieldCases() []fieldCase {
	module := func(field, name, key, value, bound string, set func(m *economysim.Migration)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{economysim.ErrOutOfBounds},
			paths: [][]string{{"modules", "migration", key}}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = migrationConfig()
				set(c.Modules.Migration)
			}}
	}
	row := func(field, name string, rule error, path []string, set func(m *economysim.Migration)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path},
			mutate: func(c *economysim.Config) {
				*c = migrationConfig()
				set(c.Modules.Migration)
			}}
	}
	return []fieldCase{
		module("Migration.MoveSpeed", "0, the missing key", "move_speed", "0", "1", func(m *economysim.Migration) { m.MoveSpeed = 0 }),
		module("Migration.MoveSpeed", "above MaxSpeed", "move_speed", strconv.Itoa(economysim.MaxSpeed+1), strconv.Itoa(economysim.MaxSpeed),
			func(m *economysim.Migration) { m.MoveSpeed = economysim.MaxSpeed + 1 }),
		module("Migration.MoveReviewTicks", "0", "move_review_ticks", "0", "1", func(m *economysim.Migration) { m.MoveReviewTicks = 0 }),
		module("Migration.MoveReviewTicks", "above MaxMoveReviewTicks", "move_review_ticks", "1001", "1000",
			func(m *economysim.Migration) { m.MoveReviewTicks = economysim.MaxMoveReviewTicks + 1 }),
		module("Migration.MoveBelow", "below 0", "move_below", "-1", "0", func(m *economysim.Migration) { m.MoveBelow = -1 }),
		module("Migration.MoveBelow", "above MaxBasisPoints", "move_below", strconv.Itoa(economysim.MaxBasisPoints+1), strconv.Itoa(economysim.MaxBasisPoints),
			func(m *economysim.Migration) { m.MoveBelow = economysim.MaxBasisPoints + 1 }),
		module("Migration.MoveGain", "below 0", "move_gain", "-1", "0", func(m *economysim.Migration) { m.MoveGain = -1 }),
		module("Migration.MoveWidth", "above MaxBasisPoints", "move_width", strconv.Itoa(economysim.MaxBasisPoints+1), strconv.Itoa(economysim.MaxBasisPoints),
			func(m *economysim.Migration) { m.MoveWidth = economysim.MaxBasisPoints + 1 }),
		module("Migration.MoveCooldown", "above MaxWelfareWindow", "move_cooldown", "3651", "3650",
			func(m *economysim.Migration) { m.MoveCooldown = economysim.MaxWelfareWindow + 1 }),
		module("Migration.MoveHorizon", "below 2", "move_horizon", "1", "2", func(m *economysim.Migration) { m.MoveHorizon = 1 }),
		module("Migration.MoveHorizon", "above MaxTravelTicks", "move_horizon", strconv.Itoa(economysim.MaxTravelTicks+1), strconv.Itoa(economysim.MaxTravelTicks),
			func(m *economysim.Migration) { m.MoveHorizon = economysim.MaxTravelTicks + 1 }),
		module("Migration.MoveOutShare", "above 10 000", "move_out_share", "10001", "10000", func(m *economysim.Migration) { m.MoveOutShare = 10_001 }),
		module("Migration.MoveInShare", "below 0", "move_in_share", "-1", "0", func(m *economysim.Migration) { m.MoveInShare = -1 }),
		row("Migration.Regions", "more rows than MaxRegions", economysim.ErrTooMany, []string{"modules", "migration", "regions"},
			func(m *economysim.Migration) { m.Regions = make([]economysim.RegionMove, economysim.MaxRegions+1) }),
		row("RegionMove.Region", "a region that is not defined", economysim.ErrUnknownRegion, []string{"modules", "migration", "region", "nowhere", "region"},
			func(m *economysim.Migration) { m.Regions = []economysim.RegionMove{{Region: "nowhere"}} }),
		row("RegionMove.OutShare", "above 10 000", economysim.ErrOutOfBounds, []string{"modules", "migration", "region", "north", "out_share"},
			func(m *economysim.Migration) {
				m.Regions = []economysim.RegionMove{{Region: "north", OutShare: 10_001}}
			}),
		row("RegionMove.InShare", "below 0", economysim.ErrOutOfBounds, []string{"modules", "migration", "region", "north", "in_share"},
			func(m *economysim.Migration) { m.Regions = []economysim.RegionMove{{Region: "north", InShare: -1}} }),
	}
}

// TestValidate_MigrationBounds verifies that each field of the module is accepted
// at both ends of its range and that a configuration that sets no MoveSpeed (the
// zero value DefaultMigration leaves) is ErrOutOfBounds at its path.
//
// Requirement: FR-MIG-001, FR-MIG-003, FR-CFG-078
func TestValidate_MigrationBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(m *economysim.Migration)
		want   error
	}{
		{"the defaults with a speed", func(m *economysim.Migration) {}, nil},
		{"MoveSpeed at its bound", func(m *economysim.Migration) { m.MoveSpeed = economysim.MaxSpeed }, nil},
		{"MoveReviewTicks at its bounds", func(m *economysim.Migration) { m.MoveReviewTicks = economysim.MaxMoveReviewTicks }, nil},
		{"MoveReviewTicks 1", func(m *economysim.Migration) { m.MoveReviewTicks = 1 }, nil},
		{"MoveBelow, MoveGain and MoveWidth 0", func(m *economysim.Migration) { m.MoveBelow, m.MoveGain, m.MoveWidth = 0, 0, 0 }, nil},
		{"MoveCooldown at its bounds", func(m *economysim.Migration) { m.MoveCooldown = economysim.MaxWelfareWindow }, nil},
		{"MoveHorizon at its bounds", func(m *economysim.Migration) { m.MoveHorizon = economysim.MaxTravelTicks }, nil},
		{"MoveHorizon 2", func(m *economysim.Migration) { m.MoveHorizon = 2 }, nil},
		{"both shares 0", func(m *economysim.Migration) { m.MoveOutShare, m.MoveInShare = 0, 0 }, nil},
		{"MoveSpeed missing", func(m *economysim.Migration) { m.MoveSpeed = 0 }, economysim.ErrOutOfBounds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := migrationConfig()
			tt.mutate(c.Modules.Migration)
			if err := c.Validate(); !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Errorf("Validate(%s) = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestValidate_MigrationNeedsInformation verifies that the module with
// Modules.Information nil is ErrModuleDependency at the module, and that the flag
// Role.Fixed with the module off is allowed and does nothing.
//
// Requirement: FR-MIG-002, FR-CFG-078
func TestValidate_MigrationNeedsInformation(t *testing.T) {
	t.Parallel()
	c := migrationConfig()
	c.Modules.Information = nil
	for i := range c.Roles {
		c.Roles[i].Informed = false
	}
	err := c.Validate()
	var ve *economysim.ValidationError
	if !errors.Is(err, economysim.ErrModuleDependency) || !errors.As(err, &ve) {
		t.Fatalf("Validate(migration without information) = %v, want ErrModuleDependency", err)
	}
	if got := join(ve.Path); got != "modules/migration" {
		t.Errorf("Validate(migration without information) path = %s, want modules/migration", got)
	}

	off := informationConfig()
	for i := range off.Roles {
		off.Roles[i].Fixed = true
	}
	if err := off.Validate(); err != nil {
		t.Errorf("Validate(Fixed roles with the module off) = %v, want nil", err)
	}
}

// join joins the segments of a path with a slash.
func join(p []string) string {
	s := ""
	for i, seg := range p {
		if i > 0 {
			s += "/"
		}
		s += seg
	}
	return s
}

// TestValidate_MigrationRegionRows verifies the rows of spec-migration.md 11.2:
// an absent row is legal, a row equal to the world's values is legal, rows given
// in any order are legal, two rows for a region are ErrDuplicateID, and a region
// that is not defined is ErrUnknownRegion.
//
// Requirement: FR-MIG-004, FR-CFG-078
func TestValidate_MigrationRegionRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows []economysim.RegionMove
		want error
		path string
	}{
		{"no row", nil, nil, ""},
		{"a row equal to the world's values", []economysim.RegionMove{{Region: "north", OutShare: 50, InShare: 100}}, nil, ""},
		{"a region that takes no immigrant", []economysim.RegionMove{{Region: "north", OutShare: 50, InShare: 0}}, nil, ""},
		{"rows out of order", []economysim.RegionMove{{Region: "south", OutShare: 1, InShare: 1}, {Region: "north", OutShare: 2, InShare: 2}}, nil, ""},
		{"two rows for one region", []economysim.RegionMove{{Region: "north"}, {Region: "north"}}, economysim.ErrDuplicateID, "modules/migration/region/north/region"},
		{"an unknown region", []economysim.RegionMove{{Region: "east"}}, economysim.ErrUnknownRegion, "modules/migration/region/east/region"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := migrationConfig()
			c.Modules.Migration.Regions = tt.rows
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			var ve *economysim.ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) || join(ve.Path) != tt.path {
				t.Errorf("Validate(%s) = %v, want %v at %s", tt.name, err, tt.want, tt.path)
			}
		})
	}
}

// TestValidate_MigrationDerived verifies the derived checks of spec-migration.md
// 12.2: a mobile role whose candidates (|K(role)| times the other regions) pass
// the bound of one view is ErrTooMany at the role, a role that is
// Fixed is not counted, and the households of the largest region times the
// largest share past MaxQuantity are ErrOverflow. Neither is computed while a
// bound already failed.
//
// Requirement: FR-MIG-040, FR-CFG-078
func TestValidate_MigrationDerived(t *testing.T) {
	t.Parallel()
	many := func(c *economysim.Config) {
		// The cook's K is at least five commodities, so 5 times 999 other regions
		// passes MaxRumoursPerView.
		grow(c, economysim.MaxRegions-len(c.Regions))
		for i := range c.Roles {
			c.Roles[i].Informed = false
			c.Roles[i].Fixed = c.Roles[i].ID != "cook"
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   string
	}{
		{"a valid configuration", func(c *economysim.Config) {}, nil, ""},
		{"too many candidates for one view", func(c *economysim.Config) {
			many(c)
			for i := range c.Roles {
				if c.Roles[i].ID == "cook" {
					c.Roles[i].Recipes = nil
					for _, r := range c.Recipes {
						c.Roles[i].Recipes = append(c.Roles[i].Recipes, r.ID)
					}
				}
			}
		}, economysim.ErrTooMany, "roles/cook/recipes"},
		{"a fixed role is not counted", func(c *economysim.Config) {
			many(c)
			for i := range c.Roles {
				if c.Roles[i].ID == "cook" {
					c.Roles[i].Recipes = nil
					for _, r := range c.Recipes {
						c.Roles[i].Recipes = append(c.Roles[i].Recipes, r.ID)
					}
					c.Roles[i].Fixed = true
				}
			}
		}, nil, ""},
		{"a group of 100 000 000 households at 50 basis points", func(c *economysim.Config) {
			c.Groups[0].Households = 100_000_000
		}, economysim.ErrOverflow, "modules/migration/move_out_share"},
		{"a row share counts", func(c *economysim.Config) {
			c.Modules.Migration.MoveOutShare, c.Modules.Migration.MoveInShare = 1, 1
			c.Groups[0].Households = 100_000_000
			c.Modules.Migration.Regions = []economysim.RegionMove{{Region: "north", OutShare: 50, InShare: 1}}
		}, economysim.ErrOverflow, "modules/migration/move_out_share"},
		{"a bound that failed first", func(c *economysim.Config) {
			c.Groups[0].Households = 100_000_000
			c.Modules.Migration.MoveSpeed = 0
		}, economysim.ErrOutOfBounds, "modules/migration/move_speed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := migrationConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			var ve *economysim.ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) {
				t.Fatalf("Validate(%s) = %v, want %v", tt.name, err, tt.want)
			}
			if tt.path != "" && join(ve.Path) != tt.path {
				t.Errorf("Validate(%s) path = %s, want %s", tt.name, join(ve.Path), tt.path)
			}
		})
	}
}

// TestWarnings_Migration verifies each of the seven warnings of spec-migration.md
// 12.3 on its condition, at its path, with the count of migration_provision and
// migration_far (CoverTicks 2, horizon 3: the journey of 3 ticks from north to
// south is the one pair of each), and that none fails validation.
//
// Requirement: FR-MIG-039, FR-MIG-043, FR-CFG-080
func TestWarnings_Migration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		want   []string
		count  int64
	}{
		{"no edge", func(c *economysim.Config) { c.Edges = nil }, "migration_without_routes", []string{"modules/migration"}, 0},
		{"edges", func(c *economysim.Config) {}, "migration_without_routes", nil, 0},
		{"every role fixed", func(c *economysim.Config) {
			for i := range c.Roles {
				c.Roles[i].Fixed = true
			}
		}, "migration_no_mobile_role", []string{"modules/migration"}, 0},
		{"a mobile role", func(c *economysim.Config) {}, "migration_no_mobile_role", nil, 0},
		{"a cohort of two households", func(c *economysim.Config) { c.Groups[0].Scale = 2 }, "migration_cohort", []string{"roles/cook"}, 0},
		{"a group of the world's scale 1", func(c *economysim.Config) {}, "migration_cohort", nil, 0},
		{"a journey above CoverTicks", func(c *economysim.Config) { c.World.CoverTicks = 2 }, "migration_provision", []string{"modules/migration/move_speed"}, 1},
		{"journeys within CoverTicks", func(c *economysim.Config) { c.World.CoverTicks = 3 }, "migration_provision", nil, 0},
		{"a journey at the horizon", func(c *economysim.Config) { c.Modules.Migration.MoveHorizon = 3 }, "migration_far", []string{"modules/migration/move_horizon"}, 1},
		{"journeys within the horizon", func(c *economysim.Config) { c.Modules.Migration.MoveHorizon = 4 }, "migration_far", nil, 0},
		// A share of 50 over a window of w ticks is 50 w: above 3 * 100 = 300 for any window of 7 ticks or more, and
		// not above 3 * 10 000 = 30 000 for a window below 600.
		{"a flow that can reverse", func(c *economysim.Config) { c.Modules.Migration.MoveGain = 100 }, "migration_flow_window", []string{"modules/migration/move_out_share"}, 0},
		{"a flow within three times the margin", func(c *economysim.Config) { c.Modules.Migration.MoveGain = 10_000 }, "migration_flow_window", nil, 0},
		// 10 000 / 30 = 333: a share of 333 and an in share of 10 000 leave only the rotation.
		{"no cap that can bind", func(c *economysim.Config) {
			c.Modules.Migration.MoveOutShare, c.Modules.Migration.MoveInShare = 333, 10_000
		}, "migration_uncapped", []string{"modules/migration"}, 0},
		{"an out share of 332", func(c *economysim.Config) {
			c.Modules.Migration.MoveOutShare, c.Modules.Migration.MoveInShare = 332, 10_000
		}, "migration_uncapped", nil, 0},
		{"an in share below 10 000", func(c *economysim.Config) {
			c.Modules.Migration.MoveOutShare, c.Modules.Migration.MoveInShare = 333, 9_999
		}, "migration_uncapped", nil, 0},
		{"the module off", func(c *economysim.Config) { c.Modules.Migration = nil; c.Edges = nil }, "migration_without_routes", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := migrationConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails validation", tt.name, err)
			}
			got := warnPaths(c, tt.rule)
			if len(got) != len(tt.want) || (len(got) == 1 && got[0] != tt.want[0]) {
				t.Errorf("Warnings(%s) %s at %v, want %v", tt.name, tt.rule, got, tt.want)
			}
			for _, w := range c.Warnings() {
				if w.Rule == tt.rule && w.Count != tt.count {
					t.Errorf("Warnings(%s) %s Count = %d, want %d", tt.name, tt.rule, w.Count, tt.count)
				}
			}
		})
	}
}

// TestOverlay_Migration verifies the overlay of the module over each variant that
// has a trade and an information overlay: the layers load, the configuration
// validates, the journeys are 1 or 2 ticks (no migration_provision or
// migration_far warning), and the world builds.
//
// Requirement: FR-MIG-001, FR-MIG-039
func TestOverlay_Migration(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"two", "three"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			layers := append(packInformation(variant), "overlay/migration.toml")
			cfg := loadValidPack(t, layers...)
			if cfg.Modules.Migration == nil || cfg.Modules.Migration.MoveSpeed != 100 {
				t.Fatalf("Modules.Migration of %v = %+v, want it on at speed 100", layers, cfg.Modules.Migration)
			}
			for _, rule := range []string{"migration_provision", "migration_far"} {
				if got := warnPaths(cfg, rule); len(got) != 0 {
					t.Errorf("Warnings(%v) gave %s at %v, want none", layers, rule, got)
				}
			}
			if _, err := economysim.NewWorld(cfg); err != nil {
				t.Errorf("NewWorld(%v) error = %v, want nil", layers, err)
			}
		})
	}
}
