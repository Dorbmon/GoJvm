package gojvm

import "fmt"

// EnvConfig holds compiler-invocation variables of interest.
type EnvConfig struct {
	GOOS   string
	GOARCH string
}

const (
	// JVM supports only executable output in the first milestone.
	SupportedBuildMode = "exe"
)

func ValidateTarget(cfg EnvConfig) error {
	if cfg.GOOS != "jvm" || cfg.GOARCH != "jvm" {
		return fmt.Errorf("gojvm: unsupported target %s/%s", cfg.GOOS, cfg.GOARCH)
	}
	return nil
}

// ValidateBuildMode rejects non-exe build modes for the JVM target.
func ValidateBuildMode(mode string) error {
	if mode == "" || mode == SupportedBuildMode {
		return nil
	}
	return fmt.Errorf("gojvm: unsupported build mode %q for GOOS=GOARCH=jvm; use -buildmode=exe", mode)
}
