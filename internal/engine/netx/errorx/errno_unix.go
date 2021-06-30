package errorx

import "golang.org/x/sys/unix"

const (
	ECANCELED       = unix.ECANCELED
	ECONNREFUSED    = unix.ECONNREFUSED
	ECONNRESET      = unix.ECONNRESET
	EHOSTUNREACH    = unix.EHOSTUNREACH
	ETIMEDOUT       = unix.ETIMEDOUT
	EAFNOSUPPORT    = unix.EAFNOSUPPORT
	EADDRINUSE      = unix.EADDRINUSE
	EADDRNOTAVAIL   = unix.EADDRNOTAVAIL
	EISCONN         = unix.EISCONN
	EFAULT          = unix.EFAULT
	EBADF           = unix.EBADF
	ECONNABORTED    = unix.ECONNABORTED
	EALREADY        = unix.EALREADY
	EDESTADDRREQ    = unix.EDESTADDRREQ
	EINTR           = unix.EINTR
	EINVAL          = unix.EINVAL
	EMSGSIZE        = unix.EMSGSIZE
	ENETDOWN        = unix.ENETDOWN
	ENETRESET       = unix.ENETRESET
	ENETUNREACH     = unix.ENETUNREACH
	ENOBUFS         = unix.ENOBUFS
	ENOPROTOOPT     = unix.ENOPROTOOPT
	ENOTSOCK        = unix.ENOTSOCK
	ENOTCONN        = unix.ENOTCONN
	EWOULDBLOCK     = unix.EWOULDBLOCK
	EACCES          = unix.EACCES
	EPROTONOSUPPORT = unix.EPROTONOSUPPORT
	EPROTOTYPE      = unix.EPROTOTYPE
)
