package economysim_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// glossaryTerms returns the terms of the Glossary section of the doc comment
// of doc.go: the text before the colon of each "  - Term: ..." item.
func glossaryTerms(t *testing.T) []string {
	t.Helper()
	text := readFile(t, "doc.go")
	i := strings.Index(text, "// # Glossary")
	if i < 0 {
		t.Fatalf("no Glossary section in doc.go")
	}
	re := regexp.MustCompile(`(?m)^//   - ([^:\n]+):`)
	var terms []string
	for _, m := range re.FindAllStringSubmatch(text[i:], -1) {
		terms = append(terms, m[1])
	}
	return terms
}

// exportedTypes returns the exported type names declared in one file.
func exportedTypes(t *testing.T, file string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	var names []string
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			if ts := s.(*ast.TypeSpec); ts.Name.IsExported() {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names
}

// TestGlossary_Vocabulary verifies the glossary of doc.go against the
// vocabulary of architecture.md 3.2: every term of the list is a glossary
// term or is named in notYetTerms (a term the code has not used yet), a
// notYetTerms entry that is a glossary term or an exported type of the root is
// stale, and no glossary term appears twice.
//
// Guard: architecture.md 3.2 says a new domain term is added to the glossary
// before use. A term used by the code with no entry, or an entry that was
// dropped, would let the shared vocabulary drift unnoticed.
func TestGlossary_Vocabulary(t *testing.T) {
	t.Parallel()
	rules := readFile(t, "doc/rules/architecture.md")
	i := strings.Index(rules, "\n3.2. Use a single ubiquitous vocabulary")
	j := strings.Index(rules, "A new domain term **MUST**")
	if i < 0 || j < i {
		t.Fatalf("vocabulary list of 3.2 not found in architecture.md")
	}
	var vocab []string
	for _, m := range backtickRe.FindAllStringSubmatch(rules[i:j], -1) {
		vocab = append(vocab, m[1])
	}
	if len(vocab) < 15 {
		t.Fatalf("parsed %d terms of 3.2, want at least 15", len(vocab))
	}

	// The terms of 3.2 that the code of the foundations does not use yet:
	// they arrive with the world, the market and the module that owns them
	// (roadmap-v1.md, Stage 1 and later). Delete an entry when its type does.
	notYetTerms := []string{
		"Run", "Strategy", "Spawner", "Snapshot", "Policy",
	}

	terms := glossaryTerms(t)
	if len(terms) < 20 {
		t.Fatalf("parsed %d glossary terms of doc.go, want at least 20", len(terms))
	}
	seen := map[string]bool{}
	for _, term := range terms {
		if seen[term] {
			t.Errorf("glossary term %q appears twice", term)
		}
		seen[term] = true
	}
	var rootTypes []string
	for _, f := range []string{"value.go", "config.go", "event.go", "observer.go", "option.go", "order.go", "clearing.go"} {
		rootTypes = append(rootTypes, exportedTypes(t, f)...)
	}
	for _, v := range vocab {
		switch deferred := slices.Contains(notYetTerms, v); {
		case !seen[v] && !deferred:
			t.Errorf("term %q of architecture.md 3.2 is neither a glossary term of doc.go nor in notYetTerms", v)
		case seen[v] && deferred:
			t.Errorf("notYetTerms lists %q, which is a glossary term: delete the stale entry", v)
		case deferred && slices.Contains(rootTypes, v):
			t.Errorf("notYetTerms lists %q, which the root declares as a type: add it to the glossary", v)
		}
	}
	for _, n := range notYetTerms {
		if !slices.Contains(vocab, n) {
			t.Errorf("notYetTerms lists %q, which is not a term of 3.2 (a typo)", n)
		}
	}
}

// TestGlossary_DeclaredTypesNamed verifies that every exported type declared
// in value.go, calendar.go and config.go (the Config and its section structs) and every
// type of event.go (the Event, the event types and their row types) is named,
// as the identifier, in doc.go, case-sensitively, so that a new type cannot
// reach the API without a word in the glossary.
//
// Guard: architecture.md 3.2, the glossary is the place the vocabulary of the
// code is defined; a type missing from it is a term used before it was added.
func TestGlossary_DeclaredTypesNamed(t *testing.T) {
	t.Parallel()
	doc := readFile(t, "doc.go")
	for _, file := range []string{"value.go", "config.go", "event.go", "calendar.go", "order.go", "clearing.go"} {
		names := exportedTypes(t, file)
		if len(names) == 0 {
			t.Fatalf("%s declares no exported type: the parse is looking at the wrong place", file)
		}
		for _, name := range names {
			re := regexp.MustCompile(`\b` + name + `\b`)
			if !re.MatchString(doc) {
				t.Errorf("type %s of %s is not named in the doc.go glossary", name, file)
			}
		}
	}
}
