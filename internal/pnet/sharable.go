package pnet

// Sharable is a type trait indicating a value can be shared
// by multiple parallel goroutines without data races.
type Sharable interface {
	Sharable()
}
