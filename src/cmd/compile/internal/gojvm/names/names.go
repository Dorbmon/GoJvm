package names

import "fmt"

// Mangle converts a Go identifier into a JVM-safe class/member name placeholder.
func Mangle(ident string) string {
	return fmt.Sprintf("g_%x", []byte(ident))
}
