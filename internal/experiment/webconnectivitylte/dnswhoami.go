package webconnectivitylte

import (
	"github.com/ooni/probe-cli/v3/internal/webconnectivityalgo"
)

// DNSWhoamiSingleton is the DNSWhoamiService singleton.
var DNSWhoamiSingleton = webconnectivityalgo.NewDNSWhoamiService()
