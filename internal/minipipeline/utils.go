package minipipeline

import (
	"strconv"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
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

func utilsGeoipxLookupASN(ipAddress string) optional.Value[int64] {
	if asn, _, err := geoipx.LookupASN(ipAddress); err == nil && asn > 0 {
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

func utilsForEachIPAddress(answers []model.ArchivalDNSAnswer, fx func(ipAddr string)) {
	for _, ans := range answers {
		// extract the IP address we resolved
		switch ans.AnswerType {
		case "A":
			fx(ans.IPv4)
		case "AAAA":
			fx(ans.IPv6)
		default:
			// nothing
		}
	}
}

func utilsEngineIsGetaddrinfo(engine optional.Value[string]) bool {
	switch engine.UnwrapOr("") {
	case "getaddrinfo", "golang_net_resolver":
		return true
	default:
		return false
	}
}

func utilsExtractTagDepth(tags []string) optional.Value[int64] {
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "depth=") {
			continue
		}
		tag = strings.TrimPrefix(tag, "depth=")
		value, err := strconv.ParseInt(tag, 10, 64)
		if err != nil {
			continue
		}
		return optional.Some(value)
	}
	return optional.None[int64]()
}

func utilsExtractTagFetchBody(tags []string) optional.Value[bool] {
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "fetch_body=") {
			continue
		}
		tag = strings.TrimPrefix(tag, "fetch_body=")
		return optional.Some(tag == "true")
	}
	return optional.None[bool]()
}

func utilsDNSEngineIsDNSOverHTTPS(obs *WebObservation) bool {
	return obs.DNSEngine.UnwrapOr("") == "doh"
}

func utilsDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" &&
		obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
}
