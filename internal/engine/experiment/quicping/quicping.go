// Package quicping implements the quicping network experiment. This
// implements, in particular, v0.1.0 of the spec.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-031-quicping.md.
package quicping

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	_ "crypto/sha256"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000000")
}

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
	ConnIdDst   string                         `json:"conn_id_dst"`
	ConnIdSrc   string                         `json:"conn_id_src"`
	Failure     *string                        `json:"failure"`
	Request     *model.ArchivalMaybeBinaryData `json:"request"`
	RequestTime string                         `json:"request_time"`
	Responses   []*SinglePingResponse          `json:"responses"`
}

type SinglePingResponse struct {
	Data              *model.ArchivalMaybeBinaryData `json:"response_data"`
	Failure           *string                        `json:"failure"`
	ResponseTime      string                         `json:"response_time"`
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
		Failure:           archival.NewFailure(resp.err),
		ResponseTime:      formatTime(resp.t),
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
	t     time.Time
	raw   []byte
	dstID string
	srcID string
	err   error
}

// responseInfo contains the information of a received ping reponse.
type responseInfo struct {
	t        time.Time
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
) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	i := 0
	for i < int(m.config.repetitions()) {
		select {
		case <-ctx.Done():
			return // user aborted or timeout expired

		case sendTime := <-ticker.C:
			packet, dstID, srcID := buildPacket()     // build QUIC Initial packet
			_, err := pconn.WriteTo(packet, destAddr) // send Initial packet

			sess.Logger().Infof("PING %s", destAddr)

			// propagate send information, including errors
			out <- requestInfo{raw: packet, t: sendTime, dstID: hex.EncodeToString(dstID), srcID: hex.EncodeToString(srcID), err: err}
			i += 1
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
) {
	for ctx.Err() == nil {
		// read (timeout was set in Run)
		buffer := make([]byte, 1024)
		n, addr, err := pconn.ReadFrom(buffer)
		respTime := time.Now()
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
	go m.sender(ctx, pconn, udpAddr, sendInfoChan, sess)
	go m.receiver(ctx, pconn, recvInfoChan, sess)
L:
	for {
		select {
		case req := <-sendInfoChan: // a new ping was sent
			if req.err != nil {
				tk.Pings = append(tk.Pings, &SinglePing{
					ConnIdDst:   req.dstID,
					ConnIdSrc:   req.srcID,
					Failure:     archival.NewFailure(req.err),
					Request:     &model.ArchivalMaybeBinaryData{Value: string(req.raw)},
					RequestTime: formatTime(req.t),
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
				ConnIdDst:   ping.request.dstID,
				ConnIdSrc:   ping.request.srcID,
				Failure:     archival.NewFailure(timeoutErr),
				Request:     &model.ArchivalMaybeBinaryData{Value: string(ping.request.raw)},
				RequestTime: formatTime(ping.request.t),
			})
			continue
		}
		var responses []*SinglePingResponse
		for _, resp := range ping.responses {
			responses = append(responses, makeResponse(resp))
		}
		tk.Pings = append(tk.Pings, &SinglePing{
			ConnIdDst:   ping.request.dstID,
			ConnIdSrc:   ping.request.srcID,
			Failure:     nil,
			Request:     &model.ArchivalMaybeBinaryData{Value: string(ping.request.raw)},
			RequestTime: formatTime(ping.request.t),
			Responses:   responses,
		})
	}
	return nil
}

// dissectVersionNegotiation dissects the Version Negotiation response.
// It returns the supported versions and the destination connection ID of the response,
// The destination connection ID of the response has to coincide with the source connection ID of the request.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-version-negotiation-packet
func (m *Measurer) dissectVersionNegotiation(i []byte) ([]uint32, connectionID, error) {
	firstByte := uint8(i[0])
	mask := 0b10000000
	mask &= int(firstByte)
	if mask == 0 {
		return nil, nil, &errUnexpectedResponse{msg: "not a long header packet"}
	}

	versionBytes := i[1:5]
	v := binary.BigEndian.Uint32(versionBytes)
	if v != 0 {
		return nil, nil, &errUnexpectedResponse{msg: "unexpected Version Negotiation format"}
	}

	dstLength := i[5]
	offset := 6 + uint8(dstLength)
	dst := i[6:offset]

	srcLength := i[offset]
	offset = offset + 1 + srcLength

	n := uint8(len(i))
	var supportedVersions []uint32
	for offset < n {
		supportedVersions = append(supportedVersions, binary.BigEndian.Uint32(i[offset:offset+4]))
		offset += 4
	}
	return supportedVersions, dst, nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}

// buildHeader creates the unprotected QUIC header.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-initial-packet
func buildHeader(destConnID, srcConnID connectionID, payloadLen int) []byte {
	hdr := []byte{0xc3} // long header type, fixed

	version := make([]byte, 4)
	binary.BigEndian.PutUint32(version, uint32(0xbabababa))
	hdr = append(hdr, version...) // version

	lendID := uint8(len(destConnID))
	hdr = append(hdr, lendID)        // destination connection ID length
	hdr = append(hdr, destConnID...) // destination connection ID

	lensID := uint8(len(srcConnID))
	hdr = append(hdr, lensID)       // source connection ID length
	hdr = append(hdr, srcConnID...) // source connection ID

	hdr = append(hdr, 0x0) // token length

	remainder := 4 + payloadLen
	remainder_mask := 0b100000000000000
	remainder_mask |= remainder
	remainder_b := make([]byte, 2)
	binary.BigEndian.PutUint16(remainder_b, uint16(remainder_mask))
	hdr = append(hdr, remainder_b...) // remainder length: packet number + encrypted payload

	pn := make([]byte, 4)
	binary.BigEndian.PutUint32(pn, uint32(2))
	hdr = append(hdr, pn...) // packet number

	return hdr
}

// buildPacket constructs an Initial QUIC packet
// and applies Initial protection.
// https://www.rfc-editor.org/rfc/rfc9001.html#name-client-initial
func buildPacket() ([]byte, connectionID, connectionID) {
	destConnID, srcConnID := generateConnectionIDs()
	// generate random payload
	minPayloadSize := 1200 - 14 - (len(destConnID) + len(srcConnID))
	randomPayload := make([]byte, minPayloadSize)
	rand.Read(randomPayload)

	clientSecret, _ := computeSecrets(destConnID)
	encrypted := encryptPayload(randomPayload, destConnID, clientSecret)
	hdr := buildHeader(destConnID, srcConnID, len(encrypted))
	raw := append(hdr, encrypted...)

	raw = encryptHeader(raw, hdr, clientSecret)
	return raw, destConnID, srcConnID
}

// generateConnectionID generates a connection ID using cryptographic random
func generateConnectionID(len int) connectionID {
	b := make([]byte, len)
	_, err := rand.Read(b)
	runtimex.PanicOnError(err, "rand.Read failed")
	return connectionID(b)
}

// generateConnectionIDForInitial generates a connection ID for the Initial packet.
// It uses a length randomly chosen between 8 and 18 bytes.
func generateConnectionIDForInitial() connectionID {
	r := make([]byte, 1)
	_, err := rand.Read(r)
	runtimex.PanicOnError(err, "rand.Read failed")
	len := minConnectionIDLenInitial + int(r[0])%(maxConnectionIDLen-minConnectionIDLenInitial+1)
	return generateConnectionID(len)
}

// generateConnectionIDs generates a destination and source connection ID.
func generateConnectionIDs() ([]byte, []byte) {
	destConnID := generateConnectionIDForInitial()
	srcConnID := generateConnectionID(defaultConnectionIDLength)
	return destConnID, srcConnID
}

// errUnexpectedResponse is thrown when the response from the server
// is not a valid Version Negotiation packet
type errUnexpectedResponse struct {
	error
	msg string
}

// Error implements error.Error()
func (e *errUnexpectedResponse) Error() string {
	return fmt.Sprintf("unexptected response: %s", e.msg)
}
