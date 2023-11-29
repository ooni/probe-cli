package minipipeline

import (
	"errors"
	"net"
	"net/url"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// ErrNoTestKeys indicates that a [*WebMeasurement] does not contain [*MeasurementTestKeys].
var ErrNoTestKeys = errors.New("minipipeline: no test keys")

// IngestWebMeasurement loads a [*WebMeasurement] into a [*WebObservationsContainter]. To this
// end, we create a [*WebObservationsContainer] and fill it with the contents of the input
// [*WebMeasurement]. An empty [*WebMeasurement] will cause this function to produce an empty
// result. It is safe to pass to this function a [*WebMeasurement] with empty Control and
// XControlRequestFields: in such a case, this function will just avoid using the test helper
// (aka control) information for generating flat [*WebObservation]. This function returns an
// error if the [*WebMeasurement] TestKeys are empty or Input is not a valid URL.
func IngestWebMeasurement(meas *WebMeasurement) (*WebObservationsContainer, error) {
	tk := meas.TestKeys.UnwrapOr(nil)
	if tk == nil {
		return nil, ErrNoTestKeys
	}

	container := NewWebObservationsContainer(optional.Some(meas.Input))
	container.IngestDNSLookupEvents(tk.Queries...)
	container.IngestTCPConnectEvents(tk.TCPConnect...)
	container.IngestTLSHandshakeEvents(tk.TLSHandshakes...)
	container.IngestHTTPRoundTripEvents(tk.Requests...)

	// be defensive in case the control request or control are not defined
	if !tk.XControlRequest.IsNone() && !tk.Control.IsNone() {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have triggered previous errors if we're
		// running this code as part of Web Connectivity LTE.
		if err := container.IngestControlMessages(tk.XControlRequest.Unwrap(), tk.Control.Unwrap()); err != nil {
			return nil, err
		}
	}

	return container, nil
}

// WebObservation is an observation of the flow that starts with a DNS lookup that
// either fails or discovers an IP address and proceeds by documenting binding such an
// address to a part to obtain and use a TCP or UDP endpoint.
//
// A key property of the [WebObservation] is that there is a single failure mode
// for the whole [WebObservation]. If the DNS fails, there is no IP address to
// construct and endpoint. If TCP connect fails, there is no connection to use for
// a TLS handshake. Likewise, if QUIC fails, there is also no connection. Finally,
// if there is no suitable connection, we cannot peform an HTTP round trip.
//
// Most fields are optional.Value fields. When the field contains an optional.None, it
// means that the related information is not available. We represent failures using flat
// strings and we use optional.Some("") to indicate the absence of any errors.
//
// We borrow this struct from https://github.com/ooni/data.
type WebObservation struct {
	// Input is the measurement input.
	Input optional.Value[string]

	// The following fields are optional.Some when you process the DNS
	// lookup events contained inside an OONI measurement:

	// DNSTransactionIDs contains the ID of the DNS transaction that caused this
	// specific [*WebObservation] to be generated by the minipipeline.
	DNSTransactionID optional.Value[int64]

	// DNSDomain is the domain from which we resolved the IP address. This field
	// is empty when this record wasn't generated by a DNS lookup. This occurs, e.g.,
	// when Web Connectivity LTE discovers new addresses from the TH response.
	DNSDomain optional.Value[string]

	// DNSLookupFailure is the failure that occurred during the DNS lookup. This field will be
	// optional.None if there's no DNS lookup information. Otherwise, it contains a string
	// representing the error, where the empty string means success.
	DNSLookupFailure optional.Value[string]

	// DNSQueryType is the type of the DNS query (e.g., "A").
	DNSQueryType optional.Value[string]

	// DNSEngine is the DNS engine that we're using (e.g., "getaddrinfo").
	DNSEngine optional.Value[string]

	// The following fields are optional.Some in these cases:
	//
	// 1. when you process successful DNS lookup events from OONI measurements;
	//
	// 2. when the experiment discovers IP addresses through the TH response;
	//
	// 3. when the input URL contains an IP address.

	// IPAddress is the optional IP address that this observation is about. We typically derive
	// this value from a DNS lookup, but sometimes we know it from other means (e.g., from
	// the Web Connectivity test helper response). When DNSLookupFailure contains an nonempty
	// error string, the DNS lookup failed and this field is an optional.None.
	IPAddress optional.Value[string]

	// IPAddressASN is the optional ASN associated to this IP address as discovered by
	// the probe while performing the measurement. When this field is optional.None, it
	// means that the probe failed to discover the IP address ASN.
	IPAddressASN optional.Value[int64]

	// IPAddressBogon is true if IPAddress is a bogon.
	IPAddressBogon optional.Value[bool]

	// The following fields are optional.Some when you process the TCP
	// connect events contained inside an OONI measurement:

	// EndpointTransactionID is the transaction ID used by this endpoint.
	EndpointTransactionID optional.Value[int64]

	// EndpointProto is either "tcp" or "udp".
	EndpointProto optional.Value[string]

	// EndpointPort is the port used by this endpoint.
	EndpointPort optional.Value[string]

	// EndpointAddress is "${IPAddress}:${EndpointPort}" where "${IPAddress}" is
	// quoted using "[" and "]" when the protocol family is IPv6.
	EndpointAddress optional.Value[string]

	// TCPConnectFailure is the optional TCP connect failure.
	TCPConnectFailure optional.Value[string]

	// The following fields are optional.Some when you process the TLS
	// handshake events contained inside an OONI measurement:

	// TLSHandshakeFailure is the optional TLS handshake failure.
	TLSHandshakeFailure optional.Value[string]

	// TLSServerName is the optional TLS server name used by the TLS handshake.
	TLSServerName optional.Value[string]

	// The following fields are optional.Some when you process the HTTP round
	// trip events contained inside an OONI measurement:

	// HTTPRequestURL is the HTTP request URL.
	HTTPRequestURL optional.Value[string]

	// HTTPFailure is the error that occurred during the HTTP round trip.
	HTTPFailure optional.Value[string]

	// HTTPResponseStatusCode is the response status code.
	HTTPResponseStatusCode optional.Value[int64]

	// HTTPResponseBodyLength is the length of the response body.
	HTTPResponseBodyLength optional.Value[int64]

	// HTTPResponseBodyIsTruncated indicates whether the response body was truncated.
	HTTPResponseBodyIsTruncated optional.Value[bool]

	// HTTPResponseHeadersKeys contains maps response headers keys to true.
	HTTPResponseHeadersKeys optional.Value[map[string]bool]

	// HTTPResponseLocation contains the location we're redirected to.
	HTTPResponseLocation optional.Value[string]

	// HTTPResponseTitle contains the response title.
	HTTPResponseTitle optional.Value[string]

	// HTTPResponseIsFinal is true if the status code is 2xx, 4xx, or 5xx.
	HTTPResponseIsFinal optional.Value[bool]

	// The following fields are optional.Some when you process the control information
	// contained inside a measurement and there's information available:

	// ControlDNSDomain is the domain used by the control for its DNS lookup. This field is
	// optional.Some only when the domain used by the control matches the domain used by the
	// probe. So, we won't see this record for redirect endpoints using another domain.
	ControlDNSDomain optional.Value[string]

	// ControlDNSLookupFailure is the corresponding control DNS lookup failure.
	ControlDNSLookupFailure optional.Value[string]

	// ControlTCPConnectFailure is the control's TCP connect failure.
	ControlTCPConnectFailure optional.Value[string]

	// MatchWithControlIPAddress is true if also the control resolved this IP address.
	MatchWithControlIPAddress optional.Value[bool]

	// MatchWithControlIPAddressASN is true if the ASN associated to IPAddress
	// is one of the ASNs obtained by mapping the TH-resolved IP addresses to ASNs.
	MatchWithControlIPAddressASN optional.Value[bool]

	// ControlTLSHandshakeFailure is the control's TLS handshake failure.
	ControlTLSHandshakeFailure optional.Value[string]

	// ControlHTTPFailure is the HTTP failure seen by the control.
	ControlHTTPFailure optional.Value[string]

	// ControlHTTPResponseStatusCode is the status code seen by the control.
	ControlHTTPResponseStatusCode optional.Value[int64]

	// ControlHTTPResponseBodyLength contains the control HTTP response body length.
	ControlHTTPResponseBodyLength optional.Value[int64]

	// ControlHTTPResponseHeadersKeys contains the response headers keys.
	ControlHTTPResponseHeadersKeys optional.Value[map[string]bool]

	// ControlHTTPResponseTitle contains the title seen by the control.
	ControlHTTPResponseTitle optional.Value[string]
}

// WebObservationsContainer contains [*WebObservations].
//
// The zero value of this struct is not ready to use, please use [NewWebObservationsContainer].
type WebObservationsContainer struct {
	// DNSLookupFailures maps transaction IDs to DNS lookup failures.
	//
	// Note that DNSLookupFailures and KnownTCPEndpoints share the same transaction
	// ID space, i.e., you can't see the same transaction ID in both. Transaction IDs
	// are strictly positive unique numbers within the same OONI measurement. Note
	// that the A and AAAA events for the same DNS lookup uses the same transaction ID
	// until we fix the https://github.com/ooni/probe/issues/2624 issue. For this
	// reason DNSLookupFailure and DNSLookupSuccesses MUST be slices.
	DNSLookupFailures []*WebObservation

	// DNSLookupSuccesses contains all the successful transactions.
	DNSLookupSuccesses []*WebObservation

	// KnownTCPEndpoints maps transaction IDs to TCP observations.
	KnownTCPEndpoints map[int64]*WebObservation

	// input is the OPTIONAL input of the measurement.
	input optional.Value[string]

	// knownIPAddresses is an internal field that maps an IP address to the
	// corresponding DNS observation that discovered it.
	knownIPAddresses map[string]*WebObservation
}

// NewWebObservationsContainer constructs a [*WebObservationsContainer].
func NewWebObservationsContainer(input optional.Value[string]) *WebObservationsContainer {
	return &WebObservationsContainer{
		DNSLookupFailures:  []*WebObservation{},
		DNSLookupSuccesses: []*WebObservation{},
		KnownTCPEndpoints:  map[int64]*WebObservation{},
		input:              input,
		knownIPAddresses:   map[string]*WebObservation{},
	}
}

// IngestDNSLookupEvents ingests DNS lookup events from a OONI measurement. You MUST
// ingest DNS lookup events before ingesting any other kind of event.
func (c *WebObservationsContainer) IngestDNSLookupEvents(evs ...*model.ArchivalDNSLookupResult) {
	c.ingestDNSLookupFailures(evs...)
	c.ingestDNSLookupSuccesses(evs...)
}

func (c *WebObservationsContainer) ingestDNSLookupFailures(evs ...*model.ArchivalDNSLookupResult) {
	for _, ev := range evs {
		// skip all the succesful queries
		if ev.Failure == nil {
			continue
		}

		// create record
		obs := &WebObservation{
			Input:            c.input,
			DNSTransactionID: optional.Some(ev.TransactionID),
			DNSDomain:        optional.Some(ev.Hostname),
			DNSLookupFailure: optional.Some(utilsStringPointerToString(ev.Failure)),
			DNSQueryType:     optional.Some(ev.QueryType),
			DNSEngine:        optional.Some(ev.Engine),
		}

		// add record
		c.DNSLookupFailures = append(c.DNSLookupFailures, obs)
	}
}

func (c *WebObservationsContainer) ingestDNSLookupSuccesses(evs ...*model.ArchivalDNSLookupResult) {
	for _, ev := range evs {
		// skip all the failed queries
		if ev.Failure != nil {
			continue
		}

		// walk through the answers
		utilsForEachIPAddress(ev.Answers, func(ipAddr string) {
			// create the record
			obs := &WebObservation{
				Input:            c.input,
				DNSTransactionID: optional.Some(ev.TransactionID),
				DNSDomain:        optional.Some(ev.Hostname),
				DNSLookupFailure: optional.Some(""),
				DNSQueryType:     optional.Some(ev.QueryType),
				DNSEngine:        optional.Some(ev.Engine),
				IPAddress:        optional.Some(ipAddr),
				IPAddressASN:     utilsGeoipxLookupASN(ipAddr),
				IPAddressBogon:   optional.Some(netxlite.IsBogon(ipAddr)),
			}

			// add record
			c.DNSLookupSuccesses = append(c.DNSLookupSuccesses, obs)

			// store the first lookup that resolved this address
			if _, found := c.knownIPAddresses[ipAddr]; !found {
				c.knownIPAddresses[ipAddr] = obs
			}
		})
	}
}

// IngestTCPConnectEvents ingests TCP connect events from a OONI measurement. You MUST ingest
// these events after DNS events and before any other kind of events.
func (c *WebObservationsContainer) IngestTCPConnectEvents(evs ...*model.ArchivalTCPConnectResult) {
	for _, ev := range evs {
		// create or fetch a record
		obs, found := c.knownIPAddresses[ev.IP]
		if !found {
			obs = &WebObservation{
				Input:          c.input,
				IPAddress:      optional.Some(ev.IP),
				IPAddressASN:   utilsGeoipxLookupASN(ev.IP),
				IPAddressBogon: optional.Some(netxlite.IsBogon(ev.IP)),
			}
		}

		// clone the record because the same IP address MAY belong
		// to multiple endpoints across the same measurement
		//
		// while there also fill endpoint specific info
		portString := strconv.Itoa(ev.Port)
		obs = &WebObservation{
			Input:                 c.input,
			DNSTransactionID:      obs.DNSTransactionID,
			DNSDomain:             obs.DNSDomain,
			DNSLookupFailure:      obs.DNSLookupFailure,
			IPAddress:             obs.IPAddress,
			IPAddressASN:          obs.IPAddressASN,
			IPAddressBogon:        obs.IPAddressBogon,
			EndpointTransactionID: optional.Some(ev.TransactionID),
			EndpointProto:         optional.Some("tcp"),
			EndpointPort:          optional.Some(portString),
			EndpointAddress:       optional.Some(net.JoinHostPort(ev.IP, portString)),
			TCPConnectFailure:     optional.Some(utilsStringPointerToString(ev.Status.Failure)),
		}

		// register the observation
		c.KnownTCPEndpoints[ev.TransactionID] = obs
	}
}

// IngestTLSHandshakeEvents ingests TLS handshake events from a OONI measurement. You MUST
// ingest these events after ingesting TCP connect events.
func (c *WebObservationsContainer) IngestTLSHandshakeEvents(evs ...*model.ArchivalTLSOrQUICHandshakeResult) {
	for _, ev := range evs {
		// find the corresponding obs
		obs, found := c.KnownTCPEndpoints[ev.TransactionID]
		if !found {
			continue
		}

		// update the record
		obs.TLSHandshakeFailure = optional.Some(utilsStringPointerToString(ev.Failure))
		obs.TLSServerName = optional.Some(ev.ServerName)
	}
}

// IngestHTTPRoundTripEvents ingests HTTP round trip events from a OONI measurement. You
// MUST ingest these events after ingesting TCP connect events.
func (c *WebObservationsContainer) IngestHTTPRoundTripEvents(evs ...*model.ArchivalHTTPRequestResult) {
	for _, ev := range evs {
		// find the corresponding obs
		obs, found := c.KnownTCPEndpoints[ev.TransactionID]
		if !found {
			continue
		}

		// start updating the record
		obs.HTTPRequestURL = optional.Some(ev.Request.URL)
		obs.HTTPFailure = optional.Some(utilsStringPointerToString(ev.Failure))

		// consider the response authoritative only in case of success
		if ev.Failure != nil {
			continue
		}
		obs.HTTPResponseStatusCode = optional.Some(ev.Response.Code)
		obs.HTTPResponseBodyLength = optional.Some(int64(len(ev.Response.Body)))
		obs.HTTPResponseBodyIsTruncated = optional.Some(ev.Request.BodyIsTruncated)
		obs.HTTPResponseHeadersKeys = utilsExtractHTTPHeaderKeys(ev.Response.Headers)
		obs.HTTPResponseTitle = optional.Some(measurexlite.WebGetTitle(string(ev.Response.Body)))
		obs.HTTPResponseLocation = utilsExtractHTTPLocation(ev.Response.Headers)
		obs.HTTPResponseIsFinal = utilsDetermineWhetherHTTPResponseIsFinal(ev.Response.Code)
	}
}

// IngestControlMessages ingests the control request and response. You MUST call
// this method last, after you've ingested all the other measurement events.
//
// This method fails if req.HTTPRequest is not a valid serialized URL.
func (c *WebObservationsContainer) IngestControlMessages(req *model.THRequest, resp *model.THResponse) error {
	URL, err := url.Parse(req.HTTPRequest)
	if err != nil {
		return err
	}
	inputDomain := URL.Hostname()

	c.controlXrefDNSQueries(inputDomain, resp)
	c.controlMatchDNSLookupResults(inputDomain, resp)
	c.controlXrefTCPIPFailures(resp)
	c.controlXrefTLSFailures(resp)
	c.controlSetHTTPFinalResponseExpectation(resp)

	return nil
}

func (c *WebObservationsContainer) controlXrefDNSQueries(inputDomain string, resp *model.THResponse) {
	for _, obs := range c.DNSLookupFailures {
		// skip cases where the domain is different
		if obs.DNSDomain.Unwrap() != inputDomain {
			continue
		}

		// register the corresponding DNS domain used by the control
		obs.ControlDNSDomain = optional.Some(inputDomain)

		// register the corresponding DNS lookup failure
		obs.ControlDNSLookupFailure = optional.Some(utilsStringPointerToString(resp.DNS.Failure))
	}
}

func (c *WebObservationsContainer) controlMatchDNSLookupResults(inputDomain string, resp *model.THResponse) {
	// map out all the IP addresses resolved by the TH
	thAddrMap := make(map[string]bool)
	for _, addr := range resp.DNS.Addrs {
		thAddrMap[addr] = true
	}

	// (re)map out all the ASNs discovered by the TH using the same ASN
	// database used to build the probe's ASN mapping
	thASNMap := make(map[int64]bool)
	for _, addr := range resp.DNS.Addrs {
		if asn, _, err := geoipx.LookupASN(addr); err == nil && asn != 0 {
			thASNMap[int64(asn)] = true
		}
	}

	// walk through the list of known TCP observations
	for _, obs := range c.KnownTCPEndpoints {
		// obtain the domain from which we obtained the endpoint's address
		domain := obs.DNSDomain.UnwrapOr("")

		// obtain the IP address
		addr := obs.IPAddress.Unwrap()

		// handle the case in which the IP address has been provided by the control, which
		// is a case where the domain is empty and the IP address is in thAddrMap
		if domain == "" && thAddrMap[addr] {
			obs.MatchWithControlIPAddress = optional.Some(true)
			obs.MatchWithControlIPAddressASN = optional.Some(true)
			continue
		}

		// skip entries using a different domain than the one used by the TH
		if domain == "" || domain != inputDomain {
			continue
		}

		// register the control DNS domain
		obs.ControlDNSDomain = optional.Some(domain)

		// register whether the control failed and skip in such a case
		obs.ControlDNSLookupFailure = optional.Some(utilsStringPointerToString(resp.DNS.Failure))
		if resp.DNS.Failure != nil {
			continue
		}

		// compute whether also the TH observed this addr
		obs.MatchWithControlIPAddress = optional.Some(thAddrMap[addr])

		// cannot continue unless we know the probe's ASN
		ourASN := obs.IPAddressASN.UnwrapOr(0)
		if ourASN <= 0 {
			continue
		}

		// register whether there is matching in terms of the ASNs
		obs.MatchWithControlIPAddressASN = optional.Some(thASNMap[ourASN])
	}
}

func (c *WebObservationsContainer) controlXrefTCPIPFailures(resp *model.THResponse) {
	for _, obs := range c.KnownTCPEndpoints {
		endpointAddress := obs.EndpointAddress.Unwrap()

		// skip when we don't have a record
		tcp, found := resp.TCPConnect[endpointAddress]
		if !found {
			continue
		}

		// save the corresponding control result
		obs.ControlTCPConnectFailure = optional.Some(utilsStringPointerToString(tcp.Failure))
	}
}

func (c *WebObservationsContainer) controlXrefTLSFailures(resp *model.THResponse) {
	for _, obs := range c.KnownTCPEndpoints {
		endpointAddress := obs.EndpointAddress.Unwrap()

		// skip entries without a TLS server name (e.g., entries where we did not TLS handshake)
		//
		// this check should be ~first to exclude cases w/o TLS
		if obs.TLSServerName.IsNone() {
			continue
		}
		serverName := obs.TLSServerName.Unwrap()

		// skip when we don't have a record
		tls, found := resp.TLSHandshake[endpointAddress]
		if !found {
			continue
		}

		// skip when the server name does not match
		if tls.ServerName != serverName {
			continue
		}

		// save the corresponding control result
		obs.ControlTLSHandshakeFailure = optional.Some(utilsStringPointerToString(tls.Failure))
	}
}

func (c *WebObservationsContainer) controlSetHTTPFinalResponseExpectation(resp *model.THResponse) {
	for _, obs := range c.KnownTCPEndpoints {
		obs.ControlHTTPFailure = optional.Some(utilsStringPointerToString(resp.HTTPRequest.Failure))

		// leave everything else nil if there was a failure, like we
		// already do when processing the probe events
		if resp.HTTPRequest.Failure != nil {
			continue
		}

		obs.ControlHTTPResponseStatusCode = optional.Some(resp.HTTPRequest.StatusCode)
		obs.ControlHTTPResponseBodyLength = optional.Some(resp.HTTPRequest.BodyLength)
		obs.ControlHTTPResponseHeadersKeys = utilsExtractHTTPHeaderKeys(resp.HTTPRequest.Headers)
		obs.ControlHTTPResponseTitle = optional.Some(resp.HTTPRequest.Title)
	}
}
