// SPDX-License-Identifier: GPL-3.0-or-later

#ifndef OONI_ENGINE_H
#define OONI_ENGINE_H

///
/// @file ooni/engine.h
///
/// C API for using the OONI engine.
///

/// OONIEvent is an event emitted by the OONI engine.
struct OONIEvent {
	/// Name is the name of the event.
	char *Name;

	/// Value is the JSON-serialized event value.
	char *Value;
};

#ifdef __cplusplus
extern "C" {
#endif

/// Starts a new task inside the OONI engine.
///
/// @param abiVersion The ABI version used by the app.
///
/// @param taskName The name of the task to start.
///
/// @param taskArguments JSON-serialized arguments for the task.
///
/// @return A negative value on failure. In such a case, the OONI engine
/// will print a diagnostic message on the standard error.
///
/// @return A zero-or-positive unique task identifier on success. In such a
/// case, you own the task and must call OONITaskFree when done using it.
int OONITaskStart(char *abiVersion, char *taskName, char *taskArguments);

/// Blocks waiting for [taskID] to emit an event or for [timeout] to expire.
///
/// @param taskID Unique task identifier returned by OONITaskStart.
///
/// @param timeout Maximum number of milliseconds to wait for the next event
/// to become available. Use a negative value to wait for the next event without
/// any timeout (not recommended in general).
///
/// @return A NULL value if the task does not exist, the timeout expires,
/// or an internal error occurs. Otherwise, the call succeded and you
/// are given ownership of an OONIEvent containing the next task-emitted event.
struct OONIEvent *OONITaskWaitForNextEvent(int taskID, int timeout);

/// Frees an [event] previously returned by OONITaskWaitForNextEvent.
void OONIEventFree(struct OONIEvent *event);

/// Returns whether the task identified by [taskID] is done. A taks is done
/// when it has finished running and its events queue has been drained.
///
/// @param taskID Unique task identifier returned by OONITaskStart.
///
/// @return Zero if the task exists and either is still running or has some
/// unread events inside its events queue, nonzero otherwise.
int OONITaskIsDone(int taskID);

/// Notifies the task identified by [taskID] to stop ASAP.
///
/// @param taskID Unique task identifier returned by OONITaskStart.
void OONITaskInterrupt(int taskID);

/// Frees the memory associated with [taskID]. If the task is still running, this
/// function will also interrupt it and drain its events queue.
///
/// @param taskID Unique task identifier returned by OONITaskStart.
void OONITaskFree(int taskID);

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
