package trimmedmean

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// maxFunctionLines is the longest function allowed in the package's source (review R1, finding 3.1:
// "short functions" made checkable).
const maxFunctionLines = 40

// TestSourceQuality checks the package's non-test source: every exported identifier has a doc comment,
// and no function is longer than maxFunctionLines.
func TestSourceQuality(t *testing.T) {
	files := token.NewFileSet()
	packages, err := parser.ParseDir(files, ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			checkDeclarations(t, files, file)
		}
	}
}

func checkDeclarations(t *testing.T, files *token.FileSet, file *ast.File) {
	t.Helper()
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			lines := files.Position(decl.End()).Line - files.Position(decl.Pos()).Line + 1
			if lines > maxFunctionLines {
				t.Errorf("func %s has %d lines, limit %d", decl.Name.Name, lines, maxFunctionLines)
			}
			if decl.Name.IsExported() && decl.Doc == nil {
				t.Errorf("exported func %s has no doc comment", decl.Name.Name)
			}
		case *ast.GenDecl:
			checkGeneralDeclaration(t, decl)
		}
	}
}

// checkGeneralDeclaration checks types, variables and constants: each exported name needs a doc comment
// on its own spec or on the enclosing declaration.
func checkGeneralDeclaration(t *testing.T, decl *ast.GenDecl) {
	t.Helper()
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if s.Name.IsExported() && s.Doc == nil && decl.Doc == nil {
				t.Errorf("exported type %s has no doc comment", s.Name.Name)
			}
		case *ast.ValueSpec:
			for _, name := range s.Names {
				if name.IsExported() && s.Doc == nil {
					t.Errorf("exported %s has no doc comment of its own", name.Name)
				}
			}
		}
	}
}
