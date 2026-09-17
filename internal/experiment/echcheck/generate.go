package echcheck

// Generates a 'GREASE ECH' extension, as described in section 6.2 of
// ietf.org/archive/id/draft-ietf-tls-esni-14.html

import (
	"io"

	"github.com/cloudflare/circl/hpke"
	"golang.org/x/crypto/cryptobyte"
)

// ECH Config List per:
// https://www.ietf.org/archive/id/draft-ietf-tls-esni-22.html#name-encrypted-clienthello-confi
func generateGreaseyECHConfigList(rand io.Reader, publicName string) ([]byte, error) {
	// Start ECHConfig
	var c cryptobyte.Builder
	version := uint16(0xfe0d)
	c.AddUint16(version)

	// Start ECHConfigContents
	var ecc cryptobyte.Builder
	// Start HpkeKeyConfig
	randConfigId := make([]byte, 1)
	if _, err := io.ReadFull(rand, randConfigId); err != nil {
		return nil, err
	}
	ecc.AddUint8(randConfigId[0])
	ecc.AddUint16(uint16(hpke.KEM_X25519_HKDF_SHA256))
	// Generate a public key
	kem := hpke.KEM(uint16(hpke.KEM_X25519_HKDF_SHA256))
	publicKey, _, err := kem.Scheme().GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	publicKeyBytes, err := publicKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ecc.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(publicKeyBytes)
	})
	// Start HpkeSymmetricCipherSuite
	kdf := hpke.KDF(uint16(hpke.KDF_HKDF_SHA256))
	aead := hpke.AEAD(uint16(hpke.AEAD_AES128GCM))
	var cs cryptobyte.Builder
	cs.AddUint16(uint16(kdf))
	cs.AddUint16(uint16(aead))
	ecc.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(cs.BytesOrPanic())
	})
	// End HpkeSymmetricCipherSuite
	// End HpkeKeyConfig
	maxNameLength := uint8(42)
	ecc.AddUint8(maxNameLength)
	publicNameBytes := []byte(publicName)
	ecc.AddUint8LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(publicNameBytes)
	})
	// Start ECHConfigExtension
	var ece cryptobyte.Builder
	ecc.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(ece.BytesOrPanic())
	})
	// End ECHConfigExtension
	// End ECHConfigContents
	c.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(ecc.BytesOrPanic())
	})
	// End ECHConfig
	var l cryptobyte.Builder
	l.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(c.BytesOrPanic())
	})

	return l.BytesOrPanic(), nil
}
