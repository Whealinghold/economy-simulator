package economysim_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// docOf returns the doc comment of the type or function named name in file,
// with every run of white space collapsed to one space so that a phrase
// wrapped over lines still matches.
func docOf(t *testing.T, file, name string) string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	var doc *ast.CommentGroup
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == name {
				doc = d.Doc
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.Name == name {
					doc = d.Doc
				}
			}
		}
	}
	if doc == nil {
		t.Fatalf("%s has no doc comment for %s", file, name)
	}
	return strings.Join(strings.Fields(doc.Text()), " ")
}

// TestDocComments_Contracts verifies that the doc comments that carry a
// contract a consumer relies on still say it: the calling contract of
// Observer (architecture.md 12.5, observability-and-logging.md 4.4) and the statement that a seed
// is reproducible by design and not for secrets (WithSeed), and the random-source
// and state contract of Strategy (architecture.md 7.7).
//
// Guard: these comments are the only place a consumer reads the contract. A
// rewrite that dropped one would change what an observer may do, or invite a
// seed to be used as a secret, with every behavioural test green.
func TestDocComments_Contracts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file, name string
		phrases    []string
	}{
		{"observer.go", "Observer", []string{
			"called synchronously on the goroutine that steps the world",
			"MUST NOT call back into the World",
			"MUST ignore event types it does not recognise",
			"Return quickly",
			"does not recover a panicking observer",
		}},
		{"strategy.go", "Strategy", []string{
			"MUST NOT create its own random source",
			"MUST NOT use a global one",
			"MUST NOT keep the rng it is given after the call returns",
			"MUST NOT block",
			"optional pair of the checkpoint",
			"MUST write every piece of state that can influence a result",
			"equal bytes for equal states",
			"MUST replace the whole state",
			"MUST validate its input",
			"MUST NOT be shared between worlds",
		}},
		{"option.go", "WithSeed", []string{
			"reproducible stream by design",
			"not for secrets",
			"unpredictability matters",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := docOf(t, tt.file, tt.name)
			for _, p := range tt.phrases {
				if !strings.Contains(doc, p) {
					t.Errorf("doc comment of %s does not say %q", tt.name, p)
				}
			}
		})
	}
}
