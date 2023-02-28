package netem

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// MLabLocateConfigResultLocation is the location of an m-lab server.
type MLabLocateConfigResultLocation struct {
	// City is the city where the server is deployed.
	City string `json:"city"`

	// Country is the server country.
	Country string `json:"country"`
}

// MLabLocateConfigResult is a single result in [MLabLocateConfigResults].
type MLabLocateConfigResult struct {
	// Machine is the name of the machine.
	Machine string `json:"machine"`

	// Location contains the location of the machine.
	Location MLabLocateConfigResultLocation `json:"location"`

	// URLs contains the URLs to use.
	URLs map[string]string `json:"urls"`
}

// MLabLocateConfigResults is the DASH specific config inside [MlabLocateConfig].
type MLabLocateConfigResults struct {
	// Results contains the results to return.
	Results []MLabLocateConfigResult
}

// DefaultMLabLocateDomain is the default domain used by m-lab locate.
const DefaultMLabLocateDomain = "locate.measurementlab.net"

// DefaultMLabLocateDASHDomain is the default domain used for DASH.
const DefaultMLabLocateDASHDomain = "mlab2-mil06.mlab-oti.measurement-lab.org"

// NewMLabLocateConfigDASH creates a new [MLabLocateConfigResults].
func NewMLabLocateConfigDASH() *MLabLocateConfigResults {
	return &MLabLocateConfigResults{
		Results: []MLabLocateConfigResult{{
			Machine: DefaultMLabLocateDASHDomain,
			Location: MLabLocateConfigResultLocation{
				City:    "Milan",
				Country: "IT",
			},
			URLs: map[string]string{
				"https:///negotiate/dash": (&url.URL{
					Scheme: "https",
					Host:   DefaultMLabLocateDASHDomain,
					Path:   "/negotiate/dash",
				}).String(),
			},
		}},
	}
}

// MLabLocateConfig contains config for [MLabLocateServer].
type MLabLocateConfig struct {
	// DASH contains the results to be returned for /v2/nearest/neubot/dash.
	DASH *MLabLocateConfigResults
}

// MLabLocateServer is a minimal m-lab locate v2 server using a [GvisorStack]. The zero
// value is invalid; please, use [NewMLabLocateServer] to instantiate.
type MLabLocateServer struct {
	// closeOnce allows us to call close just once.
	closeOnce sync.Once

	// config contains the config.
	config *MLabLocateConfig

	// http is the underlying HTTP server.
	http *http.Server
}

// NewMLabLocateServer creates a new [MLabLocateServer] instance. This
// function calls [runtimex.PanicOnError] on failure.
func NewMLabLocateServer(
	stack *GvisorStack,
	mitmConfig *TLSMITMConfig,
	ipAddress string,
	config *MLabLocateConfig,
) *MLabLocateServer {
	// start listening and create server
	addr := &net.TCPAddr{
		IP:   net.ParseIP(ipAddress),
		Port: 443,
		Zone: "",
	}
	runtimex.Assert(addr.IP != nil, "NewMLabLocateServer: cannot parse ipAddress")
	listener := runtimex.Try1(stack.ListenTCP("tcp", addr))
	srvr := &MLabLocateServer{
		closeOnce: sync.Once{},
		config:    config,
		http:      nil,
	}

	// create and populate HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/nearest/neubot/dash", srvr.nearestNeubotDASH)

	// listen and serve using TLS
	srvr.http = &http.Server{
		Handler:   mux,
		TLSConfig: mitmConfig.config.TLS(),
	}
	go srvr.http.ServeTLS(listener, "", "") // using httpSrvr.TLSConfig

	// return to the caller
	return srvr
}

// Stop stops the running [MLabLocateServer] instance.
func (s *MLabLocateServer) Stop() {
	s.closeOnce.Do(func() {
		s.http.Close()
	})
}

// nearestNeubotDASH returns information about the nearest neubot/dash server
func (s *MLabLocateServer) nearestNeubotDASH(w http.ResponseWriter, r *http.Request) {
	if s.config.DASH == nil {
		w.WriteHeader(500)
		return
	}
	data := runtimex.Try1(json.Marshal(s.config.DASH))
	w.Write(data)
}
