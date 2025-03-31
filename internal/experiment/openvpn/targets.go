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

var defaultCA = "-----BEGIN CERTIFICATE-----\nMIIDSzCCAjOgAwIBAgIUOPlwhp2s96qqGF5zgLOp0noN2uwwDQYJKoZIhvcNAQEL\nBQAwFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwHhcNMjQwNzMxMTc1MDI3WhcNMzQw\nNzI5MTc1MDI3WjAWMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBALfhmQ6YndIaq9K2ya1HNv9e3DiwKO8X7Ferh8KV\n/Yobs1jPJYfK/l1SZTO97FnIptqxPzGAWuxhS/+4n4ZB2RpszJKdu3sHYNY6lZCR\nw8dtxKYDIS5v/1by6AJk052wV3NWizw1QiawCOJl5cNN5Vb4OpLPvBzrx3IN7jvO\n0HxaaRYIiPdQy++cJ/wqQazTvPYpws0rIAF0A9jxzgsJZoWshg8MhQm9OYIMyZ2C\n4WeuBKU5bR7vqjAQnVH6ZsZ8ZX1UILq++PcuLeDYbg7M5YmT0v0SO+3ealgg48SO\nxqStAawEAXI2sOZqWTvFfXiq9l6Uw2uxPwXnzSO8hjjVqc0CAwEAAaOBkDCBjTAM\nBgNVHRMEBTADAQH/MB0GA1UdDgQWBBRyvhkgys8dIIzvcH7+TlcATT6bGTBRBgNV\nHSMESjBIgBRyvhkgys8dIIzvcH7+TlcATT6bGaEapBgwFjEUMBIGA1UEAwwLRWFz\neS1SU0EgQ0GCFDj5cIadrPeqqhhec4CzqdJ6DdrsMAsGA1UdDwQEAwIBBjANBgkq\nhkiG9w0BAQsFAAOCAQEAPpb2z/wBj9tULuzBQ1j6qkIUCkyH6e+QATHcCcJGWQsU\naeEc1w/qBXaJcRS0ahALXC3d/Tz8R2dAj1sO1HEsfjEs5fv1dKGgeVb1rNuZuUW8\n9xEtUdp3jL3xumcqfxKIwOv8Y1fz+AKGJbbPC3yoHptwMDW9zyaRTQ+McKE7Y497\nFZDF2RWQjgpxwCi7P3cScNBLNtt42TPnj6Up3D6Sj57YVDK9dXbrDj94bwmkQa8s\nl8Mp/PFaFeLNXXuGGVEbIlFuw9RY32vbJ1CrS9rrWlVq9Q17NrAmSYSBi9T19mDh\nMFslRMPBN4Jfd/45V26iW2XMpWCONY5aqAfx+2Oz2g==\n-----END CERTIFICATE-----"
var newCA = "-----BEGIN CERTIFICATE-----\nMIIDSzCCAjOgAwIBAgIUJ3ZRUh82LK928D6RJ+W5HBsPBREwDQYJKoZIhvcNAQEL\nBQAwFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwHhcNMjUwMzMxMTQ0ODA4WhcNMzUw\nMzI5MTQ0ODA4WjAWMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBAKxpF6BQPwFEQZjYrXkBtsPKSwUKLI+OtNDWS0va\nmyIl1X54dGxgGffTqMlzlTs0tvgqI9pUo85LMuKspq5ZC8PRJX6K5UXRF2DzAgQR\nMlG3f5nRnNQc2FahTVJSQQCRGODa7v6ouNWGbhlF4/TklZGsKh+iOKzWWohQjeiv\nbcI9l+jXGV9n8WBU+gfQUno/BSHc0Ifq2CK5g0Xff7dPUQi7XExwnxVsL8HMo9fx\n+HHnRphJjJCZn0Zv6QPoBvh7DUoJeC8qNwdXbaXJjLXGsZuaousS1zSw10Pxk1lc\neCiayIn0MqgyXFXsMMGLvkM39f/Bl/U6gF5/eOnIsCr6P28CAwEAAaOBkDCBjTAM\nBgNVHRMEBTADAQH/MB0GA1UdDgQWBBQAEj/upbhT8mgZ0aQZgkSAP72aeDBRBgNV\nHSMESjBIgBQAEj/upbhT8mgZ0aQZgkSAP72aeKEapBgwFjEUMBIGA1UEAwwLRWFz\neS1SU0EgQ0GCFCd2UVIfNiyvdvA+kSfluRwbDwURMAsGA1UdDwQEAwIBBjANBgkq\nhkiG9w0BAQsFAAOCAQEAf0iB54UXJ7/PD4euPaKPQxfmx7ohg8SGuYOjxRA6JUP6\n9gfKZZ4jA0qOraui/o+qyl+IchvdLcC+UhUCAOofcFSIdG5iFdIc1WAdWSomTCHC\nwS7X3B5ysP6fpbNwgmLVpzo7hbbYO85JqA+2Q2z/gEeO5nSZ5o7OlkZJ27ASGflE\naWE5C/lR7TESXw5Ugfp0YlhthFGn1uQ+3H74c3t5YiO7p3BAqOHoVQLww+9ksYxL\ntwP4hssO+b34rZ10R0sSGITrFCZKT4kCHjpDyFWaNtxH3Ite/Brmrss6b7D4SbZm\nNpMihPh/35AwbKVzcL1qmfyjUVnWfQvZCZXAjyHfLw==\n-----END CERTIFICATE-----"
var defaultOONIOpenVPNConfig = []*Config{
	{
		Cipher:      "AES-256-GCM",
		Auth:        "SHA512",
		Compress:    "stub",
		Provider:    "oonivpn",
		Obfuscation: "none",
		SafeCA:      newCA,
		SafeCert:    "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIQFydEI7c3W2HgPMstWzMDjTANBgkqhkiG9w0BAQsFADAW\nMRQwEgYDVQQDDAtFYXN5LVJTQSBDQTAeFw0yNTAzMzExNDQ4NTVaFw0yNzA3MDQx\nNDQ4NTVaMBoxGDAWBgNVBAMMD2NsaWVudC1jZXJ0LTAwMTCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBAOIjQaaex/4fk230wVI5d3NV2sEAbOxnqMjFrv/K\nCj0Ue/j+1pvC/mWDDkohSRt6W7/jyzI8j3MzQU3xfsR8pWPXCsTk1303f1DJI4Pj\neiKzqYMQ9m9rEH0VukWnJ7FqMXRRtDs5YyrnIf+9rhf+hdDMx7ZgtHPnqOBNJ9Ns\nqVOFkpADUdmGBvE0on7rRqBWFaRMF0Fvo01ghlmtOJxGZbHe6i4CulYacg7G80bA\nFJOsh0hWYRDttkuPCPtEtJt4fIK993I4u0tZ6pJ7ikHSEPyZTOxILfQZNuydrxcL\n5kmG5bTcSula5jkWOe+aAc5P/s3ijpU9LeU0jQv6+TohPX8CAwEAAaOBojCBnzAJ\nBgNVHRMEAjAAMB0GA1UdDgQWBBTMISuUtHT5z0GHvr0r/R4MAWJfljBRBgNVHSME\nSjBIgBQAEj/upbhT8mgZ0aQZgkSAP72aeKEapBgwFjEUMBIGA1UEAwwLRWFzeS1S\nU0EgQ0GCFCd2UVIfNiyvdvA+kSfluRwbDwURMBMGA1UdJQQMMAoGCCsGAQUFBwMC\nMAsGA1UdDwQEAwIHgDANBgkqhkiG9w0BAQsFAAOCAQEATLihT6gGMfVyH2LNp+eA\n/SVTnbhPkl3ZvboCnxbqBgktnfjgukDfPqxxn7hh8EGhwE3BJJrxiI2//nGyvwQ/\nUPnyUpEDJ0F64nP9SjgmggGZ7NF3lkiQgUDPDs8rtp9dD4fsgeOLJRhK5kTxQHDV\nwdyIER//sN+ZSfnObq9uvA9BJeTgv7XQL8i/vlk2F1toGn3Pg6gHP1pu6bdo3Rax\nXwg70K5hWW0+pMflqTZ2N37X6l1dleAA8E+gyuPnHzN5k7lr4wzvCwPAZUHv8mpX\nCpT37JZNJ+L3pBx82nSYBkVtUaeMcuXc7fPgoeVKZnqKfKwzjtveJn6kvFS6iIxE\n7g==\n-----END CERTIFICATE-----",
		SafeKey:     "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDiI0Gmnsf+H5Nt\n9MFSOXdzVdrBAGzsZ6jIxa7/ygo9FHv4/tabwv5lgw5KIUkbelu/48syPI9zM0FN\n8X7EfKVj1wrE5Nd9N39QySOD43ois6mDEPZvaxB9FbpFpyexajF0UbQ7OWMq5yH/\nva4X/oXQzMe2YLRz56jgTSfTbKlThZKQA1HZhgbxNKJ+60agVhWkTBdBb6NNYIZZ\nrTicRmWx3uouArpWGnIOxvNGwBSTrIdIVmEQ7bZLjwj7RLSbeHyCvfdyOLtLWeqS\ne4pB0hD8mUzsSC30GTbsna8XC+ZJhuW03ErpWuY5FjnvmgHOT/7N4o6VPS3lNI0L\n+vk6IT1/AgMBAAECggEAPuwXeuDIfi3UDG7jkdj6o0V62mQfasJHJSJYWMUjhy5Z\ndoWWOn4N4joOslLDsFkiCWk9Y2uVcLUM0T7Giq6yKG8wCGwyh513GAf3kV2s4NYD\n7zu7jnIIgv2WOIZT1ZpaS2bmkuobbVclzy68r+F644fvdBBBddS6WTsuvzYHgXzt\nroSvLUQZaJ2qCAQUcAd9OH+XPCo59jBiQ/FAnEo2En7BRjLa+AgG73vkH/mJ848R\n0z/kdOo9QYjF2O1+nRz4p822IxCbxkxuyEKP3INL/OIYtZbm3dzwBT2ZBiGVsSZz\n2d0b71Sx/rdzxr/7uGlISxcJQHQpnd7oI/2wThl5JQKBgQD7QbWbSuxjOggGFuD7\naRUyVe3eO1sPgJOSVFE9vSU3KMwgrVWVFuWuXp0d3mKcwKnvCvC7QXXUM6bhrlNE\nRWvgWKmdnOWftzS0nzzbH7ZSi153ouXYtiwlMJarv96wVsHhSHLZuODPyyAUGtZD\n+wKOhdv9wMCAkASiVx5oNxx0mwKBgQDmaCaB+fEq3JQ2oaOKDtEC3W45MhWy2r44\na1Bj8Up9/aGBmtem4WaLqsoBRBgeYFJnzWbGYCPL//iAyVdH6R2tV/zH8NgBhHYd\nIJdw7zPOVNe6CQQLLYSKNCy3hanq7EntX/BuwvQMyxZEoPyefipo17vR17srMBqg\nkDWWd/9+7QKBgBQBPVZp9rxHyI/DjpI14awqT1AcZbmsQROrhvRldOsZYndPlbrT\nIeJqQhQ4up1EccNImbYvFhSZMDZTzOlopZgtNwQHk/jNe3LOIuhHdBrKYfEglTBW\nANnNLpHqKu7bWYB4ZjyTjcK9p4YyRNEQ0TlLyyI/EJSr4tKSISenC6r1AoGBAJZz\nJ8HznWDpi/ZocctyhVSjlpfOdpDFeTSKDoJUStoeRdnFbE/SGw3URfexqTYcm3Kg\nkx2yJW3OBUzDLDUOqYxJSgyc3c+P2TiH0Zuy9lci24Lk5DkQTd2Y2CI2K4/1Hwjb\nfro0nvvDmRTTMmGj7LltNX1+qv82eAykXfmU/ca5AoGBAK0GyvzLCwXDq3eX6OtK\nOrHY10bK9LYrYxs30CAbMOFIV6kyemHLQM9VkOmhOQgNiqB8xHwyWhQUT/2MrOaL\n/pRHwNMno9XtfhJdgF71FyJoYvlm2yQwrSWMcytuGl31+hPAwLoGkB8yNdLN4p1w\nvN6new/aip0snmy4gJlFz3Cd\n-----END PRIVATE KEY-----",
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
