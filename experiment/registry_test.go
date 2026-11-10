package experiment_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/Whealinghold/economy-sim/experiment"
)

// paramRows reads the parameter-owner table of doc/specs/README.md as text: a row
// is "| `Name` | Scope (note) | spec |", and the result maps the scope to the
// names of its rows.
func paramRows(t testing.TB) map[string]map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "doc", "specs", "README.md"))
	if err != nil {
		t.Fatalf("read the parameter-owner table: %v", err)
	}
	rows := map[string]map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 4 || !strings.HasPrefix(strings.TrimSpace(cells[1]), "`") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(cells[1]), "`"), "`")
		scope := strings.Fields(strings.NewReplacer("(", " (").Replace(cells[2]))
		if name == "" || len(scope) == 0 {
			continue
		}
		if rows[scope[0]] == nil {
			rows[scope[0]] = map[string]bool{}
		}
		rows[scope[0]][name] = true
	}
	return rows
}

// snake spells a Go field name as the loader spells its key: an acronym is one
// word (RateBP is rate_bp, ID is id). The exceptions of spec-config.md 6.2.2 are
// not handled: a path that needs one fails here and is added when it appears.
func snake(name string) string {
	var out []rune
	rs := []rune(name)
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]) ||
			(i+1 < len(rs) && unicode.IsLower(rs[i+1]))) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

// scopeOf resolves the table prefix of a path to a scope of the parameter-owner
// table (spec-experiments.md 11.3.1): world to WorldParams, role.<id> to Role,
// region.<id> to Region, region.<id>.group.<id> to Group. A prefix it does not
// know is not a row yet.
func scopeOf(segs []string) (scope, key string, ok bool) {
	last := len(segs) - 1
	switch {
	case len(segs) == 2 && segs[0] == "world":
		return "WorldParams", segs[last], true
	case len(segs) == 3 && segs[0] == "role":
		return "Role", segs[last], true
	case len(segs) == 3 && segs[0] == "region":
		return "Region", segs[last], true
	case len(segs) == 5 && segs[0] == "region" && segs[2] == "group":
		return "Group", segs[last], true
	}
	return "", "", false
}

// sweptPaths returns every path an experiment sets: the path of a range factor and
// the paths of the changes of the levels of an explicit one.
func sweptPaths(e experiment.Spec) []string {
	var out []string
	for _, f := range e.Factors {
		if f.Path != "" {
			out = append(out, f.Path)
		}
		for _, l := range f.Levels {
			for _, c := range l.Changes {
				out = append(out, c.Path)
			}
		}
	}
	return out
}

// TestRegistry_SweptPathsAreRows verifies the forward direction of the registry
// link for the experiment shipped in this package, the golden's: each swept path
// resolves, by the scope of its table prefix and the snake form of the row's name,
// to a row of the parameter-owner table of doc/specs/README.md. A path the table
// does not know, or a scope the test cannot resolve, fails.
//
// Requirement: FR-EXP-072
func TestRegistry_SweptPathsAreRows(t *testing.T) {
	t.Parallel()
	rows := paramRows(t)
	if len(rows["WorldParams"]) < 20 {
		t.Fatalf("read %d WorldParams rows, want at least 20: the table parse looks at the wrong place", len(rows["WorldParams"]))
	}
	paths := sweptPaths(goldenSpec(t))
	if len(paths) == 0 {
		t.Fatal("the golden sweeps no path")
	}
	for _, p := range paths {
		scope, key, ok := scopeOf(strings.Split(p, "."))
		if !ok {
			t.Errorf("path %q: no scope of the table for its prefix", p)
			continue
		}
		found := false
		for name := range rows[scope] {
			found = found || snake(name) == key
		}
		if !found {
			t.Errorf("path %q: no row of scope %s has the snake form %q", p, scope, key)
		}
	}
}

// TestRegistry_ParseDetects verifies the helpers the registry test stands on, so it
// cannot pass by looking at nothing: the snake form, the scope of a prefix, and a
// path that is not a row.
//
// Requirement: FR-EXP-072
func TestRegistry_ParseDetects(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"BeliefStep": "belief_step", "RateBP": "rate_bp", "ID": "id", "Scale": "scale", "ExpectedWorkTicks": "expected_work_ticks"} {
		if got := snake(name); got != want {
			t.Errorf("snake(%q) = %q, want %q", name, got, want)
		}
	}
	for path, want := range map[string]string{"world.x": "WorldParams", "role.r.x": "Role", "region.n.x": "Region", "region.n.group.g.x": "Group", "wat.x": ""} {
		if scope, _, _ := scopeOf(strings.Split(path, ".")); scope != want {
			t.Errorf("scopeOf(%q) = %q, want %q", path, scope, want)
		}
	}
	rows := paramRows(t)
	if !rows["Role"]["ExpectedWorkTicks"] || rows["WorldParams"]["NoSuchParameter"] {
		t.Errorf("the table parse missed ExpectedWorkTicks of Role or invented NoSuchParameter")
	}
}
