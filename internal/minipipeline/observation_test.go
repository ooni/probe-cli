package minipipeline

import (
	"errors"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

func TestLoadWebObservations(t *testing.T) {
	t.Run("we handle the case where the test keys are nil", func(t *testing.T) {
		meas := &WebMeasurement{ /* empty */ }
		container, err := IngestWebMeasurement(meas)
		if !errors.Is(err, ErrNoTestKeys) {
			t.Fatal("expected", ErrNoTestKeys, "got", err)
		}
		if container != nil {
			t.Fatal("expected nil container, got", container)
		}
	})

	t.Run("we handle the case where the input is not a valid URL", func(t *testing.T) {
		meas := &WebMeasurement{
			Input: "https://www.example.com", // invalid URL
			TestKeys: optional.Some(&WebMeasurementTestKeys{
				Control:        optional.Some(&model.THResponse{}),
				NetworkEvents:  []*model.ArchivalNetworkEvent{},
				Queries:        []*model.ArchivalDNSLookupResult{},
				Requests:       []*model.ArchivalHTTPRequestResult{},
				TCPConnect:     []*model.ArchivalTCPConnectResult{},
				TLSHandshakes:  []*model.ArchivalTLSOrQUICHandshakeResult{},
				QUICHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{},
				XControlRequest: optional.Some(&model.THRequest{
					HTTPRequest:        "\t", // this should fail to parse
					HTTPRequestHeaders: map[string][]string{},
					TCPConnect:         []string{},
					XQUICEnabled:       false,
				}),
			}),
		}
		container, err := IngestWebMeasurement(meas)
		if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
			t.Fatal("unexpected err", err)
		}
		if container != nil {
			t.Fatal("expected nil container, got", container)
		}
	})
}

func TestWebObservationsContainerIngestTLSHandshakeEvents(t *testing.T) {
	t.Run("when we don't have any known TCP endpoint", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: []*WebObservation{},
			KnownTCPEndpoints: map[int64]*WebObservation{}, // this map must be empty in this test
			knownIPAddresses:  map[string]*WebObservation{},
		}

		handshake := &model.ArchivalTLSOrQUICHandshakeResult{
			Network:            "",
			Address:            "",
			CipherSuite:        "",
			Failure:            nil,
			SoError:            nil,
			NegotiatedProtocol: "",
			NoTLSVerify:        false,
			PeerCertificates:   []model.ArchivalBinaryData{},
			ServerName:         "",
			T0:                 0,
			T:                  0,
			Tags:               []string{},
			TLSVersion:         "",
			TransactionID:      0, // any transaction ID would do since the map is empty
		}

		container.IngestTLSHandshakeEvents(handshake)

		// we should not crash and we should not have added new endpoints
		if len(container.KnownTCPEndpoints) != 0 {
			t.Fatal("the number of known TCP endpoints should not have changed")
		}
	})
}

func TestWebObservationsContainerIngestHTTPRoundTripEvents(t *testing.T) {
	t.Run("when we don't have any known TCP endpoint", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: []*WebObservation{},
			KnownTCPEndpoints: map[int64]*WebObservation{}, // this map must be empty in this test
			knownIPAddresses:  map[string]*WebObservation{},
		}

		roundTrip := &model.ArchivalHTTPRequestResult{
			Network:       "",
			Address:       "",
			ALPN:          "",
			Failure:       nil,
			Request:       model.ArchivalHTTPRequest{},
			Response:      model.ArchivalHTTPResponse{},
			T0:            0,
			T:             0,
			Tags:          []string{},
			TransactionID: 0, // any transaction ID would do since the map is empty
		}

		container.IngestHTTPRoundTripEvents(roundTrip)

		// we should not crash and we should not have added new endpoints
		if len(container.KnownTCPEndpoints) != 0 {
			t.Fatal("the number of known TCP endpoints should not have changed")
		}
	})
}

func TestWebObservationsContainerIngestControlMessages(t *testing.T) {
	t.Run("we don't save TLS handshake failures when the SNI is different", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: []*WebObservation{},
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					IPAddress:             optional.Some("8.8.8.8"),
					EndpointTransactionID: optional.Some(int64(1)),
					EndpointPort:          optional.Some("443"),
					EndpointAddress:       optional.Some("8.8.8.8:443"),
					TLSServerName:         optional.Some("dns.google.com"),
				},
			},
			knownIPAddresses: map[string]*WebObservation{},
		}

		thRequest := &model.THRequest{
			HTTPRequest: "https://dns.google/",
		}

		thResponse := &model.THResponse{
			TLSHandshake: map[string]model.THTLSHandshakeResult{
				"8.8.8.8:443": {
					ServerName: "dns.google",
					Status:     true,
					Failure:    nil,
				},
			},
		}

		if err := container.IngestControlMessages(thRequest, thResponse); err != nil {
			t.Fatal(err)
		}

		entry := container.KnownTCPEndpoints[1]

		if !entry.ControlTLSHandshakeFailure.IsNone() {
			t.Fatal("ControlTLSHandshakeFailure should be none")
		}
	})
}
