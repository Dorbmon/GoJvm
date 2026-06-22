package gojvm

import "fmt"

// ErrNotImplemented communicates explicit unsupported-path behavior for milestone scaffolding.
type ErrNotImplemented struct {
	Feature string
}

func (e ErrNotImplemented) Error() string {
	return fmt.Sprintf("gojvm: not implemented: %s", e.Feature)
}
