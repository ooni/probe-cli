package pipeline

import (
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// TODO(bassosimone): we are not extracting the data required to produce
// duplicate-response based failure, which is also not implemented inside
// of the v0.5 codebase, hence we're not regressing here.

// Analysis aggregates the results of several analysis algorithms.
//
// Some values are optional. When a value IsNone, it means we could not run the
// corresponding analysis algorithm, so we don't basically know.
//
// All the methods in this struct ARE NOT goroutine safe.
//
// The zero value of this struct is ready to use.
type Analysis struct {
	// DNSUnexpectedAddr lists all the DNS transaction IDs with unexpected addrs (i.e., IP
	// addrs that have not have been also resolved by the TH).
	DNSUnexpectedAddr []int64

	// DNSUnexpectedAddrASN lists all the DNS transaction IDS with unexpected addrs ASNs.
	DNSUnexpectedAddrASN []int64

	// DNSUnexpectedBogon lists all the DNS transaction IDs for which we saw unexpected bogons.
	DNSUnexpectedBogon []int64

	// DNSUnexpectedFailure lists all the DNS transaction IDs containing unexpected DNS lookup failures.
	DNSUnexpectedFailure []int64

	// DNSWithTLSHandshakeFailureAddr lists all the DNS transaction IDs containing IP addresses
	// for which the TH could perform a successful TLS handshake where the probe failed.
	DNSWithTLSHandshakeFailureAddr []int64

	// DNSExperimentFailure is a backward-compatibility value that contains the
	// failure obtained when using getaddrinfo for the URL's domain
	DNSExperimentFailure optional.Value[Failure]

	// HTTPDiffBodyProportionFactor is the ratio of the two final bodies.
	HTTPDiffBodyProportionFactor optional.Value[float64]

	// HTTPDiffTitleMatch indicates whether the titles have common words in them.
	HTTPDiffTitleMatch optional.Value[bool]

	// HTTPDiffStatusCodeMatch indicates whether the status code matches taking into
	// account some false positives that may arise.
	HTTPDiffStatusCodeMatch optional.Value[bool]

	// HTTPDiffUncommonHeadersMatch indicates whether uncommon headers match.
	HTTPDiffUncommonHeadersMatch optional.Value[bool]

	// HTTPExperimentFailure is a backward-compatibility value that contains the
	// failure obtained for the final HTTP request made by the probe
	HTTPExperimentFailure optional.Value[Failure]

	// HTTPUnexpectedFailure contains all the endpoint transaction IDs where
	// the TH succeded while the probe failed to fetch a response
	HTTPUnexpectedFailure []int64

	// TCPUnexpectedFailure contains all the endpoint transaction IDs where the TH succeeded
	// while the probe failed to connect (excluding obvious IPv6 issues).
	TCPUnexpectedFailure []int64

	// TLSUnexpectedFailure is like TCPUnexpectedFailure but for TLS.
	TLSUnexpectedFailure []int64
}

// ComputeAllValues computes all the analysis values using the DB.
func (ax *Analysis) ComputeAllValues(db *DB) {
	// DNS
	ax.ComputeDNSExperimentFailure(db)
	ax.ComputeDNSBogon(db)
	ax.ComputeDNSUnexpectedFailure(db)
	ax.ComputeDNSUnexpectedAddr(db)
	ax.ComputeDNSUnexpectedAddrASN(db)
	ax.ComputeDNSWithTLSHandshakeFailureAddr(db)

	// TCP/IP
	ax.ComputeTCPUnexpectedFailure(db)

	// TLS
	ax.ComputeTLSUnexpectedFailure(db)

	// HTTP (core)
	ax.ComputeHTTPUnexpectedFailure(db)
	ax.ComputeHTTPExperimentFailure(db)

	// HTTP (diff)
	ax.ComputeHTTPDiffBodyProportionFactor(db)
	ax.ComputeHTTPDiffStatusCodeMatch(db)
	ax.ComputeHTTPDiffUncommonHeadersMatch(db)
	ax.ComputeHTTPDiffTitleMatch(db)
}
