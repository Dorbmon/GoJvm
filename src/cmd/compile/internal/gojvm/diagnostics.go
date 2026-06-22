package gojvm

import "fmt"

// Diagnostic is a user-facing error for a disabled feature or unsupported target.
type Diagnostic struct {
	Kind    string
	Message string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s", d.Kind, d.Message)
}

const (
	ErrCGO    = "unsupported: cgo"
	ErrUnsafe = "unsupported: unsafe"
	ErrAsm    = "unsupported: assembly"
)
