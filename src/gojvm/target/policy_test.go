package target

import "testing"

func TestValidateTarget(t *testing.T) {
	if err := ValidateTarget("jvm", "jvm"); err != nil {
		t.Fatalf("expected supported target, got %v", err)
	}
	if err := ValidateTarget("linux", "amd64"); err == nil {
		t.Fatalf("expected unsupported target error")
	}
}

func TestValidateBuildMode(t *testing.T) {
	if err := ValidateBuildMode(""); err != nil {
		t.Fatalf("unexpected empty mode error: %v", err)
	}
	if err := ValidateBuildMode(BuildModeExe); err != nil {
		t.Fatalf("unexpected exe mode error: %v", err)
	}
	if err := ValidateBuildMode("c-shared"); err == nil {
		t.Fatalf("expected non-exe build mode error")
	}
}
