// Package torcontrol contains networking code to control tor using
// the control protocol defined by [tor-control-spec].
//
// To use this package, you need to upgrade a [net.Conn] connected to the
// control port to a [*Conn] using the [NewConn] factory.
//
// Creating a [*Conn] causes two goroutines to be created: the read loop
// and the write loop. The read loop reads messages from the control
// channel, while the write loop writes messages to the control channel.
//
// You can perform three operations on a [*Conn]:
//
// 1. the Close method allows you to close a control connection and shutdown
// the two background goroutines associated with it;
//
// 2. create and Attach an [*EventReader] that will receive asynchronous
// eventsuntil you Detach the [*EventReader];
//
// 3. the SendRecv method allows you to send a specific synchronous
// request and receive the corresponding response.
//
// This package is rather low-level. Its main concern is to provide an
// interface for reading and writing messages. We recommend using the [torcontrol]
// package along with a [*Conn] for a higer-level experience.
//
// This package is a fork of [github.com/cretz/bine] and shares the
// same MIT license of the original package.
//
// [github.com/cretz/bine]: https://github.com/cretz/bine
// [tor-control-spec]: https://spec.torproject.org/control-spec/
package torcontrol
