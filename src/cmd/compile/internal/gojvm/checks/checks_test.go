package checks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSourcesRejectsCGOAndUnsafe(t *testing.T) {
	dir := t.TempDir()

	cgoFile := filepath.Join(dir, "cgo.go")
	unsafeFile := filepath.Join(dir, "unsafe.go")

	if err := os.WriteFile(cgoFile, []byte("package main\nimport \"C\"\nfunc f() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unsafeFile, []byte("package main\nimport \"unsafe\"\nfunc f() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := ValidateSources([]string{dir})
	if err != nil {
		t.Fatalf("ValidateSources error: %v", err)
	}
	if got, want := len(issues), 2; got != want {
		t.Fatalf("got %d issues, want %d", got, want)
	}
}

func TestValidateSourcesRejectsLinkname(t *testing.T) {
	dir := t.TempDir()
	linkFile := filepath.Join(dir, "linkname.go")
	if err := os.WriteFile(linkFile, []byte("package main\n\n//go:linkname putsymbol runtime.putsymbol\nfunc f() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := ValidateSources([]string{linkFile})
	if err != nil {
		t.Fatalf("ValidateSources error: %v", err)
	}
	if got, want := len(issues), 1; got != want {
		t.Fatalf("got %d issues, want %d", got, want)
	}
}

func TestValidateSourcesRejectsAssembly(t *testing.T) {
	dir := t.TempDir()
	asm := filepath.Join(dir, "x.s")
	if err := os.WriteFile(asm, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := ValidateSources([]string{dir})
	if err != nil {
		t.Fatalf("ValidateSources error: %v", err)
	}
	if got, want := len(issues), 1; got != want {
		t.Fatalf("got %d issues, want %d", got, want)
	}
	if issues[0].Kind != IssueAssembly {
		t.Fatalf("got kind %s, want %s", issues[0].Kind, IssueAssembly)
	}
}

func TestValidateBuildMode(t *testing.T) {
	if err := ValidateBuildMode(""); err != nil {
		t.Fatalf("unexpected error for empty mode: %v", err)
	}
	if err := ValidateBuildMode("exe"); err != nil {
		t.Fatalf("unexpected error for exe: %v", err)
	}
	if err := ValidateBuildMode("c-shared"); err == nil {
		t.Fatalf("expected error for unsupported buildmode")
	}
}
