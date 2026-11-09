package calibration

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	reOverlayHeader = regexp.MustCompile(`^# calibration C[0-9][abc]? attempt [1-9][0-9]* of lineage [0-9a-f]{8}: see \S+$`)
	reOverlayKey    = regexp.MustCompile(`^([a-z][a-z0-9_]*) = -?[0-9]+$`)
)

// registryClass maps a registry key to its class letter (K, D, S or P): a knob
// row's own class, and the class of the group that lists a field. A key the
// registry does not hold is 0.
func registryClass(r *registry) func(key string) byte {
	m := map[string]byte{}
	for _, g := range r.groups {
		for _, f := range strings.Split(g[gFields], ",") {
			m[g[gScope]+"."+strings.TrimSpace(f)] = g[gClass][0]
		}
	}
	for _, k := range r.knobs {
		m[k[cKey]] = k[cClass][0]
	}
	return func(key string) byte { return m[key] }
}

// tableScope is the table-prefix rule of EXP 11.3.1: the registry scope a table
// header of a configuration document names. The deeper module tables are the ones
// the first table of VIA 11 lists; a table not here is unresolved and so refused.
func tableScope(table string) (string, bool) {
	seg := strings.Split(table, ".")
	if seg[0] == "modules" {
		return moduleScope(seg)
	}
	for _, i := range []int{1, 3} { // the instance segments of role.<id> and region.<id>.group.<role>
		if i < len(seg) {
			seg[i] = "*"
		}
	}
	scope, ok := map[string]string{"world": "World", "role.*": "Role", "region.*": "Region", "region.*.group.*": "Group"}[strings.Join(seg, ".")]
	return scope, ok
}

// moduleScope resolves modules.<m> to the module's scope and modules.<m>.<table>.<id>
// to the scope of the table.
func moduleScope(seg []string) (string, bool) {
	switch {
	case len(seg) == 2:
		return pascal(seg[1]), true
	case len(seg) >= 4:
		scope, ok := moduleTables()[seg[2]]
		return scope, ok
	}
	return "", false
}

// moduleTables maps the table name inside a module (modules.<m>.<table>.<id>) to
// its registry scope.
func moduleTables() map[string]string {
	return map[string]string{
		"mode": "Mode", "resource": "Resource", "endowment": "Endowment", "want": "Want", "taste": "Taste",
		"region_taste": "RegionTaste", "node": "OutsideNode", "control": "Control", "term": "TenureTerm",
	}
}

// overlayProblems checks one document of testdata/chains/calibrated/: it begins
// with the header comment of 9.1 (FR-CLB-051), and every key of every table is a
// K or D row of the registry (FR-CLB-050). It reads the form EXP 3.3.4 writes: a
// table header, then `key = integer` lines.
func overlayProblems(doc string, class func(key string) byte) []string {
	lines := strings.Split(strings.TrimSuffix(doc, "\n"), "\n")
	var out []string
	if !reOverlayHeader.MatchString(lines[0]) {
		out = append(out, "line 1 is not the header comment")
	}
	scope, table := "", ""
	for i, ln := range lines[1:] {
		switch {
		case ln == "" || strings.HasPrefix(ln, "#"):
		case strings.HasPrefix(ln, "[") && strings.HasSuffix(ln, "]"):
			table = ln[1 : len(ln)-1]
			s, ok := tableScope(table)
			if !ok {
				out = append(out, "line "+strconv.Itoa(i+2)+": table "+table+" names no registry scope")
			}
			scope = s
		case reOverlayKey.MatchString(ln):
			out = append(out, keyProblem(i+2, scope, reOverlayKey.FindStringSubmatch(ln)[1], class)...)
		default:
			out = append(out, "line "+strconv.Itoa(i+2)+": not a table header or an integer key")
		}
	}
	return out
}

// keyProblem says why a key of a table is not a K or D row, or nothing.
func keyProblem(line int, scope, key string, class func(string) byte) []string {
	at := "line " + strconv.Itoa(line) + ": "
	if scope == "" {
		return []string{at + key + " has no table"}
	}
	reg := scope + "." + pascal(key)
	if c := class(reg); c != 'K' && c != 'D' {
		return []string{at + reg + " is class " + string(rune(c)) + ", not K or D"}
	}
	return nil
}

// TestOverlays_Purity verifies that every document of testdata/chains/calibrated/
// holds only K and D keys and begins with the header comment (none exists yet, so
// that set passes), and that the check can fail: it passes the overlay that Overlay
// writes for the choice of 19.6 and refuses a document with a key of class S, an
// unknown key, no header, a key before any table, an unresolved table or a line it
// cannot read.
//
// Requirement: FR-CLB-050, FR-CLB-051
func TestOverlays_Purity(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	class := registryClass(parseRegistry(s.clb))
	files, err := filepath.Glob("../../testdata/chains/calibrated/*.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if ps := overlayProblems(string(b), class); len(ps) != 0 {
			t.Errorf("%s: %v", f, ps)
		}
	}
	good, err := Overlay(ovHeader(), []Change{{"World.CostFloor", "world.cost_floor", 3500}, {"Role.TargetRatio", "role.baker.target_ratio", 12500}}, class)
	if err != nil {
		t.Fatal(err)
	}
	if ps := overlayProblems(string(good), class); len(ps) != 0 {
		t.Errorf("overlayProblems(Overlay(19.6 and a role key)) = %v, want none", ps)
	}
	const header = "# calibration C4a attempt 1 of lineage 5a1c0e44: see testdata/calibration/one/c4a-1.txt\n"
	bad := map[string]string{
		"a key of class S":            header + "[world]\nscale = 4\n",
		"a key the registry lacks":    header + "[world]\nnope = 1\n",
		"no header":                   "[world]\ncost_floor = 3500\n",
		"a header of another form":    "# calibration for the pack\n[world]\ncost_floor = 3500\n",
		"a key before any table":      header + "cost_floor = 3500\n",
		"a table that names no scope": header + "[nowhere]\ncost_floor = 3500\n",
		"a line that is no key":       header + "[world]\ncost_floor: 3500\n",
		"a float":                     header + "[world]\ncost_floor = 3500.5\n",
	}
	for name, doc := range bad {
		if ps := overlayProblems(doc, class); len(ps) == 0 {
			t.Errorf("overlayProblems(%s) = none, want a problem", name)
		}
	}
}

// TestOverlays_TableScope verifies the table-prefix rule of EXP 11.3.1: the tables
// the rule maps resolve to their scope, and a table it does not know is
// unresolved, so a key under it is refused.
//
// Requirement: FR-CLB-050
func TestOverlays_TableScope(t *testing.T) {
	t.Parallel()
	for table, want := range map[string]string{
		"world": "World", "role.baker": "Role", "region.north": "Region", "region.north.group.baker": "Group",
		"modules.migration": "Migration", "modules.money_supply": "MoneySupply", "modules.transport.mode.cart": "Mode",
		"modules.external.node.far": "OutsideNode",
	} {
		if got, ok := tableScope(table); !ok || got != want {
			t.Errorf("tableScope(%q) = %q, %v, want %q", table, got, ok, want)
		}
	}
	for _, table := range []string{"", "region.north.group", "modules.x.unknown.y", "role", "modules", "world.x"} {
		if got, ok := tableScope(table); ok {
			t.Errorf("tableScope(%q) = %q, true, want it unresolved", table, got)
		}
	}
}
