package model

//
// Definition of a key-value store.
//

// KeyValueStore is a generic key-value store.
type KeyValueStore interface {
	// Get gets the value of the given key or returns an
	// error if there is no such key or we cannot read
	// from the key-value store.
	Get(key string) (value []byte, err error)

	// Set sets the value of the given key and returns
	// whether the operation was successful or not.
	Set(key string, value []byte) (err error)
}
