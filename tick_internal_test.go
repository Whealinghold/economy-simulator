package economysim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// specPosition is one row of the tables of spec-world.md 6: the identifier and
// the first area code of the Owner cell.
type specPosition struct{ id, owner string }

var (
	// positionRowRe matches the ID cell of a position row, P0.1 to P8.6 with an
	// optional letter.
	positionRowRe = regexp.MustCompile(`^P\d\.\d+[a-z]?$`)
	// ownerCodeRe matches the first area code of an Owner cell.
	ownerCodeRe = regexp.MustCompile(`^[A-Z]{3}\b`)
	// positionCallRe matches the name of a position method: p, the phase digit,
	// the position digit, an optional lowercase letter and an optional
	// capitalised word (p02, p21Complete, p33b, p84Audit).
	positionCallRe = regexp.MustCompile(`^p(\d)(\d)([a-z]?)(?:[A-Z]\w*)?$`)
)

// readSpecPositions reads the position rows of section 6 of spec-world.md by a
// plain text parse, in file order, from the heading "## 6. The tick" to the
// heading "### 6.11". The owner is the first area code of the Owner cell; "this
// spec" reads as WLD.
func readSpecPositions(t *testing.T) []specPosition {
	t.Helper()
	raw, err := os.ReadFile("doc/specs/spec-world.md")
	if err != nil {
		t.Fatalf("read spec-world.md: %v", err)
	}
	var rows []specPosition
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "## 6. The tick"):
			in = true
		case strings.HasPrefix(line, "### 6.11"):
			in = false
		case in && strings.HasPrefix(line, "| P"):
			cells := strings.Split(line, "|")
			if len(cells) < 4 {
				continue
			}
			id, owner := strings.TrimSpace(cells[1]), strings.TrimSpace(cells[3])
			if !positionRowRe.MatchString(id) {
				continue
			}
			if strings.HasPrefix(owner, "this spec") {
				owner = "WLD"
			} else {
				owner = ownerCodeRe.FindString(owner)
			}
			rows = append(rows, specPosition{id, owner})
		}
	}
	return rows
}

// straightLineCalls reads the functions positions and positionsTail of tick.go with go/parser and
// returns the position IDs of the calls made on the receiver, in source order,
// and the names of any other call it makes. The ID of p21Complete is "P2.1".
func straightLineCalls(t *testing.T, src string) (ids, others []string) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "tick.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse tick.go: %v", err)
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || (fd.Name.Name != "positions" && fd.Name.Name != "positionsTail") {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				others = append(others, "a call that is not a method of the receiver")
				return true
			}
			if sel.Sel.Name == "positionsTail" {
				return true // the hand-over from positions to its second half
			}
			if m := positionCallRe.FindStringSubmatch(sel.Sel.Name); m != nil {
				ids = append(ids, "P"+m[1]+"."+m[2]+m[3])
			} else {
				others = append(others, sel.Sel.Name)
			}
			return true
		})
	}
	return ids, others
}

// bracketIDs are the positions that run inside beginStep or endStep (D141,
// spec-world.md 6.1.12). owedBrackets are the ones of them that may still be
// listed in notBuiltPositions: none, since Phase 4 built P0.2a.
var (
	bracketIDs   = []string{"P0.1", "P0.2a", "P0.3", "P8.5", "P8.6"}
	owedBrackets = []string{}
)

// checkPositions returns what is wrong, one line each, with the list of built
// positions, the list of those not built and the IDs the straight line calls:
//
//   - a built position that is not in the spec, has another owner, is also
//     listed not built (a stale entry) or is out of the spec's order;
//   - a spec position that is neither built nor listed, and a listed one that is
//     not in the spec (a typo) or is out of order;
//   - a call in the straight line that is not a built `line` entry, a built
//     `line` entry with no call, and calls out of order;
//   - a bracket position that is a line entry, a line position that is a
//     bracket, and a bracket listed not built unless it is owed.
func checkPositions(spec []specPosition, built []position, notBuilt, calls []string) []string {
	var problems []string
	order := make([]string, len(spec))
	for i, s := range spec {
		order[i] = s.id
	}
	last := -1
	for _, p := range built {
		i := slices.Index(order, p.id)
		switch {
		case i < 0:
			problems = append(problems, p.id+": built, but not a position of the spec")
			continue
		case spec[i].owner != p.owner:
			problems = append(problems, p.id+": owner "+p.owner+", the spec says "+spec[i].owner)
		case slices.Contains(notBuilt, p.id):
			problems = append(problems, p.id+": built and listed not built (stale entry)")
		}
		if i < last {
			problems = append(problems, p.id+": out of the spec's order")
		}
		last = max(last, i)
		isBracket := slices.Contains(bracketIDs, p.id)
		if isBracket != (p.where == bracket) {
			problems = append(problems, p.id+": where does not match the bracket positions of the spec")
		}
	}
	last = -1
	for _, id := range notBuilt {
		i := slices.Index(order, id)
		if i < 0 {
			problems = append(problems, id+": listed not built, but not a position of the spec (a typo)")
			continue
		}
		if i < last {
			problems = append(problems, id+": listed out of the spec's order")
		}
		last = max(last, i)
		if slices.Contains(bracketIDs, id) && !slices.Contains(owedBrackets, id) {
			problems = append(problems, id+": a bracket position listed not built, but none is owed (owed: "+strings.Join(owedBrackets, ", ")+")")
		}
	}
	for _, id := range order {
		if !slices.ContainsFunc(built, func(p position) bool { return p.id == id }) && !slices.Contains(notBuilt, id) {
			problems = append(problems, id+": a position of the spec that is neither built nor listed not built")
		}
	}
	var lines []string
	for _, p := range built {
		if p.where == line {
			lines = append(lines, p.id)
		}
	}
	if !slices.Equal(calls, lines) {
		problems = append(problems, "the straight line calls "+strings.Join(calls, ", ")+", the line entries are "+strings.Join(lines, ", "))
	}
	return problems
}

// TestPositions_TableMatchesTheSpec verifies the position table test of
// testing.md 6.8 in its staged form: the list of tick.go and notBuiltPositions
// together are the positions of the table of spec-world.md 6, in its order and
// with its owners; the straight line of tick.go calls the `line` entries in
// order and nothing else; and the bracket entries are P0.1, P0.2a, P0.3, P8.5
// and P8.6, none of which may be listed not built.
//
// Requirement: FR-WLD-040, FR-WLD-044
func TestPositions_TableMatchesTheSpec(t *testing.T) {
	t.Parallel()
	spec := readSpecPositions(t)
	if len(spec) < 60 {
		t.Fatalf("read %d positions of spec-world.md 6, want the whole table (at least 60)", len(spec))
	}
	src, err := os.ReadFile("tick.go")
	if err != nil {
		t.Fatalf("read tick.go: %v", err)
	}
	calls, others := straightLineCalls(t, string(src))
	if len(others) > 0 {
		t.Errorf("the straight line of tick.go makes calls other than positions: %v", others)
	}
	for _, p := range checkPositions(spec, positionList[:], notBuiltPositions[:], calls) {
		t.Error(p)
	}
}

// TestPositions_SwappedCallsInTickGoFail verifies the position test against
// the real source of tick.go with two calls of its straight line swapped, the
// mutation of plan-phase-1-11.md Phase 4: the calls read out of order and the
// test names the straight line. The unswapped source passes.
//
// Requirement: FR-WLD-040
//
// Guard: a swap of two positions in tick.go changes the order of a tick and so
// the golden output; the position test is what shows it.
func TestPositions_SwappedCallsInTickGoFail(t *testing.T) {
	t.Parallel()
	spec := readSpecPositions(t)
	raw, err := os.ReadFile("tick.go")
	if err != nil {
		t.Fatalf("read tick.go: %v", err)
	}
	const a, b = "w.p21Complete()", "w.p12Anchors()"
	src := string(raw)
	if !strings.Contains(src, a) || !strings.Contains(src, b) {
		t.Fatalf("tick.go does not call %s and %s, want both in the straight line", a, b)
	}
	swapped := strings.NewReplacer(a, b, b, a).Replace(src)
	for _, tt := range []struct {
		name    string
		src     string
		wantBad bool
	}{{"as written", src, false}, {"two positions swapped", swapped, true}} {
		calls, _ := straightLineCalls(t, tt.src)
		problems := checkPositions(spec, positionList[:], notBuiltPositions[:], calls)
		if got := len(problems) > 0; got != tt.wantBad {
			t.Errorf("checkPositions(tick.go %s) = %v, want a problem: %t", tt.name, problems, tt.wantBad)
		}
	}
}

// TestPositions_TheCheckCanFail verifies the position table test against copies
// of the real data that are edited the way a change goes wrong: a real position
// removed from both lists, a built position that is also listed not built, a
// typo in the list, a call with no entry, an entry with no call, a wrong owner,
// two positions swapped, a bracket listed not built that is not owed and a line
// entry marked as a bracket.
//
// Requirement: FR-WLD-040
//
// Guard: a test that cannot fail proves nothing, and the staged form of the
// table is only safe if a stale or missing entry is seen.
func TestPositions_TheCheckCanFail(t *testing.T) {
	t.Parallel()
	spec := readSpecPositions(t)
	src, err := os.ReadFile("tick.go")
	if err != nil {
		t.Fatalf("read tick.go: %v", err)
	}
	calls, _ := straightLineCalls(t, string(src))
	tests := []struct {
		name   string
		edit   func(built []position, notBuilt, calls []string) ([]position, []string, []string)
		wantIn string // a fragment of the problem the edit must raise
	}{
		{"unedited data", func(b []position, n, c []string) ([]position, []string, []string) { return b, n, c }, ""},
		{"a real position removed from both lists", func(b []position, n, c []string) ([]position, []string, []string) {
			return b, slices.DeleteFunc(n, func(id string) bool { return id == "P4.1a" }), c
		}, "P4.1a: a position of the spec that is neither built nor listed"},
		{"a built position also listed not built", func(b []position, n, c []string) ([]position, []string, []string) {
			return b, append(n, "P0.2"), c
		}, "P0.2: built and listed not built"},
		{"a typo in the list", func(b []position, n, c []string) ([]position, []string, []string) {
			return b, append(n, "P9.9"), c
		}, "P9.9: listed not built, but not a position"},
		{"a call with no entry", func(b []position, n, c []string) ([]position, []string, []string) {
			return b, n, append(c, "P4.1")
		}, "the straight line calls"},
		{"an entry with no call", func(b []position, n, c []string) ([]position, []string, []string) {
			return b, n, nil
		}, "the straight line calls"},
		{"a wrong owner", func(b []position, n, c []string) ([]position, []string, []string) {
			b[1].owner = "CLR"
			return b, n, c
		}, "P0.2: owner CLR"},
		{"two built positions swapped", func(b []position, n, c []string) ([]position, []string, []string) {
			b[0], b[1] = b[1], b[0]
			return b, n, c
		}, "out of the spec's order"},
		{"a bracket listed not built that is not owed", func(b []position, n, c []string) ([]position, []string, []string) {
			return b[1:], append([]string{"P0.1"}, n...), c
		}, "a bracket position listed not built"},
		{"a line entry marked as a bracket", func(b []position, n, c []string) ([]position, []string, []string) {
			b[1].where = bracket
			return b, n, nil
		}, "where does not match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			built, notBuilt, c := tt.edit(slices.Clone(positionList[:]), slices.Clone(notBuiltPositions[:]), slices.Clone(calls))
			problems := checkPositions(spec, built, notBuilt, c)
			if tt.wantIn == "" {
				if len(problems) != 0 {
					t.Errorf("checkPositions(real data) = %v, want none", problems)
				}
				return
			}
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, tt.wantIn) }) {
				t.Errorf("checkPositions(%s) = %v, want a problem containing %q", tt.name, problems, tt.wantIn)
			}
		})
	}
}

// mapRanges returns the places in src where a range statement ranges over a
// value that is declared or built as a map in the same source: a field,
// variable or parameter of a map type, a name assigned a map literal or a make
// of a map, or a map literal or make ranged over directly. It is a plain parse
// without type information, which is what FR-WLD-041's text check asks for.
func mapRanges(t *testing.T, name, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	isMapExpr := func(e ast.Expr) bool {
		switch e := e.(type) {
		case *ast.CompositeLit:
			_, ok := e.Type.(*ast.MapType)
			return ok
		case *ast.CallExpr:
			if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "make" && len(e.Args) > 0 {
				_, ok := e.Args[0].(*ast.MapType)
				return ok
			}
		}
		return false
	}
	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			if _, ok := n.Type.(*ast.MapType); ok {
				for _, id := range n.Names {
					names = append(names, id.Name)
				}
			}
		case *ast.ValueSpec:
			_, typed := n.Type.(*ast.MapType)
			for i, id := range n.Names {
				if typed || (i < len(n.Values) && isMapExpr(n.Values[i])) {
					names = append(names, id.Name)
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && i < len(n.Rhs) && isMapExpr(n.Rhs[i]) {
					names = append(names, id.Name)
				}
			}
		}
		return true
	})
	var found []string
	ast.Inspect(f, func(n ast.Node) bool {
		rs, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		bad := isMapExpr(rs.X)
		switch x := rs.X.(type) {
		case *ast.Ident:
			bad = slices.Contains(names, x.Name)
		case *ast.SelectorExpr:
			bad = slices.Contains(names, x.Sel.Name)
		}
		if bad {
			found = append(found, fset.Position(rs.Pos()).String())
		}
		return true
	})
	return found
}

// worldFiles are the files of the world row that the map guard reads. Each
// phase that adds a file of the row appends it here.
var worldFiles = []string{"world.go", "tick.go", "phase_start.go", "phase_production.go", "fanin.go", "market.go", "construct.go", "construct_warm.go", "phase_trade.go", "phase_outcomes.go", "phase_exit.go", "snapshot.go"}

// TestWorldFiles_NoRangeOverAMap verifies that the files of the world row range
// over no map (FR-WLD-041): the position test cannot show it, because a map
// range would change results only by its order. The checker is itself checked
// on source that does range over a map, held in a field, a variable, a
// parameter and a literal.
//
// Requirement: FR-WLD-041
//
// Guard: architecture.md 7.5, no iteration over a map may influence a result; a
// range over a map in a step is a defect that shows only as an occasional
// difference between two runs.
func TestWorldFiles_NoRangeOverAMap(t *testing.T) {
	t.Parallel()
	for _, name := range worldFiles {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if found := mapRanges(t, name, string(raw)); len(found) > 0 {
			t.Errorf("%s ranges over a map at %v, want none (architecture.md 7.5)", name, found)
		}
	}
	const bad = `package p
type s struct{ m map[string]int }
func f(a map[int]int, x s) {
	v := make(map[int]int)
	var w = map[int]int{}
	for range a {}
	for range x.m {}
	for range v {}
	for range w {}
	for range map[int]int{} {}
	for range []int{} {}
}`
	if got := mapRanges(t, "bad.go", bad); len(got) != 5 {
		t.Errorf("mapRanges(bad) found %d ranges over a map (%v), want 5", len(got), got)
	}
}
