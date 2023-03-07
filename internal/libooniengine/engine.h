// SPDX-License-Identifier: GPL-3.0-or-later

#ifndef OONI_ENGINE_H
#define OONI_ENGINE_H

///
/// C API for using the OONI engine.
///

#include <stdint.h>

/// OONITask is an asynchronous thread of execution managed by the OONI
/// engine that performs a background operation and emits meaningful
/// events such as, for example, the results of measurements.
typedef uintptr_t OONITask;

#ifdef __cplusplus
extern "C" {
#endif

/// OONIEngineVersion return the current engine version.
///
/// @return A char pointer with the current version string.
char *OONIEngineVersion(void);

/// OONIEngineFreeMemory frees the memory allocated by the engine.
///
/// @param ptr a void pointer refering to the memory to be freed.
void OONIEngineFreeMemory(void *ptr);

/// OONIEngineCall starts a new OONITask using the given [req].
///
/// @param req A JSON string, owned by the caller, that
/// contains the configuration for the task to start.  
///
/// @return Zero on failure, nonzero on success. If the return value
/// is nonzero, a task is running. In such a case, the caller is
/// responsible to eventually dispose of the task using OONIEngineFreeMemory.
OONITask OONIEngineCall(char *req);

/// OONIEngineWaitForNextEvent awaits on the [task] event queue until
/// a new event is available or the given [timeout] expires.
///
/// @param task Task handle returned by OONIEngineCall.
///
/// @param timeout Timeout in milliseconds. If the timeout is zero
/// or negative, this function would potentially block forever.
///
/// @return A NULL pointer on failure, non-NULL otherwise. If the return
/// value is non-NULL, the caller takes ownership of the OONIMessage
/// pointer and MUST free it using OONIMessageFree when done using it.
///
/// This function will return NULL:
///
/// 1. when the timeout expires;
///
/// 2. if [task] is done;
///
/// 3. if [task] is zero or does not refer to a valid task;
///
/// 4. if we cannot protobuf serialize the message;
///
/// 5. possibly because of other unknown internal errors.
///
/// In short, you cannot reliably determine whether a task is done by
/// checking whether this function has returned NULL.
char *OONIEngineWaitForNextEvent(OONITask task, int32_t timeout);

/// OONIEngineInterrupt tells [task] to stop ASAP.
///
/// @param task Task handle returned by OONIEngineCall. If task is zero
/// or does not refer to a valid task, this function will just do nothing.
void OONIEngineInterrupt(OONITask task);

// OONITask

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
