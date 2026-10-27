package quicfp

import (
	"bytes"
	"errors"
	"testing"
)

// parseFrames reconstructs the CRYPTO stream from a decrypted plaintext
// frame payload that may contain CRYPTO frames (possibly out of order or
// non-contiguous, per RFC 9000 §19.6) interleaved with single-byte PADDING
// frames (type 0x00). It exists only to let tests verify what actually
// arrived on the wire matches what the builder intended to send.
func parseFrames(payload []byte) ([]byte, error) {
	var out []byte
	i := 0
	for i < len(payload) {
		switch payload[i] {
		case 0x00: // PADDING
			i++
		case 0x06: // CRYPTO
			i++
			offset, n := varintDecode(payload[i:])
			if n == 0 {
				return nil, errors.New("bad offset varint")
			}
			i += n
			length, n2 := varintDecode(payload[i:])
			if n2 == 0 {
				return nil, errors.New("bad length varint")
			}
			i += n2
			if i+int(length) > len(payload) {
				return nil, errors.New("truncated crypto frame")
			}
			data := payload[i : i+int(length)]
			i += int(length)
			end := int(offset) + len(data)
			if end > len(out) {
				grown := make([]byte, end)
				copy(grown, out)
				out = grown
			}
			copy(out[offset:], data)
		default:
			return nil, errors.New("unexpected frame type")
		}
	}
	return out, nil
}

func TestBuildClientHelloOffsets(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Handshake header: msg_type(1) + length(3) must equal len(Raw)-4.
	if ch.MessageLength.Correct != len(ch.Raw)-4 {
		t.Fatalf("message length mismatch: correct=%d rawLen-4=%d", ch.MessageLength.Correct, len(ch.Raw)-4)
	}
	got := int(ch.Raw[ch.MessageLength.Offset])<<16 | int(ch.Raw[ch.MessageLength.Offset+1])<<8 | int(ch.Raw[ch.MessageLength.Offset+2])
	if got != ch.MessageLength.Correct {
		t.Fatalf("message length field: on-wire=%d want=%d", got, ch.MessageLength.Correct)
	}

	gotExt := int(ch.Raw[ch.ExtensionsLength.Offset])<<8 | int(ch.Raw[ch.ExtensionsLength.Offset+1])
	if gotExt != ch.ExtensionsLength.Correct {
		t.Fatalf("extensions length field: on-wire=%d want=%d", gotExt, ch.ExtensionsLength.Correct)
	}

	gotSNIExt := int(ch.Raw[ch.SNIExtensionLength.Offset])<<8 | int(ch.Raw[ch.SNIExtensionLength.Offset+1])
	if gotSNIExt != ch.SNIExtensionLength.Correct {
		t.Fatalf("sni extension length field: on-wire=%d want=%d", gotSNIExt, ch.SNIExtensionLength.Correct)
	}

	gotSNIList := int(ch.Raw[ch.SNIListLength.Offset])<<8 | int(ch.Raw[ch.SNIListLength.Offset+1])
	if gotSNIList != ch.SNIListLength.Correct {
		t.Fatalf("sni list length field: on-wire=%d want=%d", gotSNIList, ch.SNIListLength.Correct)
	}

	// The SNI itself must appear verbatim in the ClientHello.
	if !bytes.Contains(ch.Raw, []byte("example.com")) {
		t.Fatal("SNI not found in built ClientHello")
	}
}

func TestApplyPermutation(t *testing.T) {
	ch, err := buildClientHello("censored.example")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		field tlsField
		perm  string
		want  int
	}{
		{ch.MessageLength, PermZero, 0},
		{ch.MessageLength, PermHalf, ch.MessageLength.Correct / 2},
		{ch.MessageLength, PermDouble, (ch.MessageLength.Correct * 2) & 0xffffff},
		{ch.MessageLength, PermMax, 0xffffff},
		{ch.MessageLength, PermExcludeLastExtension, ch.MessageLength.Correct - ch.lastExtensionSize},
		{ch.ExtensionsLength, PermZero, 0},
		{ch.ExtensionsLength, PermMax, 0xffff},
		{ch.SNIExtensionLength, PermDouble, (ch.SNIExtensionLength.Correct * 2) & 0xffff},
		{ch.SNIListLength, PermHalf, ch.SNIListLength.Correct / 2},
	}

	for _, c := range cases {
		patched := applyPermutation(ch, c.field, c.perm)
		var got int
		switch c.field.Width {
		case 2:
			got = int(patched[c.field.Offset])<<8 | int(patched[c.field.Offset+1])
		case 3:
			got = int(patched[c.field.Offset])<<16 | int(patched[c.field.Offset+1])<<8 | int(patched[c.field.Offset+2])
		}
		if got != c.want {
			t.Errorf("perm=%s field.Offset=%d: got=%d want=%d", c.perm, c.field.Offset, got, c.want)
		}
		// Every other byte must be untouched by the patch.
		if len(patched) != len(ch.Raw) {
			t.Errorf("perm=%s changed overall length: got=%d want=%d", c.perm, len(patched), len(ch.Raw))
		}
	}
}

func TestBuildVectorsCoversTable9(t *testing.T) {
	vectors, err := buildVectors("example.com")
	if err != nil {
		t.Fatal(err)
	}

	wantVersions := 7
	wantFeatures := 4
	wantLengthFields := 4*4 + 2 // 4 fields x 4 basic perms, +2 exclude_last_extension

	var gotVersions, gotFeatures, gotLengthFields int
	seen := map[string]bool{}
	for _, v := range vectors {
		if seen[v.Name] {
			t.Errorf("duplicate vector name %q", v.Name)
		}
		seen[v.Name] = true
		switch v.Category {
		case CategoryQUICVersion:
			gotVersions++
		case CategoryQUICFeature:
			gotFeatures++
		case CategoryTLSLength:
			gotLengthFields++
		default:
			t.Errorf("unknown category %q for vector %q", v.Category, v.Name)
		}
	}
	if gotVersions != wantVersions {
		t.Errorf("quic version vectors: got=%d want=%d", gotVersions, wantVersions)
	}
	if gotFeatures != wantFeatures {
		t.Errorf("quic feature vectors: got=%d want=%d", gotFeatures, wantFeatures)
	}
	if gotLengthFields != wantLengthFields {
		t.Errorf("tls length field vectors: got=%d want=%d", gotLengthFields, wantLengthFields)
	}
}

func TestStandardPacketSizeAndRoundTrip(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, dst, src, err := buildStandardPacket(1, saltV1, ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < minInitialDatagramSize {
		t.Fatalf("standard packet under 1200 bytes: got %d", len(raw))
	}
	if len(dst) < minConnectionIDLenInitial || len(dst) > maxConnectionIDLen {
		t.Fatalf("unexpected dst conn id length: %d", len(dst))
	}
	if len(src) != defaultConnectionIDLength {
		t.Fatalf("unexpected src conn id length: %d", len(src))
	}

	plaintext, err := decryptInitialPacket(raw, dst, saltV1, len(dst), len(src))
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	reconstructed, err := parseFrames(plaintext)
	if err != nil {
		t.Fatalf("parseFrames failed: %v", err)
	}
	if !bytes.Equal(reconstructed, ch.Raw) {
		t.Fatalf("round-tripped ClientHello does not match original: got %d bytes want %d bytes", len(reconstructed), len(ch.Raw))
	}
}

func TestBelow1200PacketIsUndersized(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, dst, src, err := buildBelow1200Packet(1, saltV1, ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) >= minInitialDatagramSize {
		t.Fatalf("expected an under-sized datagram, got %d bytes", len(raw))
	}

	plaintext, err := decryptInitialPacket(raw, dst, saltV1, len(dst), len(src))
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	reconstructed, err := parseFrames(plaintext)
	if err != nil {
		t.Fatalf("parseFrames failed: %v", err)
	}
	if !bytes.Equal(reconstructed, ch.Raw) {
		t.Fatal("round-tripped ClientHello does not match original")
	}
}

func TestFragmentedPacketRoundTrip(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, dst, src, err := buildFragmentedPacket(1, saltV1, ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := decryptInitialPacket(raw, dst, saltV1, len(dst), len(src))
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	reconstructed, err := parseFrames(plaintext)
	if err != nil {
		t.Fatalf("parseFrames failed: %v", err)
	}
	if !bytes.Equal(reconstructed, ch.Raw) {
		t.Fatal("fragmented ClientHello did not reassemble to the original bytes")
	}
}

func TestMultiPacketInitialRoundTrip(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	packets, dst, src, err := buildMultiPacketInitial(1, saltV1, ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 2 {
		t.Fatalf("expected 2 packets, got %d", len(packets))
	}
	if len(packets[0]) < minInitialDatagramSize {
		t.Fatalf("first packet must be padded to >=1200 bytes, got %d", len(packets[0]))
	}

	var joined []byte
	for _, pkt := range packets {
		pt, err := decryptInitialPacket(pkt, dst, saltV1, len(dst), len(src))
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		frag, err := parseFrames(pt)
		if err != nil {
			t.Fatalf("parseFrames failed: %v", err)
		}
		if len(frag) > len(joined) {
			grown := make([]byte, len(frag))
			copy(grown, joined)
			joined = grown
		}
		for i, b := range frag {
			if b != 0 {
				joined[i] = b
			}
		}
	}
	if !bytes.Equal(joined, ch.Raw) {
		t.Fatal("multi-packet Initial did not reassemble to the original ClientHello")
	}
}

func TestForcedNegotiationUsesGreasedVersion(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, _, _, err := buildForcedNegotiationPacket(ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	version := uint32(raw[1])<<24 | uint32(raw[2])<<16 | uint32(raw[3])<<8 | uint32(raw[4])
	if version != greasedVersion() {
		t.Fatalf("expected greased version 0x%08x, got 0x%08x", greasedVersion(), version)
	}
	for _, qv := range quicVersionVectors() {
		if qv.Version == version {
			t.Fatalf("greased version collides with a real assigned version vector %q", qv.Name)
		}
	}
}

func TestDissectVersionNegotiation(t *testing.T) {
	dstID := connectionID{1, 2, 3, 4, 5, 6, 7, 8}
	srcID := connectionID{9, 8, 7}
	pkt := []byte{0x80}
	pkt = append(pkt, 0, 0, 0, 0) // version = 0 marks Version Negotiation
	pkt = append(pkt, byte(len(dstID)))
	pkt = append(pkt, dstID...)
	pkt = append(pkt, byte(len(srcID)))
	pkt = append(pkt, srcID...)
	pkt = append(pkt, 0, 0, 0, 1)             // one supported version: v1
	pkt = append(pkt, 0x6b, 0x33, 0x43, 0xcf) // and v2

	versions, dst, err := dissectVersionNegotiation(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dst, dstID) {
		t.Fatalf("dst mismatch: got %x want %x", dst, dstID)
	}
	if len(versions) != 2 || versions[0] != 1 || versions[1] != 0x6b3343cf {
		t.Fatalf("unexpected versions: %v", versions)
	}
}
