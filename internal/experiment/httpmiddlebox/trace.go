package httpmiddlebox

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CompleteTrace records the result of the network trace
// using a control SNI and a target SNI
type CompleteTrace struct {
	Address       string               `json:"address"`
	TCPTraceroute *IterativeTraceroute `json:"tcp_traceroute"`
	ControlTrace  *IterativeTrace      `json:"control_trace"`
	TargetTrace   *IterativeTrace      `json:"target_trace"`
}

// Trace is an iterative trace for the corresponding servername and address
type IterativeTrace struct {
	SNI        string       `json:"server_name"`
	Iterations []*Iteration `json:"iterations"`

	mu sync.Mutex
}

// Iteration is a single network iteration with variable TTL
type Iteration struct {
	TTL               int                              `json:"ttl"`
	HTTPRequestResult *model.ArchivalHTTPRequestResult `json:"http_request_result"`
}

// IterativeTraceroute is an iterative traceroute towards the address
type IterativeTraceroute struct {
	SNI        string           `json:"server_name"`
	Iterations []*ICMPIteration `json:"iterations"`

	mu sync.Mutex
}

// ICMPIteration is a single ICMP error message associated with a TTL-limited
// probe when used for traceroute
type ICMPIteration struct {
	TTL       int                             `json:"ttl"`
	ICMPError *model.ArchivalICMPErrorMessage `json:"icmp_error"`
}

// matches a bracketed IPv6 address (with an optional port) OR an IPv4 address (with an optional port) OR an unbracketed IPv6 address with NO port
var ipRegex = regexp.MustCompile(
	`\[[0-9a-fA-F:.]+\](?::\d+)?|\b(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?|[0-9a-fA-F]*:[0-9a-fA-F:]+`,
)

// scrubs IPv4 and IPv6 addresses from error messages
func scrubIPsFromError(s string) string {
	return ipRegex.ReplaceAllStringFunc(s, func(candidate string) string {
		ip := strings.Trim(candidate, "[]")

		host, _, err := net.SplitHostPort(candidate)
		if err == nil {
			ip = host
		}

		if net.ParseIP(ip) != nil {
			return "[scrubbed]"
		}

		return candidate
	})
}

// errorChain unwraps and iterates through the error list and records
// the type and error at each step
func errorChain(err error) []model.FailureChainData {
	if err == nil {
		return nil
	}

	var chain []model.FailureChainData

	for err != nil {
		chain = append(chain, model.FailureChainData{
			Type:  fmt.Sprintf("%T", err),
			Error: scrubIPsFromError(err.Error()),
		})

		err = errors.Unwrap(err)
	}

	return chain
}

// NewIterationFromArchivalHTTPRequestResult returns a new iteration from a model.ArchivalHTTPRequestResult
func newIterationFromArchivalHTTPRequestResult(ttl int, err error, soErr error, httpRequestResult *model.ArchivalHTTPRequestResult, sni string) *Iteration {
	fmt.Println(err)
	if err != nil {
		httpRequestResult.Failure = tracex.NewFailure(err)
		httpRequestResult.FailureChain = errorChain(err)

		return &Iteration{
			TTL:               ttl,
			HTTPRequestResult: httpRequestResult,
		}
	}
	return &Iteration{
		TTL:               ttl,
		HTTPRequestResult: httpRequestResult,
	}
}

// addIterations adds iterations to the trace
func (t *IterativeTrace) addIterations(ev ...*Iteration) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev...)
	t.mu.Unlock()
}

// addIterationsTraceroute adds iterations to the traceroute
func (t *IterativeTraceroute) addIterationsTraceroute(ev ...*ICMPIteration) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev...)
	t.mu.Unlock()
}
