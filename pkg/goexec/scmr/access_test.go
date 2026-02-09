package scmrexec

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sourceDir returns the directory containing the current test file's package source.
func sourceDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine source directory")
	}
	return filepath.Dir(filename)
}

// TestScManagerConnectConstantExists verifies that the ScManagerConnect constant
// is defined in scmr.go with value 0x0001. The design requires this constant for
// minimal SCM permission in Init().
func TestScManagerConnectConstantExists(t *testing.T) {
	dir := sourceDir(t)
	src, err := os.ReadFile(filepath.Join(dir, "scmr.go"))
	if err != nil {
		t.Fatalf("failed to read scmr.go: %v", err)
	}

	content := string(src)
	if !strings.Contains(content, "ScManagerConnect") {
		t.Fatal("scmr.go does not define ScManagerConnect constant")
	}

	// Also verify it appears as a const with value 0x0001.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "scmr.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse scmr.go: %v", err)
	}

	found := false
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name == "ScManagerConnect" {
					if i < len(vs.Values) {
						lit, ok := vs.Values[i].(*ast.BasicLit)
						if ok && lit.Value == "0x0001" {
							found = true
						}
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("ScManagerConnect constant not found with value 0x0001")
	}
}

// TestInitUsesMinimalScmPermission verifies that Init() in module.go does not use
// ServiceAllAccess for the OpenSCMW DesiredAccess field. The design requires
// ScManagerConnect | ScManagerCreateService instead.
func TestInitUsesMinimalScmPermission(t *testing.T) {
	dir := sourceDir(t)
	src, err := os.ReadFile(filepath.Join(dir, "module.go"))
	if err != nil {
		t.Fatalf("failed to read module.go: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "module.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse module.go: %v", err)
	}

	// Find the Init method and look for OpenSCMW call's DesiredAccess field.
	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		// Look for svcctl.OpenSCMWRequest composite literal.
		sel, ok := cl.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "OpenSCMWRequest" {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "DesiredAccess" {
				continue
			}
			// The value must NOT be ServiceAllAccess.
			if ident, ok := kv.Value.(*ast.Ident); ok && ident.Name == "ServiceAllAccess" {
				t.Fatal("Init() OpenSCMW still uses ServiceAllAccess; expected minimal permission (ScManagerConnect | ScManagerCreateService)")
			}
		}
		return true
	})

	// Also verify by simple string check as a safety net.
	content := string(src)
	if idx := strings.Index(content, "OpenSCMW"); idx >= 0 {
		// Find the surrounding context (next ~200 chars).
		end := idx + 200
		if end > len(content) {
			end = len(content)
		}
		snippet := content[idx:end]
		if strings.Contains(snippet, "ServiceAllAccess") {
			t.Fatal("Init() OpenSCMW uses ServiceAllAccess; expected minimal permission")
		}
	}
}

// TestCreateUsesMinimalPermission verifies that CreateServiceW in create.go does
// not use ServiceAllAccess for DesiredAccess. The design requires ServiceStart | ServiceDelete.
func TestCreateUsesMinimalPermission(t *testing.T) {
	dir := sourceDir(t)
	src, err := os.ReadFile(filepath.Join(dir, "create.go"))
	if err != nil {
		t.Fatalf("failed to read create.go: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "create.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse create.go: %v", err)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := cl.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "CreateServiceWRequest" {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "DesiredAccess" {
				continue
			}
			if ident, ok := kv.Value.(*ast.Ident); ok && ident.Name == "ServiceAllAccess" {
				t.Fatal("CreateServiceW still uses ServiceAllAccess; expected ServiceStart | ServiceDelete")
			}
		}
		return true
	})

	// String-based safety net.
	content := string(src)
	if idx := strings.Index(content, "CreateServiceWRequest"); idx >= 0 {
		end := idx + 300
		if end > len(content) {
			end = len(content)
		}
		snippet := content[idx:end]
		if strings.Contains(snippet, "ServiceAllAccess") {
			t.Fatal("CreateServiceW uses ServiceAllAccess; expected minimal permission")
		}
	}
}

// TestChangeUsesMinimalPermission verifies that OpenServiceW in change.go does not
// use ServiceAllAccess for DesiredAccess. The design requires ServiceModifyAccess.
func TestChangeUsesMinimalPermission(t *testing.T) {
	dir := sourceDir(t)
	src, err := os.ReadFile(filepath.Join(dir, "change.go"))
	if err != nil {
		t.Fatalf("failed to read change.go: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "change.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse change.go: %v", err)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := cl.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "OpenServiceWRequest" {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "DesiredAccess" {
				continue
			}
			if ident, ok := kv.Value.(*ast.Ident); ok && ident.Name == "ServiceAllAccess" {
				t.Fatal("change.go OpenServiceW still uses ServiceAllAccess; expected ServiceModifyAccess")
			}
		}
		return true
	})

	content := string(src)
	if idx := strings.Index(content, "OpenServiceWRequest"); idx >= 0 {
		end := idx + 200
		if end > len(content) {
			end = len(content)
		}
		snippet := content[idx:end]
		if strings.Contains(snippet, "ServiceAllAccess") {
			t.Fatal("change.go OpenServiceW uses ServiceAllAccess; expected ServiceModifyAccess")
		}
	}
}

// TestOpenServiceHasDesiredAccessParam verifies that the openService method in
// module.go accepts a desiredAccess uint32 parameter. The design requires changing
// the signature from openService(ctx, name) to openService(ctx, name, desiredAccess).
func TestOpenServiceHasDesiredAccessParam(t *testing.T) {
	dir := sourceDir(t)
	src, err := os.ReadFile(filepath.Join(dir, "module.go"))
	if err != nil {
		t.Fatalf("failed to read module.go: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "module.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse module.go: %v", err)
	}

	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "openService" {
			return true
		}
		// Check that the function has a desiredAccess parameter.
		if fd.Type.Params == nil {
			return true
		}
		for _, param := range fd.Type.Params.List {
			for _, name := range param.Names {
				if name.Name == "desiredAccess" {
					found = true
				}
			}
		}
		return true
	})

	if !found {
		t.Fatal("openService method does not have desiredAccess parameter; expected signature: openService(ctx context.Context, name string, desiredAccess uint32)")
	}
}
