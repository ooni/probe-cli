// Package platform returns the platform name. The name returned here
// is compatible with the names returned by Measurement Kit.
package platform

import "runtime"

// Name returns the platform name. The returned value is one of:
//
// 1. "android"
// 2. "ios"
// 3. "linux"
// 5. "macos"
// 4. "windows"
// 5. "unknown"
//
// The android, ios, linux, macos, windows, and unknown strings are
// also returned by Measurement Kit. As a known bug, the detection of
// darwin-based systems relies on the architecture, when CGO support
// has been disabled. In such case, the code will return "ios" when
// using arm{,64} and "macos" when using x86{,_64}.
func Name() string {
	if name := cgoname(); name != "unknown" {
		return name
	}
	return puregoname(runtime.GOOS, runtime.GOARCH)
}

func puregoname(goos, goarch string) string {
	switch goos {
	case "android", "linux", "windows":
		return goos
	case "darwin":
		return detectDarwin(goarch)
	}
	return "unknown"
}

func detectDarwin(goarch string) string {
	switch goarch {
	case "386", "amd64":
		return "macos"
	case "arm", "arm64":
		return "ios"
	}
	return "unknown"
}
