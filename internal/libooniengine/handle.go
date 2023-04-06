package main

//
// Handle mimics cgo.Handle but uses a intptr
//

//#include <stdlib.h>
//
//#include "engine.h"
import "C"

import (
	"errors"
	"log"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/motor"
)

var (
	handler Handler

	// errHandleMisuse indicates that an invalid handle was misused
	errHandleMisuse = errors.New("misuse of a invalid handle")

	// errHandleSpaceExceeded
	errHandleSpaceExceeded = errors.New("ran out of handle space")
)

func init() {
	handler = Handler{
		handles: make(map[Handle]interface{}),
	}
}

type Handle C.intptr_t

// Handler handles the entirety of handle operations.
type Handler struct {
	handles   map[Handle]interface{}
	handleIdx Handle
	mu        sync.Mutex
}

// newHandle returns a handle for a given value
// if we run out of handle space, a zero handle is returned.
func (h *Handler) newHandle(v any) (Handle, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ptr := C.intptr_t(h.handleIdx)
	newId := ptr + 1
	if newId < 0 {
		return Handle(0), errHandleSpaceExceeded
	}
	h.handleIdx = Handle(newId)
	h.handles[h.handleIdx] = v
	return h.handleIdx, nil
}

// delete invalidates a handle
func (h *Handler) delete(hd Handle) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.handles, hd)
}

// value returns the associated go value for a valid handle
func (h *Handler) value(hd Handle) (any, error) {
	v, ok := h.handles[hd]
	if !ok {
		return nil, errHandleMisuse
	}
	return v, nil
}

// getTaskHandle checks if the task handle is valid and returns the corresponding TaskAPI.
func (h *Handler) getTaskHandle(task C.OONITask) (tp motor.TaskAPI) {
	hd := Handle(task)
	val, err := h.value(hd)
	if err != nil {
		log.Printf("getTaskHandle: %s", err.Error())
		return
	}
	tp, ok := val.(motor.TaskAPI)
	if !ok {
		log.Printf("getTaskHandle: invalid type assertion")
		return
	}
	return
}
