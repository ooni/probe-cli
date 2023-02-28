package netem

//
// DASH minimal server implementation adapted
// from github.com/neubot/dash
//

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DASHServer is a minimal DASH server using a [GvisorStack]. The zero
// value is invalid; please, use [NewDASHServer] to instantiate.
type DASHServer struct {
	// closeOnce allows us to call close just once.
	closeOnce sync.Once

	// http is the underlying HTTP server.
	http *http.Server
}

// NewDASHServer creates a new [DASHServer] instance. This function
// calls [runtimex.PanicOnError] on failure.
func NewDASHServer(
	stack *GvisorStack,
	mitmConfig *TLSMITMConfig,
	ipAddress string,
) *DASHServer {
	// start listening and create server
	addr := &net.TCPAddr{
		IP:   net.ParseIP(ipAddress),
		Port: 443,
		Zone: "",
	}
	runtimex.Assert(addr.IP != nil, "NewDASHServer: cannot parse ipAddress")
	listener := runtimex.Try1(stack.ListenTCP("tcp", addr))
	srvr := &DASHServer{
		closeOnce: sync.Once{},
		http:      nil,
	}

	// create and populate HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/negotiate/dash", srvr.negotiate)
	mux.HandleFunc("/collect/dash", srvr.collect)
	mux.HandleFunc("/dash/download/", srvr.download)

	// listen and serve using TLS
	srvr.http = &http.Server{
		Handler:   mux,
		TLSConfig: mitmConfig.config.TLS(),
	}
	go srvr.http.ServeTLS(listener, "", "") // using httpSrvr.TLSConfig

	// return to the caller
	return srvr
}

// Stop stops the running [DASHServer] instance.
func (s *DASHServer) Stop() {
	s.closeOnce.Do(func() {
		s.http.Close()
	})
}

// dashNegotiateResponse contains the response of negotiation
type dashNegotiateResponse struct {
	Authorization string `json:"authorization"`
	QueuePos      int64  `json:"queue_pos"`
	RealAddress   string `json:"real_address"`
	Unchoked      int    `json:"unchoked"`
}

// negotiate handles the negotiate request
func (s *DASHServer) negotiate(w http.ResponseWriter, r *http.Request) {
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	resp := &dashNegotiateResponse{
		Authorization: runtimex.Try1(uuid.NewRandom()).String(),
		QueuePos:      0,
		RealAddress:   addr,
		Unchoked:      1,
	}
	data := runtimex.Try1(json.Marshal(resp))
	w.Write(data)
}

// collect handles the collect request
func (s *DASHServer) collect(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{}`))
}

// dashMinSize is the minimum segment size that this server can return.
//
// The client requests two second chunks. The minimum emulated streaming
// speed is the minimum streaming speed (in kbit/s) multiplied by 1000
// to obtain bit/s, divided by 8 to obtain bytes/s and multiplied by the
// two seconds to obtain the minimum segment size.
const dashMinSize = 100 * 1000 / 8 * 2

// dashMaxSize is the maximum segment size that this server can return. See
// the docs of MinSize for more information on how it is computed.
const dashMaxSize = 30000 * 1000 / 8 * 2

// dashMinSizeString is [dashMinSize] as a string.
var dashMinSizeString = fmt.Sprintf("%d", dashMinSize)

// download handles the download request
func (s *DASHServer) download(w http.ResponseWriter, r *http.Request) {
	siz := strings.Replace(r.URL.Path, "/dash/download", "", -1)
	siz = strings.TrimPrefix(siz, "/")
	if siz == "" {
		siz = dashMinSizeString
	}
	count, err := strconv.Atoi(siz)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	if count < dashMinSize {
		count = dashMinSize
	}
	if count > dashMaxSize {
		count = dashMaxSize
	}
	data := make([]byte, count)
	if _, err := rand.Read(data); err != nil {
		w.WriteHeader(400)
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
