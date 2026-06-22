package mir

// Serialize is a stub serializer used by linker/link-stage scaffolding.
func Serialize(_ *Module) ([]byte, error) {
	return []byte("{}"), nil
}
