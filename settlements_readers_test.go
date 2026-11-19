package economysim_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestViews_CarryNoClass verifies FR-SET-018: no view a strategy, a spawner or an
// outcome reads carries a settlement class, a flag, a limit or a ceiling. The spawner's
// view carries the role's openness (Closed) and headroom (Headroom, NoLimit) and nothing
// of the class itself.
//
// Requirement: FR-SET-018
func TestViews_CarryNoClass(t *testing.T) {
	t.Parallel()
	forbidden := []string{"Class", "Flag", "Limit", "Needs", "Ceiling"}
	// NoLimit is the ceiling's headroom flag; OutcomeView.Limit is a trade's limit price, unrelated.
	allowed := map[string]bool{"RoleView.NoLimit": true, "OutcomeView.Limit": true}
	views := []reflect.Type{
		reflect.TypeOf(economysim.RegionView{}), reflect.TypeOf(economysim.RoleView{}), reflect.TypeOf(economysim.RegionAgent{}),
		reflect.TypeOf(economysim.AgentView{}), reflect.TypeOf(economysim.CommodityView{}), reflect.TypeOf(economysim.OutcomeView{}),
		reflect.TypeOf(economysim.RecipeView{}), reflect.TypeOf(economysim.EndowmentView{}), reflect.TypeOf(economysim.WantView{}),
	}
	for _, typ := range views {
		for i := range typ.NumField() {
			name := typ.Field(i).Name
			for _, bad := range forbidden {
				if strings.Contains(name, bad) && !allowed[typ.Name()+"."+name] {
					t.Errorf("%s.%s names a settlement class, flag, limit or ceiling, want only openness and headroom in a view", typ.Name(), name)
				}
			}
			if typ.Field(i).Type == reflect.TypeOf(economysim.SettlementClassID(0)) {
				t.Errorf("%s.%s is a SettlementClassID, want none in a view", typ.Name(), name)
			}
		}
	}
}

// readsClassByRow reports whether the file may read the class: a test file, a file of the
// settlements module, and infrastructure_calls.go, which is the row H8 of 6.3 (the class of
// a facility may need a flag of the region's class, read at AddFacilities through regionHas
// and nowhere else).
func readsClassByRow(f string) bool {
	return strings.HasSuffix(f, "_test.go") || strings.HasPrefix(f, "settlements") || f == "infrastructure_calls.go"
}

// TestReaders_StaticCheck verifies FR-SET-019: the readers of a class are exactly the
// adopted rows of spec-settlements.md 6.3. The class is read through regionHas and
// regionLimit only, which the settlements files alone call (and, for regionHas, the
// calls of the infrastructure module, the row H8), and no other file reads the
// class table or the class of a region off the module's state.
//
// Requirement: FR-SET-019
//
// Guard: a hook that read the class directly (an H7 or H9 the spec declines) would
// bypass the rule that a reader reads at the position its row states.
func TestReaders_StaticCheck(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob() = %v, want nil", err)
	}
	for _, f := range files {
		if readsClassByRow(f) {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) = %v, want nil", f, err)
		}
		for n, line := range strings.Split(string(src), "\n") {
			if call := classReadIn(f, line); call != "" {
				t.Errorf("%s:%d reads the class with %q, want only the settlements files to (spec-settlements.md 6.3)", f, n+1, call)
			}
		}
	}
}

// classReadIn returns the call through which a line of the file f reads the class of a
// region, or "" when it reads none. The institutions round reads the flag of a body's
// Needs through regionHas, at P1.9a, which is the reader of SET's hook H6
// (spec-institutions.md 3.5.2), so that call is not a read of the class there.
func classReadIn(f, line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return ""
	}
	for _, call := range []string{"regionHas(", "regionLimit(", "set.class[", "set.tab", "set.e[", "set.l["} {
		if f == "institutions_round.go" && call == "regionHas(" {
			continue
		}
		if strings.Contains(line, call) {
			return call
		}
	}
	return ""
}

// TestEntryLimited_RoleCeilingField verifies that the two counts of the ceiling are
// fields of the events, plain integers: EntryLimited.RoleCeiling and
// PopulationChanged.BirthsCeiling.
//
// Requirement: FR-SPN-056, FR-POP-041
func TestEntryLimited_RoleCeilingField(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		typ   reflect.Type
		field string
	}{
		{reflect.TypeOf(economysim.EntryLimited{}), "RoleCeiling"},
		{reflect.TypeOf(economysim.PopulationChanged{}), "BirthsCeiling"},
	} {
		f, ok := tt.typ.FieldByName(tt.field)
		if !ok || f.Type.Kind() != reflect.Int64 {
			t.Errorf("%s.%s = %v, %v, want an int64 field", tt.typ.Name(), tt.field, f.Type, ok)
		}
	}
}
