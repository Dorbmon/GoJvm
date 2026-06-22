package mir

// String returns a readable MIR representation.
func String(m *Module) string {
	if m == nil {
		return "<nil>"
	}
	return "mir-module:" + m.Name
}
