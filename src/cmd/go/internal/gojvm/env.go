package gojvm

import "fmt"

// EnvConfig holds compiler-invocation variables of interest.
type EnvConfig struct {
	GOOS   string
	GOARCH string
}

func ValidateTarget(cfg EnvConfig) error {
	if cfg.GOOS != "jvm" || cfg.GOARCH != "jvm" {
		return fmt.Errorf("gojvm: unsupported target %s/%s", cfg.GOOS, cfg.GOARCH)
	}
	return nil
}
