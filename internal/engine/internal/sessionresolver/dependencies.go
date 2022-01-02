package sessionresolver

// KVStore is a generic key-value store. We use it to store
// on disk persistent state used by this package.
type KVStore interface {
	// Get gets the value for the given key.
	Get(key string) ([]byte, error)

	// Set sets the value of the given key.
	Set(key string, value []byte) error
}
