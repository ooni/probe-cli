// Package periodic contains code to manage periodic runs
package periodic

import "sync"

const (
	// StatusScheduled indicates that OONI is scheduled to run
	// periodically in the background.
	StatusScheduled = "scheduled"

	// StatusStopped indicates that OONI is not scheduled to
	// run periodically in the background.
	StatusStopped = "stopped"

	// StatusRunning indicates that OONI is currently
	// running in the background.
	StatusRunning = "running"
)

// Manager manages periodic runs
type Manager interface {
	LogShow() error
	LogStream() error
	Start() error
	Status() (string, error)
	Stop() error
}

var (
	registry map[string]Manager
	mtx      sync.Mutex
)

func register(platform string, manager Manager) {
	defer mtx.Unlock()
	mtx.Lock()
	if registry == nil {
		registry = make(map[string]Manager)
	}
	registry[platform] = manager
}

// Get gets the specified periodic manager. This function
// returns nil if no periodic manager exists.
func Get(platform string) Manager {
	defer mtx.Unlock()
	mtx.Lock()
	return registry[platform]
}
