// SPDX-License-Identifier: GPL-3.0-or-later

#ifndef OONI_ENGINE_H
#define OONI_ENGINE_H

///
/// C API for using the OONI engine.
///

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

#ifdef __cplusplus
}
#endif
#endif /* OONI_ENGINE_H */
