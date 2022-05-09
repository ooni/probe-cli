// Package platform allows you to obtain the platform name. We use this
// information to annotate measurements.
package platform

import "runtime"

// Name returns the platform name. The returned value is one of:
//
// 1. "android"
//
// 2. "ios"
//
// 3. "linux"
//
// 5. "macos"
//
// 4. "windows"
//
// 5. "freebsd"
//
// 6. "openbsd"
//
// 7. "unknown"
//
// You should use this name to annotate measurements.
func Name() string {
	return name(runtime.GOOS)
}

// name is a utility function for implementing Name.
func name(goos string) string {
	// Note: since go1.16 we have the ios port, so the ambiguity
	// between ios and darwin is now gone.
	//
	// See https://golang.org/doc/go1.16#darwin
	switch goos {
	case "android", "freebsd", "openbsd", "ios", "linux", "windows":
		return goos
	case "darwin":
		return "macos"
	}
	return "unknown"
}
