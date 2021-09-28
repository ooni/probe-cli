// Code generated by go generate; DO NOT EDIT.
// Generated: 2021-09-28 18:13:54.431042 +0200 CEST m=+0.522721376

package netxlite

import (
	"io"
	"syscall"
	"testing"
)

func TestClassifySyscallError(t *testing.T) {
	t.Run("for a non-syscall error", func(t *testing.T) {
		if v := classifySyscallError(io.EOF); v != "" {
			t.Fatalf("expected empty string, got '%s'", v)
		}
	})

	t.Run("for ECONNREFUSED", func(t *testing.T) {
		if v := classifySyscallError(ECONNREFUSED); v != FailureConnectionRefused {
			t.Fatalf("expected '%s', got '%s'", FailureConnectionRefused, v)
		}
	})

	t.Run("for ECONNRESET", func(t *testing.T) {
		if v := classifySyscallError(ECONNRESET); v != FailureConnectionReset {
			t.Fatalf("expected '%s', got '%s'", FailureConnectionReset, v)
		}
	})

	t.Run("for EHOSTUNREACH", func(t *testing.T) {
		if v := classifySyscallError(EHOSTUNREACH); v != FailureHostUnreachable {
			t.Fatalf("expected '%s', got '%s'", FailureHostUnreachable, v)
		}
	})

	t.Run("for ETIMEDOUT", func(t *testing.T) {
		if v := classifySyscallError(ETIMEDOUT); v != FailureTimedOut {
			t.Fatalf("expected '%s', got '%s'", FailureTimedOut, v)
		}
	})

	t.Run("for EAFNOSUPPORT", func(t *testing.T) {
		if v := classifySyscallError(EAFNOSUPPORT); v != FailureAddressFamilyNotSupported {
			t.Fatalf("expected '%s', got '%s'", FailureAddressFamilyNotSupported, v)
		}
	})

	t.Run("for EADDRINUSE", func(t *testing.T) {
		if v := classifySyscallError(EADDRINUSE); v != FailureAddressInUse {
			t.Fatalf("expected '%s', got '%s'", FailureAddressInUse, v)
		}
	})

	t.Run("for EADDRNOTAVAIL", func(t *testing.T) {
		if v := classifySyscallError(EADDRNOTAVAIL); v != FailureAddressNotAvailable {
			t.Fatalf("expected '%s', got '%s'", FailureAddressNotAvailable, v)
		}
	})

	t.Run("for EISCONN", func(t *testing.T) {
		if v := classifySyscallError(EISCONN); v != FailureAlreadyConnected {
			t.Fatalf("expected '%s', got '%s'", FailureAlreadyConnected, v)
		}
	})

	t.Run("for EFAULT", func(t *testing.T) {
		if v := classifySyscallError(EFAULT); v != FailureBadAddress {
			t.Fatalf("expected '%s', got '%s'", FailureBadAddress, v)
		}
	})

	t.Run("for EBADF", func(t *testing.T) {
		if v := classifySyscallError(EBADF); v != FailureBadFileDescriptor {
			t.Fatalf("expected '%s', got '%s'", FailureBadFileDescriptor, v)
		}
	})

	t.Run("for ECONNABORTED", func(t *testing.T) {
		if v := classifySyscallError(ECONNABORTED); v != FailureConnectionAborted {
			t.Fatalf("expected '%s', got '%s'", FailureConnectionAborted, v)
		}
	})

	t.Run("for EALREADY", func(t *testing.T) {
		if v := classifySyscallError(EALREADY); v != FailureConnectionAlreadyInProgress {
			t.Fatalf("expected '%s', got '%s'", FailureConnectionAlreadyInProgress, v)
		}
	})

	t.Run("for EDESTADDRREQ", func(t *testing.T) {
		if v := classifySyscallError(EDESTADDRREQ); v != FailureDestinationAddressRequired {
			t.Fatalf("expected '%s', got '%s'", FailureDestinationAddressRequired, v)
		}
	})

	t.Run("for EINTR", func(t *testing.T) {
		if v := classifySyscallError(EINTR); v != FailureInterrupted {
			t.Fatalf("expected '%s', got '%s'", FailureInterrupted, v)
		}
	})

	t.Run("for EINVAL", func(t *testing.T) {
		if v := classifySyscallError(EINVAL); v != FailureInvalidArgument {
			t.Fatalf("expected '%s', got '%s'", FailureInvalidArgument, v)
		}
	})

	t.Run("for EMSGSIZE", func(t *testing.T) {
		if v := classifySyscallError(EMSGSIZE); v != FailureMessageSize {
			t.Fatalf("expected '%s', got '%s'", FailureMessageSize, v)
		}
	})

	t.Run("for ENETDOWN", func(t *testing.T) {
		if v := classifySyscallError(ENETDOWN); v != FailureNetworkDown {
			t.Fatalf("expected '%s', got '%s'", FailureNetworkDown, v)
		}
	})

	t.Run("for ENETRESET", func(t *testing.T) {
		if v := classifySyscallError(ENETRESET); v != FailureNetworkReset {
			t.Fatalf("expected '%s', got '%s'", FailureNetworkReset, v)
		}
	})

	t.Run("for ENETUNREACH", func(t *testing.T) {
		if v := classifySyscallError(ENETUNREACH); v != FailureNetworkUnreachable {
			t.Fatalf("expected '%s', got '%s'", FailureNetworkUnreachable, v)
		}
	})

	t.Run("for ENOBUFS", func(t *testing.T) {
		if v := classifySyscallError(ENOBUFS); v != FailureNoBufferSpace {
			t.Fatalf("expected '%s', got '%s'", FailureNoBufferSpace, v)
		}
	})

	t.Run("for ENOPROTOOPT", func(t *testing.T) {
		if v := classifySyscallError(ENOPROTOOPT); v != FailureNoProtocolOption {
			t.Fatalf("expected '%s', got '%s'", FailureNoProtocolOption, v)
		}
	})

	t.Run("for ENOTSOCK", func(t *testing.T) {
		if v := classifySyscallError(ENOTSOCK); v != FailureNotASocket {
			t.Fatalf("expected '%s', got '%s'", FailureNotASocket, v)
		}
	})

	t.Run("for ENOTCONN", func(t *testing.T) {
		if v := classifySyscallError(ENOTCONN); v != FailureNotConnected {
			t.Fatalf("expected '%s', got '%s'", FailureNotConnected, v)
		}
	})

	t.Run("for EWOULDBLOCK", func(t *testing.T) {
		if v := classifySyscallError(EWOULDBLOCK); v != FailureOperationWouldBlock {
			t.Fatalf("expected '%s', got '%s'", FailureOperationWouldBlock, v)
		}
	})

	t.Run("for EACCES", func(t *testing.T) {
		if v := classifySyscallError(EACCES); v != FailurePermissionDenied {
			t.Fatalf("expected '%s', got '%s'", FailurePermissionDenied, v)
		}
	})

	t.Run("for EPROTONOSUPPORT", func(t *testing.T) {
		if v := classifySyscallError(EPROTONOSUPPORT); v != FailureProtocolNotSupported {
			t.Fatalf("expected '%s', got '%s'", FailureProtocolNotSupported, v)
		}
	})

	t.Run("for EPROTOTYPE", func(t *testing.T) {
		if v := classifySyscallError(EPROTOTYPE); v != FailureWrongProtocolType {
			t.Fatalf("expected '%s', got '%s'", FailureWrongProtocolType, v)
		}
	})

	t.Run("for the zero errno value", func(t *testing.T) {
		if v := classifySyscallError(syscall.Errno(0)); v != "" {
			t.Fatalf("expected empty string, got '%s'", v)
		}
	})
}
