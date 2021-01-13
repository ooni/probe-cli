// Package periodic contains code to manage periodic runs
package periodic

import "sync"

// Manager manages periodic runs
type Manager interface {
	Start() error
	Stop() error
}

var (
	registry map[string]Manager
	mtx      sync.Mutex
)

func register(name string, manager Manager) {
	defer mtx.Unlock()
	mtx.Lock()
	if registry == nil {
		registry = make(map[string]Manager)
	}
	registry[name] = manager
}

// Get gets the specified periodic manager. This function
// returns nil if no periodic manager exists.
func Get(name string) Manager {
	defer mtx.Unlock()
	mtx.Lock()
	return registry[name]
}
