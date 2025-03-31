package openvpn

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// defaultOONIEndpoints is the array of hostnames that will return valid
// endpoints to be probed. Do note that this is a workaround for the lack
// of a backend service; if you maintain this experiment in the future please
// feel free to remove this workaround after the probe-services for distributing
// endpoints has been deployed to production.
var defaultOONIEndpoints = []string{
	"a.composer-presenter.com",
	"a.goodyear2dumpster.com",
}

// maxDefaultOONIAddresses is how many IPs to use from the
// set of resolved IPs.
var maxDefaultOONIAddresses = 3

// sampleN takes max n elements sampled ramdonly from the array a.
func sampleN(a []string, n int) []string {
	if n > len(a) {
		n = len(a)
	}
	rand.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a[:n]
}

// resolveOONIAddresses returns a max of maxDefaultOONIAddresses after
// performing DNS resolution. The returned IP addreses exclude possible
// bogons.
func resolveOONIAddresses(logger model.Logger) ([]string, error) {

	// We explicitely resolve with BogonIsError set to false, and
	// later remove bogons from the list. The reason is that in this way
	// we are able to control the rate at which we run tests by adding bogon addresses to the
	// domain records for the test.

	resolver := netx.NewResolver(netx.Config{
		BogonIsError: false,
		Logger:       logger,
		Saver:        nil,
	})

	addrs := []string{}

	var lastErr error

	// Get the set of all IPs for all the hostnames we have.

	for _, hostname := range defaultOONIEndpoints {
		resolved, err := lookupHost(context.Background(), hostname, resolver)
		if err != nil {
			lastErr = err
			continue
		}
		for _, ipaddr := range resolved {
			if !slices.Contains(addrs, ipaddr) {
				addrs = append(addrs, ipaddr)
			}
		}
	}

	// Sample a max of maxDefaultOONIAddresses

	sampled := sampleN(addrs, maxDefaultOONIAddresses)

	// Remove the bogons

	valid := []string{}

	for _, addr := range sampled {
		if !netxlite.IsBogon(addr) {
			valid = append(valid, addr)
		}
	}

	// We only return error if the filtered list is zero len.

	if (len(valid) == 0) && (lastErr != nil) {
		return valid, lastErr
	}

	return valid, nil
}

func lookupHost(ctx context.Context, hostname string, r model.Resolver) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return r.LookupHost(ctx, hostname)
}

// pickOONIOpenVPNTargets crafts targets from the passed array of IP addresses.
func pickOONIOpenVPNTargets(ipaddrList []string) ([]string, error) {
	ipaddrList = []string{"49.12.5.142"}

	if len(ipaddrList) == 0 {
		return []string{}, nil
	}

	// Step 1. Create endpoint list.

	endpoints := []endpoint{}
	for _, ipaddr := range ipaddrList {

		// Probe the canonical 1194/udp and 1194/tcp ports

		endpoints = append(endpoints, endpoint{
			Obfuscation: "none",
			Port:        "1194",
			Protocol:    "openvpn",
			Provider:    "oonivpn",
			IPAddr:      ipaddr,
			Transport:   "tcp",
		})
		endpoints = append(endpoints, endpoint{
			Obfuscation: "none",
			Port:        "1194",
			Protocol:    "openvpn",
			Provider:    "oonivpn",
			IPAddr:      ipaddr,
			Transport:   "udp",
		})

	}

	// Pick one IP from the list and sample on non-standard ports
	// to check if the standard port was filtered.

	extra := ipaddrList[rand.Intn(len(ipaddrList))]
	endpoints = append(endpoints, endpoint{
		Obfuscation: "none",
		Protocol:    "openvpn",
		Provider:    "oonivpn",
		IPAddr:      extra,
		Port:        "53",
		Transport:   "udp",
	})
	endpoints = append(endpoints, endpoint{
		Obfuscation: "none",
		Protocol:    "openvpn",
		Provider:    "oonivpn",
		IPAddr:      extra,
		Port:        "443",
		Transport:   "tcp",
	})

	// Step 2. Create targets for the selected endpoints.

	targets := make([]string, 0)
	for _, e := range endpoints {
		targets = append(targets, e.AsInputURI())
	}
	if len(targets) > 0 {
		return targets, nil
	}
	return nil, fmt.Errorf("cannot find any usable endpoint")
}

func pickFromDefaultOONIOpenVPNConfig() *Config {
	idx := rand.Intn(len(defaultOONIOpenVPNConfig))
	return defaultOONIOpenVPNConfig[idx]
}

var oldDefaultCA = "-----BEGIN CERTIFICATE-----\nMIIDSzCCAjOgAwIBAgIUOPlwhp2s96qqGF5zgLOp0noN2uwwDQYJKoZIhvcNAQEL\nBQAwFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwHhcNMjQwNzMxMTc1MDI3WhcNMzQw\nNzI5MTc1MDI3WjAWMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBALfhmQ6YndIaq9K2ya1HNv9e3DiwKO8X7Ferh8KV\n/Yobs1jPJYfK/l1SZTO97FnIptqxPzGAWuxhS/+4n4ZB2RpszJKdu3sHYNY6lZCR\nw8dtxKYDIS5v/1by6AJk052wV3NWizw1QiawCOJl5cNN5Vb4OpLPvBzrx3IN7jvO\n0HxaaRYIiPdQy++cJ/wqQazTvPYpws0rIAF0A9jxzgsJZoWshg8MhQm9OYIMyZ2C\n4WeuBKU5bR7vqjAQnVH6ZsZ8ZX1UILq++PcuLeDYbg7M5YmT0v0SO+3ealgg48SO\nxqStAawEAXI2sOZqWTvFfXiq9l6Uw2uxPwXnzSO8hjjVqc0CAwEAAaOBkDCBjTAM\nBgNVHRMEBTADAQH/MB0GA1UdDgQWBBRyvhkgys8dIIzvcH7+TlcATT6bGTBRBgNV\nHSMESjBIgBRyvhkgys8dIIzvcH7+TlcATT6bGaEapBgwFjEUMBIGA1UEAwwLRWFz\neS1SU0EgQ0GCFDj5cIadrPeqqhhec4CzqdJ6DdrsMAsGA1UdDwQEAwIBBjANBgkq\nhkiG9w0BAQsFAAOCAQEAPpb2z/wBj9tULuzBQ1j6qkIUCkyH6e+QATHcCcJGWQsU\naeEc1w/qBXaJcRS0ahALXC3d/Tz8R2dAj1sO1HEsfjEs5fv1dKGgeVb1rNuZuUW8\n9xEtUdp3jL3xumcqfxKIwOv8Y1fz+AKGJbbPC3yoHptwMDW9zyaRTQ+McKE7Y497\nFZDF2RWQjgpxwCi7P3cScNBLNtt42TPnj6Up3D6Sj57YVDK9dXbrDj94bwmkQa8s\nl8Mp/PFaFeLNXXuGGVEbIlFuw9RY32vbJ1CrS9rrWlVq9Q17NrAmSYSBi9T19mDh\nMFslRMPBN4Jfd/45V26iW2XMpWCONY5aqAfx+2Oz2g==\n-----END CERTIFICATE-----"
var defaultCA = "-----BEGIN CERTIFICATE-----\nMIIDSzCCAjOgAwIBAgIUS1qTQldi+MiYJPup88OyafWpEP4wDQYJKoZIhvcNAQEL\nBQAwFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwHhcNMjUwMzMxMjA1NjU0WhcNMzUw\nMzI5MjA1NjU0WjAWMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBAOSNM23LqZNDdwlXSMO24q8TFesQES4YcuxYCjs2\nk2Co6afjB0yflRHncixqCQJ8LLDvuXFAHUVSjz/AFXENCE4ur0rMOhwq21ozKv5t\nmThIjM8FdAuos0vODVl3BJ5j/pd0Q90DV1YsN/z2Tzo7kGuIzwLOZ16p+YgRBpO3\n+OPQb0RlJ8fH6dQb7nMxCjjDgBMrUg1DyfWBpbr7m37PwarS4f9QCLWhPPGLIvet\niZRYXF3NOHd+okiQpTZgr+OtxJM+qoPvp3qFsxNNr/nd4qLSf1C0HhWlkhjYfzCq\nEbV1aXph+n0JpMgjeFIX8ynD1bqg1BQsCu6Jgrl5S/+HaiUCAwEAAaOBkDCBjTAM\nBgNVHRMEBTADAQH/MB0GA1UdDgQWBBRVTv58/89R4+QoYHG/ELYrAVvZzzBRBgNV\nHSMESjBIgBRVTv58/89R4+QoYHG/ELYrAVvZz6EapBgwFjEUMBIGA1UEAwwLRWFz\neS1SU0EgQ0GCFEtak0JXYvjImCT7qfPDsmn1qRD+MAsGA1UdDwQEAwIBBjANBgkq\nhkiG9w0BAQsFAAOCAQEAU9rfqNDho8BzFsIHfqNwa3w1YPJrltdTfmoizAdmZ/Oa\nYLJEHYjTp07t5SHbjmjhvjW4juJJzH5Og4dOoe2IfiRspkNqrGtQuWzC4ftk1MUQ\nvIKBH2qfsIA+G+c6dImE3051kwpMYH4O5Jo67ckKVxtKzn+UG6Txr1DHXstlVrgZ\n6ec2/BD+PNs3P4cnKOPisIqZ7HVqzEABSDVw7gcl+VyC2nVoeWqdjwQKyLkrbr4o\nqopHTBuAGT599jlkscBqTYhR63iGt4/ca+APzVD2rSw68I6DZ/IWSgbygz8ENtat\nDbwfwr/eyovKwDxVOzWKCSLX9B+dyrUMzWVLM3UZpw==\n-----END CERTIFICATE-----"
var defaultOONIOpenVPNConfig = []*Config{
	{
		Cipher:      "AES-256-GCM",
		Auth:        "SHA512",
		Compress:    "stub",
		Provider:    "oonivpn",
		Obfuscation: "none",
		SafeCA:      defaultCA,
		SafeCert:    "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIQRifv8pseonQK7PaKN+POIzANBgkqhkiG9w0BAQsFADAW\nMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTAeFw0yNTAzMzEyMDU4MDJaFw0yNzA3MDQy\nMDU4MDJaMBoxGDAWBgNVBAMMD2NsaWVudC1jZXJ0LTAwMTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBAOpT8sUg/hVxzCdTZvCQ2MSVJiQxZjg5f4ZlU60G\nGc1NCVclm307VfE20WxJGizia84DS7F5eRiTw9L7bmneC7gAs4LxvBUCq18XJRN0\nOu2z9nTMwRkj56sg+ZaMEXXi+zT2CNFrsnUGEZMplXiNZBjFMSRwmMxhGf7QbBXM\nQvTFIT9tlJt4m72+NLso+X04Zj92zseIU4CX3pHGwHLR7ttRFcdakPEVF3+yLv1g\nV5yG31zS0Rww7m4leH/v9vb30QpfXX3quy0WMpcna+Rm0eBkkoLMp37N/TTMpbNh\nTddfUeiaZo2Dnr64Y7PqniS/hqqLN7cC99Sz3o0WW5Tmrw0CAwEAAaOBojCBnzAJ\nBgNVHRMEAjAAMB0GA1UdDgQWBBQED1+Y9PEVbQ4ICfwuUOMnaaf55zBRBgNVHSME\nSjBIgBRVTv58/89R4+QoYHG/ELYrAVvZz6EapBgwFjEUMBIGA1UEAwwLRWFzeS1S\nU0EgQ0GCFEtak0JXYvjImCT7qfPDsmn1qRD+MBMGA1UdJQQMMAoGCCsGAQUFBwMC\nMAsGA1UdDwQEAwIHgDANBgkqhkiG9w0BAQsFAAOCAQEAd7elCoS8Hrke1S23afyp\nWl/ypEoOBlyRlbCqtBSMjrpaGGC6bOTt2wK/d8MMECOnEhzhXi386gV9i/78JEcL\nYiHbC9cmvcBKCZTGfwFT/3cGl4flEssgJZpzAuUzyS1WquuoEjQoPW4dUE3Yv3Dp\nN00WLHamSdImSq/ryHK0ON20kp+NnCxYlj1mDGwofya/nHJ2D0kVtD2pVCuB/X9M\nzCEDoskBvAiyD3+fkMvb9PfN4F6oiRKqOx99r22LaqcOVO3vTxtfdZ/3zYaH4QHE\nGCZNUGsrQiVNCk7jOzUk98kOVWtfH5tqJcq4n8TS++ASAF6mXnXm/Y00c/F20n/s\npw==\n-----END CERTIFICATE-----",
		SafeKey:     "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDqU/LFIP4Vccwn\nU2bwkNjElSYkMWY4OX+GZVOtBhnNTQlXJZt9O1XxNtFsSRos4mvOA0uxeXkYk8PS\n+25p3gu4ALOC8bwVAqtfFyUTdDrts/Z0zMEZI+erIPmWjBF14vs09gjRa7J1BhGT\nKZV4jWQYxTEkcJjMYRn+0GwVzEL0xSE/bZSbeJu9vjS7KPl9OGY/ds7HiFOAl96R\nxsBy0e7bURXHWpDxFRd/si79YFecht9c0tEcMO5uJXh/7/b299EKX1196rstFjKX\nJ2vkZtHgZJKCzKd+zf00zKWzYU3XX1HommaNg56+uGOz6p4kv4aqize3AvfUs96N\nFluU5q8NAgMBAAECggEACt5y9nQlz5kpgYd2UQvplLtgULHJOSwS6gkdfPs5S32v\nxGQ2Vp7fySJO9h6xQUoNUGR0aIUnD0icqYFyaSRcrHUSiS0s7q04MAnISqM2WoaM\nP3Wsh+h+RetFIxetyvyIxLecHW6ekUat861W8Ae1kAqRfCG24f/9BIXmsDTQbumh\nJy8Nc2OaUpaEsOqvVuPvdXeTEBVUwtsQJVihmreqsqGYVCuSvtbA9d7GdCNXRrP/\nCX0ZztDyBklihaacw5qgNWZqF1VmlA7bc23da4Pu/uuaPLCDg/4S2hCpaWLMCdHR\n87fq+wvcKX2JjJHa/yBAHdtq5HZ/19g+PQQKWkXm5wKBgQD4M7+7Wf4H+hGWN8Zr\nHqyjUJgj+sXFp1k9FOirbbPnTD3tE6caUUsSSUYSdnCngdGrG1i6uaY41vuP0Euj\n+P9OQ9QnnFSU2BVpiPSC1JfxTQ0Wi+0zHHnp2d/uoR784mFx8eyHdfElnP/AuGNk\nb8fpC07LWMENJHNIC0R/GzyenwKBgQDxsJx1/8W/3OyVzWuoad0gQG3qJPhhc5Vt\nrM4ATUC4IgifFgtkHkfFPaGEx1mfrb0ScdU2GVLRY9eaz+m9GVX9zDQz85EwPq1s\n+FxNAdOibEy7yfN3beoeNQ4GL/vDrSBA/X9eMJVcbU0z75Pk6vhv+32nrgSx1Azt\n0QApnjTO0wKBgQDgzBXifQ5scRxOnrOSP5UC0bMKG03Wx8w2W2KkKVbgrZgEymD1\noB1LMZxKioVb4WNiAwGpFQ4suuHbDkAEAjhRzXMwcRHWQaObExTKDfyT60JoYlFy\nkl8E43VDLyDez7aMOh4NTlAbzgeBqD81L1yzgK9b00X+Pj4/SR0/tg6AZwKBgCih\nANxRP+Pt9pOEMcng6fxG+HM4/cwcCw2h1At28R9DEWH06btN39DHeISCoo1WPoeA\nPVBX13U9rHvo4akZPjxo/ImTM2AB2VONOK71VKdkP03+OABmqMmlL5NYs6EEVHy4\nYJXr4t/ju+u0JY+A9HyWsVvjxARE2luMG9PjNYtjAoGATHOUzKm5KohrNC8ZTWtF\nGsUjFeHqnX7KM/yLMziPaR0L0Pc5OZsqdJbsilQW8a0KkHv6uC+t2IhOWNqBBmVV\neDcpPsrgJ/1gRzzuGS6FgmUnPzybkXY6DhgdpSKkut7ZkjpzB+UMWBwLoFlY8cfV\ndeNKwfs/4DCz61glzwkMrwE=\n-----END PRIVATE KEY-----",
	},
}
var oldDefaultOONIOpenVPNConfig = []*Config{
	{
		Cipher:      "AES-256-GCM",
		Auth:        "SHA512",
		Compress:    "stub",
		Provider:    "oonivpn",
		Obfuscation: "none",
		// yes, github, I know this looks like a leaked certificate. That is exactly what it is.
		SafeCA:   defaultCA,
		SafeCert: "-----BEGIN CERTIFICATE-----\nMIIDXjCCAkagAwIBAgIRALM/5njrVcneGXfmqnIX278wDQYJKoZIhvcNAQELBQAw\nFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwHhcNMjQwNzMxMTc1MjAxWhcNMjYxMTAz\nMTc1MjAxWjAaMRgwFgYDVQQDDA9jbGllbnQtY2VydC0wMDIwggEiMA0GCSqGSIb3\nDQEBAQUAA4IBDwAwggEKAoIBAQDfVfY+RK1Wl4Dw+KPJMOu7UT4g8VoWS0r5B3z8\nqzL/RAL9xEMaeJbeJPCkOCMaPiS5Xyuj2X/idSlejINmC+XhAx0+ANbxD7oilhBt\nLO43u8QRE5N2HBt045dJdFiN/lt2OwQOrYAL4p7hEn91zObT35wzK6jfNFMON9HQ\n3JZzEqcs/5SfnCvyAtAnV+Qfr4TolX2lRhu74Yl88OzjNFiGADniK/jJGJWfPEzn\nhqfzbcpXCVKUD38kFje3wBN+DrWQabuXTlJhOfhHANMgUnqoS91ea/TbfdiQ4kni\n1sE9RG/X+v8/Xm1BmJO2db1t1K/Px4wqE5Ku7XvdyVU4U4YHAgMBAAGjgaIwgZ8w\nCQYDVR0TBAIwADAdBgNVHQ4EFgQUAT/5mtUeUGCeFKrgBb/6i0B+2ycwUQYDVR0j\nBEowSIAUcr4ZIMrPHSCM73B+/k5XAE0+mxmhGqQYMBYxFDASBgNVBAMMC0Vhc3kt\nUlNBIENBghQ4+XCGnaz3qqoYXnOAs6nSeg3a7DATBgNVHSUEDDAKBggrBgEFBQcD\nAjALBgNVHQ8EBAMCB4AwDQYJKoZIhvcNAQELBQADggEBAA1VNnz0jz+1uLqQBdH2\nc5D97BdANVHjE6NptELekeoYni4IrqhJ8sjx60tq459nhaHZc4XaCMpuSb/rdxhF\nxh/D+PJlpQQxQkrIFLGTwDGVz0J6OI/PCLgjRwHqWIp7Y1DYtGEUtojhrRYCq6Dt\nHT3tG6Osd08tZTKeW1kOf35JZqu5JFOz52uIO7qmk5DZoR3O4Oxk4mCyA6kdu9tp\nk3n9OnrhQFVWy98N6cQ+k5UIyN1HgdWfhwIjxFJXVt4JfsF3jRyUyUDpuGXPQs6Y\nywVyfOE5EYqUfGDBqgUBEChaQTY2aTHQ9S9QVrIXHE3Gjj6pqjZg4TDUPv+h2fom\nu4c=\n-----END CERTIFICATE-----",
		SafeKey:  "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDfVfY+RK1Wl4Dw\n+KPJMOu7UT4g8VoWS0r5B3z8qzL/RAL9xEMaeJbeJPCkOCMaPiS5Xyuj2X/idSle\njINmC+XhAx0+ANbxD7oilhBtLO43u8QRE5N2HBt045dJdFiN/lt2OwQOrYAL4p7h\nEn91zObT35wzK6jfNFMON9HQ3JZzEqcs/5SfnCvyAtAnV+Qfr4TolX2lRhu74Yl8\n8OzjNFiGADniK/jJGJWfPEznhqfzbcpXCVKUD38kFje3wBN+DrWQabuXTlJhOfhH\nANMgUnqoS91ea/TbfdiQ4kni1sE9RG/X+v8/Xm1BmJO2db1t1K/Px4wqE5Ku7Xvd\nyVU4U4YHAgMBAAECggEAFXyT4h4XCSeD+xWzeEXxyrCVyB3q8Lh8tY8atMUUdR+f\neWJaTF/Wr/eg2UPMs20leQTRfNmOPATRSUHpMGEA/vrJjkS5aTF9/cQvP2RdD3/2\nlz1UE2m/2yk8TbpY/LbrKL75Hx+3xoJJOSvflxSdY+agWgH0z3NIHFboI5kytk5N\nOQZ+9zlMPx5FOhl38IizbaKG5xXbIFfZqz9TeShtycY1Uj4c1ghveFh/EwEcj51+\nFHoUo6IGVsODpuiSGIj23vlY/D6H4J2/wkwlhUHKg4zPLHEHL1PWlijQt80mYIx/\n+yUKl9QuRCLsEQTRX4h1iUuM8w2jV6FulpljPAZCNQKBgQD53+PjfXnrw+Ft0FoV\nJfTIHScS7U5jS25lRS0p4QSB+GcME0FY/sRPP4L0cloNNK6Ml5rKi90KHnoGGBTt\nIuIqZpChXbeSFLswiel1ye67GYgELXhdZZPI2xAgL2804nzMpoYx7+VHLJCVsskL\nkWHbO3VHmkiiyng5Fib/Bg9ANQKBgQDkz4Z2BaClDu8jn1JyoIPkmrz+Noako0wo\nGtfPW93FV+OqXoitJHJvXQZ962DZbnty6PsjCJixchqHep4HT9m4WUuzwZqJJG4J\ngS4+hlWS9pcvLpobeBeiJ1OnOQXk5FRgAVJ2suwtErPYCmtNsH9RwCkrRrTMxk6V\n7/yBkbisywKBgQCxJpELVIceplXpI+Dpw2oigcCVA5cigHT46S1W1of6mSB9iB05\nOg31XUK7iWLcn+/sDwOX+8avCOJb9bDIWoXbp7F8JdQihf4cMHpKnupYzYYH6DDA\njmZS7TQmjVqbNMNj19+mAb0cU7UB3Kn6QI0O/71rES/T8hV/63ukLCidzQKBgGPJ\nEUXFPILfWXE6mTU+RWbcCNIAq4V/ZcYTtsxKrxPSOYpiQc7olzNz6VHe5dTNNu8t\nJeDobdbtAR7WXbaonzWjU71oEGIAzjA88xL3eLhn7BT6iOCz5fKknfnOh4CEBzv7\nN6BmdVNO1bnBCXzPHSdk209xPYYUcc834vIKv/QzAoGAaciaMwabaecjjZPgUtCq\n+0hE6yWg8YQ7t60+jA8IanU29vpXVFJKr/yTgnExde96sD3POxBln8F2R2tHZK0Z\nR+BFhw5TUzDuSgQJRmhskbcR7u70I39fgm5G0ed9Qt5tx7bl32r9OGxMiWY6yGb+\nabwHOfqbU+03upE8+Of17sg=\n-----END PRIVATE KEY-----",
	},
	{
		Cipher:      "AES-256-GCM",
		Auth:        "SHA512",
		Compress:    "stub",
		Provider:    "oonivpn",
		Obfuscation: "none",
		// yes, github, I know this looks like a leaked certificate. That is exactly what it is.
		SafeCA:   defaultCA,
		SafeCert: "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIQFiJOrUbahl4vlYa6xv7SmzANBgkqhkiG9w0BAQsFADAW\nMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTAeFw0yNDA3MzExNzUyMDBaFw0yNjExMDMx\nNzUyMDBaMBoxGDAWBgNVBAMMD2NsaWVudC1jZXJ0LTAwMTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBANXyxnIb9tLqe6di6xIJaCDm0Ue4D8Cy0XYQKnbB\n8Ko9xJiglUm4BXAkjkOHLfSB38hOx9exXTW4whMuYOEJoo26JcdbmLJhaxiVAwTQ\nzMROgCbpJi1lu5cQ8F0U4Sq1/+IZKIGfmiWtxa2YP4Kc4qgEESk+AZ6rtxuKUvQU\nY0rLO1J1FuH8CgYnPG/dkwekVn47v7VnLzIM6XgPdezFNqwGYDAINrxutvnh8dI1\n9hoUZ5sTS9+747kXBy8049xfZqd7rUst9aC47Bt2BOXPUaKCeu1S4v6yEwQcQuSm\nHHoKHJsmlI1DQRS9ZRMq4e0ugFxwIMWz2Wwf2uC1VTOFbCUCAwEAAaOBojCBnzAJ\nBgNVHRMEAjAAMB0GA1UdDgQWBBR4NMh/CIuWF1L7MGM81Hg54ChATzBRBgNVHSME\nSjBIgBRyvhkgys8dIIzvcH7+TlcATT6bGaEapBgwFjEUMBIGA1UEAwwLRWFzeS1S\nU0EgQ0GCFDj5cIadrPeqqhhec4CzqdJ6DdrsMBMGA1UdJQQMMAoGCCsGAQUFBwMC\nMAsGA1UdDwQEAwIHgDANBgkqhkiG9w0BAQsFAAOCAQEARhgY3kmrJ5QP2cz9OcFB\nTjFQaQlnEts7Z4xcl/DNz3WNmqP2HVe1jzcHvZgkkcFNoP3BR/45rW2UiAAw3gx5\nupjcxceJ1GtStmZHM2ReO8mSumtkMZ60Qwo8z+xmbY8art28U2exXRCijtD2BYku\nVV6jaZGrWNk5JgSf4Eaj8oB5SDhuO18flogDAY4Y0iQDScYc8JYLXP9cgYJDLICM\n7wKanE8g3IL/Ruy5/nqNPRIPc28YP2U4sUDSNgIDQJKwXCHQmdUjXhMRNPy/I00I\nccK1qapMxfZHy+zQbUM2OPEsdr4oeqe2GRJowoI4Chb5w0s+GfTGopV5J2o8QfYz\nvw==\n-----END CERTIFICATE-----",
		SafeKey:  "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDV8sZyG/bS6nun\nYusSCWgg5tFHuA/AstF2ECp2wfCqPcSYoJVJuAVwJI5Dhy30gd/ITsfXsV01uMIT\nLmDhCaKNuiXHW5iyYWsYlQME0MzEToAm6SYtZbuXEPBdFOEqtf/iGSiBn5olrcWt\nmD+CnOKoBBEpPgGeq7cbilL0FGNKyztSdRbh/AoGJzxv3ZMHpFZ+O7+1Zy8yDOl4\nD3XsxTasBmAwCDa8brb54fHSNfYaFGebE0vfu+O5FwcvNOPcX2ane61LLfWguOwb\ndgTlz1GignrtUuL+shMEHELkphx6ChybJpSNQ0EUvWUTKuHtLoBccCDFs9lsH9rg\ntVUzhWwlAgMBAAECggEAGhllZXatV8H0hzEdOYRNwFO//HaC73Aw9qzWrUmP3Qiv\ncqoGRCmveiRveWPkcoMkZDheDx0rIHpTLIlYFqO5AX6PjMALAtm4+ZT1+xCophro\nbba76kZSicVueQBqzm3I0xFcHGH2qTmHV5uuxbVzPelPGZ+fjXZnnjOz1mQlT7J9\nf1846ICm2U3RtD6HtD0SD5SK/r6qsOawxdq2aVXn5Kgb6zwqvrSfzGgKHLwmFh60\nGXFCVko/ohsvesvpQd/ONbMXQEjq99vLMwLa/vjXtU2mSaHKvpR2uokTO9f5LJFq\n1VNzU7x2bCnUs7O3wk6TTYaV0zXsWo5OjMgWayB1MQKBgQDxtSFf1vfq+jCjhMMg\nmE2asnS/rnyH4pog1k8TiJZncubKdZU5QuaozDJhsGkSL16EAa7X2pvi17BhDJhe\ns0fprsziWqjp9isTQjMFMdrZhnL5uEVkK/JiSUGQdvlXxUpd0quhlVWkh8ZmZ8ce\nOQArEUJHWWgtvu5gS78izPnEUQKBgQDimW/7CE7zfdczUlj6Sp4v1+Tbdz/Iket3\nIQDj4f3DD6f1v5Hd9k43oPFPP6jbOwSW9dFMuHzsQW4G7WNSJmA+x0fe6qeKRLoV\nDe89WVMcCVlNX0kvwF68ojyFQ+/V+NjF2A6yeOAiM8tQf4YDHuNokzjSP7TVZIAc\nbeRERwZZlQKBgCqCVppKblOvKLq5cK/c2VkppYrInzIu0jiQOFwRG5KaDKjywQnP\nEE4Di6DOq8v89Lx2p09jLSNaF7UZx/pvwWgBzBrLIwXyu2SpsdtqBzlWggYVOG8D\no59RjuxfYD7lfcy+blz+rI9BKc181vIjyDnK0UNHICFbgQUCjV0Le6nhAoGBAM4M\nKehBuNDuZ+YSBjip60ej8EWkHMq77TnpN87/62kY7minJvOHib5JycN/JoMbGmRO\n6F/0DhwirvL7n2nO3YuYWAEarPgs4GxOvHGzrL/8vEh/0aPrL/olKBUiHo8Z9buJ\naGvfQCe5ozHyk6B40N6BqJR+O2gjN98iCgQP9XU1AoGBAJF9+aKNz5eirXnCT60n\nDjZ+o4jJ1c0C/dpuGBU9Sm0Q0qqLsZKIBLSrfMccmgpxhnsQ8a/9yXAkFkL+E0yX\nSOlGzbO1VhnsreBQg7oIe5PMsie+zulHwZ9gqvwH5T3xYGJJc6AzB1V3CS00jPsf\ncAnOQHux3yai6ZZkdXAAWJ8w\n-----END PRIVATE KEY-----",
	},
}
