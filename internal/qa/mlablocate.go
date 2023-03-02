package qa

//
// Minimal m-lab locate v2 API implementation
// using netem.GvisorStack
//

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// mlabLocateDomain is the domain used by m-lab locate itself.
const mlabLocateDomain = "locate.measurementlab.net"

// mlabLocateIPAddress is the IP address used by m-lab locate itself.
const mlabLocateIPAddress = "142.250.185.115"

// mlabLocateResultsDASH is the result returned for /v2/neareast/neubot/dash queries
var mlabLocateResultsDASH = &model.MLabLocateResults{
	Results: []model.MLabLocateSingleResult{{
		Machine: dashServerDomain,
		Location: model.MLabLocateServerLocation{
			City:    "Milan",
			Country: "IT",
		},
		URLs: map[string]string{
			"https:///negotiate/dash": (&url.URL{
				Scheme: "https",
				Host:   dashServerDomain,
				Path:   "/negotiate/dash",
			}).String(),
		},
	}},
}

// mlabLocateServer is a minimal m-lab locate v2 server using a [netem.GvisorStack] as
// the underlying network stack. The zero value is invalid; please, use
// [newMLabLocateServer] to instantiate.
type mlabLocateServer struct {
	// closeOnce allows us to call close just once.
	closeOnce sync.Once

	// http is the underlying HTTP server.
	http *http.Server
}

// newMLabLocateServer creates a new [mlabLocateServer] instance. This
// function calls [runtimex.PanicOnError] on failure.
func newMLabLocateServer(
	stack *netem.UNetStack,
	mitmConfig *netem.TLSMITMConfig,
	ipAddress string,
) *mlabLocateServer {
	// start listening and create server
	parsedIP := net.ParseIP(ipAddress)
	runtimex.Assert(parsedIP != nil, "NewMLabLocateServer: cannot parse ipAddress")
	addr := &net.TCPAddr{
		IP:   parsedIP,
		Port: 443,
		Zone: "",
	}
	listener := runtimex.Try1(stack.ListenTCP("tcp", addr))
	srvr := &mlabLocateServer{
		closeOnce: sync.Once{},
		http:      nil,
	}

	// create and populate HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/nearest/neubot/dash", srvr.nearestNeubotDASH)

	// listen and serve using TLS
	srvr.http = &http.Server{
		Handler:   mux,
		TLSConfig: mitmConfig.TLSConfig(),
	}
	go srvr.http.ServeTLS(listener, "", "") // using httpSrvr.TLSConfig

	// return to the caller
	return srvr
}

// Stop stops the running [mlabLocateServer] instance immediately.
func (s *mlabLocateServer) Stop() {
	s.closeOnce.Do(func() {
		s.http.Close()
	})
}

// nearestNeubotDASH returns information about the nearest neubot/dash server
func (s *mlabLocateServer) nearestNeubotDASH(w http.ResponseWriter, r *http.Request) {
	data := runtimex.Try1(json.Marshal(mlabLocateResultsDASH))
	w.Write(data)
}
