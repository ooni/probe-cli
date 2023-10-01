package dslmodel

// Sharable is a type trait indicating you can share some data
// across parallel goroutines safely because the data is:
//
// 1. immutable; and
//
// 2. does not contain resources (e.g., network connections) that
// it only makes sense to use in a single goroutine.
type Sharable interface {
	Sharable()
}
