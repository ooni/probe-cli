package echcheck

// Generates a 'GREASE ECH' extension, as described in section 6.2 of
// ietf.org/archive/id/draft-ietf-tls-esni-14.html

import (
	"fmt"
	"github.com/cloudflare/circl/hpke"
	"golang.org/x/crypto/cryptobyte"
	"io"
)

const clientHelloOuter uint8 = 0

// echExtension is the Encrypted Client Hello extension that is part of
// ClientHelloOuter as specified in:
// ietf.org/archive/id/draft-ietf-tls-esni-14.html#section-5
type echExtension struct {
	kdfID    uint16
	aeadID   uint16
	configID uint8
	enc      []byte
	payload  []byte
}

func (ech *echExtension) marshal() []byte {
	var b cryptobyte.Builder
	b.AddUint8(clientHelloOuter)
	b.AddUint16(ech.kdfID)
	b.AddUint16(ech.aeadID)
	b.AddUint8(ech.configID)
	b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(ech.enc)
	})
	b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(ech.payload)
	})
	return b.BytesOrPanic()
}

// generateGreaseExtension generates an ECH extension with random values as
// specified in ietf.org/archive/id/draft-ietf-tls-esni-14.html#section-6.2
func generateGreaseExtension(rand io.Reader) ([]byte, error) {
	// initialize HPKE suite parameters
	kem := hpke.KEM(uint16(hpke.KEM_X25519_HKDF_SHA256))
	kdf := hpke.KDF(uint16(hpke.KDF_HKDF_SHA256))
	aead := hpke.AEAD(uint16(hpke.AEAD_AES128GCM))

	if !kem.IsValid() || !kdf.IsValid() || !aead.IsValid() {
		return nil, fmt.Errorf("required parameters not supported")
	}

	defaultHPKESuite := hpke.NewSuite(kem, kdf, aead)

	// generate a public key to place in 'enc' field
	publicKey, _, err := kem.Scheme().GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %s", err)
	}

	// initiate HPKE Sender
	sender, err := defaultHPKESuite.NewSender(publicKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender: %s", err)
	}

	// Set ECH Extension Fields
	var ech echExtension

	ech.kdfID = uint16(kdf)
	ech.aeadID = uint16(aead)

	randomByte := make([]byte, 1)
	_, err = io.ReadFull(rand, randomByte)
	if err != nil {
		return nil, err
	}
	ech.configID = randomByte[0]

	ech.enc, _, err = sender.Setup(rand)
	if err != nil {
		return nil, err
	}

	// TODO: compute this correctly as per https://www.ietf.org/archive/id/draft-ietf-tls-esni-14.html#name-recommended-padding-scheme
	randomEncodedClientHelloInnerLen := 100
	cipherLen := int(aead.CipherLen(uint(randomEncodedClientHelloInnerLen)))
	ech.payload = make([]byte, randomEncodedClientHelloInnerLen+cipherLen)
	if _, err = io.ReadFull(rand, ech.payload); err != nil {
		return nil, err
	}

	return ech.marshal(), nil
}
