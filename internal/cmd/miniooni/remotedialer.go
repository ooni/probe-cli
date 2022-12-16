package main

//
// Common code for dialing TCP connections
//

import "net"

// remoteDialer implements remoteClientDialer.
type remoteDialer struct {
	// remoteAddr is the remote address to use.
	remoteAddr string

	// wrapConn wraps the established conn.
	wrapConn remoteConnWrapper
}

var _ remoteClientDialer = &remoteDialer{}

// Dial implements remoteClientDialer.
func (rcd *remoteDialer) Dial() (remoteConn, error) {
	conn, err := net.Dial("tcp", rcd.remoteAddr)
	if err != nil {
		return nil, err
	}
	cw, err := rcd.wrapConn(conn)
	if err != nil {
		return nil, err
	}
	return cw, nil
}
