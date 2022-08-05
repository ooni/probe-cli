// SPDX-License-Identifier: GPL-3.0-or-later

#ifndef OONI_ENGINE_H
#define OONI_ENGINE_H

///
/// @file ooni/engine.h
///
/// C API for using the OONI engine.
///

#include <stdbool.h>
#include <stdint.h>

/// OONIMessage is a message sent to or received from the OONI engine.
struct OONIMessage {
	/// Key identifies the message type and allows a protobuf v3
	/// parser to unserialize to the correct value.
	char *Key;

	/// Base is the base pointer of the byte array containing
	/// protobuf v3 serialized data.
	uint8_t *Base;

	/// Size is the size of the byte array.
	uint32_t Size;
};

/// OONITask is an asynchronous thread of execution managed by the OONI
/// engine that performs a background operation and emits meaningful
/// events such as, for example, the results of measurements.
typedef uintptr_t OONITask;

#ifdef __cplusplus
extern "C" {
#endif

/// OONICall calls an OONI engine function and returns the result.
///
/// @param req An OONIMessage structure, owned by the caller, that
/// describes which API to call and with which arguments. The engine
/// will use the message Key to determine which function to call. The
/// engine will reply immediately. It's safe to free [req] once this
/// function has returned a result to the caller.
///
/// @return A NULL pointer on failure, non-NULL otherwise. If the return
/// value is non-NULL, the caller takes ownership of the OONIMessage
/// pointer and MUST free it using OONIMessageFree when done using it.
struct OONIMessage *OONICall(struct OONIMessage *req);

/// OONITaskStart starts a new OONITask using the given [cfg].
///
/// @param cfg An OONIMessage structure, owned by the caller, that
/// contains the configuration for the task to start. The engine will
/// use the message Key to determine which task to start. The engine
/// will copy the contents of [cfg], therefore it's safe to free
/// [cfg] once this function has returned.
///
/// @return Zero on failure, nonzero on success. If the return value
/// is nonzero, a task is running. In such a case, the caller is
/// responsible to eventually dispose of the task using OONITaskFree.
OONITask OONITaskStart(struct OONIMessage *cfg);

/// OONITaskWaitForNextEvent awaits on the [task] event queue until
/// a new event is available or the given [timeout] expires.
///
/// @param task Task handle returned by OONITaskStart.
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
struct OONIMessage *OONITaskWaitForNextEvent(OONITask task, int32_t timeout);

/// OONIMessageFree frees a [msg] returned by OONITaskWaitForNextEvent. You MUST
/// NOT free these messages yourself by calling `free` because the OONI engine MAY
/// be using a different allocator. In the same vein, you MUST NOT use this
/// function to free OONIMessages allocated by the app.
///
/// @param msg OONIMessage previousely returned by OONITaskWaitForNextEvent. If
/// msg is a NULL pointer, this function will just ignore it.
void OONIMessageFree(struct OONIMessage *msg);

/// OONITaskIsDone returns whether the task identified by [taskID] is done. A taks is
/// done when it has finished running _and_ its events queue has been drained.
///
/// @param task Task handle returned by OONITaskStart.
///
/// @return Nonzero if the task exists and either is still running or has some
/// unread events inside its events queue, zero otherwise.
uint8_t OONITaskIsDone(OONITask task);

/// OONITaskInterrupt tells [task] to stop ASAP.
///
/// @param task Task handle returned by OONITaskStart. If task is zero
/// or does not refer to a valid task, this function will just do nothing.
void OONITaskInterrupt(OONITask task);

/// OONITaskFree free the memory associated with [task]. If the task is still running, this
/// function will also interrupt it and drain its events queue.
///
/// @param task Task handle returned by OONITaskStart. If task is zero
/// or does not refer to a valid task, this function will just do nothing.
void OONITaskFree(OONITask task);

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
