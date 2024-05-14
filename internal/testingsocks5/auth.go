package testingsocks5

import (
	"fmt"
	"io"
	"net"
)

// Codes representing authentication mechanisms
const (
	noAuth          = uint8(0)
	noAcceptable    = uint8(255)
	userPassAuth    = uint8(2)
	userAuthVersion = uint8(1)
	authSuccess     = uint8(0)
	authFailure     = uint8(1)
)

var (
	errNoSupportedAuth = fmt.Errorf("no supported authentication mechanism")
)

// A Request encapsulates authentication state provided
// during negotiation
type authContext struct {
	// Provided auth method
	Method uint8

	// Payload provided during negotiation.
	// Keys depend on the used auth method.
	// For UserPassauth contains Username
	Payload map[string]string
}

// noAuthAuthenticator is used to handle the "No Authentication" mode
type noAuthAuthenticator struct{}

func (a noAuthAuthenticator) Authenticate(cconn net.Conn) (*authContext, error) {
	_, err := cconn.Write([]byte{socks5Version, noAuth})
	return &authContext{noAuth, nil}, err
}

// authenticate is used to handle connection authentication
func (s *Server) authenticate(cconn net.Conn) (*authContext, error) {
	// Get the methods
	methods, err := readAuthMethods(cconn)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth methods: %w", err)
	}

	// Select a usable method
	for _, method := range methods {
		switch method {
		case noAuth:
			return (noAuthAuthenticator{}).Authenticate(cconn)

		default:
			// nothing
		}
	}

	// No usable method found
	return nil, noAcceptableAuth(cconn)
}

// noAcceptableAuth is used to handle when we have no eligible authentication mechanism
func noAcceptableAuth(conn net.Conn) error {
	_, _ = conn.Write([]byte{socks5Version, noAcceptable})
	return errNoSupportedAuth
}

// readAuthMethods is used to read the number of methods and proceeding auth methods
func readAuthMethods(cconn net.Conn) ([]byte, error) {
	header := []byte{0}
	if _, err := io.ReadFull(cconn, header); err != nil {
		return nil, err
	}

	numMethods := uint8(header[0])
	methods := make([]byte, numMethods)
	_, err := io.ReadFull(cconn, methods)
	return methods, err
}
