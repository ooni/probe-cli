package quicfp

import (
	"encoding/binary"
	"errors"
	"net"
	"time"
)

// cryptoFrameInfo is one CRYPTO frame's offset and payload, as scanned from
// a decrypted Initial packet's frame payload.
type cryptoFrameInfo struct {
	offset uint64
	data   []byte
}

// scanCryptoFrames walks a decrypted plaintext frame payload (CRYPTO and/or
// PADDING frames) and returns each CRYPTO frame's offset/data separately,
// unlike parseFrames (in quicfp_test.go) which merges them into one
// reassembled stream. The router needs frame COUNT and OFFSETS themselves
// as classification signals (multiple frames -> crypto_frame_split; a
// nonzero offset -> a multi-packet-initial continuation), not just the
// reassembled bytes.
func scanCryptoFrames(payload []byte) ([]cryptoFrameInfo, error) {
	var frames []cryptoFrameInfo
	i := 0
	for i < len(payload) {
		switch payload[i] {
		case 0x00: // PADDING
			i++
		case 0x06: // CRYPTO
			i++
			offset, n := varintDecode(payload[i:])
			if n == 0 {
				return nil, errors.New("mockcensor: bad crypto frame offset")
			}
			i += n
			length, n2 := varintDecode(payload[i:])
			if n2 == 0 {
				return nil, errors.New("mockcensor: bad crypto frame length")
			}
			i += n2
			if i+int(length) > len(payload) {
				return nil, errors.New("mockcensor: truncated crypto frame")
			}
			frames = append(frames, cryptoFrameInfo{offset: offset, data: payload[i : i+int(length)]})
			i += int(length)
		default:
			return nil, errors.New("mockcensor: unexpected frame type")
		}
	}
	return frames, nil
}

// minPlausibleClientHelloSize is smaller than any complete ClientHello
// quicfp builds for a normal short test SNI (roughly 126 bytes plus the
// SNI itself), but larger than half of one -- used to recognize a
// multi-packet-initial's first fragment by size alone, without trusting
// that fragment's own (necessarily incomplete, and in the message-length
// vectors' case possibly separately mangled) declared body length.
const minPlausibleClientHelloSize = 100

// classifyDatagram inspects a raw UDP datagram the way a DPI device
// implementing Table 9/12's techniques would, and names which quicfp
// vector it matches, reusing the exact same cleartext-header, version,
// size, decryption (RFC 9001, same salts quicfp itself uses) and
// ClientHello-field logic already validated in
// wireclienthello_test.go/wireclienthello_classify_test.go. ok is false
// for anything it can't confidently classify (parse failure, or a
// genuinely clean/unmangled ClientHello with no corresponding vector).
func classifyDatagram(datagram []byte) (vectorName string, ok bool) {
	// Size-based categories never require decryption.
	if len(datagram) < minInitialDatagramSize {
		return "quic_feature_below_1200_bytes", true
	}
	if len(datagram) > 2000 {
		return "quic_feature_oversized_datagram", true
	}
	if len(datagram) < 7 || datagram[0]&0x80 == 0 {
		return "", false
	}

	// The version field is always cleartext (RFC 9000 §17.2). Every
	// version EXCEPT v1 is exclusive to its own dedicated
	// quic_version_gquic_*/ietf_draft22_28/ietf_v2 vector, so those can be
	// classified immediately. v1 is different: quicfp's TLS-length-field,
	// crypto_frame_split, and multi_packet_initial vectors all use it too
	// (v1 is just the "ordinary" version for anything not specifically
	// testing version support), so a v1 packet must be decrypted and its
	// content inspected FIRST -- only a genuinely clean, complete,
	// single-frame ClientHello falls back to meaning "this is testing
	// draft-29/v1 support itself", checked last, not first.
	version := binary.BigEndian.Uint32(datagram[1:5])
	if version == greasedVersion() {
		return "quic_feature_forced_negotiation", true
	}
	for _, qv := range quicVersionVectors() {
		if qv.Version == version && qv.Version != 1 {
			return qv.Name, true
		}
	}
	if version != 1 {
		// Not one of quicfp's known version vectors and not v1 either;
		// nothing further to classify.
		return "", false
	}

	dstLen := int(datagram[5])
	if 6+dstLen >= len(datagram) {
		return "", false
	}
	dst := connectionID(datagram[6 : 6+dstLen])
	off := 6 + dstLen
	srcLen := int(datagram[off])
	off++
	if off+srcLen > len(datagram) {
		return "", false
	}

	plaintext, err := decryptInitialPacket(datagram, dst, saltV1, dstLen, srcLen)
	if err != nil {
		return "", false
	}
	frames, err := scanCryptoFrames(plaintext)
	if err != nil || len(frames) == 0 {
		return "", false
	}
	if len(frames) > 1 {
		// This IS Table 9/12's "Fragmentation" row (Section 6.5: "TSG
		// can ... reassemble fragmented CRYPTO frames, which may span
		// multiple UDP datagrams") -- QUIC-layer CRYPTO frame
		// fragmentation, tested here via multiple frames in one packet.
		return "quic_feature_fragmentation", true
	}
	f := frames[0]
	if f.offset > 0 || len(f.data) < minPlausibleClientHelloSize {
		return "quic_feature_multi_packet_initial", true
	}

	fields, _, err := parseWireClientHello(f.data)
	if err != nil {
		return "", false
	}
	for _, field := range fields {
		if perm, ok := classifyLengthPermutation(field); ok {
			return "tls_length_" + field.Name + "_" + perm, true
		}
	}
	// A genuinely clean, complete, single-frame v1 ClientHello: nothing
	// content-based matched, so this packet's only distinguishing feature
	// is using v1 at all.
	return "quic_version_ietf_draft29_v1", true
}

// buildVersionNegotiationPacket builds a plaintext (never encrypted, per
// RFC 9000 §17.2.1) Version Negotiation packet in reply to a client whose
// connection IDs were (dstConnID, srcConnID) -- note VN swaps them: the
// client's Source Connection ID becomes this packet's Destination
// Connection ID.
func buildVersionNegotiationPacket(clientSrcConnID connectionID, versions []uint32) []byte {
	pkt := []byte{0xc0} // long header, fixed bit set; low bits unused for VN
	pkt = append(pkt, 0, 0, 0, 0)
	pkt = append(pkt, byte(len(clientSrcConnID)))
	pkt = append(pkt, clientSrcConnID...)
	pkt = append(pkt, 0) // empty source connection ID for this reply
	for _, v := range versions {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], v)
		pkt = append(pkt, buf[:]...)
	}
	return pkt
}

// extractClientSrcConnID reads the Source Connection ID out of a client's
// Initial datagram (cleartext long-header fields only; no decryption
// needed), for the mock backend to address its Version Negotiation reply.
func extractClientSrcConnID(datagram []byte) (connectionID, error) {
	if len(datagram) < 7 {
		return nil, errors.New("mockcensor: datagram too short")
	}
	dstLen := int(datagram[5])
	off := 6 + dstLen
	if off >= len(datagram) {
		return nil, errors.New("mockcensor: truncated destination connection ID")
	}
	srcLen := int(datagram[off])
	off++
	if off+srcLen > len(datagram) {
		return nil, errors.New("mockcensor: truncated source connection ID")
	}
	return connectionID(datagram[off : off+srcLen]), nil
}

// mockBackend is a trivial "real server" stand-in: for any datagram it
// receives, it replies with a Version Negotiation packet addressed back to
// the sender, exactly as a spec-compliant QUIC endpoint would for a
// version it does not support -- which is all quicfp's own client ever
// looks for as a "the packet got through" signal (see
// dissectVersionNegotiation).
type mockBackend struct {
	conn *net.UDPConn
}

func startMockBackend() (*mockBackend, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		return nil, err
	}
	b := &mockBackend{conn: conn}
	go b.serve()
	return b, nil
}

func (b *mockBackend) addr() *net.UDPAddr { return b.conn.LocalAddr().(*net.UDPAddr) }

func (b *mockBackend) serve() {
	buf := make([]byte, 4096)
	for {
		n, from, err := b.conn.ReadFromUDP(buf)
		if err != nil {
			return // socket closed
		}
		datagram := append([]byte{}, buf[:n]...)
		srcID, err := extractClientSrcConnID(datagram)
		if err != nil {
			continue
		}
		resp := buildVersionNegotiationPacket(srcID, []uint32{1})
		_, _ = b.conn.WriteToUDP(resp, from)
	}
}

func (b *mockBackend) close() { b.conn.Close() }

// mockCensor is the software router itself: it listens as the "public"
// address quicfp targets, classifies every incoming datagram, and either
// forwards it on to backendAddr (relaying the reply back) or silently
// drops it, according to policy -- simulating exactly the on/off,
// content-and-protocol-based interference Table 9/12 describe, entirely
// over loopback.
type mockCensor struct {
	conn       *net.UDPConn
	policy     Signature
	backend    *net.UDPAddr
	forwardLog []string // vector names actually forwarded, for test introspection
	dropLog    []string // vector names actually dropped, for test introspection
}

func startMockCensor(policy Signature, backend *net.UDPAddr) (*mockCensor, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		return nil, err
	}
	c := &mockCensor{conn: conn, policy: policy, backend: backend}
	go c.serve()
	return c, nil
}

func (c *mockCensor) addr() *net.UDPAddr { return c.conn.LocalAddr().(*net.UDPAddr) }

func (c *mockCensor) serve() {
	buf := make([]byte, 4096)
	for {
		n, from, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return // socket closed
		}
		datagram := append([]byte{}, buf[:n]...)
		go c.handleOne(datagram, from)
	}
}

// decide reports whether datagram should be forwarded, per c.policy.
// Anything classifyDatagram can't identify, or that the policy has no
// (non-unknown) opinion on, defaults to forwarded -- a censor implementing
// only these specific published techniques has no basis to block traffic
// it doesn't recognize as matching one of them.
func (c *mockCensor) decide(datagram []byte) (vectorName string, forward bool) {
	name, ok := classifyDatagram(datagram)
	if !ok {
		return "", true
	}
	outcome, exists := c.policy.Vectors[name]
	if !exists || outcome == OutcomeUnknown {
		return name, true
	}
	return name, outcome == OutcomeBypassed
}

func (c *mockCensor) handleOne(datagram []byte, clientAddr *net.UDPAddr) {
	name, forward := c.decide(datagram)
	if !forward {
		c.dropLog = append(c.dropLog, name)
		return
	}
	c.forwardLog = append(c.forwardLog, name)

	backendConn, err := net.DialUDP("udp", nil, c.backend)
	if err != nil {
		return
	}
	defer backendConn.Close()
	if _, err := backendConn.Write(datagram); err != nil {
		return
	}
	_ = backendConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp := make([]byte, 2048)
	n, err := backendConn.Read(resp)
	if err != nil {
		return // backend didn't answer in time; nothing to relay
	}
	_, _ = c.conn.WriteToUDP(resp[:n], clientAddr)
}

func (c *mockCensor) close() { c.conn.Close() }
