package httpapi

//
// API specification
//

// SimpleSpec describes an API that returns a response only consisting of bytes.
//
// The corresponding API-calling function is [SimpleCall].
type SimpleSpec interface {
	// Descriptor returns the descriptor to use.
	Descriptor() *Descriptor
}

// TypedSpec[T] describes an API that returns *T as the response.
//
// the corresponding API-calling function is [TypedCall].
type TypedSpec[T any] interface {
	// Descriptor returns the descriptor to use.
	Descriptor() (*Descriptor, error)

	// ZeroResponse returns T's zero value.
	ZeroResponse() T
}
