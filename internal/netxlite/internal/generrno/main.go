package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/iancoleman/strcase"
	"golang.org/x/sys/execabs"
)

// ErrorSpec specifies the error we care about.
type ErrorSpec struct {
	// errno is the error name as an errno value (e.g., ECONNREFUSED).
	errno string

	// failure is the error name according to OONI (e.g., FailureConnectionRefused).
	failure string

	// system specifies for which system this error is valid. If
	// this value is empty then the spec is valid for all systems.
	system string
}

// IsForSystem returns true when the spec's system matches the
// given system or when the spec's system is "".
func (es *ErrorSpec) IsForSystem(system string) bool {
	return es.system == system || es.system == ""
}

// AsErrnoName returns the name of the corresponding errno, if this
// is a system error, or panics otherwise.
func (es *ErrorSpec) AsErrnoName(system string) string {
	if !es.IsSystemError() {
		panic("not a system error")
	}
	s := es.errno
	if system == "windows" {
		s = "WSA" + s
	}
	return s
}

// AsCanonicalErrnoName attempts to canonicalize the errno name
// using the following algorithm:
//
// - if the error is present on all systems, use the unix name;
//
// - otherwise, use the system-name name for the error.
//
// So, for example, we will get:
//
// - EWOULDBLOCK because it's present on both Unix and Windows;
//
// - WSANO_DATA because it's Windows only.
func (es *ErrorSpec) AsCanonicalErrnoName() string {
	if !es.IsSystemError() {
		panic("not a system error")
	}
	switch es.system {
	case "windows":
		return es.AsErrnoName(es.system)
	default:
		return es.errno
	}
}

// AsFailureVar returns the name of the failure var.
func (es *ErrorSpec) AsFailureVar() string {
	return "Failure" + strcase.ToCamel(es.failure)
}

// AsFailureString returns the OONI failure string.
func (es *ErrorSpec) AsFailureString() string {
	return strcase.ToSnake(es.failure)
}

// NewSystemError constructs a new ErrorSpec representing a system
// error, i.e., an error returned by a system call.
func NewSystemError(errno, failure string) *ErrorSpec {
	return &ErrorSpec{errno: errno, failure: failure, system: ""}
}

// NewWindowsError constructs a new ErrorSpec representing a
// Windows-only system error, i.e., an error returned by a system call.
func NewWindowsError(errno, failure string) *ErrorSpec {
	return &ErrorSpec{errno: errno, failure: failure, system: "windows"}
}

// NewLibraryError constructs a new ErrorSpec representing a library
// error, i.e., an error returned by the Go standard library or by other
// dependecies written typicall in Go (e.g., quic-go).
func NewLibraryError(failure string) *ErrorSpec {
	return &ErrorSpec{failure: failure}
}

// IsSystemError returns whether this ErrorSpec describes a system
// error, i.e., an error returned by a syscall.
func (es *ErrorSpec) IsSystemError() bool {
	return es.errno != ""
}

// Specs contains all the error specs.
var Specs = []*ErrorSpec{
	NewSystemError("ECONNREFUSED", "connection_refused"),
	NewSystemError("ECONNRESET", "connection_reset"),
	NewSystemError("EHOSTUNREACH", "host_unreachable"),
	NewSystemError("ETIMEDOUT", "timed_out"),
	NewSystemError("EAFNOSUPPORT", "address_family_not_supported"),
	NewSystemError("EADDRINUSE", "address_in_use"),
	NewSystemError("EADDRNOTAVAIL", "address_not_available"),
	NewSystemError("EISCONN", "already_connected"),
	NewSystemError("EFAULT", "bad_address"),
	NewSystemError("EBADF", "bad_file_descriptor"),
	NewSystemError("ECONNABORTED", "connection_aborted"),
	NewSystemError("EALREADY", "connection_already_in_progress"),
	NewSystemError("EDESTADDRREQ", "destination_address_required"),
	NewSystemError("EINTR", "interrupted"),
	NewSystemError("EINVAL", "invalid_argument"),
	NewSystemError("EMSGSIZE", "message_size"),
	NewSystemError("ENETDOWN", "network_down"),
	NewSystemError("ENETRESET", "network_reset"),
	NewSystemError("ENETUNREACH", "network_unreachable"),
	NewSystemError("ENOBUFS", "no_buffer_space"),
	NewSystemError("ENOPROTOOPT", "no_protocol_option"),
	NewSystemError("ENOTSOCK", "not_a_socket"),
	NewSystemError("ENOTCONN", "not_connected"),
	NewSystemError("EWOULDBLOCK", "operation_would_block"),
	NewSystemError("EACCES", "permission_denied"),
	NewSystemError("EPROTONOSUPPORT", "protocol_not_supported"),
	NewSystemError("EPROTOTYPE", "wrong_protocol_type"),

	// Windows-only system errors.
	//
	// Why do we have these extra errors here? Because on Windows
	// GetAddrInfoW is a system call while it's a library call
	// on Unix. Because of that, the Go stdlib treats Windows and
	// Unix differently and allows more syscall errors to slip
	// through when we're performing DNS resolutions.
	//
	// Because MK handled _some_ getaddrinfo return codes, I've
	// marked names compatible with MK using [*].
	//
	// Implementation note: we need to specify acronyms we
	// want to be upper case in uppercase here. For example,
	// we must write "DNS" rather than writing "dns".
	NewWindowsError("NO_DATA", "DNS_no_answer"),                   // [ ] WSANO_DATA
	NewWindowsError("NO_RECOVERY", "DNS_non_recoverable_failure"), // [*] WSANO_RECOVERY
	NewWindowsError("TRY_AGAIN", "DNS_temporary_failure"),         // [*] WSATRY_AGAIN
	NewWindowsError("HOST_NOT_FOUND", "DNS_NXDOMAIN_error"),       // [*] WSAHOST_NOT_FOUND

	// Implementation note: we need to specify acronyms we
	// want to be upper case in uppercase here. For example,
	// we must write "DNS" rather than writing "dns".
	NewLibraryError("DNS_bogon_error"),
	NewLibraryError("DNS_NXDOMAIN_error"),
	NewLibraryError("DNS_refused_error"),
	NewLibraryError("DNS_server_misbehaving"),
	NewLibraryError("DNS_no_answer"),
	NewLibraryError("DNS_servfail_error"),
	NewLibraryError("DNS_reply_with_wrong_query_ID"),
	NewLibraryError("EOF_error"),
	NewLibraryError("generic_timeout_error"),
	NewLibraryError("QUIC_incompatible_version"),
	NewLibraryError("SSL_failed_handshake"),
	NewLibraryError("SSL_invalid_hostname"),
	NewLibraryError("SSL_unknown_authority"),
	NewLibraryError("SSL_invalid_certificate"),
	NewLibraryError("JSON_parse_error"),
	NewLibraryError("connection_already_closed"),

	// QUIRKS: the following errors exist to clearly flag strange
	// underlying behavior implemented by platforms.
	NewLibraryError("Android_DNS_cache_no_data"),
}

// mapSystemToLibrary maps the operating system name to the name
// of the related golang.org/x/sys/$name library.
func mapSystemToLibrary(system string) string {
	switch system {
	case "darwin", "freebsd", "openbsd", "linux":
		return "unix"
	case "windows":
		return "windows"
	default:
		panic(fmt.Sprintf("unsupported system: %s", system))
	}
}

func fileCreate(filename string) *os.File {
	filep, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	return filep
}

func fileWrite(filep *os.File, content string) {
	if _, err := filep.WriteString(content); err != nil {
		log.Fatal(err)
	}
}

func fileClose(filep *os.File) {
	if err := filep.Close(); err != nil {
		log.Fatal(err)
	}
}

func filePrintf(filep *os.File, format string, v ...interface{}) {
	fileWrite(filep, fmt.Sprintf(format, v...))
}

func gofmt(filename string) {
	cmd := execabs.Command("go", "fmt", filename)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func writeSystemSpecificFile(system string) {
	filename := "errno_" + system + ".go"
	filep := fileCreate(filename)
	library := mapSystemToLibrary(system)
	fileWrite(filep, "// Code generated by go generate; DO NOT EDIT.\n")
	filePrintf(filep, "// Generated: %+v\n\n", time.Now())
	fileWrite(filep, "package netxlite\n\n")
	fileWrite(filep, "import (\n")
	fileWrite(filep, "\t\"errors\"\n")
	fileWrite(filep, "\t\"syscall\"\n")
	fileWrite(filep, "\n")
	filePrintf(filep, "\t\"golang.org/x/sys/%s\"\n", library)
	fileWrite(filep, ")\n\n")

	fileWrite(filep, "// This enumeration provides a canonical name for\n")
	fileWrite(filep, "// every system-call error we support. Note: this list\n")
	fileWrite(filep, "// is system dependent. You're currently looking at\n")
	filePrintf(filep, "// the list of errors for %s.\n", system)
	fileWrite(filep, "const (\n")
	for _, spec := range Specs {
		if !spec.IsSystemError() || !spec.IsForSystem(system) {
			continue
		}
		filePrintf(filep, "\t%s = %s.%s\n",
			spec.AsCanonicalErrnoName(), library, spec.AsErrnoName(system))
	}
	fileWrite(filep, ")\n\n")

	fileWrite(filep, "// classifySyscallError converts a syscall error to the\n")
	fileWrite(filep, "// proper OONI error. Returns the OONI error string\n")
	fileWrite(filep, "// on success, an empty string otherwise.\n")
	fileWrite(filep, "func classifySyscallError(err error) string {\n")
	fileWrite(filep, "\tvar errno syscall.Errno\n")
	fileWrite(filep, "\tif !errors.As(err, &errno) {\n")
	fileWrite(filep, "\t\treturn \"\"\n")
	fileWrite(filep, "\t}\n")
	fileWrite(filep, "\tswitch errno {\n")
	for _, spec := range Specs {
		if !spec.IsSystemError() || !spec.IsForSystem(library) {
			continue
		}
		filePrintf(filep, "\tcase %s.%s:\n", library, spec.AsErrnoName(system))
		filePrintf(filep, "\t\treturn %s\n", spec.AsFailureVar())
	}
	fileWrite(filep, "\t}\n")
	fileWrite(filep, "\treturn \"\"\n")
	fileWrite(filep, "}\n\n")

	fileClose(filep)
	gofmt(filename)
}

func writeGenericFile() {
	filename := "errno.go"
	filep := fileCreate(filename)
	fileWrite(filep, "// Code generated by go generate; DO NOT EDIT.\n")
	filePrintf(filep, "// Generated: %+v\n\n", time.Now())
	fileWrite(filep, "package netxlite\n\n")
	fileWrite(filep, "//go:generate go run ./internal/generrno/\n\n")

	fileWrite(filep, "// This enumeration lists the failures defined at\n")
	fileWrite(filep, "// https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md.\n")
	fileWrite(filep, "// Please, refer to that document for more information.\n")
	fileWrite(filep, "const (\n")
	names := make(map[string]string)
	for _, spec := range Specs {
		names[spec.AsFailureVar()] = spec.AsFailureString()
	}
	var nameskeys []string
	for key := range names {
		nameskeys = append(nameskeys, key)
	}
	sort.Strings(nameskeys)
	for _, key := range nameskeys {
		filePrintf(filep, "\t%s = \"%s\"\n", key, names[key])
	}
	fileWrite(filep, ")\n\n")

	fileWrite(filep, "// failureMap lists all failures so we can match them\n")
	fileWrite(filep, "// when they are wrapped by quic.TransportError.\n")
	fileWrite(filep, "var failuresMap = map[string]string{\n")
	failures := make(map[string]string)
	for _, spec := range Specs {
		failures[spec.AsFailureString()] = spec.AsFailureString()
	}
	var failureskey []string
	for key := range failures {
		failureskey = append(failureskey, key)
	}
	sort.Strings(failureskey)
	for _, key := range failureskey {
		filePrintf(filep, "\t\"%s\": \"%s\",\n", key, failures[key])
	}
	fileWrite(filep, "}\n\n")

	fileClose(filep)
	gofmt(filename)
}

func writeSystemSpecificTestFile(system string) {
	filename := fmt.Sprintf("errno_%s_test.go", system)
	filep := fileCreate(filename)
	library := mapSystemToLibrary(system)

	fileWrite(filep, "// Code generated by go generate; DO NOT EDIT.\n")
	filePrintf(filep, "// Generated: %+v\n\n", time.Now())
	fileWrite(filep, "package netxlite\n\n")
	fileWrite(filep, "import (\n")
	fileWrite(filep, "\t\"io\"\n")
	fileWrite(filep, "\t\"syscall\"\n")
	fileWrite(filep, "\t\"testing\"\n")
	fileWrite(filep, "\n")
	filePrintf(filep, "\t\"golang.org/x/sys/%s\"\n", library)
	fileWrite(filep, ")\n\n")

	fileWrite(filep, "func TestClassifySyscallError(t *testing.T) {\n")
	fileWrite(filep, "\tt.Run(\"for a non-syscall error\", func (t *testing.T) {\n")
	fileWrite(filep, "\t\tif v := classifySyscallError(io.EOF); v != \"\" {\n")
	fileWrite(filep, "\t\t\tt.Fatalf(\"expected empty string, got '%s'\", v)\n")
	fileWrite(filep, "\t\t}\n")
	fileWrite(filep, "\t})\n\n")

	for _, spec := range Specs {
		if !spec.IsSystemError() || !spec.IsForSystem(library) {
			continue
		}
		filePrintf(filep, "\tt.Run(\"for %s\", func (t *testing.T) {\n",
			spec.AsErrnoName(system))
		filePrintf(filep, "\t\tif v := classifySyscallError(%s.%s); v != %s {\n",
			library, spec.AsErrnoName(system), spec.AsFailureVar())
		filePrintf(filep, "\t\t\tt.Fatalf(\"expected '%%s', got '%%s'\", %s, v)\n",
			spec.AsFailureVar())
		fileWrite(filep, "\t\t}\n")
		fileWrite(filep, "\t})\n\n")
	}

	fileWrite(filep, "\tt.Run(\"for the zero errno value\", func (t *testing.T) {\n")
	fileWrite(filep, "\t\tif v := classifySyscallError(syscall.Errno(0)); v != \"\" {\n")
	fileWrite(filep, "\t\t\tt.Fatalf(\"expected empty string, got '%s'\", v)\n")
	fileWrite(filep, "\t\t}\n")
	fileWrite(filep, "\t})\n")
	fileWrite(filep, "}\n")

	fileClose(filep)
	gofmt(filename)
}

// SupportedSystems contains the list of supported systems.
var SupportedSystems = []string{
	"darwin",
	"freebsd",
	"openbsd",
	"linux",
	"windows",
}

func main() {
	for _, system := range SupportedSystems {
		writeSystemSpecificFile(system)
		writeSystemSpecificTestFile(system)
	}
	writeGenericFile()
}
