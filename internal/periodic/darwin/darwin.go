package darwin

// Manager allows to start/stop running periodically.
type Manager struct{}

// Start starts running periodically.
func (Manager) Start() error {
	return nil
}

// Stop stops running periodically.
func (Manager) Stop() error {
	return nil
}
