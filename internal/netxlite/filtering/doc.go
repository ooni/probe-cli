// Package filtering allows to implement self-censorship.
//
// The top-level struct is the TProxy. It implements netxlite's
// TProxable interface. Therefore, you can use TProxy to
// implement filtering and blocking of TCP, TLS, QUIC, DNS, HTTP.
//
// We also expose proxies that implement filtering policies for
// DNS, TLS, and HTTP.
//
// The typical usage of this package's functionality is to
// load a censoring policy into TProxyConfig and then to create
// and start a TProxy instance using NewTProxy.
package filtering
