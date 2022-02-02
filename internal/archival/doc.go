// Package implements a Saver type that saves network, TCP, DNS,
// and TLS events. Given a Saver, you can export a Trace. Given a
// Trace you can obtain data in the OONI archival data format.
//
// Given a Saver and an interface implementing the netx model (e.g.,
// a model.Resolver, a model.QUICDialer), you can always create a
// wrapper of such an interface that saves using the saver. For example,
// if d is a model.Dialer and s is a *Saver, you can use s.WrapDialer
// to get a new model.Dialer using s for saving events.
package archival
