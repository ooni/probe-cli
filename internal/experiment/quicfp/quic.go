package quicfp

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

// A connectionID in QUIC (same convention as internal/experiment/quicping).
type connectionID []byte

const (
	maxConnectionIDLen        = 18
	minConnectionIDLenInitial = 8
	defaultConnectionIDLength = 16

	// pnLen is the (fixed, for simplicity) packet number length in bytes
	// we use for every Initial packet we send. 4 bytes is always valid
	// per RFC 9000 §17.1 and matches internal/experiment/quicping.
	pnLen = 4

	// minInitialDatagramSize is the minimum UDP datagram size RFC 9000
	// §14.1 requires for the first Initial packet a client sends.
	minInitialDatagramSize = 1200

	// aeadTagLen is the AES-GCM authentication tag length in bytes.
	aeadTagLen = 16
)

func generateConnectionID(length int) connectionID {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic("quicfp: rand.Read failed: " + err.Error())
	}
	return connectionID(b)
}

// generateConnectionIDForInitial generates a connection ID of a length
// randomly chosen between 8 and 18 bytes, as valid for a client-chosen
// destination connection ID on the first Initial packet.
func generateConnectionIDForInitial() connectionID {
	r := make([]byte, 1)
	if _, err := rand.Read(r); err != nil {
		panic("quicfp: rand.Read failed: " + err.Error())
	}
	length := minConnectionIDLenInitial + int(r[0])%(maxConnectionIDLen-minConnectionIDLenInitial+1)
	return generateConnectionID(length)
}

func generateConnectionIDs() (dst, src connectionID) {
	return generateConnectionIDForInitial(), generateConnectionID(defaultConnectionIDLength)
}

// varint encodes v as a QUIC variable-length integer (RFC 9000 §16),
// using the smallest of the four legal encodings (1/2/4/8 bytes).
func varint(v uint64) []byte {
	switch {
	case v <= 63:
		return []byte{byte(v)}
	case v <= 16383:
		out := make([]byte, 2)
		binary.BigEndian.PutUint16(out, uint16(v))
		out[0] |= 0x40
		return out
	case v <= 1073741823:
		out := make([]byte, 4)
		binary.BigEndian.PutUint32(out, uint32(v))
		out[0] |= 0x80
		return out
	default:
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, v)
		out[0] |= 0xc0
		return out
	}
}

// varintDecode decodes a QUIC variable-length integer (RFC 9000 §16) from
// the start of b, returning its value and the number of bytes consumed.
func varintDecode(b []byte) (value uint64, n int) {
	if len(b) == 0 {
		return 0, 0
	}
	switch b[0] >> 6 {
	case 0:
		return uint64(b[0] & 0x3f), 1
	case 1:
		if len(b) < 2 {
			return 0, 0
		}
		return uint64(binary.BigEndian.Uint16(b[:2]) & 0x3fff), 2
	case 2:
		if len(b) < 4 {
			return 0, 0
		}
		return uint64(binary.BigEndian.Uint32(b[:4]) & 0x3fffffff), 4
	default:
		if len(b) < 8 {
			return 0, 0
		}
		return binary.BigEndian.Uint64(b[:8]) & 0x3fffffffffffffff, 8
	}
}

// cryptoFrame builds a CRYPTO frame (RFC 9000 §19.6) carrying data starting
// at stream offset offset.
func cryptoFrame(offset uint64, data []byte) []byte {
	out := []byte{0x06} // CRYPTO frame type
	out = append(out, varint(offset)...)
	out = append(out, varint(uint64(len(data)))...)
	out = append(out, data...)
	return out
}

// paddingFrames returns n PADDING frames (RFC 9000 §19.1), each one byte
// (type 0x00), which is the simplest legal encoding.
func paddingFrames(n int) []byte {
	if n <= 0 {
		return nil
	}
	return make([]byte, n) // zero-valued bytes are already 0x00
}

// buildUnprotectedHeader builds the long-header bytes (RFC 9000 §17.2.2, RFC
// 9001 for Initial specifically) up to and including the cleartext packet
// number, plus the numeric Length field's target so the caller can size the
// frame payload accordingly. remainderLen is len(framePayload)+aeadTagLen.
func buildUnprotectedHeader(version uint32, destConnID, srcConnID connectionID, pn int64, remainderLen int) []byte {
	hdr := []byte{0xc0 | byte(pnLen-1)} // long header, fixed bit, Initial type, reserved=0
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], version)
	hdr = append(hdr, v[:]...)
	hdr = append(hdr, byte(len(destConnID)))
	hdr = append(hdr, destConnID...)
	hdr = append(hdr, byte(len(srcConnID)))
	hdr = append(hdr, srcConnID...)
	hdr = append(hdr, 0x00) // token length = 0 (no retry token)
	hdr = append(hdr, varint(uint64(remainderLen))...)
	pnBytes := make([]byte, pnLen)
	binary.BigEndian.PutUint32(pnBytes[pnLen-4:], uint32(pn))
	hdr = append(hdr, pnBytes...)
	return hdr
}

// sealPacket assembles and protects one Initial packet/datagram given its
// plaintext frame payload (CRYPTO frame(s), optionally followed by
// PADDING). If padTo > 0, PADDING frames are appended until the resulting
// UDP datagram would be at least padTo bytes; pass padTo <= 0 to disable
// padding entirely (used by the below-1200-bytes vector).
func sealPacket(version uint32, salt [20]byte, destConnID, srcConnID connectionID, pn int64, framePayload []byte, padTo int) ([]byte, error) {
	payload := append([]byte{}, framePayload...)
	if padTo > 0 {
		// Account for header + AEAD tag overhead when deciding how much
		// PADDING is needed to reach padTo total bytes on the wire.
		hdrLenEstimate := 7 + len(destConnID) + len(srcConnID) + 2 + pnLen // conservative
		need := padTo - hdrLenEstimate - len(payload) - aeadTagLen
		payload = append(payload, paddingFrames(need)...)
	}
	// The Length field covers the Packet Number field plus the (AEAD-
	// sealed) Payload field that follows it (RFC 9000 §17.2).
	remainderLen := pnLen + len(payload) + aeadTagLen
	hdr := buildUnprotectedHeader(version, destConnID, srcConnID, pn, remainderLen)
	return cryptoProtect(hdr, payload, destConnID, salt, pn, pnLen)
}

// buildStandardPacket builds a single Initial datagram carrying the whole
// ClientHello in one CRYPTO frame at offset 0, padded to at least 1200
// bytes as RFC 9000 §14.1 requires of a client's first Initial packet. This
// is the baseline shape used for the QUIC-version vectors and every TLS
// length-field permutation vector.
func buildStandardPacket(version uint32, salt [20]byte, ch []byte) (raw []byte, dst, src connectionID, err error) {
	dst, src = generateConnectionIDs()
	frame := cryptoFrame(0, ch)
	raw, err = sealPacket(version, salt, dst, src, 2, frame, minInitialDatagramSize)
	return
}

// buildBelow1200Packet is identical to buildStandardPacket except it does
// NOT pad the datagram up to the spec-mandated 1200-byte minimum, testing
// whether the target still parses/blocks an under-sized Initial datagram
// (Table 9's "Below 1200 Bytes" row).
func buildBelow1200Packet(version uint32, salt [20]byte, ch []byte) (raw []byte, dst, src connectionID, err error) {
	dst, src = generateConnectionIDs()
	frame := cryptoFrame(0, ch)
	raw, err = sealPacket(version, salt, dst, src, 2, frame, 0)
	return
}

// buildFragmentedPacket carries the whole ClientHello in a SINGLE datagram
// but split across two separate CRYPTO frames with contiguous, correctly
// offset stream data (Table 9's "Fragmentation" row): this tests whether
// the target reassembles multiple CRYPTO frames within one packet before
// attempting to parse the TLS message.
func buildFragmentedPacket(version uint32, salt [20]byte, ch []byte) (raw []byte, dst, src connectionID, err error) {
	if len(ch) < 2 {
		return nil, nil, nil, errors.New("quicfp: ClientHello too short to fragment")
	}
	mid := len(ch) / 2
	frames := append(cryptoFrame(0, ch[:mid]), cryptoFrame(uint64(mid), ch[mid:])...)
	dst, src = generateConnectionIDs()
	raw, err = sealPacket(version, salt, dst, src, 2, frames, minInitialDatagramSize)
	return
}

// buildMultiPacketInitial carries the ClientHello across TWO separate UDP
// datagrams (Table 9's "Multi-Packet Initial" row): the first packet (which
// RFC 9000 §14.1 requires to be padded to 1200 bytes) carries CRYPTO offset
// [0,mid), and the second -- which needs no such padding -- carries the
// remainder at the correct offset. Both packets share the same connection
// IDs and use increasing packet numbers so their AEAD nonces do not repeat.
func buildMultiPacketInitial(version uint32, salt [20]byte, ch []byte) (packets [][]byte, dst, src connectionID, err error) {
	if len(ch) < 2 {
		return nil, nil, nil, errors.New("quicfp: ClientHello too short to split")
	}
	mid := len(ch) / 2
	dst, src = generateConnectionIDs()

	first, err := sealPacket(version, salt, dst, src, 2, cryptoFrame(0, ch[:mid]), minInitialDatagramSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("quicfp: sealing first packet: %w", err)
	}
	second, err := sealPacket(version, salt, dst, src, 3, cryptoFrame(uint64(mid), ch[mid:]), 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("quicfp: sealing second packet: %w", err)
	}
	return [][]byte{first, second}, dst, src, nil
}

// buildForcedNegotiationPacket sends the ClientHello under a reserved,
// unassigned ("greased") version number, per RFC 9000 §15 -- a
// spec-compliant target will not recognize the version and must reply with
// a Version Negotiation packet without attempting to decrypt the payload at
// all. This tests whether a middlebox blocks/drops the datagram purely on
// the visible (unencrypted) long-header fields, irrespective of content it
// cannot legitimately read (Table 9's "Forced Negotiation" row). The payload
// is still AEAD-"protected" (using the same best-effort salt as other
// unverified version vectors) purely so the datagram is syntactically well
// formed; no real endpoint is expected to decrypt it.
func buildForcedNegotiationPacket(ch []byte) (raw []byte, dst, src connectionID, err error) {
	return buildStandardPacket(greasedVersion(), saltDraft29, ch)
}

// dissectVersionNegotiation parses a server Version Negotiation packet
// (RFC 9000 §17.2.1), returning the versions it advertises and its
// Destination Connection ID (which must equal the srcConnID of the
// triggering request for the response to be matched to it).
func dissectVersionNegotiation(pkt []byte) (versions []uint32, dst connectionID, err error) {
	if len(pkt) < 7 {
		return nil, nil, errors.New("quicfp: packet too short")
	}
	if pkt[0]&0x80 == 0 {
		return nil, nil, errors.New("quicfp: not a long header packet")
	}
	if binary.BigEndian.Uint32(pkt[1:5]) != 0 {
		return nil, nil, errors.New("quicfp: not a Version Negotiation packet")
	}
	dstLen := int(pkt[5])
	if 6+dstLen > len(pkt) {
		return nil, nil, errors.New("quicfp: truncated destination connection ID")
	}
	dst = connectionID(pkt[6 : 6+dstLen])
	offset := 6 + dstLen
	if offset >= len(pkt) {
		return nil, nil, errors.New("quicfp: truncated source connection ID length")
	}
	srcLen := int(pkt[offset])
	offset += 1 + srcLen
	for offset+4 <= len(pkt) {
		versions = append(versions, binary.BigEndian.Uint32(pkt[offset:offset+4]))
		offset += 4
	}
	return versions, dst, nil
}

// dissectLongHeader parses the version and Destination Connection ID of a
// long-header packet (RFC 8999 §5.1), which every QUIC version shares.
func dissectLongHeader(pkt []byte) (version uint32, dst connectionID, err error) {
	if len(pkt) < 6 || pkt[0]&0x80 == 0 {
		return 0, nil, errors.New("quicfp: not a long header packet")
	}
	dstLen := int(pkt[5])
	if 6+dstLen > len(pkt) {
		return 0, nil, errors.New("quicfp: truncated destination connection ID")
	}
	return binary.BigEndian.Uint32(pkt[1:5]), connectionID(pkt[6 : 6+dstLen]), nil
}

// longHeaderPacketType names the type of a long-header packet of the given
// version from the type bits of its first byte (RFC 9000 §17.2 for v1, RFC
// 9369 §3.2 for v2, which numbers the same types differently).
func longHeaderPacketType(first byte, version uint32) string {
	bits := (first >> 4) & 0x03
	if version == versionV2 {
		return [4]string{"retry", "initial", "0rtt", "handshake"}[bits]
	}
	return [4]string{"initial", "0rtt", "handshake", "retry"}[bits]
}
