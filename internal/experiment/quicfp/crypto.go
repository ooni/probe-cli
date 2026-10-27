package quicfp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// This file adapts the Initial-packet protection logic from OONI's own
// internal/experiment/quicping/crypto.go (itself derived from
// github.com/marten-seemann/qtls-go1-15, BSD-3-Clause, and
// github.com/quic-go/quic-go, MIT) to (a) take the Initial salt as a
// parameter instead of hardcoding QUIC v1's, so it works for every version
// vector in versions.go, and (b) implement HKDF-Expand locally via
// crypto/hmac so this package needs no dependency beyond the Go standard
// library.

const aeadNonceLength = 12

// hkdfExtract implements HKDF-Extract (RFC 5869 §2.2) using HMAC-SHA256.
func hkdfExtract(salt, ikm []byte) []byte {
	mac := hmac.New(sha256.New, salt)
	mac.Write(ikm)
	return mac.Sum(nil)
}

// hkdfExpand implements HKDF-Expand (RFC 5869 §2.3) using HMAC-SHA256,
// producing exactly length bytes of output keying material.
func hkdfExpand(prk, info []byte, length int) []byte {
	var (
		out []byte
		t   []byte
		ctr byte = 1
	)
	for len(out) < length {
		mac := hmac.New(sha256.New, prk)
		mac.Write(t)
		mac.Write(info)
		mac.Write([]byte{ctr})
		t = mac.Sum(nil)
		out = append(out, t...)
		ctr++
	}
	return out[:length]
}

// hkdfExpandLabel implements TLS 1.3's HKDF-Expand-Label (RFC 8446 §7.1),
// as used by QUIC-TLS to derive Initial/HP keys (RFC 9001 §5.1).
func hkdfExpandLabel(secret, context []byte, label string, length int) []byte {
	b := make([]byte, 3, 3+6+len(label)+1+len(context))
	binary.BigEndian.PutUint16(b, uint16(length))
	b[2] = uint8(6 + len(label))
	b = append(b, []byte("tls13 ")...)
	b = append(b, []byte(label)...)
	b = b[:3+6+len(label)+1]
	b[3+6+len(label)] = uint8(len(context))
	b = append(b, context...)
	return hkdfExpand(secret, b, length)
}

// computeSecrets derives the client and server Initial secrets from the
// destination connection ID and the version-specific Initial salt.
// https://www.rfc-editor.org/rfc/rfc9001.html#name-initial-secrets
func computeSecrets(destConnID []byte, salt [20]byte) (clientSecret, serverSecret []byte) {
	initialSecret := hkdfExtract(salt[:], destConnID)
	clientSecret = hkdfExpandLabel(initialSecret, []byte{}, "client in", sha256.Size)
	serverSecret = hkdfExpandLabel(initialSecret, []byte{}, "server in", sha256.Size)
	return
}

// keyLabelPrefix returns the prefix of the HKDF labels that derive the packet
// protection keys: "quic " (RFC 9001 §5.1), or "quicv2 " for QUIC v2 (RFC 9369
// §3.3.2). The other versions are best-effort (see versions.go) and use v1's.
func keyLabelPrefix(version uint32) string {
	if version == versionV2 {
		return "quicv2 "
	}
	return "quic "
}

// computeInitialKeyAndIV derives the AEAD key and IV from an Initial secret.
// https://www.rfc-editor.org/rfc/rfc9001.html#protection-keys
func computeInitialKeyAndIV(secret []byte, version uint32) (key, iv []byte) {
	key = hkdfExpandLabel(secret, []byte{}, keyLabelPrefix(version)+"key", 16)
	iv = hkdfExpandLabel(secret, []byte{}, keyLabelPrefix(version)+"iv", 12)
	return
}

// computeHP derives the header-protection key from an Initial secret.
func computeHP(secret []byte, version uint32) []byte {
	return hkdfExpandLabel(secret, []byte{}, keyLabelPrefix(version)+"hp", 16)
}

// xorNonceAEAD wraps an AEAD, XORing a fixed mask into the nonce before each
// call, matching TLS 1.3 record protection (RFC 8446 §5.3), which QUIC
// packet protection reuses (RFC 9001 §5.3).
type xorNonceAEAD struct {
	nonceMask [aeadNonceLength]byte
	aead      cipher.AEAD
}

func newAEADAESGCM(key, nonceMask []byte) (*xorNonceAEAD, error) {
	if len(nonceMask) != aeadNonceLength {
		return nil, errors.New("quicfp: wrong nonce length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("quicfp: aes.NewCipher failed: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("quicfp: cipher.NewGCM failed: %w", err)
	}
	ret := &xorNonceAEAD{aead: aead}
	copy(ret.nonceMask[:], nonceMask)
	return ret, nil
}

func (f *xorNonceAEAD) Seal(pn int64, plaintext, additionalData []byte) []byte {
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, uint64(pn))
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	out := f.aead.Seal(nil, f.nonceMask[:], plaintext, additionalData)
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	return out
}

// Open reverses Seal, for tests that want to confirm a built packet
// round-trips back to the intended plaintext.
func (f *xorNonceAEAD) Open(pn int64, ciphertext, additionalData []byte) ([]byte, error) {
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, uint64(pn))
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	out, err := f.aead.Open(nil, f.nonceMask[:], ciphertext, additionalData)
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	return out, err
}

// encryptPayload applies Initial packet protection (AEAD) to the payload
// (the concatenated frames that make up the packet body), using packet
// number pn and the associated-data (unprotected header) hdr.
// https://www.rfc-editor.org/rfc/rfc9001.html#name-packet-protection
func encryptPayload(payload []byte, clientSecret []byte, hdr []byte, pn int64, version uint32) ([]byte, error) {
	key, iv := computeInitialKeyAndIV(clientSecret, version)
	aead, err := newAEADAESGCM(key, iv)
	if err != nil {
		return nil, err
	}
	return aead.Seal(pn, payload, hdr), nil
}

// encryptHeader applies QUIC header protection to raw (which must already
// contain the AEAD-sealed payload appended after hdr).
// https://www.rfc-editor.org/rfc/rfc9001.html#name-header-protection
func encryptHeader(raw, hdr, clientSecret []byte, pnLen int, version uint32) ([]byte, error) {
	hp := computeHP(clientSecret, version)
	block, err := aes.NewCipher(hp)
	if err != nil {
		return nil, fmt.Errorf("quicfp: aes.NewCipher failed: %w", err)
	}
	payloadOffset := len(hdr)
	if payloadOffset+16 > len(raw) {
		return nil, errors.New("quicfp: packet too short to sample for header protection")
	}
	sample := raw[payloadOffset : payloadOffset+16]
	mask := make([]byte, block.BlockSize())
	block.Encrypt(mask, sample)
	// long header: protect the low 4 bits of the first byte
	raw[0] ^= mask[0] & 0x0f
	pnOffset := len(hdr) - pnLen
	pnBytes := raw[pnOffset:payloadOffset]
	for i := range pnBytes {
		pnBytes[i] ^= mask[i+1]
	}
	return raw, nil
}

// decryptInitialPacket reverses sealPacket/cryptoProtect given the known
// destination/source connection ID lengths, for use in tests. Header
// protection is XOR-based and thus self-inverting: the sample it uses comes
// entirely from the still-ciphertext payload, which is unaffected by
// toggling the header bits, so applying encryptHeader a second time with a
// same-length (content-independent) header argument exactly undoes it.
func decryptInitialPacket(raw []byte, dstConnID connectionID, salt [20]byte, dstLen, srcLen int) (plaintext []byte, err error) {
	offset := 1 + 4 + 1 + dstLen + 1 + srcLen
	if offset >= len(raw) {
		return nil, errors.New("quicfp: packet too short (token length)")
	}
	tokenLen, n := varintDecode(raw[offset:])
	offset += n + int(tokenLen)
	if offset >= len(raw) {
		return nil, errors.New("quicfp: packet too short (length field)")
	}
	remLen, n2 := varintDecode(raw[offset:])
	offset += n2
	hdrLen := offset + pnLen
	if hdrLen > len(raw) {
		return nil, errors.New("quicfp: packet too short (header)")
	}

	clientSecret, _ := computeSecrets(dstConnID, salt)
	version := binary.BigEndian.Uint32(raw[1:5])

	// Undo header protection (self-inverse XOR, see comment above).
	work := append([]byte{}, raw...)
	if _, err := encryptHeader(work, make([]byte, hdrLen), clientSecret, pnLen, version); err != nil {
		return nil, fmt.Errorf("quicfp: undoing header protection: %w", err)
	}

	pn := int64(binary.BigEndian.Uint32(work[hdrLen-pnLen : hdrLen]))
	ciphertext := work[hdrLen:]
	if uint64(len(ciphertext)) != remLen-uint64(pnLen) {
		return nil, fmt.Errorf("quicfp: length mismatch: got %d want %d", len(ciphertext), remLen-uint64(pnLen))
	}

	key, iv := computeInitialKeyAndIV(clientSecret, version)
	aead, err := newAEADAESGCM(key, iv)
	if err != nil {
		return nil, err
	}
	return aead.Open(pn, ciphertext, work[:hdrLen])
}

// cryptoProtect is a convenience wrapper: given the fully-formed unprotected
// header hdr (whose last pnLen bytes are the cleartext packet number) and
// the plaintext frame payload, it returns the fully protected UDP datagram
// bytes ready to send.
func cryptoProtect(hdr, payload, destConnID []byte, salt [20]byte, pn int64, pnLen int, version uint32) ([]byte, error) {
	clientSecret, _ := computeSecrets(destConnID, salt)
	sealed, err := encryptPayload(payload, clientSecret, hdr, pn, version)
	if err != nil {
		return nil, err
	}
	raw := append(append([]byte{}, hdr...), sealed...)
	return encryptHeader(raw, hdr, clientSecret, pnLen, version)
}
