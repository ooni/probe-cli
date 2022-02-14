package quicping

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// buildHeader creates the unprotected QUIC header.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-initial-packet
func buildHeader(destConnID, srcConnID connectionID, payloadLen int) []byte {
	hdr := []byte{0xc3} // long header type, fixed

	version := make([]byte, 4)
	binary.BigEndian.PutUint32(version, uint32(0xbabababa))
	hdr = append(hdr, version...) // version

	lendID := uint8(len(destConnID))
	hdr = append(hdr, lendID)        // destination connection ID length
	hdr = append(hdr, destConnID...) // destination connection ID

	lensID := uint8(len(srcConnID))
	hdr = append(hdr, lensID)       // source connection ID length
	hdr = append(hdr, srcConnID...) // source connection ID

	hdr = append(hdr, 0x0) // token length

	remainder := 4 + payloadLen
	remainder_mask := 0b100000000000000
	remainder_mask |= remainder
	remainder_b := make([]byte, 2)
	binary.BigEndian.PutUint16(remainder_b, uint16(remainder_mask))
	hdr = append(hdr, remainder_b...) // remainder length: packet number + encrypted payload

	pn := make([]byte, 4)
	binary.BigEndian.PutUint32(pn, uint32(2))
	hdr = append(hdr, pn...) // packet number

	return hdr
}

// buildPacket constructs an Initial QUIC packet
// and applies Initial protection.
// https://www.rfc-editor.org/rfc/rfc9001.html#name-client-initial
func buildPacket() ([]byte, connectionID, connectionID) {
	destConnID, srcConnID := generateConnectionIDs()
	// generate random payload
	minPayloadSize := 1200 - 14 - (len(destConnID) + len(srcConnID))
	randomPayload := make([]byte, minPayloadSize)
	rand.Read(randomPayload)

	clientSecret, _ := computeSecrets(destConnID)
	encrypted := encryptPayload(randomPayload, destConnID, clientSecret)
	hdr := buildHeader(destConnID, srcConnID, len(encrypted))
	raw := append(hdr, encrypted...)

	raw = encryptHeader(raw, hdr, clientSecret)
	return raw, destConnID, srcConnID
}

// generateConnectionID generates a connection ID using cryptographic random
func generateConnectionID(len int) connectionID {
	b := make([]byte, len)
	_, err := rand.Read(b)
	runtimex.PanicOnError(err, "rand.Read failed")
	return connectionID(b)
}

// generateConnectionIDForInitial generates a connection ID for the Initial packet.
// It uses a length randomly chosen between 8 and 18 bytes.
func generateConnectionIDForInitial() connectionID {
	r := make([]byte, 1)
	_, err := rand.Read(r)
	runtimex.PanicOnError(err, "rand.Read failed")
	len := minConnectionIDLenInitial + int(r[0])%(maxConnectionIDLen-minConnectionIDLenInitial+1)
	return generateConnectionID(len)
}

// generateConnectionIDs generates a destination and source connection ID.
func generateConnectionIDs() ([]byte, []byte) {
	destConnID := generateConnectionIDForInitial()
	srcConnID := generateConnectionID(defaultConnectionIDLength)
	return destConnID, srcConnID
}

// dissectVersionNegotiation dissects the Version Negotiation response.
// It returns the supported versions and the destination connection ID of the response,
// The destination connection ID of the response has to coincide with the source connection ID of the request.
// https://www.rfc-editor.org/rfc/rfc9000.html#name-version-negotiation-packet
func (m *Measurer) dissectVersionNegotiation(i []byte) ([]uint32, connectionID, error) {
	firstByte := uint8(i[0])
	mask := 0b10000000
	mask &= int(firstByte)
	if mask == 0 {
		return nil, nil, &errUnexpectedResponse{msg: "not a long header packet"}
	}

	versionBytes := i[1:5]
	v := binary.BigEndian.Uint32(versionBytes)
	if v != 0 {
		return nil, nil, &errUnexpectedResponse{msg: "unexpected Version Negotiation format"}
	}

	dstLength := i[5]
	offset := 6 + uint8(dstLength)
	dst := i[6:offset]

	srcLength := i[offset]
	offset = offset + 1 + srcLength

	n := uint8(len(i))
	var supportedVersions []uint32
	for offset < n {
		supportedVersions = append(supportedVersions, binary.BigEndian.Uint32(i[offset:offset+4]))
		offset += 4
	}
	return supportedVersions, dst, nil
}

// errUnexpectedResponse is thrown when the response from the server
// is not a valid Version Negotiation packet
type errUnexpectedResponse struct {
	error
	msg string
}

// Error implements error.Error()
func (e *errUnexpectedResponse) Error() string {
	return fmt.Sprintf("unexptected response: %s", e.msg)
}
