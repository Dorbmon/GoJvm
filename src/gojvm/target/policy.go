package target

import "fmt"

const (
	TargetGOOS        = "jvm"
	TargetGOARCH      = "jvm"
	MinClassFileMajor = 65
	MinJavaMajor      = 21
)

const (
	BuildModeExe = "exe"
)

const (
	MsgTargetUnsupported    = "gojvm: unsupported target %s/%s"
	MsgBuildModeUnsupported = "gojvm: unsupported build mode %q for GOOS=%s GOARCH=%s; use -buildmode=" + BuildModeExe
	MsgCGOUnsupported       = `go: GOOS=jvm does not support cgo; use CGO_ENABLED=0 and remove imports of "C"`
	MsgUnsafeUnsupported    = `go: import of "unsafe" is not supported when GOOS=GOARCH=jvm`
	MsgAssemblyUnsupported  = "go: GOOS=jvm does not support Go assembly (.s/.S) files"
	MsgLinknameUnsupported  = "go: //go:linkname is disabled for GOOS=jvm"
	MsgJavaNotAvailable     = "gojvm: JAVA_HOME is not set and is required for JVM target checks"
)

var supportedBuildModes = map[string]struct{}{
	BuildModeExe: {},
}

// IsJVM returns true when the given target is JVM.
func IsJVM(goos, goarch string) bool {
	return goos == TargetGOOS && goarch == TargetGOARCH
}

// ValidateTarget validates GOOS/GOARCH for the JVM backend.
func ValidateTarget(goos, goarch string) error {
	if !IsJVM(goos, goarch) {
		return fmt.Errorf(MsgTargetUnsupported, goos, goarch)
	}
	return nil
}

// ValidateBuildMode validates permitted Go build modes for JVM.
func ValidateBuildMode(mode string) error {
	if mode == "" {
		return nil
	}
	if _, ok := supportedBuildModes[mode]; !ok {
		return fmt.Errorf(MsgBuildModeUnsupported, mode, TargetGOOS, TargetGOARCH)
	}
	return nil
}

// IsBuildModeSupported reports if a mode is accepted by JVM tooling.
func IsBuildModeSupported(mode string) bool {
	if mode == "" {
		return true
	}
	_, ok := supportedBuildModes[mode]
	return ok
}
