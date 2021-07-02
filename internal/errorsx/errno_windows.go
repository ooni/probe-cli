package errorsx

//go:generate go run ./generator/ -m windows

import "golang.org/x/sys/windows"

const (
	ECANCELED       = windows.ECANCELED
	ECONNREFUSED    = windows.ECONNREFUSED
	ECONNRESET      = windows.ECONNRESET
	EHOSTUNREACH    = windows.EHOSTUNREACH
	ETIMEDOUT       = windows.ETIMEDOUT
	EAFNOSUPPORT    = windows.EAFNOSUPPORT
	EADDRINUSE      = windows.EADDRINUSE
	EADDRNOTAVAIL   = windows.EADDRNOTAVAIL
	EISCONN         = windows.EISCONN
	EFAULT          = windows.EFAULT
	EBADF           = windows.EBADF
	ECONNABORTED    = windows.ECONNABORTED
	EALREADY        = windows.EALREADY
	EDESTADDRREQ    = windows.EDESTADDRREQ
	EINTR           = windows.EINTR
	EINVAL          = windows.EINVAL
	EMSGSIZE        = windows.EMSGSIZE
	ENETDOWN        = windows.ENETDOWN
	ENETRESET       = windows.ENETRESET
	ENETUNREACH     = windows.ENETUNREACH
	ENOBUFS         = windows.ENOBUFS
	ENOPROTOOPT     = windows.ENOPROTOOPT
	ENOTSOCK        = windows.ENOTSOCK
	ENOTCONN        = windows.ENOTCONN
	EWOULDBLOCK     = windows.EWOULDBLOCK
	EACCES          = windows.EACCES
	EPROTONOSUPPORT = windows.EPROTONOSUPPORT
	EPROTOTYPE      = windows.EPROTOTYPE
)
