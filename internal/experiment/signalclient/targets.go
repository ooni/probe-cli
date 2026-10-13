package signalclient

//
// What the experiment measures: Signal's Signal-hosted endpoints, each with the hello and the
// protocol of the client that uses it.
//

// Method is how a target is exercised after TLS.
type Method string

const (
	// MethodProvisioningH2 is libsignal's RFC 8441 extended CONNECT to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH2 Method = "provisioning_h2"

	// MethodProvisioningH1 is Android OkHttp's HTTP/1.1 WebSocket upgrade to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH1 Method = "provisioning_h1"

	// MethodEnclaveH1 is libsignal's HTTP/1.1 WebSocket upgrade to an enclave, without
	// credentials: 401 (current enclave id) or 404 (other ids) both mean Signal answered.
	MethodEnclaveH1 Method = "enclave_h1"

	// MethodHTTPSGet is one HTTPS GET of "/" (HTTP/2 when negotiated).
	MethodHTTPSGet Method = "https_get"
)

// Target is one connection a real Signal client makes.
type Target struct {
	Name     string
	Hostname string
	Hello    Hello
	Method   Method
	Path     string
}

// enclaveProbePath is a syntactically valid enclave path that names no real enclave, so the
// result never depends on the quarterly enclave ids.
const enclaveProbePath = "/v1/0000000000000000000000000000000000000000000000000000000000000000"

// DefaultTargets are Signal's Signal-hosted endpoints, each with the hello and protocol of the
// client that uses it (signal.md §1; hellos/README.md).
var DefaultTargets = []Target{
	{"chat_libsignal", "grpc.chat.signal.org", HelloLibsignal, MethodProvisioningH2, "/v1/websocket/provisioning/"},
	{"chat_android", "chat.signal.org", HelloAndroidWebSocket, MethodProvisioningH1, "/v1/websocket/provisioning/"},
	{"cdsi", "cdsi.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath + "/discovery"},
	{"svr2", "svr2.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"svrb", "svrb.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"storage", "storage.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn", "cdn.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn2", "cdn2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn3", "cdn3.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2", "updates2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2_desktop", "updates2.signal.org", HelloDesktop, MethodHTTPSGet, "/desktop/latest-linux.yml"},
	{"sfu", "sfu.voip.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
}

// keepalivePaddings are the keepalive body sizes: none, message-sized, attachment-chunk-sized.
var keepalivePaddings = []int{0, 1024, 16384}

// request is what exercise sends on a target's TLS connection.
type request struct {
	method     Method
	hostHeader string // the target's hostname, or a front's Host header
	path       string
	provision  bool // run the provisioning flow after an accepted upgrade (else stop there)
}

// request is the target's own request, with the provisioning flow.
func (t Target) request() request {
	return request{method: t.Method, hostHeader: t.Hostname, path: t.Path, provision: true}
}
