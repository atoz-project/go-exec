package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNoContextTodoInCmdPackage scans all non-test Go source files in the cmd/
// package and verifies that none contain context.TODO() calls. The design requires
// all 6 occurrences to be replaced with context.Background().
func TestNoContextTodoInCmdPackage(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine source directory")
	}
	dir := filepath.Dir(filename)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read cmd directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("failed to read %s: %v", name, err)
		}

		if strings.Contains(string(src), "context.TODO()") {
			t.Errorf("%s contains context.TODO(); expected context.Background()", name)
		}
	}
}
