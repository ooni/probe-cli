package engine

// KVStore is a simple, atomic key-value store. The user of
// probe-engine should supply an implementation of this interface,
// which will be used by probe-engine to store specific data.
type KVStore interface {
	Get(key string) (value []byte, err error)
	Set(key string, value []byte) (err error)
}
