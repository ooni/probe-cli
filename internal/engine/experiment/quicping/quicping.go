package quicping

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	random "math/rand"
	"net"
	"net/url"
	"time"

	_ "crypto/sha256"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// A ConnectionID in QUIC
type ConnectionID []byte

const maxConnectionIDLen = 18
const MinConnectionIDLenInitial = 8
const DefaultConnectionIDLength = 16

const (
	testName    = "quicping"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`

	// Port is the port to test.
	Port string `ooni:"port is the port to test"`

	// WaitSeconds is the number of seconds to wait for the ping response
	WaitSeconds int `ooni:"waitseconds is the number of seconds to wait for the ping response"`
}

func (c *Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

func (c *Config) port() string {
	if c.Port != "" {
		return c.Port
	}
	return "443"
}

func (c *Config) waitseconds() int {
	if c.WaitSeconds != 0 {
		return c.WaitSeconds
	}
	return 5
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Domain      string
	Pings       []*SinglePing `json:"pings"`
	Repetitions int64
}

type SinglePing struct {
	ConnIdDst         ConnectionID
	ConnIdSrc         ConnectionID
	Failure           *string
	Ping              *model.ArchivalMaybeBinaryData
	Response          *model.ArchivalMaybeBinaryData
	SupportedVersions []uint32
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

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	url, err := url.Parse(string(measurement.Input))
	if err != nil {
		return errors.New("input is not an URL")
	}
	tk := new(TestKeys)
	tk.Domain = url.Host
	tk.Repetitions = m.config.repetitions()
	measurement.TestKeys = tk

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	service := net.JoinHostPort(url.Host, m.config.port())

	// create UDP socket
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	for i := int64(0); i < m.config.repetitions(); i++ {
		<-ticker.C
		sess.Logger().Infof("PING %s", service)

		sent, dstID, srcID, err := buildPacket() // build QUIC Initial packet
		if err != nil {
			return errors.New(fmt.Sprintf("buildPacket failed: %s", err.Error()))
		}
		_, err = conn.Write(sent) // send Initial packet
		if err != nil {
			return errors.New(fmt.Sprintf("UDP send failed: %s", err.Error()))
		}
		resp, err := m.waitResponse(conn) // wait for server response
		if err != nil {
			tk.Pings = append(tk.Pings, &SinglePing{
				ConnIdDst: dstID,
				ConnIdSrc: srcID,
				Failure:   archival.NewFailure(err),
				Ping:      &model.ArchivalMaybeBinaryData{Value: string(sent)},
				Response:  nil,
			})
			continue
		}
		supportedVersions, err := m.dissectVersionNegotiation(resp, dstID, srcID) // dissect server response
		if err != nil {
			tk.Pings = append(tk.Pings, &SinglePing{
				ConnIdDst: dstID,
				ConnIdSrc: srcID,
				Failure:   archival.NewFailure(err),
				Ping:      &model.ArchivalMaybeBinaryData{Value: string(sent)},
				Response:  &model.ArchivalMaybeBinaryData{Value: string(resp)},
			})
			continue
		}
		tk.Pings = append(tk.Pings, &SinglePing{
			ConnIdDst:         dstID,
			ConnIdSrc:         srcID,
			Failure:           nil,
			Ping:              &model.ArchivalMaybeBinaryData{Value: string(sent)},
			Response:          &model.ArchivalMaybeBinaryData{Value: string(resp)},
			SupportedVersions: supportedVersions,
		})
	}

	return nil
}

// waitResponse reads the server response. Times out after m.config.waitseconds() seconds (default: 5).
func (m *Measurer) waitResponse(conn *net.UDPConn) ([]byte, error) {
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Duration(m.config.waitseconds()) * time.Second))
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[0:n], nil
}

// dissectVersionNegotiation dissects the Version Negotiation response
// and prints it to the command line.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-version-negotiation-packet
func (m *Measurer) dissectVersionNegotiation(i []byte, dstID, srcID ConnectionID) ([]uint32, error) {
	firstByte := uint8(i[0])
	mask := 0b10000000
	mask &= int(firstByte)
	if mask == 0 {
		return nil, &errUnexpectedResponse{msg: "not a long header packet"}
	}

	versionBytes := i[1:5]
	v := binary.BigEndian.Uint32(versionBytes)
	if v != 0 {
		return nil, &errUnexpectedResponse{msg: "unexpected Version Negotiation format"}
	}

	dstLength := i[5]
	offset := 6 + uint8(dstLength)
	dst := i[6:offset]
	if hex.EncodeToString(dst) != hex.EncodeToString(srcID) {
		return nil, &errUnexpectedResponse{msg: fmt.Sprintf("destination connection ID: is %s, was %s", dst, srcID)}
	}
	srcLength := i[offset]
	src := i[offset+1 : offset+1+srcLength]
	offset = offset + 1 + srcLength
	if hex.EncodeToString(src) != hex.EncodeToString(dstID) {
		return nil, &errUnexpectedResponse{msg: fmt.Sprintf("destination connection ID: is %s, was %s", src, dstID)}
	}

	n := uint8(len(i))
	var supportedVersions []uint32
	for offset < n {
		supportedVersions = append(supportedVersions, binary.BigEndian.Uint32(i[offset:offset+4]))
		offset += 4
	}
	return supportedVersions, nil
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
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
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
func buildPacket() ([]byte, ConnectionID, ConnectionID, error) {
	destConnID, srcConnID, err := generateConnectionIDs()
	if err != nil {
		return nil, nil, nil, err
	}
	// generate random payload
	minPayloadSize := 1200 - 14 - (len(destConnID) + len(srcConnID))
	randomPayload := make([]byte, minPayloadSize)
	random.Seed(time.Now().UnixNano())
	random.Read(randomPayload)

	clientSecret, _ := computeSecrets(destConnID)
	encrypted := encryptPayload(randomPayload, destConnID, clientSecret)
	hdr := buildHeader(destConnID, srcConnID, len(encrypted))
	raw := append(hdr, encrypted...)

	raw = encryptHeader(raw, hdr, clientSecret)
	return raw, destConnID, srcConnID, nil
}

// generateConnectionID generates a connection ID using cryptographic random
func generateConnectionID(len int) (ConnectionID, error) {
	b := make([]byte, len)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return ConnectionID(b), nil
}

// generateConnectionIDForInitial generates a connection ID for the Initial packet.
// It uses a length randomly chosen between 8 and 18 bytes.
func generateConnectionIDForInitial() (ConnectionID, error) {
	r := make([]byte, 1)
	if _, err := rand.Read(r); err != nil {
		return nil, err
	}
	len := MinConnectionIDLenInitial + int(r[0])%(maxConnectionIDLen-MinConnectionIDLenInitial+1)
	return generateConnectionID(len)
}

// generateConnectionIDs generates a destination and source connection ID.
func generateConnectionIDs() ([]byte, []byte, error) {
	destConnID, err := generateConnectionIDForInitial()
	if err != nil {
		return nil, nil, err
	}
	srcConnID, err := generateConnectionID(DefaultConnectionIDLength)
	if err != nil {
		return nil, nil, err
	}
	return destConnID, srcConnID, nil
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
