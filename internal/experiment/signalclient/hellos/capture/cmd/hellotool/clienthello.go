package main

//
// ClientHello handshake messages: reading one from the start of a TLS stream, its fields,
// zeroing its per-connection random values, and its JA3 and JA4 fingerprints.
//

import (
	"crypto/md5" // #nosec G501 -- JA3 is defined with MD5
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"golang.org/x/crypto/cryptobyte"
)

// TLS extension types used here.
const (
	extServerName        = 0
	extSupportedGroups   = 10
	extECPointFormats    = 11
	extSignatureAlgs     = 13
	extALPN              = 16
	extPadding           = 21
	extSupportedVersions = 43
	extKeyShare          = 51
)

// isGREASE says whether v is a GREASE value (RFC 8701): 0x0a0a, 0x1a1a ... 0xfafa.
func isGREASE(v uint16) bool {
	return v&0x0f0f == 0x0a0a && v>>8 == v&0xff
}

// clientHello returns the ClientHello handshake message at the start of a TLS stream (which may
// span several handshake records), or nil.
func clientHello(stream []byte) []byte {
	var hs []byte
	s := cryptobyte.String(stream)
	for {
		var contentType uint8
		var version uint16
		var fragment cryptobyte.String
		if !s.ReadUint8(&contentType) || contentType != 22 || !s.ReadUint16(&version) ||
			!s.ReadUint16LengthPrefixed(&fragment) {
			break
		}
		hs = append(hs, fragment...)
		if len(hs) >= 4 && len(hs) >= 4+handshakeLength(hs) {
			break
		}
	}
	if len(hs) < 4 || hs[0] != 1 || len(hs) < 4+handshakeLength(hs) {
		return nil
	}
	return hs[:4+handshakeLength(hs)]
}

// handshakeLength is a handshake message's body length (its header's 24-bit length).
func handshakeLength(hs []byte) int {
	return int(hs[1])<<16 | int(hs[2])<<8 | int(hs[3])
}

// helloFields are the ClientHello fields the fingerprints use.
type helloFields struct {
	legacyVersion  uint16
	ciphers        []uint16
	extensions     []uint16 // types, in order
	sni            string
	alpn           []string
	groups         []uint16
	sigAlgs        []uint16
	pointFormats   []uint8
	versions       []uint16
	extensionData  map[uint16][]byte
	sessionIDStart int // offsets in the handshake message, for normalize
	sessionIDLen   int
	keyShares      [][2]int // [start, end) of each non-GREASE key_exchange
}

var errMalformed = errors.New("malformed ClientHello")

// parseHello reads a ClientHello handshake message (starting with its 4-byte header).
func parseHello(hs []byte) (*helloFields, error) {
	if len(hs) < 4 || hs[0] != 1 || len(hs) != 4+handshakeLength(hs) {
		return nil, errMalformed
	}
	f := &helloFields{extensionData: map[uint16][]byte{}}
	s := cryptobyte.String(hs[4:])
	var random, sessionID, ciphers, compression, extensions cryptobyte.String
	if !s.ReadUint16(&f.legacyVersion) || !s.ReadBytes((*[]byte)(&random), 32) {
		return nil, errMalformed
	}
	f.sessionIDStart = len(hs) - len(s) + 1
	if !s.ReadUint8LengthPrefixed(&sessionID) || !s.ReadUint16LengthPrefixed(&ciphers) ||
		!s.ReadUint8LengthPrefixed(&compression) || !s.ReadUint16LengthPrefixed(&extensions) || !s.Empty() {
		return nil, errMalformed
	}
	f.sessionIDLen = len(sessionID)
	for !ciphers.Empty() {
		var c uint16
		if !ciphers.ReadUint16(&c) {
			return nil, errMalformed
		}
		f.ciphers = append(f.ciphers, c)
	}
	for !extensions.Empty() {
		var typ uint16
		var data cryptobyte.String
		if !extensions.ReadUint16(&typ) || !extensions.ReadUint16LengthPrefixed(&data) {
			return nil, errMalformed
		}
		if _, dup := f.extensionData[typ]; dup {
			return nil, errMalformed
		}
		dataStart := len(hs) - len(extensions) - len(data)
		f.extensions = append(f.extensions, typ)
		f.extensionData[typ] = data
		if err := f.readExtension(typ, data, dataStart); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// readExtension records what the fingerprints need from one extension. dataStart is where its
// data starts in the handshake message.
func (f *helloFields) readExtension(typ uint16, data cryptobyte.String, dataStart int) error {
	var list cryptobyte.String
	switch typ {
	case extServerName: // server_name_list: one host_name
		var nameType uint8
		var name cryptobyte.String
		if data.ReadUint16LengthPrefixed(&list) && list.ReadUint8(&nameType) && list.ReadUint16LengthPrefixed(&name) {
			f.sni = string(name)
		}
	case extALPN:
		if data.ReadUint16LengthPrefixed(&list) {
			for !list.Empty() {
				var proto cryptobyte.String
				if !list.ReadUint8LengthPrefixed(&proto) {
					return errMalformed
				}
				f.alpn = append(f.alpn, string(proto))
			}
		}
	case extSupportedGroups, extSignatureAlgs:
		values := readUint16s(data, 2)
		if typ == extSupportedGroups {
			f.groups = values
		} else {
			f.sigAlgs = values
		}
	case extECPointFormats:
		if data.ReadUint8LengthPrefixed(&list) {
			f.pointFormats = append([]uint8(nil), list...)
		}
	case extSupportedVersions:
		f.versions = readUint16s(data, 1)
	case extKeyShare: // client_shares: (group, key_exchange) entries
		if !data.ReadUint16LengthPrefixed(&list) {
			return errMalformed
		}
		offset := dataStart + 2
		for !list.Empty() {
			var group uint16
			var key cryptobyte.String
			before := len(list)
			if !list.ReadUint16(&group) || !list.ReadUint16LengthPrefixed(&key) {
				return errMalformed
			}
			start := offset + 4
			if !isGREASE(group) {
				f.keyShares = append(f.keyShares, [2]int{start, start + len(key)})
			}
			offset += before - len(list)
		}
	}
	return nil
}

// readUint16s reads a list of uint16 values after a lengthBytes-long length prefix.
func readUint16s(data cryptobyte.String, lengthBytes int) []uint16 {
	var list cryptobyte.String
	ok := false
	if lengthBytes == 1 {
		ok = data.ReadUint8LengthPrefixed(&list)
	} else {
		ok = data.ReadUint16LengthPrefixed(&list)
	}
	var out []uint16
	for ok && !list.Empty() {
		var v uint16
		if !list.ReadUint16(&v) {
			break
		}
		out = append(out, v)
	}
	return out
}

// randomSpans are the [start, end) ranges of a ClientHello's per-connection random values: the
// client random, the legacy session id and each non-GREASE key share's key_exchange.
func randomSpans(hs []byte) ([][2]int, error) {
	f, err := parseHello(hs)
	if err != nil {
		return nil, err
	}
	random := [2]int{4 + 2, 4 + 2 + 32} // after the handshake header and legacy_version
	spans := [][2]int{random, {f.sessionIDStart, f.sessionIDStart + f.sessionIDLen}}
	return append(spans, f.keyShares...), nil
}

// normalize zeroes a ClientHello's per-connection random values, keeping their lengths, so a
// capture of the same handshake gives the same bytes.
func normalize(hs []byte) ([]byte, error) {
	spans, err := randomSpans(hs)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), hs...)
	for _, sp := range spans {
		clear(out[sp[0]:sp[1]])
	}
	return out, nil
}

// withoutGREASE returns values minus GREASE values.
func withoutGREASE(values []uint16) []uint16 {
	return slices.DeleteFunc(slices.Clone(values), isGREASE)
}

// join formats values with format and joins them with sep.
func join[T uint8 | uint16](values []T, format, sep string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf(format, v)
	}
	return strings.Join(parts, sep)
}

// ja3 is the hello's JA3 fingerprint (MD5 of version, ciphers, extensions, groups, point formats).
func (f *helloFields) ja3() string {
	s := fmt.Sprintf("%d,%s,%s,%s,%s", f.legacyVersion, join(withoutGREASE(f.ciphers), "%d", "-"),
		join(withoutGREASE(f.extensions), "%d", "-"), join(withoutGREASE(f.groups), "%d", "-"),
		join(f.pointFormats, "%d", "-"))
	sum := md5.Sum([]byte(s)) // #nosec G401 -- JA3 is defined with MD5
	return hex.EncodeToString(sum[:])
}

// ja4 is the hello's JA4 fingerprint (FoxIO's JA4, TCP).
func (f *helloFields) ja4() string {
	versions := withoutGREASE(f.versions)
	if len(versions) == 0 {
		versions = []uint16{f.legacyVersion}
	}
	version := map[uint16]string{0x0304: "13", 0x0303: "12", 0x0302: "11", 0x0301: "10"}[slices.Max(versions)]
	if version == "" {
		version = "00"
	}
	destination := "i" // an IP address, or no SNI
	if f.sni != "" && net.ParseIP(f.sni) == nil {
		destination = "d"
	}
	ciphers, extensions := withoutGREASE(f.ciphers), withoutGREASE(f.extensions)
	alpn := "00"
	if len(f.alpn) > 0 && f.alpn[0] != "" {
		first := f.alpn[0]
		alpn = first[:1] + first[len(first)-1:]
	}
	sortedCiphers := slices.Sorted(slices.Values(ciphers))
	var rest []uint16
	for _, e := range slices.Sorted(slices.Values(extensions)) {
		if e != extServerName && e != extALPN {
			rest = append(rest, e)
		}
	}
	tail := join(rest, "%04x", ",")
	if sigs := join(withoutGREASE(f.sigAlgs), "%04x", ","); sigs != "" {
		tail += "_" + sigs
	}
	return fmt.Sprintf("t%s%s%02d%02d%s_%s_%s", version, destination, len(ciphers), len(extensions), alpn,
		hash12(join(sortedCiphers, "%04x", ",")), hash12(tail))
}

// hash12 is the first 12 hex digits of s's SHA-256, or zeros for an empty s.
func hash12(s string) string {
	if s == "" {
		return "000000000000"
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}
