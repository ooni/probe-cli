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
	"sync"
	"time"

	_ "crypto/sha256"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// A ConnectionID in QUIC
type ConnectionID []byte

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

	// Timeout is the number of milliseconds to wait for the ping response
	Timeout int64 `ooni:"Timeout is the number of milliseconds to wait for the ping response"`

	// NetworkLibrary is the underlying network library. Can be used for testing.
	NetworkLibrary model.UnderlyingNetworkLibrary
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

func (c *Config) timeout() int64 {
	if c.Timeout != 0 {
		return c.Timeout
	}
	return 5000
}

func (c *Config) networkLibrary() model.UnderlyingNetworkLibrary {
	if c.NetworkLibrary != nil {
		return c.NetworkLibrary
	}
	return &netxlite.TProxyStdlib{}
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Domain      string        `json:"domain"`
	Pings       []*SinglePing `json:"pings"`
	Repetitions int64         `json:"repetitions"`
}

// SinglePing is a result of a single ping operation.
type SinglePing struct {
	ConnIdDst         ConnectionID                   `json:"conn_id_dst"`
	ConnIdSrc         ConnectionID                   `json:"conn_id_src"`
	Failure           *string                        `json:"failure"`
	Request           *model.ArchivalMaybeBinaryData `json:"request"`
	RequestTime       string                         `json:"request_time"`
	Response          *model.ArchivalMaybeBinaryData `json:"response"`
	ResponseTime      string                         `json:"response_time"`
	SupportedVersions []uint32                       `json:"supported_versions"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
	mu     sync.Mutex
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

type sendInfo struct {
	dstID    ConnectionID
	srcID    ConnectionID
	sendTime time.Time
	raw      []byte
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	host := string(measurement.Input)
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

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// create UDP socket
	conn, err := m.config.networkLibrary().ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return err
	}
	conn.SetReadDeadline(time.Now().Add(time.Duration(rep*m.config.timeout()) * time.Millisecond))
	defer conn.Close()

	sendInfoMap := make(map[string]*sendInfo)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for i := int64(0); i < rep; i++ {
			sess.Logger().Infof("PING %s", service)

			sent, dstID, srcID := buildPacket()  // build QUIC Initial packet
			_, err = conn.WriteTo(sent, udpAddr) // send Initial packet
			sendTime := time.Now()
			if err != nil {
				tk.Pings = append(tk.Pings, &SinglePing{
					Failure:     archival.NewFailure(err),
					RequestTime: formatTime(sendTime),
				})
				continue
			}
			m.mu.Lock()
			sendInfoMap[hex.EncodeToString(srcID)] = &sendInfo{dstID: dstID, srcID: srcID, raw: sent, sendTime: sendTime}
			m.mu.Unlock()
			<-ticker.C
		}
		wg.Done()
	}()

	for {
		resp, respTime, err := m.waitResponse(conn) // wait for server response
		if err != nil {
			break
		}
		supportedVersions, dst, err := m.DissectVersionNegotiation(resp) // dissect server response
		if err != nil {
			sess.Logger().Infof(fmt.Sprintf("response dissection failed: %s", err))
			continue
		}
		var (
			req *sendInfo
			ok  bool
		)
		m.mu.Lock()
		if req, ok = sendInfoMap[dst]; !ok {
			continue // we have not send a request for this response, so let's discard it for now
		}
		delete(sendInfoMap, dst)
		m.mu.Unlock()

		tk.Pings = append(tk.Pings, &SinglePing{
			ConnIdDst:         req.dstID,
			ConnIdSrc:         req.srcID,
			Failure:           nil,
			Request:           &model.ArchivalMaybeBinaryData{Value: string(req.raw)},
			RequestTime:       formatTime(req.sendTime),
			Response:          &model.ArchivalMaybeBinaryData{Value: string(resp)},
			ResponseTime:      formatTime(*respTime),
			SupportedVersions: supportedVersions,
		})
		sess.Logger().Infof("PING got response from %s", service)

		if len(tk.Pings) == int(rep) {
			break
		}
	}
	wg.Wait()
	timeoutErr := errors.New("i/o timeout")
	for _, req := range sendInfoMap {
		tk.Pings = append(tk.Pings, &SinglePing{
			ConnIdDst:         req.dstID,
			ConnIdSrc:         req.srcID,
			Failure:           archival.NewFailure(timeoutErr),
			Request:           &model.ArchivalMaybeBinaryData{Value: string(req.raw)},
			Response:          nil,
			SupportedVersions: nil,
		})
	}

	return nil
}

// waitResponse reads the server response. Times out after m.config.timeout() seconds (default: 5000).
func (m *Measurer) waitResponse(conn model.UDPLikeConn) ([]byte, *time.Time, error) {
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFrom(buffer)
	respTime := time.Now()
	if err != nil {
		return nil, nil, err
	}
	return buffer[:n], &respTime, nil
}

// DissectVersionNegotiation dissects the Version Negotiation response
// and prints it to the command line.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-version-negotiation-packet
func (m *Measurer) DissectVersionNegotiation(i []byte) ([]uint32, string, error) {
	firstByte := uint8(i[0])
	mask := 0b10000000
	mask &= int(firstByte)
	if mask == 0 {
		return nil, "", &errUnexpectedResponse{msg: "not a long header packet"}
	}

	versionBytes := i[1:5]
	v := binary.BigEndian.Uint32(versionBytes)
	if v != 0 {
		return nil, "", &errUnexpectedResponse{msg: "unexpected Version Negotiation format"}
	}

	dstLength := i[5]
	offset := 6 + uint8(dstLength)
	dst := i[6:offset]

	n := uint8(len(i))
	var supportedVersions []uint32
	for offset < n {
		supportedVersions = append(supportedVersions, binary.BigEndian.Uint32(i[offset:offset+4]))
		offset += 4
	}
	return supportedVersions, hex.EncodeToString(dst), nil
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
func buildHeader(destConnID, srcConnID ConnectionID, payloadLen int) []byte {
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
func buildPacket() ([]byte, ConnectionID, ConnectionID) {
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
func generateConnectionID(len int) ConnectionID {
	b := make([]byte, len)
	_, err := rand.Read(b)
	runtimex.PanicOnError(err, "rand.Read failed")
	return ConnectionID(b)
}

// generateConnectionIDForInitial generates a connection ID for the Initial packet.
// It uses a length randomly chosen between 8 and 18 bytes.
func generateConnectionIDForInitial() ConnectionID {
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
