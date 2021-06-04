// +build cgo

package platform

//
// /* Guess the platform in which we are.
//
//     See: <https://sourceforge.net/p/predef/wiki/OperatingSystems/>
//          <http://stackoverflow.com/a/18729350> */
//
//#if defined __ANDROID__
//#  define OONI_PLATFORM "android"
//#elif defined __linux__
//#  define OONI_PLATFORM "linux"
//#elif defined _WIN32
//#  define OONI_PLATFORM "windows"
//#elif defined __APPLE__
//#  include <TargetConditionals.h>
//#  if TARGET_OS_IPHONE
//#    define OONI_PLATFORM "ios"
//#  else
//#    define OONI_PLATFORM "macos"
//#  endif
//#else
//#  define OONI_PLATFORM "unknown"
//#endif
import "C"

func cgoname() string {
	return C.OONI_PLATFORM
}
