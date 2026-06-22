package gojvm

import (
	"gojvm-backend/src/cmd/compile/internal/gojvm/checks"
	"gojvm-backend/src/gojvm/target"
)

// Config is the minimum metadata passed from the front-end driver.
type Config struct {
	// Target triples as accepted by the Go build target variables.
	GOOS   string
	GOARCH string
	// BuildMode is validated before backend-specific compilation begins.
	BuildMode string
	// SourceFiles are the files that should be pre-validated for unsupported features.
	SourceFiles []string
	// RuntimeABIVersion identifies this MILESTONE protocol version.
	RuntimeABIVersion int
}

// CompilePackage is the JVM backend entrypoint once front-end integration reaches
// the insertion point defined by ADR-0002.
func CompilePackage(cfg Config) error {
	if err := target.ValidateTarget(cfg.GOOS, cfg.GOARCH); err != nil {
		return err
	}
	issues, err := checks.ValidateSources(cfg.SourceFiles)
	if err != nil {
		return err
	}
	if err := checks.ValidateBuildMode(cfg.BuildMode); err != nil {
		return err
	}
	if len(issues) > 0 {
		return issues[0]
	}
	return ErrNotImplemented{Feature: "frontend->gojvm integration"}
}
