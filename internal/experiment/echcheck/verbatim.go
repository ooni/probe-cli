// This file contains verbatim copies of the code from go's tls package at 1.23.4.
// https://github.com/golang/go/blob/go1.23.4/src/crypto/tls/common.go
// https://github.com/golang/go/blob/go1.23.4/src/crypto/tls/ech.go
// This should be refreshed to newer implementations if the ECH RFC change the format.

// This file is:
// Copyright 2024 The Go Authors. All rights reserved.
// Use of this code is governed by the BSD-style license found at:
// https://github.com/golang/go/blob/go1.23.4/LICENSE

package echcheck

import (
	"errors"

	"golang.org/x/crypto/cryptobyte"
)

const extensionEncryptedClientHello uint16 = 0xfe0d

type echCipher struct {
	KDFID  uint16
	AEADID uint16
}

type echExtension struct {
	Type uint16
	Data []byte
}

type echConfig struct {
	raw []byte

	Version uint16
	Length  uint16

	ConfigID             uint8
	KemID                uint16
	PublicKey            []byte
	SymmetricCipherSuite []echCipher

	MaxNameLength uint8
	PublicName    []byte
	Extensions    []echExtension
}

var errMalformedECHConfig = errors.New("tls: malformed ECHConfigList")

// parseECHConfigList parses a draft-ietf-tls-esni-18 ECHConfigList, returning a
// slice of parsed ECHConfigs, in the same order they were parsed, or an error
// if the list is malformed.
func parseECHConfigList(data []byte) ([]echConfig, error) {
	s := cryptobyte.String(data)
	// Skip the length prefix
	var length uint16
	if !s.ReadUint16(&length) {
		return nil, errMalformedECHConfig
	}
	if length != uint16(len(data)-2) {
		return nil, errMalformedECHConfig
	}
	var configs []echConfig
	for len(s) > 0 {
		var ec echConfig
		ec.raw = []byte(s)
		if !s.ReadUint16(&ec.Version) {
			return nil, errMalformedECHConfig
		}
		if !s.ReadUint16(&ec.Length) {
			return nil, errMalformedECHConfig
		}
		if len(ec.raw) < int(ec.Length)+4 {
			return nil, errMalformedECHConfig
		}
		ec.raw = ec.raw[:ec.Length+4]
		if ec.Version != extensionEncryptedClientHello {
			s.Skip(int(ec.Length))
			continue
		}
		if !s.ReadUint8(&ec.ConfigID) {
			return nil, errMalformedECHConfig
		}
		if !s.ReadUint16(&ec.KemID) {
			return nil, errMalformedECHConfig
		}
		if !s.ReadUint16LengthPrefixed((*cryptobyte.String)(&ec.PublicKey)) {
			return nil, errMalformedECHConfig
		}
		var cipherSuites cryptobyte.String
		if !s.ReadUint16LengthPrefixed(&cipherSuites) {
			return nil, errMalformedECHConfig
		}
		for !cipherSuites.Empty() {
			var c echCipher
			if !cipherSuites.ReadUint16(&c.KDFID) {
				return nil, errMalformedECHConfig
			}
			if !cipherSuites.ReadUint16(&c.AEADID) {
				return nil, errMalformedECHConfig
			}
			ec.SymmetricCipherSuite = append(ec.SymmetricCipherSuite, c)
		}
		if !s.ReadUint8(&ec.MaxNameLength) {
			return nil, errMalformedECHConfig
		}
		var publicName cryptobyte.String
		if !s.ReadUint8LengthPrefixed(&publicName) {
			return nil, errMalformedECHConfig
		}
		ec.PublicName = publicName
		var extensions cryptobyte.String
		if !s.ReadUint16LengthPrefixed(&extensions) {
			return nil, errMalformedECHConfig
		}
		for !extensions.Empty() {
			var e echExtension
			if !extensions.ReadUint16(&e.Type) {
				return nil, errMalformedECHConfig
			}
			if !extensions.ReadUint16LengthPrefixed((*cryptobyte.String)(&e.Data)) {
				return nil, errMalformedECHConfig
			}
			ec.Extensions = append(ec.Extensions, e)
		}

		configs = append(configs, ec)
	}
	return configs, nil
}
