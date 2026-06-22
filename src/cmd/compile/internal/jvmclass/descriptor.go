package jvmclass

// Descriptor is a placeholder descriptor serializer.
func Descriptor(name string) string {
	return "L" + name + ";"
}
