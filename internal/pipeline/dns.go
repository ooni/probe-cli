package pipeline

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSObservation is a DNS observation made by the probe.
//
// Optional values represent data that may not be there if we do not
// find the expected events. Non-optional data should always be there.
//
// This type is inspired by and adapted from https://github.com/ooni/data
// and adapts the WebObservation type to probe-engine.
type DNSObservation struct {
	// TransactionID is the ID of the transaction.
	TransactionID int64

	// QueryType is the DNS query type (e.g., "A").
	QueryType string

	// QueryHostname is the hostname inside the query.
	QueryHostname string

	// Failure is the failure that occurred.
	Failure Failure

	// Engine is the engined used by the probe to lookup.
	Engine string

	// ResolverAddress contains the resolver endpoint address.
	ResolverAddress string

	// IPAddrs contains the resolved IP addresses.
	IPAddrs []string

	// T0 is when we started performing the lookup.
	T0 float64

	// T is when the lookup completed.
	T float64
}

func (db *DB) addDNSLookups(evs ...*model.ArchivalDNSLookupResult) error {
	for _, ev := range evs {
		dobs, err := db.newDNSObservation(ev.TransactionID)
		if err != nil {
			return err
		}
		dobs.QueryType = ev.QueryType
		dobs.QueryHostname = ev.Hostname
		dobs.Failure = NewFailure(ev.Failure)
		dobs.Engine = ev.Engine
		dobs.ResolverAddress = ev.ResolverAddress
		dobs.T0 = ev.T0
		dobs.T = ev.T

		for _, ans := range ev.Answers {
			switch ans.AnswerType {
			case "A":
				dobs.IPAddrs = append(dobs.IPAddrs, ans.IPv4)

			case "AAAA":
				dobs.IPAddrs = append(dobs.IPAddrs, ans.IPv6)

			default:
				// nothing
			}
		}
	}
	return nil
}

func (db *DB) newDNSObservation(txid int64) (*DNSObservation, error) {
	if _, good := db.WebByTxID[txid]; good {
		return nil, errTransactionAlreadyExists
	}
	dobs := &DNSObservation{
		TransactionID: txid,
	}
	db.DNSByTxID[txid] = dobs
	return dobs, nil
}

func dnsNormalizeEngineName(engine string) string {
	switch engine {
	case "system", "getaddrinfo", "golang_net_resolver", "go":
		return "getaddrinfo"
	default:
		return engine
	}
}
