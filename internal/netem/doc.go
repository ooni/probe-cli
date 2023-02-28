// Package netem contains code to emulate networking.
//
// This package exists to facilitate running integration tests where
// we run OONI Probe code with a completely emulated Internet.
//
// # Implementation overview
//
// A [GvisorStack] is a TCP/IP stack in userspace. This type implements
// [model.UnderlyingNetwork], which means that we can use it with [netxlite]
// by calling [netxlite.WithCustomTProxy].
//
// Once you have a [GvisorStack], you need to create two [NIC]s. You need to
// attach the first [NIC] to the stack using [GvisorStack.Attach]. Then,
// you need to connect the first and the second [NICs] using a [Link].
//
// A [Link] allows you to connect two [NICs]. For simple use cases, you can
// directly use a [Link] to connect a client [GvisorStack] and a server
// [GvisorStack]. You use [Link.Up] to start actively forwarding traffic
// between the two [NIC]s of a given [Link]. Each [Link] can possibly add
// extra latency to degrade the performance and simulate throttling. A [Link]
// is also the place where you can apply censorship policies using DPI.
//
// [Link] constructors such as [NewLinkFastest] accept a [LinkDPIEngine]
// argument. If you pass [DPINone] as that argument there will be no
// censorship using DPI. However, by passing a [DPIDropTrafficForServerEndpoint]
// instance you can, e.g., drop traffic directed towards an endpoint. Other
// DPI-based censorship policies are also available.
package netem
