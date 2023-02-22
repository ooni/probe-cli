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
void OONIENgineFreeMemory(void *ptr);

/// NewSession creates a new session with the given config.
///
/// @param config a JSON string representing the configuration for the session. 
///
/// @return Zero on failure, nonzero on success. If the return value
/// is nonzero, a task is running. In such a case, the caller is
/// responsible to eventually dispose of the task using OONIEngineFree.
OONITask NewSession(char *config);

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
