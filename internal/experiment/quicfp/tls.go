package quicfp

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// permutation names, matching the methodology in Section 6.4/6.5 of the
// paper: each length field is set to zero, half its correct value, double
// its correct value, its field-width maximum, and (for Message Length and
// Extensions Length only) a value that excludes the final extension.
// PermTooLong is used only for Session ID Length, which the paper tests
// with a single "too long" value rather than the zero/half/double/max set
// (the field's correct/default value is 0, so "half"/"double" of it would
// be meaningless).
const (
	PermZero                 = "zero"
	PermHalf                 = "half"
	PermDouble               = "double"
	PermMax                  = "max"
	PermExcludeLastExtension = "exclude_last_extension"
	PermTooLong              = "too_long"
)

// tlsField locates one patchable length field inside a built ClientHello.
type tlsField struct {
	// Offset is the byte offset of the field within the ClientHello.
	Offset int
	// Width is the field's size in bytes (2 or 3).
	Width int
	// Correct is the field's correct (spec-compliant) value.
	Correct int
}

// clientHello holds a built ClientHello and the offsets of every field
// Table 12 manipulates for QUIC (Table 9 only surfaces four of these as a
// condensed view; the other four -- Session ID / Cipher Suite /
// Compression Methods / SNI Name Length -- are additional rows Table 12
// tests that Table 9 collapses out of its summary).
type clientHello struct {
	Raw []byte

	MessageLength            tlsField // the Handshake header's 3-byte length field
	SessionIDLength          tlsField // the 1-byte legacy_session_id length prefix
	CipherSuiteLength        tlsField // the 2-byte cipher_suites length prefix
	CompressionMethodsLength tlsField // the 1-byte legacy_compression_methods length prefix
	ExtensionsLength         tlsField // the 2-byte length prefix of the extensions block
	SNIExtensionLength       tlsField // the 2-byte extension_data length of the server_name extension
	SNIListLength            tlsField // the 2-byte server_name_list length inside that extension_data
	SNINameLength            tlsField // the 2-byte HostName length inside the ServerName entry

	// lastExtensionSize is the on-wire size (type+length+data) of the
	// final extension, used to compute the "exclude_last_extension"
	// permutation for MessageLength and ExtensionsLength.
	lastExtensionSize int
}

// chBuilder is a small append-only byte builder that also lets us patch
// length-prefixed sections after the fact once we know their final size.
type chBuilder struct {
	buf []byte
}

func (b *chBuilder) u8(v byte)      { b.buf = append(b.buf, v) }
func (b *chBuilder) bytes(v []byte) { b.buf = append(b.buf, v...) }

func (b *chBuilder) u16(v uint16) {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], v)
	b.buf = append(b.buf, tmp[:]...)
}

func (b *chBuilder) u24(v uint32) {
	b.buf = append(b.buf, byte(v>>16), byte(v>>8), byte(v))
}

// reserveU16 appends a placeholder 2-byte length field and returns its
// offset, so the caller can patch in the real value once known.
func (b *chBuilder) reserveU16() int {
	off := len(b.buf)
	b.u16(0)
	return off
}

func (b *chBuilder) patchU16(off int, v int) {
	binary.BigEndian.PutUint16(b.buf[off:off+2], uint16(v))
}

func (b *chBuilder) reserveU24() int {
	off := len(b.buf)
	b.u24(0)
	return off
}

func (b *chBuilder) patchU24(off int, v int) {
	b.buf[off], b.buf[off+1], b.buf[off+2] = byte(v>>16), byte(v>>8), byte(v)
}

// buildClientHello constructs a minimal, well-formed TLS 1.3 ClientHello
// (handshake header included, i.e. what a QUIC CRYPTO frame actually
// carries) advertising SNI sni, and returns it together with the offsets of
// the four fields exercised by Table 9.
func buildClientHello(sni string) (*clientHello, error) {
	b := &chBuilder{}

	// -- Handshake header --------------------------------------------
	b.u8(0x01) // msg_type = client_hello
	msgLenOff := b.reserveU24()
	bodyStart := len(b.buf)

	// -- ClientHello body ----------------------------------------------
	b.u16(0x0303) // legacy_version = TLS 1.2 (compatibility value)

	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("quicfp: rand.Read failed: %w", err)
	}
	b.bytes(random)

	sessionIDLenOff := len(b.buf)
	b.u8(0x00) // legacy_session_id length = 0 (empty)

	// cipher_suites: three standard TLS 1.3 suites.
	cipherSuites := []byte{
		0x13, 0x01, // TLS_AES_128_GCM_SHA256
		0x13, 0x02, // TLS_AES_256_GCM_SHA384
		0x13, 0x03, // TLS_CHACHA20_POLY1305_SHA256
	}
	cipherSuiteLenOff := len(b.buf)
	b.u16(uint16(len(cipherSuites)))
	b.bytes(cipherSuites)

	compressionMethodsLenOff := len(b.buf)
	b.u8(0x01) // legacy_compression_methods length = 1
	b.u8(0x00) // null compression

	// -- extensions ------------------------------------------------------
	extsLenOff := b.reserveU16()
	extsStart := len(b.buf)

	// server_name (SNI), extension type 0x0000. We record its offsets
	// precisely because this is one of the four fields Table 9 tests.
	b.u16(0x0000)
	sniExtLenOff := b.reserveU16()
	sniExtDataStart := len(b.buf)

	sniListLenOff := b.reserveU16()
	sniListStart := len(b.buf)
	b.u8(0x00) // name_type = host_name
	sniNameLenOff := len(b.buf)
	b.u16(uint16(len(sni)))
	b.bytes([]byte(sni))
	sniListLen := len(b.buf) - sniListStart
	b.patchU16(sniListLenOff, sniListLen)

	sniExtDataLen := len(b.buf) - sniExtDataStart
	b.patchU16(sniExtLenOff, sniExtDataLen)

	// supported_versions (0x002b): advertise TLS 1.3 only.
	b.u16(0x002b)
	b.u16(3) // extension_data length
	b.u8(2)  // list length
	b.u16(0x0304)

	// supported_groups (0x000a): a single group, x25519.
	b.u16(0x000a)
	b.u16(4)      // extension_data length
	b.u16(2)      // list length
	b.u16(0x001d) // x25519

	// signature_algorithms (0x000d): a single scheme.
	b.u16(0x000d)
	b.u16(4)      // extension_data length
	b.u16(2)      // list length
	b.u16(0x0403) // ecdsa_secp256r1_sha256

	// key_share (0x0033): one x25519 key share with a placeholder key.
	// This is the final extension, used for exclude_last_extension.
	lastExtStart := len(b.buf)
	b.u16(0x0033)
	keyShareBody := make([]byte, 32)
	if _, err := rand.Read(keyShareBody); err != nil {
		return nil, fmt.Errorf("quicfp: rand.Read failed: %w", err)
	}
	b.u16(uint16(2 + 2 + 2 + len(keyShareBody))) // extension_data length
	b.u16(uint16(2 + 2 + len(keyShareBody)))     // client_shares list length
	b.u16(0x001d)                                // group = x25519
	b.u16(uint16(len(keyShareBody)))
	b.bytes(keyShareBody)
	lastExtSize := len(b.buf) - lastExtStart

	extsLen := len(b.buf) - extsStart
	b.patchU16(extsLenOff, extsLen)

	bodyLen := len(b.buf) - bodyStart
	b.patchU24(msgLenOff, bodyLen)

	ch := &clientHello{
		Raw:                      b.buf,
		MessageLength:            tlsField{Offset: msgLenOff, Width: 3, Correct: bodyLen},
		SessionIDLength:          tlsField{Offset: sessionIDLenOff, Width: 1, Correct: 0},
		CipherSuiteLength:        tlsField{Offset: cipherSuiteLenOff, Width: 2, Correct: len(cipherSuites)},
		CompressionMethodsLength: tlsField{Offset: compressionMethodsLenOff, Width: 1, Correct: 1},
		ExtensionsLength:         tlsField{Offset: extsLenOff, Width: 2, Correct: extsLen},
		SNIExtensionLength:       tlsField{Offset: sniExtLenOff, Width: 2, Correct: sniExtDataLen},
		SNIListLength:            tlsField{Offset: sniListLenOff, Width: 2, Correct: sniListLen},
		SNINameLength:            tlsField{Offset: sniNameLenOff, Width: 2, Correct: len(sni)},
		lastExtensionSize:        lastExtSize,
	}
	return ch, nil
}

// applyPermutation returns a copy of ch.Raw with the given field patched to
// the value permutation dictates. It panics on an unknown permutation name
// or a permutation not valid for the given field (callers only ever pass
// combinations vectors.go itself generates, so this is a programmer error,
// not a runtime/network condition).
func applyPermutation(ch *clientHello, field tlsField, permutation string) []byte {
	out := make([]byte, len(ch.Raw))
	copy(out, ch.Raw)

	maxVal := 1<<(8*field.Width) - 1
	var value int
	switch permutation {
	case PermZero:
		value = 0
	case PermHalf:
		value = field.Correct / 2
	case PermDouble:
		value = (field.Correct * 2) & maxVal
	case PermMax, PermTooLong:
		value = maxVal
	case PermExcludeLastExtension:
		value = field.Correct - ch.lastExtensionSize
	default:
		panic("quicfp: unknown permutation " + permutation)
	}

	switch field.Width {
	case 1:
		out[field.Offset] = byte(value)
	case 2:
		binary.BigEndian.PutUint16(out[field.Offset:field.Offset+2], uint16(value))
	case 3:
		out[field.Offset], out[field.Offset+1], out[field.Offset+2] =
			byte(value>>16), byte(value>>8), byte(value)
	default:
		panic("quicfp: unsupported field width")
	}
	return out
}
