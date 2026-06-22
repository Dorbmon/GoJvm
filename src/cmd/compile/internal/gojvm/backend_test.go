package gojvm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompilePackageRejectsCgoImport(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nimport \"C\"\nfunc main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CompilePackage(Config{
		GOOS:      "jvm",
		GOARCH:    "jvm",
		BuildMode: "exe",
		SourceFiles: []string{
			src,
		},
	})
	if err == nil || err.Error() != `go: GOOS=jvm does not support cgo; use CGO_ENABLED=0 and remove imports of "C"` {
		t.Fatalf("got error %v, want cgo diagnostic", err)
	}
}

func TestCompilePackageRejectsUnsupportedBuildMode(t *testing.T) {
	err := CompilePackage(Config{
		GOOS:      "jvm",
		GOARCH:    "jvm",
		BuildMode: "c-archive",
	})
	if err == nil {
		t.Fatal("expected buildmode error")
	}
}

func TestCompilePackageNotImplementedAfterValidation(t *testing.T) {
	err := CompilePackage(Config{
		GOOS:      "jvm",
		GOARCH:    "jvm",
		BuildMode: "exe",
	})
	if _, ok := err.(ErrNotImplemented); !ok {
		t.Fatalf("expected not implemented placeholder, got %v", err)
	}
}
