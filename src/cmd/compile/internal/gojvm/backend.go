package gojvm

// Package gojvm is the JVM backend branch point for the compiler frontend.
//
// This package intentionally contains only the milestone-zero scaffolding today.
// Implementations for MIR construction, lowering, emission, and diagnostics are
// intentionally introduced in later milestones.

// Config is the minimum metadata passed from the front-end driver.
type Config struct {
	// Target triples as accepted by the Go build target variables.
	GOOS   string
	GOARCH string
	// RuntimeABIVersion identifies this MILESTONE protocol version.
	RuntimeABIVersion int
}

// CompilePackage is the JVM backend entrypoint once front-end integration reaches
// the insertion point defined by ADR-0002.
func CompilePackage(cfg Config) error {
	return ErrNotImplemented{Feature: "frontend->gojvm integration"}
}
