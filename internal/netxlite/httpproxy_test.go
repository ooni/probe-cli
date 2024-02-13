package netxlite

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
)

func TestHTTPProxyDialer(t *testing.T) {
	// REMINDER: This test need a http proxy running locally
	dialer := NewHTTPDialer("tcp", "localhost:7890")
	t.Run("DialContextSuccess", func(t *testing.T) {
		conn, err := dialer.DialContext(context.Background(), "tcp", "google.com:443")
		if err != nil {
			t.Fatal(fmt.Sprintf("unexpected error: %v", err))
		}
		if conn == nil {
			t.Fatal("unexpected nil connection")
		}
	})

	t.Run("DialSuccess", func(t *testing.T) {
		conn, err := dialer.Dial("tcp", "google.com:443")
		if err != nil {
			t.Fatal(fmt.Sprintf("unexpected error: %v", err))
		}
		if conn == nil {
			t.Fatal("unexpected nil connection")
		}
	})

	t.Run("DialContextInvalidNetwork", func(t *testing.T) {
		expected := errors.New("network not implemented")
		conn, err := dialer.DialContext(context.Background(), "udp", "google.com:443")
		if conn != nil {
			t.Fatal("unexpected connection")
		}
		if err.(*net.OpError).Err.Error() != expected.Error() {
			t.Fatal(fmt.Sprintf("unexpected error: %v", err))
		}
	})

	t.Run("DialInvalidNetwork", func(t *testing.T) {
		expected := errors.New("network not implemented")
		conn, err := dialer.Dial("udp", "google.com:443")
		if conn != nil {
			t.Fatal("unexpected connection")
		}
		if err.(*net.OpError).Err.Error() != expected.Error() {
			t.Fatal(fmt.Sprintf("unexpected error: %v", err))
		}
	})
}

func TestHTTPProxyDialerFailure(t *testing.T) {
	dialer := NewHTTPDialer("tcp", "localhost:8888")
	go func() {
		listener, err := net.Listen("tcp", "localhost:8888")
		defer listener.Close()
		if err != nil {
			t.Error(fmt.Sprintf("error: listen failed%v", err))
			return
		}
		err = http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				w.WriteHeader(500)
				return
			} else {
				t.Error("error: unexpected request")
				return
			}
		}))
	}()
	t.Run("DialContextFailure", func(t *testing.T) {
		expected := errors.New("cannot establish connection")

		conn, err := dialer.DialContext(context.Background(), "tcp", "google.com:443")
		if conn != nil {
			t.Fatal("unexpected connection")
		}
		if err.(*net.OpError).Err.Error() != expected.Error() {
			t.Fatal("unexpected error")
		}
	})

	t.Run("DialFailure", func(t *testing.T) {
		expected := errors.New("cannot establish connection")

		conn, err := dialer.Dial("tcp", "google.com:443")
		if conn != nil {
			t.Fatal("unexpected connection")
		}
		if err.(*net.OpError).Err.Error() != expected.Error() {
			t.Fatal(fmt.Sprintf("unexpected error: %v", err))
		}
	})

	return
}
