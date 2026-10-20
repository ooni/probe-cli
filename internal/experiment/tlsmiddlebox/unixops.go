//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tlsmiddlebox

import "golang.org/x/sys/unix"

// unixOpsImpl is an implementation of an interface consisting of unix functions
type unixOps interface {
	Socket(domain, typ, proto int) (int, error)
	Close(fd int) error
	SetsockoptInt(fd, level, opt, value int) error
	SetNonblock(fd int, nonblocking bool) error
	Connect(fd int, sa unix.Sockaddr) error
	Poll(fds []unix.PollFd, timeout int) (int, error)
	Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error)
	GetsockoptInt(fd, level, opt int) (int, error)
}

type unixOpsImpl struct{}

var _ unixOps = &unixOpsImpl{}

// Socket is an implementation of the unix Socket function
func (unixOpsImpl) Socket(domain, typ, proto int) (int, error) {
	return unix.Socket(domain, typ, proto)
}

// Close is an implementation of the unix Close function
func (unixOpsImpl) Close(fd int) error {
	return unix.Close(fd)
}

// SetsockoptInt is an implementation of the unix SetsockoptInt function
func (unixOpsImpl) SetsockoptInt(fd, level, opt, value int) error {
	return unix.SetsockoptInt(fd, level, opt, value)
}

// SetNonblock is an implementation of the unix SetNonblock function
func (unixOpsImpl) SetNonblock(fd int, nonblocking bool) error {
	return unix.SetNonblock(fd, nonblocking)
}

// Connect is an implementation of the unix Connect function
func (unixOpsImpl) Connect(fd int, sa unix.Sockaddr) error {
	return unix.Connect(fd, sa)
}

// Poll is an implementation of the unix Poll function
func (unixOpsImpl) Poll(fds []unix.PollFd, timeout int) (int, error) {
	return unix.Poll(fds, timeout)
}

// Recvmsg is an implementation of the unix Recvmsg function
func (unixOpsImpl) Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
	return unix.Recvmsg(fd, p, oob, flags)
}

// GetsockoptInt is an implementation of the unix GetsockoptInt function
func (unixOpsImpl) GetsockoptInt(fd, level, opt int) (int, error) {
	return unix.GetsockoptInt(fd, level, opt)
}
