package model

// KeyValueStore is a key-value store used by the session.
type KeyValueStore interface {
	Get(key string) (value []byte, err error)
	Set(key string, value []byte) (err error)
}
