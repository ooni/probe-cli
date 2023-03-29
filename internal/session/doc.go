// Package session implements a measurement session. The design of
// this package is such that we can split the measurement engine proper
// and the application using it. In particular, this design is such
// that it would be easy to expose this API as a C library.
package session
