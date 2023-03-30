package internal

// Run is used in tests.
func Run(callback func() error) error {
	return Run2(callback)
}

// Run2 is used in tests.
func Run2(callback func() error) error {
	return callback()
}
