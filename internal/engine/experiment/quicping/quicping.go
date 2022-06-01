// Package quicping implements the quicping network experiment. This
// implements, in particular, v0.1.0 of the spec.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-031-quicping.md.
package quicping

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"time"

	_ "crypto/sha256"

	"github.com/ooni/probe-cli/v3/internal/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// A connectionID in QUIC
type connectionID []byte

const (
	maxConnectionIDLen        = 18
	minConnectionIDLenInitial = 8
	defaultConnectionIDLength = 16
)

const (
	testName    = "quicping"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`

	// Port is the port to test.
	Port int64 `ooni:"port is the port to test"`

	// networkLibrary is the underlying network library. Can be used for testing.
	networkLib model.UnderlyingNetworkLibrary
}

func (c *Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

func (c *Config) port() string {
	if c.Port != 0 {
		return strconv.FormatInt(c.Port, 10)
	}
	return "443"
}

func (c *Config) networkLibrary() model.UnderlyingNetworkLibrary {
	if c.networkLib != nil {
		return c.networkLib
	}
	return &netxlite.TProxyStdlib{}
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Domain              string                `json:"domain"`
	Pings               []*SinglePing         `json:"pings"`
	UnexpectedResponses []*SinglePingResponse `json:"unexpected_responses"`
	Repetitions         int64                 `json:"repetitions"`
}

// SinglePing is a result of a single ping operation.
type SinglePing struct {
	ConnIdDst string                         `json:"conn_id_dst"`
	ConnIdSrc string                         `json:"conn_id_src"`
	Failure   *string                        `json:"failure"`
	Request   *model.ArchivalMaybeBinaryData `json:"request"`
	T         float64                        `json:"t"`
	Responses []*SinglePingResponse          `json:"responses"`
}

type SinglePingResponse struct {
	Data              *model.ArchivalMaybeBinaryData `json:"response_data"`
	Failure           *string                        `json:"failure"`
	T                 float64                        `json:"t"`
	SupportedVersions []uint32                       `json:"supported_versions"`
}

// makeResponse is a utility function to create a SinglePingResponse
func makeResponse(resp *responseInfo) *SinglePingResponse {
	var data *model.ArchivalMaybeBinaryData
	if resp.raw != nil {
		data = &model.ArchivalMaybeBinaryData{Value: string(resp.raw)}
	}
	return &SinglePingResponse{
		Data:              data,
		Failure:           tracex.NewFailure(resp.err),
		T:                 resp.t,
		SupportedVersions: resp.versions,
	}
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// pingInfo contains information about a ping request
// and the corresponding ping responses
type pingInfo struct {
	request   *requestInfo
	responses []*responseInfo
}

// requestInfo contains the information of a sent ping request.
type requestInfo struct {
	t     float64
	raw   []byte
	dstID string
	srcID string
	err   error
}

// responseInfo contains the information of a received ping reponse.
type responseInfo struct {
	t        float64
	raw      []byte
	dstID    string
	versions []uint32
	err      error
}

// sender sends a ping requests to the target hosts every second
func (m *Measurer) sender(
	ctx context.Context,
	pconn model.UDPLikeConn,
	destAddr *net.UDPAddr,
	out chan<- requestInfo,
	sess model.ExperimentSession,
	measurement *model.Measurement,
) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for i := int64(0); i < m.config.repetitions(); i++ {
		select {
		case <-ctx.Done():
			return // user aborted or timeout expired

		case stime := <-ticker.C:
			sendTime := stime.Sub(measurement.MeasurementStartTimeSaved).Seconds()
			packet, dstID, srcID := buildPacket()     // build QUIC Initial packet
			_, err := pconn.WriteTo(packet, destAddr) // send Initial packet
			if errors.Is(err, net.ErrClosed) {
				return
			}

			sess.Logger().Infof("PING %s", destAddr)

			// propagate send information, including errors
			out <- requestInfo{raw: packet, t: sendTime, dstID: hex.EncodeToString(dstID), srcID: hex.EncodeToString(srcID), err: err}
		}
	}
}

// receiver receives incoming server responses and
// dissects the payload of the version negotiation response
func (m *Measurer) receiver(
	ctx context.Context,
	pconn model.UDPLikeConn,
	out chan<- responseInfo,
	sess model.ExperimentSession,
	measurement *model.Measurement,
) {
	for ctx.Err() == nil {
		// read (timeout was set in Run)
		buffer := make([]byte, 1024)
		n, addr, err := pconn.ReadFrom(buffer)
		respTime := time.Since(measurement.MeasurementStartTimeSaved).Seconds()
		if err != nil {
			// stop if the connection is already closed
			if errors.Is(err, net.ErrClosed) {
				break
			}
			// store read failures and continue receiving
			out <- responseInfo{t: respTime, err: err}
			continue
		}
		resp := buffer[:n]

		// dissect server response
		supportedVersions, dst, err := m.dissectVersionNegotiation(resp)
		if err != nil {
			// the response was likely not the expected version negotiation response
			sess.Logger().Infof(fmt.Sprintf("response dissection failed: %s", err))
			out <- responseInfo{raw: resp, t: respTime, err: err}
			continue
		}
		// propagate receive information
		out <- responseInfo{raw: resp, t: respTime, dstID: hex.EncodeToString(dst), versions: supportedVersions}

		sess.Logger().Infof("PING got response from %s", addr)
	}
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	host := string(measurement.Input)
	// allow URL input
	if u, err := url.ParseRequestURI(host); err == nil {
		host = u.Host
	}
	service := net.JoinHostPort(host, m.config.port())
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	if err != nil {
		return err
	}
	rep := m.config.repetitions()
	tk := &TestKeys{
		Domain:      host,
		Repetitions: rep,
	}
	measurement.TestKeys = tk

	// create UDP socket
	pconn, err := m.config.networkLibrary().ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return err
	}
	defer pconn.Close()

	// set context and read timeouts
	deadline := time.Duration(rep*2) * time.Second
	pconn.SetDeadline(time.Now().Add(deadline))
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	sendInfoChan := make(chan requestInfo)
	recvInfoChan := make(chan responseInfo)
	pingMap := make(map[string]*pingInfo)

	// start sender and receiver goroutines
	go m.sender(ctx, pconn, udpAddr, sendInfoChan, sess, measurement)
	go m.receiver(ctx, pconn, recvInfoChan, sess, measurement)
L:
	for {
		select {
		case req := <-sendInfoChan: // a new ping was sent
			if req.err != nil {
				tk.Pings = append(tk.Pings, &SinglePing{
					ConnIdDst: req.dstID,
					ConnIdSrc: req.srcID,
					Failure:   tracex.NewFailure(req.err),
					Request:   &model.ArchivalMaybeBinaryData{Value: string(req.raw)},
					T:         req.t,
				})
				continue
			}
			pingMap[req.srcID] = &pingInfo{request: &req}

		case resp := <-recvInfoChan: // a new response has been received
			if resp.err != nil {
				// resp failure means we cannot assign the response to a request
				tk.UnexpectedResponses = append(tk.UnexpectedResponses, makeResponse(&resp))
				continue
			}
			var (
				ping *pingInfo
				ok   bool
			)
			// match response to request
			if ping, ok = pingMap[resp.dstID]; !ok {
				// version negotiation response with an unknown destination ID
				tk.UnexpectedResponses = append(tk.UnexpectedResponses, makeResponse(&resp))
				continue
			}
			ping.responses = append(ping.responses, &resp)

		case <-ctx.Done():
			break L
		}
	}
	// transform ping requests into TestKeys.Pings
	timeoutErr := errors.New("i/o timeout")
	for _, ping := range pingMap {
		if ping.request == nil { // this should not happen
			return errors.New("internal error: ping.request is nil")
		}
		if len(ping.responses) <= 0 {
			tk.Pings = append(tk.Pings, &SinglePing{
				ConnIdDst: ping.request.dstID,
				ConnIdSrc: ping.request.srcID,
				Failure:   tracex.NewFailure(timeoutErr),
				Request:   &model.ArchivalMaybeBinaryData{Value: string(ping.request.raw)},
				T:         ping.request.t,
			})
			continue
		}
		var responses []*SinglePingResponse
		for _, resp := range ping.responses {
			responses = append(responses, makeResponse(resp))
		}
		tk.Pings = append(tk.Pings, &SinglePing{
			ConnIdDst: ping.request.dstID,
			ConnIdSrc: ping.request.srcID,
			Failure:   nil,
			Request:   &model.ArchivalMaybeBinaryData{Value: string(ping.request.raw)},
			T:         ping.request.t,
			Responses: responses,
		})
	}
	sort.Slice(tk.Pings, func(i, j int) bool {
		return tk.Pings[i].T < tk.Pings[j].T
	})
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
