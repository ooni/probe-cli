// SPDX-License-Identifier: GPL-3.0-or-later

#ifndef OONI_ENGINE_H
#define OONI_ENGINE_H

///
/// C API for using the OONI engine.
///

#include <stdint.h>

/// OONITask is an asynchronous thread of execution managed by the OONI
/// engine that performs a background operation and emits interim outpus 
/// like logs and progress and results of the operation with meaningful
/// events such as, for example, the results of measurements.
typedef intptr_t OONITask;

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

/// OONIEngineCall starts a new OONITask using the given @p req.
///
/// @param req A JSON string, owned by the caller, that contains 
/// the configuration for the task to start.  
///
/// @return Zero on failure, nonzero on success. If the return value
/// is nonzero, a task is running. In such a case, the caller is
/// responsible to eventually dispose of the task using OONIEngineFreeTask.
OONITask OONIEngineCall(char *req);

/// OONIEngineWaitForNextEvent awaits on the @p task event queue until
/// a new event is available or the given @p timeout expires.
///
/// @param task Task handle returned by OONIEngineCall.
///
/// @param timeout Timeout in milliseconds. If the timeout is zero
/// or negative, this function would potentially block forever.
///
/// @return A NULL pointer on failure, non-NULL JSON string otherwise. 
/// If the return value is non-NULL, the caller takes ownership of the 
/// char pointer and MUST free it using OONIEngineFreeMemory when done 
/// using it.
///
/// This function will return a NULL pointer:
///
/// 1. when the timeout expires;
///
/// 2. if @p task is done;
///
/// 3. if @p task is zero or does not refer to a valid task;
///
/// 4. if we cannot JSON serialize the message;
///
/// 5. possibly because of other unknown internal errors.
///
/// In short, you cannot reliably determine whether a task is done by
/// checking whether this function has returned an empty string.
char *OONIEngineWaitForNextEvent(OONITask task, int32_t timeout);

/// OONIEngineTaskGetResult awaits on the result queue until the final 
/// result is available.
///
/// @param task Task handle returned by OONIEngineCall.
///
/// @return A NULL pointer on failure, non-NULL JSON string otherwise. 
/// If the return value is non-NULL, the caller takes ownership of the 
/// char pointer and MUST free it using OONIEngineFreeMemory when done 
/// using it.
///
/// This function will return a NULL pointer:
///
/// 1. if @p task is zero or does not refer to a valid task;
///
/// 2. if we cannot JSON serialize the message;
///
/// 3. possibly because of other unknown internal errors.
///
/// In short, you cannot reliably determine whether a task is done by
/// checking whether this function has returned an empty string.
char *OONIEngineTaskGetResult(OONITask task);

/// OONIEngineTaskIsDone returns whether @p task is done. A task is
/// done when it has finished running _and_ its events queue has been drained.
///
/// @param task Task handle returned by OONIEngineCall.
///
/// @return Nonzero if the task exists and either is still running or has some
/// unread events inside its events queue, zero otherwise.
uint8_t OONIEngineTaskIsDone(OONITask task);

/// OONIEngineInterruptTask tells @p task to stop ASAP.
///
/// @param task Task handle returned by OONIEngineCall. If task is zero
/// or does not refer to a valid task, this function will just do nothing.
void OONIEngineInterruptTask(OONITask task);

/// OONIEngineFreeTask frees the memory associated with @p task. If the task is 
/// still running, this function will also interrupt it.
///
/// @param task Task handle returned by OONIEngineCall. If task is zero
/// or does not refer to a valid task, this function will just do nothing.
void OONIEngineFreeTask(OONITask task);

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
