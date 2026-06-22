package abi

// CallingConvention identifies backend target call signatures.
type CallingConvention string

const (
	// ConventionJVM is the initial target marker for JVM emission.
	ConventionJVM CallingConvention = "jvm"
)
