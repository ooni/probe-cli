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
// #include <limits.h>
// #include <stdbool.h>
// #include <stdlib.h>
//
// #include <tor_api.h>
//
// /* Note: we need to define inline helpers because we cannot index C arrays in Go. */
//
// static char** makeCharArray(int size) {
// 	return calloc(sizeof(char*), size);
// }
// static void setArrayString(char **a, char *s, int n) {
// 	a[n] = s;
// }
// static void freeCharArray(char **a, int size) {
// 	int i;
// 	for (i = 0; i < size; i++)
// 		free(a[i]);
// 	free(a);
// }
//
import "C"

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/cretz/bine/process"
)

// MaybeCreator returns a valid [process.Creator], if possible, otherwise false.
func MaybeCreator() (process.Creator, bool) {
	return Creator, true
}

// Creator implements the bine.process.Creator, permitting libtor to act as an API
// backend for the bine/tor Go interface.
var Creator process.Creator = new(embeddedCreator)

// embeddedCreator implements process.Creator, permitting libtor to act as an API
// backend for the bine/tor Go interface.
type embeddedCreator struct{}

// New implements process.Creator, creating a new embedded tor process.
func (embeddedCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return &embeddedProcess{
		ctx:  ctx,
		conf: C.tor_main_configuration_new(),
		args: args,
	}, nil
}

// embeddedProcess implements process.Process, permitting libtor to act as an API
// backend for the bine/tor Go interface.
type embeddedProcess struct {
	ctx  context.Context
	conf *C.struct_tor_main_configuration_t
	args []string
	done chan int
}

// Start implements process.Process, starting up the libtor embedded process.
func (e *embeddedProcess) Start() error {
	if e.done != nil {
		return errors.New("already started")
	}
	// Create the char array for the args
	args := append([]string{"tor"}, e.args...)

	charArray := C.makeCharArray(C.int(len(args)))
	for i, a := range args {
		C.setArrayString(charArray, C.CString(a), C.int(i))
	}
	// Build the tor configuration
	if code := C.tor_main_configuration_set_command_line(e.conf, C.int(len(args)), charArray); code != 0 {
		C.tor_main_configuration_free(e.conf)
		C.freeCharArray(charArray, C.int(len(args)))
		return fmt.Errorf("failed to set arguments: %v", int(code))
	}
	// Start tor and return
	e.done = make(chan int, 1)
	go func() {
		defer C.freeCharArray(charArray, C.int(len(args)))
		defer C.tor_main_configuration_free(e.conf)
		e.done <- int(C.tor_run_main(e.conf))
	}()
	return nil
}

// Wait implements process.Process, blocking until the embedded process terminates.
func (e *embeddedProcess) Wait() error {
	if e.done == nil {
		return errors.New("not started")
	}
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()

	case code := <-e.done:
		if code == 0 {
			return nil
		}
		return fmt.Errorf("embedded tor failed: %v", code)
	}
}

// EmbeddedControlConn implements process.Process, connecting to the control port
// of the embedded Tor isntance.
func (e *embeddedProcess) EmbeddedControlConn() (net.Conn, error) {
	file := os.NewFile(uintptr(C.tor_main_configuration_setup_control_socket(e.conf)), "")
	conn, err := net.FileConn(file)
	if err != nil {
		return nil, fmt.Errorf("unable to create control socket: %v", err)
	}
	return conn, nil
}
