//go:build ooni_libtor

package libtor

// Adapted from https://github.com/cretz/bine
// SPDX-License-Identifier: MIT

//
// #cgo darwin,arm64 CFLAGS: -I${SRCDIR}/darwin/arm64/include
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/darwin/arm64/lib -ltor -levent -lssl -lcrypto -lz
//
// #cgo darwin,amd64 CFLAGS: -I${SRCDIR}/darwin/amd64/include
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/darwin/amd64/lib -ltor -levent -lssl -lcrypto -lz
//
// #cgo linux,amd64 CFLAGS: -I${SRCDIR}/linux/amd64/include
// #cgo linux,amd64 LDFLAGS: -L${SRCDIR}/linux/amd64/lib -ltor -levent -lssl -lcrypto -lz -lm
//
// #include <limits.h>
// #include <stdbool.h>
// #include <stdlib.h>
//
// #include <tor_api.h>
//
// /* Note: we need to define inline helpers because we cannot index C arrays in Go. */
//
// static char **cstringArrayNew(size_t size) {
//     char **argv = calloc(sizeof(char *), size);
//     if (argv == NULL) {
//         abort();
//     }
//     return argv;
// }
//
// static void cstringArraySet(char **argv, size_t index, char *entry) {
//     argv[index] = entry;
// }
//
// static void cstringArrayFree(char **argv, size_t size) {
//     for (size_t idx = 0; idx < size; idx++) {
//         free(argv[idx]);
//     }
//     free(argv);
// }
//
// static bool filedescIsGood(tor_control_socket_t fd) {
//     return fd != INVALID_TOR_CONTROL_SOCKET;
// }
//
import "C"

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"

	"github.com/cretz/bine/process"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// MaybeCreator returns a valid [process.Creator], if possible, otherwise false.
func MaybeCreator() (process.Creator, bool) {
	return &torCreator{}, true
}

// torCreator implements [process.Creator].
type torCreator struct{}

var _ process.Creator = &torCreator{}

// New implements [process.Creator].
func (c *torCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	left, right := net.Pipe()
	proc := &torProcess{
		awaitStart:  make(chan any, 1),
		controlConn: left,
		startErr:    make(chan error, 1),
		startOnce:   sync.Once{},
		waitErr:     make(chan error, 1),
		waitOnce:    sync.Once{},
	}
	go proc.runtor(ctx, right, args...)
	return proc, nil
}

// torProcess implements [process.Process].
type torProcess struct {
	awaitStart  chan any
	controlConn net.Conn
	startErr    chan error
	startOnce   sync.Once
	waitErr     chan error
	waitOnce    sync.Once
}

var _ process.Process = &torProcess{}

// EmbeddedControlConn implements [process.Process].
func (p *torProcess) EmbeddedControlConn() (net.Conn, error) {
	// Implementation note: this function SHOULD only be called
	// once and BEFORE Start is called 😬😬😬
	return p.controlConn, nil
}

// Start implements [process.Process].
func (p *torProcess) Start() (err error) {
	p.startOnce.Do(func() {
		p.awaitStart <- true
		err = <-p.startErr
	})
	return err
}

// Wait implements [process.Process].
func (p *torProcess) Wait() (err error) {
	p.waitOnce.Do(func() {
		err = <-p.waitErr
	})
	return
}

var (
	// ErrTooManyArguments indicates that p.args contains too many arguments
	ErrTooManyArguments = errors.New("libtor: too many arguments")

	// ErrCannotCreateControlSocket indicates that we cannot create a control socket.
	ErrCannotCreateControlSocket = errors.New("libtor: cannot create a control socket")
)

// runtor runs tor until completion and ensures that tor exits when
// the given ctx is cancelled or its deadline expires.
func (p *torProcess) runtor(ctx context.Context, cc net.Conn, args ...string) {
	// wait for Start or context to expire
	select {
	case <-p.awaitStart:
	case <-ctx.Done():
		return
	}

	// Create argc and argv for tor
	argv := append([]string{"tor"}, args...)
	const toomany = 256 // arbitrary low limit to make C.int and C.size_t casts always work
	if len(argv) > 256 {
		p.startErr <- ErrTooManyArguments // nonblocking channel
		return
	}
	argc := C.size_t(len(argv))
	cargv := C.cstringArrayNew(argc)
	defer C.cstringArrayFree(cargv, C.size_t(argc))
	for idx, entry := range argv {
		C.cstringArraySet(cargv, C.size_t(idx), C.CString(entry))
	}

	// Add to config a WEAK REFERENCE to argc and argv
	config := C.tor_main_configuration_new()
	runtimex.PanicIfNil(config, "C.tor_main_configuration_new failed")
	defer C.tor_main_configuration_free(config)
	code := C.tor_main_configuration_set_command_line(config, C.int(argc), cargv)
	runtimex.Assert(code == 0, "C.tor_main_configuration_set_command_line failed")

	// Create OWNING file descriptor
	filedesc := C.tor_main_configuration_setup_control_socket(config)
	if !C.filedescIsGood(filedesc) {
		p.startErr <- ErrCannotCreateControlSocket // nonblocking channel
		return
	}

	// Convert the OWNING file descriptor into a proper file.
	filep, err := net.FileConn(os.NewFile(uintptr(filedesc), ""))
	if err != nil {
		p.startErr <- err // nonblocking channel
		return
	}

	// Make sure we close filep when the context is done. Because the
	// socket is OWNING, this will also cause tor to return.
	go func() {
		defer filep.Close()
		defer cc.Close()
		<-ctx.Done()
	}()

	// Route messages from and to the control connection
	go sendrecv(cc, filep)
	go sendrecv(filep, cc)

	// Tell user that startup is successful.
	p.startErr <- nil

	// Run tor until completion. Note that return codes are not
	// currently documented and they're never zero (WTF?!).
	_ = C.tor_run_main(config)

	// Tell user that we terminated.
	p.waitErr <- nil // nonblocking channel
}

// sendrecv routes traffic between two connections.
func sendrecv(left, right net.Conn) {
	io.Copy(left, right)
}
