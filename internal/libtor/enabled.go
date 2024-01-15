//go:build ooni_libtor

package libtor

// Adapted from https://github.com/cretz/bine
// SPDX-License-Identifier: MIT

//
// #cgo linux,amd64 CFLAGS: -I${SRCDIR}/linux/amd64/include
// #cgo linux,amd64 LDFLAGS: -L${SRCDIR}/linux/amd64/lib -ltor -levent -lssl -lcrypto -lz -lm
//
// #cgo android,arm CFLAGS: -I${SRCDIR}/android/arm/include
// #cgo android,arm LDFLAGS: -L${SRCDIR}/android/arm/lib -ltor -levent -lssl -lcrypto -lz -lm
// #cgo android,arm64 CFLAGS: -I${SRCDIR}/android/arm64/include
// #cgo android,arm64 LDFLAGS: -L${SRCDIR}/android/arm64/lib -ltor -levent -lssl -lcrypto -lz -lm
// #cgo android,386 CFLAGS: -I${SRCDIR}/android/386/include
// #cgo android,386 LDFLAGS: -L${SRCDIR}/android/386/lib -ltor -levent -lssl -lcrypto -lz -lm
// #cgo android,amd64 CFLAGS: -I${SRCDIR}/android/amd64/include
// #cgo android,amd64 LDFLAGS: -L${SRCDIR}/android/amd64/lib -ltor -levent -lssl -lcrypto -lz -lm
//
// #cgo ios CFLAGS: -I${SRCDIR}
//
// #include <limits.h>
// #include <stdbool.h>
// #include <stdlib.h>
//
// /* Select the correct header depending on the Apple's platform and architecture, otherwise, for
//    other operating systems just use the header in the include path defined above.
//
//    See https://stackoverflow.com/a/18729350 for details. */
// #if defined(__APPLE__) && defined(__MACH__)
//   #include <TargetConditionals.h>
//   #if TARGET_OS_IPHONE && TARGET_OS_SIMULATOR
//     #if TARGET_CPU_X86_64
//       #include <iphonesimulator/amd64/include/tor_api.h>
//     #elif TARGET_CPU_ARM64
//       #include <iphonesimulator/arm64/include/tor_api.h>
//     #else
//       #error "internal/libtor/enabled.go: unhandled Apple architecture"
//     #endif
//   #elif TARGET_OS_IPHONE && TARGET_OS_MACCATALYST
//     #error "internal/libtor/enabled.go: unhandled Apple platform"
//   #elif TARGET_OS_IPHONE
//     #include <iphoneos/arm64/include/tor_api.h>
//   #else
//     #error "internal/libtor/enabled.go: unhandled Apple platform"
//   #endif
// #else
//   #include <tor_api.h>
// #endif
//
// /* Note: we need to define inline helpers because we cannot index C arrays in Go. */
//
// static char **cstringArrayNew(size_t size) {
//     char **argv = calloc(size, sizeof(char *));
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
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cretz/bine/process"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
		awaitStart:  make(chan any, 1), // buffer
		controlConn: left,
		startErr:    make(chan error, 1), // buffer
		startOnce:   sync.Once{},
		waitErr:     make(chan error, 1), // buffer
		waitOnce:    sync.Once{},

		closedWhenNotStarted:     make(chan any, 1), // buffer
		simulateBadControlSocket: false,
		simulateFileConnFailure:  false,
		simulateNonzeroExitCode:  false,
	}
	go proc.runtor(ctx, right, args...)
	return proc, nil
}

// torProcess implements [process.Process].
type torProcess struct {
	// ordinary state variables
	awaitStart  chan any
	controlConn net.Conn
	startErr    chan error
	startOnce   sync.Once
	waitErr     chan error
	waitOnce    sync.Once

	// for testing
	closedWhenNotStarted     chan any
	simulateBadControlSocket bool
	simulateFileConnFailure  bool
	simulateNonzeroExitCode  bool
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

// ErrConcurrentCalls indicates there have been concurrent libtor calls, which
// would lead to memory corruption inside of libtor.a.
var ErrConcurrentCalls = errors.New("libtor: another thread is already running tor")

// ErrTooManyArguments indicates that p.args contains too many arguments
var ErrTooManyArguments = errors.New("libtor: too many arguments")

// ErrCannotCreateControlSocket indicates that we cannot create a control socket.
var ErrCannotCreateControlSocket = errors.New("libtor: cannot create a control socket")

// ErrNonzeroExitCode indicates that tor returned a nonzero exit code
var ErrNonzeroExitCode = errors.New("libtor: command completed with nonzero exit code")

// concurrentCalls prevents concurrent libtor.a calls.
var concurrentCalls = &atomic.Int64{}

// runtor runs tor until completion and ensures that tor exits when
// the given ctx is cancelled or its deadline expires.
func (p *torProcess) runtor(ctx context.Context, cc net.Conn, args ...string) {
	// make sure we lock to an OS thread otherwise the goroutine can get
	// preempted midway and cause data races
	//
	// See https://github.com/ooni/probe/issues/2406#issuecomment-1479138677
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// wait for Start or context to expire
	select {
	case <-p.awaitStart:
	case <-ctx.Done():
		p.startErr <- ctx.Err() // nonblocking chan
		close(p.closedWhenNotStarted)
		return
	}

	// make sure we're not going to have actual concurrent calls.
	if !concurrentCalls.CompareAndSwap(0, 1) {
		p.startErr <- ErrConcurrentCalls // nonblocking channel
		return
	}
	defer concurrentCalls.Store(0)

	// Note: when writing this code I was wondering whether I needed to
	// use unsafe.Pointer to track pointers that matter to C code. Reading
	// this message[1] has been useful to understand that the most likely
	// answer to this question is "obviously, no".
	//
	// See https://groups.google.com/g/golang-nuts/c/yNis7bQG_rY/m/yaJFoSx1hgIJ

	// Create argc and argv for tor
	argv := append([]string{"tor"}, args...)
	const toomany = 256 // arbitrary low limit to make C.int and C.size_t casts always work
	if len(argv) > toomany {
		p.startErr <- ErrTooManyArguments // nonblocking channel
		return
	}
	argc := C.size_t(len(argv))
	// Note: here we allocate argc + 1 because a "null pointer always follows
	// the last element: argv[argc] is this null pointer."
	//
	// See https://www.gnu.org/software/libc/manual/html_node/Program-Arguments.html
	allocSiz := argc + 1
	cargv := C.cstringArrayNew(allocSiz)
	defer C.cstringArrayFree(cargv, argc)
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
	if p.simulateBadControlSocket {
		filedesc = C.INVALID_TOR_CONTROL_SOCKET
	}
	if !C.filedescIsGood(filedesc) {
		p.startErr <- ErrCannotCreateControlSocket // nonblocking channel
		return
	}

	// Convert the OWNING file descriptor into a proper file. Because
	// filedesc is good, os.NewFile shouldn't fail.
	filep := os.NewFile(uintptr(filedesc), "")
	runtimex.Assert(filep != nil, "os.NewFile should not fail")

	// Create a new net.Conn using a copy of the underlying
	// file descriptor using net.FileConn (see below).
	conn, err := net.FileConn(filep)
	if p.simulateFileConnFailure {
		err = ErrCannotCreateControlSocket
	}
	if err != nil {
		filep.Close()
		p.startErr <- err // nonblocking channel
		return
	}

	// From the documentation of [net.FileConn]:
	//
	//	It is the caller's responsibility to close f when
	//	finished. Closing c does not affect f, and closing
	//	f does not affect c.
	//
	// So, it's safe to close the filep now.
	filep.Close()

	// In the following we're going to possibly call Close multiple
	// times. Let's be very sure that this close is idempotent.
	conn = withIdempotentClose(conn)
	cc = withIdempotentClose(cc)

	// Make sure we close filep when the context is done. Because the
	// socket is OWNING, this will also cause tor to return.
	go func() {
		defer conn.Close()
		defer cc.Close()
		<-ctx.Done()
	}()

	// Route messages to and from the control connection.
	go sendrecvThenClose(cc, conn)
	go sendrecvThenClose(conn, cc)

	// Let the user know that startup was successful.
	p.startErr <- nil // nonblocking channel

	// Run tor until completion.
	if !p.simulateNonzeroExitCode {
		code = C.tor_run_main(config)
	} else {
		// when simulating nonzero exit code we also want to sleep for a bit
		// of time, to make sure we're able to see overalapped runs.
		code = 1
		time.Sleep(time.Second)
	}
	if code != 0 {
		p.waitErr <- fmt.Errorf("%w: %d", ErrNonzeroExitCode, code) // nonblocking channel
		return
	}
	p.waitErr <- nil // nonblocking channel
}

// sendrecvThenClose routes traffic between two connections and then
// closes both of them when done with routing traffic.
func sendrecvThenClose(left, right net.Conn) {
	defer left.Close()
	defer right.Close()
	netxlite.CopyContext(context.Background(), left, right)
}

// withIdempotentClose ensures that a connection has idempotent close.
func withIdempotentClose(c net.Conn) net.Conn {
	return &idempotentClose{
		Conn: c,
		once: sync.Once{},
	}
}

// idempotentClose ensures close is idempotent for a net.Conn
type idempotentClose struct {
	net.Conn
	once sync.Once
}

func (c *idempotentClose) Close() (err error) {
	c.once.Do(func() {
		err = c.Conn.Close()
	})
	return
}
