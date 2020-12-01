// Package syslog contains a syslog handler.
//
// We use this handler on macOS systems to log messages
// when ooniprobe is running in the background.
package syslog

import (
	"fmt"
	"unsafe"

	"github.com/apex/log"
)

/*
#include<stdlib.h>

void ooniprobe_openlog(void);
void ooniprobe_log_debug(const char *message);
void ooniprobe_log_info(const char *message);
void ooniprobe_log_warning(const char *message);
void ooniprobe_log_err(const char *message);
void ooniprobe_log_crit(const char *message);
*/
import "C"

// Default is the handler that emits logs with syslog
var Default log.Handler = newhandler()

type handler struct{}

func newhandler() handler {
	C.ooniprobe_openlog()
	return handler{}
}

func (h handler) HandleLog(e *log.Entry) error {
	message := fmt.Sprintf("%s %+v", e.Message, e.Fields)
	cstr := C.CString(message)
	defer C.free(unsafe.Pointer(cstr))
	switch e.Level {
	case log.DebugLevel:
		C.ooniprobe_log_debug(cstr)
	case log.InfoLevel:
		C.ooniprobe_log_info(cstr)
	case log.WarnLevel:
		C.ooniprobe_log_warning(cstr)
	case log.ErrorLevel:
		C.ooniprobe_log_err(cstr)
	default:
		C.ooniprobe_log_crit(cstr)
	}
	return nil
}
