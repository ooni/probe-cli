package qa

//
// DASH minimal server implementation using netem.GvisorStack
// and adapted from github.com/neubot/dash
//

import (
	"context"
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/google/uuid"
	"github.com/ooni/probe-cli/v3/internal/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// DASHEnvironment is the environment in which we run DASH QA tests. The zero value
// is invalid; please, use [NewDASHEnvironment] to instantiate.
type DASHEnvironment struct {
	// backbone is the [netem.backbone] to which all the servers relevant
	// to perform DASH QA checks have been attached.
	backbone *netem.Backbone

	// cancel stops all the goroutines running in the background.
	cancel context.CancelFunc

	// dashServer is the dash server.
	dashServer *dashServer

	// locateServer is the m-lab locate v2 server.
	locateServer *mlabLocateServer

	// probeIP generates the next probe IP.
	probeIP *probeIP

	// stopOnce allows to call close just once.
	stopOnce sync.Once

	// tlsMITMConfig is the config that allows clients and servers to use PKI
	// with certificate validation even though all certificates are fake.
	tlsMITMConfig *netem.TLSMITMConfig
}

// NewDASHEnvironment creates a new [DASHEnvironment]. This function will start
// goroutines to handle emulated network I/O. To stop all the emulated network
// activity you MUST call the [DASHEnvironment.Stop] method when done. This function
// will call [runtimex.PanicOnError] in case of failure.
func NewDASHEnvironment() *DASHEnvironment {
	// create context for cancelling background operations.
	ctx, cancel := context.WithCancel(context.Background())

	// create configuration for performing TLS MITM
	mitmConfig := netem.NewTLSMITMConfig()

	// create empty getaddrinfo configuration for servers.
	gginfo := &netem.StaticGetaddrinfo{}

	// create a backbone
	backbone := netem.NewBackbone()

	// create the locate v2 server
	locateStack := netem.NewGvisorStack(mlabLocateIPAddress, mitmConfig, gginfo)
	backbone.AddServer(ctx, locateStack, netem.NewLinkFastest)
	locateServer := newMLabLocateServer(locateStack, mitmConfig, mlabLocateIPAddress)

	// create the dash server
	dashStack := netem.NewGvisorStack(dashServerIPAddress, mitmConfig, gginfo)
	backbone.AddServer(ctx, dashStack, netem.NewLinkFastest)
	dashServer := newDASHServer(dashStack, mitmConfig, dashServerIPAddress)

	return &DASHEnvironment{
		backbone:      backbone,
		cancel:        cancel,
		dashServer:    dashServer,
		locateServer:  locateServer,
		probeIP:       &probeIP{},
		stopOnce:      sync.Once{},
		tlsMITMConfig: mitmConfig,
	}
}

// NonCensoredStaticGetaddrinfo returns a non-censored
// [netem.StaticGetaddrinfo] suitable for running the
// DASH experiment successfully.
func (env *DASHEnvironment) NonCensoredStaticGetaddrinfo() *netem.StaticGetaddrinfo {
	gginfo := netem.NewStaticGetaddrinfo()
	gginfo.AddStaticEntry(mlabLocateDomain, &netem.StaticGetaddrinfoEntry{
		Addresses: []string{
			mlabLocateIPAddress,
		},
		CNAME: "",
	})
	gginfo.AddStaticEntry(dashServerDomain, &netem.StaticGetaddrinfoEntry{
		Addresses: []string{
			dashServerIPAddress,
		},
		CNAME: "",
	})
	return gginfo
}

// RunExperiment runs the DASH experiment and returns the resulting
// [model.Measurement] (on success) or an error (on failure).
func (env *DASHEnvironment) RunExperiment(
	gginfo netem.GvisorGetaddrinfo,
	linkFactory netem.LinkFactory,
	dpi netem.LinkDPIEngine,
) (*model.Measurement, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create and attach client stack
	addr := env.probeIP.Next()
	stack := netem.NewGvisorStack(addr, env.tlsMITMConfig, gginfo)
	env.backbone.AddClient(ctx, stack, linkFactory, dpi)

	// create measurer for the dash experiment
	measurer := dash.NewExperimentMeasurer(dash.Config{})

	// create measurement to fill
	measurement := newMeasurement(measurer.ExperimentName(), measurer.ExperimentVersion())

	// create args for Run
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mocks.Session{
			MockLogger: func() model.Logger {
				return log.Log
			},
			MockUserAgent: func() string {
				return "miniooni/" + version.Version
			},
		},
	}

	// measure inside a modified netxlite environment using stack
	var err error
	netxlite.WithCustomTProxy(stack, func() {
		err = measurer.Run(ctx, args)
	})

	// return result to the caller
	if err != nil {
		return nil, err
	}
	return measurement, nil
}

// Stop stops all the goroutines running in the background.
func (env *DASHEnvironment) Stop() {
	env.stopOnce.Do(func() {
		env.dashServer.Stop()
		env.locateServer.Stop()
		env.cancel()
	})
}

// dashServerDomain is the domain used for neubot/dash.
const dashServerDomain = "mlab2-mil06.mlab-oti.measurement-lab.org"

// dashServerIPAddress is the IP address for [dashServerDomain].
const dashServerIPAddress = "162.213.96.86"

// dashServer is a minimal DASH server using a [netem.GvisorStack]. The zero
// value is invalid; please, use [newDASHServer] to instantiate.
type dashServer struct {
	// closeOnce allows us to call close just once.
	closeOnce sync.Once

	// http is the underlying HTTP server.
	http *http.Server
}

// newDASHServer creates a new [dashServer] instance. This function
// calls [runtimex.PanicOnError] on failure.
func newDASHServer(
	stack *netem.GvisorStack,
	mitmConfig *netem.TLSMITMConfig,
	ipAddress string,
) *dashServer {
	// start listening and create server
	parsedIP := net.ParseIP(ipAddress)
	runtimex.Assert(parsedIP != nil, "NewDASHServer: cannot parse ipAddress")
	addr := &net.TCPAddr{
		IP:   parsedIP,
		Port: 443,
		Zone: "",
	}
	listener := runtimex.Try1(stack.ListenTCP("tcp", addr))
	srvr := &dashServer{
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
		TLSConfig: mitmConfig.TLSConfig(),
	}
	go srvr.http.ServeTLS(listener, "", "") // using httpSrvr.TLSConfig

	// return to the caller
	return srvr
}

// Stop stops the running [dashServer] instance.
func (s *dashServer) Stop() {
	s.closeOnce.Do(func() {
		s.http.Close()
	})
}

// negotiate handles the negotiate request
func (s *dashServer) negotiate(w http.ResponseWriter, r *http.Request) {
	// Unlike the official DASH server, here we just make sure we
	// tell the client it has been unchoked
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	resp := &model.DASHNegotiateResponse{
		Authorization: runtimex.Try1(uuid.NewRandom()).String(),
		QueuePos:      0,
		RealAddress:   addr,
		Unchoked:      1,
	}
	data := runtimex.Try1(json.Marshal(resp))
	w.Write(data)
}

// collect handles the collect request
func (s *dashServer) collect(w http.ResponseWriter, r *http.Request) {
	// Unlike the official DASH server, here we just return an empty object to
	// the client (which ignores this response message anyway).
	w.Write([]byte(`{}`))
}

// download handles the download request
func (s *dashServer) download(w http.ResponseWriter, r *http.Request) {
	// Code adapted from github.com/neubot/dash

	// Get the size parameter from the URL
	siz := strings.Replace(r.URL.Path, "/dash/download", "", -1)
	siz = strings.TrimPrefix(siz, "/")
	if siz == "" {
		siz = model.DASHMinSizeString
	}
	count, err := strconv.Atoi(siz)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// Make sure the parameter is within the acceptable range
	if count < model.DASHMinSize {
		count = model.DASHMinSize
	}
	if count > model.DASHMaxSize {
		count = model.DASHMaxSize
	}

	// Create a random message of the desired size
	data := make([]byte, count)
	if _, err := rand.Read(data); err != nil {
		w.WriteHeader(400)
		return
	}

	// Send response to the client
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
