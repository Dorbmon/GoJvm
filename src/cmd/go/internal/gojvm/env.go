package gojvm

import (
	"gojvm-backend/src/gojvm/target"
)

// EnvConfig holds compiler-invocation variables of interest.
type EnvConfig struct {
	GOOS   string
	GOARCH string
}

func ValidateTarget(cfg EnvConfig) error {
	return target.ValidateTarget(cfg.GOOS, cfg.GOARCH)
}

// ValidateBuildMode rejects non-exe build modes for the JVM target.
func ValidateBuildMode(mode string) error {
	return target.ValidateBuildMode(mode)
}

// Constants used by command glue.
const SupportedBuildMode = target.BuildModeExe

func IsGoJVMTarget(goos, goarch string) bool {
	return target.IsJVM(goos, goarch)
}

func IsBuildModeSupported(mode string) bool {
	return target.IsBuildModeSupported(mode)
}
