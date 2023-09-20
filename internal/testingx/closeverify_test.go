package testingx_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestCloseVerifyWAI(t *testing.T) {
	t.Run("when it contains no connections", func(t *testing.T) {
		cv := &testingx.CloseVerify{}
		if err := cv.CheckForOpenConns(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("when we have closed all connections", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		func() {
			wg := &sync.WaitGroup{}

			unet := cv.WrapUnderlyingNetwork(&netxlite.DefaultTProxy{})

			listener := runtimex.Try1(unet.ListenTCP("tcp", &net.TCPAddr{}))
			defer listener.Close()
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn := runtimex.Try1(listener.Accept())
				defer conn.Close()
			}()

			pconn := runtimex.Try1(unet.ListenUDP("udp", &net.UDPAddr{}))
			defer pconn.Close()

			ctx := context.Background()
			conn := runtimex.Try1(unet.DialContext(ctx, "tcp", listener.Addr().String()))
			defer conn.Close()

			wg.Wait()
		}()

		if err := cv.CheckForOpenConns(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("when we've not closed some connections", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		func() {
			wg := &sync.WaitGroup{}

			var (
				udpPort = &atomic.Int64{}
				tcpPort = &atomic.Int64{}
			)

			unet := cv.WrapUnderlyingNetwork(&mocks.UnderlyingNetwork{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					conn := &mocks.Conn{
						MockLocalAddr: func() net.Addr {
							return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(tcpPort.Add(1))}
						},
					}
					return conn, nil
				},
				MockListenTCP: func(network string, addr *net.TCPAddr) (net.Listener, error) {
					listener := &mocks.Listener{
						MockAccept: func() (net.Conn, error) {
							conn := &mocks.Conn{
								MockLocalAddr: func() net.Addr {
									return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(tcpPort.Add(1))}
								},
							}
							return conn, nil
						},
						MockAddr: func() net.Addr {
							return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(tcpPort.Add(1))}
						},
					}
					return listener, nil
				},
				MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
					pconn := &mocks.UDPLikeConn{
						MockLocalAddr: func() net.Addr {
							return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(udpPort.Add(1))}
						},
					}
					return pconn, nil
				},
			})

			listener := runtimex.Try1(unet.ListenTCP("tcp", &net.TCPAddr{}))
			//defer listener.Close() // <- not closing the listener!
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn := runtimex.Try1(listener.Accept())
				//defer conn.Close() // <- not closing the conn!
				_ = conn
			}()

			pconn := runtimex.Try1(unet.ListenUDP("udp", &net.UDPAddr{}))
			//defer pconn.Close() <- not closing the pconn!
			_ = pconn

			ctx := context.Background()
			conn := runtimex.Try1(unet.DialContext(ctx, "tcp", listener.Addr().String()))
			//defer conn.Close() // <- not closing the conn!
			_ = conn

			wg.Wait()
		}()

		if err := cv.CheckForOpenConns(); err == nil {
			t.Fatal("expected an error here")
		}
	})

	t.Run("on DialContext error", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		expected := errors.New("mocked error")

		unet := cv.WrapUnderlyingNetwork(&mocks.UnderlyingNetwork{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, expected
			},
		})

		conn, err := unet.DialContext(context.Background(), "tcp", "127.0.0.1:443")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("on ListenTCP error", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		expected := errors.New("mocked error")

		unet := cv.WrapUnderlyingNetwork(&mocks.UnderlyingNetwork{
			MockListenTCP: func(network string, addr *net.TCPAddr) (net.Listener, error) {
				return nil, expected
			},
		})

		listener, err := unet.ListenTCP("tcp", &net.TCPAddr{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if listener != nil {
			t.Fatal("expected nil listener")
		}
	})

	t.Run("on ListenUDP error", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		expected := errors.New("mocked error")

		unet := cv.WrapUnderlyingNetwork(&mocks.UnderlyingNetwork{
			MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expected
			},
		})

		pconn, err := unet.ListenUDP("tcp", &net.UDPAddr{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if pconn != nil {
			t.Fatal("expected nil pconn")
		}
	})

	t.Run("on Accept error", func(t *testing.T) {
		cv := &testingx.CloseVerify{}

		expected := errors.New("mocked error")

		tcpPort := &atomic.Int64{}
		unet := cv.WrapUnderlyingNetwork(&mocks.UnderlyingNetwork{
			MockListenTCP: func(network string, addr *net.TCPAddr) (net.Listener, error) {
				listener := &mocks.Listener{
					MockAccept: func() (net.Conn, error) {
						return nil, expected
					},
					MockAddr: func() net.Addr {
						return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(tcpPort.Add(1))}
					},
				}
				return listener, nil
			},
		})

		listener, err := unet.ListenTCP("tcp", &net.TCPAddr{})
		if err != nil {
			t.Fatal(err)
		}

		conn, err := listener.Accept()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
