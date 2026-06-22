package jvmclass

import (
	"bytes"
)

// Write emits a placeholder class file payload.
func Write(_ *Class) ([]byte, error) {
	return bytes.Clone([]byte{0xca, 0xfe, 0xba, 0xbe, 0, 65}), nil
}
