package opt

// Pass is a name for a MIR optimization pass.
type Pass struct {
	Name string
}

// Run executes a scaffolded optimization pass.
func Run(_ interface{}, _ ...Pass) error {
	return nil
}
