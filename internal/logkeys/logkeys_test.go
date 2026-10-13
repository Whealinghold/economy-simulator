package logkeys_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const rulesFile = "../../doc/rules/observability-and-logging.md"

// tableKeys reads the table of rule 3.3.2 by a plain text parse: the rows of
// the markdown table between the line that starts rule 3.3.2 and the line
// that starts "Add a key", each key being a backtick-quoted token of the first
// cell.
func tableKeys(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read %s: %v", rulesFile, err)
	}
	quoted := regexp.MustCompile("`([^`]+)`")
	keys := map[string]bool{}
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "3.3.2."):
			in = true
		case strings.HasPrefix(line, "Add a key to this table"):
			in = false
		case in && strings.HasPrefix(line, "|"):
			cells := strings.Split(line, "|")
			if len(cells) < 3 {
				continue
			}
			for _, m := range quoted.FindAllStringSubmatch(cells[1], -1) {
				keys[m[1]] = true
			}
		}
	}
	return keys
}

// constants parses the non-test source of the package and returns every
// string constant it declares, by name, and the number of imports.
func constants(t *testing.T) (consts map[string]string, imports int) {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	var files []*ast.File
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, n, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", n, err)
		}
		files = append(files, f)
	}
	consts = map[string]string{}
	{
		for _, f := range files {
			imports += len(f.Imports)
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					t.Fatalf("declaration %T in logkeys, want constants only", d)
				}
				if gd.Tok != token.CONST {
					t.Fatalf("%s declaration in logkeys, want constants only", gd.Tok)
				}
				for _, s := range gd.Specs {
					vs := s.(*ast.ValueSpec)
					for i, name := range vs.Names {
						lit, ok := vs.Values[i].(*ast.BasicLit)
						if !ok || lit.Kind != token.STRING {
							t.Fatalf("constant %s is not a string literal (no alias, no expression)", name.Name)
						}
						v, err := strconv.Unquote(lit.Value)
						if err != nil {
							t.Fatalf("constant %s: %v", name.Name, err)
						}
						consts[name.Name] = v
					}
				}
			}
		}
	}
	return consts, imports
}

// TestKeys_AreRowsOfTheTable verifies that every key constant is a key of the
// table in observability-and-logging.md 3.3.2, that the keys are lowercase
// snake_case and unique, that the package imports nothing and declares only
// constants, and that the slice's 33 keys are all present.
//
// Requirement: FR-OBS-073
//
// Guard: a key spelled differently from the table would make a log line
// unsearchable by the documented name, and a key absent from the table would
// escape the rule that a key is added to the table in the same change.
func TestKeys_AreRowsOfTheTable(t *testing.T) {
	t.Parallel()
	table := tableKeys(t)
	consts, imports := constants(t)

	if len(table) < 40 {
		t.Fatalf("parsed %d keys from the table of 3.3.2, want at least 40: the parse is broken", len(table))
	}
	if imports != 0 {
		t.Errorf("logkeys has %d imports, want 0", imports)
	}

	// The 33 keys the slice's log lines use (plan phase 2, assumption A2).
	want := strings.Fields(`tick ticks seed commodity role recipe agent price
		quantity side reason error region bound agents households unfed path rule
		config_hash prices wages groups jobs cause commodities recipes roles
		regions modules warm_start account amount`)
	if len(want) != 33 {
		t.Fatalf("test list holds %d keys, want 33", len(want))
	}

	snake := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	seen := map[string]string{}
	for name, v := range consts {
		t.Run(name, func(t *testing.T) {
			if !table[v] {
				t.Errorf("constant %s = %q is not a key of the table in %s 3.3.2", name, v, rulesFile)
			}
			if !snake.MatchString(v) {
				t.Errorf("constant %s = %q, want lowercase snake_case", name, v)
			}
		})
		if prev, dup := seen[v]; dup {
			t.Errorf("constants %s and %s share the value %q", prev, name, v)
		}
		seen[v] = name
	}
	for _, k := range want {
		if _, ok := seen[k]; !ok {
			t.Errorf("no constant has the value %q, want one of the 33 slice keys", k)
		}
	}
	if len(consts) != len(want) {
		t.Errorf("logkeys declares %d constants, want %d", len(consts), len(want))
	}
}
