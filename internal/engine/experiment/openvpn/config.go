package openvpn

//
// Config
//

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ooni/minivpn/vpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	ErrBadBase64Blob = errors.New("wrong base64 encoding")
	BadOONIRunInput  = errors.New("bad oonirun input")
)

// Config contains openvpn experiment configuration.
// TODO(ainghazal): add an optional (and reasonable) truncation threshold for each url (needs to be overriden).
type Config struct {
	URLs        string `ooni:"comma-separated list of extra URLs to fetch through the tunnel"`
	Cipher      string `ooni:"cipher to use"`
	Auth        string `ooni:"auth to use"`
	Obfuscation string `ooni:"obfuscation type for the tunnel"`
	Compress    string `ooni:"compression to use"`
	// Safe_XXX optins are not sent to the backend for archival.
	SafeKey        string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert       string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCa         string `ooni:"ca to connect to the OpenVPN endpoint"`
	SafeLocalCreds bool   `ooni:"whether to use local credentials for the given provider"`
	SafeProxyURI   string `ooni:"obfuscating proxy to be used"` // empty if Obfuscation is "none"
}

var vpnConfigTemplate = `{{ if eq .Config.Obfuscation "obfs4" }}proxy-obfs4 {{ .Config.ProxyURI }}{{ else }}remote {{ .Hostname }} {{ .Port }}{{ end }}
proto {{ .Transport }}
cipher {{ .Config.Cipher }}
auth {{ .Config.Auth }}
{{ if .Config.LocalCreds }}
auth-user-pass /tmp/ooni-vpn-creds{{ end }}
{{ if eq .Config.Compress "comp-lzo-no"}}
comp-lzo no{{ end }}
<ca>
{{ .Config.Ca }}</ca>
{{ if ne .Config.Cert "" }}
<cert>
{{ .Config.Cert }}</cert>{{ end }}
{{ if ne .Config.Key "" }}
<key>
{{ .Config.Key }}</key>{{ end }}`

func vpnExperimentFromURI(uri string) (*model.VPNExperiment, error) {
	ve := &model.VPNExperiment{}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", BadOONIRunInput, err)
	}
	if u.Scheme != "vpn" {
		return nil, fmt.Errorf("%w: %s", BadOONIRunInput, "expected vpn:// uri")
	}
	provider := "unknown"
	openvpn := "openvpn"

	hostParts := strings.Split(u.Hostname(), ".")
	switch len(hostParts) {
	case 2:
		provider = hostParts[1]
		if hostParts[0] != openvpn {
			return nil, fmt.Errorf("%w: unknown proto (%s)", BadOONIRunInput, hostParts[0])
		}
	case 1:
		if hostParts[0] != openvpn {
			return nil, fmt.Errorf("%w: unknown proto (%s)", BadOONIRunInput, hostParts)
		}
	default:
		return nil, fmt.Errorf("%w: %s", BadOONIRunInput, "wrong domain in experiment URI")
	}
	ve.Provider = provider
	ve.Protocol = openvpn

	params := u.Query()

	addr := params.Get("addr")

	addrParts := strings.Split(addr, ":")
	if len(addrParts) != 2 {
		return nil, fmt.Errorf("%w: wrong addr %s", BadOONIRunInput, addr)
	}
	ve.Hostname = addrParts[0]
	ve.Port = addrParts[1]
	obfs := params.Get("obfuscation")
	if obfs == "" {
		obfs = "none"
	}
	ve.Obfuscation = obfs
	tr := params.Get("transport")
	switch tr {
	case "udp", "tcp":
		ve.Transport = tr
	case "":
		return nil, fmt.Errorf("%w: missing transport", BadOONIRunInput)
	default:
		return nil, fmt.Errorf("%w: bad transport %s", BadOONIRunInput, tr)
	}

	return ve, nil
}

// Validate returns true if all the fields for a VPNValidate have valid values.
// TODO(ainghazal): do when VPNExperiment is turned into an interface.
/*
 func (e *model.VPNExperiment) Validate() bool {
 	return true
 }
*/

func extractBase64Blob(val string) (string, error) {
	s := strings.TrimPrefix(val, "base64:")
	if len(s) == len(val) {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, "missing prefix")
	}
	dec, err := base64.URLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, err)
	}
	if len(dec) == 0 {
		return "", nil
	}
	return string(dec), nil
}

func protoToString(val int) string {
	switch val {
	case vpn.UDPMode:
		return "udp"
	case vpn.TCPMode:
		return "tcp"
	default:
		return "unknown"
	}
}
