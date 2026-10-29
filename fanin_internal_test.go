package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// composition reads the body of the function or method fn of src with go/parser
// and returns the names of the calls it makes, in source order, and what is wrong
// with its shape: a fan-in is a composition of calls, `if` guards and composite
// literals, so a `for`, a `range`, a `switch`, an increment, a compound assignment
// or arithmetic on a value is a problem (architecture.md 2.10.4, D195). A
// comparison or a logical operator in a guard is allowed.
func composition(t *testing.T, src, fn string) (calls, problems []string) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "fanin.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var body *ast.BlockStmt
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == fn {
			body = fd.Body
		}
	}
	if body == nil {
		t.Fatalf("no function %s in the source", fn)
	}
	arithmetic := []token.Token{token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.SHL, token.SHR, token.AND, token.OR, token.XOR, token.AND_NOT}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.GoStmt, *ast.DeferStmt, *ast.IncDecStmt:
			problems = append(problems, "a statement that is not a call, a guard or a literal")
		case *ast.AssignStmt:
			if x.Tok != token.DEFINE && x.Tok != token.ASSIGN {
				problems = append(problems, "a compound assignment")
			}
		case *ast.BinaryExpr:
			if slices.Contains(arithmetic, x.Op) {
				problems = append(problems, "arithmetic: "+x.Op.String())
			}
		case *ast.CallExpr:
			switch fun := x.Fun.(type) {
			case *ast.SelectorExpr:
				calls = append(calls, fun.Sel.Name)
			case *ast.Ident:
				calls = append(calls, fun.Name)
			default:
				problems = append(problems, "a call that is neither a method nor a function")
			}
		}
		return true
	})
	return calls, problems
}

// TestFanIns_AreCompositionsOfTheExpectedCalls verifies the shape of the fan-ins
// that are built (D195, architecture.md 2.10.4): the body of fillAgentView, checkTradePlan, removeAgent,
// relocationRefusal and tradableItems is only calls, guards and literals, with no loop and no
// arithmetic, and the calls are the expected ones in the expected order. A
// fan-in has no list to append a test fake to and no hook is allowed (D166), so its
// composition is tested by shape. What the shape cannot show, that a module step
// appended later is called in its slot, is that module's own test; a module's change
// edits the expected list below in the same change.
//
// Guard: logic in a fan-in is a defect (architecture.md 2.10.4), and a reordered
// call changes the order in which the plan is checked.
//
// Requirement: FR-WLD-040
func TestFanIns_AreCompositionsOfTheExpectedCalls(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("fanin.go")
	if err != nil {
		t.Fatalf("read fanin.go: %v", err)
	}
	for _, tt := range []struct {
		fn   string
		want []string
	}{
		{"fillAgentView", []string{"fillCore", "fillWants", "fillEndowments"}},
		{"checkTradePlan", []string{"normalisePlan", "checkDiscards", "checkWants"}},
		{"removeAgent", []string{"Tick", "regionOf", "hasJob", "abandonJob", "receive", "Errorf", "mustClose", "Close", "deliver", "deliver", "pool", "dropAgent", "dropAgent", "logRemoval", "forget"}},
		{"relocationRefusal", []string{"refusal"}},
		{"tradableItems", []string{"demandItems", "subsistenceItems"}},
	} {
		t.Run(tt.fn, func(t *testing.T) {
			t.Parallel()
			calls, problems := composition(t, string(src), tt.fn)
			if len(problems) != 0 {
				t.Errorf("%s has %v, want only calls, guards and composite literals", tt.fn, problems)
			}
			if !slices.Equal(calls, tt.want) {
				t.Errorf("%s calls %v, want %v in that order", tt.fn, calls, tt.want)
			}
		})
	}
}

// TestFanIns_TheShapeCheckCanFail verifies that the check of the previous test
// sees what it exists to forbid: a loop, a range, arithmetic on a value and a
// compound assignment are problems, a comparison in a guard is not, and the calls
// come back in source order.
//
// Requirement: FR-WLD-040
func TestFanIns_TheShapeCheckCanFail(t *testing.T) {
	t.Parallel()
	const src = `package p
func (w *W) loop(xs []int) { for range xs { w.a() } }
func (w *W) sum() int { return w.x + 1 }
func (w *W) bump() { w.n += 2 }
func (w *W) guard() error {
	if err := w.a(); err != nil {
		return err
	}
	w.b()
	return nil
}`
	for _, tt := range []struct {
		fn        string
		wantBad   string
		wantCalls []string
	}{
		{"loop", "a statement that is not", []string{"a"}},
		{"sum", "arithmetic: +", nil},
		{"bump", "a compound assignment", nil},
		{"guard", "", []string{"a", "b"}},
	} {
		calls, problems := composition(t, src, tt.fn)
		if got := strings.Join(problems, ";"); !strings.Contains(got, tt.wantBad) || (tt.wantBad == "" && got != "") {
			t.Errorf("composition(%s) problems = %q, want %q", tt.fn, got, tt.wantBad)
		}
		if !slices.Equal(calls, tt.wantCalls) {
			t.Errorf("composition(%s) calls = %v, want %v", tt.fn, calls, tt.wantCalls)
		}
	}
}

// onlyOnce counts the occurrences of each needle in the production files of the
// root package given as name to source, and returns what does not occur exactly
// once, as "needle" or "needle in file" when it is somewhere else than where.
func onlyOnce(files map[string]string, needle, where string) (problems []string) {
	n := 0
	for name, src := range files {
		if c := strings.Count(src, needle); c > 0 {
			n += c
			if name != where {
				problems = append(problems, needle+" in "+name)
			}
		}
	}
	if n != 1 {
		problems = append(problems, needle+" occurs "+strconv.Itoa(n)+" times")
	}
	slices.Sort(problems)
	return problems
}

// productionFiles reads the non-test Go files of the root package.
func productionFiles(t *testing.T) map[string]string {
	t.Helper()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	files := map[string]string{}
	for _, n := range names {
		if strings.HasSuffix(n, "_test.go") {
			continue
		}
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatalf("read %s: %v", n, err)
		}
		files[n] = string(src)
	}
	return files
}

// TestRemoval_HasOneCallSiteOfTheCloseAndOneCompositionOfItsEvents verifies the
// one removal path (architecture.md 2.10.4, plan-phase-1-11.md Phase 7 item 5): the
// ledger's Close is called in one place of the production code, fanin.go, and the
// events AgentBankrupted and EstateOpened are built in one place, removeAgent. A
// second call site or a second composition fails it.
//
// Guard: a second removal path would drift from the order of spec-world.md 6.11
// item 8 and from the steps of spec-agents.md 3.6.
//
// Requirement: FR-AGT-008, FR-AGT-077, FR-WLD-056
func TestRemoval_HasOneCallSiteOfTheCloseAndOneCompositionOfItsEvents(t *testing.T) {
	t.Parallel()
	files := productionFiles(t)
	for _, needle := range []string{"ledger.Close(", "AgentBankrupted{", "EstateOpened{"} {
		if got := onlyOnce(files, needle, "fanin.go"); len(got) != 0 {
			t.Errorf("%q: %v, want exactly one occurrence, in fanin.go", needle, got)
		}
	}
}

// TestOnlyOnce_SeesASecondPathAndAMissingOne verifies that the guard above can
// fail: a second occurrence, an occurrence elsewhere and none at all are each a
// problem, and exactly one in the right file is not.
//
// Requirement: FR-AGT-008
func TestOnlyOnce_SeesASecondPathAndAMissingOne(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"one", map[string]string{"fanin.go": "l.Close(x)", "a.go": ""}, nil},
		{"two in the file", map[string]string{"fanin.go": "l.Close(x) l.Close(y)"}, []string{"l.Close( occurs 2 times"}},
		{"elsewhere", map[string]string{"fanin.go": "l.Close(x)", "b.go": "l.Close(y)"}, []string{"l.Close( in b.go", "l.Close( occurs 2 times"}},
		{"none", map[string]string{"fanin.go": ""}, []string{"l.Close( occurs 0 times"}},
	} {
		if got := onlyOnce(tt.files, "l.Close(", "fanin.go"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: problems = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestMustClose verifies that a refusal of the ledger to close the account of a
// removed agent is a violated invariant that panics with the ledger's reason, and
// that no refusal does nothing.
//
// Requirement: FR-WLD-053
// Guard: the intake empties the account before it is closed, so a refusal means
// the removal path moved less than it should; silently continuing would lose goods
// or money and leave a closed account in the agent table.
func TestMustClose(t *testing.T) {
	t.Parallel()
	mustClose(nil)
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "invariant") || !strings.Contains(msg, "closing of a removed agent's account: account not empty") {
			t.Errorf("mustClose(error) panicked with %q, want an invariant panic carrying the ledger's reason", msg)
		}
	}()
	mustClose(errors.New("account not empty"))
	t.Errorf("mustClose(error) returned, want a panic")
}
