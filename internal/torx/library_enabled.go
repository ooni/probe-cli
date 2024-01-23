package torx

//
// library_enabled.go - code to use tor as a library.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

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
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// These flags help with simulating specific testing conditions.
const (
	libraryProcessSimulateBadControlSocket = 1 << iota
	libraryProcessSimulateNonZeroExitCode
)

// libraryProcess is the [Process] returned by [libraryExec].
//
// WARNING: we know that starting embedded tor multiple times is going to cause
// crashes as described by [ooni/probe#2406]. This API allows the caller to start/stop
// embedded tor multiple times, but this is definitely NOT RECOMMENDED.
//
// [ooni/probe#2406]: https://github.com/ooni/probe/issues/2406.
type libraryProcess struct {
	// config contains the exec options.
	config *execOptions

	// cconn is the owning control connection.
	cconn io.Closer

	// cconnch is used to send the control connection from the
	// background goroutine back to the call function.
	cconnch chan io.Closer

	// closeOnce provides once semantics for shutting tor down.
	closeOnce *sync.Once

	// datadir contains the data directory state.
	datadir *DataDirState

	// exitCode is used to communicate the tor_run_main exit code.
	exitCode chan int

	// logger is the logger to use.
	logger model.Logger

	// startErr is where we post any startup error.
	startErr chan error

	// testflags contains OPTIONAL test flags.
	testflags int
}

// libraryExec is like [Exec] but the returned [Process] is not an external
// operating system process, rather a thread running tor_run_main.
//
// WARNING: we know that starting embedded tor multiple times is going to cause
// crashes as described by [ooni/probe#2406]. This API allows the caller to start/stop
// embedded tor multiple times, but this is definitely NOT RECOMMENDED.
//
// [ooni/probe#2406]: https://github.com/ooni/probe/issues/2406.
func libraryExec(datadir *DataDirState, logger model.Logger, options ...ExecOption) (Process, error) {
	// init the config
	config := &execOptions{
		deps:         &execDepsStdlib{},
		torBinary:    "",
		torExtraArgs: []string{},
	}
	for _, option := range options {
		option(config)
	}

	// initialize the library process
	lp := &libraryProcess{
		config:    config,
		cconn:     nil,
		cconnch:   make(chan io.Closer),
		closeOnce: &sync.Once{},
		datadir:   datadir,
		exitCode:  make(chan int),
		logger:    logger,
		startErr:  make(chan error),
		testflags: 0,
	}

	// run tor in the background as a library function
	go libraryProcessMain(lp)

	// wait for errors
	if err := <-lp.startErr; err != nil {
		return nil, err
	}

	// obtain the owning control conn
	lp.cconn = <-lp.cconnch

	// return to the caller.
	return lp, nil
}

// DialControl implements Process.
func (lp *libraryProcess) DialControl(ctx context.Context) (conn *ControlConn, err error) {
	return dialControl(ctx, lp.datadir, lp.logger)
}

// Kill implements Process.
func (lp *libraryProcess) Kill() (err error) {
	lp.closeOnce.Do(func() {
		err = lp.cconn.Close()
	})
	return
}

// Wait implements Process.
func (lp *libraryProcess) Wait() (ProcessState, error) {
	state := &libraryProcessState{<-lp.exitCode}
	return state, nil
}

// libraryProcessState implements [ProcessState] for [*libraryProcess].
type libraryProcessState struct {
	exitcode int
}

// ExitCode implements [ProcessState].
func (lps *libraryProcessState) ExitCode() int {
	return lps.exitcode
}

// ErrLibraryCallTooManyArguments indicates that p.args contains too many arguments
var ErrLibraryCallTooManyArguments = errors.New("torx: too many arguments")

// ErrLibraryCallCannotCreateControlSocket indicates that we cannot create a control socket.
var ErrLibraryCallCannotCreateControlSocket = errors.New("torx: cannot create a control socket")

func libraryProcessMain(lp *libraryProcess) {
	// make sure we lock to an OS thread otherwise the goroutine can get
	// preempted midway and cause data races
	//
	// See https://github.com/ooni/probe/issues/2406#issuecomment-1479138677
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Note: when writing this code I was wondering whether I needed to
	// use unsafe.Pointer to track pointers that matter to C code. Reading
	// this message[1] has been useful to understand that the most likely
	// answer to this question is "obviously, no".
	//
	// See https://groups.google.com/g/golang-nuts/c/yNis7bQG_rY/m/yaJFoSx1hgIJ

	// Create argc and argv for tor
	argv := []string{
		"tor",
		"-f", lp.datadir.TorRcFile,
		//"__OwningControllerFD", "1", // already provided by tor_run_main!
		"__DisableSignalHandlers", "1",
		"ControlPort", "auto",
		"ControlPortWriteToFile", lp.datadir.ControlPortFile,
		"CookieAuthentication", "1",
		"CookieAuthFile", lp.datadir.CookieAuthFile,
		"DataDirectory", lp.datadir.DirPath,
		"DisableNetwork", "1",
		"SocksPort", "auto",
	}
	argv = append(argv, lp.config.torExtraArgs...)
	const toomany = 256 // arbitrary low limit to make C.int and C.size_t casts always work
	if len(argv) > toomany {
		lp.startErr <- ErrLibraryCallTooManyArguments
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

	lp.logger.Infof("torx: library: exec: %s", strings.Join(argv, " "))

	// Add to config a WEAK REFERENCE to argc and argv
	config := C.tor_main_configuration_new()
	runtimex.PanicIfNil(config, "C.tor_main_configuration_new failed")
	defer C.tor_main_configuration_free(config)
	code := C.tor_main_configuration_set_command_line(config, C.int(argc), cargv)
	runtimex.Assert(code == 0, "C.tor_main_configuration_set_command_line failed")

	// Create OWNING file descriptor
	filedesc := C.tor_main_configuration_setup_control_socket(config)
	if lp.testflags&libraryProcessSimulateBadControlSocket != 0 {
		filedesc = C.INVALID_TOR_CONTROL_SOCKET
	}
	if !C.filedescIsGood(filedesc) {
		lp.startErr <- ErrLibraryCallCannotCreateControlSocket
		return
	}

	// Convert the OWNING file descriptor into a proper file. Because
	// filedesc is good, os.NewFile shouldn't fail.
	filep := os.NewFile(uintptr(filedesc), "")
	runtimex.Assert(filep != nil, "os.NewFile should not fail")

	// Let the parent function know that startup was successful.
	lp.startErr <- nil

	// Provide the control conn to the parent func.
	//
	// Implementation note: we wrap the filep to have idempotent close
	// because we want to be really sure about closing it only once since
	// Android may crash if we attempt to close files more than once
	// as documented by [ooni/probe#2405].
	//
	// [ooni/probe#2405]: https://github.com/ooni/probe/issues/2405
	lp.cconnch <- withIdempotentClose(filep)

	// Add facilities to simulating tor_run_main failures in tests.
	if lp.testflags&libraryProcessSimulateNonZeroExitCode != 0 {
		lp.exitCode <- 1
		return
	}

	// Run tor until completion.
	exitcode := C.tor_run_main(config)
	lp.exitCode <- int(exitcode)
}

// withIdempotentClose ensures that a connection has idempotent close.
func withIdempotentClose(c io.Closer) io.Closer {
	return &idempotentClose{
		Closer: c,
		once:   sync.Once{},
	}
}

// idempotentClose ensures close is idempotent for a net.Conn
type idempotentClose struct {
	io.Closer
	once sync.Once
}

func (c *idempotentClose) Close() (err error) {
	c.once.Do(func() {
		err = c.Closer.Close()
	})
	return
}
