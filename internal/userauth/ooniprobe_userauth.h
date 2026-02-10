#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct ClientResponse {
  char *json;
  char *error;
} ClientResponse;

/**
 * Free memory allocated by ClientResponse
 *
 * # Safety
 * This function must be called exactly once for each ClientResponse
 * returned by other FFI functions to avoid memory leaks.
 */
void client_response_free(struct ClientResponse response);

/**
 * Perform HTTP GET request
 *
 * # Safety
 * - `url` must be a valid null-terminated C string
 * - Caller must call `client_response_free` on the returned value
 */
struct ClientResponse client_get(const char *url);

/**
 * Perform HTTP POST request
 *
 * # Safety
 * - `url` and `payload` must be valid null-terminated C strings
 * - Caller must call `client_response_free` on the returned value
 */
struct ClientResponse client_post(const char *url, const char *payload);

/**
 * Register a user and obtain a credential
 *
 * # Safety
 * - All parameters must be valid null-terminated C strings
 * - Caller must call `client_response_free` on the returned value
 */
struct ClientResponse userauth_register(const char *url,
                                        const char *public_params,
                                        const char *manifest_version);

/**
 * Submit user credentials with measurement data
 *
 * # Safety
 * - All parameters must be valid null-terminated C strings
 * - `credential_b64` must be a valid base64-encoded credential
 * - `public_params` must be valid base64 public parameters
 * - Caller must call `client_response_free` on the returned value
 */
struct ClientResponse userauth_submit(const char *url,
                                      const char *credential_b64,
                                      const char *public_params,
                                      const char *probe_cc,
                                      const char *probe_asn,
                                      const char *manifest_version);
