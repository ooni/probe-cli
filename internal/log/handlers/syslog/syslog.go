// Package syslog contains a syslog handler.
package syslog

import (
	"fmt"
	"unsafe"

	"github.com/apex/log"
)

/*
#include<stdlib.h>
#include<syslog.h>

void ooniprobe_openlog(void);
void ooniprobe_syslog(int level, const char *message);
*/
import "C"

// Default is the handler that emits logs with syslog
var Default log.Handler = newhandler()

type handler struct{}

func newhandler() handler {
	C.ooniprobe_openlog()
	return handler{}
}

var levelmap = map[log.Level]C.int{
	log.DebugLevel: C.LOG_DEBUG,
	log.InfoLevel:  C.LOG_INFO,
	log.WarnLevel:  C.LOG_WARNING,
	log.ErrorLevel: C.LOG_ERR,
	log.FatalLevel: C.LOG_CRIT,
}

func getlevel(level log.Level) C.int {
	if value, found := levelmap[level]; found {
		return value
	}
	return C.LOG_CRIT
}

func (h handler) HandleLog(e *log.Entry) error {
	message := fmt.Sprintf("%s %+v", e.Message, e.Fields)
	cstr := C.CString(message)
	defer C.free(unsafe.Pointer(cstr))
	C.ooniprobe_syslog(getlevel(e.Level), cstr)
	return nil
}
