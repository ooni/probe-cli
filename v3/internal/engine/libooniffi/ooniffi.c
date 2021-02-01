#include "ooniffi.h"

#include "_cgo_export.h"

ooniffi_task_t *ooniffi_task_start(const char *settings) {
    /* Implementation note: Go does not have the concept of const but
       we know that the code is just making a copy of settings. */
    return ooniffi_task_start_((char *)settings);
}

const char *ooniffi_event_serialization(ooniffi_event_t *event) {
    /* Implementation note: Go does not have the concept of const but
       we want to return const to very clearly communicate that the
       returned string is owned by the event. This is what tools like
       python's ctypes and SWIG expect from us. */
    return (const char *)ooniffi_event_serialization_(event);
}
