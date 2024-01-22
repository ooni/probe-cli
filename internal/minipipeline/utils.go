package minipipeline

import (
	"strconv"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

func utilsStringPointerToString(failure *string) (out string) {
	if failure != nil {
		out = *failure
	}
	return
}

func utilsGeoipxLookupASN(ipAddress string, lookupper model.GeoIPASNLookupper) optional.Value[int64] {
	if asn, _, err := lookupper.LookupASN(ipAddress); err == nil && asn > 0 {
		return optional.Some(int64(asn))
	}
	return optional.None[int64]()
}

func utilsExtractHTTPHeaderKeys[T ~string](input map[string]T) optional.Value[map[string]bool] {
	output := make(map[string]bool)
	for key := range input {
		output[key] = true
	}
	return optional.Some(output)
}

func utilsExtractHTTPLocation(input map[string]model.ArchivalScrubbedMaybeBinaryString) optional.Value[string] {
	for key, value := range input {
		if strings.ToLower(key) == "location" {
			return optional.Some(string(value))
		}
	}
	return optional.None[string]()
}

func utilsDetermineWhetherHTTPResponseIsFinal(status int64) optional.Value[bool] {
	switch status / 100 {
	case 2, 4, 5:
		return optional.Some(true)
	default:
		return optional.Some(false)
	}
}

func utilsResolvedAddresses(answers []model.ArchivalDNSAnswer) (addrs []string) {
	for _, ans := range answers {
		// extract the IP address we resolved
		switch ans.AnswerType {
		case "A":
			addrs = append(addrs, ans.IPv4)
		case "AAAA":
			addrs = append(addrs, ans.IPv6)
		default:
			// nothing
		}
	}
	return
}

func utilsEngineIsGetaddrinfo(engine optional.Value[string]) bool {
	switch engine.UnwrapOr("") {
	case "getaddrinfo", "golang_net_resolver":
		return true
	default:
		return false
	}
}

func utilsExtractTagDepth(tags []string) (result optional.Value[int64]) {
	result = optional.None[int64]()
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "depth=") {
			continue
		}
		tag = strings.TrimPrefix(tag, "depth=")
		value, err := strconv.ParseInt(tag, 10, 64)
		if err != nil {
			continue
		}
		result = optional.Some(value)
	}
	return
}

func utilsExtractTagFetchBody(tags []string) (result optional.Value[bool]) {
	result = optional.None[bool]()
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "fetch_body=") {
			continue
		}
		tag = strings.TrimPrefix(tag, "fetch_body=")
		result = optional.Some(tag == "true")
	}
	return
}

func utilsDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" &&
		obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
}

func utilsDNSEngineIsDNSOverHTTPS(obs *WebObservation) bool {
	return obs.DNSEngine.UnwrapOr("") == "doh"
}

// utilsTCPConnectFailureSeemsMisconfiguredIPv6 returns whether IPv6 seems to be
// misconfigured for this specific TCP connect attemptt.
//
// See https://github.com/ooni/probe/issues/2284 for more info.
func utilsTCPConnectFailureSeemsMisconfiguredIPv6(obs *WebObservation) bool {
	switch obs.TCPConnectFailure.UnwrapOr("") {
	case netxlite.FailureNetworkUnreachable, netxlite.FailureHostUnreachable:
		isv6, err := netxlite.IsIPv6(obs.IPAddress.UnwrapOr(""))
		return err == nil && isv6

	default: // includes the case of missing TCPConnectFailure
		return false
	}
}
