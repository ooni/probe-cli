// Package quicfp implements the quicfp network experiment: a QUIC parser
// fingerprinting probe that reproduces every field of Table 9 ("QUIC Parser
// Fingerprints") and Table 12 ("TLS and QUIC Parser Fingerprints") from
// Ablove et al., "Technical Analysis of the Geedge Networks Firewall Source
// Code Leak" (USENIX Security 2026).
//
// Where internal/experiment/quicping sends a single, content-free QUIC
// Initial packet to elicit a Version Negotiation response, quicfp sends one
// Initial datagram (or, for two vectors, a short sequence of datagrams) per
// test vector, each carrying a synthetic TLS 1.3 ClientHello for the target
// SNI. The vectors span three of Table 9's dimensions:
//
//   - QUIC Version:  one representative version per row (gQUIC Q023-Q043,
//     Q041/Q044-45, Q046-49, T050-51; IETF draft-22-28, draft-29/v1, v2).
//   - QUIC Features: fragmentation, below-1200-byte datagrams, forced
//     version negotiation, and a ClientHello split across multiple Initial
//     packets.
//   - TLS Length Fields: all eight fields Table 12 tests (Message,
//     Session ID, Cipher Suite, Compression Methods, Extensions, SNI
//     Extension, SNI List, and SNI Name Length), each set to the
//     permutations the paper describes.
//
// Table 9's final row, "Bypasses Censor", is not something a single vantage
// point can determine on its own -- the paper derives it by comparing
// results from censored and uncensored vantage points. What this experiment
// records instead, per vector and repetition, is the raw signal an analyst
// needs to make that call: whether a response (Version Negotiation or
// otherwise) arrived before the deadline, and the raw bytes of whatever did
// or did not come back. See SinglePing.Blocked for the same "no response
// within the deadline" heuristic internal/experiment/quicping already uses.
package quicfp

import (
	"context"
	"encoding/hex"
	"net"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/legacymodel"
	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "quicfp"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Repetitions is the number of times to repeat each vector.
	Repetitions int64 `json:"repetitions,omitempty" ooni:"number of times to repeat each test vector"`

	// Port is the port to test.
	Port int64 `json:"port,omitempty" ooni:"port is the port to test"`

	// SNI overrides the SNI placed in the synthetic ClientHello. If
	// empty, the resolved input domain is used, matching normal QUIC/TLS
	// probing convention.
	SNI string `json:"sni,omitempty" ooni:"overrides the SNI used in the synthetic ClientHello"`

	// VectorTimeout is how long (in milliseconds) to wait for a response
	// to a single vector repetition before declaring it timed out.
	VectorTimeout int64 `json:"vector_timeout,omitempty" ooni:"milliseconds to wait for a response before declaring a vector timed out"`

	// netListenUDP allows mocking the real net.ListenUDP call in tests.
	netListenUDP func(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error)
}

func (c *Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 1
}

func (c *Config) port() string {
	if c.Port != 0 {
		return strconv.FormatInt(c.Port, 10)
	}
	return "443"
}

func (c *Config) vectorTimeout() time.Duration {
	if c.VectorTimeout > 0 {
		return time.Duration(c.VectorTimeout) * time.Millisecond
	}
	return 2 * time.Second
}

func (c *Config) doListenUDP(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
	if c.netListenUDP != nil {
		return c.netListenUDP(network, laddr)
	}
	return net.ListenUDP(network, laddr)
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Domain      string          `json:"domain"`
	SNI         string          `json:"sni"`
	Repetitions int64           `json:"repetitions"`
	Vectors     []*VectorResult `json:"vectors"`
}

// VectorResult holds every repetition of a single test vector, i.e. one row
// (or one field/permutation pair) of Table 9.
type VectorResult struct {
	Name        string        `json:"vector_name"`
	Category    string        `json:"category"`
	Field       string        `json:"field,omitempty"`
	Permutation string        `json:"permutation,omitempty"`
	QUICGroup   string        `json:"quic_version_group,omitempty"`
	QUICVersion *uint32       `json:"quic_version,omitempty"`
	Pings       []*SinglePing `json:"pings"`
}

// SinglePing is the result of sending one vector once. Requests holds one
// entry per datagram sent (more than one only for quic_feature_multi_packet_initial).
type SinglePing struct {
	ConnIDDst string                                 `json:"conn_id_dst"`
	ConnIDSrc string                                 `json:"conn_id_src"`
	Failure   *string                                `json:"failure"`
	Requests  []*legacymodel.ArchivalMaybeBinaryData `json:"requests"`
	T         float64                                `json:"t"`
	Responses []*SinglePingResponse                  `json:"responses"`
	// Blocked is a best-effort heuristic: true if no response of any kind
	// arrived before the deadline. As documented at the package level,
	// this is NOT the same thing as Table 9's "Bypasses Censor" row,
	// which requires comparing against a control measurement.
	Blocked bool `json:"blocked"`
}

// SinglePingResponse is one datagram received in reply to a SinglePing.
type SinglePingResponse struct {
	Data              *legacymodel.ArchivalMaybeBinaryData `json:"response_data"`
	Failure           *string                              `json:"failure"`
	T                 float64                              `json:"t"`
	Type              string                               `json:"type"` // "version_negotiation" | "unknown"
	SupportedVersions []uint32                             `json:"supported_versions,omitempty"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string { return testName }

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string { return testVersion }

// resolveTarget parses a bare host, host:port, or URL (matching
// internal/experiment/quicping's convention) into a target address and the
// SNI to embed in the ClientHello.
func (m *Measurer) resolveTarget(input string) (addr *net.UDPAddr, sni string, err error) {
	host := input
	port := ""
	if u, uerr := url.ParseRequestURI(input); uerr == nil && u.Hostname() != "" {
		host = u.Hostname()
		port = u.Port()
	} else if h, p, serr := net.SplitHostPort(input); serr == nil {
		// A plain "host:port" input, no URL scheme -- e.g. a bare IP:port.
		// Must be tried before falling back to "the whole input is one
		// bare host", or a string like "127.0.0.1:49772" gets treated as
		// a single (unresolvable) hostname with the default port
		// appended on top.
		host = h
		port = p
	}
	if port == "" {
		port = m.config.port()
	}
	sni = host
	if m.config.SNI != "" {
		sni = m.config.SNI
	}
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, "", err
	}
	return udpAddr, sni, nil
}

// sendAndAwait sends every datagram in reqs (back-to-back, on the same
// 4-tuple) from pconn, a socket of this ping's own, and waits up to the
// configured vector timeout for a response
// whose Version-Negotiation Destination Connection ID matches srcID (i.e.
// the source connection ID we sent), classifying whatever arrives.
func (m *Measurer) sendAndAwait(
	pconn model.UDPLikeConn,
	destAddr *net.UDPAddr,
	reqs [][]byte,
	srcID connectionID,
	startedAt time.Time,
	timeout time.Duration,
) (sendErr error, responses []*SinglePingResponse) {
	_ = pconn.SetDeadline(time.Now().Add(timeout))
	for _, req := range reqs {
		if _, err := pconn.WriteTo(req, destAddr); err != nil {
			return err, nil
		}
	}
	buf := make([]byte, 2048)
	for {
		n, _, err := pconn.ReadFrom(buf)
		respT := time.Since(startedAt).Seconds()
		if err != nil {
			// Deadline exceeded: stop waiting for this vector,
			// reported as "no response", i.e. Blocked=true upstream.
			return nil, responses
		}
		raw := append([]byte{}, buf[:n]...)
		versions, dst, verr := dissectVersionNegotiation(raw)
		if verr != nil {
			responses = append(responses, &SinglePingResponse{
				Data:    &legacymodel.ArchivalMaybeBinaryData{Value: string(raw)},
				Failure: tracex.NewFailure(verr),
				T:       respT,
				Type:    "unknown",
			})
			continue
		}
		if hex.EncodeToString(dst) != hex.EncodeToString(srcID) {
			// Not addressed to us (or a stray packet); keep waiting
			// until the deadline.
			continue
		}
		responses = append(responses, &SinglePingResponse{
			Data:              &legacymodel.ArchivalMaybeBinaryData{Value: string(raw)},
			T:                 respT,
			Type:              "version_negotiation",
			SupportedVersions: versions,
		})
		return nil, responses
	}
}

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// Obtain the richer-input target, exactly like quicping.
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}
	m.config = *target.Config

	destAddr, sni, err := m.resolveTarget(target.URL)
	if err != nil {
		return err
	}

	vectors, err := buildVectors(sni)
	if err != nil {
		return err
	}

	tk := &TestKeys{
		Domain:      target.URL,
		SNI:         sni,
		Repetitions: m.config.repetitions(),
	}
	measurement.TestKeys = tk

	timeout := m.config.vectorTimeout()

	for _, v := range vectors {
		if ctx.Err() != nil {
			break
		}
		vr := &VectorResult{
			Name:        v.Name,
			Category:    v.Category,
			Field:       v.Field,
			Permutation: v.Permutation,
			QUICGroup:   v.QUICGroup,
		}
		if v.Category == CategoryQUICVersion {
			ver := v.QUICVersion
			vr.QUICVersion = &ver
		}

		for i := int64(0); i < m.config.repetitions(); i++ {
			if ctx.Err() != nil {
				break
			}
			reqs, dst, src, berr := v.Build()
			sendT := time.Since(measurement.MeasurementStartTimeSaved).Seconds()
			if berr != nil {
				vr.Pings = append(vr.Pings, &SinglePing{
					Failure: tracex.NewFailure(berr),
					T:       sendT,
				})
				continue
			}

			var archivalReqs []*legacymodel.ArchivalMaybeBinaryData
			for _, r := range reqs {
				archivalReqs = append(archivalReqs, &legacymodel.ArchivalMaybeBinaryData{Value: string(r)})
			}

			sess.Logger().Infof("quicfp: %s -> %s", v.Name, destAddr)
			// A fresh socket, so a fresh source port, for every ping: a late
			// reply to one vector can't be counted for the next, and a censor
			// that blocks a 4-tuple after a trigger (residual censorship, which
			// the paper's Section 6.4 avoids by rotating client ports) can't
			// make every later vector look blocked.
			pconn, err := m.config.doListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				return err
			}
			senderr, responses := m.sendAndAwait(pconn, destAddr, reqs, src, measurement.MeasurementStartTimeSaved, timeout)
			pconn.Close()

			sp := &SinglePing{
				ConnIDDst: hex.EncodeToString(dst),
				ConnIDSrc: hex.EncodeToString(src),
				Failure:   tracex.NewFailure(senderr),
				Requests:  archivalReqs,
				T:         sendT,
				Responses: responses,
				Blocked:   senderr == nil && len(responses) == 0,
			}
			vr.Pings = append(vr.Pings, sp)
		}
		tk.Vectors = append(tk.Vectors, vr)
	}

	sort.Slice(tk.Vectors, func(i, j int) bool { return tk.Vectors[i].Name < tk.Vectors[j].Name })
	return nil
}

// NewExperimentMeasurer creates a new model.ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}
