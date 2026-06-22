package gojvm

// RuntimeTarget captures target-level constants for a GoJVM build.
type RuntimeTarget struct {
	MinClassVersion int
	GoVersion       string
	UseVirtualCPU   bool
}

// DefaultRuntimeTarget returns the baseline target requested by the spec.
func DefaultRuntimeTarget() RuntimeTarget {
	return RuntimeTarget{
		MinClassVersion: 65,
		GoVersion:       "1.26.4",
		UseVirtualCPU:   true,
	}
}
