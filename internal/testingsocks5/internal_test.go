package testingsocks5

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestReadVersionError(t *testing.T) {
	server := &Server{
		closeOnce: sync.Once{},
		listener: &mocks.Listener{
			MockClose: func() error {
				return nil
			},
		},
		logger: log.Log,
		netx:   &netxlite.Netx{Underlying: nil},
	}
	defer server.Close()

	conn := &mocks.Conn{
		MockClose: func() error {
			return nil
		},
		MockRead: func(b []byte) (int, error) {
			return 0, io.EOF
		},
	}

	err := server.serveConn(conn)
	if !errors.Is(err, io.EOF) {
		t.Fatal("unexpected error", err)
	}
}

func TestServerClosesConn(t *testing.T) {
	server := &Server{
		closeOnce: sync.Once{},
		listener: &mocks.Listener{
			MockClose: func() error {
				return nil
			},
		},
		logger: log.Log,
		netx:   &netxlite.Netx{Underlying: nil},
	}
	defer server.Close()

	called := false
	conn := &mocks.Conn{
		MockClose: func() error {
			called = true
			return nil
		},
		MockRead: func(b []byte) (int, error) {
			return 0, io.EOF
		},
	}

	err := server.serveConn(conn)
	if !errors.Is(err, io.EOF) {
		t.Fatal("unexpected error", err)
	}
	if !called {
		t.Fatal("did not call close")
	}
}

func TestInvalidVersion(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: the protocol version must be 5
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{17, 0, 0, 1}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				17, // version
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAuthMethodsFailure(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: the protocol expects something after we have sent the version
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestNoAcceptableAuth(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we don't support username and password authentication
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5,             // version
				1,             // number of authentication methods supported
				2,             // username and password
				1,             // version of the username and password authentication
				3,             // username length
				'f', 'o', 'o', // username
				'3',           // password length
				'b', 'a', 'r', // password
			},
			expect: []byte{5, 255},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestNewRequestReadError(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: the second message should contain something after the version
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5, // version
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestNewRequestWithIncompatibleVersion(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: the second message should contain again version equal to 5
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				17, // version
				2,  // bind command
				0,  // reserved
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedCommand(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5,            // version
				2,            // bind command
				0,            // reserved
				1,            // IPv4
				127, 0, 0, 1, // address
				0, 80, // port
			},
			expect: []byte{5, 7, 0, 1, 0, 0, 0, 0, 0, 0},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestUnrecognizedAddrType(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5,            // version
				2,            // bind command
				0,            // reserved
				55,           // ???
				127, 0, 0, 1, // address
				0, 80, // port
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingAddrType(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5, // version
				2, // bind command
				0, // reserved
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingIPv4Address(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5, // version
				2, // bind command
				0, // reserved
				1, // IPv4
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingIPv6Address(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5, // version
				2, // bind command
				0, // reserved
				4, // IPv6
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingFQDNLength(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5, // version
				2, // bind command
				0, // reserved
				3, // FQDN
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingFQDNString(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5,  // version
				2,  // bind command
				0,  // reserved
				3,  // FQDN
				10, // length of FQDN
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}

func TestReadAddrSpecFailureReadingPortWithIPv6(t *testing.T) {
	server := MustNewServer(
		log.Log,
		&netxlite.Netx{Underlying: nil},
		&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		},
	)
	defer server.Close()

	// Note: we only support the connect command
	conn := runtimex.Try1(net.Dial("tcp", server.Endpoint()))
	_ = runtimex.Try1(conn.Write([]byte{}))
	defer conn.Close()

	client := &client{
		exchanges: []exchange{{
			send: []byte{
				5, // version
				1, // number of authentication methods supported
				0, // no authentication
			},
			expect: []byte{5, 0},
		}, {
			send: []byte{
				5,          // version
				2,          // bind command
				0,          // reserved
				4,          // IPv6,
				0, 0, 0, 0, // IPv6 addr
				0, 0, 0, 0, // IPv6 addr
				0, 0, 0, 0, // IPv6 addr
				0, 0, 0, 0, // IPv6 addr
			},
			expect: []byte{},
		}},
	}
	if err := client.run(log.Log, conn); err != nil {
		t.Fatal(err)
	}
}
