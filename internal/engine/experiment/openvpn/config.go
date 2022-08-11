package openvpn

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ooni/minivpn/vpn"
)

var (
	ErrBadBase64Blob = errors.New("wrong base64 encoding")
	BadOONIRunInput  = errors.New("bad oonirun input")
)

var vpnConfigTemplate = `remote {{ .Hostname }} {{ .Port }}
proto {{ .Transport }}
cipher {{ .Config.Cipher }}
auth {{ .Config.Auth }}
<ca>
{{ .Config.Ca }}</ca>
<cert>
{{ .Config.Cert }}</cert>
<key>
{{ .Config.Key }}</key>`

// TODO(ainghazal): should share with wireguard -> move to model.
type VPNExperiment struct {
	// Provider is the entity to which the endpoints belong. We might want
	// to keep a list of known providers (for which we have experiments).
	// If the provider is not known to OONI probe, it should be marked as
	// "unknown".
	Provider string
	// Hostname is the Hostname for the VPN Endpoint
	Hostname string
	// Port is the Port for the VPN Endpoint
	Port string
	// Protocol is the VPN protocol: openvpn, wg
	Protocol string
	// Transport is the underlying protocol: udp, tcp
	Transport string
	// Obfuscation is any obfuscation used for the tunnel: none, obfs4, ...
	Obfuscation string
	// Config is a pointer to a VPNExperimentConfig
	Config *VPNExperimentConfig
}

func vpnExperimentFromURI(uri string) (*VPNExperiment, error) {
	ve := &VPNExperiment{}
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
		provider = hostParts[0]
		if hostParts[1] != openvpn {
			return nil, fmt.Errorf("%w: unknown proto (%s)", BadOONIRunInput, hostParts[1])
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
// TODO(ainghazal): implement
func (e *VPNExperiment) Validate() bool {
	return true
}

type VPNExperimentConfig struct {
	Cipher   string
	Auth     string
	Compress string
	Ca       string
	Cert     string
	Key      string
}

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
