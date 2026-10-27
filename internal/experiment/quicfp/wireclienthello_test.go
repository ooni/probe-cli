package quicfp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// wireField is one field's declared (on-the-wire) value alongside what the
// correct value would be given the ClientHello's actual byte layout, which
// never changes regardless of which length prefix a vector patched.
type wireField struct {
	Name     string
	Width    int
	Declared int
	Correct  int
}

// parseWireClientHello reads a ClientHello built by buildClientHello (any
// of its eight length fields possibly patched by applyPermutation) and
// recovers, for each field, both its on-the-wire declared value and what
// the correct value actually is. This mirrors buildClientHello's own
// control flow exactly -- same fixed extension order, same offsets -- just
// reading instead of writing, which works regardless of which single field
// was mangled: patching only ever changes a length prefix's declared
// value, never the actual surrounding byte layout.
func parseWireClientHello(raw []byte) (fields []wireField, sni string, err error) {
	if len(raw) < 4 {
		return nil, "", errors.New("mockcensor: ClientHello too short")
	}
	if raw[0] != 0x01 {
		return nil, "", fmt.Errorf("mockcensor: not a client_hello (msg_type=0x%02x)", raw[0])
	}
	msgLenDeclared := int(raw[1])<<16 | int(raw[2])<<8 | int(raw[3])
	bodyStart := 4
	msgLenCorrect := len(raw) - bodyStart

	off := bodyStart
	need := func(n int) error {
		if off+n > len(raw) {
			return errors.New("mockcensor: truncated ClientHello")
		}
		return nil
	}

	if err := need(2 + 32); err != nil {
		return nil, "", err
	}
	off += 2  // legacy_version
	off += 32 // random

	if err := need(1); err != nil {
		return nil, "", err
	}
	sessionIDDeclared := int(raw[off])
	off += 1 + 0 // session_id is always empty on the wire regardless of the declared length

	if err := need(2); err != nil {
		return nil, "", err
	}
	cipherSuiteLenDeclared := int(binary.BigEndian.Uint16(raw[off : off+2]))
	off += 2
	// quicfp always writes exactly 6 bytes (3 suites) here, regardless of
	// what the length prefix claims.
	if err := need(6); err != nil {
		return nil, "", err
	}
	off += 6

	if err := need(2); err != nil {
		return nil, "", err
	}
	compressionMethodsDeclared := int(raw[off])
	off += 2 // 1-byte length + 1 null compression method

	if err := need(2); err != nil {
		return nil, "", err
	}
	extsLenDeclared := int(binary.BigEndian.Uint16(raw[off : off+2]))
	off += 2
	extsStart := off

	// server_name extension: type(2) + extension_data length(2) + [
	//   server_name_list length(2) + name_type(1) + host_name length(2) + host_name ]
	if err := need(2); err != nil {
		return nil, "", err
	}
	off += 2 // extension type 0x0000
	if err := need(2); err != nil {
		return nil, "", err
	}
	sniExtLenDeclared := int(binary.BigEndian.Uint16(raw[off : off+2]))
	off += 2
	sniExtDataStart := off

	if err := need(2); err != nil {
		return nil, "", err
	}
	sniListLenDeclared := int(binary.BigEndian.Uint16(raw[off : off+2]))
	off += 2
	sniListStart := off

	if err := need(1); err != nil {
		return nil, "", err
	}
	off += 1 // name_type
	if err := need(2); err != nil {
		return nil, "", err
	}
	sniNameLenDeclared := int(binary.BigEndian.Uint16(raw[off : off+2]))
	off += 2
	hostnameStart := off

	// The declared SNI Name Length is the one field where, if mangled, a
	// naive parser reading exactly that many bytes would desync every
	// offset after it -- unlike the other seven fields, where the prefix
	// lies but the actual byte layout around it never moves. Real DPI
	// faces exactly this problem; we sidestep it the way a parser with
	// full knowledge of quicfp's fixed schema can: everything except the
	// hostname itself is fixed-size, so the true hostname length is
	// recoverable algebraically from the total packet size regardless of
	// what the declared prefix claims.
	fixedTail := lastExtensionSizeOf() + // key_share
		(2 + 2 + 2 + 2) + // signature_algorithms: type+len+listlen+scheme
		(2 + 2 + 2 + 2) + // supported_groups: type+len+listlen+group
		(2 + 2 + 1 + 2) // supported_versions: type+len+listlen+version
	sniNameLenCorrect := len(raw) - hostnameStart - fixedTail
	if sniNameLenCorrect < 0 || hostnameStart+sniNameLenCorrect > len(raw) {
		return nil, "", fmt.Errorf("mockcensor: computed hostname length %d is impossible for a %d-byte ClientHello", sniNameLenCorrect, len(raw))
	}
	sni = string(raw[hostnameStart : hostnameStart+sniNameLenCorrect])
	off = hostnameStart + sniNameLenCorrect

	sniListLenCorrect := off - sniListStart
	sniExtLenCorrect := off - sniExtDataStart

	// The remaining extensions (supported_versions, supported_groups,
	// signature_algorithms, key_share) are fixed-size and always present
	// after the server_name extension -- skip over them structurally to
	// find the true end of the extensions block, exactly like
	// buildClientHello's own layout.
	lastExtStart := off + (2 + 2 + 1 + 2) + (2 + 2 + 2 + 2) + (2 + 2 + 2 + 2)
	if err := need(fixedTail); err != nil {
		return nil, "", err
	}
	off += fixedTail
	if got := off - lastExtStart; got != lastExtensionSizeOf() {
		return nil, "", fmt.Errorf("mockcensor: unexpected last-extension size %d, want %d", got, lastExtensionSizeOf())
	}

	extsLenCorrect := off - extsStart
	if off != len(raw) {
		return nil, "", fmt.Errorf("mockcensor: %d trailing bytes after expected ClientHello end", len(raw)-off)
	}

	fields = []wireField{
		{"message_length", 3, msgLenDeclared, msgLenCorrect},
		{"session_id_length", 1, sessionIDDeclared, 0},
		{"cipher_suite_length", 2, cipherSuiteLenDeclared, 6},
		{"compression_methods_length", 1, compressionMethodsDeclared, 1},
		{"extensions_length", 2, extsLenDeclared, extsLenCorrect},
		{"sni_extension_length", 2, sniExtLenDeclared, sniExtLenCorrect},
		{"sni_list_length", 2, sniListLenDeclared, sniListLenCorrect},
		{"sni_name_length", 2, sniNameLenDeclared, sniNameLenCorrect},
	}
	return fields, sni, nil
}

// lastExtensionSizeOf re-derives the last extension's on-wire size from a
// parsed ClientHello, for classifyLengthPermutation's exclude_last_extension
// case. Since the tail after the server_name extension is fixed-size and
// fixed-content in every ClientHello quicfp builds, this is a constant:
// key_share = type(2) + extension_data length(2) + client_shares list
// length(2) + group(2) + key length(2) + key(32) = 42 bytes.
func lastExtensionSizeOf() int {
	return 2 + 2 + 2 + 2 + 2 + 32
}

// classifyLengthPermutation compares f.Declared against f.Correct and
// names which permutation (see PermZero etc.) produces that declared
// value, mirroring applyPermutation's own arithmetic in reverse. ok is
// false if the field looks unmangled (declared == correct).
//
// Two fields have a genuine, unavoidable encoding ambiguity here, not a
// classifier bug: session_id_length's "too_long" permutation computes the
// exact same byte value as "max" would (both write the field's maximum),
// so it's special-cased below by field name; and compression_methods_length
// has Correct=1, so PermHalf (Correct/2, integer division) produces 0,
// bit-identical to PermZero -- a real DPI parsing this field cannot tell
// those two apart either, so this function reports "zero" for both and
// callers must not expect "half" back for this specific field.
func classifyLengthPermutation(f wireField) (permutation string, ok bool) {
	if f.Declared == f.Correct {
		return "", false
	}
	maxVal := 1<<(8*f.Width) - 1
	switch {
	case f.Name == "session_id_length" && f.Declared == maxVal:
		return PermTooLong, true
	case f.Declared == 0:
		return PermZero, true
	case f.Declared == maxVal:
		return PermMax, true
	case f.Declared == f.Correct/2:
		return PermHalf, true
	case f.Declared == (f.Correct*2)&maxVal:
		return PermDouble, true
	case (f.Name == "message_length" || f.Name == "extensions_length") && f.Declared == f.Correct-lastExtensionSizeOf():
		return PermExcludeLastExtension, true
	default:
		// Declared differs from correct but doesn't match any known
		// permutation shape (e.g. it's the SNI itself changing length
		// for a real, different SNI) -- not a length-field vector.
		return "", false
	}
}
