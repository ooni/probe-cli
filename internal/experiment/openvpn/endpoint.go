package openvpn

import (
	"fmt"
	"math/rand"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

// endpoint is a single endpoint to be probed.
// The information contained in here is not generally not sufficient to complete a connection:
// we need more info, as cipher selection or obfuscating proxy credentials.
type endpoint struct {
	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// Obfuscation is any obfuscation method use to connect to this endpoint.
	// Valid values are: obfs4, none.
	Obfuscation string

	// Port is the Port for this endpoint.
	Port string

	// Protocol is the tunneling protocol (openvpn, openvpn+obfs4).
	Protocol string

	// Provider is a unique label identifying the provider maintaining this endpoint.
	Provider string

	// Transport is the underlying transport used for this endpoint. Valid transports are `tcp` and `udp`.
	Transport string
}

// newEndpointFromInputString constructs an endpoint after parsing an input string.
//
// The input URI is in the form:
// "openvpn://1.2.3.4:443/udp/&provider=tunnelbear&obfs=none"
// "openvpn+obfs4://1.2.3.4:443/tcp/&provider=riseup&obfs=obfs4&cert=deadbeef"
// TODO(ainghazal): implement
func newEndpointFromInputString(input string) endpoint {
	fmt.Println("should parse:", input)
	return endpoint{}
}

// String implements Stringer. This is a subset of the input URI scheme.
func (e *endpoint) String() string {
	var proto string
	if e.Obfuscation == "obfs4" {
		proto = e.Protocol + "+obfs4"
	} else {
		proto = e.Protocol
	}
	return fmt.Sprintf("%s://%s:%s/%s", proto, e.IPAddr, e.Port, e.Transport)
}

// AsInputURI is a string representation of this endpoint. It contains more information than the endpoint itself.
func (e *endpoint) AsInputURI() string {
	provider := e.Provider
	if provider == "" {
		provider = "unknown"
	}
	i := fmt.Sprintf("%s/?provider=%s", e.String(), provider)
	return i
}

// endpointList is a list of endpoints.
type endpointList []endpoint

// allEndpoints contains a subset of known endpoints to be used if no input is passed to the experiment.
// This is a hardcoded list for now, but the idea is that we can receive this from the check-in api in the future.
// In any case, having hardcoded endpoints is good as a fallback for the cases in which we cannot contact
// OONI's backend.
var allEndpoints = endpointList{
	{
		Provider:  "riseup",
		IPAddr:    "51.15.187.53",
		Port:      "1194",
		Protocol:  "openvpn",
		Transport: "tcp",
	},
	{
		Provider:  "riseup",
		IPAddr:    "51.15.187.53",
		Port:      "1194",
		Protocol:  "openvpn",
		Transport: "udp",
	},
}

// Shuffle returns a shuffled copy of the endpointList.
func (e endpointList) Shuffle() endpointList {
	rand.Shuffle(len(e), func(i, j int) {
		e[i], e[j] = e[j], e[i]
	})
	return e
}

// TODO(ainghazal): this is extremely hacky, but it's a first step
// until we manage to have the check-in API handing credentials.
// Do note that these certificates will expire ca. Apr 6 2024
// OTOH, yes, I do understand the risks of exposing key material
// on a public git repo. Thanks for caring.
var defaultOptionsByProvider = map[string]*vpnconfig.OpenVPNOptions{
	"riseup": {
		Auth:   "SHA512",
		Cipher: "AES-256-GCM",
		CA: []byte(`-----BEGIN CERTIFICATE-----
MIIBYjCCAQigAwIBAgIBATAKBggqhkjOPQQDAjAXMRUwEwYDVQQDEwxMRUFQIFJv
b3QgQ0EwHhcNMjExMTAyMTkwNTM3WhcNMjYxMTAyMTkxMDM3WjAXMRUwEwYDVQQD
EwxMRUFQIFJvb3QgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQxOXBGu+gf
pjHzVteGTWL6XnFxtEnKMFpKaJkA/VOHmESzoLsZRQxt88GssxaqC01J17idQiqv
zgNpedmtvFtyo0UwQzAOBgNVHQ8BAf8EBAMCAqQwEgYDVR0TAQH/BAgwBgEB/wIB
ATAdBgNVHQ4EFgQUZdoUlJrCIUNFrpffAq+LQjnwEz4wCgYIKoZIzj0EAwIDSAAw
RQIgfr3w4tnRG+NdI3LsGPlsRktGK20xHTzsB3orB0yC6cICIQCB+/9y8nmSStfN
VUMUyk2hNd7/kC8nL222TTD7VZUtsg==
-----END CERTIFICATE-----`),
		Key: []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqprWmGJKLgZBFbdJUEMzKpJkWnVLoALSTTZqmzX8vuQD7W2J
HbwptiD+a7qCvikpX+bsRb9b84VctYZq/tnLwqRVeDfoega+pGws0KGMo74KWlUZ
1k+AjCbqxWJPlaYKNkDXAInsc6alEv09ZbeuGGpWtQSVpP+sudgDf9JpIEsnTLSK
5t0i1QX/53Vltr+omLCqd52a2bUxK8WNIwtsSs9lLGrpKTVJ1zKDpVBNmNFgahpk
kX5KAkoS0TVzBLPwNNq14GLnTd6YnJ66m9k5iiUBbML81bnE3qbxG7C/qoXIP4eH
0Y7RDBB0dlZ8PTBjeg0pnEtPF5MrglVRVeUimQIDAQABAoIBAGVgMspEBa5Jmx0r
V44xEFNov+ccsf54Dr1A66IlN3W7CjZok0SvDd4ixuv+3TfgP6y0DIv5hMs04P0g
za14f+K+Qed42VTBc0FC4nJqvKaEA6Tf0sWNYmZlrbXykDXtfz3z046HZpDmYkrh
Xj12IyZw8esIuV9daibYnGO1BTDhXy/B53zDjx6wYMDC3DFVa2gLSRWONtMnCYY8
Hw7FbaP1Jxs6sNS/AKVZZo4SyBL1te80HN9Wo2syDmdc3o3aBMkCY9+u560dj8+5
4xvn+d8ojp91Ts3o33DB6PY88r2UTg00ejGMn8LN7dCnZDO2mQ98nczKfcpYL0nW
CKxG6AECgYEA0L9XBdfps60nKNS5n4+rNvtYvhkvHOKkz7wFmYSo6r8M1ID3m7g3
x6wwTY9MrlSPPsF9x6GnrmGIGIZsc8lNRuYFq/yemNhKfMi6KU9wnjqVQYDSg9S2
fq4lutPxbeiQmSx5WYtjeaJXzTAzx9jT6t8QiAUXM06QgPPjLK7G+ZkCgYEA0Tku
iSz8Y2uHyBWOYFTIaEvvyCEJqyZ+hMgVRRgN7QzDjP4VUVmQClwdK7JIPNBaIf6V
Gvi+CXgb/oDMrcduMM4ZGoVN1ttpC3htn7qn35+38VsYPD3hgmF7r3WFSxoBd0vj
Yh7rO4tQo91tm0DkCs+NZvNRrFr1yL/VAHnDEQECgYAi3XJpdXCBJBCAT1dZgSN1
oXFm/snRp0EjuSGuTGvyGUrJS2kPxyr53JaMvbxu+YybTLH3X9aj14Jlpj4C8MJJ
by3PVfgfSzDVuqjtMWl75Aj90chXYHXCns+Kbs/KLafJDZaPECrjK+xCRyS+4kYy
2mLmdQM0/JBCGXn+AosVMQKBgEFgy/DjlM6AaIKWcdIaTDGDIR95a2sG8VwOpc7c
cGWVqnmhYAn2obMLC7Z+1GHkfXXH9tHhzoho9t51YwAepIktrdyCsUslbtK9xAu4
qQKRB0qtO4p/j7tNOPggEhHgw3qCxUABB2Ko6v75j2mHQns6ViZIfEoOdmVPxICM
i+8BAoGBAIn2RfbrcbrH3jZKNkklUMg7fU8K0w7PQ7kCR8vql4chA6Lzf7BO2uUb
+KoDT6FZRI7vSZFqMmcqs/LEEPsBYtr0GKNmH3pcFHQ5HvfZdXkMILADHj0gxwZ0
ng58SKQl8yU3B3wIoBOV+YEo8D+pLzlmH9XTRUl6sX0NvX1xeP/d
-----END RSA PRIVATE KEY-----`),
		Cert: []byte(`-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIRAPB+TCOgYy6vkH4CTz0UDdMwCgYIKoZIzj0EAwIwMzEx
MC8GA1UEAwwoTEVBUCBSb290IENBIChjbGllbnQgY2VydGlmaWNhdGVzIG9ubHkh
KTAeFw0yNDAyMjgyMTU4MTJaFw0yNDA0MDMyMTU4MTJaMBQxEjAQBgNVBAMTCVVO
TElNSVRFRDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKqa1phiSi4G
QRW3SVBDMyqSZFp1S6AC0k02aps1/L7kA+1tiR28KbYg/mu6gr4pKV/m7EW/W/OF
XLWGav7Zy8KkVXg36HoGvqRsLNChjKO+ClpVGdZPgIwm6sViT5WmCjZA1wCJ7HOm
pRL9PWW3rhhqVrUElaT/rLnYA3/SaSBLJ0y0iubdItUF/+d1Zba/qJiwqnedmtm1
MSvFjSMLbErPZSxq6Sk1Sdcyg6VQTZjRYGoaZJF+SgJKEtE1cwSz8DTateBi503e
mJyeupvZOYolAWzC/NW5xN6m8Ruwv6qFyD+Hh9GO0QwQdHZWfD0wY3oNKZxLTxeT
K4JVUVXlIpkCAwEAAaNnMGUwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsG
AQUFBwMCMB0GA1UdDgQWBBSWz2IckC83grgDEuwOSfHtxBy3OTAfBgNVHSMEGDAW
gBR9SmLY/ytJxHm2orHcjj5jB1yo/jAKBggqhkjOPQQDAgNIADBFAiEA/J7Y0zfR
QxEBzQJEfSjT3Q9/cJkZJ11KwehMQYJTwGICIEMg3zaOg2XUlUg6jshTYx7S9xfE
vly8wNG42zeRWAXz
-----END CERTIFICATE-----`),
	},
}

// getVPNConfig gets a properly configured [*vpnconfig.Config] object for the given endpoint.
// To obtain that, we merge the endpoint specific configuration with base options.
// These base options are for the moment hardcoded. In the future we will want to be smarter
// about getting information for different providers.
func getVPNConfig(tracer *vpntracex.Tracer, endpoint *endpoint) (*vpnconfig.Config, error) {
	// TODO(ainghazal): use options merge (pending PR)
	provider := endpoint.Provider
	// TODO(ainghazal): return error if provider unknown. we're in the happy path for now.
	baseOptions := defaultOptionsByProvider[provider]

	cfg := vpnconfig.NewConfig(
		vpnconfig.WithOpenVPNOptions(
			&vpnconfig.OpenVPNOptions{
				// endpoint-specific options.
				Remote: endpoint.IPAddr,
				Port:   endpoint.Port,
				Proto:  vpnconfig.Proto(endpoint.Transport),

				// options coming from the default values for the provider,
				// to switch to values provided by the check-in API in the future.
				CA:     baseOptions.CA,
				Cert:   baseOptions.Cert,
				Key:    baseOptions.Key,
				Cipher: baseOptions.Cipher,
				Auth:   baseOptions.Auth,
			},
		),
		vpnconfig.WithHandshakeTracer(tracer))

	// TODO: validate options here and return an error.
	return cfg, nil
}
