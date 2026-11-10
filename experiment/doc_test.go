package experiment_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestExported_HaveDocComments verifies, over the non-test files of experiment/,
// that the package has a package comment and that every exported type, function,
// method, constant, variable and struct field has a doc comment that begins with
// its name (architecture.md 6.5, NFR-EXP-006).
//
// Requirement: NFR-EXP-006
//
// Guard: the root's TestDocComments_Contracts checks only a few fixed phrases and
// does not cover this package.
func TestExported_HaveDocComments(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked, packageDoc := 0, false
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		packageDoc = packageDoc || (f.Doc != nil && strings.HasPrefix(f.Doc.Text(), "Package experiment "))
		for _, d := range f.Decls {
			for _, p := range undocumented(d) {
				t.Errorf("%s: %s has no doc comment beginning with its name", name, p)
			}
			checked++
		}
	}
	if !packageDoc || checked < 40 {
		t.Errorf("package comment found = %v, declarations checked = %d, want a comment beginning \"Package experiment \" and at least 40", packageDoc, checked)
	}
}

// documented reports whether the comment group begins with the name, after an
// article.
func documented(g *ast.CommentGroup, name string) bool {
	if g == nil {
		return false
	}
	text := strings.TrimSpace(g.Text())
	for _, article := range []string{"A ", "An ", "The "} { // "A Group is ..." is a doc comment of Group
		text = strings.TrimPrefix(text, article)
	}
	return strings.HasPrefix(text, name)
}

// undocumented returns the exported identifiers of a declaration that lack a doc
// comment. A name in a const or var group is documented by its own comment or, for
// a group of constants, by the comment of the group.
func undocumented(d ast.Decl) []string {
	var out []string
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Name.IsExported() && (d.Recv == nil || exportedRecv(d.Recv)) && !documented(d.Doc, d.Name.Name) {
			out = append(out, "func "+d.Name.Name)
		}
	case *ast.GenDecl:
		for _, s := range d.Specs {
			switch s := s.(type) {
			case *ast.TypeSpec:
				if !s.Name.IsExported() {
					continue
				}
				if !documented(s.Doc, s.Name.Name) && !(len(d.Specs) == 1 && documented(d.Doc, s.Name.Name)) {
					out = append(out, "type "+s.Name.Name)
				}
				out = append(out, undocumentedFields(s)...)
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() && !documented(s.Doc, n.Name) && !documented(s.Comment, n.Name) && !(d.Doc != nil && (d.Lparen.IsValid() || len(d.Specs) == 1)) {
						out = append(out, n.Name)
					}
				}
			}
		}
	}
	return out
}

// exportedRecv reports whether a method's receiver type is exported.
func exportedRecv(r *ast.FieldList) bool {
	t := r.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.IsExported()
}

// undocumentedFields returns the exported fields of a struct type without a
// comment. A comment on a line of names covers all of them.
func undocumentedFields(s *ast.TypeSpec) []string {
	st, ok := s.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	var out []string
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			if n.IsExported() && f.Doc == nil && f.Comment == nil {
				out = append(out, "field "+s.Name.Name+"."+n.Name)
			}
		}
	}
	return out
}
