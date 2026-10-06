//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tlsmiddlebox

import "golang.org/x/sys/unix"

// mockUnixOpsImpl is an implementation of an interface consisting of unix functions
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

// Socket is an implementation of the unix Socket function for testing
func (unixOpsImpl) Socket(domain, typ, proto int) (int, error) {
	return unix.Socket(domain, typ, proto)
}

// Close is an implementation of the unix Close function for testing
func (unixOpsImpl) Close(fd int) error {
	return unix.Close(fd)
}

// SetsockoptInt is an implementation of the unix SetsockoptInt function for testing
func (unixOpsImpl) SetsockoptInt(fd, level, opt, value int) error {
	return unix.SetsockoptInt(fd, level, opt, value)
}

// SetNonblock is an implementation of the unix SetNonblock function for testing
func (unixOpsImpl) SetNonblock(fd int, nonblocking bool) error {
	return unix.SetNonblock(fd, nonblocking)
}

// Connect is an implementation of the unix Connect function for testing
func (unixOpsImpl) Connect(fd int, sa unix.Sockaddr) error {
	return unix.Connect(fd, sa)
}

// Poll is an implementation of the unix Poll function for testing
func (unixOpsImpl) Poll(fds []unix.PollFd, timeout int) (int, error) {
	return unix.Poll(fds, timeout)
}

// Recvmsg is an implementation of the unix Recvmsg function for testing
func (unixOpsImpl) Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
	return unix.Recvmsg(fd, p, oob, flags)
}

// GetsockoptInt is an implementation of the unix GetsockoptInt function for testing
func (unixOpsImpl) GetsockoptInt(fd, level, opt int) (int, error) {
	return unix.GetsockoptInt(fd, level, opt)
}
